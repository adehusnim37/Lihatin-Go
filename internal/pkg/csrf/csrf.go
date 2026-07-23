package csrf

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// ===========================================================================
// CONSTANTS & ERRORS
// ===========================================================================

const (
	// TokenLength adalah ukuran raw token dalam bytes (32 bytes = 256 bits)
	TokenLength = 32

	// DefaultCookieName untuk CSRF cookie
	DefaultCookieName = "_csrf"

	// DefaultHeaderName untuk CSRF token di request header
	DefaultHeaderName = "X-CSRF-Token"

	// DefaultFormField untuk CSRF token di form body
	DefaultFormField = "_csrf"

	// DefaultMaxAge untuk cookie (12 jam dalam detik)
	DefaultMaxAge = 43200

	// Redis key prefix untuk CSRF tokens
	RedisKeyPrefix = "csrf:"
)

// Predefined errors
var (
	ErrNoToken      = errors.New("csrf token not found in request")
	ErrBadToken     = errors.New("csrf token invalid or expired")
	ErrBadOrigin    = errors.New("origin not allowed")
	ErrBadReferer   = errors.New("referer validation failed")
	ErrTokenExpired = errors.New("csrf token has expired")
)

// Safe browser methods that do not require a CSRF token.
var safeMethods = []string{"GET", "HEAD", "OPTIONS"}

// ===========================================================================
// OPTIONS STRUCT
// ===========================================================================

// Options berisi konfigurasi untuk CSRF middleware
type Options struct {
	// Secret key untuk HMAC signing (WAJIB, minimal 32 bytes)
	Secret []byte

	// Cookie settings
	CookieName   string
	CookieDomain string
	CookiePath   string
	MaxAge       int
	Secure       bool // true = HTTPS only
	HttpOnly     bool // true = tidak bisa diakses JavaScript
	SameSite     http.SameSite

	// Header & form field names
	HeaderName string
	FormField  string

	// Trusted origins untuk cross-origin requests
	TrustedOrigins []string

	// Authentication cookies used to bind tokens to the current login session.
	// The first cookie found is used.
	SessionCookieNames []string

	// Redis client untuk token storage (optional, default: cookie-based)
	RedisClient *redis.Client

	// Error handler kustom
	ErrorHandler gin.HandlerFunc

	// Skip CSRF check untuk path tertentu (legacy exact-path matching).
	SkipPaths []string

	// SkipRules mendefinisikan route dan method yang boleh bypass CSRF.
	// Path dapat berupa raw URL path ("/v1/auth/login") atau gin full path
	// pattern ("/v1/support/tickets/:ticketCode/messages").
	SkipRules []SkipRule
}

// SkipRule mendefinisikan satu rule bypass CSRF yang explicit.
type SkipRule struct {
	Method          string
	Path            string
	SkipOriginCheck bool
}

// ===========================================================================
// TOKEN STRUCTURE
// ===========================================================================

// Token structure:
// [timestamp:8bytes][random:24bytes] -> 32 bytes raw
// Kemudian di-sign dengan HMAC-SHA256
// Final format: base64(raw) + "." + base64(signature)

type csrfToken struct {
	Raw       []byte
	Signature []byte
	Timestamp time.Time
}

// ===========================================================================
// CSRF MIDDLEWARE
// ===========================================================================

// Middleware creates a new CSRF protection middleware
func Middleware(opts Options) gin.HandlerFunc {
	// Set defaults
	if opts.CookieName == "" {
		opts.CookieName = DefaultCookieName
	}
	if opts.HeaderName == "" {
		opts.HeaderName = DefaultHeaderName
	}
	if opts.FormField == "" {
		opts.FormField = DefaultFormField
	}
	if opts.MaxAge == 0 {
		opts.MaxAge = DefaultMaxAge
	}
	if opts.CookiePath == "" {
		opts.CookiePath = "/"
	}
	if opts.SameSite == 0 {
		opts.SameSite = http.SameSiteLaxMode
	}
	if len(opts.SessionCookieNames) == 0 {
		opts.SessionCookieNames = []string{"refresh_token", "access_token", "session_id"}
	}
	if opts.ErrorHandler == nil {
		opts.ErrorHandler = defaultErrorHandler
	}
	if len(opts.Secret) < 32 {
		panic("csrf: secret key must be at least 32 bytes")
	}

	return func(c *gin.Context) {
		// Safe requests initialize the CSRF cookie so token endpoints and
		// server-rendered forms can expose a masked token.
		if isSafeMethod(c.Request.Method) {
			token, err := getOrCreateToken(c, opts)
			if err != nil {
				logger.Logger.Error("CSRF: Failed to get/create token", "error", err)
				opts.ErrorHandler(c)
				return
			}
			c.Set("csrf_token", token)
			c.Next()
			return
		}

		// Origin and Referer checks apply to every unsafe browser request,
		// including public/login routes that intentionally skip token checks.
		// Explicit non-browser routes may bypass them when downstream
		// middleware still authenticates the request.
		if !shouldSkipOriginCheck(c, opts.SkipRules) {
			if err := validateOrigin(c, opts); err != nil {
				logger.Logger.Warn("CSRF: Origin validation failed",
					"error", err,
					"origin", c.Request.Header.Get("Origin"),
					"path", c.Request.URL.Path,
				)
				opts.ErrorHandler(c)
				return
			}

			if err := validateReferer(c, opts); err != nil {
				logger.Logger.Warn("CSRF: Referer validation failed",
					"error", err,
					"referer", c.Request.Referer(),
					"path", c.Request.URL.Path,
				)
				opts.ErrorHandler(c)
				return
			}
		}

		// Token bypass never bypasses the origin checks above.
		if shouldSkipRequest(c, opts.SkipRules, opts.SkipPaths) {
			c.Next()
			return
		}

		token, err := getOrCreateToken(c, opts)
		if err != nil {
			logger.Logger.Error("CSRF: Failed to get/create token", "error", err)
			opts.ErrorHandler(c)
			return
		}
		c.Set("csrf_token", token)

		requestToken := getTokenFromRequest(c, opts)
		if requestToken == "" {
			logger.Logger.Warn("CSRF: No token in request",
				"path", c.Request.URL.Path,
				"method", c.Request.Method,
			)
			c.Error(ErrNoToken)
			opts.ErrorHandler(c)
			return
		}

		if err := validateToken(requestToken, token, sessionBinding(c, opts), opts); err != nil {
			logger.Logger.Warn("CSRF: Token validation failed",
				"error", err,
				"path", c.Request.URL.Path,
			)
			c.Error(ErrBadToken)
			opts.ErrorHandler(c)
			return
		}

		logger.Logger.Debug("CSRF: Token validated successfully",
			"path", c.Request.URL.Path,
		)
		c.Next()
	}
}

// ===========================================================================
// TOKEN GENERATION
// ===========================================================================

// generateToken creates a new CSRF token
func generateToken(secret, binding []byte) (string, error) {
	// 1. Generate raw token: timestamp (8 bytes) + random (24 bytes)
	raw := make([]byte, TokenLength)

	// Embed timestamp (untuk expiry check)
	var timestampBytes bytes.Buffer
	if err := binary.Write(&timestampBytes, binary.BigEndian, time.Now().Unix()); err != nil {
		return "", fmt.Errorf("failed to encode timestamp: %w", err)
	}
	copy(raw[:8], timestampBytes.Bytes())

	// Fill rest with random bytes
	if _, err := rand.Read(raw[8:]); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// 2. Create HMAC signature
	signature := signToken(raw, binding, secret)

	// 3. Encode: base64(raw).base64(signature)
	rawEncoded := base64.RawURLEncoding.EncodeToString(raw)
	sigEncoded := base64.RawURLEncoding.EncodeToString(signature)

	return rawEncoded + "." + sigEncoded, nil
}

// signToken creates HMAC-SHA256 signature
func signToken(data, binding, secret []byte) []byte {
	h := hmac.New(sha256.New, secret)
	var bindingLength [8]byte
	binary.BigEndian.PutUint64(bindingLength[:], uint64(len(binding)))
	_, _ = h.Write(bindingLength[:])
	_, _ = h.Write(binding)
	_, _ = h.Write(data)
	return h.Sum(nil)
}

// parseToken decodes and validates a token string
func parseToken(tokenStr string, secret, binding []byte, maxAge int) (*csrfToken, error) {
	// 1. Split into parts
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 2 {
		return nil, errors.New("invalid token format")
	}

	// 2. Decode raw token
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("failed to decode token: %w", err)
	}
	if len(raw) != TokenLength {
		return nil, errors.New("invalid token length")
	}

	// 3. Decode signature
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode signature: %w", err)
	}

	// 4. Verify signature (timing-safe comparison)
	expectedSig := signToken(raw, binding, secret)
	if !hmac.Equal(signature, expectedSig) {
		return nil, errors.New("invalid signature")
	}

	// 5. Extract and check timestamp
	var timestamp int64
	if err := binary.Read(bytes.NewReader(raw[:8]), binary.BigEndian, &timestamp); err != nil {
		return nil, fmt.Errorf("failed to decode timestamp: %w", err)
	}

	tokenTime := time.Unix(timestamp, 0)
	if time.Since(tokenTime) > time.Duration(maxAge)*time.Second {
		return nil, ErrTokenExpired
	}

	return &csrfToken{
		Raw:       raw,
		Signature: signature,
		Timestamp: tokenTime,
	}, nil
}

// ===========================================================================
// TOKEN MASKING (BREACH Protection)
// ===========================================================================

// maskToken applies one-time pad masking to prevent BREACH attacks
// Output: base64(pad + XOR(token, pad))
func maskToken(token string) (string, error) {
	tokenBytes := []byte(token)

	// Generate random pad
	pad := make([]byte, len(tokenBytes))
	if _, err := rand.Read(pad); err != nil {
		return "", err
	}

	// XOR token with pad
	masked := make([]byte, len(tokenBytes))
	for i := range tokenBytes {
		masked[i] = tokenBytes[i] ^ pad[i]
	}

	// Combine: pad + masked
	combined := append(pad, masked...)
	return base64.RawURLEncoding.EncodeToString(combined), nil
}

// unmaskToken reverses the masking
func unmaskToken(maskedToken string) (string, error) {
	combined, err := base64.RawURLEncoding.DecodeString(maskedToken)
	if err != nil {
		return "", err
	}

	if len(combined)%2 != 0 {
		return "", errors.New("invalid masked token length")
	}

	halfLen := len(combined) / 2
	pad := combined[:halfLen]
	masked := combined[halfLen:]

	// XOR to recover original
	original := make([]byte, halfLen)
	for i := range masked {
		original[i] = masked[i] ^ pad[i]
	}

	return string(original), nil
}

// ===========================================================================
// VALIDATION FUNCTIONS
// ===========================================================================

// validateOrigin checks Origin header against the request origin and trusted origins.
func validateOrigin(c *gin.Context, opts Options) error {
	origin := c.Request.Header.Get("Origin")
	if origin == "" {
		return nil // No origin header, will check referer
	}

	requestOrigin, err := canonicalRequestOrigin(c)
	if err != nil {
		return ErrBadOrigin
	}

	candidateOrigin, err := canonicalOrigin(origin, false)
	if err != nil {
		return ErrBadOrigin
	}

	if candidateOrigin == requestOrigin {
		return nil
	}

	for _, trusted := range opts.TrustedOrigins {
		if isTrustedOrigin(candidateOrigin, trusted) {
			return nil
		}
	}

	return ErrBadOrigin
}

// validateReferer checks Referer header for HTTPS requests
func validateReferer(c *gin.Context, opts Options) error {
	requestOrigin, err := canonicalRequestOrigin(c)
	if err != nil {
		return ErrBadReferer
	}

	if !strings.HasPrefix(requestOrigin, "https://") {
		return nil
	}

	origin := c.Request.Header.Get("Origin")
	if origin != "" {
		return nil // Already validated via Origin
	}

	referer := c.Request.Referer()
	if referer == "" {
		return ErrBadReferer
	}

	refererOrigin, err := canonicalOrigin(referer, true)
	if err != nil {
		return ErrBadReferer
	}

	if !strings.HasPrefix(refererOrigin, "https://") {
		return ErrBadReferer
	}

	if refererOrigin == requestOrigin {
		return nil
	}

	for _, trusted := range opts.TrustedOrigins {
		if isTrustedOrigin(refererOrigin, trusted) {
			return nil
		}
	}

	return ErrBadReferer
}

func canonicalRequestOrigin(c *gin.Context) (string, error) {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	} else if forwarded := strings.TrimSpace(c.GetHeader("X-Forwarded-Proto")); forwarded != "" {
		switch strings.ToLower(forwarded) {
		case "http", "https":
			scheme = strings.ToLower(forwarded)
		default:
			return "", ErrBadOrigin
		}
	}

	return canonicalOrigin(scheme+"://"+c.Request.Host, false)
}

// canonicalOrigin normalizes an HTTP(S) origin as scheme://host[:port].
// Referer URLs may contain a path; Origin and trusted-origin values may not.
func canonicalOrigin(raw string, allowPath bool) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil {
		return "", ErrBadOrigin
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", ErrBadOrigin
	}
	if !allowPath && parsed.Path != "" && parsed.Path != "/" {
		return "", ErrBadOrigin
	}
	if !allowPath && (parsed.RawQuery != "" || parsed.Fragment != "") {
		return "", ErrBadOrigin
	}

	hostname := strings.ToLower(parsed.Hostname())
	if hostname == "" {
		return "", ErrBadOrigin
	}
	port := parsed.Port()
	if (scheme == "http" && port == "80") || (scheme == "https" && port == "443") {
		port = ""
	}

	host := hostname
	if strings.Contains(hostname, ":") {
		host = "[" + hostname + "]"
	}
	if port != "" {
		host = net.JoinHostPort(hostname, port)
	}

	return (&url.URL{Scheme: scheme, Host: host}).String(), nil
}

func isTrustedOrigin(requestOrigin, trusted string) bool {
	trusted = strings.TrimSpace(trusted)
	if trusted == "" {
		return false
	}

	trustedOrigin, err := canonicalOrigin(trusted, false)
	if err != nil {
		return false
	}
	return requestOrigin == trustedOrigin
}

// validateToken compares request token with stored token
func validateToken(requestToken, storedToken string, binding []byte, opts Options) error {
	// Unmask if masked
	unmaskedRequest, err := unmaskToken(requestToken)
	if err != nil {
		// Try as unmasked token
		unmaskedRequest = requestToken
	}

	// Parse and validate both tokens
	_, err = parseToken(unmaskedRequest, opts.Secret, binding, opts.MaxAge)
	if err != nil {
		return err
	}

	// Unmask stored token for comparison
	unmaskedStored, err := unmaskToken(storedToken)
	if err != nil {
		unmaskedStored = storedToken
	}

	// Timing-safe comparison
	if subtle.ConstantTimeCompare([]byte(unmaskedRequest), []byte(unmaskedStored)) != 1 {
		return ErrBadToken
	}

	return nil
}

// ===========================================================================
// HELPER FUNCTIONS
// ===========================================================================

// getOrCreateToken retrieves existing token or creates new one
func getOrCreateToken(c *gin.Context, opts Options) (string, error) {
	binding := sessionBinding(c, opts)

	// Try to get from cookie
	cookieToken, err := c.Cookie(opts.CookieName)
	if err == nil && cookieToken != "" {
		// Validate it's still valid
		if _, err := parseToken(cookieToken, opts.Secret, binding, opts.MaxAge); err == nil {
			return cookieToken, nil
		}
	}

	// Generate new token
	token, err := generateToken(opts.Secret, binding)
	if err != nil {
		return "", err
	}

	// Derive secure flag dynamically:
	// - keep secure cookie on HTTPS requests
	// - allow non-secure cookie for localhost HTTP dev/testing
	secureCookie := opts.Secure
	if opts.Secure {
		isHTTPS := c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https")
		if !isHTTPS && isLocalHost(c.Request.Host) {
			secureCookie = false
			logger.Logger.Warn("CSRF: Secure cookie disabled for localhost over HTTP",
				"host", c.Request.Host,
				"path", c.Request.URL.Path,
			)
		}
	}

	// Set SameSite before SetCookie so cookie attributes are applied consistently.
	c.SetSameSite(opts.SameSite)

	// Save to cookie
	c.SetCookie(
		opts.CookieName,
		token,
		opts.MaxAge,
		opts.CookiePath,
		opts.CookieDomain,
		secureCookie,
		opts.HttpOnly,
	)

	return token, nil
}

func sessionBinding(c *gin.Context, opts Options) []byte {
	for _, cookieName := range opts.SessionCookieNames {
		cookieName = strings.TrimSpace(cookieName)
		if cookieName == "" {
			continue
		}
		value, err := c.Cookie(cookieName)
		if err != nil || value == "" {
			continue
		}

		sum := sha256.Sum256([]byte(cookieName + "\x00" + value))
		return sum[:]
	}

	return []byte("anonymous")
}

func isLocalHost(host string) bool {
	// Handle host:port and IPv6 bracket format.
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	host = strings.Trim(host, "[]")
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

// getTokenFromRequest extracts CSRF token from request
func getTokenFromRequest(c *gin.Context, opts Options) string {
	// 1. Check header first
	if token := c.GetHeader(opts.HeaderName); token != "" {
		return token
	}

	// 2. Check form field
	if token := c.PostForm(opts.FormField); token != "" {
		return token
	}

	return ""
}

// isSafeMethod checks whether the method is safe and read-only.
func isSafeMethod(method string) bool {
	for _, m := range safeMethods {
		if m == method {
			return true
		}
	}
	return false
}

// shouldSkipRequest checks if request should skip CSRF.
func shouldSkipRequest(c *gin.Context, skipRules []SkipRule, skipPaths []string) bool {
	return shouldSkipRoute(
		c.Request.Method,
		c.Request.URL.Path,
		c.FullPath(),
		skipRules,
		skipPaths,
	)
}

func shouldSkipOriginCheck(c *gin.Context, skipRules []SkipRule) bool {
	requestMethod := strings.ToUpper(c.Request.Method)
	requestPath := c.Request.URL.Path
	fullPath := c.FullPath()

	for _, rule := range skipRules {
		if !rule.SkipOriginCheck || strings.ToUpper(rule.Method) != requestMethod {
			continue
		}
		if rule.Path == requestPath || (fullPath != "" && rule.Path == fullPath) {
			return true
		}
	}
	return false
}

func shouldSkipRoute(method, requestPath, fullPath string, skipRules []SkipRule, skipPaths []string) bool {
	requestMethod := strings.ToUpper(method)

	for _, rule := range skipRules {
		if strings.ToUpper(rule.Method) != requestMethod {
			continue
		}
		if rule.Path == requestPath || (fullPath != "" && rule.Path == fullPath) {
			return true
		}
	}

	for _, skip := range skipPaths {
		if skip == requestPath || (fullPath != "" && skip == fullPath) {
			return true
		}
	}
	return false
}

// defaultErrorHandler handles CSRF errors
func defaultErrorHandler(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
		"success": false,
		"message": "CSRF validation failed",
		"error":   "Invalid or missing CSRF token",
	})
}

// ===========================================================================
// PUBLIC HELPER FUNCTIONS
// ===========================================================================

// GetToken returns the CSRF token for the current request
// Use this to include token in API responses
func GetToken(c *gin.Context) string {
	if token, exists := c.Get("csrf_token"); exists {
		return token.(string)
	}
	return ""
}

// GetMaskedToken returns a masked version of the token (for forms/headers)
func GetMaskedToken(c *gin.Context) string {
	token := GetToken(c)
	if token == "" {
		return ""
	}

	masked, err := maskToken(token)
	if err != nil {
		logger.Logger.Error("CSRF: Failed to mask token", "error", err)
		return token
	}
	return masked
}

// TemplateFunc returns a function for use in templates
func TemplateFunc(c *gin.Context) string {
	return fmt.Sprintf(`<input type="hidden" name="%s" value="%s">`, DefaultFormField, GetMaskedToken(c))
}

// ===========================================================================
// CONFIGURATION HELPER
// ===========================================================================

// DefaultOptions returns sensible defaults for production
func DefaultOptions() Options {
	secret := config.GetEnvOrDefault(config.EnvCSRFSecret, "")
	if secret == "" {
		if isProductionEnv() {
			panic("csrf: CSRF_SECRET is required in production")
		}

		randomBytes := make([]byte, 32)
		if _, err := rand.Read(randomBytes); err != nil {
			panic(fmt.Sprintf("csrf: failed to generate development secret: %v", err))
		}
		secret = hex.EncodeToString(randomBytes)
		logger.Logger.Warn("CSRF: No CSRF_SECRET set, using an ephemeral development key")
	}

	return Options{
		Secret:             []byte(secret),
		CookieName:         DefaultCookieName,
		CookieDomain:       "", // Host-only prevents sibling subdomain cookie injection.
		CookiePath:         "/",
		MaxAge:             DefaultMaxAge,
		Secure:             isProductionEnv(),
		HttpOnly:           true,
		SameSite:           http.SameSiteLaxMode,
		HeaderName:         DefaultHeaderName,
		FormField:          DefaultFormField,
		SessionCookieNames: []string{"refresh_token", "access_token", "session_id"},
		TrustedOrigins: strings.Split(
			config.GetEnvOrDefault(config.EnvAllowedOrigins, ""),
			",",
		),
	}
}

func isProductionEnv() bool {
	return strings.EqualFold(strings.TrimSpace(config.GetEnvOrDefault(config.Env, "development")), "production")
}

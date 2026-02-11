package csrf

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
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

// Safe HTTP methods yang tidak perlu CSRF check (RFC 7231)
var safeMethods = []string{"GET", "HEAD", "OPTIONS", "TRACE"}

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

	// Redis client untuk token storage (optional, default: cookie-based)
	RedisClient *redis.Client

	// Error handler kustom
	ErrorHandler gin.HandlerFunc

	// Skip CSRF check untuk path tertentu
	SkipPaths []string
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
	if opts.ErrorHandler == nil {
		opts.ErrorHandler = defaultErrorHandler
	}
	if len(opts.Secret) < 32 {
		panic("csrf: secret key must be at least 32 bytes")
	}

	return func(c *gin.Context) {
		// 1. Check if path should be skipped
		if shouldSkipPath(c.Request.URL.Path, opts.SkipPaths) {
			c.Next()
			return
		}

		// 2. Get or generate token
		token, err := getOrCreateToken(c, opts)
		if err != nil {
			logger.Logger.Error("CSRF: Failed to get/create token", "error", err)
			opts.ErrorHandler(c)
			return
		}

		// 3. Save token to context (untuk dipakai di handler)
		c.Set("csrf_token", token)

		// 4. For safe methods, just continue
		if isSafeMethod(c.Request.Method) {
			c.Next()
			return
		}

		// 5. For unsafe methods, validate token
		// 5a. Check Origin header
		if err := validateOrigin(c, opts); err != nil {
			logger.Logger.Warn("CSRF: Origin validation failed",
				"error", err,
				"origin", c.Request.Header.Get("Origin"),
				"path", c.Request.URL.Path,
			)
			opts.ErrorHandler(c)
			return
		}

		// 5b. Check Referer (untuk HTTPS)
		if err := validateReferer(c, opts); err != nil {
			logger.Logger.Warn("CSRF: Referer validation failed",
				"error", err,
				"referer", c.Request.Referer(),
				"path", c.Request.URL.Path,
			)
			opts.ErrorHandler(c)
			return
		}

		// 5c. Get token from request (header or form)
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

		// 5d. Validate token
		if err := validateToken(requestToken, token, opts); err != nil {
			logger.Logger.Warn("CSRF: Token validation failed",
				"error", err,
				"path", c.Request.URL.Path,
			)
			c.Error(ErrBadToken)
			opts.ErrorHandler(c)
			return
		}

		// 6. Token valid, continue
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
func generateToken(secret []byte) (string, error) {
	// 1. Generate raw token: timestamp (8 bytes) + random (24 bytes)
	raw := make([]byte, TokenLength)

	// Embed timestamp (untuk expiry check)
	timestamp := time.Now().Unix()
	raw[0] = byte(timestamp >> 56)
	raw[1] = byte(timestamp >> 48)
	raw[2] = byte(timestamp >> 40)
	raw[3] = byte(timestamp >> 32)
	raw[4] = byte(timestamp >> 24)
	raw[5] = byte(timestamp >> 16)
	raw[6] = byte(timestamp >> 8)
	raw[7] = byte(timestamp)

	// Fill rest with random bytes
	if _, err := rand.Read(raw[8:]); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// 2. Create HMAC signature
	signature := signToken(raw, secret)

	// 3. Encode: base64(raw).base64(signature)
	rawEncoded := base64.RawURLEncoding.EncodeToString(raw)
	sigEncoded := base64.RawURLEncoding.EncodeToString(signature)

	return rawEncoded + "." + sigEncoded, nil
}

// signToken creates HMAC-SHA256 signature
func signToken(data, secret []byte) []byte {
	h := hmac.New(sha256.New, secret)
	h.Write(data)
	return h.Sum(nil)
}

// parseToken decodes and validates a token string
func parseToken(tokenStr string, secret []byte, maxAge int) (*csrfToken, error) {
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
	expectedSig := signToken(raw, secret)
	if !hmac.Equal(signature, expectedSig) {
		return nil, errors.New("invalid signature")
	}

	// 5. Extract and check timestamp
	timestamp := int64(raw[0])<<56 | int64(raw[1])<<48 | int64(raw[2])<<40 |
		int64(raw[3])<<32 | int64(raw[4])<<24 | int64(raw[5])<<16 |
		int64(raw[6])<<8 | int64(raw[7])

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

// validateOrigin checks Origin header against trusted origins
func validateOrigin(c *gin.Context, opts Options) error {
	origin := c.Request.Header.Get("Origin")
	if origin == "" {
		return nil // No origin header, will check referer
	}

	parsedOrigin, err := url.Parse(origin)
	if err != nil {
		return ErrBadOrigin
	}

	// Check if same origin
	if parsedOrigin.Host == c.Request.Host {
		return nil
	}

	// Check trusted origins
	for _, trusted := range opts.TrustedOrigins {
		if isTrustedOrigin(parsedOrigin.Host, trusted) {
			return nil
		}
	}

	return ErrBadOrigin
}

// validateReferer checks Referer header for HTTPS requests
func validateReferer(c *gin.Context, opts Options) error {
	// Skip for HTTP (development)
	if c.Request.TLS == nil && c.Request.Header.Get("X-Forwarded-Proto") != "https" {
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

	parsedReferer, err := url.Parse(referer)
	if err != nil {
		return ErrBadReferer
	}

	// Must be HTTPS
	if parsedReferer.Scheme != "https" {
		return ErrBadReferer
	}

	// Check if same host
	if parsedReferer.Host == c.Request.Host {
		return nil
	}

	// Check trusted origins
	for _, trusted := range opts.TrustedOrigins {
		if isTrustedOrigin(parsedReferer.Host, trusted) {
			return nil
		}
	}

	return ErrBadReferer
}

// isTrustedOrigin supports env formats:
// - host only (e.g. lihat.in)
// - full origin URL (e.g. https://lihat.in)
func isTrustedOrigin(requestHost, trusted string) bool {
	trusted = strings.TrimSpace(trusted)
	if trusted == "" {
		return false
	}

	if !strings.Contains(trusted, "://") {
		return requestHost == trusted
	}

	parsed, err := url.Parse(trusted)
	if err != nil {
		return false
	}

	return requestHost == parsed.Host
}

// validateToken compares request token with stored token
func validateToken(requestToken, storedToken string, opts Options) error {
	// Unmask if masked
	unmaskedRequest, err := unmaskToken(requestToken)
	if err != nil {
		// Try as unmasked token
		unmaskedRequest = requestToken
	}

	// Parse and validate both tokens
	_, err = parseToken(unmaskedRequest, opts.Secret, opts.MaxAge)
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
	// Try to get from cookie
	cookieToken, err := c.Cookie(opts.CookieName)
	if err == nil && cookieToken != "" {
		// Validate it's still valid
		if _, err := parseToken(cookieToken, opts.Secret, opts.MaxAge); err == nil {
			return cookieToken, nil
		}
	}

	// Generate new token
	token, err := generateToken(opts.Secret)
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

	// Set SameSite via header (Gin's SetCookie doesn't support SameSite well)
	c.SetSameSite(opts.SameSite)

	return token, nil
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

	// 3. Check query param (less common)
	if token := c.Query(opts.FormField); token != "" {
		return token
	}

	return ""
}

// isSafeMethod checks if HTTP method is safe (idempotent)
func isSafeMethod(method string) bool {
	for _, m := range safeMethods {
		if m == method {
			return true
		}
	}
	return false
}

// shouldSkipPath checks if path should skip CSRF
func shouldSkipPath(path string, skipPaths []string) bool {
	for _, skip := range skipPaths {
		if strings.HasPrefix(path, skip) {
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
	return "Kosong/Dalam tahap pengembangan"
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
		// Generate a random secret if not set (will change on restart!)
		randomBytes := make([]byte, 32)
		rand.Read(randomBytes)
		secret = hex.EncodeToString(randomBytes)
		logger.Logger.Warn("CSRF: No CSRF_SECRET set, using random key (will change on restart)")
	}

	return Options{
		Secret:       []byte(secret),
		CookieName:   DefaultCookieName,
		CookieDomain: config.GetEnvOrDefault(config.EnvDomain, ""),
		CookiePath:   "/",
		MaxAge:       DefaultMaxAge,
		Secure:       config.GetEnvOrDefault(config.Env, "development") == "production",
		HttpOnly:     false, // Must be false for JS to read
		SameSite:     http.SameSiteLaxMode,
		HeaderName:   DefaultHeaderName,
		FormField:    DefaultFormField,
		TrustedOrigins: strings.Split(
			config.GetEnvOrDefault(config.EnvAllowedOrigins, ""),
			",",
		),
	}
}

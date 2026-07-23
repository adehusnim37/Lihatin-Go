package csrf

import (
	"crypto/tls"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

var testSecret = []byte("0123456789abcdef0123456789abcdef")

func testOptions() Options {
	return Options{
		Secret:             testSecret,
		CookieName:         DefaultCookieName,
		CookiePath:         "/",
		MaxAge:             DefaultMaxAge,
		Secure:             true,
		HttpOnly:           true,
		SameSite:           http.SameSiteLaxMode,
		HeaderName:         DefaultHeaderName,
		FormField:          DefaultFormField,
		SessionCookieNames: []string{"refresh_token", "access_token"},
		TrustedOrigins:     []string{"https://app.example.com"},
	}
}

func TestShouldSkipRouteUsesExactAndStructuredMatching(t *testing.T) {
	rules := []SkipRule{
		{Method: "POST", Path: "/v1/auth/login"},
		{Method: "POST", Path: "/v1/support/tickets/:ticketCode/messages"},
	}

	tests := []struct {
		name        string
		method      string
		requestPath string
		fullPath    string
		want        bool
	}{
		{
			name:        "exact path matches",
			method:      "POST",
			requestPath: "/v1/auth/login",
			fullPath:    "/v1/auth/login",
			want:        true,
		},
		{
			name:        "method mismatch does not skip",
			method:      "GET",
			requestPath: "/v1/auth/login",
			fullPath:    "/v1/auth/login",
			want:        false,
		},
		{
			name:        "dynamic gin route pattern matches full path",
			method:      "POST",
			requestPath: "/v1/support/tickets/LHTK-123/messages",
			fullPath:    "/v1/support/tickets/:ticketCode/messages",
			want:        true,
		},
		{
			name:        "legacy path list now exact only",
			method:      "POST",
			requestPath: "/v1/support/access/request-otp",
			fullPath:    "/v1/support/access/request-otp",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldSkipRoute(tt.method, tt.requestPath, tt.fullPath, rules, []string{"/v1/support/access"})
			if got != tt.want {
				t.Fatalf("shouldSkipRoute() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTokenIsBoundToSession(t *testing.T) {
	bindingA := []byte("session-a")
	bindingB := []byte("session-b")

	token, err := generateToken(testSecret, bindingA)
	if err != nil {
		t.Fatalf("generateToken() error = %v", err)
	}
	if _, err := parseToken(token, testSecret, bindingA, DefaultMaxAge); err != nil {
		t.Fatalf("token should validate for its session: %v", err)
	}
	if _, err := parseToken(token, testSecret, bindingB, DefaultMaxAge); err == nil {
		t.Fatal("token unexpectedly validated for a different session")
	}
}

func TestMaskRoundTrip(t *testing.T) {
	token, err := generateToken(testSecret, []byte("session"))
	if err != nil {
		t.Fatalf("generateToken() error = %v", err)
	}
	masked, err := maskToken(token)
	if err != nil {
		t.Fatalf("maskToken() error = %v", err)
	}
	if masked == token {
		t.Fatal("masked token must differ from the raw token")
	}
	unmasked, err := unmaskToken(masked)
	if err != nil {
		t.Fatalf("unmaskToken() error = %v", err)
	}
	if unmasked != token {
		t.Fatalf("unmasked token = %q, want %q", unmasked, token)
	}
}

func TestOriginValidationComparesSchemeHostAndPort(t *testing.T) {
	tests := []struct {
		name    string
		origin  string
		trusted []string
		wantErr bool
	}{
		{name: "same origin", origin: "https://api.example.com"},
		{name: "default port normalized", origin: "https://api.example.com:443"},
		{name: "trusted exact origin", origin: "https://app.example.com", trusted: []string{"https://app.example.com"}},
		{name: "scheme downgrade rejected", origin: "http://api.example.com", wantErr: true},
		{name: "trusted wrong scheme rejected", origin: "http://app.example.com", trusted: []string{"https://app.example.com"}, wantErr: true},
		{name: "host-only trusted entry rejected", origin: "https://app.example.com", trusted: []string{"app.example.com"}, wantErr: true},
		{name: "different port rejected", origin: "https://api.example.com:444", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodPost, "https://api.example.com/action", nil)
			ctx.Request.Host = "api.example.com"
			ctx.Request.TLS = &tls.ConnectionState{}
			ctx.Request.Header.Set("Origin", tt.origin)

			err := validateOrigin(ctx, Options{TrustedOrigins: tt.trusted})
			if tt.wantErr && err == nil {
				t.Fatal("validateOrigin() unexpectedly succeeded")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("validateOrigin() error = %v", err)
			}
		})
	}
}

func TestSkippedTokenRouteStillValidatesOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	opts := testOptions()
	opts.Secure = false
	opts.SkipRules = []SkipRule{{Method: http.MethodPost, Path: "/login"}}

	router := gin.New()
	router.Use(Middleware(opts))
	router.POST("/login", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	untrusted := httptest.NewRequest(http.MethodPost, "http://api.example.com/login", strings.NewReader(`{}`))
	untrusted.Header.Set("Origin", "https://evil.example")
	untrustedRecorder := httptest.NewRecorder()
	router.ServeHTTP(untrustedRecorder, untrusted)
	if untrustedRecorder.Code != http.StatusForbidden {
		t.Fatalf("untrusted skipped route status = %d, want %d", untrustedRecorder.Code, http.StatusForbidden)
	}

	trusted := httptest.NewRequest(http.MethodPost, "http://api.example.com/login", strings.NewReader(`{}`))
	trusted.Header.Set("Origin", "https://app.example.com")
	trustedRecorder := httptest.NewRecorder()
	router.ServeHTTP(trustedRecorder, trusted)
	if trustedRecorder.Code != http.StatusNoContent {
		t.Fatalf("trusted skipped route status = %d, want %d", trustedRecorder.Code, http.StatusNoContent)
	}
}

func TestExplicitNonBrowserRouteCanSkipOriginAndToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	opts := testOptions()
	opts.SkipRules = []SkipRule{{
		Method:          http.MethodPost,
		Path:            "/api/short",
		SkipOriginCheck: true,
	}}

	router := gin.New()
	router.Use(Middleware(opts))
	router.POST("/api/short", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodPost, "https://api.example.com/api/short", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("non-browser route status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
}

func TestCSRFCookieAttributesAndSessionRotation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	opts := testOptions()

	router := gin.New()
	router.Use(Middleware(opts))
	router.GET("/csrf", func(c *gin.Context) {
		c.String(http.StatusOK, GetMaskedToken(c))
	})
	router.POST("/protected", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	getRequest := httptest.NewRequest(http.MethodGet, "https://api.example.com/csrf", nil)
	getRequest.AddCookie(&http.Cookie{Name: "refresh_token", Value: "session-a"})
	getRecorder := httptest.NewRecorder()
	router.ServeHTTP(getRecorder, getRequest)

	var csrfCookie *http.Cookie
	for _, cookie := range getRecorder.Result().Cookies() {
		if cookie.Name == DefaultCookieName {
			csrfCookie = cookie
			break
		}
	}
	if csrfCookie == nil {
		t.Fatal("CSRF cookie was not set")
	}
	if !csrfCookie.HttpOnly || !csrfCookie.Secure || csrfCookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("unexpected cookie attributes: HttpOnly=%v Secure=%v SameSite=%v", csrfCookie.HttpOnly, csrfCookie.Secure, csrfCookie.SameSite)
	}
	if csrfCookie.Domain != "" {
		t.Fatalf("CSRF cookie domain = %q, want host-only", csrfCookie.Domain)
	}

	maskedToken := getRecorder.Body.String()
	validRequest := httptest.NewRequest(http.MethodPost, "https://api.example.com/protected", nil)
	validRequest.Header.Set("Origin", "https://api.example.com")
	validRequest.Header.Set(DefaultHeaderName, maskedToken)
	validRequest.AddCookie(csrfCookie)
	validRequest.AddCookie(&http.Cookie{Name: "refresh_token", Value: "session-a"})
	validRecorder := httptest.NewRecorder()
	router.ServeHTTP(validRecorder, validRequest)
	if validRecorder.Code != http.StatusNoContent {
		t.Fatalf("same-session request status = %d, want %d", validRecorder.Code, http.StatusNoContent)
	}

	rotatedRequest := httptest.NewRequest(http.MethodPost, "https://api.example.com/protected", nil)
	rotatedRequest.Header.Set("Origin", "https://api.example.com")
	rotatedRequest.Header.Set(DefaultHeaderName, maskedToken)
	rotatedRequest.AddCookie(csrfCookie)
	rotatedRequest.AddCookie(&http.Cookie{Name: "refresh_token", Value: "session-b"})
	rotatedRecorder := httptest.NewRecorder()
	router.ServeHTTP(rotatedRecorder, rotatedRequest)
	if rotatedRecorder.Code != http.StatusForbidden {
		t.Fatalf("different-session request status = %d, want %d", rotatedRecorder.Code, http.StatusForbidden)
	}
}

func TestQueryStringTokenIsRejected(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "http://api.example.com/action?_csrf=leaked", nil)

	if token := getTokenFromRequest(ctx, testOptions()); token != "" {
		t.Fatalf("getTokenFromRequest() accepted query token %q", token)
	}
}

func TestProductionRequiresConfiguredSecret(t *testing.T) {
	t.Setenv("ENV", "production")
	t.Setenv("CSRF_SECRET", "")

	defer func() {
		if recover() == nil {
			t.Fatal("DefaultOptions() did not panic without production CSRF_SECRET")
		}
	}()
	_ = DefaultOptions()
}

func TestCanonicalOriginRejectsNonHTTPAndPaths(t *testing.T) {
	for _, value := range []string{
		"javascript://api.example.com",
		"https://user@example.com",
		"https://api.example.com/not-an-origin",
		"https://api.example.com?query=1",
	} {
		if _, err := canonicalOrigin(value, false); !errors.Is(err, ErrBadOrigin) {
			t.Fatalf("canonicalOrigin(%q) error = %v, want ErrBadOrigin", value, err)
		}
	}
}

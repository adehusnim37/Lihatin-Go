package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCORSMiddlewareUsesExactCredentialedAllowlist(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("ENV", "production")
	t.Setenv("ALLOWED_ORIGINS", "https://app.example.com")

	router := gin.New()
	router.Use(CORSMiddleware())
	router.GET("/resource", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	tests := []struct {
		name        string
		origin      string
		wantAllowed bool
	}{
		{name: "configured origin", origin: "https://app.example.com", wantAllowed: true},
		{name: "localhost substring attack", origin: "https://localhost.evil.example"},
		{name: "configured host wrong scheme", origin: "http://app.example.com"},
		{name: "configured host wrong port", origin: "https://app.example.com:444"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "https://api.example.com/resource", nil)
			request.Header.Set("Origin", tt.origin)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)

			allowedOrigin := recorder.Header().Get("Access-Control-Allow-Origin")
			if tt.wantAllowed && allowedOrigin != tt.origin {
				t.Fatalf("allowed origin = %q, want %q", allowedOrigin, tt.origin)
			}
			if !tt.wantAllowed && allowedOrigin != "" {
				t.Fatalf("untrusted origin was allowed: %q", allowedOrigin)
			}
			if tt.wantAllowed && recorder.Header().Get("Access-Control-Allow-Credentials") != "true" {
				t.Fatal("credentialed configured origin was not enabled")
			}
			if recorder.Header().Get("Vary") != "Origin" {
				t.Fatalf("Vary = %q, want Origin", recorder.Header().Get("Vary"))
			}
		})
	}
}

func TestCORSDevelopmentLocalhostParsing(t *testing.T) {
	t.Setenv("ENV", "development")
	t.Setenv("ALLOWED_ORIGINS", "")

	if !isLocalhostOrigin("http://localhost:3000") {
		t.Fatal("localhost development origin was rejected")
	}
	if isLocalhostOrigin("https://localhost.evil.example") {
		t.Fatal("localhost substring origin was accepted")
	}
	if isLocalhostOrigin("https://user@localhost:3000") {
		t.Fatal("origin containing user info was accepted")
	}
}

func TestCORSRejectsUntrustedPreflight(t *testing.T) {
	t.Setenv("ENV", "production")
	t.Setenv("ALLOWED_ORIGINS", "https://app.example.com")

	router := gin.New()
	router.Use(CORSMiddleware())

	request := httptest.NewRequest(http.MethodOptions, "https://api.example.com/resource", nil)
	request.Header.Set("Origin", "https://evil.example")
	request.Header.Set("Access-Control-Request-Method", http.MethodPost)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("untrusted preflight status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
	if recorder.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("untrusted preflight received Access-Control-Allow-Origin")
	}
}

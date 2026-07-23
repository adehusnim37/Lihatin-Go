package auth

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestResolveSameSiteSecureDefaults(t *testing.T) {
	tests := []struct {
		value     string
		secure    bool
		wantMode  http.SameSite
		wantLabel string
	}{
		{value: "", secure: true, wantMode: http.SameSiteLaxMode, wantLabel: "Lax"},
		{value: "lax", secure: true, wantMode: http.SameSiteLaxMode, wantLabel: "Lax"},
		{value: "strict", secure: true, wantMode: http.SameSiteStrictMode, wantLabel: "Strict"},
		{value: "none", secure: true, wantMode: http.SameSiteNoneMode, wantLabel: "None"},
		{value: "none", secure: false, wantMode: http.SameSiteLaxMode, wantLabel: "Lax"},
	}

	for _, tt := range tests {
		mode, label := resolveSameSite(tt.value, tt.secure)
		if mode != tt.wantMode || label != tt.wantLabel {
			t.Fatalf("resolveSameSite(%q, %v) = (%v, %q), want (%v, %q)", tt.value, tt.secure, mode, label, tt.wantMode, tt.wantLabel)
		}
	}
}

func TestHTTPSAuthCookiesDefaultToLax(t *testing.T) {
	t.Setenv("ENV", "production")
	t.Setenv("AUTH_COOKIE_SAME_SITE", "lax")
	t.Setenv("DOMAIN", "example.com")

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "https://api.example.com", nil)
	ctx.Request.TLS = &tls.ConnectionState{}

	settings := ResolveAuthCookieSettings(ctx)
	if settings.RejectInsecureRequest {
		t.Fatal("HTTPS request was marked insecure")
	}
	if !settings.Secure || settings.SameSite != http.SameSiteLaxMode {
		t.Fatalf("settings = Secure:%v SameSite:%v, want Secure and Lax", settings.Secure, settings.SameSite)
	}
}

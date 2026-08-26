package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestClearAuthCookiesExpiresAllSessionCookies(t *testing.T) {
	t.Setenv("ENV", "development")
	t.Setenv("DOMAIN", "")
	t.Setenv("AUTH_COOKIE_SAME_SITE", "lax")
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest("GET", "https://api.example.test/protected", nil)

	ClearAuthCookies(ctx)

	cookies := recorder.Result().Cookies()
	want := map[string]bool{
		"access_token":  false,
		"refresh_token": false,
		"session_id":    false,
		"_csrf":         false,
	}

	for _, cookie := range cookies {
		if _, tracked := want[cookie.Name]; !tracked {
			continue
		}
		if cookie.Value != "" {
			t.Errorf("cookie %q value = %q, want empty", cookie.Name, cookie.Value)
		}
		if cookie.MaxAge >= 0 {
			t.Errorf("cookie %q MaxAge = %d, want negative", cookie.Name, cookie.MaxAge)
		}
		want[cookie.Name] = true
	}

	for name, found := range want {
		if !found {
			t.Errorf("missing expired Set-Cookie header for %q", name)
		}
	}
}

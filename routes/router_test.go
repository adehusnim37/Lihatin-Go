package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCSRFTokenHandlerDisablesCaching(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/csrf-token", nil)
	ctx.Set("csrf_token", "token")

	csrfTokenHandler(ctx)

	if got := recorder.Header().Get("Cache-Control"); got != "no-store, max-age=0" {
		t.Fatalf("Cache-Control = %q", got)
	}
	if got := recorder.Header().Get("Pragma"); got != "no-cache" {
		t.Fatalf("Pragma = %q", got)
	}
}

func TestAPIKeyMutationRoutesSkipOnlyTokenValidation(t *testing.T) {
	rules := csrfTokenSkipRules()
	tests := []struct {
		method string
		path   string
		full   string
	}{
		{method: http.MethodPost, path: "/v1/api/short", full: "/v1/api/short"},
		{method: http.MethodPut, path: "/v1/api/short/code", full: "/v1/api/short/:code"},
		{method: http.MethodDelete, path: "/v1/api/short/code", full: "/v1/api/short/:code"},
	}

	for _, tt := range tests {
		found := false
		for _, rule := range rules {
			if rule.Method == tt.method && rule.Path == tt.full {
				if !rule.SkipOriginCheck {
					t.Fatalf("API-key rule %s %s must explicitly skip browser origin checks", tt.method, tt.path)
				}
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing API-key CSRF skip rule for %s %s", tt.method, tt.path)
		}
	}
}

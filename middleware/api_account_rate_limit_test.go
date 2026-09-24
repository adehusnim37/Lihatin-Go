package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAPIAccountRateLimitKeyIsSharedAcrossKeysIPsAndRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	makeContext := func(ip, method, path, userID, apiKeyID string) *gin.Context {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(method, path, nil)
		c.Request.RemoteAddr = ip + ":12345"
		c.Set("user_id", userID)
		c.Set("api_key_id", apiKeyID)
		return c
	}

	first, ok := accountRateLimitKey(makeContext("192.0.2.1", "GET", "/api/short", "owner", "key-a"), "standard")
	if !ok {
		t.Fatal("authenticated account has no rate limit key")
	}
	second, ok := accountRateLimitKey(makeContext("198.51.100.2", "POST", "/api/short/other", "owner", "key-b"), "standard")
	if !ok || first != second {
		t.Fatalf("same account used different counters: %q and %q", first, second)
	}
	otherUser, _ := accountRateLimitKey(makeContext("192.0.2.1", "GET", "/api/short", "other", "key-c"), "standard")
	premium, _ := accountRateLimitKey(makeContext("192.0.2.1", "GET", "/api/short", "owner", "key-a"), "premium")
	if first == otherUser || first == premium {
		t.Fatal("different accounts or tiers shared a counter")
	}
	missing := makeContext("192.0.2.1", "GET", "/api/short", "", "key-a")
	if _, ok := accountRateLimitKey(missing, "standard"); ok {
		t.Fatal("missing account ID accepted")
	}
}

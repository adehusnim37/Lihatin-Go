package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCaptureQueryParamsRedactsSecrets(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("GET", "/support?ticket=LHTK-ABC123&email=user%2Btag%40example.com&code=secret&access_token=bearer&otp_code=123456&page=2", nil)

	got := captureQueryParams(ctx)
	for _, secret := range []string{"user+tag@example.com", "secret", "bearer", "123456"} {
		if strings.Contains(got, secret) {
			t.Fatalf("query log leaked %q: %s", secret, got)
		}
	}
	if !strings.Contains(got, `"ticket":"LHTK-ABC123"`) || !strings.Contains(got, `"page":"2"`) {
		t.Fatalf("non-sensitive query fields missing: %s", got)
	}
}

func TestSanitizeBodiesRedactsNestedSupportSecrets(t *testing.T) {
	body := []byte(`{"email":"user@example.com","code":"access-code","data":{"challenge_token":"challenge","ticket_code":"LHTK-ABC123"},"items":[{"otp_code":"123456"}]}`)
	got := sanitizeRequestBody(body, "application/json")

	for _, secret := range []string{"user@example.com", "access-code", `:"challenge"`, "123456"} {
		if strings.Contains(got, secret) {
			t.Fatalf("body log leaked %q: %s", secret, got)
		}
	}
	if !strings.Contains(got, "LHTK-ABC123") {
		t.Fatalf("non-secret ticket identifier missing: %s", got)
	}
}

func TestSanitizeResponseBodyOmitsAttachments(t *testing.T) {
	headers := make(http.Header)
	headers.Set("Content-Type", "application/pdf")
	headers.Set("Content-Disposition", `attachment; filename="private.pdf"`)

	got := sanitizeResponseBody([]byte("private attachment bytes"), headers)
	if strings.Contains(got, "private attachment bytes") || !strings.Contains(got, `"omitted":true`) {
		t.Fatalf("attachment response was not omitted: %s", got)
	}
}

func TestExtractHeadersInfoRedactsCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("GET", "/support", nil)
	ctx.Request.Header.Set("Authorization", "Bearer secret-token")
	ctx.Request.Header.Set("Cookie", "support_session=cookie-secret")
	ctx.Request.Header.Set("X-Support-Access-Token", "support-secret")
	ctx.Request.Header.Set("User-Agent", "safe-agent")

	got := extractHeadersInfo(ctx)
	for _, secret := range []string{"secret-token", "cookie-secret", "support-secret"} {
		if strings.Contains(got, secret) {
			t.Fatalf("header log leaked %q: %s", secret, got)
		}
	}
	if !strings.Contains(got, "safe-agent") {
		t.Fatalf("non-sensitive header missing: %s", got)
	}
}

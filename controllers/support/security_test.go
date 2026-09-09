package support

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/adehusnim37/lihatin-go/internal/pkg/auth"
	supportmodel "github.com/adehusnim37/lihatin-go/models/support"
	"github.com/gin-gonic/gin"
)

func TestIsPublicTicketClosedNormalizesStatus(t *testing.T) {
	for _, status := range []string{"closed", " CLOSED ", "resolved", " Resolved "} {
		if !isPublicTicketClosed(status) {
			t.Fatalf("expected %q to be closed", status)
		}
	}
	for _, status := range []string{"open", " in_progress ", ""} {
		if isPublicTicketClosed(status) {
			t.Fatalf("expected %q to remain active", status)
		}
	}
}

func TestSupportAccessTokenFromRequestNeverReadsURLOrForm(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest("POST", "http://localhost/v1/support/tickets/LHTK-ABC123/messages?access_token=query-secret", strings.NewReader("access_token=form-secret"))
	ctx.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	ctx.Request.AddCookie(&http.Cookie{Name: supportAccessCookieName, Value: "cookie-token"})

	if got := supportAccessTokenFromRequest(ctx); got != "cookie-token" {
		t.Fatalf("token = %q, want cookie token", got)
	}

	ctx.Request.Header.Set(supportAccessTokenHeader, "header-token")
	if got := supportAccessTokenFromRequest(ctx); got != "header-token" {
		t.Fatalf("token = %q, want header token", got)
	}

	ctx.Request.Header.Del(supportAccessTokenHeader)
	ctx.Request.Header.Set("Cookie", "")
	if got := supportAccessTokenFromRequest(ctx); got != "" {
		t.Fatalf("query token must be ignored, got %q", got)
	}
}

func TestSupportAccessPayloadMatchesOnlyItsTicketAndAccessVersion(t *testing.T) {
	ticket := &supportmodel.SupportTicket{
		ID:                   "ticket-a",
		TicketCode:           "LHTK-ABC123",
		Email:                "User@Example.com",
		PublicAccessCodeHash: "version-1",
	}
	payload := &auth.SupportAccessTokenPayload{
		TicketID:      "ticket-a",
		TicketCode:    "lhtk-abc123",
		Email:         "user@example.com",
		AccessVersion: "version-1",
	}
	if !supportAccessPayloadMatchesTicket(payload, ticket) {
		t.Fatal("expected matching payload to be accepted")
	}

	payload.TicketID = "ticket-b"
	if supportAccessPayloadMatchesTicket(payload, ticket) {
		t.Fatal("token for ticket B must not authorize ticket A")
	}
	payload.TicketID = "ticket-a"
	payload.AccessVersion = "revoked-version"
	if supportAccessPayloadMatchesTicket(payload, ticket) {
		t.Fatal("revoked access version must not authorize the ticket")
	}
}

func TestSetSupportAccessCookieIsHttpOnlyAndScoped(t *testing.T) {
	t.Setenv("ENV", "development")
	t.Setenv("DOMAIN", "")
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest("POST", "http://localhost/v1/support/access/verify-code", nil)

	if err := setSupportAccessCookie(ctx, "opaque-token", "LHTK-ABC123"); err != nil {
		t.Fatalf("set support cookie: %v", err)
	}
	cookies := recorder.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %d, want 1", len(cookies))
	}
	cookie := cookies[0]
	if cookie.Name != supportAccessCookieName || cookie.Value != "opaque-token" || !cookie.HttpOnly || cookie.Path != "/v1/support/tickets/LHTK-ABC123" {
		t.Fatalf("unexpected support cookie: %#v", cookie)
	}
}

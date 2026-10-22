package support

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCredentialResetHandlerRejectsPublicAndMissingProof(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name, role, body string
		status           int
	}{
		{"user cannot reset", "user", `{"method":"totp","approved":true,"reason":"Reviewed the recovery ownership evidence","ticket_code":"LHTK-ABC123","verification_token":"` + strings.Repeat("a", 64) + `"}`, 403},
		{"admin needs MFA proof", "admin", `{"method":"totp","approved":true,"reason":"Reviewed the recovery ownership evidence","ticket_code":"LHTK-ABC123","verification_token":"` + strings.Repeat("a", 64) + `"}`, 401},
		{"approval never defaults to yes", "admin", `{"method":"totp","reason":"Reviewed the recovery ownership evidence","ticket_code":"LHTK-ABC123","verification_token":"` + strings.Repeat("a", 64) + `"}`, 400},
		{"false approval rejected", "admin", `{"method":"totp","approved":false,"reason":"Reviewed the recovery ownership evidence","ticket_code":"LHTK-ABC123","verification_token":"` + strings.Repeat("a", 64) + `"}`, 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(response)
			ctx.Request = httptest.NewRequest("POST", "/credential-reset", strings.NewReader(tc.body))
			ctx.Request.Header.Set("Content-Type", "application/json")
			ctx.Set("role", tc.role)
			ctx.Set("user_id", "admin")
			(&Controller{}).ResetCredentials(ctx)
			if response.Code != tc.status {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
		})
	}
}

package csrf

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestDefaultErrorHandlerUsesAPIErrorMap(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	defaultErrorHandler(ctx)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", recorder.Code)
	}

	var response struct {
		Success bool              `json:"success"`
		Message string            `json:"message"`
		Error   map[string]string `json:"error"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Success {
		t.Fatal("CSRF error response must not be successful")
	}
	if response.Error["csrf"] == "" {
		t.Fatalf("expected csrf error key, got %#v", response.Error)
	}
}

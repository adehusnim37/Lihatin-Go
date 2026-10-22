package support

import (
	"net/http"

	apperrors "github.com/adehusnim37/lihatin-go/internal/pkg/errors"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
)

// validateSupportMessageBody checks trimmed user prose before attachment uploads.
// An empty body is allowed here; the handler requires an attachment in that case.
func validateSupportMessageBody(body string) error {
	if len(body) > maxSupportMessageBodyLength {
		return apperrors.NewAppError("SUPPORT_MESSAGE_TOO_LONG", "Message must be less than or equal to 5000 characters", http.StatusBadRequest, "body")
	}
	if !validator.IsMeaningfulText(body) {
		return apperrors.NewAppError("SUPPORT_MESSAGE_NOT_MEANINGFUL", "Pesan tidak boleh berupa teks acak/tidak bermakna", http.StatusBadRequest, "body")
	}
	return nil
}

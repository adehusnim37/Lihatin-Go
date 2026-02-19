package email

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
	apperrors "github.com/adehusnim37/lihatin-go/internal/pkg/errors"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/gin-gonic/gin"
)

// UndoChangeEmail reverts the email change process using the provided token
func (c *Controller) UndoChangeEmail(ctx *gin.Context) {
	token := ctx.Query("token")
	frontendURL := config.GetEnvOrDefault(config.EnvFrontendURL, "http://localhost:3000")

	if token == "" {
		logger.Logger.Warn("Undo email change attempted without token")
		ctx.Redirect(http.StatusFound, frontendURL+"/auth/revoke-email-change?status=error&error=token_required")
		return
	}

	// Call repository to undo the email change
	email, username, err := c.repo.GetUserAuthRepository().UndoChangeEmail(token)
	if err != nil {
		logger.Logger.Error("Failed to undo email change",
			"token", token,
			"error", err.Error(),
		)
		ctx.Redirect(http.StatusFound, fmt.Sprintf("%s/auth/revoke-email-change?status=error&error=%s", frontendURL, mapUndoChangeEmailError(err)))
		return
	}

	if err := c.emailService.SendSuccessRetrieveEmail(email, username); err != nil {
		logger.Logger.Error("Failed to send email change revoke success email",
			"username", username,
			"email", email,
			"error", err.Error(),
		)
		// Revocation has already succeeded. Keep user flow successful.
		logger.Logger.Warn("Email change revoked but success notification email failed to send")
	}

	logger.Logger.Info("Email change revoked successfully", "token", token)

	// Redirect to frontend status page
	ctx.Redirect(http.StatusFound, frontendURL+"/auth/revoke-email-change?status=success")
}

func mapUndoChangeEmailError(err error) string {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		switch appErr.Code {
		case "REVOKE_TOKEN_EXPIRED":
			return "token_expired"
		case "REVOKE_TOKEN_NOT_FOUND":
			return "token_not_found"
		case "INVALID_ACTION_TYPE":
			return "invalid_request"
		}
	}

	return "revoke_failed"
}

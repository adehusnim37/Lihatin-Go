package email

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/gin-gonic/gin"
)

// UndoChangeEmail reverts the email change process using the provided token
func (c *Controller) UndoChangeEmail(ctx *gin.Context) {
	token := ctx.Query("token")

	if token == "" {
		utils.Logger.Warn("Undo email change attempted without token")
		utils.SendErrorResponse(ctx, http.StatusBadRequest, "TOKEN_REQUIRED", "Revoke token is required", "token", nil)
		return
	}

	// Call repository to undo the email change
	email, username, err := c.repo.GetUserAuthRepository().UndoChangeEmail(token)
	if err != nil {
		utils.Logger.Error("Failed to undo email change",
			"token", token,
			"error", err.Error(),
		)
		utils.HandleError(ctx, err, nil)
		return
	}

	if err := c.emailService.SendSuccessRetrieveEmail(email, username); err != nil {
		utils.Logger.Error("Failed to send email change revoke success email",
			"username", username,
			"email", email,
			"error", err.Error(),
		)
		utils.SendErrorResponse(ctx, http.StatusInternalServerError, "EMAIL_SEND_FAILED", "Failed to send email notification", "email", email)
		return
	}

	utils.Logger.Info("Email change revoked successfully", "token", token)

	// Send success response
	utils.SendOKResponse(ctx, map[string]interface{}{
		"message": "Email change has been revoked. Your original email has been restored and verified.",
	}, "Email change revoked successfully")
}

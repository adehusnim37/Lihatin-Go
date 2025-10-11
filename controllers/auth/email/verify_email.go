package email

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/models/user"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/gin-gonic/gin"
)

// VerifyEmail verifies email with the provided token
func (c *Controller) VerifyEmail(ctx *gin.Context) {
	var response dto.VerifyEmailResponse
	token := ctx.Query("token")
	if token == "" {
		utils.Logger.Warn("Email verification attempted without token")
		utils.SendErrorResponse(ctx, http.StatusBadRequest, "TOKEN_REQUIRED", "Verification token is required", "token", nil)
		return
	}

	// Verify token
	response, err := c.repo.GetUserAuthRepository().VerifyEmail(token)
	if err != nil {
		utils.Logger.Error("Email verification failed",
			"token", token,
			"error", err.Error(),
		)
		utils.HandleError(ctx, err, nil)
		return
	}

	// Send appropriate notification based on verification source
	switch response.Source {
	case user.EmailSourceSignup:
		// For signup, send success verification email
		if err := c.emailService.SendSuccessEmailVerification(response.Email, response.Username); err != nil {
			utils.Logger.Error("Failed to send verification email",
				"username", response.Username,
				"email", response.Email,
				"error", err.Error(),
			)
			utils.SendErrorResponse(ctx, http.StatusInternalServerError, "EMAIL_SEND_FAILED", "Failed to send verification email", "email", response.Email)
			return
		}
	case user.EmailSourceChange:
		// For email change, send notification to OLD email with revoke link
		if response.OldEmail != "" && response.Token != "" {
			if err := c.emailService.SendChangeOldEmailNotification(response.OldEmail, response.Email, response.Username, response.Token); err != nil {
				utils.Logger.Error("Failed to send change email notification to old email",
					"username", response.Username,
					"old_email", response.OldEmail,
					"new_email", response.Email,
					"error", err.Error(),
				)
				// Don't fail the verification if notification fails, just log it
				utils.Logger.Warn("Email verification succeeded but notification to old email failed")
			}
		}

		// Also send success notification to NEW email
		if err := c.emailService.SendSuccessEmailVerification(response.Email, response.Username); err != nil {
			utils.Logger.Error("Failed to send success verification to new email",
				"username", response.Username,
				"email", response.Email,
				"error", err.Error(),
			)
			// Don't fail, just log
			utils.Logger.Warn("Email verification succeeded but success notification failed")
		}
	}

	utils.Logger.Info("Email verified successfully", "token", token)

	// Send success response
	utils.SendOKResponse(ctx, map[string]interface{}{
		"message": "Email verification completed successfully",
	}, "Email verified successfully")
}

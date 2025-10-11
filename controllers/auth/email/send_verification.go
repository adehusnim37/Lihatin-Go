package email

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/models/user"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/gin-gonic/gin"
)

// SendVerificationEmails sends verification emails to user
func (c *Controller) SendVerificationEmails(ctx *gin.Context) {
	userID := ctx.GetString("user_id")
	userEmail := ctx.GetString("email")
	userFirstName := ctx.GetString("username")

	// Add validation for required fields
	if userID == "" || userEmail == "" {
		utils.SendErrorResponse(ctx, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated", "auth", userID)
		return
	}

	// Generate verification token
	token, err := utils.GenerateVerificationToken()
	if err != nil {
		utils.Logger.Error("Failed to generate verification token",
			"user_id", userID,
			"error", err.Error(),
		)
		utils.SendErrorResponse(ctx, http.StatusInternalServerError, "TOKEN_GENERATION_FAILED", "Failed to generate verification token", "token", userID)
		return
	}

	// Save token to database
	if err := c.repo.GetUserAuthRepository().SetEmailVerificationToken(userID, token, user.EmailSourceSignup); err != nil {
		utils.Logger.Error("Failed to save verification token",
			"user_id", userID,
			"error", err.Error(),
		)
		utils.SendErrorResponse(ctx, http.StatusInternalServerError, "TOKEN_SAVE_FAILED", "Failed to save verification token", "database", userID)
		return
	}

	// Send verification email using injected service
	if err := c.emailService.SendVerificationEmail(userEmail, userFirstName, token); err != nil {
		utils.Logger.Error("Failed to send verification email",
			"user_id", userID,
			"email", userEmail,
			"error", err.Error(),
		)
		utils.SendErrorResponse(ctx, http.StatusInternalServerError, "EMAIL_SEND_FAILED", "Failed to send verification email", "email", userID)
		return
	}

	utils.Logger.Info("Verification email sent successfully",
		"user_id", userID,
		"email", userEmail,
	)

	// Send success response
	utils.SendOKResponse(ctx, map[string]interface{}{
		"sent_to": userEmail,
		"message": "Please check your email and click the verification link",
	}, "Verification email sent successfully")
}

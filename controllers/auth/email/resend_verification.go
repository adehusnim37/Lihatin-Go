package email

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/gin-gonic/gin"
)

// ResendVerificationEmail resends verification email to user
func (c *Controller) ResendVerificationEmail(ctx *gin.Context) {
	userID := ctx.GetString("user_id")
	userEmail := ctx.GetString("email")
	userFirstName := ctx.GetString("username")

	// Add validation for required fields
	if userID == "" || userEmail == "" {
		utils.SendErrorResponse(ctx, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated", "auth", userID)
		return
	}

	// Check if user email is already verified
	userAuth, err := c.repo.GetUserAuthRepository().GetUserAuthByUserID(userID)
	if err != nil {
		utils.Logger.Error("Failed to get user auth info",
			"user_id", userID,
			"error", err.Error(),
		)
		utils.HandleError(ctx, err, userID)
		return
	}

	if userAuth.IsEmailVerified {
		utils.Logger.Warn("Attempting to resend verification to already verified email",
			"user_id", userID,
			"email", userEmail,
		)
		utils.SendErrorResponse(ctx, http.StatusBadRequest, "EMAIL_ALREADY_VERIFIED", "Email is already verified", "email", userID)
		return
	}

	// Generate new verification token
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
	if err := c.repo.GetUserAuthRepository().SetEmailVerificationToken(userID, token); err != nil {
		utils.Logger.Error("Failed to save verification token",
			"user_id", userID,
			"error", err.Error(),
		)
		utils.SendErrorResponse(ctx, http.StatusInternalServerError, "TOKEN_SAVE_FAILED", "Failed to save verification token", "database", userID)
		return
	}

	// Send verification email
	if err := c.emailService.SendVerificationEmail(userEmail, userFirstName, token); err != nil {
		utils.Logger.Error("Failed to resend verification email",
			"user_id", userID,
			"email", userEmail,
			"error", err.Error(),
		)
		utils.SendErrorResponse(ctx, http.StatusInternalServerError, "EMAIL_SEND_FAILED", "Failed to resend verification email", "email", userID)
		return
	}

	utils.Logger.Info("Verification email resent successfully",
		"user_id", userID,
		"email", userEmail,
	)

	// Send success response
	utils.SendOKResponse(ctx, map[string]interface{}{
		"sent_to": userEmail,
		"message": "Verification email has been resent. Please check your email.",
	}, "Verification email resent successfully")
}

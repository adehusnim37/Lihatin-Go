package totp

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/models/user"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/gin-gonic/gin"
)

// VerifyTOTP verifies TOTP code and enables 2FA
func (c *Controller) VerifyTOTP(ctx *gin.Context) {
	userID := ctx.GetString("user_id")

	if userID == "" {
		utils.SendErrorResponse(ctx, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated", "auth", userID)
		return
	}

	// Bind request
	var req dto.VerifyTOTPRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.SendValidationError(ctx, err, &req)
		return
	}

	// Get user auth info
	userAuth, err := c.repo.GetUserAuthRepository().GetUserAuthByUserID(userID)
	if err != nil {
		utils.Logger.Error("Failed to get user auth info",
			"user_id", userID,
			"error", err.Error(),
		)
		utils.HandleError(ctx, err, userID)
		return
	}
	// Verify TOTP code
	encryptedSecret, err := c.repo.GetAuthMethodRepository().GetTOTPSecret(userAuth.ID)
	if err != nil {
		utils.Logger.Error("Failed to get TOTP secret",
			"user_id", userID,
			"error", err.Error(),
		)
		utils.HandleError(ctx, err, userID)
		return
	}

	secret, err := utils.DecryptTOTPSecret(encryptedSecret)
	if err != nil {
		utils.Logger.Error("Failed to decrypt TOTP secret",
			"user_id", userID,
			"error", err.Error(),
		)
		utils.HandleError(ctx, err, userID)
		return
	}

	// Validate TOTP code
	if !utils.ValidateTOTPCodeWithWindow(secret, req.TOTPCode, 1) {
		utils.Logger.Warn("Invalid TOTP code provided for verification",
			"user_id", userID,
		)
		utils.SendErrorResponse(ctx, http.StatusBadRequest, "INVALID_TOTP", "Invalid TOTP code", "totp", userID)
		return
	}

	// Get TOTP auth method to verify it
	totpMethod, err := c.repo.GetAuthMethodRepository().GetAuthMethodByType(userAuth.ID, user.AuthMethodTypeTOTP)
	if err != nil {
		utils.Logger.Error("Failed to get TOTP method",
			"user_id", userID,
			"error", err.Error(),
		)
		utils.SendErrorResponse(ctx, http.StatusInternalServerError, "TOTP_METHOD_NOT_FOUND", "Failed to get TOTP method", "totp", userID)
		return
	}

	// Mark TOTP as verified
	if err := c.repo.GetAuthMethodRepository().VerifyAuthMethod(totpMethod.ID); err != nil {
		utils.Logger.Error("Failed to verify TOTP method",
			"user_id", userID,
			"error", err.Error(),
		)
		utils.SendErrorResponse(ctx, http.StatusInternalServerError, "TOTP_VERIFY_FAILED", "Failed to verify TOTP method", "totp", userID)
		return
	}

	// Get user for email notification
	user, err := c.repo.GetUserRepository().GetUserByID(userID)
	if err == nil {
		// Send TOTP setup confirmation email (async)
		go c.emailService.SendTOTPSetupEmail(user.Email, user.FirstName)
	}

	utils.SendSuccessResponse(ctx, http.StatusOK, map[string]interface{}{"totp_enabled": true}, "TOTP verified and enabled successfully")
}

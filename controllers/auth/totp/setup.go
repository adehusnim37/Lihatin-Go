package totp

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/gin-gonic/gin"
)

// SetupTOTP generates TOTP secret and QR code for user
func (c *Controller) SetupTOTP(ctx *gin.Context) {
	userID := ctx.GetString("user_id")
	userEmail := ctx.GetString("email")

	if userID == "" {
		utils.SendErrorResponse(ctx, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated", "auth", userID)
		return
	}

	// Check if TOTP is already enabled
	userAuth, err := c.repo.GetUserAuthRepository().GetUserAuthByUserID(userID)
	if err != nil {
		utils.Logger.Error("Failed to get user auth info",
			"user_id", userID,
			"error", err.Error(),
		)
		utils.HandleError(ctx, err, userID)
		return
	}

	if userAuth.IsTOTPEnabled {
		utils.Logger.Warn("TOTP setup attempted for user who already has TOTP enabled",
			"user_id", userID,
		)
		utils.SendErrorResponse(ctx, http.StatusBadRequest, "TOTP_ALREADY_ENABLED", "TOTP is already enabled for this account", "totp", userID)
		return
	}

	// Generate TOTP secret
	secret, err := utils.GenerateTOTPSecret("Lihatin", userEmail)
	if err != nil {
		utils.Logger.Error("Failed to generate TOTP secret",
			"user_id", userID,
			"error", err.Error(),
		)
		utils.SendErrorResponse(ctx, http.StatusInternalServerError, "TOTP_GENERATION_FAILED", "Failed to generate TOTP secret", "totp", userID)
		return
	}

	// Generate QR code
	encryptedSecret, err := utils.EncryptTOTPSecret(secret.Secret)
	if err != nil {
		utils.Logger.Error("Failed to generate TOTP QR code",
			"user_id", userID,
			"error", err.Error(),
		)
		utils.SendErrorResponse(ctx, http.StatusInternalServerError, "QR_GENERATION_FAILED", "Failed to generate QR code", "totp", userID)
		return
	}

	// Generate recovery codes
	recoveryCodes, err := utils.GenerateRecoveryCodes(9)
	if err != nil {
		utils.Logger.Error("Failed to generate recovery codes",
			"user_id", userID,
			"error", err.Error(),
		)
		utils.SendErrorResponse(ctx, http.StatusInternalServerError, "RECOVERY_CODES_FAILED", "Failed to generate recovery codes", "totp", userID)
		return
	}

	hashedRecoveryCodes := utils.HashRecoveryCodes(recoveryCodes)

	// Setup TOTP in database (but not verified yet)
	err = c.repo.GetAuthMethodRepository().SetupTOTP(
		userAuth.ID,
		encryptedSecret,
		hashedRecoveryCodes,
		"Authenticator App",
	)
	if err != nil {
		utils.SendErrorResponse(ctx, http.StatusInternalServerError, "TOTP_SETUP_FAILED", "Failed to setup TOTP", "database", userID)
		return
	}

	// Generate QR code URL
	qrCodeURL := utils.GenerateQRCodeURL(secret)

	response := map[string]any{
		"secret":         secret.Secret,
		"qr_code_url":    qrCodeURL,
		"recovery_codes": recoveryCodes,
	}

	utils.SendOKResponse(ctx, response, "TOTP setup initiated. Please scan QR code and verify to complete setup")
}

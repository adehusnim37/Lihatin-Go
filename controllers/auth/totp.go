package auth

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/models"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/gin-gonic/gin"
)

// SetupTOTP initiates TOTP setup for a user
func (c *Controller) SetupTOTP(ctx *gin.Context) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Authentication required",
			Error:   map[string]string{"auth": "Please login to access this feature"},
		})
		return
	}

	// Get user information
	user, err := c.repo.GetUserRepository().GetUserByID(userID.(string))
	if err != nil {
		ctx.JSON(http.StatusNotFound, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "User not found",
			Error:   map[string]string{"user": "User not found"},
		})
		return
	}

	// Get user auth data
	userAuth, err := c.repo.GetUserAuthRepository().GetUserAuthByUserID(user.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to get auth data",
			Error:   map[string]string{"error": "Internal server error"},
		})
		return
	}

	// Check if TOTP is already enabled
	hasTOTP, err := c.repo.GetAuthMethodRepository().HasTOTPEnabled(userAuth.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to check TOTP status",
			Error:   map[string]string{"error": "Internal server error"},
		})
		return
	}

	if hasTOTP {
		ctx.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "TOTP already enabled",
			Error:   map[string]string{"totp": "Two-factor authentication is already enabled"},
		})
		return
	}

	// Generate TOTP secret
	totpConfig, err := utils.GenerateTOTPSecret("Lihatin", user.Email)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to generate TOTP secret",
			Error:   map[string]string{"error": "Internal server error"},
		})
		return
	}

	// Encrypt secret for storage
	encryptedSecret, err := utils.EncryptTOTPSecret(totpConfig.Secret)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to encrypt secret",
			Error:   map[string]string{"error": "Internal server error"},
		})
		return
	}

	// Generate recovery codes
	recoveryCodes, err := utils.GenerateRecoveryCodes(8)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to generate recovery codes",
			Error:   map[string]string{"error": "Internal server error"},
		})
		return
	}

	// Hash recovery codes for storage
	hashedRecoveryCodes := utils.HashRecoveryCodes(recoveryCodes)

	// Setup TOTP in database (but not verified yet)
	err = c.repo.GetAuthMethodRepository().SetupTOTP(
		userAuth.ID,
		encryptedSecret,
		hashedRecoveryCodes,
		"Authenticator App",
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to setup TOTP",
			Error:   map[string]string{"error": "Internal server error"},
		})
		return
	}

	// Generate QR code URL
	qrCodeURL := utils.GenerateQRCodeURL(totpConfig)

	ctx.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"secret":         utils.FormatTOTPSecret(totpConfig.Secret),
			"qr_code_url":    qrCodeURL,
			"recovery_codes": recoveryCodes,
			"backup_codes":   recoveryCodes, // Alternative name
		},
		Message: "TOTP setup initiated. Please verify with your authenticator app.",
		Error:   nil,
	})
}

// VerifyTOTP verifies TOTP code during setup
func (c *Controller) VerifyTOTP(ctx *gin.Context) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Authentication required",
			Error:   map[string]string{"auth": "Please login to access this feature"},
		})
		return
	}

	var req struct {
		Code string `json:"code" validate:"required,len=6"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid input",
			Error:   map[string]string{"input": "TOTP code is required"},
		})
		return
	}

	if err := c.Validate.Struct(req); err != nil {
		ctx.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Validation failed",
			Error:   map[string]string{"code": "TOTP code must be 6 digits"},
		})
		return
	}

	// Get user auth data
	userAuth, err := c.repo.GetUserAuthRepository().GetUserAuthByUserID(userID.(string))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to get auth data",
			Error:   map[string]string{"error": "Internal server error"},
		})
		return
	}

	// Get TOTP secret
	encryptedSecret, err := c.repo.GetAuthMethodRepository().GetTOTPSecret(userAuth.ID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "TOTP not found",
			Error:   map[string]string{"totp": "TOTP setup not found. Please setup TOTP first."},
		})
		return
	}

	// Decrypt secret
	secret, err := utils.DecryptTOTPSecret(encryptedSecret)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to decrypt secret",
			Error:   map[string]string{"error": "Internal server error"},
		})
		return
	}

	// Validate TOTP code
	if !utils.ValidateTOTPCodeWithWindow(secret, req.Code, 1) {
		ctx.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid TOTP code",
			Error:   map[string]string{"code": "Invalid verification code"},
		})
		return
	}

	// Get TOTP auth method to verify it
	totpMethod, err := c.repo.GetAuthMethodRepository().GetAuthMethodByType(userAuth.ID, models.AuthMethodTypeTOTP)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to get TOTP method",
			Error:   map[string]string{"error": "Internal server error"},
		})
		return
	}

	// Mark TOTP as verified
	if err := c.repo.GetAuthMethodRepository().VerifyAuthMethod(totpMethod.ID); err != nil {
		ctx.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to verify TOTP",
			Error:   map[string]string{"error": "Internal server error"},
		})
		return
	}

	// Get user for email notification
	user, err := c.repo.GetUserRepository().GetUserByID(userID.(string))
	if err == nil {
		// Send TOTP setup confirmation email (async)
		go c.emailService.SendTOTPSetupEmail(user.Email, user.FirstName)
	}

	ctx.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    nil,
		Message: "Two-factor authentication enabled successfully",
		Error:   nil,
	})
}

// ValidateTOTPLogin validates TOTP during login process
func (c *Controller) ValidateTOTPLogin(ctx *gin.Context) {
	var req struct {
		Token string `json:"token" validate:"required"`
		Code  string `json:"code" validate:"required,len=6"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid input",
			Error:   map[string]string{"input": "Token and TOTP code are required"},
		})
		return
	}

	if err := c.Validate.Struct(req); err != nil {
		ctx.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Validation failed",
			Error:   map[string]string{"code": "TOTP code must be 6 digits"},
		})
		return
	}

	// Validate JWT token
	claims, err := utils.ValidateJWT(req.Token)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid token",
			Error:   map[string]string{"token": "Invalid or expired token"},
		})
		return
	}

	// Get user auth data
	userAuth, err := c.repo.GetUserAuthRepository().GetUserAuthByUserID(claims.UserID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to get auth data",
			Error:   map[string]string{"error": "Internal server error"},
		})
		return
	}

	// Get TOTP secret
	encryptedSecret, err := c.repo.GetAuthMethodRepository().GetTOTPSecret(userAuth.ID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "TOTP not enabled",
			Error:   map[string]string{"totp": "Two-factor authentication is not enabled"},
		})
		return
	}

	// Decrypt secret
	secret, err := utils.DecryptTOTPSecret(encryptedSecret)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to decrypt secret",
			Error:   map[string]string{"error": "Internal server error"},
		})
		return
	}

	// Validate TOTP code
	if !utils.ValidateTOTPCodeWithWindow(secret, req.Code, 1) {
		// Try recovery codes
		recoveryCodes, err := c.repo.GetAuthMethodRepository().GetRecoveryCodes(userAuth.ID)
		if err == nil && utils.ValidateRecoveryCode(req.Code, recoveryCodes) {
			// Remove used recovery code
			updatedCodes := utils.RemoveRecoveryCode(recoveryCodes, req.Code)
			c.repo.GetAuthMethodRepository().UpdateRecoveryCodes(userAuth.ID, updatedCodes)
		} else {
			ctx.JSON(http.StatusBadRequest, models.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Invalid code",
				Error:   map[string]string{"code": "Invalid verification or recovery code"},
			})
			return
		}
	}

	// Update last used timestamp
	totpMethod, err := c.repo.GetAuthMethodRepository().GetAuthMethodByType(userAuth.ID, models.AuthMethodTypeTOTP)
	if err == nil {
		c.repo.GetAuthMethodRepository().UpdateLastUsed(totpMethod.ID)
	}

	ctx.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    map[string]interface{}{"authenticated": true},
		Message: "Two-factor authentication successful",
		Error:   nil,
	})
}

// DisableTOTP disables TOTP for a user
func (c *Controller) DisableTOTP(ctx *gin.Context) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Authentication required",
			Error:   map[string]string{"auth": "Please login to access this feature"},
		})
		return
	}

	var req struct {
		Password string `json:"password" validate:"required"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid input",
			Error:   map[string]string{"input": "Password confirmation is required"},
		})
		return
	}

	// Get user auth data
	userAuth, err := c.repo.GetUserAuthRepository().GetUserAuthByUserID(userID.(string))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to get auth data",
			Error:   map[string]string{"error": "Internal server error"},
		})
		return
	}

	// Verify password
	if err := utils.CheckPassword(userAuth.PasswordHash, req.Password); err != nil {
		ctx.JSON(http.StatusUnauthorized, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid password",
			Error:   map[string]string{"password": "Password is incorrect"},
		})
		return
	}

	// Get TOTP method
	totpMethod, err := c.repo.GetAuthMethodRepository().GetAuthMethodByType(userAuth.ID, models.AuthMethodTypeTOTP)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "TOTP not enabled",
			Error:   map[string]string{"totp": "Two-factor authentication is not enabled"},
		})
		return
	}

	// Delete TOTP method
	if err := c.repo.GetAuthMethodRepository().DeleteAuthMethod(totpMethod.ID); err != nil {
		ctx.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to disable TOTP",
			Error:   map[string]string{"error": "Internal server error"},
		})
		return
	}

	ctx.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    nil,
		Message: "Two-factor authentication disabled successfully",
		Error:   nil,
	})
}

// GetTOTPStatus returns TOTP status for a user
func (c *Controller) GetTOTPStatus(ctx *gin.Context) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Authentication required",
			Error:   map[string]string{"auth": "Please login to access this feature"},
		})
		return
	}

	// Get user auth data
	userAuth, err := c.repo.GetUserAuthRepository().GetUserAuthByUserID(userID.(string))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to get auth data",
			Error:   map[string]string{"error": "Internal server error"},
		})
		return
	}

	// Check if TOTP is enabled
	hasTOTP, err := c.repo.GetAuthMethodRepository().HasTOTPEnabled(userAuth.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to check TOTP status",
			Error:   map[string]string{"error": "Internal server error"},
		})
		return
	}

	var totpData map[string]interface{}
	if hasTOTP {
		// Get TOTP method details
		totpMethod, err := c.repo.GetAuthMethodRepository().GetAuthMethodByType(userAuth.ID, models.AuthMethodTypeTOTP)
		if err == nil {
			recoveryCodes, _ := c.repo.GetAuthMethodRepository().GetRecoveryCodes(userAuth.ID)
			totpData = map[string]interface{}{
				"enabled":             true,
				"verified":            totpMethod.IsVerified,
				"friendly_name":       totpMethod.FriendlyName,
				"last_used":           totpMethod.LastUsedAt,
				"recovery_codes_left": len(recoveryCodes),
			}
		}
	} else {
		totpData = map[string]interface{}{
			"enabled": false,
		}
	}

	ctx.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    totpData,
		Message: "TOTP status retrieved successfully",
		Error:   nil,
	})
}

// RegenerateRecoveryCodes generates new recovery codes
func (c *Controller) RegenerateRecoveryCodes(ctx *gin.Context) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Authentication required",
			Error:   map[string]string{"auth": "Please login to access this feature"},
		})
		return
	}

	var req struct {
		Password string `json:"password" validate:"required"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid input",
			Error:   map[string]string{"input": "Password confirmation is required"},
		})
		return
	}

	// Get user auth data
	userAuth, err := c.repo.GetUserAuthRepository().GetUserAuthByUserID(userID.(string))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to get auth data",
			Error:   map[string]string{"error": "Internal server error"},
		})
		return
	}

	// Verify password
	if err := utils.CheckPassword(userAuth.PasswordHash, req.Password); err != nil {
		ctx.JSON(http.StatusUnauthorized, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid password",
			Error:   map[string]string{"password": "Password is incorrect"},
		})
		return
	}

	// Check if TOTP is enabled
	hasTOTP, err := c.repo.GetAuthMethodRepository().HasTOTPEnabled(userAuth.ID)
	if err != nil || !hasTOTP {
		ctx.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "TOTP not enabled",
			Error:   map[string]string{"totp": "Two-factor authentication is not enabled"},
		})
		return
	}

	// Generate new recovery codes
	recoveryCodes, err := utils.GenerateRecoveryCodes(8)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to generate recovery codes",
			Error:   map[string]string{"error": "Internal server error"},
		})
		return
	}

	// Hash recovery codes for storage
	hashedRecoveryCodes := utils.HashRecoveryCodes(recoveryCodes)

	// Update recovery codes in database
	if err := c.repo.GetAuthMethodRepository().UpdateRecoveryCodes(userAuth.ID, hashedRecoveryCodes); err != nil {
		ctx.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to update recovery codes",
			Error:   map[string]string{"error": "Internal server error"},
		})
		return
	}

	ctx.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"recovery_codes": recoveryCodes,
			"backup_codes":   recoveryCodes, // Alternative name
		},
		Message: "Recovery codes regenerated successfully",
		Error:   nil,
	})
}

package totp

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/adehusnim37/lihatin-go/models/user"
	"github.com/adehusnim37/lihatin-go/internal/pkg/auth"
	"github.com/gin-gonic/gin"
)

// ValidateTOTPLogin validates TOTP during login process
func (c *Controller) ValidateTOTPLogin(ctx *gin.Context) {
	var req struct {
		Token string `json:"token" validate:"required"`
		Code  string `json:"code" validate:"required,len=6"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid input",
			Error:   map[string]string{"input": "Token and TOTP code are required"},
		})
		return
	}

	if err := c.Validate.Struct(req); err != nil {
		ctx.JSON(http.StatusBadRequest, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Validation failed",
			Error:   map[string]string{"code": "TOTP code must be 6 digits"},
		})
		return
	}

	// Validate JWT token
	claims, err := auth.ValidateJWT(req.Token)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, common.APIResponse{
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
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
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
		ctx.JSON(http.StatusBadRequest, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "TOTP not enabled",
			Error:   map[string]string{"totp": "Two-factor authentication is not enabled"},
		})
		return
	}

	// Decrypt secret
	secret, err := auth.DecryptTOTPSecret(encryptedSecret)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to decrypt secret",
			Error:   map[string]string{"error": "Internal server error"},
		})
		return
	}

	// Validate TOTP code
	if !auth.ValidateTOTPCodeWithWindow(secret, req.Code, 1) {
		// Try recovery codes
		recoveryCodes, err := c.repo.GetAuthMethodRepository().GetRecoveryCodes(userAuth.ID)
		if err == nil && auth.ValidateRecoveryCode(req.Code, recoveryCodes) {
			// Remove used recovery code
			updatedCodes := auth.RemoveRecoveryCode(recoveryCodes, req.Code)
			c.repo.GetAuthMethodRepository().UpdateRecoveryCodes(userAuth.ID, updatedCodes)
		} else {
			ctx.JSON(http.StatusBadRequest, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Invalid code",
				Error:   map[string]string{"code": "Invalid verification or recovery code"},
			})
			return
		}
	}

	// Update last used timestamp
	totpMethod, err := c.repo.GetAuthMethodRepository().GetAuthMethodByType(userAuth.ID, user.AuthMethodTypeTOTP)
	if err == nil {
		c.repo.GetAuthMethodRepository().UpdateLastUsed(totpMethod.ID)
	}

	ctx.JSON(http.StatusOK, common.APIResponse{
		Success: true,
		Data:    map[string]interface{}{"authenticated": true},
		Message: "Two-factor authentication successful",
		Error:   nil,
	})
}

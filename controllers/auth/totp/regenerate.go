package totp

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/internal/pkg/auth"
	"github.com/adehusnim37/lihatin-go/middleware"
	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/gin-gonic/gin"
)

// RegenerateRecoveryCodes generates new recovery codes
func (c *Controller) RegenerateRecoveryCodes(ctx *gin.Context) {
	userID := ctx.GetString("user_id")

	var req struct {
		dto.DisableTOTPRequest
		VerificationToken string `json:"verification_token"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid input",
			Error:   map[string]string{"input": "Identity verification is required"},
		})
		return
	}

	// Get user auth data
	userAuth, err := c.repo.GetUserAuthRepository().GetUserAuthByUserID(userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to get auth data",
			Error:   map[string]string{"error": "Internal server error"},
		})
		return
	}

	// Require a recent verified action or a current authenticator code.
	valid := false
	if manager := middleware.GetSessionManager(); manager != nil {
		valid = auth.ConsumeSecurityGrant(ctx.Request.Context(), manager.GetRedisClient(), userID, ctx.GetString("session_id"), req.VerificationToken)
	}
	if !valid && req.TOTPCode != "" {
		encrypted, e := c.repo.GetAuthMethodRepository().GetTOTPSecret(userAuth.ID)
		if e == nil {
			secret, e := auth.DecryptTOTPSecret(encrypted)
			valid = e == nil && auth.ValidateTOTPCodeWithWindow(secret, req.TOTPCode, 1)
		}
	}

	if !valid {
		ctx.JSON(http.StatusUnauthorized, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Identity verification failed. Verify your identity again.",
			Error:   map[string]string{"auth": "A current authenticator code or recent verification is required"},
		})
		return
	}

	// Check if TOTP is enabled
	hasTOTP, err := c.repo.GetAuthMethodRepository().HasTOTPEnabled(userAuth.ID)
	if err != nil || !hasTOTP {
		ctx.JSON(http.StatusBadRequest, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "TOTP not enabled",
			Error:   map[string]string{"totp": "Two-factor authentication is not enabled"},
		})
		return
	}

	// Generate new recovery codes
	recoveryCodes, err := auth.GenerateRecoveryCodes(9)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to generate recovery codes",
			Error:   map[string]string{"error": "Internal server error"},
		})
		return
	}

	// Hash recovery codes for storage
	hashedRecoveryCodes := auth.HashRecoveryCodes(recoveryCodes)

	// Update recovery codes in database
	if err := c.repo.GetAuthMethodRepository().UpdateRecoveryCodes(userAuth.ID, hashedRecoveryCodes); err != nil {
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to update recovery codes",
			Error:   map[string]string{"error": "Internal server error"},
		})
		return
	}

	ctx.Header("Cache-Control", "no-store")
	ctx.JSON(http.StatusOK, common.APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"recovery_codes": recoveryCodes,
			"backup_codes":   recoveryCodes, // Alternative name
		},
		Message: "Recovery codes regenerated successfully",
		Error:   nil,
	})
}

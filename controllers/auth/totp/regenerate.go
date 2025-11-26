package totp

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/gin-gonic/gin"
)

// RegenerateRecoveryCodes generates new recovery codes
func (c *Controller) RegenerateRecoveryCodes(ctx *gin.Context) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, common.APIResponse{
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
		ctx.JSON(http.StatusBadRequest, common.APIResponse{
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
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to get auth data",
			Error:   map[string]string{"error": "Internal server error"},
		})
		return
	}

	// Verify password
	if err := utils.CheckPassword(userAuth.PasswordHash, req.Password); err != nil {
		ctx.JSON(http.StatusUnauthorized, common.APIResponse{
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
		ctx.JSON(http.StatusBadRequest, common.APIResponse{
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
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
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
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to update recovery codes",
			Error:   map[string]string{"error": "Internal server error"},
		})
		return
	}

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
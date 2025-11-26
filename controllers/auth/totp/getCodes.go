package totp

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/adehusnim37/lihatin-go/models/user"
	"github.com/gin-gonic/gin"
)

// GetRecoveryCodes returns TOTP recovery codes
func (c *Controller) GetRecoveryCodes(ctx *gin.Context) {
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

	// Get user auth record first
	userAuth, err := c.repo.GetUserAuthRepository().GetUserAuthByUserID(userID.(string))
	if err != nil {
		ctx.JSON(http.StatusNotFound, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "User authentication data not found",
			Error:   map[string]string{"auth": "Authentication data not found"},
		})
		return
	}

	// Get TOTP auth method
	authMethod, err := c.repo.GetAuthMethodRepository().GetAuthMethodByType(userAuth.ID, user.AuthMethodTypeTOTP)
	if err != nil {
		ctx.JSON(http.StatusNotFound, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "No TOTP setup found",
			Error:   map[string]string{"totp": "TOTP is not configured for this account"},
		})
		return
	}

	if !authMethod.IsEnabled {
		ctx.JSON(http.StatusBadRequest, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "TOTP not enabled",
			Error:   map[string]string{"totp": "TOTP must be enabled to view recovery codes"},
		})
		return
	}

	ctx.JSON(http.StatusOK, common.APIResponse{
		Success: true,
		Data: gin.H{
			"recovery_codes": authMethod.RecoveryCodes,
			"codes_count":    len(authMethod.RecoveryCodes),
		},
		Message: "Recovery codes retrieved successfully",
		Error:   nil,
	})
}
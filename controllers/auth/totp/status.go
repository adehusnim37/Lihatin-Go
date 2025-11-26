package totp

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/adehusnim37/lihatin-go/models/user"
	"github.com/gin-gonic/gin"
)

// GetTOTPStatus returns TOTP status for a user
func (c *Controller) GetTOTPStatus(ctx *gin.Context) {
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

	// Check if TOTP is enabled
	hasTOTP, err := c.repo.GetAuthMethodRepository().HasTOTPEnabled(userAuth.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
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
		totpMethod, err := c.repo.GetAuthMethodRepository().GetAuthMethodByType(userAuth.ID, user.AuthMethodTypeTOTP)
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

	ctx.JSON(http.StatusOK, common.APIResponse{
		Success: true,
		Data:    totpData,
		Message: "TOTP status retrieved successfully",
		Error:   nil,
	})
}
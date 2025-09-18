package api

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/gin-gonic/gin"
)

// RevokeAPIKey revokes an API key
func (c *Controller) RevokeAPIKey(ctx *gin.Context) {
	// Get user ID from JWT token context
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Authentication required",
			Error:   map[string]string{"auth": "Please authenticate to access this resource"},
		})
		return
	}

	keyID := ctx.Param("id")
	if keyID == "" {
		ctx.JSON(http.StatusBadRequest, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "API key ID is required",
			Error:   map[string]string{"key_id": "API key ID parameter is required"},
		})
		return
	}

	// Check if API key exists and belongs to the user
	apiKey, err := c.repo.GetAPIKeyRepository().GetAPIKeyByID(keyID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "API key not found",
			Error:   map[string]string{"key_id": "API key with this ID does not exist"},
		})
		return
	}

	// Check ownership
	if apiKey.UserID != userID.(string) {
		ctx.JSON(http.StatusForbidden, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Access denied",
			Error:   map[string]string{"auth": "You can only revoke your own API keys"},
		})
		return
	}

	// Revoke the API key
	if err := c.repo.GetAPIKeyRepository().RevokeAPIKey(keyID); err != nil {
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to revoke API key",
			Error:   map[string]string{"error": "Failed to revoke API key, please try again later"},
		})
		return
	}

	ctx.JSON(http.StatusOK, common.APIResponse{
		Success: true,
		Data:    nil,
		Message: "API key revoked successfully",
		Error:   nil,
	})
}

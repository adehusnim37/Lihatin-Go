package api

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/gin-gonic/gin"
)

// GetAPIKey returns a specific API key by ID
func (c *Controller) GetAPIKey(ctx *gin.Context) {
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

	// Get the API key
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
			Error:   map[string]string{"auth": "You can only access your own API keys"},
		})
		return
	}

	// Convert to response format (hide sensitive data)
	response := dto.APIKeyResponse{
		ID:          apiKey.ID,
		Name:        apiKey.Name,
		KeyPreview:  utils.GetKeyPreview(apiKey.Key),
		LastUsedAt:  apiKey.LastUsedAt,
		ExpiresAt:   apiKey.ExpiresAt,
		IsActive:    apiKey.IsActive,
		Permissions: apiKey.Permissions,
		CreatedAt:   apiKey.CreatedAt,
	}

	ctx.JSON(http.StatusOK, common.APIResponse{
		Success: true,
		Data:    response,
		Message: "API key retrieved successfully",
		Error:   nil,
	})
}

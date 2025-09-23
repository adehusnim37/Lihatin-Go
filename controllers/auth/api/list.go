package api

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/gin-gonic/gin"
)

// GetAPIKeys returns user's API keys
func (c *Controller) GetAPIKeys(ctx *gin.Context) {
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

	// Get user's API keys
	apiKeys, err := c.repo.GetAPIKeyRepository().GetAPIKeysByUserID(userID.(string))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to retrieve API keys",
			Error:   map[string]string{"error": "Failed to retrieve API keys, please try again later"},
		})
		return
	}

	// Convert to response format (hide sensitive data)
	responses := make([]dto.APIKeyResponse, len(apiKeys))
	for i, key := range apiKeys {
		responses[i] = dto.APIKeyResponse{
			ID:          key.ID,
			Name:        key.Name,
			KeyPreview:  utils.GetKeyPreview(key.Key), // Use method to get preview
			LastUsedAt:  key.LastUsedAt,
			ExpiresAt:   key.ExpiresAt,
			IsActive:    key.IsActive,
			Permissions: []string(key.Permissions),
			CreatedAt:   key.CreatedAt,
		}
	}

	ctx.JSON(http.StatusOK, common.APIResponse{
		Success: true,
		Data:    responses,
		Message: "API keys retrieved successfully",
		Error:   nil,
	})
}

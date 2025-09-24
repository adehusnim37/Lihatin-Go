package api

import (
	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/gin-gonic/gin"
)

// GetAPIKeys returns user's API keys
func (c *Controller) GetAPIKeys(ctx *gin.Context) {
	// Get user ID from JWT token context
	userID, _ := ctx.Get("user_id")

	// Get user's API keys
	apiKeys, err := c.repo.GetAPIKeyRepository().GetAPIKeysByUserID(userID.(string))
	if err != nil {
		utils.HandleError(ctx, err, userID)
		return
	}

	utils.Logger.Info("User's API keys retrieved successfully",
		"user_id", userID,
		"api_key_count", len(apiKeys),
	)

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

	utils.SendOKResponse(ctx, responses, "API keys retrieved successfully")
}

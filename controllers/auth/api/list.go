package api

import (
	"github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/gin-gonic/gin"
)

// GetAPIKeys returns user's API keys
func (c *Controller) GetAPIKeys(ctx *gin.Context) {
	// Get user ID from JWT token context
	userID, _ := ctx.Get("user_id")

	// Get user's API keys (now returns DTOs directly)
	apiKeys, err := c.repo.GetAPIKeyRepository().GetAPIKeysByUserID(userID.(string))
	if err != nil {
		http.HandleError(ctx, err, userID)
		return
	}

	logger.Logger.Info("User's API keys retrieved successfully",
		"user_id", userID,
		"api_key_count", len(apiKeys),
	)

	// Repository already returns DTOs, no conversion needed
	http.SendOKResponse(ctx, apiKeys, "API keys retrieved successfully")
}

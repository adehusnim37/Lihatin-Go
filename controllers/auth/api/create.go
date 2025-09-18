package api

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/gin-gonic/gin"
)

func (c *Controller) CreateAPIKey(ctx *gin.Context) {
	// Get user ID from JWT token context
	userID, _ := ctx.Get("user_id")

	var req dto.APIKeyRequest

	// Bind and validate request
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.SendValidationError(ctx, err, &req)
		return
	}

	// Create API key
	apiKey, keyString, err := c.repo.GetAPIKeyRepository().CreateAPIKey(
		userID.(string),
		req.Name,
		req.ExpiresAt,
		req.Permissions,
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to create API key",
			Error:   map[string]string{"error": "Failed to create API key, please try again later"},
		})
		return
	}

	// Return the key with the full key string (this is the only time it's shown)
	response := map[string]interface{}{
		"id":          apiKey.ID,
		"name":        apiKey.Name,
		"key":         keyString, // Full key shown only once
		"expires_at":  apiKey.ExpiresAt,
		"permissions": apiKey.Permissions,
		"created_at":  apiKey.CreatedAt,
		"warning":     "This is the only time the full API key will be shown. Please save it securely.",
	}

	ctx.JSON(http.StatusCreated, common.APIResponse{
		Success: true,
		Data:    response,
		Message: "API key created successfully",
		Error:   nil,
	})
}

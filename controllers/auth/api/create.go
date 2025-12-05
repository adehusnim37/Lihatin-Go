package api

import (
	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	"github.com/gin-gonic/gin"
)

func (c *Controller) CreateAPIKey(ctx *gin.Context) {
	// Get user ID from JWT token context
	userID, _ := ctx.Get("user_id")

	var req dto.CreateAPIKeyRequest

	// Bind and validate request
	if err := ctx.ShouldBindJSON(&req); err != nil {
		validator.SendValidationError(ctx, err, &req)
		return
	}

	// Create API key
	apiKey, fullAPIKey, err := c.repo.GetAPIKeyRepository().CreateAPIKey(
		userID.(string),
		req,
	)

	// Handle errors using universal error handler
	if err != nil {
		http.HandleError(ctx, err, userID)
		return
	}

	// Success response using proper DTO
	logger.Logger.Info("API key created successfully",
		"user_id", userID,
		"api_key_id", apiKey.ID,
		"api_key_name", apiKey.Name,
	)

	// Create response using DTO
	response := dto.CreateAPIKeyResponse{
		ID:          apiKey.ID,
		Name:        apiKey.Name,
		CreatedAt:   apiKey.CreatedAt,
		ExpiresAt:   apiKey.ExpiresAt,
		Permissions: []string(apiKey.Permissions),
		BlockedIPs:  apiKey.BlockedIPs,
		AllowedIPs:  apiKey.AllowedIPs,
		LimitUsage:  apiKey.LimitUsage,
		UsageCount:  apiKey.UsageCount,
		IsActive:    apiKey.IsActive,
		// Sensitive info
		Key:         fullAPIKey, // Full key with secret (only shown once!)
		Warning:     "Please save this key as it will not be shown again.",
	}

	http.SendCreatedResponse(ctx, response, "API key created successfully")
}

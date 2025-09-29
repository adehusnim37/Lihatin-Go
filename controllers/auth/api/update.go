package api

import (
	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/gin-gonic/gin"
)

// UpdateAPIKey updates an API key
func (c *Controller) UpdateAPIKey(ctx *gin.Context) {
	var reqId dto.APIKeyIDRequest
	// Get user ID from context
	userID, _ := ctx.Get("user_id")
	userIDStr := userID.(string)

	// Bind and validate request
	var req dto.UpdateAPIKeyRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.SendValidationError(ctx, err, &req)
		return
	}

	// Bind and validate URI parameters
	if err := ctx.ShouldBindUri(&reqId); err != nil {
		utils.SendValidationError(ctx, err, &reqId)
		return
	}

	// Update the API key using the corrected repository method
	updatedKey, err := c.repo.GetAPIKeyRepository().UpdateAPIKey(reqId, userIDStr, req)
	if err != nil {
		utils.HandleError(ctx, err, userID)
		return
	}

	// Convert to response model using DTO
	response := dto.APIKeyResponse{
		ID:          updatedKey.ID,
		Name:        updatedKey.Name,
		KeyPreview:  utils.GetKeyPreview(updatedKey.Key),
		LimitUsage:  updatedKey.LimitUsage,
		UsageCount:  updatedKey.UsageCount,
		LastIPUsed:  updatedKey.LastIPUsed,
		LastUsedAt:  updatedKey.LastUsedAt,
		ExpiresAt:   updatedKey.ExpiresAt,
		IsActive:    updatedKey.IsActive,
		Permissions: []string(updatedKey.Permissions),
		CreatedAt:   updatedKey.CreatedAt,
	}

	utils.SendOKResponse(ctx, response, "API key updated successfully")
}

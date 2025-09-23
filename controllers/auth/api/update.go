package api

import (
	"time"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/models/user"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/gin-gonic/gin"
)

// UpdateAPIKey updates an API key
func (c *Controller) UpdateAPIKey(ctx *gin.Context) {
	// Get user ID from context
	userID, exists := ctx.Get("user_id")
	if !exists {
		utils.HandleError(ctx, utils.ErrUserUnauthorized, nil)
		return
	}

	// Get API key ID from URL
	keyID := ctx.Param("id")
	if keyID == "" {
		utils.SendErrorResponse(ctx, 400, "INVALID_REQUEST", "API key ID is required", "id", userID)
		return
	}

	// Bind and validate request
	var req dto.UpdateAPIKeyRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.SendValidationError(ctx, err, &req)
		return
	}

	// Check if API key exists and belongs to user
	apiKey, err := c.repo.GetAPIKeyRepository().GetAPIKeyByID(keyID)
	if err != nil {
		utils.HandleError(ctx, utils.ErrAPIKeyNotFound, userID)
		return
	}

	// Verify ownership
	if apiKey.UserID != userID.(string) {
		utils.HandleError(ctx, utils.ErrAPIKeyUnauthorized, userID)
		return
	}

	// Prepare updates map
	updates := make(map[string]interface{})

	if req.Name != nil {
		updates["name"] = *req.Name
	}

	if req.ExpiresAt != nil {
		// Validate expiration date is in the future
		if req.ExpiresAt.Before(time.Now()) {
			utils.SendErrorResponse(ctx, 400, "INVALID_EXPIRATION_DATE", "Expiration date must be in the future", "expires_at", userID)
			return
		}
		updates["expires_at"] = *req.ExpiresAt
	}

	if req.Permissions != nil {
		// Validate permissions
		validPermissions := []string{"read", "write", "delete", "admin"}
		for _, perm := range req.Permissions {
			valid := false
			for _, validPerm := range validPermissions {
				if perm == validPerm {
					valid = true
					break
				}
			}
			if !valid {
				utils.SendErrorResponse(ctx, 400, "INVALID_PERMISSION", "Invalid permission: "+perm, "permissions", userID)
				return
			}
		}
		updates["permissions"] = user.PermissionsList(req.Permissions)
	}

	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	// Always update the updated_at timestamp
	updates["updated_at"] = time.Now()

	// Update the API key
	if err := c.repo.GetAPIKeyRepository().UpdateAPIKey(keyID, updates); err != nil {
		utils.HandleError(ctx, utils.ErrAPIKeyUpdateFailed, userID)
		return
	}

	// Fetch the updated API key to return
	updatedKey, err := c.repo.GetAPIKeyRepository().GetAPIKeyByID(keyID)
	if err != nil {
		utils.HandleError(ctx, utils.ErrAPIKeyFailedFetching, userID)
		return
	}

	// Convert to response model using DTO
	response := dto.APIKeyResponse{
		ID:          updatedKey.ID,
		Name:        updatedKey.Name,
		KeyPreview:  utils.GetKeyPreview(updatedKey.Key),
		LastUsedAt:  updatedKey.LastUsedAt,
		ExpiresAt:   updatedKey.ExpiresAt,
		IsActive:    updatedKey.IsActive,
		Permissions: []string(updatedKey.Permissions),
		CreatedAt:   updatedKey.CreatedAt,
	}

	utils.SendOKResponse(ctx, response, "API key updated successfully")
}

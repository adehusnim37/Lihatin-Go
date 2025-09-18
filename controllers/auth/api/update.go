package api

import (
	"net/http"
	"time"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/gin-gonic/gin"
)

// UpdateAPIKey updates an API key
func (c *Controller) UpdateAPIKey(ctx *gin.Context) {
	// Get user ID from context
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Unauthorized",
			Error:   map[string]string{"auth": "User not authenticated"},
		})
		return
	}

	// Get API key ID from URL
	keyID := ctx.Param("id")
	if keyID == "" {
		ctx.JSON(http.StatusBadRequest, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid request",
			Error:   map[string]string{"id": "API key ID is required"},
		})
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
		ctx.JSON(http.StatusNotFound, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "API key not found",
			Error:   map[string]string{"api_key": "API key not found or access denied"},
		})
		return
	}

	// Verify ownership
	if apiKey.UserID != userID.(string) {
		ctx.JSON(http.StatusForbidden, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Access denied",
			Error:   map[string]string{"api_key": "You can only update your own API keys"},
		})
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
			ctx.JSON(http.StatusBadRequest, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Invalid expiration date",
				Error:   map[string]string{"expires_at": "Expiration date must be in the future"},
			})
			return
		}
		updates["expires_at"] = *req.ExpiresAt
	}

	if req.Permissions != nil {
		// Validate permissions (you can add custom validation logic here)
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
				ctx.JSON(http.StatusBadRequest, common.APIResponse{
					Success: false,
					Data:    nil,
					Message: "Invalid permission",
					Error:   map[string]string{"permissions": "Invalid permission: " + perm},
				})
				return
			}
		}
		updates["permissions"] = req.Permissions
	}

	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	// Always update the updated_at timestamp
	updates["updated_at"] = time.Now()

	// Update the API key
	if err := c.repo.GetAPIKeyRepository().UpdateAPIKey(keyID, updates); err != nil {
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to update API key",
			Error:   map[string]string{"database": "Could not update API key"},
		})
		return
	}

	// Fetch the updated API key to return
	updatedKey, err := c.repo.GetAPIKeyRepository().GetAPIKeyByID(keyID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "API key updated but failed to fetch updated data",
			Error:   map[string]string{"database": "Could not fetch updated API key"},
		})
		return
	}

	// Convert to response model
	response := dto.APIKeyResponse{
		ID:          updatedKey.ID,
		Name:        updatedKey.Name,
		KeyPreview:  utils.GetKeyPreview(updatedKey.Key),
		LastUsedAt:  updatedKey.LastUsedAt,
		ExpiresAt:   updatedKey.ExpiresAt,
		IsActive:    updatedKey.IsActive,
		Permissions: updatedKey.Permissions,
		CreatedAt:   updatedKey.CreatedAt,
	}

	ctx.JSON(http.StatusOK, common.APIResponse{
		Success: true,
		Data:    response,
		Message: "API key updated successfully",
		Error:   nil,
	})
}

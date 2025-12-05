package api

import (
	"fmt"
	"time"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/internal/pkg/auth"
	"github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	"github.com/gin-gonic/gin"
)

// RefreshAPIKey refreshes an API key's expiration date and regenerates the secret key
func (c *Controller) RefreshAPIKey(ctx *gin.Context) {
    var reqId dto.APIKeyIDRequest
    userID, _ := ctx.Get("user_id")

    if err := ctx.ShouldBindUri(&reqId); err != nil {
        validator.SendValidationError(ctx, err, &reqId)
        return
    }

    updatedKey, fullAPIKey, err := c.repo.GetAPIKeyRepository().RegenerateAPIKey(reqId.ID, userID.(string))
    if err != nil {
        http.HandleError(ctx, err, userID)
        return
    }

    // ✅ STRUCTURED RESPONSE WITH SEPARATION
   response := dto.APIKeyRefreshResponse{
        ID:          updatedKey.ID,
        Name:        updatedKey.Name,
        KeyPreview:  auth.GetKeyPreview(updatedKey.Key),
        IsActive:    updatedKey.IsActive,
        ExpiresAt:   updatedKey.ExpiresAt,
        Permissions: []string(updatedKey.Permissions),
        CreatedAt:   updatedKey.CreatedAt,
        UpdatedAt:   updatedKey.UpdatedAt,
        Secret: dto.APIKeySecretInfo{
            FullAPIKey: fullAPIKey,
            Warning:    "Save this key securely - it will not be shown again",
            Format:     "Use as: X-API-Key: " + fullAPIKey,
            ExpiresIn:  calculateExpirationTime(updatedKey.ExpiresAt),
        },
        Usage: dto.APIKeyUsageInfo{
            LastUsedAt:         updatedKey.LastUsedAt,
            LastIPUsed:         updatedKey.LastIPUsed,
            IsRegenerated:      true,
            PreviousUsageReset: true,
        },
    }

    http.SendOKResponse(ctx, response, "API key refreshed successfully")
}

// Helper function
func calculateExpirationTime(expiresAt *time.Time) string {
    if expiresAt == nil {
        return "never"
    }
    
    duration := time.Until(*expiresAt)
    if duration < 0 {
        return "expired"
    }
    
    days := int(duration.Hours() / 24)
    if days > 0 {
        return fmt.Sprintf("%d days", days)
    }
    
    hours := int(duration.Hours())
    if hours > 0 {
        return fmt.Sprintf("%d hours", hours)
    }
    
    return fmt.Sprintf("%d minutes", int(duration.Minutes()))
}
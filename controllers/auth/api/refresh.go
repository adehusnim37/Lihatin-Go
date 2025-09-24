package api

import (
	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/gin-gonic/gin"
)

// RefreshAPIKey refreshes an API key's expiration date
func (c *Controller) RefreshAPIKey(ctx *gin.Context) {
	var reqId dto.APIKeyIDRequest
	// Get user ID from JWT token context
	userID, _ := ctx.Get("user_id")

	// Bind and validate URI parameters
	if err := ctx.ShouldBindUri(&reqId); err != nil {
		utils.SendValidationError(ctx, err, &reqId)
		return
	}

	// Refresh the API key's expiration date
	updatedKey, _, err := c.repo.GetAPIKeyRepository().RegenerateAPIKey(reqId.ID, userID.(string))
	if err != nil {
		utils.HandleError(ctx, err, userID)
		return
	}

	utils.SendOKResponse(ctx, updatedKey, "API key refreshed successfully")
}
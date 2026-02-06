package api

import (

	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/dto"
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

	response, err := c.repo.GetAPIKeyRepository().RegenerateAPIKey(reqId.ID, userID.(string))
	if err != nil {
		http.HandleError(ctx, err, userID)
		return
	}

	logger.Logger.Info("API key refreshed successfully",
		"user_id", userID,
		"api_key_id", response.ID,
		"new_expires_at", response.ExpiresAt,
	)

	http.SendOKResponse(ctx, response, "API key refreshed successfully")
}


package api

import (
	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	"github.com/gin-gonic/gin"
)

// RevokeAPIKey revokes an API key
func (c *Controller) RevokeAPIKey(ctx *gin.Context) {
	var req dto.APIKeyIDRequest
	// Get user ID from JWT token context
	userID, _ := ctx.Get("user_id")

	// Bind and validate request
	if err := ctx.ShouldBindUri(&req); err != nil {
		validator.SendValidationError(ctx, err, &req)
		return
	}

	// Revoke the API key
	if err := c.repo.GetAPIKeyRepository().RevokeAPIKey(req, userID.(string)); err != nil {
		http.HandleError(ctx, err, userID)
		return
	}

	http.SendOKResponse(ctx, nil, "API key revoked successfully")
}

package api

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/gin-gonic/gin"
)

// GetAPIKey returns a specific API key by ID
func (c *Controller) GetAPIKey(ctx *gin.Context) {
	var req dto.APIKeyIDRequest
	// Get user ID from JWT token context
	userID, _ := ctx.Get("user_id")

	// Bind and validate request
	if err := ctx.ShouldBindUri(&req); err != nil {
		utils.SendValidationError(ctx, err, &req)
		return
	}

	// Get the API key
	apiKey, err := c.repo.GetAPIKeyRepository().GetAPIKeyByID(req, userID.(string))
	if err != nil {
		utils.SendErrorResponse(ctx, http.StatusNotFound, "API key not found", "error_code_not_found", "API key retrieval failed", map[string]string{"key_id": "API key with this ID does not exist"})
		return
	}

	utils.SendOKResponse(ctx, apiKey, "API key retrieved successfully")
}

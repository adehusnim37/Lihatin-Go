package api

import (
	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	"github.com/gin-gonic/gin"
)

// Controller handles API key deactivation requests
func (c *Controller) DeactivateAPIKey(ctx *gin.Context) {
	var reqId dto.APIKeyIDRequest

	userID := ctx.GetString("user_id")
	userRole := ctx.GetString("role")

	if err := ctx.ShouldBindUri(&reqId); err != nil {
		validator.SendValidationError(ctx, err, &reqId)
		return
	}

	// Call the service to deactivate the account
	success, err := c.repo.GetAPIKeyRepository().DeactivateAPIKey(reqId, userID, userRole)
	if err != nil {
		http.HandleError(ctx, err, userID)
		return
	}

	http.SendOKResponse(ctx, success, "Account deactivated successfully")
}

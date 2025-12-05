package api

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/dto"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/gin-gonic/gin"
)

// Controller handles API key activation requests
func (c *Controller) ActivateAPIKey(ctx *gin.Context) {
	var reqId dto.APIKeyIDRequest

	userID := ctx.GetString("user_id")
	userRole := ctx.GetString("role")

	if err := ctx.ShouldBindUri(&reqId); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Call the service to activate the account
	success, err := c.repo.GetAPIKeyRepository().ActivateAPIKey(reqId, userID, userRole)
	if err != nil {
		httputil.HandleError(ctx, err, userID)
		return
	}

	httputil.SendOKResponse(ctx, success, "Account activated successfully")
}
package api

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/gin-gonic/gin"
)

// Controller handles API key deactivation requests
func (c *Controller) DeactivateAPIKey(ctx *gin.Context) {
	var reqId dto.APIKeyIDRequest

	userID := ctx.GetString("user_id")
	userRole := ctx.GetString("role")

	if err := ctx.ShouldBindUri(&reqId); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Call the service to deactivate the account
	success, err := c.repo.GetAPIKeyRepository().DeactivateAPIKey(reqId, userID, userRole)
	if err != nil {
		utils.HandleError(ctx, err, userID)
		return
	}

	utils.SendOKResponse(ctx, success, "Account deactivated successfully")
}
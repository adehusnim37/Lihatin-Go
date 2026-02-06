package api

import (
	"github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/gin-gonic/gin"
)

// GetAPIKeyStats retrieves statistics for a specific API key
func (c *Controller) GetAPIKeyStats(ctx *gin.Context) {
	userID := ctx.GetString("user_id")

	stats, err := c.repo.GetAPIKeyRepository().GetAPIKeyStats(userID)
	if err != nil {
		http.HandleError(ctx, err, userID)
		return
	}

	http.SendOKResponse(ctx, stats, "API key stats retrieved successfully")
}

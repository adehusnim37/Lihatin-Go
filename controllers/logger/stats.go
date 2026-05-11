package logger

import (
	httpPkg "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/gin-gonic/gin"
)

// GetAllCountedLogs retrieves count of all logs by method type
func (c *Controller) GetAllCountedLogs(ctx *gin.Context) {
	userRole := ctx.GetString("role")
	userID := ctx.GetString("user_id")

	// Fetch counted logs from repository
	counts, err := c.repo.GetAllCountedLogs(userRole, userID)
	if err != nil {
		httpPkg.HandleError(ctx, err, "")
		return
	}

	httpPkg.SendOKResponse(ctx, counts, "Log counts retrieved successfully")
}

// GetLogsStats retrieves stats of all logs
func (c *Controller) GetLogsStats(ctx *gin.Context) {
	userRole := ctx.GetString("role")
	userID := ctx.GetString("user_id")

	// Fetch stats from repository
	stats, err := c.repo.GetLogsStats(userRole, userID)
	if err != nil {
		httpPkg.HandleError(ctx, err, "")
		return
	}

	httpPkg.SendOKResponse(ctx, stats, "Logs stats retrieved successfully")
}

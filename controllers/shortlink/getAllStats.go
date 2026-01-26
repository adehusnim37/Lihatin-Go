package shortlink

import (
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/gin-gonic/gin"
)

func (c *Controller) GetAllStatsShorts(ctx *gin.Context) {
	userId := ctx.GetString("user_id")
	userRole := ctx.GetString("role")
	startDate := ctx.Query("start_date")
	endDate := ctx.Query("end_date")

	stats, err := c.repo.GetDashboardStats(userId, userRole, startDate, endDate)
	if err != nil {
		httputil.HandleError(ctx, err, userId)
		return
	}

	httputil.SendOKResponse(ctx, stats, "Dashboard stats retrieved successfully")
}

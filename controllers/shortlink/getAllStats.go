package shortlink

import (
	"net/http"

	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/gin-gonic/gin"
)

func (c *Controller) GetAllStatsShorts(ctx *gin.Context) {
	userId, _ := ctx.Get("user_id")
	userRole, _ := ctx.Get("role")
	userRoleStr := userRole.(string)
	pageStr := ctx.DefaultQuery("page", "1")
	limitStr := ctx.DefaultQuery("limit", "10")
	sort := ctx.DefaultQuery("sort", "created_at")
	orderBy := ctx.DefaultQuery("order_by", "desc")

	page, limit, sort, orderBy, vErrs := httputil.PaginateValidate(pageStr, limitStr, sort, orderBy, httputil.Role(userRoleStr))
	if vErrs != nil {
		ctx.JSON(http.StatusBadRequest, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid pagination parameters",
			Error:   vErrs,
		})
		return
	}
	stats, err := c.repo.GetStatsAllShortLinks(userId.(string), userRoleStr, page, limit, sort, orderBy)
	if err != nil {
		httputil.HandleError(ctx, err, userId)
		return
	}

	httputil.SendOKResponse(ctx, stats, "All stats retrieved successfully")
}

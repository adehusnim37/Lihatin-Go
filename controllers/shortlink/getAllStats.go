package shortlink

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/adehusnim37/lihatin-go/utils"
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

	page, limit, sort, orderBy, vErrs := utils.PaginateValidate(pageStr, limitStr, sort, orderBy, utils.Role(userRoleStr))
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
		utils.Logger.Error("Failed to get all stats",
			"error", err.Error(),
		)
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to get all stats",
			Error:   map[string]string{"error": "Terjadi kesalahan pada server"},
		})
		return
	}

	ctx.JSON(http.StatusOK, common.APIResponse{
		Success: true,
		Data:    stats,
		Message: "All stats retrieved successfully",
		Error:   nil,
	})
}

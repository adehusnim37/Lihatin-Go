package shortlink

import (
	"github.com/adehusnim37/lihatin-go/dto"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	"github.com/gin-gonic/gin"
)

func (c *Controller) GetShortLinkStats(ctx *gin.Context) {
	// Parse URI parameters
	var req dto.CodeRequest

	if err := ctx.ShouldBindUri(&req); err != nil {
		validator.SendValidationError(ctx, err, &req)
		return
	}

	stats, err := c.repo.GetStatsShortLink(req.Code, ctx.GetString("user_id"), ctx.GetString("role"))
	if err != nil {
		httputil.HandleError(ctx, err, ctx.GetString("user_id"))
		return
	}

	httputil.SendOKResponse(ctx, stats, "Short link stats retrieved successfully")
}

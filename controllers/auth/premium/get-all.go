package premium

import (
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/gin-gonic/gin"
)

func (c *Controller) GetAllPremiumKeys(ctx *gin.Context) {

	pageStr := ctx.DefaultQuery("page", "1")
	limitStr := ctx.DefaultQuery("limit", "10")
	sort := ctx.DefaultQuery("sort", "created_at")
	orderBy := ctx.DefaultQuery("order_by", "desc")

	userRole := ctx.GetString("role")

	// Validate and convert pagination parameters
	page, limit, sort, orderBy, vErrs := httputil.PaginateValidate(pageStr, limitStr, sort, orderBy, httputil.Role(userRole))
	if vErrs != nil {
		httputil.SendValidationErrorResponse(ctx, "Invalid pagination parameters", vErrs)
		return
	}

	logger.Logger.Info("Fetching short links",
		"user_role", userRole,
		"page", page,
		"limit", limit,
		"sort", sort,
		"order_by", orderBy,
	)

	paginatedResponse, err := c.premiumRepo.GetUserPremiumKeyList(page, limit, sort, orderBy)
	if err != nil {
		httputil.HandleError(ctx, err, nil)
		return
	}

	httputil.SendOKResponse(ctx, paginatedResponse, "Premium keys retrieved successfully")
}

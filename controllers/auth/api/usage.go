package api

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/dto"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/gin-gonic/gin"
)

// GetAPIKeyActivityLogs retrieves activity logs associated with a specific API key
func (c *Controller) GetAPIKeyActivityLogs(ctx *gin.Context) {

	var reqId dto.APIKeyIDRequest
	if err := ctx.ShouldBindUri(&reqId); err != nil {
		validator.SendValidationError(ctx, err, &reqId)
		return
	}

	// Get user ID from context
	userID := ctx.GetString("user_id")
	userRole := ctx.GetString("role")
	pageStr := ctx.DefaultQuery("page", "1")
	limitStr := ctx.DefaultQuery("limit", "10")
	sort := ctx.DefaultQuery("sort", "created_at")
	orderBy := ctx.DefaultQuery("order_by", "desc")
	// Pagination parameters

	page, limit, sort, orderBy, vErrs := httputil.PaginateValidate(pageStr, limitStr, sort, orderBy, httputil.Role(userRole))
	if vErrs != nil {
		ctx.JSON(http.StatusBadRequest, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid pagination parameters",
			Error:   vErrs,
		})
		return
	}

	// Fetch activity logs for the API key (now returns complete DTO)
	response, err := c.repo.GetAPIKeyRepository().APIKeyUsageHistory(reqId, userID, userRole, page, limit, sort, orderBy)
	if err != nil {
		httputil.HandleError(ctx, err, userID)
		return
	}

	// Repository already returns APIKeyActivityLogsResponse DTO with all data and pagination
	httputil.SendOKResponse(ctx, response, "API key activity logs fetched successfully")
}

package loginattempts

import (
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/gin-gonic/gin"
)

// GetLoginAttempts returns login attempt history with enhanced filtering
func (c *Controller) GetLoginAttempts(ctx *gin.Context) {
	logger.Logger.Info("GetLoginAttempts called")

	// Check if user is admin
	isAdmin := ctx.GetBool("is_admin")

	// Validate pagination parameters
	page, limit, sortField, sortOrder, validationErrs := httputil.PaginateValidateLoginAttempts(
		ctx.Query("page"),
		ctx.Query("limit"),
		ctx.Query("sort"),
		ctx.Query("order_by"),
		isAdmin,
	)

	if validationErrs != nil {
		httputil.SendErrorResponse(ctx, 400, "VALIDATION_ERROR", "Invalid pagination parameters", "pagination", validationErrs)
		return
	}

	// Get user email for non-admin filtering
	userEmail := ""
	if !isAdmin {
		if email, exists := ctx.Get("email"); exists {
			userEmail = email.(string)
			logger.Logger.Info("Non-admin user filtering own attempts", "email", userEmail)
		}
	}

	// Parse and validate filters
	queryParams := map[string]string{
		"success":    ctx.Query("success"),
		"id":         ctx.Query("id"),
		"search":     ctx.Query("search"),
		"username":   ctx.Query("username"),
		"ip_address": ctx.Query("ip_address"),
		"from_date":  ctx.Query("from_date"),
		"to_date":    ctx.Query("to_date"),
	}

	filters, filterErrs := httputil.ParseLoginAttemptsFilters(queryParams, isAdmin, userEmail)
	if len(filterErrs) > 0 {
		httputil.SendErrorResponse(ctx, 400, "FILTER_ERROR", "Invalid filter parameters", "filters", filterErrs)
		return
	}

	// Add sort parameters to filters
	filters["sort_field"] = sortField
	filters["sort_order"] = sortOrder

	offset := (page - 1) * limit

	// Get login attempts with enhanced filters
	attempts, totalCount, err := c.repo.GetLoginAttemptRepository().GetAllLoginAttempts(limit, offset, filters)
	if err != nil {
		httputil.HandleError(ctx, err, nil)
		return
	}

	totalPages := int((totalCount + int64(limit) - 1) / int64(limit))

	response := map[string]any{
		"attempts":    attempts,
		"total_count": totalCount,
		"page":        page,
		"limit":       limit,
		"total_pages": totalPages,
		"has_next":    page < totalPages,
		"has_prev":    page > 1,
	}

	httputil.SendOKResponse(ctx, response, "Successfully retrieved login attempts")
}

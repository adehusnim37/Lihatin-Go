package shortlink

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/models/common"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/gin-gonic/gin"
)

// ListShortLinks handles both user and admin requests with role-based filtering
func (c *Controller) ListShortLinks(ctx *gin.Context) {
	// Get user info from context (set by auth middleware)
	userID, _ := ctx.Get("user_id")

	// Get user role from context (set by auth middleware)
	userRole, _ := ctx.Get("role")

	// Convert to strings
	userIDStr := userID.(string)
	userRoleStr := userRole.(string)

	// Get pagination parameters from query string
	pageStr := ctx.DefaultQuery("page", "1")
	limitStr := ctx.DefaultQuery("limit", "10")
	sort := ctx.DefaultQuery("sort", "created_at")
	orderBy := ctx.DefaultQuery("order_by", "desc")

	// Validate and convert pagination parameters
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

	logger.Logger.Info("Fetching short links",
		"user_id", userIDStr,
		"user_role", userRoleStr,
		"page", page,
		"limit", limit,
		"sort", sort,
		"order_by", orderBy,
	)

	// ✅ SMART FILTERING: Choose repository method based on role
	var paginatedResponse interface{}
	var repositoryErr error

	if userRoleStr == "admin" {
		// ✅ Admin: Get all short links (no user filter)
		logger.Logger.Info("Admin accessing all short links", "admin_user", userIDStr)
		paginatedResponse, repositoryErr = c.repo.ListAllShortLinks(page, limit, sort, orderBy)
	} else {
		// ✅ User: Get only user's short links (filtered by user_id)
		logger.Logger.Info("User accessing own short links", "user_id", userIDStr)
		paginatedResponse, repositoryErr = c.repo.GetShortsByUserIDWithPagination(userIDStr, page, limit, sort, orderBy)
	}

	if repositoryErr != nil {
		logger.Logger.Error("Failed to retrieve short links",
			"user_id", userIDStr,
			"user_role", userRoleStr,
			"page", page,
			"limit", limit,
			"sort", sort,
			"order_by", orderBy,
			"error", repositoryErr.Error(),
		)
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to retrieve short links",
			Error:   map[string]string{"database": "Database error occurred"},
		})
		return
	}

	// ✅ SUCCESS: Log different messages based on role
	if userRoleStr == "admin" {
		logger.Logger.Info("Admin retrieved all short links successfully",
			"admin_user", userIDStr,
			"page", page,
		)
	} else {
		logger.Logger.Info("User retrieved own short links successfully",
			"user_id", userIDStr,
			"page", page,
		)
	}

	// ✅ Return unified response
	ctx.JSON(http.StatusOK, common.APIResponse{
		Success: true,
		Data:    paginatedResponse,
		Message: "Short links retrieved successfully",
		Error:   nil,
	})
}

// ✅ DEPRECATED: Keep for backward compatibility (optional)
func (c *Controller) ListUserShortLinks(ctx *gin.Context) {
	// Force user role for backward compatibility
	ctx.Set("role", "user")
	c.ListShortLinks(ctx)
}

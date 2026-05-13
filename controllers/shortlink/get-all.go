package shortlink

import (
	"net/http"

	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/gin-gonic/gin"
)

// ListShortLinks handles both user and admin requests with role-based filtering
func (c *Controller) ListShortLinks(ctx *gin.Context) {
	// Get user info from context (set by auth middleware)
	userID := ctx.GetString("user_id")
	userRole := ctx.GetString("role")

	targetUserID := ""
	if userRole == "admin" {
		targetUserID = ctx.Param("userID")
	}

	// Get pagination parameters from query string
	pageStr := ctx.DefaultQuery("page", "1")
	limitStr := ctx.DefaultQuery("limit", "10")
	sort := ctx.DefaultQuery("sort", "created_at")
	orderBy := ctx.DefaultQuery("order_by", "desc")

	// Validate and convert pagination parameters
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

	logger.Logger.Info("Fetching short links",
		"user_id", userID,
		"user_role", userRole,
		"page", page,
		"limit", limit,
		"sort", sort,
		"order_by", orderBy,
	)

	// ✅ SMART FILTERING: Choose repository method based on role
	var paginatedResponse any
	var repositoryErr error

	if userRole == "admin" {
		detail := ctx.Query("detail") == "true"
		if targetUserID != "" && !detail {
			// ✅ Admin: Get specific user's short links without details
			logger.Logger.Info("Admin accessing specific user short links without details", "admin_user", userID, "target_user", targetUserID)
			paginatedResponse, repositoryErr = c.repo.GetShortsByUserIDWithPagination(targetUserID, page, limit, sort, orderBy)
		} else {
			// ✅ Admin: Get all short links (with or without target user filter, but WITH details)
			logger.Logger.Info("Admin accessing short links with details", "admin_user", userID, "target_user", targetUserID)
			paginatedResponse, repositoryErr = c.repo.ListAllShortLinks(targetUserID, page, limit, sort, orderBy)
		}
	} else {
		// ✅ User: Get only user's short links (filtered by user_id)
		logger.Logger.Info("User accessing own short links", "user_id", userID)
		paginatedResponse, repositoryErr = c.repo.GetShortsByUserIDWithPagination(userID, page, limit, sort, orderBy)
	}

	if repositoryErr != nil {
		httputil.HandleError(ctx, repositoryErr, userID)
		return
	}

	httputil.SendOKResponse(ctx, paginatedResponse, "Short links retrieved successfully")
}



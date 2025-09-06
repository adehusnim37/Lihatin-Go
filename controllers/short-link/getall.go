package shortlink

import (
	"net/http"
	"strconv"

	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/gin-gonic/gin"
)

// ListShortLinks handles both user and admin requests with role-based filtering
func (c *Controller) ListShortLinks(ctx *gin.Context) {
	// Get user info from context (set by auth middleware)
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Authentication required",
			Error:   map[string]string{"auth": "User ID not found in context"},
		})
		return
	}

	// Get user role from context (set by auth middleware)
	userRole, roleExists := ctx.Get("role")
	if !roleExists {
		// Default to "user" role if not specified
		userRole = "user"
	}

	// Convert to strings
	userIDStr := userID.(string)
	userRoleStr := userRole.(string)

	// Get pagination parameters from query string
	pageStr := ctx.DefaultQuery("page", "1")
	limitStr := ctx.DefaultQuery("limit", "10")
	sort := ctx.DefaultQuery("sort", "created_at")
	orderBy := ctx.DefaultQuery("order_by", "desc")

	// Convert to integers with validation
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		utils.Logger.Warn("Invalid page parameter", "page", pageStr, "user_id", userIDStr)
		ctx.JSON(http.StatusBadRequest, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid page parameter",
			Error:   map[string]string{"page": "Page must be a positive integer"},
		})
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		utils.Logger.Warn("Invalid limit parameter", "limit", limitStr, "user_id", userIDStr)
		ctx.JSON(http.StatusBadRequest, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid limit parameter",
			Error:   map[string]string{"limit": "Limit must be between 1 and 100"},
		})
		return
	}

	// Validate sort parameter (admin gets more options)
	validSorts := map[string]bool{
		"created_at": true,
		"updated_at": true,
		"short_code": true,
		"title":      true,
	}

	// Admin gets additional sort options
	if userRoleStr == "admin" {
		validSorts["original_url"] = true
		validSorts["is_active"] = true
		validSorts["user_id"] = true // Admin can sort by owner
	}

	if !validSorts[sort] {
		sortOptions := "created_at, updated_at, short_code, title"
		if userRoleStr == "admin" {
			sortOptions += ", original_url, is_active, user_id"
		}

		ctx.JSON(http.StatusBadRequest, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid sort parameter",
			Error:   map[string]string{"sort": "Sort must be one of: " + sortOptions},
		})
		return
	}

	// Validate order_by parameter
	if orderBy != "asc" && orderBy != "desc" {
		ctx.JSON(http.StatusBadRequest, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid order_by parameter",
			Error:   map[string]string{"order_by": "Order must be 'asc' or 'desc'"},
		})
		return
	}

	utils.Logger.Info("Fetching short links",
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
		utils.Logger.Info("Admin accessing all short links", "admin_user", userIDStr)
		paginatedResponse, repositoryErr = c.repo.ListAllShortLinks(page, limit, sort, orderBy)
	} else {
		// ✅ User: Get only user's short links (filtered by user_id)
		utils.Logger.Info("User accessing own short links", "user_id", userIDStr)
		paginatedResponse, repositoryErr = c.repo.GetShortsByUserIDWithPagination(userIDStr, page, limit, sort, orderBy)
	}

	if repositoryErr != nil {
		utils.Logger.Error("Failed to retrieve short links",
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
		utils.Logger.Info("Admin retrieved all short links successfully",
			"admin_user", userIDStr,
			"page", page,
		)
	} else {
		utils.Logger.Info("User retrieved own short links successfully",
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

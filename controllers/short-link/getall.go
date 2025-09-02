package shortlink

import (
	"net/http"
	"strconv"

	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/gin-gonic/gin"
)

func (c *Controller) ListUserShortLinks(ctx *gin.Context) {
	// Get user ID from context (set by auth middleware)
	var userID string
	if userIDVal, exists := ctx.Get("user_id"); exists {
		userID = userIDVal.(string)
		utils.Logger.Info("User authenticated", "user_id", userID)
	} else {
		ctx.JSON(http.StatusUnauthorized, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Authentication required",
			Error:   map[string]string{"auth": "User ID not found in context"},
		})
		return
	}

	// Get pagination parameters from query string
	pageStr := ctx.DefaultQuery("page", "1")
	limitStr := ctx.DefaultQuery("limit", "10")
	sort := ctx.DefaultQuery("sort", "created_at")
	orderBy := ctx.DefaultQuery("order_by", "desc")

	// Convert to integers with validation
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
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
		ctx.JSON(http.StatusBadRequest, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid limit parameter",
			Error:   map[string]string{"limit": "Limit must be between 1 and 100"},
		})
		return
	}

	// Validate sort parameter
	validSorts := map[string]bool{
		"created_at": true,
		"updated_at": true,
		"short_code": true,
		"title":      true,
	}
	if !validSorts[sort] {
		ctx.JSON(http.StatusBadRequest, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid sort parameter",
			Error:   map[string]string{"sort": "Sort must be one of: created_at, updated_at, short_code, title"},
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

	// Get paginated short links from repository
	paginatedResponse, err := c.repo.GetShortsByUserIDWithPagination(userID, page, limit, sort, orderBy)
	if err != nil {
		utils.Logger.Error("Failed to retrieve paginated short links",
			"user_id", userID,
			"page", page,
			"limit", limit,
			"sort", sort,
			"order_by", orderBy,
			"error", err.Error(),
		)
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to retrieve short links",
			Error:   map[string]string{"database": "Database error occurred"},
		})
		return
	}

	utils.Logger.Info("Short links retrieved successfully",
		"user_id", userID,
		"page", page,
		"limit", limit,
		"sort", sort,
		"order_by", orderBy,
		"total_count", paginatedResponse.TotalCount,
		"total_pages", paginatedResponse.TotalPages,
	)

	ctx.JSON(http.StatusOK, common.APIResponse{
		Success: true,
		Data:    paginatedResponse,
		Message: "Short links retrieved successfully",
		Error:   nil,
	})
}

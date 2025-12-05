package admin

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/models/common"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/gin-gonic/gin"
)


func (c *Controller) GetAllUsers(ctx *gin.Context) {
	// Get pagination parameters from query string
	pageStr := ctx.DefaultQuery("page", "1")
	limitStr := ctx.DefaultQuery("limit", "10")
	sort := ctx.DefaultQuery("sort", "created_at")
	orderBy := ctx.DefaultQuery("order_by", "desc")
	// Get user role from context (set by auth middleware)
	userRole, _ := ctx.Get("role")
	userRoleStr := userRole.(string)

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

	// Calculate offset
	offset := (page - 1) * limit

	//logging
	logger.Logger.Info("Fetching all users",
		"page", page,
		"limit", limit,
		"sort", sort,
		"order_by", orderBy,
	)

	// Get users with pagination
	users, totalCount, err := c.repo.GetUserAdminRepository().GetAllUsersWithPagination(limit, offset)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to retrieve users",
			Error:   map[string]string{"error": "Failed to retrieve users, please try again later"},
		})
		return
	}

	// Convert to admin response format (remove passwords)
	adminUsers := make([]dto.AdminUserResponse, len(users))
	for i, u := range users {
		adminUsers[i] = dto.AdminUserResponse{
			ID:           u.ID,
			Username:     u.Username,
			FirstName:    u.FirstName,
			LastName:     u.LastName,
			Email:        u.Email,
			CreatedAt:    u.CreatedAt,
			UpdatedAt:    u.UpdatedAt,
			IsPremium:    u.IsPremium,
			IsLocked:     u.IsLocked,
			LockedAt:     u.LockedAt,
			LockedReason: u.LockedReason,
			Role:         u.Role,
		}
	}

	totalPages := int((totalCount + int64(limit) - 1) / int64(limit))

	response := dto.PaginatedUsersResponse{
		Users:      adminUsers,
		TotalCount: totalCount,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}

	ctx.JSON(http.StatusOK, common.APIResponse{
		Success: true,
		Data:    response,
		Message: "Users retrieved successfully",
		Error:   nil,
	})
}
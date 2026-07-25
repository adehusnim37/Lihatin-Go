package admin

import (
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/adehusnim37/lihatin-go/dto"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/repositories/userrepo"
	"github.com/gin-gonic/gin"
)

const maxAdminUserSearchLength = 100

func (c *Controller) GetAllUsers(ctx *gin.Context) {
	pageStr := ctx.DefaultQuery("page", "1")
	limitStr := ctx.DefaultQuery("limit", "20")
	sort := ctx.DefaultQuery("sort", "created_at")
	orderBy := ctx.DefaultQuery("order_by", "desc")
	search := strings.TrimSpace(ctx.Query("search"))
	role := strings.ToLower(strings.TrimSpace(ctx.Query("role")))
	premiumStatus := strings.ToLower(strings.TrimSpace(ctx.Query("premium_status")))
	lockStatus := strings.ToLower(strings.TrimSpace(ctx.Query("lock_status")))

	page, limit, sort, orderBy, vErrs := httputil.PaginateValidateAdminUsers(pageStr, limitStr, sort, orderBy)
	if vErrs != nil {
		httputil.SendErrorResponse(ctx, http.StatusBadRequest, "INVALID_PAGINATION_PARAMS", "Invalid pagination parameters", "pagination", vErrs)
		return
	}

	filterErrors := make(map[string]string)
	if utf8.RuneCountInString(search) > maxAdminUserSearchLength {
		filterErrors["search"] = "Search must not exceed 100 characters"
	}
	if role != "" && role != "user" && role != "admin" && role != "super_admin" {
		filterErrors["role"] = "Role must be one of: user, admin, super_admin"
	}
	if premiumStatus != "" && premiumStatus != "free" && premiumStatus != "premium" && premiumStatus != "revoked" {
		filterErrors["premium_status"] = "Premium status must be one of: free, premium, revoked"
	}
	if lockStatus != "" && lockStatus != "locked" && lockStatus != "unlocked" {
		filterErrors["lock_status"] = "Lock status must be one of: locked, unlocked"
	}
	if len(filterErrors) > 0 {
		httputil.SendErrorResponse(ctx, http.StatusBadRequest, "INVALID_USER_FILTERS", "Invalid user filters", "filters", filterErrors)
		return
	}

	offset := (page - 1) * limit

	logger.Logger.Info("Fetching all users",
		"page", page,
		"limit", limit,
		"sort", sort,
		"order_by", orderBy,
		"has_search", search != "",
		"role", role,
		"premium_status", premiumStatus,
		"lock_status", lockStatus,
	)

	users, totalCount, err := c.repo.GetUserAdminRepository().GetAllUsersWithPagination(
		limit,
		offset,
		userrepo.AdminUserListFilters{
			Search:        search,
			Role:          role,
			PremiumStatus: premiumStatus,
			LockStatus:    lockStatus,
			Sort:          sort,
			OrderBy:       orderBy,
		},
	)
	if err != nil {
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "FAILED_TO_RETRIEVE_USERS", "Failed to retrieve users, please try again later", "error", err.Error())
		return
	}

	adminUsers := make([]dto.AdminUserResponse, len(users))
	for i, u := range users {
		adminUsers[i] = toAdminUserResponse(u)
	}

	totalPages := int((totalCount + int64(limit) - 1) / int64(limit))

	response := dto.PaginatedUsersResponse{
		Users:      adminUsers,
		TotalCount: totalCount,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
		Sort:       sort,
		OrderBy:    orderBy,
		Search:     search,
		Role:       role,
		Premium:    premiumStatus,
		LockStatus: lockStatus,
	}

	httputil.SendOKResponse(ctx, response, "Users retrieved successfully")
}

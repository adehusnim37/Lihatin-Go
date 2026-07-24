package admin

import (
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/adehusnim37/lihatin-go/dto"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/gin-gonic/gin"
)

const maxRecipientSearchLength = 100

// ListUserEmail returns paginated non-premium users for recipient pickers.
func (c *Controller) ListUserEmail(ctx *gin.Context) {
	pageStr := ctx.DefaultQuery("page", "1")
	limitStr := ctx.DefaultQuery("limit", "20")
	sort := ctx.DefaultQuery("sort", "created_at")
	orderBy := ctx.DefaultQuery("order_by", "desc")
	search := strings.TrimSpace(ctx.Query("search"))
	userRole := ctx.GetString("role")

	page, limit, sort, orderBy, vErrs := httputil.PaginateValidate(pageStr, limitStr, sort, orderBy, httputil.Role(userRole))
	if vErrs != nil {
		httputil.SendErrorResponse(ctx, http.StatusBadRequest, "INVALID_PAGINATION_PARAMS", "Invalid pagination parameters", "pagination", vErrs)
		return
	}
	if sort != "created_at" && sort != "updated_at" {
		httputil.SendErrorResponse(
			ctx,
			http.StatusBadRequest,
			"INVALID_SORT",
			"Sort must be either created_at or updated_at",
			"sort",
		)
		return
	}

	if utf8.RuneCountInString(search) > maxRecipientSearchLength {
		httputil.SendErrorResponse(
			ctx,
			http.StatusBadRequest,
			"INVALID_SEARCH",
			"Search must not exceed 100 characters",
			"search",
		)
		return
	}

	offset := (page - 1) * limit

	logger.Logger.Info("Fetching non-premium recipient options",
		"page", page,
		"limit", limit,
		"sort", sort,
		"order_by", orderBy,
		"has_search", search != "",
	)

	users, totalCount, err := c.repo.GetUserAdminRepository().
		GetNonPremiumUserEmailsWithPagination(limit, offset, search, sort, orderBy)
	if err != nil {
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "FAILED_TO_RETRIEVE_USERS", "Failed to retrieve users, please try again later", "error", err.Error())
		return
	}

	totalPages := int((totalCount + int64(limit) - 1) / int64(limit))

	response := dto.PaginatedUserEmailsResponse{
		Users:      users,
		TotalCount: totalCount,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}

	httputil.SendOKResponse(ctx, response, "Non-premium users retrieved successfully")
}

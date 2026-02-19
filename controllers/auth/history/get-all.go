package history

import (
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/gin-gonic/gin"
)

// GetUserHistory handles GET /api/auth/history
// Returns paginated user history with filtering and sorting
func (c *Controller) GetUserHistory(ctx *gin.Context) {
	// Get query parameters
	pageStr := ctx.DefaultQuery("page", "1")
	limitStr := ctx.DefaultQuery("limit", "10")
	sort := ctx.DefaultQuery("sort", "changed_at")
	orderBy := ctx.DefaultQuery("order_by", "desc")
	userID := ctx.GetString("user_id")

	// Validate pagination parameters
	page, limit, validSort, validOrderBy, errs := httputil.PaginateValidateUserHistory(pageStr, limitStr, sort, orderBy)
	if errs != nil {
		httputil.SendValidationErrorResponse(ctx, "Validation failed", errs)
		return
	}

	// Get paginated history from repository
	result, err := c.historyRepo.GetHistoryUsers(page, limit, validSort, validOrderBy)
	if err != nil {
		httputil.HandleError(ctx, err, userID)
		return
	}

	// Return success response
	httputil.SendOKResponse(ctx, result, "User history list")
}

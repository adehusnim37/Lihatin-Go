package history

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/dto"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/gin-gonic/gin"
)

// GetHistoryByID handles GET /api/auth/history/:id
// Returns a specific history record by ID
func (c *Controller) GetHistoryByID(ctx *gin.Context) {

	id := dto.UserHistoryIDRequest{}
	userID := ctx.GetString("user_id")
	if err := ctx.ShouldBindUri(&id); err != nil {
		httputil.SendErrorResponse(ctx, http.StatusBadRequest, "USER_ID_REQUIRED", "User ID is required", "user_id", id)
		return
	}

	// Get history record by ID
	history, err := c.historyRepo.GetHistoryByID(id.ID, userID)
	if err != nil {
		httputil.HandleError(ctx, err, userID)
		return
	}

	// Return success response
	httputil.SendOKResponse(ctx, history, "History record")
}

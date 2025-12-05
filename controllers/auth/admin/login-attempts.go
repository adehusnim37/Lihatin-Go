package admin

import (
	"net/http"
	"strconv"

	"github.com/adehusnim37/lihatin-go/models/common"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/gin-gonic/gin"
)

// GetLoginAttempts returns login attempt history (admin only)
func (c *Controller) GetLoginAttempts(ctx *gin.Context) {

	logger.Logger.Info("Admin GetLoginAttempts called")
	// Parse pagination parameters
	page := 1
	limit := 50

	if pageStr := ctx.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if limitStr := ctx.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 500 {
			limit = l
		}
	}

	// Parse success filter (optional)
	var successFilter *bool
	if successStr := ctx.Query("success"); successStr != "" {
		if success, err := strconv.ParseBool(successStr); err == nil {
			successFilter = &success
			logger.Logger.Info("Filtering login attempts by success", "success", success)
		}
	}

	id := ctx.Query("id")
	username := ctx.Query("username")

	offset := (page - 1) * limit

	// Get login attempts
	attempts, totalCount, err := c.repo.GetLoginAttemptRepository().GetAllLoginAttempts(limit, offset, successFilter, id, username)
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
	}

	ctx.JSON(http.StatusOK, common.APIResponse{
		Success: true,
		Data:    response,
		Message: "Login attempts retrieved successfully",
		Error:   nil,
	})
}

package logger

import (
	"net/http"
	"strconv"

	"github.com/adehusnim37/lihatin-go/controllers"
	httpPkg "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/adehusnim37/lihatin-go/repositories"
	"github.com/gin-gonic/gin"
)

// LoggerController handles logging operations
type LoggerController struct {
	*controllers.BaseController
	repo *repositories.LoggerRepository
}

// NewLoggerController creates a new logger controller
func NewLoggerController(base *controllers.BaseController) *LoggerController {
	loggerRepo := repositories.NewLoggerRepository(base.GormDB)
	return &LoggerController{
		BaseController: base,
		repo:           loggerRepo,
	}
}

// GetAllLogs retrieves all logs
func (c *LoggerController) GetAllLogs(ctx *gin.Context) {
	logs, err := c.repo.GetAllLogs()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to retrieve logs",
			Error:   map[string]string{"error": "Failed to retrieve logs, please try again later"},
		})
		return
	}
	ctx.JSON(http.StatusOK, common.APIResponse{
		Success: true,
		Data:    logs,
		Message: "Logs retrieved successfully",
		Error:   nil,
	})
}

// GetLogsByUsername retrieves logs by username
func (c *LoggerController) GetLogsByUsername(ctx *gin.Context) {
	username := ctx.Param("username")
	logs, err := c.repo.GetLogsByUsername(username)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to retrieve logs",
			Error:   map[string]string{"error": "Failed to retrieve logs, please try again later"},
		})
		return
	}
	ctx.JSON(http.StatusOK, common.APIResponse{
		Success: true,
		Data:    logs,
		Message: "Logs retrieved successfully",
		Error:   nil,
	})
}

func (c *LoggerController) GetLogsByShortLink(ctx *gin.Context) {
	code := ctx.Param("code")
	pageStr := ctx.DefaultQuery("page", "1")
	limitStr := ctx.DefaultQuery("limit", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 10
	}

	logs, totalCount, err := c.repo.GetLogsByShortLink(code, page, limit)
	if err != nil {
		httpPkg.HandleError(ctx, err, "")
		return
	}

	totalPages := int((totalCount + int64(limit) - 1) / int64(limit))

	responseData := map[string]interface{}{
		"logs": logs,
		"meta": map[string]interface{}{
			"current_page": page,
			"limit":        limit,
			"total_items":  totalCount,
			"total_pages":  totalPages,
		},
	}

	httpPkg.SendOKResponse(ctx, responseData, "Logs retrieved successfully")
}

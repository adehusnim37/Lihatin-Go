package controllers

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/adehusnim37/lihatin-go/repositories"
	"github.com/gin-gonic/gin"
)

// LoggerController handles logging operations
type LoggerController struct {
	*BaseController
	repo *repositories.LoggerRepository
}

// NewLoggerController creates a new logger controller
func NewLoggerController(base *BaseController) *LoggerController {
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

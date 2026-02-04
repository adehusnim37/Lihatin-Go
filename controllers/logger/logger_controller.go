package logger

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/controllers"
	"github.com/adehusnim37/lihatin-go/dto"
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

// GetAllLogs retrieves all logs with pagination
// Query params: page (default: 1), limit (default: 10, max: 100), sort (default: created_at), order_by (default: desc)
func (c *LoggerController) GetAllLogs(ctx *gin.Context) {
	userRole := ctx.GetString("role")
	userID := ctx.GetString("user_id")
	// Parse and validate pagination parameters
	page, limit, sort, orderBy, errs := httpPkg.PaginateValidateLogger(
		ctx.Query("page"),
		ctx.Query("limit"),
		ctx.Query("sort"),
		ctx.Query("order_by"),
	)

	if errs != nil {
		ctx.JSON(http.StatusBadRequest, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid pagination parameters",
			Error:   errs,
		})
		return
	}

	// Fetch logs from repository
	logs, totalCount, err := c.repo.GetAllLogs(page, limit, sort, orderBy, userRole, userID)
	if err != nil {
		httpPkg.HandleError(ctx, err, "")
		return
	}

	// Calculate pagination metadata
	totalPages := int((totalCount + int64(limit) - 1) / int64(limit))

	// Build response using DTO
	response := dto.PaginatedActivityLogsResponse{
		Logs:       dto.ToActivityLogResponseList(logs),
		TotalCount: totalCount,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}

	httpPkg.SendOKResponse(ctx, response, "Logs retrieved successfully")
}

// GetLogByID retrieves a single log by its ID
func (c *LoggerController) GetLogByID(ctx *gin.Context) {
	logID := ctx.Param("id")
	userRole := ctx.GetString("role")
	userID := ctx.GetString("user_id")

	// Fetch log from repository
	log, err := c.repo.GetLogByID(logID, userRole, userID)
	if err != nil {
		httpPkg.HandleError(ctx, err, "")
		return
	}

	httpPkg.SendOKResponse(ctx, dto.ToActivityLogResponse(log), "Log retrieved successfully")
}

// GetAllCountedLogs retrieves count of all logs by method type
func (c *LoggerController) GetAllCountedLogs(ctx *gin.Context) {
	userRole := ctx.GetString("role")
	userID := ctx.GetString("user_id")

	// Fetch counted logs from repository
	counts, err := c.repo.GetAllCountedLogs(userRole, userID)
	if err != nil {
		httpPkg.HandleError(ctx, err, "")
		return
	}

	httpPkg.SendOKResponse(ctx, counts, "Log counts retrieved successfully")
}

// GetLogsByUsername retrieves logs by username with pagination
// Query params: page (default: 1), limit (default: 10, max: 100), sort (default: created_at), order_by (default: desc)
func (c *LoggerController) GetLogsByUsername(ctx *gin.Context) {
	userRole := ctx.GetString("role")
	username := ctx.Param("username")
	userID := ctx.GetString("user_id")

	// Parse and validate pagination parameters
	page, limit, sort, orderBy, errs := httpPkg.PaginateValidateLogger(
		ctx.Query("page"),
		ctx.Query("limit"),
		ctx.Query("sort"),
		ctx.Query("order_by"),
	)

	if errs != nil {
		ctx.JSON(http.StatusBadRequest, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid pagination parameters",
			Error:   errs,
		})
		return
	}

	// Fetch logs from repository
	logs, totalCount, err := c.repo.GetLogsByUsername(username, page, limit, sort, orderBy, userRole, userID)
	if err != nil {
		httpPkg.HandleError(ctx, err, "")
		return
	}

	// Calculate pagination metadata
	totalPages := int((totalCount + int64(limit) - 1) / int64(limit))

	// Build response using DTO
	response := dto.PaginatedActivityLogsResponse{
		Logs:       dto.ToActivityLogResponseList(logs),
		TotalCount: totalCount,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}

	httpPkg.SendOKResponse(ctx, response, "Logs retrieved successfully")
}

// GetLogsByShortLink retrieves logs for a specific short link with pagination
// Query params: page (default: 1), limit (default: 10, max: 100), sort (default: created_at), order_by (default: desc)
func (c *LoggerController) GetLogsByShortLink(ctx *gin.Context) {
	code := ctx.Param("code")
	userRole := ctx.GetString("role")
	userID := ctx.GetString("user_id")
	// Parse and validate pagination parameters
	page, limit, sort, orderBy, errs := httpPkg.PaginateValidateLogger(
		ctx.Query("page"),
		ctx.Query("limit"),
		ctx.Query("sort"),
		ctx.Query("order_by"),
	)

	if errs != nil {
		ctx.JSON(http.StatusBadRequest, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid pagination parameters",
			Error:   errs,
		})
		return
	}

	// Fetch logs from repository
	logs, totalCount, err := c.repo.GetLogsByShortLink(code, page, limit, sort, orderBy, userRole, userID)
	if err != nil {
		httpPkg.HandleError(ctx, err, "")
		return
	}

	// Calculate pagination metadata
	totalPages := int((totalCount + int64(limit) - 1) / int64(limit))

	// Build response using DTO
	response := dto.PaginatedActivityLogsResponse{
		Logs:       dto.ToActivityLogResponseList(logs),
		TotalCount: totalCount,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}

	httpPkg.SendOKResponse(ctx, response, "Logs retrieved successfully")
}

// GetLogsWithFilter retrieves logs with advanced filtering and pagination
// Supports filtering by username, action, method, route, level, status_code, ip_address, date_from, date_to, api_key
// Query params: page, limit, sort, order_by + filter parameters
func (c *LoggerController) GetLogsWithFilter(ctx *gin.Context) {
	// Parse filter from query parameters
	var filter dto.ActivityLogFilter
	if err := ctx.ShouldBindQuery(&filter); err != nil {
		ctx.JSON(http.StatusBadRequest, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid filter parameters",
			Error:   map[string]string{"error": err.Error()},
		})
		return
	}

	// Parse and validate pagination parameters
	page, limit, sort, orderBy, errs := httpPkg.PaginateValidateLogger(
		ctx.Query("page"),
		ctx.Query("limit"),
		ctx.Query("sort"),
		ctx.Query("order_by"),
	)

	if errs != nil {
		ctx.JSON(http.StatusBadRequest, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid pagination parameters",
			Error:   errs,
		})
		return
	}

	// Fetch logs from repository with filters
	logs, totalCount, err := c.repo.GetLogsWithFilter(filter, page, limit, sort, orderBy)
	if err != nil {
		httpPkg.HandleError(ctx, err, "")
		return
	}

	// Calculate pagination metadata
	totalPages := int((totalCount + int64(limit) - 1) / int64(limit))

	// Build response using DTO
	response := dto.PaginatedActivityLogsResponse{
		Logs:       dto.ToActivityLogResponseList(logs),
		TotalCount: totalCount,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}

	httpPkg.SendOKResponse(ctx, response, "Logs retrieved successfully")
}

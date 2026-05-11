package logger

import (
    "net/http"

    "github.com/adehusnim37/lihatin-go/dto"
    httpPkg "github.com/adehusnim37/lihatin-go/internal/pkg/http"
    "github.com/adehusnim37/lihatin-go/models/common"
    "github.com/gin-gonic/gin"
)

// GetAllLogs retrieves all logs with pagination
// Query params: page (default: 1), limit (default: 10, max: 100), sort (default: created_at), order_by (default: desc)
func (c *Controller) GetAllLogs(ctx *gin.Context) {
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
func (c *Controller) GetLogByID(ctx *gin.Context) {
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

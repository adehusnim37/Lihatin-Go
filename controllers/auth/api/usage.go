package api

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/adehusnim37/lihatin-go/models/logging"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/gin-gonic/gin"
)

// GetAPIKeyActivityLogs retrieves activity logs associated with a specific API key
func (c *Controller) GetAPIKeyActivityLogs(ctx *gin.Context) {

	var reqId dto.APIKeyIDRequest
	if err := ctx.ShouldBindUri(&reqId); err != nil {
		utils.SendValidationError(ctx, err, &reqId)
		return
	}

	// Get user ID from context
	userID := ctx.GetString("user_id")
	userRole := ctx.GetString("role")
	pageStr := ctx.DefaultQuery("page", "1")
	limitStr := ctx.DefaultQuery("limit", "10")
	sort := ctx.DefaultQuery("sort", "created_at")
	orderBy := ctx.DefaultQuery("order_by", "desc")
	// Pagination parameters

	page, limit, sort, orderBy, vErrs := utils.PaginateValidate(pageStr, limitStr, sort, orderBy, utils.Role(userRole))
	if vErrs != nil {
		ctx.JSON(http.StatusBadRequest, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid pagination parameters",
			Error:   vErrs,
		})
		return
	}



	// Fetch activity logs for the API key
	apiKey, activityLogs, totalRecords, err := c.repo.GetAPIKeyRepository().APIKeyUsageHistory(reqId, userID, userRole, page, limit, sort, orderBy)
	if err != nil {
		utils.HandleError(ctx, err, userID)
		return
	}

	// Transform to DTOs
	var activityLogResponses []dto.APIKeyActivityLogResponse
    for _, log := range activityLogs {
        activityLogResponses = append(activityLogResponses, dto.APIKeyActivityLogResponse{
            ID:     log.ID,
            APIKey: &apiKey.Key,  // ✅ Fixed: Use single apiKey, not array
            UserID: &apiKey.UserID,
            ActivityLog: dto.ActivityLogResponse{
                ID:             log.ID,
                Level:          logging.LogLevel(log.Level),
                Message:        log.Message,
                Username:       log.Username,
                Timestamp:      log.Timestamp,
                IPAddress:      log.IPAddress,
                UserAgent:      log.UserAgent,
                BrowserInfo:    log.BrowserInfo,
                Action:         log.Action,
                Route:          log.Route,
                Method:         log.Method,
                StatusCode:     log.StatusCode,
                RequestBody:    log.RequestBody,
                RequestHeaders: log.RequestHeaders,
                QueryParams:    log.QueryParams,
                RouteParams:    log.RouteParams,
                ResponseBody:   log.ResponseBody,
                ResponseTime:   log.ResponseTime,
                CreatedAt:      log.CreatedAt,
                UpdatedAt:      log.UpdatedAt,
            },
        })
    }

    // ✅ ENHANCED: Calculate pagination metadata
    totalPages := int((totalRecords + int64(limit) - 1) / int64(limit))
    hasNext := page < totalPages
    hasPrev := page > 1

    response := dto.APIKeyActivityLogsResponse{
        ActivityLogs: activityLogResponses,
        TotalCount:   totalRecords,
        Page:         page,
        Limit:        limit,
        TotalPages:   totalPages,
        HasNext:      hasNext,
        HasPrev:      hasPrev,
        APIKeyInfo: dto.APIKeyBasicInfo{
            ID:         apiKey.ID,
            Name:       apiKey.Name,
            KeyPreview: utils.GetKeyPreview(apiKey.Key),
            IsActive:   apiKey.IsActive,
            CreatedAt:  apiKey.CreatedAt,
        },
    }


	utils.SendOKResponse(ctx, response, "API key activity logs fetched successfully")
}
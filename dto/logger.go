// dto/logger.go
package dto

import (
	"slices"
	"time"

	"github.com/adehusnim37/lihatin-go/models/logging"
)

// ActivityLogRequest for creating activity logs
type ActivityLogRequest struct {
	Level          string `json:"level" validate:"required,oneof=debug info warn error fatal"`
	Message        string `json:"message" validate:"required,min=1,max=500"`
	Username       string `json:"username" validate:"required,min=3,max=50"`
	APIKey         string `json:"api_key,omitempty" validate:"omitempty,max=50"`
	Action         string `json:"action" validate:"required,min=1,max=100"`
	Route          string `json:"route" validate:"required,min=1,max=255"`
	Method         string `json:"method" validate:"required,oneof=GET POST PUT DELETE PATCH OPTIONS HEAD"`
	StatusCode     int    `json:"status_code" validate:"required,min=100,max=599"`
	RequestBody    string `json:"request_body,omitempty"`
	RequestHeaders string `json:"request_headers,omitempty"`
	QueryParams    string `json:"query_params,omitempty"`
	RouteParams    string `json:"route_params,omitempty"`
	ResponseBody   string `json:"response_body,omitempty"`
	ResponseTime   int64  `json:"response_time,omitempty" validate:"min=0"`
}

// ActivityLogResponse represents activity log in API responses
type ActivityLogResponse struct {
	ID             string           `json:"id"`
	Level          logging.LogLevel `json:"level"`
	Message        string           `json:"message"`
	Username       string           `json:"username"`
	APIKey         string           `json:"api_key,omitempty"`
	Timestamp      time.Time        `json:"timestamp"`
	IPAddress      string           `json:"ip_address"`
	UserAgent      string           `json:"user_agent"`
	BrowserInfo    string           `json:"browser_info"`
	Action         string           `json:"action"`
	Route          string           `json:"route"`
	Method         string           `json:"method"`
	StatusCode     int              `json:"status_code"`
	RequestBody    string           `json:"request_body,omitempty"`
	RequestHeaders string           `json:"request_headers,omitempty"`
	QueryParams    string           `json:"query_params,omitempty"`
	RouteParams    string           `json:"route_params,omitempty"`
	ContextLocals  string           `json:"context_locals,omitempty"`
	ResponseBody   string           `json:"response_body,omitempty"`
	ResponseTime   int64            `json:"response_time,omitempty"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
}

// ActivityLogFilter for filtering logs
type ActivityLogFilter struct {
	Username   string           `json:"username,omitempty" form:"username"`
	Action     string           `json:"action,omitempty" form:"action"`
	Method     string           `json:"method,omitempty" form:"method" validate:"omitempty,oneof=GET POST PUT DELETE PATCH OPTIONS HEAD"`
	Route      string           `json:"route,omitempty" form:"route"`
	Level      logging.LogLevel `json:"level,omitempty" form:"level" validate:"omitempty,oneof=debug info warn error fatal"`
	StatusCode *int             `json:"status_code,omitempty" form:"status_code" validate:"omitempty,min=100,max=599"`
	IPAddress  string           `json:"ip_address,omitempty" form:"ip_address"`
	DateFrom   *time.Time       `json:"date_from,omitempty" form:"date_from" time_format:"2006-01-02T15:04:05Z07:00"`
	DateTo     *time.Time       `json:"date_to,omitempty" form:"date_to" time_format:"2006-01-02T15:04:05Z07:00"`
	APIKey     string           `json:"api_key,omitempty" form:"api_key"`
}

// PaginatedActivityLogsResponse represents paginated logs
type PaginatedActivityLogsResponse struct {
	Logs       []ActivityLogResponse `json:"logs"`
	TotalCount int64                 `json:"total_count"`
	Page       int                   `json:"page"`
	Limit      int                   `json:"limit"`
	TotalPages int                   `json:"total_pages"`
	HasNext    bool                  `json:"has_next"`
	HasPrev    bool                  `json:"has_prev"`
}

// ActivityLogStatsResponse for log statistics
type ActivityLogStatsResponse struct {
	TotalLogs           int64                      `json:"total_logs"`
	LogsByLevel         map[logging.LogLevel]int64 `json:"logs_by_level"`
	LogsByMethod        map[string]int64           `json:"logs_by_method"`
	LogsByStatus        map[int]int64              `json:"logs_by_status"`
	TopUsers            []UserLogCount             `json:"top_users"`
	TopRoutes           []RouteLogCount            `json:"top_routes"`
	LogsOverTime        []TimeLogCount             `json:"logs_over_time"`
	RecentErrors        []ActivityLogResponse      `json:"recent_errors"`
	AverageResponseTime int64                      `json:"average_response_time"`
	ErrorRate           float64                    `json:"error_rate"`
}

// TimeLogCount for time-series statistics
type TimeLogCount struct {
	Time  string `json:"time"`
	Count int64  `json:"count"`
}

// UserLogCount for user activity statistics
type UserLogCount struct {
	Username string `json:"username"`
	Count    int64  `json:"count"`
}

// RouteLogCount for route activity statistics
type RouteLogCount struct {
	Route  string `json:"route"`
	Method string `json:"method"`
	Count  int64  `json:"count"`
}

// LogLevelUpdateRequest for updating log level
type LogLevelUpdateRequest struct {
	Level logging.LogLevel `json:"level" validate:"required,oneof=debug info warn error fatal"`
}

// BulkDeleteLogsRequest for bulk deleting logs
type BulkDeleteLogsRequest struct {
	IDs      []string          `json:"ids" validate:"required,min=1"`
	DateFrom *time.Time        `json:"date_from,omitempty"`
	DateTo   *time.Time        `json:"date_to,omitempty"`
	Level    *logging.LogLevel `json:"level,omitempty"`
	Username string            `json:"username,omitempty"`
}

// LogExportRequest for exporting logs
type LogExportRequest struct {
	Format string            `json:"format" validate:"required,oneof=csv json excel pdf"`
	Filter ActivityLogFilter `json:"filter"`
	Fields []string          `json:"fields,omitempty"`
}

// Helper methods

// ToActivityLogResponse converts logging.ActivityLog to ActivityLogResponse
func ToActivityLogResponse(log *logging.ActivityLog) ActivityLogResponse {
	// Safely handle nullable APIKey pointer
	apiKey := ""
	if log.APIKey != nil {
		apiKey = *log.APIKey
	}

	return ActivityLogResponse{
		ID:             log.ID,
		Level:          logging.LogLevel(log.Level),
		Message:        log.Message,
		Username:       log.Username,
		APIKey:         apiKey,
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
		ContextLocals:  log.ContextLocals,
		ResponseBody:   log.ResponseBody,
		ResponseTime:   log.ResponseTime,
		CreatedAt:      log.CreatedAt,
		UpdatedAt:      log.UpdatedAt,
	}
}

// ToActivityLogResponseList converts slice of logging.ActivityLog to ActivityLogResponse
func ToActivityLogResponseList(logs []logging.ActivityLog) []ActivityLogResponse {
	responses := make([]ActivityLogResponse, len(logs))
	for i, log := range logs {
		responses[i] = ToActivityLogResponse(&log)
	}
	return responses
}

// ToActivityLog converts ActivityLogRequest to logging.ActivityLog
func (req *ActivityLogRequest) ToActivityLog() *logging.ActivityLog {
	log := &logging.ActivityLog{
		Level:          req.Level,
		Message:        req.Message,
		Username:       req.Username,
		Action:         req.Action,
		Route:          req.Route,
		Method:         req.Method,
		StatusCode:     req.StatusCode,
		APIKey:         &req.APIKey,
		RequestBody:    req.RequestBody,
		RequestHeaders: req.RequestHeaders,
		QueryParams:    req.QueryParams,
		RouteParams:    req.RouteParams,
		ResponseBody:   req.ResponseBody,
		ResponseTime:   req.ResponseTime,
		Timestamp:      time.Now(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	return log
}

// Validation helper functions

// ValidateLogLevel validates if the log level is valid
func ValidateLogLevel(level string) bool {
	switch logging.LogLevel(level) {
	case logging.LogLevelDebug, logging.LogLevelInfo, logging.LogLevelWarn, logging.LogLevelError, logging.LogLevelFatal:
		return true
	default:
		return false
	}
}

// ValidateHTTPMethod validates if the HTTP method is valid
func ValidateHTTPMethod(method string) bool {
	validMethods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD"}
	return slices.Contains(validMethods, method)
}

// ValidateStatusCode validates if the status code is valid
func ValidateStatusCode(code int) bool {
	return code >= 100 && code <= 599
}

package logging

import "time"

type ActivityLog struct {
	ID            string     `json:"id" gorm:"primaryKey"`
	Level         string     `json:"level" gorm:"size:20;not null" validate:"required,min=1,max=50"`
	Message       string     `json:"message" gorm:"size:500;not null" validate:"required,min=1,max=255"`
	Username      string     `json:"username" gorm:"size:50" validate:"required,min=3,max=50"`
	Timestamp     time.Time  `json:"timestamp" gorm:"not null"`
	IPAddress     string     `json:"ip_address" gorm:"size:45" validate:"required,min=1,max=50"` // IPv4/IPv6
	UserAgent     string     `json:"user_agent" gorm:"type:text"`
	BrowserInfo   string     `json:"browser_info" gorm:"size:500"`
	Action        string     `json:"action" gorm:"size:100;not null" validate:"required,min=1,max=100"`
	Route         string     `json:"route" gorm:"size:255;not null" validate:"required,min=1,max=100"`
	Method        string     `json:"method" gorm:"size:10;not null" validate:"required,min=1,max=10"`
	StatusCode    int        `json:"status_code" gorm:"not null"`
	RequestBody   string     `json:"request_body,omitempty" gorm:"type:text"`
	QueryParams   string     `json:"query_params,omitempty" gorm:"type:text"`
	RouteParams   string     `json:"route_params,omitempty" gorm:"type:text"`
	ContextLocals string     `json:"context_locals,omitempty" gorm:"type:text"`
	ResponseTime  int64      `json:"response_time,omitempty"` // Response time in milliseconds
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	DeletedAt     *time.Time `json:"deleted_at,omitempty" gorm:"index"`
}

// TableName specifies the table name for GORM
func (ActivityLog) TableName() string {
	return "activity_logs"
}

// LogLevel represents log levels
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
	LogLevelFatal LogLevel = "fatal"
)

// ActivityLogFilter for filtering logs
type ActivityLogFilter struct {
	Username   string    `json:"username,omitempty"`
	Action     string    `json:"action,omitempty"`
	Method     string    `json:"method,omitempty"`
	Route      string    `json:"route,omitempty"`
	Level      LogLevel  `json:"level,omitempty"`
	StatusCode *int      `json:"status_code,omitempty"`
	IPAddress  string    `json:"ip_address,omitempty"`
	DateFrom   time.Time `json:"date_from,omitempty"`
	DateTo     time.Time `json:"date_to,omitempty"`
}

// PaginatedActivityLogsResponse represents paginated logs
type PaginatedActivityLogsResponse struct {
	Logs       []ActivityLog `json:"logs"`
	TotalCount int64         `json:"total_count"`
	Page       int           `json:"page"`
	Limit      int           `json:"limit"`
	TotalPages int           `json:"total_pages"`
}

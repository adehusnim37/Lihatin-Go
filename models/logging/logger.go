package logging

import "time"

type ActivityLog struct {
	ID             string     `json:"id" gorm:"primaryKey;column:id"`
	Level          string     `json:"level" gorm:"column:level;size:20;not null;index:idx_level_created_at,composite:level" validate:"required,min=1,max=50"`
	Message        string     `json:"message" gorm:"column:message;size:500;not null" validate:"required,min=1,max=255"`
	Username       string     `json:"username" gorm:"column:username;size:50;index:idx_username_created_at,composite:username" validate:"required,min=3,max=50"`
	APIKey         *string    `json:"api_key,omitempty" gorm:"column:api_key;size:100;index"` // Foreign key to api_keys table (nullable for optional auth)
	UserID         *string    `json:"user_id,omitempty" gorm:"column:user_id;size:191;index"` // Foreign key to users table (nullable for optional auth)
	Timestamp      time.Time  `json:"timestamp" gorm:"column:timestamp;not null;index:idx_timestamp"`
	IPAddress      string     `json:"ip_address" gorm:"column:ip_address;size:45;index:idx_ip_address_created_at,composite:ip_address" validate:"required,min=1,max=50"` // IPv4/IPv6
	UserAgent      string     `json:"user_agent" gorm:"column:user_agent;type:text"`
	BrowserInfo    string     `json:"browser_info" gorm:"column:browser_info;size:500"`
	Action         string     `json:"action" gorm:"column:action;size:100;not null;index:idx_action_created_at,composite:action" validate:"required,min=1,max=100"`
	Route          string     `json:"route" gorm:"column:route;size:255;not null;index:idx_route_created_at,composite:route" validate:"required,min=1,max=100"`
	Method         string     `json:"method" gorm:"column:method;size:10;not null" validate:"required,min=1,max=10"`
	StatusCode     int        `json:"status_code" gorm:"column:status_code;not null;index:idx_status_code_created_at,composite:status_code"`
	RequestBody    string     `json:"request_body,omitempty" gorm:"column:request_body;type:text"`
	RequestHeaders string     `json:"request_headers,omitempty" gorm:"column:request_headers;type:text"`
	QueryParams    string     `json:"query_params,omitempty" gorm:"column:query_params;type:text"`
	RouteParams    string     `json:"route_params,omitempty" gorm:"column:route_params;type:text"`
	ContextLocals  string     `json:"context_locals,omitempty" gorm:"column:context_locals;type:text"`
	ResponseBody   string     `json:"response_body,omitempty" gorm:"column:response_body;type:text"`
	ResponseTime   int64      `json:"response_time,omitempty" gorm:"column:response_time"` // Response time in milliseconds
	CreatedAt      time.Time  `json:"created_at" gorm:"column:created_at;index:idx_username_created_at,composite:created_at;index:idx_level_created_at,composite:created_at;index:idx_status_code_created_at,composite:created_at;index:idx_action_created_at,composite:created_at;index:idx_route_created_at,composite:created_at;index:idx_ip_address_created_at,composite:created_at"`
	UpdatedAt      time.Time  `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty" gorm:"column:deleted_at;index"`
}

// TableName specifies the table name for GORM
func (ActivityLog) TableName() string {
	return "activitylog"
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

package logging

import "time"

type ActivityLog struct {
	ID             string     `json:"id" gorm:"primaryKey;column:id"`
	Level          string     `json:"level" gorm:"column:level;size:20;not null" validate:"required,min=1,max=50"`
	Message        string     `json:"message" gorm:"column:message;size:500;not null" validate:"required,min=1,max=255"`
	Username       string     `json:"username" gorm:"column:username;size:50" validate:"required,min=3,max=50"`
	APIKey         *string    `json:"api_key,omitempty" gorm:"column:apikey;size:100;index"` // Foreign key to api_keys table (nullable for optional auth)
	UserID         *string    `json:"user_id,omitempty" gorm:"column:userid;size:191;index"` // Foreign key to users table (nullable for optional auth)
	Timestamp      time.Time  `json:"timestamp" gorm:"column:timestamp;not null"`
	IPAddress      string     `json:"ip_address" gorm:"column:ipaddress;size:45" validate:"required,min=1,max=50"` // IPv4/IPv6
	UserAgent      string     `json:"user_agent" gorm:"column:useragent;type:text"`
	BrowserInfo    string     `json:"browser_info" gorm:"column:browserinfo;size:500"`
	Action         string     `json:"action" gorm:"column:action;size:100;not null" validate:"required,min=1,max=100"`
	Route          string     `json:"route" gorm:"column:route;size:255;not null" validate:"required,min=1,max=100"`
	Method         string     `json:"method" gorm:"column:method;size:10;not null" validate:"required,min=1,max=10"`
	StatusCode     int        `json:"status_code" gorm:"column:statuscode;not null"`
	RequestBody    string     `json:"request_body,omitempty" gorm:"column:requestbody;type:text"`
	RequestHeaders string     `json:"request_headers,omitempty" gorm:"column:requestheaders;type:text"`
	QueryParams    string     `json:"query_params,omitempty" gorm:"column:queryparams;type:text"`
	RouteParams    string     `json:"route_params,omitempty" gorm:"column:routeparams;type:text"`
	ContextLocals  string     `json:"context_locals,omitempty" gorm:"column:contextlocals;type:text"`
	ResponseBody   string     `json:"response_body,omitempty" gorm:"column:responsebody;type:text"`
	ResponseTime   int64      `json:"response_time,omitempty" gorm:"column:responsetime"` // Response time in milliseconds
	CreatedAt      time.Time  `json:"created_at" gorm:"column:createdat"`
	UpdatedAt      time.Time  `json:"updated_at" gorm:"column:updatedat"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty" gorm:"column:deletedat;index"`
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

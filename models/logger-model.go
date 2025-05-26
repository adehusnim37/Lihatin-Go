package models

import "time"

type LoggerUser struct {
	ID            string     `json:"id" validate:"required,min=1,max=50"`
	Level         string     `json:"level" validate:"required,min=1,max=50"`
	Message       string     `json:"message" validate:"required,min=1,max=255"`
	Username      string     `json:"username" validate:"required,min=3,max=50"`
	Timestamp     string     `json:"timestamp" validate:"required,min=1,max=50"`
	IPAddress     string     `json:"ip_address" validate:"required,min=1,max=50"`
	UserAgent     string     `json:"user_agent" validate:"required,min=1,max=255"`
	BrowserInfo   string     `json:"browser_info" validate:"required,min=1,max=255"`
	Action        string     `json:"action" validate:"required,min=1,max=100"`
	Route         string     `json:"route" validate:"required,min=1,max=100"`
	Method        string     `json:"method" validate:"required,min=1,max=10"`
	StatusCode    int        `json:"status_code" validate:"required"`
	RequestBody   string     `json:"request_body,omitempty"`
	QueryParams   string     `json:"query_params,omitempty"`
	RouteParams   string     `json:"route_params,omitempty"`
	ContextLocals string     `json:"context_locals,omitempty"`
	ResponseTime  int64      `json:"response_time,omitempty"` // Response time in milliseconds
	CreatedAt     *time.Time `json:"created_at,omitempty" default:"time.Now()"`
	UpdatedAt     *time.Time `json:"updated_at,omitempty" default:"time.Now()"`
	DeletedAt     *time.Time `json:"deleted_at,omitempty" default:"null"`
}

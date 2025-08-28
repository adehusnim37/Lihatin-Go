package common

// APIResponse represents standard API response format
type APIResponse struct {
	Success bool              `json:"success"`
	Data    any               `json:"data,omitempty"`
	Message string            `json:"message,omitempty"`
	Error   map[string]string `json:"error,omitempty"`
}

// Pagination represents pagination metadata
type Pagination struct {
	Total       int `json:"total"`
	CurrentPage int `json:"current_page"`
	PerPage     int `json:"per_page"`
	TotalPages  int `json:"total_pages"`
}

// PaginatedResponse represents paginated API response
type PaginatedResponse struct {
	Success    bool              `json:"success"`
	Data       any               `json:"data"`
	Pagination *Pagination       `json:"pagination,omitempty"`
	Message    string            `json:"message,omitempty"`
	Error      map[string]string `json:"error,omitempty"`
}

// ErrorDetail represents detailed error information
type ErrorDetail struct {
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// ValidationErrorResponse represents validation errors
type ValidationErrorResponse struct {
	Success bool          `json:"success" example:"false"`
	Message string        `json:"message" example:"Validation failed"`
	Errors  []ErrorDetail `json:"errors"`
}

package dto

import "time"

// APIKeyRequest represents the request to create a new API key
type APIKeyRequest struct {
	Name        string     `json:"name" binding:"required,min=3,max=100,no_special"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	Permissions []string   `json:"permissions,omitempty" binding:"dive,oneof=read write delete update"`
}

type UpdateAPIKeyRequest struct {
	Name        *string    `json:"name,omitempty" binding:"omitempty,min=3,max=100,no_special"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty" binding:"omitempty,gtfield=Now,datetime,"` // Must be in the future
	Permissions []string   `json:"permissions,omitempty" binding:"dive,oneof=read write delete update"`
	IsActive    *bool      `json:"is_active,omitempty"`
}

type APIKeyIDRequest struct {
	ID string `json:"id" binding:"required,uuid4" uri:"id"`
}

// APIKeyResponse represents the API key response (without sensitive data)
type APIKeyResponse struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	KeyPreview  string     `json:"key_preview"` // Only first 8 characters + "..."
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	IsActive    bool       `json:"is_active"`
	Permissions []string   `json:"permissions"`
	CreatedAt   time.Time  `json:"created_at"`
}

// CreateAPIKeyResponse represents the response when creating a new API key
type CreateAPIKeyResponse struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	Permissions []string   `json:"permissions"`
	Key         string     `json:"key"`     // Only shown once during creation
	Warning     string     `json:"warning"` // Warning message
}

// PaginatedAPIKeysResponse represents paginated API keys
type PaginatedAPIKeysResponse struct {
	APIKeys    []APIKeyResponse `json:"api_keys"`
	TotalCount int64            `json:"total_count"`
	Page       int              `json:"page"`
	Limit      int              `json:"limit"`
	TotalPages int              `json:"total_pages"`
}

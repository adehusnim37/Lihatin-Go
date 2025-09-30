package dto

import "time"

type UpdateAPIKeyRequest struct {
	Name        *string    `json:"name,omitempty" binding:"omitempty,min=3,max=100,no_special"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty" binding:"omitempty,gtfield=CreatedAt"` // Must be in the future
	Permissions []string   `json:"permissions,omitempty" binding:"dive,oneof=read write delete update"`
	IsActive    *bool      `json:"is_active,omitempty"`
	BlockedIPs  []string   `json:"blocked_ips,omitempty" binding:"dive,ip"`
	AllowedIPs  []string   `json:"allowed_ips,omitempty" binding:"dive,ip"`
	LimitUsage  *int64     `json:"limit_usage,omitempty" binding:"omitempty,gte=0"` // nil means unlimited
}

type APIKeyIDRequest struct {
	ID string `json:"id" binding:"required,uuid4" uri:"id"`
}

type ActivateAccountRequest struct {
}

// APIKeyResponse represents the API key response (without sensitive data)
type APIKeyResponse struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	KeyPreview  string     `json:"key_preview"` // Only first 8 characters + "..."
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	LimitUsage  *int64     `json:"limit_usage,omitempty"`
	UsageCount  int64      `json:"usage_count"`
	LastIPUsed  *string    `json:"last_ip_used,omitempty"`
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
	BlockedIPs  []string   `json:"blocked_ips,omitempty"`
	AllowedIPs  []string   `json:"allowed_ips,omitempty"`
	LimitUsage  *int64     `json:"limit_usage,omitempty"`
	UsageCount  int64      `json:"usage_count"`
	IsActive    bool       `json:"is_active"`
	// Sensitive info
	Key     string `json:"key"`     // Only shown once during creation
	Warning string `json:"warning"` // Warning message
}

// CreateAPIKeyRequest represents the request to create a new API key
type CreateAPIKeyRequest struct {
	Name        string     `json:"name" binding:"required,min=3,max=100,no_special"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty" binding:"omitempty,gtfield=CreatedAt"` // Must be in the future
	Permissions []string   `json:"permissions,omitempty" binding:"dive,oneof=read write delete update"`
	BlockedIPs  []string   `json:"blocked_ips,omitempty" binding:"dive,ip"`
	AllowedIPs  []string   `json:"allowed_ips,omitempty" binding:"dive,ip"`
	LimitUsage  *int64     `json:"limit_usage,omitempty" binding:"omitempty,gte=0"` // nil means unlimited
	IsActive    bool       `json:"is_active" binding:"omitempty"`
}

// Enhanced response with API key info and pagination
type APIKeyActivityLogsResponse struct {
    ActivityLogs []APIKeyActivityLogResponse `json:"activity_logs"`
    TotalCount   int64                       `json:"total_count"`
    Page         int                         `json:"page"`
    Limit        int                         `json:"limit"`
    TotalPages   int                         `json:"total_pages"`
    HasNext      bool                        `json:"has_next"`
    HasPrev      bool                        `json:"has_prev"`
    APIKeyInfo   APIKeyBasicInfo             `json:"api_key_info"`
}

// Basic API key info for responses
type APIKeyBasicInfo struct {
    ID         string    `json:"id"`
    Name       string    `json:"name"`
    KeyPreview string    `json:"key_preview"`
    IsActive   bool      `json:"is_active"`
    CreatedAt  time.Time `json:"created_at"`
}

// Enhanced activity log response
type APIKeyActivityLogResponse struct {
    ID          string              `json:"id"`
    APIKey      *string             `json:"api_key,omitempty"`
    UserID      *string             `json:"user_id,omitempty"`
    ActivityLog ActivityLogResponse `json:"activity_log"`
}

// PaginatedAPIKeysResponse represents paginated API keys
type PaginatedAPIKeysResponse struct {
	APIKeys    []APIKeyResponse `json:"api_keys"`
	TotalCount int64            `json:"total_count"`
	Page       int              `json:"page"`
	Limit      int              `json:"limit"`
	TotalPages int              `json:"total_pages"`
}

// APIKeyRefreshResponse represents the response when refreshing an API key
type APIKeyRefreshResponse struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	KeyPreview  string           `json:"key_preview"`
	IsActive    bool             `json:"is_active"`
	ExpiresAt   *time.Time       `json:"expires_at,omitempty"`
	Permissions []string         `json:"permissions"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
	Secret      APIKeySecretInfo `json:"secret"`
	Usage       APIKeyUsageInfo  `json:"usage"`
}

// APIKeySecretInfo contains sensitive API key information
type APIKeySecretInfo struct {
	FullAPIKey string `json:"full_api_key"`
	Warning    string `json:"warning"`
	Format     string `json:"format"`
	ExpiresIn  string `json:"expires_in"`
}

// APIKeyUsageInfo contains usage-related information about the API key
type APIKeyUsageInfo struct {
	LastUsedAt         *time.Time `json:"last_used_at,omitempty"`
	LastIPUsed         *string    `json:"last_ip_used,omitempty"`
	IsRegenerated      bool       `json:"is_regenerated"`
	PreviousUsageReset bool       `json:"previous_usage_reset"`
}

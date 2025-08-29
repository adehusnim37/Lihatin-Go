package user

import (
	"time"
)

// APIKey represents an API key for user authentication
type APIKey struct {
	ID          string     `json:"id" gorm:"primaryKey"`
	UserID      string     `json:"user_id" gorm:"size:191;not null;index"`
	Name        string     `json:"name" gorm:"size:100;not null" validate:"required,min=3,max=100"`
	Key         string     `json:"key" gorm:"uniqueIndex;size:255;not null"`
	KeyHash     string     `json:"-" gorm:"column:key_hash;size:255;not null"` // Store hashed version
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	IsActive    bool       `json:"is_active" gorm:"default:true"`
	Permissions []string   `json:"permissions" gorm:"type:json"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty" gorm:"index"`

	// Associations
	User *User `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

// TableName specifies the table name for GORM
func (APIKey) TableName() string {
	return "api_keys"
}

// KeyPreview returns the first 8 characters of the key followed by "..."
func (a *APIKey) KeyPreview() string {
	if len(a.Key) <= 8 {
		return a.Key + "..."
	}
	return a.Key[:8] + "..."
}

// APIKeyRequest represents the request to create a new API key
type APIKeyRequest struct {
	Name        string     `json:"name" validate:"required,min=3,max=100"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	Permissions []string   `json:"permissions,omitempty"`
}

type UpdateAPIKeyRequest struct {
	Name        *string    `json:"name,omitempty" validate:"omitempty,min=3,max=100"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	Permissions []string   `json:"permissions,omitempty"`
	IsActive    *bool      `json:"is_active,omitempty"`
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
	APIKey *APIKeyResponse `json:"api_key"`
	Key    string          `json:"key"` // Only shown once during creation
}

// PaginatedAPIKeysResponse represents paginated API keys
type PaginatedAPIKeysResponse struct {
	APIKeys    []APIKeyResponse `json:"api_keys"`
	TotalCount int64            `json:"total_count"`
	Page       int              `json:"page"`
	Limit      int              `json:"limit"`
	TotalPages int              `json:"total_pages"`
}

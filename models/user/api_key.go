package user

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/adehusnim37/lihatin-go/internal/pkg/auth"
	apperrors "github.com/adehusnim37/lihatin-go/internal/pkg/errors"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
)

// PermissionsList is a custom type for handling JSON serialization of permissions
type PermissionsList []string

// Value implements the driver.Valuer interface for database storage
func (p PermissionsList) Value() (driver.Value, error) {
	if len(p) == 0 {
		return json.Marshal([]string{})
	}
	return json.Marshal([]string(p))
}

// Scan implements the sql.Scanner interface for database retrieval
func (p *PermissionsList) Scan(value interface{}) error {
	if value == nil {
		*p = PermissionsList{}
		return nil
	}

	var permissions []string
	switch v := value.(type) {
	case []byte:
		err := json.Unmarshal(v, &permissions)
		*p = PermissionsList(permissions)
		return err
	case string:
		err := json.Unmarshal([]byte(v), &permissions)
		*p = PermissionsList(permissions)
		return err
	default:
		*p = PermissionsList{}
		return nil
	}
}

// APIKey represents an API key for user authentication
type APIKey struct {
	ID          string          `json:"id" gorm:"primaryKey"`
	UserID      string          `json:"user_id" gorm:"size:191;not null;index"`
	Name        string          `json:"name" gorm:"size:100;not null" validate:"required,min=3,max=100"`
	Key         string          `json:"key" gorm:"uniqueIndex;size:255;not null"`
	KeyHash     string          `json:"-" gorm:"column:key_hash;size:255;not null"` // Store hashed version
	UsageCount  int64           `json:"usage_count" gorm:"default:0"`
	LimitUsage  *int64          `json:"limit_usage,omitempty" gorm:"default:null"` // nil means unlimited
	LastIPUsed  *string         `json:"last_ip_used,omitempty" gorm:"size:45"`     // IPv6 max length is 45 characters
	AllowedIPs  IPList          `json:"allowed_ips,omitempty" gorm:"type:json"`    // List of allowed IPs in JSON format
	BlockedIPs  IPList          `json:"blocked_ips,omitempty" gorm:"type:json"`    // List of blocked IPs in JSON format
	LastUsedAt  *time.Time      `json:"last_used_at,omitempty"`
	ExpiresAt   *time.Time      `json:"expires_at,omitempty"`
	IsActive    bool            `json:"is_active" gorm:"default:true"`
	Permissions PermissionsList `json:"permissions" gorm:"type:json"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	DeletedAt   *time.Time      `json:"deleted_at,omitempty" gorm:"index"`

	// Associations
	User *User `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

// TableName specifies the table name for GORM
func (APIKey) TableName() string {
	return "api_keys"
}

// KeyPreview returns a preview of the API key for safe display
func (a *APIKey) KeyPreview() string {
	return auth.GetKeyPreview(a.Key)
}

// models/user/api_key.go - Enhanced IPList with validation

type IPList []string

// Value implements driver.Valuer for database storage
func (ips IPList) Value() (driver.Value, error) {
	if len(ips) == 0 {
		return []byte("[]"), nil // Empty JSON array
	}

	// ✅ Add validation before marshaling
	if slices.Contains(ips, "") {
		logger.Logger.Error("Empty IP address validation failed", "ips", ips)
		return nil, apperrors.ErrEmptyIPAddress
	}

	jsonBytes, err := json.Marshal([]string(ips))
	if err != nil {
		logger.Logger.Error("Failed to marshal IPList", "error", err)
		return nil, fmt.Errorf("failed to marshal IPList: %w", err)
	}

	return jsonBytes, nil
}

// Scan implements sql.Scanner for database retrieval
func (ips *IPList) Scan(value interface{}) error {
	if value == nil {
		*ips = IPList{}
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into IPList", value)
	}

	// Handle empty or null JSON
	if len(bytes) == 0 || string(bytes) == "null" {
		*ips = IPList{}
		return nil
	}

	var stringSlice []string
	if err := json.Unmarshal(bytes, &stringSlice); err != nil {
		return fmt.Errorf("failed to unmarshal IPList: %w", err)
	}

	*ips = IPList(stringSlice)
	return nil
}

// Helper methods
func (ips IPList) Contains(ip string) bool {
	return slices.Contains(ips, ip)
}

func (ips IPList) IsEmpty() bool {
	return len(ips) == 0
}

// Add method for validation
func (ips IPList) IsValid() bool {
	return !slices.Contains(ips, "")
}

package user

import (
	"time"

	"github.com/adehusnim37/lihatin-go/utils"
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

// KeyPreview returns a preview of the API key for safe display
func (a *APIKey) KeyPreview() string {
	return utils.GetKeyPreview(a.Key)
}

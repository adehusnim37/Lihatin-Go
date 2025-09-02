package shortlink

import (
	"time"

	"gorm.io/gorm"
)

// ShortLink represents the main short link entity
type ShortLink struct {
	ID          string         `json:"id" gorm:"primaryKey"`                     // Changed to string for consistency
	UserID      *string        `json:"user_id,omitempty" gorm:"size:191;index"`  // Foreign key to users table (nullable for optional auth)
	ShortCode   string         `json:"short_code" gorm:"uniqueIndex;size:10;not null"`
	OriginalURL string         `json:"original_url,omitempty" gorm:"type:text;not null"`
	Title       string         `json:"title,omitempty" gorm:"size:255"`
	Description string         `json:"description,omitempty" gorm:"type:text"`
	IsActive    bool           `json:"is_active" gorm:"default:true"`
	ExpiresAt   *time.Time     `json:"expires_at" gorm:"index"`
	CreatedAt   time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	// Relationships - Note: User tidak di-include untuk menghindari circular import
	// Gunakan service layer untuk populate user data jika diperlukan
	Detail *ShortLinkDetail `json:"detail,omitempty" gorm:"foreignKey:ShortLinkID"`
	Views  []ViewLinkDetail `json:"views,omitempty" gorm:"foreignKey:ShortLinkID"`
}

// TableName specifies the table name for GORM
func (ShortLink) TableName() string {
	return "short_links"
}

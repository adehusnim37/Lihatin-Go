package shortlink

import (
	"time"

	"gorm.io/gorm"
)

// ShortLink represents the main short link entity
type ShortLink struct {
	ID          string         `json:"id" gorm:"primaryKey"`                                                                     // Changed to string for consistency
	UserID      *string        `json:"user_id,omitempty" gorm:"size:191;index;index:idx_short_links_user_created_at,priority:1"` // Foreign key to users table (nullable for optional auth)
	ShortCode   string         `json:"short_code" gorm:"uniqueIndex;size:100;not null"`
	OriginalURL string         `json:"original_url" gorm:"type:text;not null"`
	Title       string         `json:"title,omitempty" gorm:"size:255"`
	Description string         `json:"description,omitempty" gorm:"type:text"`
	IsActive    bool           `json:"is_active" gorm:"default:true;index"`
	ExpiresAt   *time.Time     `json:"expires_at" gorm:"index"`
	CreatedAt   time.Time      `json:"created_at" gorm:"autoCreateTime;index:idx_short_links_user_created_at,priority:3,sort:desc"`
	UpdatedAt   time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at" gorm:"index;index:idx_short_links_user_created_at,priority:2"`

	// Relationships - Note: User tidak di-include untuk menghindari circular import
	// Gunakan service layer untuk populate user data jika diperlukan
	Detail *ShortLinkDetail `json:"detail,omitempty" gorm:"foreignKey:ShortLinkID;constraint:OnDelete:CASCADE"`
	Views  []ViewLinkDetail `json:"views,omitempty" gorm:"foreignKey:ShortLinkID;constraint:OnDelete:CASCADE"`
}

// TableName specifies the table name for GORM
func (ShortLink) TableName() string {
	return "short_links"
}

package shortlink

import (
	"time"

	"gorm.io/gorm"
)

// ShortLink represents the main short link entity
type ShortLink struct {
	ID          string         `json:"id" gorm:"primaryKey"`                    // Changed to string for consistency
	UserID      string         `json:"user_id,omitempty" gorm:"size:191;index"` // Foreign key to users table (string to match User.ID)
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

// ShortLinkResponse represents short link data for API responses
type ShortLinkResponse struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	ShortCode   string     `json:"short_code"`
	OriginalURL string     `json:"original_url"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	IsActive    bool       `json:"is_active"`
	ExpiresAt   *time.Time `json:"expires_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ClickCount  int        `json:"click_count,omitempty"`
}

// CreateShortLinkRequest represents request to create short link
type CreateShortLinkRequest struct {
	UserID      string     `json:"user_id,omitempty"`
	OriginalURL string     `json:"original_url" validate:"required,url"`
	Title       string     `json:"title,omitempty" validate:"max=255"`
	Description string     `json:"description,omitempty"`
	CustomCode  string     `json:"custom_code,omitempty" validate:"max=10,alphanum"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// UpdateShortLinkRequest represents request to update short link
type UpdateShortLinkRequest struct {
	Title       *string    `json:"title,omitempty" validate:"omitempty,max=255"`
	Description *string    `json:"description,omitempty"`
	IsActive    *bool      `json:"is_active,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// PaginatedShortLinksResponse represents paginated short links
type PaginatedShortLinksResponse struct {
	ShortLinks []ShortLinkResponse `json:"short_links"`
	TotalCount int64               `json:"total_count"`
	Page       int                 `json:"page"`
	Limit      int                 `json:"limit"`
	TotalPages int                 `json:"total_pages"`
}

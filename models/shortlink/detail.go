package shortlink

import (
	"time"

	"gorm.io/gorm"
)

// ShortLinkDetail stores additional metadata and settings for short links
type ShortLinkDetail struct {
	ID            string         `json:"id" gorm:"primaryKey"`                         // Changed to string for consistency
	ShortLinkID   string         `json:"short_link_id" gorm:"size:191;not null;index"` // Foreign key, changed to string
	Password      string         `json:"password,omitempty" gorm:"size:255"`           // For password protection
	ClickLimit    int            `json:"click_limit" gorm:"default:0"`                 // 0 means unlimited
	CurrentClicks int            `json:"current_clicks" gorm:"default:0"`
	EnableStats   bool           `json:"enable_stats" gorm:"default:true"`
	CustomDomain  string         `json:"custom_domain" gorm:"size:255"`
	UTMSource     string         `json:"utm_source" gorm:"size:100"`
	UTMMedium     string         `json:"utm_medium" gorm:"size:100"`
	UTMCampaign   string         `json:"utm_campaign" gorm:"size:100"`
	UTMTerm       string         `json:"utm_term" gorm:"size:100"`
	UTMContent    string         `json:"utm_content" gorm:"size:100"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	// Relationships
	ShortLink ShortLink `json:"short_link,omitempty" gorm:"foreignKey:ShortLinkID;references:ID"`
}

// TableName specifies the table name for GORM
func (ShortLinkDetail) TableName() string {
	return "short_link_details"
}

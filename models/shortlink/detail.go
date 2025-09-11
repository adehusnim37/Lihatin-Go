package shortlink

import (
	"time"

	"gorm.io/gorm"
)

// ShortLinkDetail stores additional metadata and settings for short links
type ShortLinkDetail struct {
	ID                     string         `json:"id" gorm:"primaryKey"`                         // Changed to string for consistency
	ShortLinkID            string         `json:"short_link_id" gorm:"size:191;not null;index"` // Foreign key, changed to string
	Passcode               int            `json:"passcode,omitempty" validate:"max=6,numeric"`  // Removed gorm:"size:6"
	ClickLimit             int            `json:"click_limit" gorm:"default:0"`                 // 0 means unlimited
	CurrentClicks          int            `json:"current_clicks" gorm:"default:0"`
	EnableStats            bool           `json:"enable_stats" gorm:"default:true"`
	PasscodeToken          *string        `json:"passcode_token,omitempty" gorm:"size:191;uniqueIndex"`
	PasscodeTokenExpiresAt time.Time      `json:"passcode_token_expires_at,omitempty"`
	IsBanned               bool           `json:"is_banned" gorm:"default:false"`
	BannedReason           string         `json:"banned_reason,omitempty" gorm:"size:255"`
	BannedBy               *string        `json:"banned_by,omitempty" gorm:"size:191"` // Admin user ID who banned the link
	CustomDomain           string         `json:"custom_domain,omitempty" gorm:"size:255"`
	UTMSource              string         `json:"utm_source,omitempty" gorm:"size:100"`
	UTMMedium              string         `json:"utm_medium,omitempty" gorm:"size:100"`
	UTMCampaign            string         `json:"utm_campaign,omitempty" gorm:"size:100"`
	UTMTerm                string         `json:"utm_term,omitempty" gorm:"size:100"`
	UTMContent             string         `json:"utm_content,omitempty" gorm:"size:100"`
	CreatedAt              time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt              time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt              gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	// Relationships
	ShortLink ShortLink `json:"short_link,omitempty" gorm:"foreignKey:ShortLinkID;references:ID"`
}

// TableName specifies the table name for GORM
func (ShortLinkDetail) TableName() string {
	return "short_link_details"
}

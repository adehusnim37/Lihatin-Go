package shortlink

import (
	"time"

	"gorm.io/gorm"
)

// ViewLinkDetail tracks individual clicks/views of short links
type ViewLinkDetail struct {
	ID          string         `json:"id" gorm:"primaryKey"`                                                                                                       // Changed to string for consistency
	ShortLinkID string         `json:"short_link_id" gorm:"size:191;not null;index;index:idx_views_link_clicked_at,priority:1;index:idx_views_link_ip,priority:1"` // Foreign key, changed to string
	IPAddress   string         `json:"ip_address" gorm:"size:45;index;index:idx_views_link_ip,priority:3"`                                                         // IPv4/IPv6
	UserAgent   string         `json:"user_agent" gorm:"type:text"`
	Referer     string         `json:"referer" gorm:"size:500"`
	Country     string         `json:"country" gorm:"size:100"`
	City        string         `json:"city" gorm:"size:100"`
	Device      string         `json:"device" gorm:"size:100"`
	Browser     string         `json:"browser" gorm:"size:100"`
	OS          string         `json:"os" gorm:"size:100"`
	ClickedAt   time.Time      `json:"clicked_at" gorm:"index;index:idx_views_link_clicked_at,priority:3,sort:desc"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at" gorm:"index;index:idx_views_link_clicked_at,priority:2;index:idx_views_link_ip,priority:2"`

	// Relationships
	ShortLink ShortLink `json:"short_link,omitempty" gorm:"foreignKey:ShortLinkID;references:ID"`
}

// TableName specifies the table name for GORM
func (ViewLinkDetail) TableName() string {
	return "view_link_details"
}

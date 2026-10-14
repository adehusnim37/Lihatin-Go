package announcements

import "time"

type InAppAnnouncementRead struct {
	ID             string    `json:"id" gorm:"primaryKey"`
	AnnouncementID string    `json:"announcement_id" gorm:"uniqueIndex:idx_announcements_in_app_reads_user_announcement,priority:2"`
	UserID         string    `json:"user_id" gorm:"uniqueIndex:idx_announcements_in_app_reads_user_announcement,priority:1"`
	IsRead         bool      `json:"is_read" gorm:"default:false"`
	CreatedAt      time.Time `json:"created_at" gorm:"autoCreateTime"`
}

type InAppAnnouncement struct {
	ID          string     `json:"id" gorm:"primaryKey,index:idx_announcements_in_app_status_start_at_ends_at,priority:1,sort:desc"`
	Title       string     `json:"title" gorm:"type:varchar(100);not null"`
	Body        string     `json:"body" gorm:"type:varchar(150);not null"`
	Href        string     `json:"href" gorm:"type:varchar(255);not null"`
	Status      string     `json:"status" gorm:"oneof:('draft','published','archived');default:'draft'"`
	StartAt     *time.Time `json:"start_at,omitempty"`
	EndsAt      *time.Time `json:"ends_at,omitempty"`
	CreatedBy   string     `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at" gorm:"autoCreateTime,index:idx_short_links_user_created_at,priority:3,sort:desc"`
	UpdatedAt   time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
}

// TableName specifies the table name for GORM
func (InAppAnnouncement) TableName() string {
	return "announcements_in_app"
}

func (InAppAnnouncementRead) TableName() string {
	return "announcements_in_app_reads"
}

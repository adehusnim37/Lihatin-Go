package user

import "time"

// InAppAnnouncementRead records that an account dismissed a product notice.
// It is independent of optional email subscriptions.
type InAppAnnouncementRead struct {
	UserID         string    `gorm:"primaryKey;size:191"`
	AnnouncementID string    `gorm:"primaryKey;size:100"`
	ReadAt         time.Time `gorm:"not null"`
}

func (InAppAnnouncementRead) TableName() string {
	return "in_app_announcement_reads"
}

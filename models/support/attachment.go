package support

import "time"

type SupportAttachment struct {
	ID          string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	TicketID    string    `json:"ticket_id" gorm:"type:varchar(36);not null;index"`
	MessageID   string    `json:"message_id" gorm:"type:varchar(36);not null;index"`
	FileName    string    `json:"file_name" gorm:"type:varchar(255);not null"`
	ContentType string    `json:"content_type" gorm:"type:varchar(100);not null"`
	SizeBytes   int64     `json:"size_bytes" gorm:"type:bigint;not null"`
	ObjectKey   string    `json:"-" gorm:"type:varchar(500);not null;default:''"`
	ObjectURL   string    `json:"object_url,omitempty" gorm:"type:text;not null;default:''"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (SupportAttachment) TableName() string {
	return "support_attachments"
}

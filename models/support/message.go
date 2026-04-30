package support

import "time"

type SupportMessageSenderType string

const (
	SupportMessageSenderPublic SupportMessageSenderType = "public"
	SupportMessageSenderUser   SupportMessageSenderType = "user"
	SupportMessageSenderAdmin  SupportMessageSenderType = "admin"
	SupportMessageSenderSystem SupportMessageSenderType = "system"
)

type SupportMessage struct {
	ID           string              `json:"id" gorm:"primaryKey;type:varchar(36)"`
	TicketID     string              `json:"ticket_id" gorm:"type:varchar(36);not null;index"`
	SenderType   string              `json:"sender_type" gorm:"type:varchar(20);not null;index"`
	SenderUserID *string             `json:"sender_user_id,omitempty" gorm:"type:varchar(36);index"`
	SenderEmail  *string             `json:"sender_email,omitempty" gorm:"type:varchar(255);index"`
	Body         string              `json:"body" gorm:"type:text;not null"`
	IsInternal   bool                `json:"is_internal" gorm:"type:boolean;default:false;index"`
	CreatedAt    time.Time           `json:"created_at"`
	UpdatedAt    time.Time           `json:"updated_at"`
	Attachments  []SupportAttachment `json:"attachments,omitempty" gorm:"foreignKey:MessageID;references:ID"`
}

func (SupportMessage) TableName() string {
	return "support_messages"
}

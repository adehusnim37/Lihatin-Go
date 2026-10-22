package support

import "time"

// CredentialReset is the single-use approval and notification outbox for a ticket.
// It deliberately contains no credential material or verification tokens.
type CredentialReset struct {
	TicketID             string     `json:"ticket_id" gorm:"primaryKey;type:varchar(36)"`
	TicketCode           string     `json:"ticket_code" gorm:"type:varchar(20);not null"`
	UserID               string     `json:"user_id" gorm:"type:varchar(36);not null;index"`
	ApprovedBy           string     `json:"approved_by" gorm:"type:varchar(36);not null"`
	Method               string     `json:"method" gorm:"type:varchar(20);not null"`
	Reason               string     `json:"reason" gorm:"type:text;not null"`
	IPAddress            string     `json:"ip_address" gorm:"type:varchar(45)"`
	CreatedAt            time.Time  `json:"created_at"`
	Email                string     `json:"-" gorm:"type:varchar(255);not null"`
	Username             string     `json:"-" gorm:"type:varchar(255)"`
	NotificationSentAt   *time.Time `json:"notification_sent_at,omitempty" gorm:"index"`
	NotificationAttempts int        `json:"-"`
	NextNotificationAt   time.Time  `json:"-" gorm:"index"`
}

func (CredentialReset) TableName() string { return "support_credential_resets" }

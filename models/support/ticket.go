package support

import "time"

type TicketStatus string

const (
	TicketStatusOpen       TicketStatus = "open"
	TicketStatusInProgress TicketStatus = "in_progress"
	TicketStatusResolved   TicketStatus = "resolved"
	TicketStatusClosed     TicketStatus = "closed"
)

type TicketCategory string

const (
	TicketCategoryAccountLocked      TicketCategory = "account_locked"
	TicketCategoryAccountDeactivated TicketCategory = "account_deactivated"
	TicketCategoryEmailVerification  TicketCategory = "email_verification"
	TicketCategoryLost2FA            TicketCategory = "lost_2fa"
	TicketCategoryBilling            TicketCategory = "billing"
	TicketCategoryBugReport          TicketCategory = "bug_report"
	TicketCategoryFeatureRequest     TicketCategory = "feature_request"
	TicketCategoryOther              TicketCategory = "other"
)

type SupportTicket struct {
	ID          string     `json:"id" gorm:"primaryKey;type:varchar(36)"`
	TicketCode  string     `json:"ticket_code" gorm:"uniqueIndex;type:varchar(20)"`
	Email       string     `json:"email" gorm:"type:varchar(255);not null;index"`
	Category    string     `json:"category" gorm:"type:varchar(50);not null;index"`
	Subject     string     `json:"subject" gorm:"type:varchar(255);not null"`
	Description string     `json:"description" gorm:"type:text;not null"`
	Status      string     `json:"status" gorm:"type:varchar(20);default:'open';index"`
	Priority    string     `json:"priority" gorm:"type:varchar(20);default:'normal'"`
	UserID      *string    `json:"user_id,omitempty" gorm:"type:varchar(36);index"`
	AdminNotes  *string    `json:"admin_notes,omitempty" gorm:"type:text"`
	ResolvedBy  *string    `json:"resolved_by,omitempty" gorm:"type:varchar(36)"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
	IPAddress   string     `json:"ip_address" gorm:"type:varchar(45)"`
	UserAgent   string     `json:"user_agent" gorm:"type:varchar(500)"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (SupportTicket) TableName() string {
	return "support_tickets"
}

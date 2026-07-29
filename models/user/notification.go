package user

import "time"

// NotificationPreference stores user-controlled, optional email subscriptions.
// Security and account-transaction emails are intentionally not configurable.
type NotificationPreference struct {
	UserID                string     `json:"user_id" gorm:"primaryKey;size:191"`
	WeeklySummaryEmail    bool       `json:"weekly_summary_email" gorm:"not null;default:false"`
	PromotionalEmail      bool       `json:"promotional_email" gorm:"not null;default:false"`
	WeeklySummaryOptInAt  *time.Time `json:"weekly_summary_opt_in_at,omitempty"`
	WeeklySummaryOptOutAt *time.Time `json:"weekly_summary_opt_out_at,omitempty"`
	PromotionalOptInAt    *time.Time `json:"promotional_opt_in_at,omitempty"`
	PromotionalOptOutAt   *time.Time `json:"promotional_opt_out_at,omitempty"`
	ConsentSource         string     `json:"consent_source,omitempty" gorm:"size:50"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

func (NotificationPreference) TableName() string {
	return "notification_preferences"
}

type WeeklySummaryDeliveryStatus string

const (
	WeeklySummaryDeliveryPending WeeklySummaryDeliveryStatus = "pending"
	WeeklySummaryDeliverySent    WeeklySummaryDeliveryStatus = "sent"
	WeeklySummaryDeliverySkipped WeeklySummaryDeliveryStatus = "skipped"
	WeeklySummaryDeliveryFailed  WeeklySummaryDeliveryStatus = "failed"
)

// WeeklySummaryDelivery makes weekly summary delivery idempotent per user and week.
type WeeklySummaryDelivery struct {
	ID             string                      `json:"id" gorm:"primaryKey;type:char(36)"`
	UserID         string                      `json:"user_id" gorm:"size:191;not null;uniqueIndex:idx_weekly_summary_user_period,priority:1"`
	PeriodStart    time.Time                   `json:"period_start" gorm:"not null;uniqueIndex:idx_weekly_summary_user_period,priority:2"`
	PeriodEnd      time.Time                   `json:"period_end" gorm:"not null"`
	Status         WeeklySummaryDeliveryStatus `json:"status" gorm:"size:20;not null;default:pending;index"`
	LinksCreated   int64                       `json:"links_created" gorm:"not null;default:0"`
	TotalClicks    int64                       `json:"total_clicks" gorm:"not null;default:0"`
	UniqueVisitors int64                       `json:"unique_visitors" gorm:"not null;default:0"`
	ErrorMessage   string                      `json:"error_message,omitempty" gorm:"type:text"`
	SentAt         *time.Time                  `json:"sent_at,omitempty"`
	CreatedAt      time.Time                   `json:"created_at"`
	UpdatedAt      time.Time                   `json:"updated_at"`
}

func (WeeklySummaryDelivery) TableName() string {
	return "weekly_summary_deliveries"
}

type PromotionalCampaignStatus string

const (
	PromotionalCampaignDraft     PromotionalCampaignStatus = "draft"
	PromotionalCampaignScheduled PromotionalCampaignStatus = "scheduled"
	PromotionalCampaignSending   PromotionalCampaignStatus = "sending"
	PromotionalCampaignCompleted PromotionalCampaignStatus = "completed"
	PromotionalCampaignCancelled PromotionalCampaignStatus = "cancelled"
	PromotionalCampaignFailed    PromotionalCampaignStatus = "failed"
)

// PromotionalCampaign is an administrator-created campaign sent only to
// users who explicitly opted in to promotional email.
type PromotionalCampaign struct {
	ID             string                     `json:"id" gorm:"primaryKey;type:char(36)"`
	Name           string                     `json:"name" gorm:"size:120;not null"`
	Subject        string                     `json:"subject" gorm:"size:180;not null"`
	Preheader      string                     `json:"preheader,omitempty" gorm:"size:255"`
	Body           string                     `json:"body" gorm:"type:text;not null"`
	ImageURL       string                     `json:"image_url,omitempty" gorm:"size:1000"`
	ImageAlt       string                     `json:"image_alt,omitempty" gorm:"size:180"`
	CTALabel       string                     `json:"cta_label,omitempty" gorm:"size:80"`
	CTAURL         string                     `json:"cta_url,omitempty" gorm:"size:1000"`
	Status         PromotionalCampaignStatus  `json:"status" gorm:"size:20;not null;default:draft;index"`
	ScheduledAt    *time.Time                 `json:"scheduled_at,omitempty" gorm:"index"`
	StartedAt      *time.Time                 `json:"started_at,omitempty"`
	CompletedAt    *time.Time                 `json:"completed_at,omitempty"`
	CreatedBy      string                     `json:"created_by" gorm:"size:191;not null;index"`
	RecipientCount int64                      `json:"recipient_count" gorm:"not null;default:0"`
	SentCount      int64                      `json:"sent_count" gorm:"not null;default:0"`
	FailedCount    int64                      `json:"failed_count" gorm:"not null;default:0"`
	CreatedAt      time.Time                  `json:"created_at"`
	UpdatedAt      time.Time                  `json:"updated_at"`
	Deliveries     []PromotionalEmailDelivery `json:"-" gorm:"foreignKey:CampaignID;constraint:OnDelete:CASCADE"`
}

func (PromotionalCampaign) TableName() string {
	return "promotional_campaigns"
}

type PromotionalDeliveryStatus string

const (
	PromotionalDeliveryPending PromotionalDeliveryStatus = "pending"
	PromotionalDeliverySending PromotionalDeliveryStatus = "sending"
	PromotionalDeliverySent    PromotionalDeliveryStatus = "sent"
	PromotionalDeliveryFailed  PromotionalDeliveryStatus = "failed"
	PromotionalDeliverySkipped PromotionalDeliveryStatus = "skipped"
)

// PromotionalEmailDelivery records one recipient outcome and prevents a
// retried campaign from sending twice to the same user.
type PromotionalEmailDelivery struct {
	ID           string                    `json:"id" gorm:"primaryKey;type:char(36)"`
	CampaignID   string                    `json:"campaign_id" gorm:"type:char(36);not null;uniqueIndex:idx_campaign_user,priority:1"`
	UserID       string                    `json:"user_id" gorm:"size:191;not null;uniqueIndex:idx_campaign_user,priority:2"`
	Email        string                    `json:"email" gorm:"size:255;not null"`
	Status       PromotionalDeliveryStatus `json:"status" gorm:"size:20;not null;default:pending;index"`
	ErrorMessage string                    `json:"error_message,omitempty" gorm:"type:text"`
	SentAt       *time.Time                `json:"sent_at,omitempty"`
	CreatedAt    time.Time                 `json:"created_at"`
	UpdatedAt    time.Time                 `json:"updated_at"`
}

func (PromotionalEmailDelivery) TableName() string {
	return "promotional_email_deliveries"
}

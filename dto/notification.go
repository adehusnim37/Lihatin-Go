package dto

import "time"

type NotificationPreferenceResponse struct {
	SecurityAlertsEmail bool       `json:"security_alerts_email"`
	WeeklySummaryEmail  bool       `json:"weekly_summary_email"`
	PromotionalEmail    bool       `json:"promotional_email"`
	WeeklySummaryOptIn  *time.Time `json:"weekly_summary_opt_in_at,omitempty"`
	PromotionalOptIn    *time.Time `json:"promotional_opt_in_at,omitempty"`
}

type UpdateNotificationPreferenceRequest struct {
	WeeklySummaryEmail *bool `json:"weekly_summary_email"`
	PromotionalEmail   *bool `json:"promotional_email"`
}

type CreatePromotionalCampaignRequest struct {
	Name      string `json:"name" binding:"required,max=120"`
	Subject   string `json:"subject" binding:"required,max=180"`
	Preheader string `json:"preheader" binding:"max=255"`
	Body      string `json:"body" binding:"required"`
	ImageURL  string `json:"image_url" binding:"omitempty,max=1000"`
	ImageAlt  string `json:"image_alt" binding:"max=180"`
	CTALabel  string `json:"cta_label" binding:"max=80"`
	CTAURL    string `json:"cta_url" binding:"omitempty,url,max=1000"`
}

type UpdatePromotionalCampaignRequest struct {
	Name      *string `json:"name" binding:"omitempty,max=120"`
	Subject   *string `json:"subject" binding:"omitempty,max=180"`
	Preheader *string `json:"preheader" binding:"omitempty,max=255"`
	Body      *string `json:"body"`
	ImageURL  *string `json:"image_url" binding:"omitempty,max=1000"`
	ImageAlt  *string `json:"image_alt" binding:"omitempty,max=180"`
	CTALabel  *string `json:"cta_label" binding:"omitempty,max=80"`
	CTAURL    *string `json:"cta_url" binding:"omitempty,url,max=1000"`
}

type SchedulePromotionalCampaignRequest struct {
	ScheduledAt *time.Time `json:"scheduled_at"`
}

type PromotionalDeliveryResponse struct {
	ID           string     `json:"id"`
	UserID       string     `json:"user_id"`
	Email        string     `json:"email"`
	Status       string     `json:"status"`
	ErrorMessage string     `json:"error_message,omitempty"`
	SentAt       *time.Time `json:"sent_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type PromotionalDeliveryListResponse struct {
	Deliveries []PromotionalDeliveryResponse `json:"deliveries"`
	Page       int                           `json:"page"`
	Limit      int                           `json:"limit"`
	Total      int64                         `json:"total"`
	TotalPages int                           `json:"total_pages"`
	Status     string                        `json:"status,omitempty"`
}

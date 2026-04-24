package dto

import "time"

type CreateSupportTicketRequest struct {
	Email        string `json:"email" binding:"required,email,max=255"`
	Category     string `json:"category" binding:"required,oneof=account_locked account_deactivated email_verification lost_2fa billing bug_report feature_request other"`
	Subject      string `json:"subject" binding:"required,min=5,max=255"`
	Description  string `json:"description" binding:"required,min=10,max=5000"`
	CaptchaToken string `json:"captcha_token" binding:"required,min=10,max=4096"`
}

type CreateSupportTicketResponse struct {
	TicketCode string `json:"ticket_code"`
}

type TrackSupportTicketQuery struct {
	Ticket string `form:"ticket" binding:"required,min=6,max=20"`
	Email  string `form:"email" binding:"required,email,max=255"`
}

type TrackSupportTicketResponse struct {
	TicketCode string     `json:"ticket_code"`
	Category   string     `json:"category"`
	Subject    string     `json:"subject"`
	Status     string     `json:"status"`
	CreatedAt  time.Time  `json:"created_at"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
}

type AdminListSupportTicketsResponse struct {
	Items      []AdminSupportTicketItem `json:"items"`
	Total      int64                    `json:"total"`
	Page       int                      `json:"page"`
	Limit      int                      `json:"limit"`
	TotalPages int                      `json:"total_pages"`
}

type AdminSupportTicketItem struct {
	ID         string     `json:"id"`
	TicketCode string     `json:"ticket_code"`
	Email      string     `json:"email"`
	Category   string     `json:"category"`
	Subject    string     `json:"subject"`
	Status     string     `json:"status"`
	Priority   string     `json:"priority"`
	UserID     *string    `json:"user_id,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
}

type AdminSupportTicketDetailResponse struct {
	ID          string     `json:"id"`
	TicketCode  string     `json:"ticket_code"`
	Email       string     `json:"email"`
	Category    string     `json:"category"`
	Subject     string     `json:"subject"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	Priority    string     `json:"priority"`
	UserID      *string    `json:"user_id,omitempty"`
	AdminNotes  *string    `json:"admin_notes,omitempty"`
	ResolvedBy  *string    `json:"resolved_by,omitempty"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type AdminUpdateSupportTicketRequest struct {
	Status     string  `json:"status" binding:"required,oneof=open in_progress resolved closed"`
	Priority   string  `json:"priority,omitempty" binding:"omitempty,oneof=low normal high urgent"`
	AdminNotes *string `json:"admin_notes,omitempty" binding:"omitempty,max=5000"`
	Action     string  `json:"action,omitempty" binding:"omitempty,oneof=unlock_user activate_user resend_verification manual_response"`
}

type AdminUpdateSupportTicketResponse struct {
	TicketCode    string     `json:"ticket_code"`
	Status        string     `json:"status"`
	AdminNotes    *string    `json:"admin_notes,omitempty"`
	ResolvedAt    *time.Time `json:"resolved_at,omitempty"`
	ActionApplied string     `json:"action_applied,omitempty"`
}

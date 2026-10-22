package dto

import (
	supportmodel "github.com/adehusnim37/lihatin-go/models/support"
	"time"
)

type CreateSupportTicketRequest struct {
	Email        string `json:"email" binding:"required,email,max=255"`
	Category     string `json:"category" binding:"required,oneof=account_locked account_deactivated email_verification lost_2fa billing bug_report feature_request suspicious_link other"`
	Subject      string `json:"subject" label:"Subjek" binding:"required,min=5,max=255,meaningful_text"`
	Description  string `json:"description" label:"Deskripsi" binding:"required,min=10,max=5000,meaningful_text"`
	CaptchaToken string `json:"captcha_token" binding:"required,min=10,max=4096"`
}

type CreateSupportTicketResponse struct {
	TicketCode string `json:"ticket_code"`
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
	UpdatedAt  time.Time  `json:"updated_at"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
}

type AdminSupportTicketDetailResponse struct {
	RequesterVerifiedAt *time.Time                    `json:"requester_verified_at,omitempty"`
	CredentialReset     *supportmodel.CredentialReset `json:"credential_reset,omitempty"`
	ID                  string                        `json:"id"`
	TicketCode          string                        `json:"ticket_code"`
	Email               string                        `json:"email"`
	Category            string                        `json:"category"`
	Subject             string                        `json:"subject"`
	Description         string                        `json:"description"`
	Status              string                        `json:"status"`
	Priority            string                        `json:"priority"`
	UserID              *string                       `json:"user_id,omitempty"`
	AdminNotes          *string                       `json:"admin_notes,omitempty"`
	ResolvedBy          *string                       `json:"resolved_by,omitempty"`
	ResolvedAt          *time.Time                    `json:"resolved_at,omitempty"`
	CreatedAt           time.Time                     `json:"created_at"`
	UpdatedAt           time.Time                     `json:"updated_at"`
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

type SupportRequestAccessOTPRequest struct {
	Ticket       string `json:"ticket" binding:"required,min=6,max=20"`
	Email        string `json:"email" binding:"required,email,max=255"`
	CaptchaToken string `json:"captcha_token" binding:"required,min=10,max=4096"`
}

type SupportVerifyAccessCodeRequest struct {
	Ticket string `json:"ticket" binding:"required,min=6,max=20"`
	Email  string `json:"email" binding:"required,email,max=255"`
	Code   string `json:"code" binding:"required,min=16,max=255,no_space"`
}

type SupportVerifyAccessOTPRequest struct {
	ChallengeToken string `json:"challenge_token" binding:"required,min=10,max=255,no_space"`
	OTPCode        string `json:"otp_code" binding:"required,len=6,numeric"`
}

type SupportResendAccessOTPRequest struct {
	ChallengeToken string `json:"challenge_token" binding:"required,min=10,max=255,no_space"`
	CaptchaToken   string `json:"captcha_token" binding:"required,min=10,max=4096"`
}

type SupportOTPChallengeResponse struct {
	ChallengeToken  string `json:"challenge_token"`
	CooldownSeconds int    `json:"cooldown_seconds"`
}

type SupportAccessResponse struct {
	ExpiresInSeconds int                        `json:"expires_in_seconds"`
	Ticket           TrackSupportTicketResponse `json:"ticket"`
}

type SupportAttachmentResponse struct {
	ID          string    `json:"id"`
	FileName    string    `json:"file_name"`
	ContentType string    `json:"content_type"`
	SizeBytes   int64     `json:"size_bytes"`
	CreatedAt   time.Time `json:"created_at"`
}

type SupportMessageResponse struct {
	ID           string                      `json:"id"`
	TicketID     string                      `json:"ticket_id"`
	SenderType   string                      `json:"sender_type"`
	SenderUserID *string                     `json:"sender_user_id,omitempty"`
	SenderEmail  *string                     `json:"sender_email,omitempty"`
	Body         string                      `json:"body"`
	IsInternal   bool                        `json:"is_internal"`
	CreatedAt    time.Time                   `json:"created_at"`
	UpdatedAt    time.Time                   `json:"updated_at"`
	Attachments  []SupportAttachmentResponse `json:"attachments,omitempty"`
}

type SupportConversationResponse struct {
	TicketCode string                   `json:"ticket_code"`
	TicketID   string                   `json:"ticket_id"`
	Category   string                   `json:"category"`
	Subject    string                   `json:"subject"`
	Status     string                   `json:"status"`
	CreatedAt  time.Time                `json:"created_at"`
	UpdatedAt  time.Time                `json:"updated_at"`
	Messages   []SupportMessageResponse `json:"messages"`
}

type UserListSupportTicketsResponse struct {
	Items      []AdminSupportTicketItem `json:"items"`
	Total      int64                    `json:"total"`
	Page       int                      `json:"page"`
	Limit      int                      `json:"limit"`
	TotalPages int                      `json:"total_pages"`
}

// AdminResetSupportCredentialsRequest is deliberately separate from ticket updates.
type AdminResetSupportCredentialsRequest struct {
	Method            string `json:"method" binding:"required,oneof=totp passkeys all"`
	Approved          bool   `json:"approved" binding:"required"`
	Reason            string `json:"reason" binding:"required,min=20,max=1000,meaningful_text"`
	TicketCode        string `json:"ticket_code" binding:"required,max=20"`
	VerificationToken string `json:"verification_token" binding:"required,len=64"`
}

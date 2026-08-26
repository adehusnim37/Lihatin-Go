package dto

import (
	"time"

	"github.com/adehusnim37/lihatin-go/models/user"
)

// Auth-related Data Transfer Objects (DTOs)

// LoginRequest represents the login request payload
type LoginRequest struct {
	EmailOrUsername string `json:"email_or_username" label:"Email atau Username" binding:"required,min=3,max=100"`
	Password        string `json:"password" label:"Kata Sandi" binding:"required,min=8,max=50,pwdcomplex"`
}

// GoogleOAuthStartRequest starts OAuth flow with explicit intent.
type GoogleOAuthStartRequest struct {
	Intent string `json:"intent,omitempty" binding:"omitempty,oneof=login signup"`
}

// GoogleOAuthStartResponse returns authorization URL and state token.
type GoogleOAuthStartResponse struct {
	AuthorizationURL string `json:"authorization_url"`
	State            string `json:"state"`
}

// GoogleOAuthCallbackRequest exchanges Google code and state after redirect.
type GoogleOAuthCallbackRequest struct {
	Code  string `json:"code" binding:"required,min=10,max=4096,no_space"`
	State string `json:"state" binding:"required,min=10,max=255,no_space"`
}

// SignupStartRequest represents the initial email-only signup payload.
type SignupStartRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// SignupStartResponse returns either OTP challenge or existing signup token.
type SignupStartResponse struct {
	ChallengeToken            string `json:"challenge_token,omitempty"`
	CooldownSeconds           int    `json:"cooldown_seconds,omitempty"`
	SignupToken               string `json:"signup_token,omitempty"`
	RequiresProfileCompletion bool   `json:"requires_profile_completion,omitempty"`
}

// SignupVerifyOTPRequest validates OTP from signup flow.
type SignupVerifyOTPRequest struct {
	ChallengeToken string `json:"challenge_token" binding:"required,min=10,max=255,no_space"`
	OTPCode        string `json:"otp_code" binding:"required,len=6,numeric"`
}

// SignupVerifyOTPResponse returns one-time token for profile completion.
type SignupVerifyOTPResponse struct {
	SignupToken string `json:"signup_token"`
}

// SignupCompletionStatusRequest validates access to the profile completion page.
type SignupCompletionStatusRequest struct {
	SignupToken string `form:"signup_token" binding:"required,len=48,hexadecimal"`
}

type SignupCompletionStatusResponse struct {
	Valid bool `json:"valid"`
}

// SignupResendOTPRequest triggers OTP resend for pending signup challenge.
type SignupResendOTPRequest struct {
	ChallengeToken string `json:"challenge_token" binding:"required,min=10,max=255,no_space"`
}

// SignupCompleteRequest finalizes signup after OTP verification.
type SignupCompleteRequest struct {
	SignupToken              string `json:"signup_token" binding:"required,min=10,max=255,no_space"`
	FirstName                string `json:"first_name" binding:"required,min=3,max=50"`
	LastName                 string `json:"last_name" binding:"required,min=3,max=50"`
	Username                 string `json:"username" binding:"required,min=3,max=30,alphanum"`
	Password                 string `json:"password" binding:"required,min=8,max=50,pwdcomplex"`
	SecretCode               string `json:"secret_code,omitempty" binding:"omitempty,secret_code"`
	OptInPromotionalEmails   bool   `json:"opt_in_promotional_emails"`
	OptInWeeklySummaryEmails bool   `json:"opt_in_weekly_summary_emails"`
	ConsentSource            string `json:"consent_source" binding:"required,oneof=signup_page"`
}

// ResendVerificationRequest represents request payload to resend email verification link
type ResendVerificationRequest struct {
	Identifier string `json:"identifier" label:"Identifier (Base64URL)" binding:"required,min=4,max=512"`
}

// LoginResponse represents the successful login response
type LoginResponse struct {
	// Token TokenResponse    `json:"token"`
	User UserProfile      `json:"user"`
	Auth UserAuthResponse `json:"auth"`
}

// PendingTOTPResponse represents login response when TOTP verification is required
// NO tokens are issued - user MUST verify TOTP first
type PendingTOTPResponse struct {
	RequiresTOTP     bool        `json:"requires_totp"`
	PendingAuthToken string      `json:"pending_auth_token"` // Temporary token for TOTP verification
	User             UserProfile `json:"user"`               // Basic user info for display
}

// PendingEmailOTPResponse represents login response when email OTP verification is required.
type PendingEmailOTPResponse struct {
	RequiresEmailOTP bool        `json:"requires_email_otp"`
	ChallengeToken   string      `json:"challenge_token"`
	CooldownSeconds  int         `json:"cooldown_seconds,omitempty"`
	Email            string      `json:"email"`
	User             UserProfile `json:"user"`
}

// VerifyTOTPLoginRequest represents the request to verify TOTP during login
type VerifyTOTPLoginRequest struct {
	PendingAuthToken string `json:"pending_auth_token" binding:"required" label:"Token Autentikasi"`
	TOTPCode         string `json:"totp_code" binding:"required,len=6,numeric" label:"Kode TOTP"`
}

// VerifyEmailOTPLoginRequest validates login email OTP challenge.
type VerifyEmailOTPLoginRequest struct {
	ChallengeToken string `json:"challenge_token" binding:"required,min=10,max=255,no_space"`
	OTPCode        string `json:"otp_code" binding:"required,len=6,numeric"`
}

// ResendEmailOTPLoginRequest triggers OTP resend during login challenge.
type ResendEmailOTPLoginRequest struct {
	ChallengeToken string `json:"challenge_token" binding:"required,min=10,max=255,no_space"`
}

// ResendOTPResponse contains cooldown payload for resend operations.
type ResendOTPResponse struct {
	CooldownSeconds          int `json:"cooldown_seconds,omitempty"`
	CooldownRemainingSeconds int `json:"cooldown_remaining_seconds,omitempty"`
}

type ProfileAuthResponse struct {
	User UserProfile               `json:"user"`
	Auth CompletedUserAuthResponse `json:"auth"`
}

// TokenResponse represents access and refresh tokens
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// Change Email Request represents the request payload to change email
type ChangeEmailRequest struct {
	NewEmail string `json:"new_email" label:"Email Baru" binding:"required,email"`
}

// UserProfile represents user information returned after login (without sensitive data)
type UserProfile struct {
	ID            string                 `json:"id"`
	Username      string                 `json:"username"`
	FirstName     string                 `json:"first_name"`
	LastName      string                 `json:"last_name"`
	Email         string                 `json:"email"`
	Avatar        string                 `json:"avatar"`
	Role          string                 `json:"role"`
	PremiumAccess *PremiumAccessResponse `json:"premium_access"`
	CreatedAt     string                 `json:"created_at"`
}

type PremiumAccessResponse struct {
	Status          string     `json:"status"`
	Tier            string     `json:"tier"`
	Source          string     `json:"source"`
	GrantedAt       time.Time  `json:"granted_at"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
	RevokeType      string     `json:"revoke_type,omitempty"`
	StatusChangedAt *time.Time `json:"status_changed_at,omitempty"`
	StatusChangedBy *string    `json:"status_changed_by,omitempty"`
	StatusReason    string     `json:"status_reason,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func NewPremiumAccessResponse(access *user.PremiumAccess) *PremiumAccessResponse {
	if access == nil {
		return nil
	}
	return &PremiumAccessResponse{
		Status:          string(access.Status),
		Tier:            access.Tier,
		Source:          access.Source,
		GrantedAt:       access.GrantedAt,
		ExpiresAt:       access.ExpiresAt,
		RevokeType:      string(access.RevokeType),
		StatusChangedAt: access.StatusChangedAt,
		StatusChangedBy: access.StatusChangedBy,
		StatusReason:    access.StatusReason,
		CreatedAt:       access.CreatedAt,
		UpdatedAt:       access.UpdatedAt,
	}
}

// RegisterRequest represents the user registration request payload
type RegisterRequest struct {
	FirstName                string `json:"first_name" binding:"required,min=3,max=50"`
	LastName                 string `json:"last_name" binding:"required,min=3,max=50"`
	Username                 string `json:"username" binding:"required,min=3,max=30,alphanum"`
	Email                    string `json:"email" binding:"required,email"`
	Password                 string `json:"password" binding:"required,min=8,max=50,pwdcomplex"`
	SecretCode               string `json:"secret_code,omitempty" binding:"omitempty,secret_code"`
	OptInPromotionalEmails   bool   `json:"opt_in_promotional_emails"`
	OptInWeeklySummaryEmails bool   `json:"opt_in_weekly_summary_emails"`
	ConsentSource            string `json:"consent_source" binding:"required,oneof=signup_page"`
}

// UpdateProfileRequest represents the user profile update request payload
type UpdateProfileRequest struct {
	FirstName *string `json:"first_name" binding:"omitempty,min=3,max=50"`
	LastName  *string `json:"last_name" binding:"omitempty,min=3,max=50"`
	Avatar    *string `json:"avatar" binding:"omitempty,url"`
}

// ResetPasswordRequest represents the request to reset password using a token
type ResetPasswordRequest struct {
	Token           string `json:"token" binding:"required"`
	NewPassword     string `json:"new_password" label:"Kata Sandi Baru" binding:"required,min=8,max=50,pwdcomplex"`
	ConfirmPassword string `json:"confirm_password" label:"Konfirmasi Kata Sandi Baru" binding:"required,eqfield=NewPassword"`
}

// DeleteAccountRequest represents the request to delete a user account
type DeleteAccountRequest struct {
	ID string `json:"id" binding:"required,uuid" label:"ID Pengguna" uri:"id"`
}

// User Id Generic Request represents a request that requires user ID in the URI
type UserIDGenericRequest struct {
	ID string `json:"id" binding:"required,uuid" label:"ID Pengguna" uri:"id"`
}

// IDGenericRequest represents a request that requires an ID in the URI
type IDGenericRequest struct {
	ID string `json:"id" binding:"required,uuid" label:"ID" uri:"id"`
}

// Generic Request password field
type PasswordGenericRequest struct {
	Password string `json:"password" binding:"required,min=8,max=50,pwdcomplex" label:"Kata Sandi"`
}

// UserAuthResponse represents user authentication details (without sensitive data)
type UserAuthResponse struct {
	ID              string              `json:"id"`
	UserID          string              `json:"user_id"`
	IsEmailVerified bool                `json:"is_email_verified"`
	TOTPEnabled     bool                `json:"totp_enabled"`
	AccountStatus   string              `json:"account_status"`
	LastLoginAt     string              `json:"last_login_at"`
	PreviousLogin   *LoginEventResponse `json:"previous_login,omitempty"`
}

type CompletedUserAuthResponse struct {
	ID                  string                  `json:"id"`
	UserID              string                  `json:"user_id"`
	IsEmailVerified     bool                    `json:"is_email_verified"`
	DeviceID            *string                 `json:"device_id,omitempty"`
	LastIP              *string                 `json:"last_ip,omitempty"`
	LastLoginAt         *time.Time              `json:"last_login_at,omitempty"`
	LastLogoutAt        *time.Time              `json:"last_logout_at,omitempty"`
	FailedLoginAttempts int                     `json:"failed_login_attempts"`
	LoginBlockedUntil   *time.Time              `json:"login_blocked_until,omitempty"`
	PasswordChangedAt   *time.Time              `json:"password_changed_at,omitempty"`
	AccountStatus       string                  `json:"account_status"`
	TOTPEnabled         bool                    `json:"totp_enabled"`
	CreatedAt           time.Time               `json:"created_at"`
	UpdatedAt           time.Time               `json:"updated_at"`
	PreviousLogin       *LoginEventResponse     `json:"previous_login,omitempty"`
	CurrentSession      *CurrentSessionResponse `json:"current_session,omitempty"`
}

// LoginEventResponse describes a completed login without exposing its session ID.
type LoginEventResponse struct {
	Method          string    `json:"method"`
	DeviceID        string    `json:"device_id,omitempty"`
	IPAddress       string    `json:"ip_address,omitempty"`
	UserAgent       string    `json:"user_agent,omitempty"`
	AuthenticatedAt time.Time `json:"authenticated_at"`
}

func NewLoginEventResponse(event *user.LoginEvent) *LoginEventResponse {
	if event == nil {
		return nil
	}
	return &LoginEventResponse{
		Method:          string(event.Method),
		DeviceID:        event.DeviceID,
		IPAddress:       event.IPAddress,
		UserAgent:       event.UserAgent,
		AuthenticatedAt: event.AuthenticatedAt,
	}
}

// CurrentSessionResponse is sourced from the authenticated Redis session,
// rather than the account-level "latest login" snapshot.
type CurrentSessionResponse struct {
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
	DeviceID  string    `json:"device_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	LastSeen  time.Time `json:"last_seen"`
	ExpiresAt time.Time `json:"expires_at"`
}

// ActiveSessionResponse is a single row in the "active devices / sessions" list.
type ActiveSessionResponse struct {
	SessionID string    `json:"session_id"`
	IsCurrent bool      `json:"is_current"`
	DeviceID  string    `json:"device_id,omitempty"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
	CreatedAt time.Time `json:"created_at"`
	LastSeen  time.Time `json:"last_seen"`
	ExpiresAt time.Time `json:"expires_at"`
}

// SessionsListResponse lists the signed-in user's active sessions.
type SessionsListResponse struct {
	Total    int                     `json:"total"`
	Sessions []ActiveSessionResponse `json:"sessions"`
}

// SessionActionRequest specifies which session to act on.
type SessionActionRequest struct {
	SessionID string `json:"session_id" binding:"required"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" label:"Kata Sandi Saat Ini" binding:"required,min=8,max=50,pwdcomplex"`
	NewPassword     string `json:"new_password" label:"Kata Sandi Baru" binding:"required,min=8,max=50,pwdcomplex,nefield=CurrentPassword"`
	ConfirmPassword string `json:"confirm_password" label:"Konfirmasi Kata Sandi Baru" binding:"required,eqfield=NewPassword"`
}

type ForgotPasswordRequest struct {
	Email    *string `json:"email" label:"Email" binding:"omitempty,email"`
	Username *string `json:"username" label:"Username" binding:"omitempty,min=3,max=30,alphanum"`
}

// AdminLockUserRequest represents the request to lock a user account
type AdminLockUserRequest struct {
	Reason string `json:"reason" validate:"required,min=10,max=500"`
}

// AdminUnlockUserRequest represents the request to unlock a user account
type AdminUnlockUserRequest struct {
	Reason string `json:"reason,omitempty" validate:"max=500"`
}

// AdminRevokePremiumRequest represents revoke premium payload.
type AdminRevokePremiumRequest struct {
	Reason     string `json:"reason" binding:"required,min=10,max=500"`
	RevokeType string `json:"revoke_type" binding:"required,oneof=temporary permanent"`
}

// AdminReactivatePremiumRequest represents reactivate premium payload.
type AdminReactivatePremiumRequest struct {
	Reason            string `json:"reason" binding:"required,min=5,max=500"`
	OverridePermanent bool   `json:"override_permanent,omitempty"`
}

// AdminUpdateUserRequest represents admin payload to update user core fields.
type AdminUpdateUserRequest struct {
	FirstName *string `json:"first_name" binding:"omitempty,min=3,max=50"`
	LastName  *string `json:"last_name" binding:"omitempty,min=3,max=50"`
	Username  *string `json:"username" binding:"omitempty,min=3,max=30,alphanum"`
	Email     *string `json:"email" binding:"omitempty,email"`
	Role      *string `json:"role" binding:"omitempty"`
}

// AdminUserResponse represents the response format for admin user data
type AdminUserResponse struct {
	ID                     string                 `json:"id"`
	Username               string                 `json:"username"`
	FirstName              string                 `json:"first_name"`
	LastName               string                 `json:"last_name"`
	Email                  string                 `json:"email"`
	CreatedAt              time.Time              `json:"created_at"`
	UpdatedAt              time.Time              `json:"updated_at"`
	AccountStatus          string                 `json:"account_status"`
	AccountStatusChangedAt *time.Time             `json:"account_status_changed_at,omitempty"`
	AccountStatusReason    string                 `json:"account_status_reason,omitempty"`
	Role                   string                 `json:"role"`
	PremiumAccess          *PremiumAccessResponse `json:"premium_access"`
}

// AdminUserAuthDetailResponse represents detailed user_auth section for admin user detail.
type AdminUserAuthDetailResponse struct {
	ID                  string     `json:"id"`
	UserID              string     `json:"user_id"`
	IsEmailVerified     bool       `json:"is_email_verified"`
	PasswordChangedAt   *time.Time `json:"password_changed_at,omitempty"`
	LastEmailSendAt     *time.Time `json:"last_email_send_at,omitempty"`
	DeviceID            *string    `json:"device_id,omitempty"`
	LastIP              *string    `json:"last_ip,omitempty"`
	LastLoginAt         *time.Time `json:"last_login_at,omitempty"`
	LastLogoutAt        *time.Time `json:"last_logout_at,omitempty"`
	FailedLoginAttempts int        `json:"failed_login_attempts"`
	LoginBlockedUntil   *time.Time `json:"login_blocked_until,omitempty"`
	AccountStatus       string     `json:"account_status"`
	TOTPEnabled         bool       `json:"totp_enabled"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
	DeletedAt           *time.Time `json:"deleted_at,omitempty"`
}

// AdminAuthMethodDetailResponse represents safe auth method fields for admin user detail.
type AdminAuthMethodDetailResponse struct {
	ID             string     `json:"id"`
	UserAuthID     string     `json:"user_auth_id"`
	Type           string     `json:"type"`
	IsEnabled      bool       `json:"is_enabled"`
	IsVerified     bool       `json:"is_verified"`
	VerifiedAt     *time.Time `json:"verified_at,omitempty"`
	LastUsedAt     *time.Time `json:"last_used_at,omitempty"`
	FriendlyName   string     `json:"friendly_name,omitempty"`
	ProviderUserID string     `json:"provider_user_id,omitempty"`
	DisabledAt     *time.Time `json:"disabled_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty"`
}

// AdminUserDetailStatsResponse represents related counters for admin user detail.
type AdminUserDetailStatsResponse struct {
	APIKeysTotal             int64 `json:"api_keys_total"`
	APIKeysActive            int64 `json:"api_keys_active"`
	HistoryEventsTotal       int64 `json:"history_events_total"`
	PremiumKeyUsageTotal     int64 `json:"premium_key_usage_total"`
	PremiumAccessEventsTotal int64 `json:"premium_access_events_total"`
	LoginAttempts24h         int64 `json:"login_attempts_24h"`
	LoginAttempts7d          int64 `json:"login_attempts_7d"`
}

// AdminUserRecentHistoryResponse represents compact history record for admin user detail.
type AdminUserRecentHistoryResponse struct {
	ID         uint      `json:"id"`
	ActionType string    `json:"action_type"`
	Reason     string    `json:"reason,omitempty"`
	ChangedBy  *string   `json:"changed_by,omitempty"`
	IPAddress  *string   `json:"ip_address,omitempty"`
	UserAgent  *string   `json:"user_agent,omitempty"`
	ChangedAt  time.Time `json:"changed_at"`
}

// AdminUserRecentLoginAttemptResponse represents compact login attempt record for admin user detail.
type AdminUserRecentLoginAttemptResponse struct {
	ID              string    `json:"id"`
	EmailOrUsername string    `json:"email_or_username"`
	IPAddress       string    `json:"ip_address"`
	UserAgent       string    `json:"user_agent"`
	Success         bool      `json:"success"`
	FailReason      string    `json:"fail_reason,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

// AdminUserDetailResponse represents very detailed admin user payload for user detail endpoint.
type AdminUserDetailResponse struct {
	ID                     string                                `json:"id"`
	Username               string                                `json:"username"`
	FirstName              string                                `json:"first_name"`
	LastName               string                                `json:"last_name"`
	Email                  string                                `json:"email"`
	Avatar                 string                                `json:"avatar,omitempty"`
	CreatedAt              time.Time                             `json:"created_at"`
	UpdatedAt              time.Time                             `json:"updated_at"`
	DeletedAt              *time.Time                            `json:"deleted_at,omitempty"`
	UsernameChanged        bool                                  `json:"username_changed"`
	AccountStatus          string                                `json:"account_status"`
	AccountStatusChangedAt *time.Time                            `json:"account_status_changed_at,omitempty"`
	AccountStatusReason    string                                `json:"account_status_reason,omitempty"`
	Role                   string                                `json:"role"`
	PremiumAccess          *PremiumAccessResponse                `json:"premium_access"`
	UserAuth               *AdminUserAuthDetailResponse          `json:"user_auth,omitempty"`
	AuthMethods            []AdminAuthMethodDetailResponse       `json:"auth_methods,omitempty"`
	Stats                  AdminUserDetailStatsResponse          `json:"stats"`
	RecentHistory          []AdminUserRecentHistoryResponse      `json:"recent_history,omitempty"`
	RecentLoginAttempts    []AdminUserRecentLoginAttemptResponse `json:"recent_login_attempts,omitempty"`
}

type AdminPremiumAccessMutationResponse struct {
	UserID        string                `json:"user_id"`
	PremiumAccess PremiumAccessResponse `json:"premium_access"`
}

type AdminPremiumAccessEventResponse struct {
	ID         uint      `json:"id"`
	UserID     string    `json:"user_id"`
	Action     string    `json:"action"`
	OldStatus  string    `json:"old_status"`
	NewStatus  string    `json:"new_status"`
	RevokeType string    `json:"revoke_type,omitempty"`
	Reason     string    `json:"reason"`
	ActorID    *string   `json:"actor_id,omitempty"`
	ActorRole  string    `json:"actor_role"`
	CreatedAt  time.Time `json:"created_at"`
}

type AdminPremiumAccessEventsListResponse struct {
	UserID string                            `json:"user_id"`
	Total  int                               `json:"total"`
	Items  []AdminPremiumAccessEventResponse `json:"items"`
}

// PaginatedUsersResponse represents paginated user results
type PaginatedUsersResponse struct {
	Users               []AdminUserResponse `json:"users"`
	TotalCount          int64               `json:"total_count"`
	Page                int                 `json:"page"`
	Limit               int                 `json:"limit"`
	TotalPages          int                 `json:"total_pages"`
	Sort                string              `json:"sort"`
	OrderBy             string              `json:"order_by"`
	Search              string              `json:"search,omitempty"`
	Role                string              `json:"role,omitempty"`
	PremiumAccessStatus string              `json:"premium_access_status,omitempty"`
	LockStatus          string              `json:"lock_status,omitempty"`
}

// PaginatedUserEmailsResponse represents paginated non-premium recipient results.
type PaginatedUserEmailsResponse struct {
	Users      []AdminUserEmailResponse `json:"users"`
	TotalCount int64                    `json:"total_count"`
	Page       int                      `json:"page"`
	Limit      int                      `json:"limit"`
	TotalPages int                      `json:"total_pages"`
}

// AdminUserEmailResponse is the compact user shape used by recipient pickers.
type AdminUserEmailResponse struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

// AdminDisposableEmailPolicyResponse represents disposable email policy state.
type AdminDisposableEmailPolicyResponse struct {
	Enabled               bool       `json:"enabled"`
	EffectiveInCurrentEnv bool       `json:"effective_in_current_env"`
	LastUpdatedBy         *string    `json:"last_updated_by,omitempty"`
	LastUpdatedAt         *time.Time `json:"last_updated_at,omitempty"`
}

// UpdateAdminDisposableEmailPolicyRequest updates disposable email policy state.
type UpdateAdminDisposableEmailPolicyRequest struct {
	Enabled *bool `json:"enabled" binding:"required"`
}

// VerifyEmailResponse represents the response after verifying an email
type VerifyEmailResponse struct {
	Email    string                       `json:"email"`
	Username string                       `json:"username"`
	Source   user.EmailVerificationSource `json:"source,omitempty"`
	OldEmail string                       `json:"old_email,omitempty"`
	Token    string                       `json:"token,omitempty"`
}

// ForgotPasswordToken represents the token extracted from URL parameters for password reset
type ForgotPasswordToken struct {
	Token string `json:"token" binding:"required,min=10,max=255,no_space" form:"token" label:"Token Reset"`
}

// LoginAttemptsStatsDTO represents the login attempts statistics data
type LoginAttemptsStatsRequest struct {
	EmailOrUsername string `json:"email_or_username" label:"Email atau Username" binding:"required,min=3,max=100" uri:"email_or_username"`
	Days            int    `json:"days" label:"Jumlah Hari" binding:"required,min=1,max=365" uri:"days"`
}

// ChangeUsernameRequest represents the request to change username
type ChangeUsernameRequest struct {
	NewUsername string `json:"new_username" label:"Username Baru" binding:"required,min=3,max=30,alphanum"`
}

// CheckUsernameRequest represents the request to check username availability
type CheckUsernameRequest struct {
	Username string `json:"username" binding:"required,min=3,max=30,alphanum"`
}

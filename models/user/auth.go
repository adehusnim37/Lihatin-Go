package user

import "time"

type UserAuth struct {
	ID     string `json:"id" gorm:"primaryKey"`                                                              // Primary Key
	UserID string `json:"user_id" gorm:"size:191;not null;uniqueIndex"`                                      // Foreign Key to User.ID, same size as User.ID
	User   *User  `json:"user,omitempty" gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"` // Optional: for eager loading user details

	// Password-based authentication
	PasswordHash string `json:"-" gorm:"size:255"` // Store the hashed password, never expose in JSON responses

	// Email verification
	IsEmailVerified                 bool       `json:"is_email_verified" gorm:"default:false"`
	EmailVerificationToken          string     `json:"-" gorm:"size:255"` // Token sent to user's email
	EmailVerificationTokenExpiresAt *time.Time `json:"-"`                 // Expiry for the verification token

	// Password reset
	PasswordResetToken          string     `json:"-" gorm:"size:255"` // Token for password reset
	PasswordResetTokenExpiresAt *time.Time `json:"-"`                 // Expiry for the reset token

	// Account status & security
	LastLoginAt         *time.Time `json:"last_login_at,omitempty"`
	FailedLoginAttempts int        `json:"-" gorm:"default:0"`                   // Counter for failed login attempts
	LockoutUntil        *time.Time `json:"-"`                                    // Timestamp until which the account is locked
	IsActive            bool       `json:"is_active" gorm:"default:true"`        // General account active status
	IsTOTPEnabled       bool       `json:"is_totp_enabled" gorm:"default:false"` // Whether TOTP is enabled for this user

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" gorm:"index"` // For soft deletes

	// Relationships
	AuthMethods []AuthMethod `json:"auth_methods,omitempty" gorm:"foreignKey:UserAuthID"`
}

// TableName specifies the table name for GORM
func (UserAuth) TableName() string {
	return "user_auth"
}

type AuthMethodType string

const (
	AuthMethodTypeTOTP        AuthMethodType = "totp"         // Time-based One-Time Password
	AuthMethodTypeEmailOTP    AuthMethodType = "email_otp"    // One-Time Password sent via email (for passwordless or 2FA)
	AuthMethodTypeMagicLink   AuthMethodType = "magic_link"   // Passwordless login via a link sent to email
	AuthMethodTypeOAuthGoogle AuthMethodType = "oauth_google" // Google OAuth
	AuthMethodTypeOAuthGithub AuthMethodType = "oauth_github" // GitHub OAuth
	AuthMethodTypeFIDO2       AuthMethodType = "fido2"        // FIDO2/WebAuthn
	// Add other methods as needed
)

// AuthMethod represents a specific authentication method enabled for a user's account.
// This allows for multiple factors like TOTP, FIDO2 keys, OAuth, etc.
type AuthMethod struct {
	ID         string    `json:"id" gorm:"primaryKey"`                                                     // Primary Key
	UserAuthID string    `json:"user_auth_id" gorm:"size:191;not null;index"`                              // Foreign Key to UserAuth.ID
	UserAuth   *UserAuth `json:"-" gorm:"foreignKey:UserAuthID;references:ID;constraint:OnDelete:CASCADE"` // Optional: for eager loading

	Type AuthMethodType `json:"type" gorm:"size:50;not null" validate:"required"` // Type of authentication method (e.g., "totp", "oauth_google")

	// Common fields for various auth methods
	IsEnabled    bool       `json:"is_enabled" gorm:"default:true"`                             // Whether this method is currently active
	IsVerified   bool       `json:"is_verified" gorm:"default:false"`                           // e.g., TOTP setup confirmed, OAuth successful
	VerifiedAt   *time.Time `json:"verified_at,omitempty"`                                      // Timestamp when this method was verified/added
	LastUsedAt   *time.Time `json:"last_used_at,omitempty"`                                     // Timestamp when this method was last used for login
	FriendlyName string     `json:"friendly_name,omitempty" gorm:"size:100" validate:"max=100"` // User-defined name (e.g., "My YubiKey", "Authenticator App")

	// Method-specific data (should be encrypted at rest if sensitive)
	// For TOTP: stores the encrypted TOTP secret.
	// For OAuth: might store encrypted refresh tokens or provider-specific identifiers.
	// For MagicLink/EmailOTP: could store the token itself or its hash.
	Secret string `json:"-" gorm:"type:text"` // Encrypted secret/token

	// For TOTP: stores hashed recovery codes.
	RecoveryCodes []string `json:"-" gorm:"type:json"` // Hashed recovery codes

	// For OAuth providers: stores the user's ID from that provider.
	ProviderUserID string `json:"provider_user_id,omitempty" gorm:"size:255"`

	// For OAuth or other methods needing to store additional non-sensitive metadata.
	// Could be a JSON string.
	Metadata string `json:"metadata,omitempty" gorm:"type:text"`

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" gorm:"index"` // For soft deletes
}

// TableName specifies the table name for GORM
func (AuthMethod) TableName() string {
	return "auth_methods"
}

// LoginAttempt represents a login attempt record
type LoginAttempt struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	UserID      *string   `json:"user_id,omitempty" gorm:"index"` // Nullable foreign key
	IPAddress   string    `json:"ip_address" gorm:"size:45"`      // IPv4/IPv6
	UserAgent   string    `json:"user_agent" gorm:"size:255"`
	Success     bool      `json:"success" gorm:"default:false"`
	FailReason  string    `json:"fail_reason,omitempty" gorm:"size:255"`
	AttemptedAt time.Time `json:"attempted_at" gorm:"autoCreateTime"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	// Associations
	User *User `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

func (LoginAttempt) TableName() string {
	return "login_attempts"
}



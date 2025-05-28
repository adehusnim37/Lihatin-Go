package models

import "time"

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
	ID         string `json:"id"`          // Primary Key
	UserAuthID string `json:"user_auth_id"` // Foreign Key to UserAuth.ID
	UserAuth   *UserAuth `json:"-" gorm:"foreignKey:UserAuthID"` // Optional: for eager loading

	Type AuthMethodType `json:"type" validate:"required"` // Type of authentication method (e.g., "totp", "oauth_google")

	// Common fields for various auth methods
	IsEnabled    bool       `json:"is_enabled" default:"true"`           // Whether this method is currently active
	IsVerified   bool       `json:"is_verified" default:"false"`         // e.g., TOTP setup confirmed, OAuth successful
	VerifiedAt   *time.Time `json:"verified_at,omitempty"`             // Timestamp when this method was verified/added
	LastUsedAt   *time.Time `json:"last_used_at,omitempty"`            // Timestamp when this method was last used for login
	FriendlyName string     `json:"friendly_name,omitempty" validate:"max=100"` // User-defined name (e.g., "My YubiKey", "Authenticator App")

	// Method-specific data (should be encrypted at rest if sensitive)
	// For TOTP: stores the encrypted TOTP secret.
	// For OAuth: might store encrypted refresh tokens or provider-specific identifiers.
	// For MagicLink/EmailOTP: could store the token itself or its hash.
	Secret string `json:"-"` // Encrypted secret/token

	// For TOTP: stores hashed recovery codes.
	RecoveryCodes []string `json:"-" gorm:"type:json"` // Hashed recovery codes

	// For OAuth providers: stores the user's ID from that provider.
	ProviderUserID string `json:"provider_user_id,omitempty"`

	// For OAuth or other methods needing to store additional non-sensitive metadata.
	// Could be a JSON string.
	Metadata string `json:"metadata,omitempty"`

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" gorm:"index"` // For soft deletes
}

// TableName specifies the table name for GORM
func (AuthMethod) TableName() string {
	return "AuthMethod"
}

package models

import "time"

type UserAuth struct {
	ID     string `json:"id"`                                      // Primary Key
	UserID string `json:"user_id"`                                 // Foreign Key to User.ID, should have a unique constraint
	User   *User  `json:"user,omitempty" gorm:"foreignKey:UserID"` // Optional: for eager loading user details

	// Password-based authentication
	PasswordHash string `json:"-"` // Store the hashed password, never expose in JSON responses

	// Email verification
	IsEmailVerified                 bool       `json:"is_email_verified" default:"false"`
	EmailVerificationToken          string     `json:"-"` // Token sent to user's email
	EmailVerificationTokenExpiresAt *time.Time `json:"-"` // Expiry for the verification token

	// Password reset
	PasswordResetToken          string     `json:"-"` // Token for password reset
	PasswordResetTokenExpiresAt *time.Time `json:"-"` // Expiry for the reset token

	// Account status & security
	LastLoginAt         *time.Time `json:"last_login_at,omitempty"`
	FailedLoginAttempts int        `json:"-" default:"0"`            // Counter for failed login attempts
	LockoutUntil        *time.Time `json:"-"`                        // Timestamp until which the account is locked
	IsActive            bool       `json:"is_active" default:"true"` // General account active status
	IsTOTPEnabled       bool       `json:"is_totp_enabled" default:"false"` // Whether TOTP is enabled for this user

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" gorm:"index"` // For soft deletes
}

// TableName specifies the table name for GORM
func (UserAuth) TableName() string {
	return "UserAuth"
}

package repositories

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/adehusnim37/lihatin-go/models/user"
	"gorm.io/gorm"
)

// UserAuthRepository handles UserAuth database operations
type UserAuthRepository struct {
	db *gorm.DB
}

// NewUserAuthRepository creates a new UserAuth repository
func NewUserAuthRepository(db *gorm.DB) *UserAuthRepository {
	return &UserAuthRepository{db: db}
}

// CreateUserAuth creates a new UserAuth record
func (r *UserAuthRepository) CreateUserAuth(userAuth *user.UserAuth) error {
	if err := r.db.Create(userAuth).Error; err != nil {
		return fmt.Errorf("failed to create user auth: %w", err)
	}
	return nil
}

// GetUserAuthByUserID retrieves UserAuth by user ID
func (r *UserAuthRepository) GetUserAuthByUserID(userID string) (*user.UserAuth, error) {
	var userAuth user.UserAuth
	if err := r.db.Where("user_id = ?", userID).First(&userAuth).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("failed to get user auth: %w", err)
	}
	return &userAuth, nil
}

// GetUserAuthByID retrieves UserAuth by ID
func (r *UserAuthRepository) GetUserAuthByID(id string) (*user.UserAuth, error) {
	var userAuth user.UserAuth
	if err := r.db.First(&userAuth, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("failed to get user auth by ID: %w", err)
	}
	return &userAuth, nil
}

// UpdateUserAuth updates UserAuth record
func (r *UserAuthRepository) UpdateUserAuth(userAuth *user.UserAuth) error {
	if err := r.db.Save(userAuth).Error; err != nil {
		return fmt.Errorf("failed to update user auth: %w", err)
	}
	return nil
}

// SetEmailVerificationToken sets email verification token and expiry
func (r *UserAuthRepository) SetEmailVerificationToken(userID, token string, expiresAt time.Time) error {
	err := r.db.Model(&user.UserAuth{}).
		Where("user_id = ?", userID).
		Updates(map[string]any{
			"email_verification_token":            token,
			"email_verification_token_expires_at": expiresAt,
		}).Error

	if err != nil {
		return fmt.Errorf("failed to set email verification token: %w", err)
	}
	return nil
}

// VerifyEmail marks email as verified and clears verification token
func (r *UserAuthRepository) VerifyEmail(token string) error {
	result := r.db.Model(&user.UserAuth{}).
		Where("email_verification_token = ? AND email_verification_token_expires_at > ?", token, time.Now()).
		Updates(map[string]interface{}{
			"is_email_verified":                   true,
			"email_verification_token":            "",
			"email_verification_token_expires_at": nil,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to verify email: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("invalid or expired verification token: %s", token)
	}

	return nil
}

// SetPasswordResetToken sets password reset token and expiry
func (r *UserAuthRepository) SetPasswordResetToken(userID, token string, expiresAt time.Time) error {
	err := r.db.Model(&user.UserAuth{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"password_reset_token":            token,
			"password_reset_token_expires_at": expiresAt,
		}).Error

	if err != nil {
		return fmt.Errorf("failed to set password reset token: %w", err)
	}
	return nil
}

// ValidatePasswordResetToken validates password reset token
func (r *UserAuthRepository) ValidatePasswordResetToken(token string) (*user.UserAuth, error) {
	var userAuth user.UserAuth
	err := r.db.Where("password_reset_token = ? AND password_reset_token_expires_at > ?", token, time.Now()).
		First(&userAuth).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("invalid or expired reset token")
		}
		return nil, fmt.Errorf("failed to validate password reset token: %w", err)
	}

	return &userAuth, nil
}

// ResetPassword updates password and clears reset token
func (r *UserAuthRepository) ResetPassword(token, hashedPassword string) error {
	result := r.db.Model(&user.UserAuth{}).
		Where("password_reset_token = ? AND password_reset_token_expires_at IS NOT NULL AND password_reset_token_expires_at > ?", token, time.Now()).
		Updates(map[string]interface{}{
			"password_hash":                   hashedPassword,
			"password_reset_token":            "",
			"password_reset_token_expires_at": nil,
			"failed_login_attempts":           0,
			"lockout_until":                   nil,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to reset password: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("invalid or expired reset token")
	}

	return nil
}

// UpdateLastLogin updates last login timestamp
func (r *UserAuthRepository) UpdateLastLogin(userID string) error {
	now := time.Now()
	err := r.db.Model(&user.UserAuth{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"last_login_at":         &now,
			"failed_login_attempts": 0,
			"lockout_until":         nil,
		}).Error

	if err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}
	return nil
}

// IncrementFailedLogin increments failed login attempts
func (r *UserAuthRepository) IncrementFailedLogin(userID string) error {
	// Get current failed attempts
	var userAuth user.UserAuth
	if err := r.db.Where("user_id = ?", userID).First(&userAuth).Error; err != nil {
		return fmt.Errorf("failed to get user auth for failed login: %w", err)
	}

	newAttempts := userAuth.FailedLoginAttempts + 1
	updates := map[string]interface{}{
		"failed_login_attempts": newAttempts,
	}

	// Lock account if too many failed attempts (e.g., 5 attempts = 15 min lockout)
	if newAttempts >= 5 {
		lockoutDuration := time.Duration(newAttempts-4) * 15 * time.Minute // Escalating lockout
		lockoutUntil := time.Now().Add(lockoutDuration)
		updates["lockout_until"] = &lockoutUntil
	}

	err := r.db.Model(&user.UserAuth{}).
		Where("user_id = ?", userID).
		Updates(updates).Error

	if err != nil {
		return fmt.Errorf("failed to increment failed login: %w", err)
	}
	return nil
}

// IsAccountLocked checks if account is currently locked
func (r *UserAuthRepository) IsAccountLocked(userID string) (bool, error) {
	var userAuth user.UserAuth
	if err := r.db.Where("user_id = ?", userID).First(&userAuth).Error; err != nil {
		return false, fmt.Errorf("failed to check account lock status: %w", err)
	}

	if userAuth.LockoutUntil != nil && time.Now().Before(*userAuth.LockoutUntil) {
		return true, nil
	}

	return false, nil
}

// UnlockAccount removes account lockout
func (r *UserAuthRepository) UnlockAccount(userID string) error {
	err := r.db.Model(&user.UserAuth{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"failed_login_attempts": 0,
			"lockout_until":         nil,
		}).Error

	if err != nil {
		return fmt.Errorf("failed to unlock account: %w", err)
	}
	return nil
}

// ActivateAccount activates a user account
func (r *UserAuthRepository) ActivateAccount(userID string) error {
	err := r.db.Model(&user.UserAuth{}).
		Where("user_id = ?", userID).
		Update("is_active", true).Error

	if err != nil {
		return fmt.Errorf("failed to activate account: %w", err)
	}
	return nil
}

// DeactivateAccount deactivates a user account
func (r *UserAuthRepository) DeactivateAccount(userID string) error {
	err := r.db.Model(&user.UserAuth{}).
		Where("user_id = ?", userID).
		Update("is_active", false).Error

	if err != nil {
		return fmt.Errorf("failed to deactivate account: %w", err)
	}
	return nil
}

// UpdatePassword updates user password
func (r *UserAuthRepository) UpdatePassword(userID, hashedPassword string) error {
	err := r.db.Model(&user.UserAuth{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"password_hash":         hashedPassword,
			"failed_login_attempts": 0,
			"lockout_until":         nil,
		}).Error

	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}
	return nil
}

// GetUserForLogin gets user with auth data for login
func (r *UserAuthRepository) GetUserForLogin(emailOrUsername string) (*user.User, *user.UserAuth, error) {
	var userx user.User
	var userAuth user.UserAuth

	// First get the user
	err := r.db.Where("email = ? OR username = ?", emailOrUsername, emailOrUsername).First(&userx).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil, sql.ErrNoRows
		}
		return nil, nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Then get the auth data
	err = r.db.Where("user_id = ?", userx.ID).First(&userAuth).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil, fmt.Errorf("user auth data not found")
		}
		return nil, nil, fmt.Errorf("failed to get user auth: %w", err)
	}

	return &userx, &userAuth, nil
}

// DeleteUserAuth soft deletes UserAuth record
func (r *UserAuthRepository) DeleteUserAuth(userID string) error {
	err := r.db.Where("user_id = ?", userID).Delete(&user.UserAuth{}).Error
	if err != nil {
		return fmt.Errorf("failed to delete user auth: %w", err)
	}
	return nil
}

// GetActiveUserAuthCount gets count of active user auth records
func (r *UserAuthRepository) GetActiveUserAuthCount() (int64, error) {
	var count int64
	err := r.db.Model(&user.UserAuth{}).Where("is_active = ?", true).Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("failed to get active user auth count: %w", err)
	}
	return count, nil
}

// GetUserAuthStats gets user authentication statistics
func (r *UserAuthRepository) GetUserAuthStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total users
	var totalUsers int64
	if err := r.db.Model(&user.UserAuth{}).Count(&totalUsers).Error; err != nil {
		return nil, fmt.Errorf("failed to count total users: %w", err)
	}
	stats["total_users"] = totalUsers

	// Active users
	var activeUsers int64
	if err := r.db.Model(&user.UserAuth{}).Where("is_active = ?", true).Count(&activeUsers).Error; err != nil {
		return nil, fmt.Errorf("failed to count active users: %w", err)
	}
	stats["active_users"] = activeUsers

	// Verified users
	var verifiedUsers int64
	if err := r.db.Model(&user.UserAuth{}).Where("is_email_verified = ?", true).Count(&verifiedUsers).Error; err != nil {
		return nil, fmt.Errorf("failed to count verified users: %w", err)
	}
	stats["verified_users"] = verifiedUsers

	// Locked users
	var lockedUsers int64
	if err := r.db.Model(&user.UserAuth{}).Where("lockout_until > ?", time.Now()).Count(&lockedUsers).Error; err != nil {
		return nil, fmt.Errorf("failed to count locked users: %w", err)
	}
	stats["locked_users"] = lockedUsers

	return stats, nil
}

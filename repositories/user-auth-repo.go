package repositories

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/models/user"
	"github.com/adehusnim37/lihatin-go/utils"
	"gorm.io/datatypes"
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
	//check if user with the userID already exists
	var existingUserAuth user.UserAuth
	if err := r.db.Where("user_id = ?", userAuth.UserID).First(&existingUserAuth).Error; err == nil {
		return utils.ErrUserAlreadyExists
	}

	if err := r.db.Create(userAuth).Error; err != nil {
		return utils.ErrUserCreationFailed
	}
	return nil
}

// GetUserAuthByUserID retrieves UserAuth by user ID
func (r *UserAuthRepository) GetUserAuthByUserID(userID string) (*user.UserAuth, error) {
	var userAuth user.UserAuth
	if err := r.db.Where("user_id = ?", userID).First(&userAuth).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, utils.ErrUserNotFound
		}
		return nil, utils.ErrUserFindFailed
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
		return nil, utils.ErrUserNotFound
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
func (r *UserAuthRepository) SetEmailVerificationToken(userID, token string, source user.EmailVerificationSource, expiry ...time.Time) error {
	// Set token expiry (2 hours from now)
	var expiresAt time.Time
	if len(expiry) > 0 {
		expiresAt = expiry[0]
	} else {
		expiresAt = time.Now().Add(time.Duration(utils.GetEnvAsInt("EXPIRE_EMAIL_VERIFICATION_TOKEN_HOURS", 2)) * time.Hour)
	}
	err := r.db.Model(&user.UserAuth{}).
		Where("user_id = ?", userID).
		Updates(map[string]any{
			"email_verification_token":            token,
			"email_verification_token_expires_at": expiresAt,
			"email_verification_source":           source,
		}).Error

	if err != nil {
		return utils.ErrCreateVerificationTokenFailed
	}
	return nil
}

// VerifyEmail marks email as verified and clears verification token
func (r *UserAuthRepository) VerifyEmail(token string) (res dto.VerifyEmailResponse, err error) {
	tx := r.db.Begin()

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			err = fmt.Errorf("panic during email verification: %v", r)
		}
	}()

	// 1️⃣ Cari userAuth berdasarkan token dan pastikan belum expired
	var userAuth user.UserAuth
	if err = tx.
		Where("email_verification_token = ? AND email_verification_token_expires_at > ?", token, time.Now()).
		First(&userAuth).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.VerifyEmailResponse{}, utils.ErrEmailVerificationTokenInvalidOrExpired
		}
		return dto.VerifyEmailResponse{}, utils.ErrEmailVerificationFailed
	}

	// 2️⃣ Ambil data user
	var usr user.User
	if err = tx.Where("id = ?", userAuth.UserID).First(&usr).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.VerifyEmailResponse{}, utils.ErrUserNotFound
		}
		return dto.VerifyEmailResponse{}, utils.ErrEmailVerificationFailed
	}

	// 3️⃣ Pastikan user belum diverifikasi
	if userAuth.IsEmailVerified {
		tx.Rollback()
		return dto.VerifyEmailResponse{
			Email:    usr.Email,
			Username: usr.Username,
			Source:   userAuth.EmailVerificationSource,
			OldEmail: "",
		}, utils.ErrEmailAlreadyVerified
	}

	// 4️⃣ Update kolom verifikasi secara aman (reset token juga)
	if err = tx.Model(&userAuth).Updates(map[string]any{
		"is_email_verified":                   true,
		"email_verification_token":            "",
		"email_verification_token_expires_at": nil,
	}).Error; err != nil {
		tx.Rollback()
		return dto.VerifyEmailResponse{}, utils.ErrUserAuthUpdateFailed
	}

	// 5️⃣ (Opsional) Insert log ke HistoryUser
	history := user.HistoryUser{
		UserID:     usr.ID,
		ActionType: "email_verification",
		OldValue:   datatypes.JSON([]byte(`{"is_email_verified": false}`)),
		NewValue:   datatypes.JSON([]byte(`{"is_email_verified": true}`)),
		Reason:     "User verified email successfully",
		ChangedAt:  time.Now(),
	}
	if err := tx.Create(&history).Error; err != nil {
		tx.Rollback()
		utils.Logger.Warn("failed to record history for email verification", "user_id", usr.ID, "error", err)
	}

	// 6️⃣ Ambil history email change terakhir untuk user ini (jika ada)
	var lastEmailChange user.HistoryUser
	var oldEmail string
	var revokeToken string

	err = tx.Where("user_id = ? AND action_type = ?", usr.ID, user.ActionEmailChange).
		Order("changed_at DESC").
		First(&lastEmailChange).Error
	if err == nil {
		var oldEmailMap map[string]string
		if err := json.Unmarshal(lastEmailChange.OldValue, &oldEmailMap); err == nil {
			oldEmail = oldEmailMap["email"]
		}
		revokeToken = lastEmailChange.RevokeToken
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		// Only rollback if there's a real database error, not if history not found
		tx.Rollback()
		return dto.VerifyEmailResponse{}, utils.ErrEmailVerificationFailed
	}

	// 7️⃣ Commit transaksi
	if err = tx.Commit().Error; err != nil {
		return dto.VerifyEmailResponse{}, utils.ErrEmailVerificationFailed
	}

	return dto.VerifyEmailResponse{
		Email:    usr.Email,
		Username: usr.Username,
		Source:   userAuth.EmailVerificationSource,
		OldEmail: oldEmail,
		Token:    revokeToken,
	}, nil
}

func (r *UserAuthRepository) ChangeEmail(userID, newEmail, ipAddress, userAgent string) error {
	var usr user.User
	var usrHistory user.HistoryUser
	var userAuth user.UserAuth

	// Get user data
	if err := r.db.Where("id = ?", userID).First(&usr).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return utils.ErrUserNotFound
		}
		return utils.ErrUserFindFailed
	}

	// Get user auth data
	if err := r.db.Where("user_id = ?", userID).First(&userAuth).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return utils.ErrUserNotFound
		}
		return utils.ErrUserFindFailed
	}

	if !userAuth.IsEmailVerified {
		return utils.ErrUserEmailNotVerified
	}

	// Check if account is active
	if !userAuth.IsActive {
		return utils.ErrUserAccountDeactivated
	}

	// Check if new email is same as current
	if usr.Email == newEmail {
		return utils.ErrUserEmailSameAsCurrent
	}

	// Check if new email already exists
	var existingUser user.User
	if err := r.db.Where("email = ?", newEmail).First(&existingUser).Error; err == nil {
		return utils.ErrUserEmailExists
	}

	// 🔒 SECURITY: Check email change history dalam 30 hari terakhir
	thirtyDaysAgo := time.Now().Add(-30 * 24 * time.Hour)

	// Check untuk email_change atau email_change_revoked dalam 30 hari
	var recentHistory []user.HistoryUser
	err := r.db.Where(
		"user_id = ? AND action_type IN (?, ?) AND changed_at > ?",
		userID,
		user.ActionEmailChange,
		"email_change_revoked",
		thirtyDaysAgo,
	).Order("changed_at DESC").Find(&recentHistory).Error

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		utils.Logger.Error("Failed to check email change history", "user_id", userID, "error", err)
		return utils.ErrUserDatabaseError
	}

	// Jika ada history dalam 30 hari terakhir
	if len(recentHistory) > 0 {
		lastChange := recentHistory[0]

		// 🚨 Jika ada revoke dalam 30 hari = SUSPICIOUS ACTIVITY
		if lastChange.ActionType == "email_change_revoked" {
			utils.Logger.Warn("Suspicious activity: Email change attempted after recent revoke",
				"user_id", userID,
				"last_revoke", lastChange.ChangedAt,
			)
			return utils.ErrEmailChangeLocked
		}

		// 🚨 Jika sudah change email dalam 30 hari = RATE LIMIT
		if lastChange.ActionType == user.ActionEmailChange {
			utils.Logger.Warn("Email change rate limit exceeded",
				"user_id", userID,
				"last_change", lastChange.ChangedAt,
			)
			return utils.ErrEmailChangeRateLimitExceeded
		}
	}

	token, err := utils.GenerateSecureToken(25)
	if err != nil {
		// handle error, e.g. log and use empty string or return error from function
		utils.Logger.Error("Failed to generate revoke token", "error", err)
		return utils.ErrTokenGenerationFailed
	}

	// Archive current user data to history table
	oldEmailJSON, _ := json.Marshal(map[string]string{"email": usr.Email})
	newEmailJSON, _ := json.Marshal(map[string]string{"email": newEmail})

	usrHistory = user.HistoryUser{
		UserID:        usr.ID,
		ActionType:    user.ActionEmailChange,
		OldValue:      datatypes.JSON(oldEmailJSON),
		NewValue:      datatypes.JSON(newEmailJSON),
		ChangedAt:     time.Now(),
		ChangedBy:     &usr.ID, // Self-change
		Reason:        "User initiated email change",
		IPAddress:     &ipAddress,
		UserAgent:     &userAgent,
		RevokeToken:   token,
		RevokeExpires: func(t time.Time) *time.Time { return &t }(time.Now().Add(24 * time.Hour)),
	}

	if err := r.db.Create(&usrHistory).Error; err != nil {
		return utils.ErrUserHistoryCreationFailed
	}

	// Update email in user table
	if err := r.db.Model(&usr).Where("id = ?", userID).Update("email", newEmail).Error; err != nil {
		return utils.ErrUserUpdateFailed
	}

	// Mark email as unverified after change
	if err := r.db.Model(&userAuth).Where("user_id = ?", userID).Updates(map[string]interface{}{
		"is_email_verified":                   false,
		"email_verification_token":            "",
		"email_verification_token_expires_at": nil,
	}).Error; err != nil {
		return utils.ErrUserAuthUpdateFailed
	}

	return nil
}

// UndoChangeEmail reverts email change if the user has not verified the new email
func (r *UserAuthRepository) UndoChangeEmail(revokeToken string) (email, username string, err error) {
	tx := r.db.Begin()

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var usrHistory user.HistoryUser
	var usr user.User
	var userAuth user.UserAuth

	// 1️⃣ Cari history berdasarkan revoke token
	if err := tx.Where("revoke_token = ?", revokeToken).First(&usrHistory).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", "", utils.ErrRevokeTokenNotFound
		}
		return "", "", utils.ErrUserHistoryFindFailed
	}

	// 2️⃣ Pastikan token belum expired
	if usrHistory.RevokeExpires == nil || time.Now().After(*usrHistory.RevokeExpires) {
		tx.Rollback()
		return "", "", utils.ErrRevokeTokenExpired
	}

	// 3️⃣ Pastikan action type adalah email change
	if usrHistory.ActionType != user.ActionEmailChange {
		tx.Rollback()
		return "", "", utils.ErrInvalidActionType
	}

	// 4️⃣ Ambil user dan auth
	if err := tx.Where("id = ?", usrHistory.UserID).First(&usr).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", "", utils.ErrUserNotFound
		}
		return "", "", utils.ErrUserFindFailed
	}

	if err := tx.Where("user_id = ?", usrHistory.UserID).First(&userAuth).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", "", utils.ErrUserNotFound
		}
		return "", "", utils.ErrUserFindFailed
	}

	// 5️⃣ Ekstrak old email dari history
	var oldEmailMap map[string]string
	if err := json.Unmarshal(usrHistory.OldValue, &oldEmailMap); err != nil {
		tx.Rollback()
		return "", "", utils.ErrInvalidHistoryData
	}
	oldEmail := oldEmailMap["email"]
	if oldEmail == "" {
		tx.Rollback()
		return "", "", utils.ErrInvalidHistoryData
	}

	// 6️⃣ Kembalikan email lama
	if err := tx.Model(&usr).Where("id = ?", usrHistory.UserID).Update("email", oldEmail).Error; err != nil {
		tx.Rollback()
		return "", "", utils.ErrUserUpdateFailed
	}

	// 7️⃣ Reset flag verifikasi (email lama sudah pasti verified)
	if err := tx.Model(&userAuth).Where("user_id = ?", usrHistory.UserID).Updates(map[string]any{
		"is_email_verified":                   true,
		"email_verification_token":            "",
		"email_verification_token_expires_at": nil,
	}).Error; err != nil {
		tx.Rollback()
		return "", "", utils.ErrUserAuthUpdateFailed
	}

	// 8️⃣ Tandai token sudah digunakan (expire immediately)
	now := time.Now()
	if err := tx.Model(&usrHistory).Updates(map[string]any{
		"revoke_expires": &now,
	}).Error; err != nil {
		tx.Rollback()
		utils.Logger.Warn("Failed to mark revoke token as used", "error", err)
	}

	// 9️⃣ Log revoke action ke history
	revokeHistory := user.HistoryUser{
		UserID:     usr.ID,
		ActionType: "email_change_revoked",
		OldValue:   usrHistory.NewValue, // New email becomes old
		NewValue:   usrHistory.OldValue, // Old email becomes new
		Reason:     "User revoked email change using revoke token",
		ChangedAt:  time.Now(),
		ChangedBy:  &usr.ID,
	}
	if err := tx.Create(&revokeHistory).Error; err != nil {
		tx.Rollback()
		utils.Logger.Warn("Failed to record revoke history", "error", err)
	}

	// 🔟 Commit transaksi
	if err := tx.Commit().Error; err != nil {
		return "", "", utils.ErrUserUpdateFailed
	}

	return oldEmail, usr.Username, nil
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

// Logout and remove sessionID for user
func (ur *UserAuthRepository) Logout(userID string) error {
	var userAuth user.UserAuth

	result := ur.db.Where("user_id = ?", userID).First(&userAuth)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return utils.ErrSessionNotFound
		}
		utils.Logger.Error("Error finding user session", "user_id", userID, "error", result.Error)
		return utils.ErrUserDatabaseError
	}

	result = ur.db.Model(&userAuth).Updates(map[string]interface{}{
		"session_id": nil,
		"device_id":  nil,
		"last_ip":    nil,
	})

	return nil
}

// CheckEmailChangeEligibility checks if user is eligible to change email
// Returns error if user has changed email or revoked in last 30 days
func (r *UserAuthRepository) CheckEmailChangeEligibility(userID string) (eligible bool, daysRemaining int, err error) {
	thirtyDaysAgo := time.Now().Add(-30 * 24 * time.Hour)

	var recentHistory user.HistoryUser
	err = r.db.Where(
		"user_id = ? AND action_type IN (?, ?) AND changed_at > ?",
		userID,
		user.ActionEmailChange,
		"email_change_revoked",
		thirtyDaysAgo,
	).Order("changed_at DESC").First(&recentHistory).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// No recent history, user is eligible
			return true, 0, nil
		}
		return false, 0, err
	}

	// Calculate days remaining until eligible
	daysSinceChange := int(time.Since(recentHistory.ChangedAt).Hours() / 24)
	daysRemaining = 30 - daysSinceChange

	if daysRemaining <= 0 {
		return true, 0, nil
	}

	return false, daysRemaining, nil
}

// GetEmailChangeHistory returns email change history for a user
func (r *UserAuthRepository) GetEmailChangeHistory(userID string, days int) ([]user.HistoryUser, error) {
	var history []user.HistoryUser

	cutoffDate := time.Now().Add(-time.Duration(days) * 24 * time.Hour)

	err := r.db.Where(
		"user_id = ? AND action_type IN (?, ?, ?) AND changed_at > ?",
		userID,
		user.ActionEmailChange,
		"email_change_revoked",
		"email_verification",
		cutoffDate,
	).Order("changed_at DESC").Find(&history).Error

	if err != nil {
		return nil, err
	}

	return history, nil
}

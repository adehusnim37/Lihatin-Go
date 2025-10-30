package repositories

import (
	"fmt"
	"time"

	"github.com/adehusnim37/lihatin-go/models/user"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// LoginAttemptRepository handles login attempt database operations
type LoginAttemptRepository struct {
	db *gorm.DB
}

// NewLoginAttemptRepository creates a new login attempt repository
func NewLoginAttemptRepository(db *gorm.DB) *LoginAttemptRepository {
	return &LoginAttemptRepository{db: db}
}

// RecordLoginAttempt records a login attempt
func (r *LoginAttemptRepository) RecordLoginAttempt(ipAddress, userAgent string, success bool, failReason string, emailOrUsername string) error {
	attempt := &user.LoginAttempt{
		ID: uuid.New().String(),
		EmailOrUsername: emailOrUsername,
		IPAddress:       ipAddress,
		UserAgent:       userAgent,
		Success:         success,
		FailReason:      failReason,
	}

	if err := r.db.Create(attempt).Error; err != nil {
		return fmt.Errorf("failed to record login attempt: %w", err)
	}
	return nil
}

// GetLoginAttemptsByUserID retrieves login attempts for a user
func (r *LoginAttemptRepository) GetLoginAttemptsByUserID(userID string, limit int) ([]user.LoginAttempt, error) {
	var attempts []user.LoginAttempt
	query := r.db.Where("user_id = ?", userID).Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&attempts).Error; err != nil {
		return nil, fmt.Errorf("failed to get login attempts: %w", err)
	}
	return attempts, nil
}

// GetLoginAttemptsByIP retrieves login attempts by IP address
func (r *LoginAttemptRepository) GetLoginAttemptsByIP(ipAddress string, since time.Time) ([]user.LoginAttempt, error) {
	var attempts []user.LoginAttempt
	if err := r.db.Where("ip_address = ? AND created_at > ?", ipAddress, since).
		Order("created_at DESC").Find(&attempts).Error; err != nil {
		return nil, fmt.Errorf("failed to get login attempts by IP: %w", err)
	}
	return attempts, nil
}

// GetRecentFailedAttempts gets recent failed login attempts for a user
func (r *LoginAttemptRepository) GetRecentFailedAttempts(userID string, since time.Time) (int64, error) {
	var count int64
	if err := r.db.Model(&user.LoginAttempt{}).
		Where("user_id = ? AND success = ? AND created_at > ?", userID, false, since).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count failed attempts: %w", err)
	}
	return count, nil
}

// GetLoginStats returns login statistics
func (r *LoginAttemptRepository) GetLoginStats(userID string, days int) (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	since := time.Now().AddDate(0, 0, -days)

	// Total attempts
	var total int64
	if err := r.db.Model(&user.LoginAttempt{}).
		Where("user_id = ? AND created_at > ?", userID, since).
		Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count total attempts: %w", err)
	}
	stats["total_attempts"] = total

	// Successful attempts
	var successful int64
	if err := r.db.Model(&user.LoginAttempt{}).
		Where("user_id = ? AND success = ? AND created_at > ?", userID, true, since).
		Count(&successful).Error; err != nil {
		return nil, fmt.Errorf("failed to count successful attempts: %w", err)
	}
	stats["successful_attempts"] = successful

	// Failed attempts
	var failed int64
	if err := r.db.Model(&user.LoginAttempt{}).
		Where("user_id = ? AND success = ? AND created_at > ?", userID, false, since).
		Count(&failed).Error; err != nil {
		return nil, fmt.Errorf("failed to count failed attempts: %w", err)
	}
	stats["failed_attempts"] = failed

	// Success rate
	if total > 0 {
		stats["success_rate"] = float64(successful) / float64(total) * 100
	} else {
		stats["success_rate"] = 0.0
	}

	// Last successful login
	var lastSuccess user.LoginAttempt
	if err := r.db.Where("user_id = ? AND success = ?", userID, true).
		Order("created_at DESC").First(&lastSuccess).Error; err == nil {
		stats["last_successful_login"] = lastSuccess.CreatedAt
	}

	return stats, nil
}
// GetAllLoginAttempts retrieves all login attempts with pagination (admin only)
func (r *LoginAttemptRepository) GetAllLoginAttempts(limit, offset int, successFilter *bool) ([]user.LoginAttempt, int64, error) {
    var attempts []user.LoginAttempt
    var total int64

    // Build base query
    query := r.db.Model(&user.LoginAttempt{})



    // Apply success filter if provided
    if successFilter != nil {
        query = query.Where("success = ?", *successFilter)
    }

    // Get total count with filter applied
    if err := query.Count(&total).Error; err != nil {
        utils.Logger.Error("Failed to count login attempts", "error", err.Error())
        return nil, 0, utils.ErrLoginAttemptFailed
    }

    // Get attempts with pagination
    if err := query.
        Offset(offset).
        Limit(limit).
        Order("created_at DESC").
        Find(&attempts).Error; err != nil {
        utils.Logger.Error("Failed to retrieve login attempts", "error", err.Error())
        return nil, 0, utils.ErrLoginAttemptFailed
    }

    return attempts, total, nil
}

// CleanupOldAttempts removes login attempts older than specified days
func (r *LoginAttemptRepository) CleanupOldAttempts(olderThanDays int) error {
	cutoff := time.Now().AddDate(0, 0, -olderThanDays)
	if err := r.db.Where("created_at < ?", cutoff).Delete(&user.LoginAttempt{}).Error; err != nil {
		return fmt.Errorf("failed to cleanup old login attempts: %w", err)
	}
	return nil
}

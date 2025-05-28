package repositories

import (
	"fmt"
	"time"

	"github.com/adehusnim37/lihatin-go/models"
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
func (r *LoginAttemptRepository) RecordLoginAttempt(userID, ipAddress, userAgent string, success bool, failReason string) error {
	attempt := &models.LoginAttempt{
		ID:          uuid.New().String(),
		UserID:      userID,
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
		Success:     success,
		FailReason:  failReason,
		AttemptedAt: time.Now(),
	}

	if err := r.db.Create(attempt).Error; err != nil {
		return fmt.Errorf("failed to record login attempt: %w", err)
	}
	return nil
}

// GetLoginAttemptsByUserID retrieves login attempts for a user
func (r *LoginAttemptRepository) GetLoginAttemptsByUserID(userID string, limit int) ([]models.LoginAttempt, error) {
	var attempts []models.LoginAttempt
	query := r.db.Where("user_id = ?", userID).Order("attempted_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&attempts).Error; err != nil {
		return nil, fmt.Errorf("failed to get login attempts: %w", err)
	}
	return attempts, nil
}

// GetLoginAttemptsByIP retrieves login attempts by IP address
func (r *LoginAttemptRepository) GetLoginAttemptsByIP(ipAddress string, since time.Time) ([]models.LoginAttempt, error) {
	var attempts []models.LoginAttempt
	if err := r.db.Where("ip_address = ? AND attempted_at > ?", ipAddress, since).
		Order("attempted_at DESC").Find(&attempts).Error; err != nil {
		return nil, fmt.Errorf("failed to get login attempts by IP: %w", err)
	}
	return attempts, nil
}

// GetRecentFailedAttempts gets recent failed login attempts for a user
func (r *LoginAttemptRepository) GetRecentFailedAttempts(userID string, since time.Time) (int64, error) {
	var count int64
	if err := r.db.Model(&models.LoginAttempt{}).
		Where("user_id = ? AND success = ? AND attempted_at > ?", userID, false, since).
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
	if err := r.db.Model(&models.LoginAttempt{}).
		Where("user_id = ? AND attempted_at > ?", userID, since).
		Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count total attempts: %w", err)
	}
	stats["total_attempts"] = total

	// Successful attempts
	var successful int64
	if err := r.db.Model(&models.LoginAttempt{}).
		Where("user_id = ? AND success = ? AND attempted_at > ?", userID, true, since).
		Count(&successful).Error; err != nil {
		return nil, fmt.Errorf("failed to count successful attempts: %w", err)
	}
	stats["successful_attempts"] = successful

	// Failed attempts
	var failed int64
	if err := r.db.Model(&models.LoginAttempt{}).
		Where("user_id = ? AND success = ? AND attempted_at > ?", userID, false, since).
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
	var lastSuccess models.LoginAttempt
	if err := r.db.Where("user_id = ? AND success = ?", userID, true).
		Order("attempted_at DESC").First(&lastSuccess).Error; err == nil {
		stats["last_successful_login"] = lastSuccess.AttemptedAt
	}

	return stats, nil
}

// GetAllLoginAttempts retrieves all login attempts with pagination (admin only)
func (r *LoginAttemptRepository) GetAllLoginAttempts(limit, offset int) ([]models.LoginAttempt, int64, error) {
	var attempts []models.LoginAttempt
	var total int64

	// Get total count
	if err := r.db.Model(&models.LoginAttempt{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count login attempts: %w", err)
	}

	// Get attempts with pagination
	if err := r.db.Preload("User").Offset(offset).Limit(limit).
		Order("attempted_at DESC").Find(&attempts).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get login attempts: %w", err)
	}

	return attempts, total, nil
}

// CleanupOldAttempts removes login attempts older than specified days
func (r *LoginAttemptRepository) CleanupOldAttempts(olderThanDays int) error {
	cutoff := time.Now().AddDate(0, 0, -olderThanDays)
	if err := r.db.Where("attempted_at < ?", cutoff).Delete(&models.LoginAttempt{}).Error; err != nil {
		return fmt.Errorf("failed to cleanup old login attempts: %w", err)
	}
	return nil
}

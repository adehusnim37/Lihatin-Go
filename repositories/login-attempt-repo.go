package repositories

import (
	"fmt"
	"strconv"
	"time"

	"github.com/adehusnim37/lihatin-go/internal/pkg/errors"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/models/user"
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
		ID:              uuid.New().String(),
		EmailOrUsername: emailOrUsername,
		IPAddress:       ipAddress,
		UserAgent:       userAgent,
		Success:         success,
		FailReason:      failReason,
	}

	if err := r.db.Create(attempt).Error; err != nil {
		return errors.ErrCreateLoginAttemptFailed
	}
	return nil
}

// GetLoginAttemptsByUserID retrieves login attempts for a user
func (r *LoginAttemptRepository) GetLoginAttemptsByIDorUsername(userID string, username string) ([]user.LoginAttempt, error) {
	var attempts []user.LoginAttempt
	query := r.db.Where("id = ? OR email_or_username = ?", userID, username).Order("created_at DESC")

	if err := query.Find(&attempts).Error; err != nil {
		return nil, errors.ErrLoginAttemptFailed
	}

	return attempts, nil
}

// GetLoginAttemptByID retrieves a login attempt by its ID
func (r *LoginAttemptRepository) GetLoginAttemptByID(attemptID string) (*user.LoginAttempt, error) {
	var attempt user.LoginAttempt
	if err := r.db.Where("id = ?", attemptID).First(&attempt).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrLoginAttemptNotFound
		}
		return nil, errors.ErrLoginAttemptFailed
	}
	return &attempt, nil
}

// GetLoginAttemptsByIP retrieves login attempts by IP address
func (r *LoginAttemptRepository) GetLoginAttemptsByIP(ipAddress string, since time.Time) ([]user.LoginAttempt, error) {
	var attempts []user.LoginAttempt
	if err := r.db.Where("ip_address = ? AND created_at > ?", ipAddress, since).
		Order("created_at DESC").Find(&attempts).Error; err != nil {
		return nil, errors.ErrLoginAttemptFailed
	}
	return attempts, nil
}

// GetRecentFailedAttempts gets recent failed login attempts for a user
func (r *LoginAttemptRepository) GetRecentFailedAttempts(userID string, since time.Time) (int64, error) {
	var count int64
	if err := r.db.Model(&user.LoginAttempt{}).
		Where("user_id = ? AND success = ? AND created_at > ?", userID, false, since).
		Count(&count).Error; err != nil {
		return 0, errors.ErrLoginAttemptFailed
	}
	return count, nil
}

// GetLoginStats returns login statistics
func (r *LoginAttemptRepository) GetLoginStats(email_or_username string, days int) (map[string]interface{}, error) {
	stats := make(map[string]any)
	since := time.Now().AddDate(0, 0, -days)

	// Total attempts
	var total int64
	if err := r.db.Model(&user.LoginAttempt{}).
		Where("email_or_username = ? AND created_at > ?", email_or_username, since).
		Count(&total).Error; err != nil {
		return nil, errors.ErrLoginAttemptFailed
	}
	stats["total_attempts"] = total

	// Successful attempts
	var successful int64
	if err := r.db.Model(&user.LoginAttempt{}).
		Where("email_or_username = ? AND success = ? AND created_at > ?", email_or_username, true, since).
		Count(&successful).Error; err != nil {
		return nil, errors.ErrLoginAttemptFailed
	}
	stats["successful_attempts"] = successful

	// Failed attempts
	var failed int64
	if err := r.db.Model(&user.LoginAttempt{}).
		Where("email_or_username = ? AND success = ? AND created_at > ?", email_or_username, false, since).
		Count(&failed).Error; err != nil {
		return nil, errors.ErrLoginAttemptFailed
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
	if err := r.db.Where("user_id = ? AND success = ?", email_or_username, true).
		Order("created_at DESC").First(&lastSuccess).Error; err == nil {
		stats["last_successful_login"] = lastSuccess.CreatedAt
	}

	return stats, nil
}

// GetAllLoginAttempts retrieves all login attempts with enhanced filtering
func (r *LoginAttemptRepository) GetAllLoginAttempts(limit, offset int, filters map[string]interface{}) ([]user.LoginAttempt, int64, error) {
	var attempts []user.LoginAttempt
	var total int64

	// Build base query
	query := r.db.Model(&user.LoginAttempt{})

	// Apply filters dynamically
	if successStr, ok := filters["success_str"].(string); ok && successStr != "" {
		if success, err := strconv.ParseBool(successStr); err == nil {
			query = query.Where("success = ?", success)
		}
	}

	if id, ok := filters["id"].(string); ok && id != "" {
		query = query.Where("id = ?", id)
	}

	if username, ok := filters["username"].(string); ok && username != "" {
		query = query.Where("email_or_username = ?", username)
	}

	// For non-admin users filtering their own attempts
	if emailOrUsername, ok := filters["email_or_username"].(string); ok && emailOrUsername != "" {
		query = query.Where("email_or_username = ?", emailOrUsername)
	}

	// Search filter (partial match on email_or_username)
	if search, ok := filters["search"].(string); ok && search != "" {
		query = query.Where("email_or_username LIKE ?", "%"+search+"%")
	}

	// IP Address filter
	if ipAddress, ok := filters["ip_address"].(string); ok && ipAddress != "" {
		query = query.Where("ip_address = ?", ipAddress)
	}

	// Date range filters
	if fromDate, ok := filters["from_date"].(string); ok && fromDate != "" {
		if parsedDate, err := time.Parse(time.RFC3339, fromDate); err == nil {
			query = query.Where("created_at >= ?", parsedDate)
		}
	}

	if toDate, ok := filters["to_date"].(string); ok && toDate != "" {
		if parsedDate, err := time.Parse(time.RFC3339, toDate); err == nil {
			query = query.Where("created_at <= ?", parsedDate)
		}
	}

	// Get total count with filters applied
	if err := query.Count(&total).Error; err != nil {
		logger.Logger.Error("Failed to count login attempts", "error", err.Error())
		return nil, 0, errors.ErrLoginAttemptFailed
	}

	// Apply sorting with field and order
	sortField := "created_at"
	if field, ok := filters["sort_field"].(string); ok && field != "" {
		sortField = field
	}

	sortOrder := "DESC"
	if order, ok := filters["sort_order"].(string); ok && order == "asc" {
		sortOrder = "ASC"
	}

	// Get attempts with pagination
	if err := query.
		Offset(offset).
		Limit(limit).
		Order(sortField + " " + sortOrder).
		Find(&attempts).Error; err != nil {
		logger.Logger.Error("Failed to retrieve login attempts", "error", err.Error())
		return nil, 0, errors.ErrLoginAttemptFailed
	}

	return attempts, total, nil
}

// CleanupOldAttempts removes login attempts older than specified days
func (r *LoginAttemptRepository) CleanupOldAttempts(olderThanDays int) error {
	cutoff := time.Now().UTC().AddDate(0, 0, -olderThanDays)
	result := r.db.Where("created_at < ?", cutoff).Delete(&user.LoginAttempt{})
	logger.Logger.Info("CleanupOldAttempts executed", "cutoff", cutoff.Format(time.RFC3339), "rows_affected", result.RowsAffected)
	if result.Error != nil {
		return fmt.Errorf("failed to cleanup old login attempts: %w", result.Error)
	}
	return nil
}

// GetTopFailedIPs returns top IP addresses with most failed login attempts
func (r *LoginAttemptRepository) GetTopFailedIPs(limit int) ([]map[string]interface{}, error) {
	type Result struct {
		IPAddress   string    `json:"ip_address"`
		FailedCount int64     `json:"failed_count"`
		LastAttempt time.Time `json:"last_attempt"`
	}

	var results []Result
	err := r.db.Model(&user.LoginAttempt{}).
		Select("ip_address, COUNT(*) as failed_count, MAX(created_at) as last_attempt").
		Where("success = ?", false).
		Group("ip_address").
		Order("failed_count DESC").
		Limit(limit).
		Scan(&results).Error

	if err != nil {
		return nil, errors.ErrLoginAttemptFailed
	}

	// Convert to map slice
	var topIPs []map[string]interface{}
	for _, r := range results {
		topIPs = append(topIPs, map[string]interface{}{
			"ip_address":   r.IPAddress,
			"failed_count": r.FailedCount,
			"last_attempt": r.LastAttempt,
		})
	}

	return topIPs, nil
}

// GetAttemptsByHour returns login attempts grouped by hour
func (r *LoginAttemptRepository) GetAttemptsByHour(days int, emailOrUsername string) ([]map[string]interface{}, error) {
	since := time.Now().AddDate(0, 0, -days)

	type HourlyResult struct {
		Hour         int   `json:"hour"`
		TotalCount   int64 `json:"total_count"`
		SuccessCount int64 `json:"success_count"`
		FailedCount  int64 `json:"failed_count"`
	}

	query := r.db.Model(&user.LoginAttempt{}).
		Select("EXTRACT(HOUR FROM created_at) as hour, COUNT(*) as total_count, SUM(CASE WHEN success THEN 1 ELSE 0 END) as success_count, SUM(CASE WHEN NOT success THEN 1 ELSE 0 END) as failed_count").
		Where("created_at > ?", since).
		Group("hour").
		Order("hour ASC")

	if emailOrUsername != "" {
		query = query.Where("email_or_username = ?", emailOrUsername)
	}

	var results []HourlyResult
	if err := query.Scan(&results).Error; err != nil {
		return nil, errors.ErrLoginAttemptFailed
	}

	// Convert to map slice
	var hourlyAttempts []map[string]interface{}
	for _, r := range results {
		hourlyAttempts = append(hourlyAttempts, map[string]interface{}{
			"hour":          r.Hour,
			"total_count":   r.TotalCount,
			"success_count": r.SuccessCount,
			"failed_count":  r.FailedCount,
		})
	}

	return hourlyAttempts, nil
}

// GetSuspiciousActivity returns suspicious login patterns
func (r *LoginAttemptRepository) GetSuspiciousActivity() ([]map[string]interface{}, error) {
	// Get IPs with > 5 failed attempts in last hour
	oneHourAgo := time.Now().Add(-1 * time.Hour)

	type SuspiciousIP struct {
		IPAddress       string    `json:"ip_address"`
		FailedCount     int64     `json:"failed_count"`
		LastAttempt     time.Time `json:"last_attempt"`
		EmailsAttempted int64     `json:"emails_attempted"`
	}

	var results []SuspiciousIP
	err := r.db.Model(&user.LoginAttempt{}).
		Select("ip_address, COUNT(*) as failed_count, MAX(created_at) as last_attempt, COUNT(DISTINCT email_or_username) as emails_attempted").
		Where("success = ? AND created_at > ?", false, oneHourAgo).
		Group("ip_address").
		Having("COUNT(*) > ?", 5).
		Order("failed_count DESC").
		Scan(&results).Error

	if err != nil {
		return nil, errors.ErrLoginAttemptFailed
	}

	var suspicious []map[string]interface{}
	for _, result := range results {
		riskLevel := "medium"
		if result.FailedCount > 20 || result.EmailsAttempted > 10 {
			riskLevel = "critical"
		} else if result.FailedCount > 10 || result.EmailsAttempted > 5 {
			riskLevel = "high"
		}

		suspicious = append(suspicious, map[string]interface{}{
			"ip_address":       result.IPAddress,
			"failed_count":     result.FailedCount,
			"last_attempt":     result.LastAttempt,
			"emails_attempted": result.EmailsAttempted,
			"risk_level":       riskLevel,
		})
	}

	return suspicious, nil
}

// GetRecentActivitySummary returns summary of recent login activity
func (r *LoginAttemptRepository) GetRecentActivitySummary(hours int, emailOrUsername string) (map[string]interface{}, error) {
	since := time.Now().Add(time.Duration(-hours) * time.Hour)

	summary := make(map[string]interface{})

	query := r.db.Model(&user.LoginAttempt{}).Where("created_at > ?", since)
	if emailOrUsername != "" {
		query = query.Where("email_or_username = ?", emailOrUsername)
	}

	// Total attempts
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, errors.ErrLoginAttemptFailed
	}
	summary["total_attempts"] = total

	// Successful attempts
	var successful int64
	successQuery := r.db.Model(&user.LoginAttempt{}).Where("created_at > ? AND success = ?", since, true)
	if emailOrUsername != "" {
		successQuery = successQuery.Where("email_or_username = ?", emailOrUsername)
	}
	if err := successQuery.Count(&successful).Error; err != nil {
		return nil, errors.ErrLoginAttemptFailed
	}
	summary["successful_attempts"] = successful

	// Failed attempts
	summary["failed_attempts"] = total - successful

	// Unique IPs
	var uniqueIPs int64
	if err := query.Distinct("ip_address").Count(&uniqueIPs).Error; err != nil {
		return nil, errors.ErrLoginAttemptFailed
	}
	summary["unique_ips"] = uniqueIPs

	// Time range
	summary["hours"] = hours
	summary["since"] = since
	summary["period_start"] = since
	summary["period_end"] = time.Now()

	return summary, nil
}

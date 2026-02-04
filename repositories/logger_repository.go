package repositories

import (
	"fmt"
	"strings"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/models/logging"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// LoggerRepository handles database operations for user activity logs
type LoggerRepository struct {
	DB *gorm.DB
}

// NewLoggerRepository creates a new logger repository
func NewLoggerRepository(db *gorm.DB) *LoggerRepository {
	return &LoggerRepository{
		DB: db,
	}
}

// CreateLog inserts a new activity log into the database
func (r *LoggerRepository) CreateLog(log *logging.ActivityLog) error {
	if log.ID == "" {
		uuidV7, err := uuid.NewV7()
		if err != nil {
			return err
		}
		log.ID = uuidV7.String()
	}

	return r.DB.Create(log).Error
}

// GetLogsByUsername retrieves logs for a specific username with pagination
// Uses composite index: idx_username_created_at for optimal performance
func (r *LoggerRepository) GetLogsByUsername(username string, page, limit int, sort, orderBy, userRoleStr, userID string) ([]logging.ActivityLog, int64, error) {
	var logs []logging.ActivityLog
	var totalCount int64

	// Build base query
	query := r.DB.Model(&logging.ActivityLog{})

	isAdmin := strings.EqualFold(userRoleStr, "admin")
	if !isAdmin {
		query = query.Where("user_id = ?", userID)
	}

	// Count total records
	if err := query.
		Where("username = ?", username).
		Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	// Calculate offset
	offset := (page - 1) * limit

	// Build order clause
	orderClause := fmt.Sprintf("%s %s", sort, orderBy)

	// Fetch paginated results
	err := query.
		Where("username = ?", username).
		Order(orderClause).
		Offset(offset).
		Limit(limit).
		Find(&logs).Error

	return logs, totalCount, err
}

// GetAllLogs retrieves all logs with pagination and sorting
func (r *LoggerRepository) GetAllLogs(page, limit int, sort, orderBy, userRoleStr, userID string) ([]logging.ActivityLog, int64, error) {
	var logs []logging.ActivityLog
	var totalCount int64

	isAdmin := strings.EqualFold(userRoleStr, "admin")

	// Build base query (apply user scope for non-admin)
	query := r.DB.Model(&logging.ActivityLog{})
	if !isAdmin {
		query = query.Where("user_id = ?", userID)
	}

	// Count total records (scoped)
	if err := query.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	// Calculate offset
	offset := (page - 1) * limit

	// Build order clause
	orderClause := fmt.Sprintf("%s %s", sort, orderBy)

	// Fetch paginated results (scoped)
	err := query.
		Order(orderClause).
		Offset(offset).
		Limit(limit).
		Find(&logs).Error

	return logs, totalCount, err
}

// GetLogsByShortLink retrieves logs for a specific short link with pagination
func (r *LoggerRepository) GetLogsByShortLink(code string, page, limit int, sort, orderBy, userRoleStr, userID string) ([]logging.ActivityLog, int64, error) {
	var logs []logging.ActivityLog
	var totalCount int64

	// Count total records
	query := r.DB.Model(&logging.ActivityLog{}).Where("route LIKE ?", "%/"+code)

	isAdmin := strings.EqualFold(userRoleStr, "admin")
	if !isAdmin {
		query = query.Where("user_id = ?", userID)
	}

	if err := query.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	// Calculate offset
	offset := (page - 1) * limit

	// Build order clause
	orderClause := fmt.Sprintf("%s %s", sort, orderBy)

	// Fetch paginated results with selected columns
	err := query.
		Select([]string{
			"id", "status_code", "method", "route", "ip_address",
			"user_agent", "browser_info", "timestamp", "response_time",
			"request_headers", "response_body", "route_params",
			"created_at", "updated_at",
		}).
		Order(orderClause).
		Offset(offset).
		Limit(limit).
		Find(&logs).Error

	return logs, totalCount, err
}

// GetLogsWithFilter retrieves logs with advanced filtering and pagination
// Supports filtering by username, action, method, route, level, status code, IP address, and date range
func (r *LoggerRepository) GetLogsWithFilter(filter dto.ActivityLogFilter, page, limit int, sort, orderBy string) ([]logging.ActivityLog, int64, error) {
	var logs []logging.ActivityLog
	var totalCount int64

	// Build base query
	query := r.DB.Model(&logging.ActivityLog{})

	// Apply filters
	if filter.Username != "" {
		query = query.Where("username = ?", filter.Username)
	}
	if filter.Action != "" {
		query = query.Where("action = ?", filter.Action)
	}
	if filter.Method != "" {
		query = query.Where("method = ?", filter.Method)
	}
	if filter.Route != "" {
		query = query.Where("route LIKE ?", "%"+filter.Route+"%")
	}
	if filter.Level != "" {
		query = query.Where("level = ?", filter.Level)
	}
	if filter.StatusCode != nil {
		query = query.Where("status_code = ?", *filter.StatusCode)
	}
	if filter.IPAddress != "" {
		query = query.Where("ip_address = ?", filter.IPAddress)
	}
	if filter.DateFrom != nil {
		query = query.Where("created_at >= ?", filter.DateFrom)
	}
	if filter.DateTo != nil {
		query = query.Where("created_at <= ?", filter.DateTo)
	}
	if filter.APIKey != "" {
		query = query.Where("api_key = ?", filter.APIKey)
	}

	// Count total records with filters applied
	if err := query.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	// Calculate offset
	offset := (page - 1) * limit

	// Build order clause
	orderClause := fmt.Sprintf("%s %s", sort, orderBy)

	// Fetch paginated results
	err := query.
		Order(orderClause).
		Offset(offset).
		Limit(limit).
		Find(&logs).Error

	return logs, totalCount, err
}

// GetAllCountedLogs retrieves count of all logs by method type
func (r *LoggerRepository) GetAllCountedLogs(userRoleStr, userID string) (map[string]int64, error) {
	type MethodCount struct {
		Method string
		Count  int64
	}

	var results []MethodCount

	// Initialize counts with all standard HTTP methods set to 0
	counts := map[string]int64{
		"GET":    0,
		"POST":   0,
		"PUT":    0,
		"PATCH":  0,
		"DELETE": 0,
	}

	// Build base query
	query := r.DB.Model(&logging.ActivityLog{}).Select("method, COUNT(*) as count").Group("method")

	isAdmin := strings.EqualFold(userRoleStr, "admin")
	if !isAdmin {
		query = query.Where("user_id = ?", userID)
	}

	// Execute query
	if err := query.Scan(&results).Error; err != nil {
		return nil, err
	}

	// Update counts with actual values from database
	for _, result := range results {
		counts[result.Method] = result.Count
	}

	return counts, nil
}

// GetLogByID retrieves a single log by its ID
func (r *LoggerRepository) GetLogByID(logID, userRoleStr, userID string) (*logging.ActivityLog, error) {
	var log logging.ActivityLog

	// Build base query
	query := r.DB.Model(&logging.ActivityLog{}).Where("id = ?", logID)

	isAdmin := strings.EqualFold(userRoleStr, "admin")
	if !isAdmin {
		query = query.Where("user_id = ?", userID)
	}

	// Execute query
	if err := query.First(&log).Error; err != nil {
		return nil, err
	}

	return &log, nil
}

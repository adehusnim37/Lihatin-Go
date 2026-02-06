package repositories

import (
	"fmt"
	"strings"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
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
	logger.Logger.Info("Checking admin status", "isAdmin", isAdmin)

	if !isAdmin {
		query = query.Where("user_id = ?", userID)
	}

	logger.Logger.Info("userID", "userID", userID)

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
	logger.Logger.Info("Checking admin status", "isAdmin", isAdmin)

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

// GetLogsStats retrieves comprehensive stats of all logs
func (r *LoggerRepository) GetLogsStats(userRoleStr, userID string) (*dto.ActivityLogStatsResponse, error) {
	stats := &dto.ActivityLogStatsResponse{
		LogsByLevel:  make(map[logging.LogLevel]int64),
		LogsByMethod: make(map[string]int64),
		LogsByStatus: make(map[int]int64),
		TopUsers:     []dto.UserLogCount{},
		TopRoutes:    []dto.RouteLogCount{},
		LogsOverTime: []dto.TimeLogCount{},
		RecentErrors: []dto.ActivityLogResponse{},
	}

	isAdmin := strings.EqualFold(userRoleStr, "admin")
	baseQuery := r.DB.Model(&logging.ActivityLog{})
	if !isAdmin {
		baseQuery = baseQuery.Where("user_id = ?", userID)
	}

	// 1. Total Logs
	if err := baseQuery.Count(&stats.TotalLogs).Error; err != nil {
		return nil, err
	}

	if stats.TotalLogs == 0 {
		return stats, nil
	}

	// 2. Logs by Level
	type LevelCount struct {
		Level logging.LogLevel
		Count int64
	}
	var levelCounts []LevelCount
	if err := baseQuery.Select("level, count(*) as count").Group("level").Scan(&levelCounts).Error; err != nil {
		return nil, err
	}
	for _, lc := range levelCounts {
		stats.LogsByLevel[lc.Level] = lc.Count
	}

	// 3. Logs by Method
	type MethodCount struct {
		Method string
		Count  int64
	}
	var methodCounts []MethodCount
	if err := baseQuery.Select("method, count(*) as count").Group("method").Scan(&methodCounts).Error; err != nil {
		return nil, err
	}
	for _, mc := range methodCounts {
		stats.LogsByMethod[mc.Method] = mc.Count
	}

	// 4. Logs by Status
	type StatusCount struct {
		StatusCode int
		Count      int64
	}
	var statusCounts []StatusCount
	if err := baseQuery.Select("status_code, count(*) as count").Group("status_code").Scan(&statusCounts).Error; err != nil {
		return nil, err
	}
	for _, sc := range statusCounts {
		stats.LogsByStatus[sc.StatusCode] = sc.Count
	}

	// 5. Top Users
	if err := baseQuery.Select("username, count(*) as count").
		Group("username").
		Order("COUNT(*) DESC").
		Limit(5).
		Scan(&stats.TopUsers).Error; err != nil {
		return nil, err
	}

	// 6. Top Routes
	if err := baseQuery.Select("route, method, count(*) as count").
		Group("route, method").
		Order("COUNT(*) DESC").
		Limit(5).
		Scan(&stats.TopRoutes).Error; err != nil {
		return nil, err
	}

	// 7. Average Response Time
	var avgRespTime float64
	if err := baseQuery.Select("AVG(response_time)").Scan(&avgRespTime).Error; err != nil {
		// handle null/error if no logs
	}
	stats.AverageResponseTime = int64(avgRespTime)

	// 8. Error Rate
	var errorCount int64
	if err := baseQuery.Where("status_code >= 400").Count(&errorCount).Error; err != nil {
		return nil, err
	}
	if stats.TotalLogs > 0 {
		stats.ErrorRate = float64(errorCount) / float64(stats.TotalLogs) * 100
	}

	// Fallback/MySQL syntax
	if err := baseQuery.
		Select("DATE_FORMAT(created_at, '%Y-%m-%d %H:00:00') as time, count(*) as count").
		Where("created_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR)").
		Group("time").
		Order("time").
		Scan(&stats.LogsOverTime).Error; err != nil {
		logger.Logger.Error("Failed to get logs over time (mysql/other)", "error", err)
	}

	// 10. Recent Errors
	var recentErrors []logging.ActivityLog
	if err := baseQuery.
		Where("level IN ?", []string{"error", "fatal"}).
		Order("created_at DESC").
		Limit(10).
		Find(&recentErrors).Error; err != nil {
		return nil, err
	}
	stats.RecentErrors = dto.ToActivityLogResponseList(recentErrors)

	return stats, nil
}

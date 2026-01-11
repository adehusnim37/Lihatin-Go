package repositories

import (
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

// GetLogsByUsername retrieves all logs for a specific username
func (r *LoggerRepository) GetLogsByUsername(username string) ([]logging.ActivityLog, error) {
	var logs []logging.ActivityLog

	err := r.DB.Where("username = ?", username).
		Order("createdat DESC").
		Find(&logs).Error

	return logs, err
}

// GetAllLogs retrieves all logs from the database
func (r *LoggerRepository) GetAllLogs() ([]logging.ActivityLog, error) {
	var logs []logging.ActivityLog

	err := r.DB.Order("createdat DESC").
		Find(&logs).Error

	return logs, err
}

func (r *LoggerRepository) GetLogsByShortLink(code string, page, limit int) ([]logging.ActivityLog, int64, error) {
	var logs []logging.ActivityLog
	var totalCount int64

	// Hitung total data
	query := r.DB.Model(&logging.ActivityLog{}).Where("route LIKE ?", "%/"+code)
	if err := query.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit

	// Fetch data dengan pagination
	err := query.
		Select([]string{
			"id", "status_code", "method", "route", "ip_address",
			"user_agent", "browser_info", "timestamp", "response_time",
			"request_headers", "response_body", "route_params",
		}).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&logs).Error

	return logs, totalCount, err
}

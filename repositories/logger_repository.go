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
		log.ID = uuid.New().String()
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

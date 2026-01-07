package jobs

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// CleanupLoginAttemptsJob removes old login attempts from the database
type CleanupLoginAttemptsJob struct {
	db            *gorm.DB
	olderThanDays int
}

// NewCleanupLoginAttemptsJob creates a new instance of the job
func NewCleanupLoginAttemptsJob(db *gorm.DB) *CleanupLoginAttemptsJob {
	return &CleanupLoginAttemptsJob{
		db:            db,
		olderThanDays: 30, // Default: cleanup attempts older than 30 days
	}
}

// WithOlderThanDays sets the number of days after which attempts are considered old
func (j *CleanupLoginAttemptsJob) WithOlderThanDays(days int) *CleanupLoginAttemptsJob {
	j.olderThanDays = days
	return j
}

// Name returns the job name for logging
func (j *CleanupLoginAttemptsJob) Name() string {
	return "cleanup-login-attempts"
}

// Schedule returns when the job should run
// Runs daily at midnight (00:00:00)
func (j *CleanupLoginAttemptsJob) Schedule() string {
	return "0 0 0 * * *" // Every day at 00:00:00
}

// Run executes the job logic
func (j *CleanupLoginAttemptsJob) Run(ctx context.Context) error {
	cutoffDate := time.Now().AddDate(0, 0, -j.olderThanDays)

	result := j.db.WithContext(ctx).
		Table("login_attempts").
		Where("created_at < ?", cutoffDate).
		Delete(nil)

	if result.Error != nil {
		return result.Error
	}

	return nil
}

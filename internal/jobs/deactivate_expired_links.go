package jobs

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
)

// DeactivateExpiredLinksJob deactivates short links that have passed their expiration date
type DeactivateExpiredLinksJob struct {
	db *gorm.DB
}

// NewDeactivateExpiredLinksJob creates a new instance of the job
func NewDeactivateExpiredLinksJob(db *gorm.DB) *DeactivateExpiredLinksJob {
	return &DeactivateExpiredLinksJob{db: db}
}

// Name returns the job name for logging
func (j *DeactivateExpiredLinksJob) Name() string {
	return "deactivate-expired-links"
}

// Schedule returns when the job should run
// Runs every hour at minute 0
func (j *DeactivateExpiredLinksJob) Schedule() string {
	return "0 0 * * * *" // Every hour at :00
}

// Run executes the job logic
func (j *DeactivateExpiredLinksJob) Run(ctx context.Context) error {
	result := j.db.WithContext(ctx).
		Table("short_links").
		Where("expires_at < ? AND is_active = ?", time.Now(), true).
		Update("is_active", false)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected > 0 {
		// Log will be handled by scheduler, but we can add specific info here
		logger.Logger.Info("Deactivated expired links", "count", result.RowsAffected)
	}

	return nil
}

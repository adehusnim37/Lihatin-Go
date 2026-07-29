package jobs

import (
	"context"

	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
	"github.com/adehusnim37/lihatin-go/internal/pkg/notifications"
	"gorm.io/gorm"
)

type WeeklySummaryJob struct {
	service *notifications.WeeklySummaryService
}

func NewWeeklySummaryJob(db *gorm.DB) *WeeklySummaryJob {
	return &WeeklySummaryJob{service: notifications.NewWeeklySummaryService(db)}
}

func (j *WeeklySummaryJob) Name() string {
	return "weekly-email-summaries"
}

// Schedule defaults to Monday retry windows in the process TZ
// (TZ=Asia/Jakarta in the provided deployment configuration). Delivery records
// make the retries idempotent, so a successful summary is still sent only once.
func (j *WeeklySummaryJob) Schedule() string {
	return config.GetEnvOrDefault("WEEKLY_SUMMARY_CRON", "0 0 8,12,16 * * 1")
}

func (j *WeeklySummaryJob) Run(ctx context.Context) error {
	return j.service.SendPreviousWeek(ctx)
}

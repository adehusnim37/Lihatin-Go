package jobs

import (
	"context"

	"github.com/adehusnim37/lihatin-go/internal/pkg/notifications"
	"gorm.io/gorm"
)

type PromotionalCampaignJob struct {
	service *notifications.CampaignService
}

func NewPromotionalCampaignJob(db *gorm.DB) *PromotionalCampaignJob {
	return &PromotionalCampaignJob{service: notifications.NewCampaignService(db)}
}

func (j *PromotionalCampaignJob) Name() string {
	return "promotional-email-campaigns"
}

func (j *PromotionalCampaignJob) Schedule() string {
	return "0 * * * * *"
}

func (j *PromotionalCampaignJob) Run(ctx context.Context) error {
	return j.service.ProcessDue(ctx)
}

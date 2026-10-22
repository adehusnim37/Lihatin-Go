package jobs

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/adehusnim37/lihatin-go/internal/pkg/mail"
	supportmodel "github.com/adehusnim37/lihatin-go/models/support"
	"gorm.io/gorm"
)

type resetNotificationSender interface {
	SendSupportCredentialResetEmail(string, string, string, string, time.Time) error
}

// SupportResetNotificationsJob drains the transactional recovery notification outbox.
// Conditional leases support multiple server instances; failed deliveries retry.
type SupportResetNotificationsJob struct {
	db     *gorm.DB
	sender resetNotificationSender
}

func NewSupportResetNotificationsJob(db *gorm.DB) *SupportResetNotificationsJob {
	return &SupportResetNotificationsJob{db: db, sender: mail.NewEmailService()}
}
func (j *SupportResetNotificationsJob) Name() string     { return "support-reset-security-notifications" }
func (j *SupportResetNotificationsJob) Schedule() string { return "@every 1m" }
func (j *SupportResetNotificationsJob) Run(ctx context.Context) error {
	var pending []supportmodel.CredentialReset
	now := time.Now().UTC()
	if err := j.db.WithContext(ctx).Where("notification_sent_at IS NULL AND next_notification_at <= ?", now).Order("created_at ASC").Limit(50).Find(&pending).Error; err != nil {
		return err
	}
	var deliveryErrors []error
	for _, item := range pending {
		if err := ctx.Err(); err != nil {
			return err
		}
		lease := j.db.WithContext(ctx).Model(&supportmodel.CredentialReset{}).Where("ticket_id = ? AND notification_sent_at IS NULL AND next_notification_at <= ?", item.TicketID, now).Updates(map[string]any{"next_notification_at": now.Add(5 * time.Minute), "notification_attempts": gorm.Expr("notification_attempts + 1")})
		if lease.Error != nil {
			return lease.Error
		}
		if lease.RowsAffected == 0 {
			continue
		}
		if err := j.sender.SendSupportCredentialResetEmail(item.Email, item.Username, item.TicketCode, item.Method, item.CreatedAt); err != nil {
			// Keep pending indefinitely; scheduler logs an actionable failure every retry.
			delay := time.Duration(min(item.NotificationAttempts+1, 60)) * time.Minute
			if updateErr := j.db.WithContext(ctx).Model(&supportmodel.CredentialReset{}).Where("ticket_id = ?", item.TicketID).Update("next_notification_at", time.Now().UTC().Add(delay)).Error; updateErr != nil {
				return updateErr
			}
			deliveryErrors = append(deliveryErrors, fmt.Errorf("recovery notification %s: %w", item.TicketID, err))
			continue
		}
		if err := j.db.WithContext(ctx).Model(&supportmodel.CredentialReset{}).Where("ticket_id = ?", item.TicketID).Update("notification_sent_at", time.Now().UTC()).Error; err != nil {
			return err
		}
	}
	return errors.Join(deliveryErrors...)
}

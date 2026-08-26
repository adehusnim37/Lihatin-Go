package notifications

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
	"github.com/adehusnim37/lihatin-go/internal/pkg/identifier"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/internal/pkg/mail"
	"github.com/adehusnim37/lihatin-go/models/user"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CampaignService struct {
	db          *gorm.DB
	email       *mail.EmailService
	now         func() time.Time
	frontendURL string
	backendURL  string
}

type campaignRecipient struct {
	UserID    string
	Email     string
	FirstName string
	Username  string
}

func NewCampaignService(db *gorm.DB) *CampaignService {
	return &CampaignService{
		db:          db,
		email:       mail.NewEmailService(),
		now:         time.Now,
		frontendURL: strings.TrimRight(config.GetEnvOrDefault(config.EnvFrontendURL, "http://localhost:3000"), "/"),
		backendURL:  strings.TrimRight(config.GetEnvOrDefault(config.EnvBackendURL, "http://localhost:8080"), "/"),
	}
}

func (s *CampaignService) ProcessDue(ctx context.Context) error {
	now := s.now().UTC()
	if err := s.db.WithContext(ctx).
		Model(&user.PromotionalCampaign{}).
		Where("status = ? AND updated_at < ?", user.PromotionalCampaignSending, now.Add(-30*time.Minute)).
		Updates(map[string]any{
			"status":     user.PromotionalCampaignScheduled,
			"updated_at": now,
		}).Error; err != nil {
		return fmt.Errorf("recover interrupted promotional campaigns: %w", err)
	}

	var campaignIDs []string
	if err := s.db.WithContext(ctx).
		Model(&user.PromotionalCampaign{}).
		Where("status = ? AND scheduled_at IS NOT NULL AND scheduled_at <= ?", user.PromotionalCampaignScheduled, now).
		Order("scheduled_at ASC").
		Limit(10).
		Pluck("id", &campaignIDs).Error; err != nil {
		return fmt.Errorf("load due promotional campaigns: %w", err)
	}

	var joined error
	for _, campaignID := range campaignIDs {
		if err := ctx.Err(); err != nil {
			return errors.Join(joined, err)
		}
		if err := s.processCampaign(ctx, campaignID); err != nil {
			logger.Logger.Error("Promotional campaign processing failed",
				"campaign_id", campaignID,
				"error", err.Error(),
			)
			joined = errors.Join(joined, err)
		}
	}
	return joined
}

func (s *CampaignService) processCampaign(ctx context.Context, campaignID string) error {
	startedAt := s.now().UTC()
	claim := s.db.WithContext(ctx).
		Model(&user.PromotionalCampaign{}).
		Where("id = ? AND status = ? AND scheduled_at <= ?", campaignID, user.PromotionalCampaignScheduled, startedAt).
		Updates(map[string]any{
			"status":     user.PromotionalCampaignSending,
			"started_at": startedAt,
			"updated_at": startedAt,
		})
	if claim.Error != nil {
		return fmt.Errorf("claim campaign: %w", claim.Error)
	}
	if claim.RowsAffected == 0 {
		return nil
	}

	var campaign user.PromotionalCampaign
	if err := s.db.WithContext(ctx).Where("id = ?", campaignID).First(&campaign).Error; err != nil {
		return s.failCampaign(ctx, campaignID, err)
	}

	var recipients []campaignRecipient
	if err := s.db.WithContext(ctx).
		Table("users").
		Select("users.id AS user_id, users.email, users.first_name, users.username").
		Joins("JOIN notification_preferences ON notification_preferences.user_id = users.id").
		Joins("JOIN user_auth ON user_auth.user_id = users.id").
		Where("notification_preferences.promotional_email = ?", true).
		Where("users.deleted_at IS NULL").
		Where("user_auth.deleted_at IS NULL AND user_auth.account_status = ? AND user_auth.is_email_verified = ?", user.AccountStatusActive, true).
		Scan(&recipients).Error; err != nil {
		return s.failCampaign(ctx, campaignID, err)
	}

	if err := s.db.WithContext(ctx).
		Model(&user.PromotionalCampaign{}).
		Where("id = ?", campaignID).
		Update("recipient_count", len(recipients)).Error; err != nil {
		return s.failCampaign(ctx, campaignID, err)
	}

	for _, recipient := range recipients {
		if err := ctx.Err(); err != nil {
			return s.requeueInterruptedCampaign(campaignID, err)
		}
		if err := s.db.WithContext(ctx).
			Model(&user.PromotionalCampaign{}).
			Where("id = ? AND status = ?", campaignID, user.PromotionalCampaignSending).
			Update("updated_at", s.now().UTC()).Error; err != nil {
			return s.requeueInterruptedCampaign(campaignID, err)
		}
		if err := s.deliverRecipient(ctx, &campaign, recipient); err != nil {
			logger.Logger.Error("Promotional email delivery failed",
				"campaign_id", campaignID,
				"user_id", recipient.UserID,
				"error", err.Error(),
			)
		}
	}

	return s.completeCampaign(ctx, campaignID)
}

func (s *CampaignService) deliverRecipient(
	ctx context.Context,
	campaign *user.PromotionalCampaign,
	recipient campaignRecipient,
) error {
	delivery := user.PromotionalEmailDelivery{
		ID:         identifier.NewUUIDV7(),
		CampaignID: campaign.ID,
		UserID:     recipient.UserID,
		Email:      recipient.Email,
		Status:     user.PromotionalDeliveryPending,
	}
	result := s.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&delivery)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		if err := s.db.WithContext(ctx).
			Where("campaign_id = ? AND user_id = ?", campaign.ID, recipient.UserID).
			First(&delivery).Error; err != nil {
			return err
		}
		if delivery.Status == user.PromotionalDeliverySent ||
			delivery.Status == user.PromotionalDeliverySkipped {
			return nil
		}
	}

	now := s.now().UTC()
	claim := s.db.WithContext(ctx).
		Model(&user.PromotionalEmailDelivery{}).
		Where("id = ?", delivery.ID).
		Where(
			"status IN ? OR (status = ? AND updated_at < ?)",
			[]user.PromotionalDeliveryStatus{
				user.PromotionalDeliveryPending,
				user.PromotionalDeliveryFailed,
			},
			user.PromotionalDeliverySending,
			now.Add(-time.Hour),
		).
		Updates(map[string]any{
			"status":     user.PromotionalDeliverySending,
			"updated_at": now,
		})
	if claim.Error != nil {
		return claim.Error
	}
	if claim.RowsAffected == 0 {
		return nil
	}

	var optedIn int64
	if err := s.db.WithContext(ctx).
		Model(&user.NotificationPreference{}).
		Where("user_id = ? AND promotional_email = ?", recipient.UserID, true).
		Count(&optedIn).Error; err != nil {
		return s.failDelivery(ctx, delivery.ID, err)
	}
	if optedIn == 0 {
		now = s.now().UTC()
		return s.db.WithContext(ctx).
			Model(&user.PromotionalEmailDelivery{}).
			Where("id = ?", delivery.ID).
			Updates(map[string]any{
				"status":        user.PromotionalDeliverySkipped,
				"error_message": "recipient unsubscribed before delivery",
				"updated_at":    now,
			}).Error
	}

	token, err := GenerateUnsubscribeToken(recipient.UserID, "promotional")
	if err != nil {
		return s.failDelivery(ctx, delivery.ID, err)
	}
	oneClickUnsubscribeURL := s.backendURL + "/v1/notifications/unsubscribe?token=" + url.QueryEscape(token)
	unsubscribeURL := s.frontendURL + "/email-preferences/unsubscribe?token=" + url.QueryEscape(token) + "&category=promotional"
	displayName := strings.TrimSpace(recipient.FirstName)
	if displayName == "" {
		displayName = recipient.Username
	}

	err = s.email.SendPromotionalCampaignEmail(mail.PromotionalCampaignEmailData{
		ToEmail:                recipient.Email,
		UserName:               displayName,
		Subject:                campaign.Subject,
		Preheader:              campaign.Preheader,
		Body:                   campaign.Body,
		ImageURL:               campaign.ImageURL,
		ImageAlt:               campaign.ImageAlt,
		CTALabel:               campaign.CTALabel,
		CTAURL:                 campaign.CTAURL,
		BaseURL:                s.frontendURL,
		PreferencesURL:         s.frontendURL + "/profile/me?tab=notifications",
		UnsubscribeURL:         unsubscribeURL,
		OneClickUnsubscribeURL: oneClickUnsubscribeURL,
	})
	if err != nil {
		return s.failDelivery(ctx, delivery.ID, err)
	}

	now = s.now().UTC()
	return s.db.WithContext(ctx).
		Model(&user.PromotionalEmailDelivery{}).
		Where("id = ?", delivery.ID).
		Updates(map[string]any{
			"status":        user.PromotionalDeliverySent,
			"sent_at":       now,
			"error_message": "",
			"updated_at":    now,
		}).Error
}

func (s *CampaignService) failDelivery(ctx context.Context, deliveryID string, cause error) error {
	now := s.now().UTC()
	_ = s.db.WithContext(ctx).
		Model(&user.PromotionalEmailDelivery{}).
		Where("id = ?", deliveryID).
		Updates(map[string]any{
			"status":        user.PromotionalDeliveryFailed,
			"error_message": cause.Error(),
			"updated_at":    now,
		}).Error
	return cause
}

func (s *CampaignService) completeCampaign(ctx context.Context, campaignID string) error {
	var counts []struct {
		Status string
		Count  int64
	}
	if err := s.db.WithContext(ctx).
		Model(&user.PromotionalEmailDelivery{}).
		Select("status, COUNT(*) AS count").
		Where("campaign_id = ?", campaignID).
		Group("status").
		Scan(&counts).Error; err != nil {
		return s.failCampaign(ctx, campaignID, err)
	}

	var sent, failed int64
	for _, count := range counts {
		switch user.PromotionalDeliveryStatus(count.Status) {
		case user.PromotionalDeliverySent:
			sent = count.Count
		case user.PromotionalDeliveryFailed:
			failed = count.Count
		}
	}

	now := s.now().UTC()
	status := user.PromotionalCampaignCompleted
	var completedAt any = now
	if failed > 0 {
		status = user.PromotionalCampaignFailed
		completedAt = nil
	}
	return s.db.WithContext(ctx).
		Model(&user.PromotionalCampaign{}).
		Where("id = ?", campaignID).
		Updates(map[string]any{
			"status":       status,
			"sent_count":   sent,
			"failed_count": failed,
			"completed_at": completedAt,
			"updated_at":   now,
		}).Error
}

func (s *CampaignService) failCampaign(ctx context.Context, campaignID string, cause error) error {
	now := s.now().UTC()
	_ = s.db.WithContext(ctx).
		Model(&user.PromotionalCampaign{}).
		Where("id = ?", campaignID).
		Updates(map[string]any{
			"status":     user.PromotionalCampaignFailed,
			"updated_at": now,
		}).Error
	return cause
}

func (s *CampaignService) requeueInterruptedCampaign(campaignID string, cause error) error {
	now := s.now().UTC()
	_ = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&user.PromotionalEmailDelivery{}).
			Where("campaign_id = ? AND status = ?", campaignID, user.PromotionalDeliverySending).
			Updates(map[string]any{
				"status":     user.PromotionalDeliveryPending,
				"updated_at": now,
			}).Error; err != nil {
			return err
		}
		return tx.Model(&user.PromotionalCampaign{}).
			Where("id = ? AND status = ?", campaignID, user.PromotionalCampaignSending).
			Updates(map[string]any{
				"status":     user.PromotionalCampaignScheduled,
				"updated_at": now,
			}).Error
	})
	return cause
}

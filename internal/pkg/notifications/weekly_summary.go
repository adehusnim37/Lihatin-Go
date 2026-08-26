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

type WeeklySummaryService struct {
	db          *gorm.DB
	email       *mail.EmailService
	now         func() time.Time
	frontendURL string
	backendURL  string
}

type weeklySummaryRecipient struct {
	UserID    string
	Email     string
	FirstName string
	Username  string
}

type weeklySummaryStats struct {
	LinksCreated     int64
	TotalClicks      int64
	UniqueVisitors   int64
	TopLinkTitle     string
	TopLinkShortCode string
	TopLinkClicks    int64
}

func NewWeeklySummaryService(db *gorm.DB) *WeeklySummaryService {
	return &WeeklySummaryService{
		db:          db,
		email:       mail.NewEmailService(),
		now:         time.Now,
		frontendURL: strings.TrimRight(config.GetEnvOrDefault(config.EnvFrontendURL, "http://localhost:3000"), "/"),
		backendURL:  strings.TrimRight(config.GetEnvOrDefault(config.EnvBackendURL, "http://localhost:8080"), "/"),
	}
}

// PreviousWeekRange returns the previous complete Monday-Sunday interval in the
// supplied time's location. The scheduler uses the process TZ.
func PreviousWeekRange(now time.Time) (time.Time, time.Time) {
	daysSinceMonday := (int(now.Weekday()) + 6) % 7
	currentMonday := time.Date(now.Year(), now.Month(), now.Day()-daysSinceMonday, 0, 0, 0, 0, now.Location())
	return currentMonday.AddDate(0, 0, -7), currentMonday
}

func (s *WeeklySummaryService) SendPreviousWeek(ctx context.Context) error {
	periodStart, periodEnd := PreviousWeekRange(s.now())

	var recipients []weeklySummaryRecipient
	if err := s.db.WithContext(ctx).
		Table("users").
		Select("users.id AS user_id, users.email, users.first_name, users.username").
		Joins("JOIN notification_preferences ON notification_preferences.user_id = users.id").
		Joins("JOIN user_auth ON user_auth.user_id = users.id").
		Where("notification_preferences.weekly_summary_email = ?", true).
		Where("users.deleted_at IS NULL").
		Where("user_auth.deleted_at IS NULL AND user_auth.account_status = ? AND user_auth.is_email_verified = ?", user.AccountStatusActive, true).
		Scan(&recipients).Error; err != nil {
		return fmt.Errorf("load weekly summary recipients: %w", err)
	}

	var joined error
	for _, recipient := range recipients {
		if err := ctx.Err(); err != nil {
			return errors.Join(joined, err)
		}
		if err := s.sendRecipient(ctx, recipient, periodStart, periodEnd); err != nil {
			logger.Logger.Error("Weekly summary delivery failed",
				"user_id", recipient.UserID,
				"period_start", periodStart,
				"error", err.Error(),
			)
			joined = errors.Join(joined, err)
		}
	}
	return joined
}

func (s *WeeklySummaryService) sendRecipient(
	ctx context.Context,
	recipient weeklySummaryRecipient,
	periodStart time.Time,
	periodEnd time.Time,
) error {
	stats, err := s.collectStats(ctx, recipient.UserID, periodStart, periodEnd)
	if err != nil {
		return err
	}

	delivery := user.WeeklySummaryDelivery{
		ID:             identifier.NewUUIDV7(),
		UserID:         recipient.UserID,
		PeriodStart:    periodStart,
		PeriodEnd:      periodEnd,
		Status:         user.WeeklySummaryDeliveryPending,
		LinksCreated:   stats.LinksCreated,
		TotalClicks:    stats.TotalClicks,
		UniqueVisitors: stats.UniqueVisitors,
	}
	result := s.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&delivery)
	if result.Error != nil {
		return fmt.Errorf("create weekly delivery: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		var existing user.WeeklySummaryDelivery
		if err := s.db.WithContext(ctx).
			Where("user_id = ? AND period_start = ?", recipient.UserID, periodStart).
			First(&existing).Error; err != nil {
			return fmt.Errorf("load weekly delivery: %w", err)
		}
		if existing.Status == user.WeeklySummaryDeliverySent ||
			existing.Status == user.WeeklySummaryDeliverySkipped {
			return nil
		}
		delivery = existing
	}

	token, err := GenerateUnsubscribeToken(recipient.UserID, "weekly_summary")
	if err != nil {
		return s.failDelivery(ctx, delivery.ID, err)
	}
	oneClickUnsubscribeURL := s.backendURL + "/v1/notifications/unsubscribe?token=" + url.QueryEscape(token)
	unsubscribeURL := s.frontendURL + "/email-preferences/unsubscribe?token=" + url.QueryEscape(token) + "&category=weekly_summary"
	displayName := strings.TrimSpace(recipient.FirstName)
	if displayName == "" {
		displayName = recipient.Username
	}

	err = s.email.SendWeeklySummaryEmail(mail.WeeklySummaryEmailData{
		ToEmail:                recipient.Email,
		UserName:               displayName,
		PeriodStart:            periodStart,
		PeriodEnd:              periodEnd,
		LinksCreated:           stats.LinksCreated,
		TotalClicks:            stats.TotalClicks,
		UniqueVisitors:         stats.UniqueVisitors,
		TopLinkTitle:           stats.TopLinkTitle,
		TopLinkShortCode:       stats.TopLinkShortCode,
		TopLinkClicks:          stats.TopLinkClicks,
		BaseURL:                s.frontendURL,
		DashboardURL:           s.frontendURL + "/main",
		PreferencesURL:         s.frontendURL + "/profile/me?tab=notifications",
		UnsubscribeURL:         unsubscribeURL,
		OneClickUnsubscribeURL: oneClickUnsubscribeURL,
	})
	if err != nil {
		return s.failDelivery(ctx, delivery.ID, err)
	}

	sentAt := s.now().UTC()
	return s.db.WithContext(ctx).
		Model(&user.WeeklySummaryDelivery{}).
		Where("id = ?", delivery.ID).
		Updates(map[string]any{
			"status":        user.WeeklySummaryDeliverySent,
			"sent_at":       sentAt,
			"error_message": "",
			"updated_at":    sentAt,
		}).Error
}

func (s *WeeklySummaryService) failDelivery(ctx context.Context, deliveryID string, cause error) error {
	now := s.now().UTC()
	_ = s.db.WithContext(ctx).
		Model(&user.WeeklySummaryDelivery{}).
		Where("id = ?", deliveryID).
		Updates(map[string]any{
			"status":        user.WeeklySummaryDeliveryFailed,
			"error_message": cause.Error(),
			"updated_at":    now,
		}).Error
	return cause
}

func (s *WeeklySummaryService) collectStats(
	ctx context.Context,
	userID string,
	periodStart time.Time,
	periodEnd time.Time,
) (weeklySummaryStats, error) {
	var stats weeklySummaryStats
	if err := s.db.WithContext(ctx).
		Table("short_links").
		Where("user_id = ? AND deleted_at IS NULL AND created_at >= ? AND created_at < ?", userID, periodStart, periodEnd).
		Count(&stats.LinksCreated).Error; err != nil {
		return stats, fmt.Errorf("count links created: %w", err)
	}

	clicks := s.db.WithContext(ctx).
		Table("view_link_details").
		Joins("JOIN short_links ON short_links.id = view_link_details.short_link_id").
		Where("short_links.user_id = ? AND short_links.deleted_at IS NULL", userID).
		Where("view_link_details.deleted_at IS NULL AND view_link_details.clicked_at >= ? AND view_link_details.clicked_at < ?", periodStart, periodEnd)
	if err := clicks.Count(&stats.TotalClicks).Error; err != nil {
		return stats, fmt.Errorf("count weekly clicks: %w", err)
	}
	if err := clicks.Distinct("view_link_details.ip_address").Count(&stats.UniqueVisitors).Error; err != nil {
		return stats, fmt.Errorf("count unique weekly visitors: %w", err)
	}

	var top struct {
		Title     string
		ShortCode string
		Clicks    int64
	}
	err := s.db.WithContext(ctx).
		Table("short_links").
		Select("short_links.title, short_links.short_code, COUNT(view_link_details.id) AS clicks").
		Joins("JOIN view_link_details ON view_link_details.short_link_id = short_links.id").
		Where("short_links.user_id = ? AND short_links.deleted_at IS NULL", userID).
		Where("view_link_details.deleted_at IS NULL AND view_link_details.clicked_at >= ? AND view_link_details.clicked_at < ?", periodStart, periodEnd).
		Group("short_links.id, short_links.title, short_links.short_code").
		Order("clicks DESC").
		Limit(1).
		Scan(&top).Error
	if err != nil {
		return stats, fmt.Errorf("load top weekly link: %w", err)
	}
	stats.TopLinkTitle = top.Title
	stats.TopLinkShortCode = top.ShortCode
	stats.TopLinkClicks = top.Clicks
	return stats, nil
}

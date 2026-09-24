package notifications

import (
	"context"
	"errors"
	"time"

	"github.com/adehusnim37/lihatin-go/models/user"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const APIRateLimitAnnouncementID = "api-rate-limits-2026-09"

type InAppAnnouncement struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Body  string `json:"body"`
	Href  string `json:"href"`
}

var apiRateLimitAnnouncement = InAppAnnouncement{
	ID:    APIRateLimitAnnouncementID,
	Title: "API request limits have changed",
	Body:  "If you use API keys, all keys in your account now share one rate limit: 50 requests per hour on Standard or 100 requests per 10 minutes on Premium. Optional total-use caps still apply separately to each key.",
	Href:  "/main/api-integrations/docs",
}

func PendingInAppAnnouncements(ctx context.Context, db *gorm.DB, userID string) ([]InAppAnnouncement, error) {
	var read user.InAppAnnouncementRead
	err := db.WithContext(ctx).Where("user_id = ? AND announcement_id = ?", userID, APIRateLimitAnnouncementID).First(&read).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return []InAppAnnouncement{apiRateLimitAnnouncement}, nil
	}
	if err != nil {
		return nil, err
	}
	return []InAppAnnouncement{}, nil
}

func MarkInAppAnnouncementRead(ctx context.Context, db *gorm.DB, userID, announcementID string) error {
	if announcementID != APIRateLimitAnnouncementID {
		return gorm.ErrRecordNotFound
	}
	return db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&user.InAppAnnouncementRead{
		UserID: userID, AnnouncementID: announcementID, ReadAt: time.Now().UTC(),
	}).Error
}

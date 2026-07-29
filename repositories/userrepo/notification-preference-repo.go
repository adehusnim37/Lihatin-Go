package userrepo

import (
	"errors"
	"strings"
	"time"

	"github.com/adehusnim37/lihatin-go/models/user"
	"gorm.io/gorm"
)

type NotificationPreferenceRepository struct {
	db *gorm.DB
}

func NewNotificationPreferenceRepository(db *gorm.DB) *NotificationPreferenceRepository {
	return &NotificationPreferenceRepository{db: db}
}

func (r *NotificationPreferenceRepository) Get(userID string) (*user.NotificationPreference, error) {
	var preference user.NotificationPreference
	err := r.db.Where("user_id = ?", userID).First(&preference).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &user.NotificationPreference{UserID: userID}, nil
	}
	if err != nil {
		return nil, err
	}
	return &preference, nil
}

func (r *NotificationPreferenceRepository) Update(
	userID string,
	weeklySummaryEmail *bool,
	promotionalEmail *bool,
	source string,
) (*user.NotificationPreference, error) {
	now := time.Now().UTC()
	source = strings.TrimSpace(source)
	if source == "" {
		source = "account_settings"
	}

	err := r.db.Transaction(func(tx *gorm.DB) error {
		var preference user.NotificationPreference
		result := tx.Where("user_id = ?", userID).First(&preference)
		if result.Error != nil {
			if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return result.Error
			}
			preference = user.NotificationPreference{UserID: userID}
			if err := tx.Create(&preference).Error; err != nil {
				return err
			}
		}

		updates := map[string]any{"consent_source": source, "updated_at": now}
		if weeklySummaryEmail != nil {
			updates["weekly_summary_email"] = *weeklySummaryEmail
			if *weeklySummaryEmail {
				updates["weekly_summary_opt_in_at"] = now
				updates["weekly_summary_opt_out_at"] = nil
			} else {
				updates["weekly_summary_opt_out_at"] = now
			}
		}
		if promotionalEmail != nil {
			updates["promotional_email"] = *promotionalEmail
			if *promotionalEmail {
				updates["promotional_opt_in_at"] = now
				updates["promotional_opt_out_at"] = nil
			} else {
				updates["promotional_opt_out_at"] = now
			}
		}

		return tx.Model(&user.NotificationPreference{}).
			Where("user_id = ?", userID).
			Updates(updates).Error
	})
	if err != nil {
		return nil, err
	}

	return r.Get(userID)
}

func (r *NotificationPreferenceRepository) Unsubscribe(userID, category string) error {
	disabled := false
	switch category {
	case "weekly_summary":
		_, err := r.Update(userID, &disabled, nil, "email_unsubscribe")
		return err
	case "promotional":
		_, err := r.Update(userID, nil, &disabled, "email_unsubscribe")
		return err
	default:
		return errors.New("unsupported notification category")
	}
}

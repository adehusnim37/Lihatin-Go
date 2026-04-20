package userrepo

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/models/user"
	"gorm.io/gorm"
)

const (
	// SettingDisposableEmailCheckEnabled controls disposable email enforcement.
	SettingDisposableEmailCheckEnabled = "disposable_email_check_enabled"
)

type SystemSettingRepository interface {
	GetByKey(key string) (*user.SystemSetting, error)
	GetBool(key string, defaultValue bool) (bool, error)
	SetBool(key string, value bool, updatedBy string) error
	EnsureBool(key string, defaultValue bool, updatedBy string) error
}

type systemSettingRepository struct {
	db *gorm.DB
}

func NewSystemSettingRepository(db *gorm.DB) SystemSettingRepository {
	return &systemSettingRepository{db: db}
}

func (r *systemSettingRepository) GetByKey(key string) (*user.SystemSetting, error) {
	var setting user.SystemSetting
	result := r.db.Where("`key` = ?", key).First(&setting)
	if result.Error != nil {
		return nil, result.Error
	}
	return &setting, nil
}

func (r *systemSettingRepository) GetBool(key string, defaultValue bool) (bool, error) {
	setting, err := r.GetByKey(key)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return defaultValue, nil
		}
		logger.Logger.Error("Failed to get system setting", "key", key, "error", err.Error())
		return defaultValue, err
	}

	parsed, parseErr := strconv.ParseBool(strings.TrimSpace(setting.Value))
	if parseErr != nil {
		logger.Logger.Warn("Invalid bool value in system setting, fallback to default",
			"key", key,
			"value", setting.Value,
		)
		return defaultValue, nil
	}

	return parsed, nil
}

func (r *systemSettingRepository) SetBool(key string, value bool, updatedBy string) error {
	valueStr := strconv.FormatBool(value)
	now := time.Now()
	var updater *string
	if strings.TrimSpace(updatedBy) != "" {
		updater = &updatedBy
	}

	return r.db.Transaction(func(tx *gorm.DB) error {
		var setting user.SystemSetting
		result := tx.Where("`key` = ?", key).First(&setting)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				newSetting := user.SystemSetting{
					Key:       key,
					Value:     valueStr,
					UpdatedBy: updater,
					CreatedAt: now,
					UpdatedAt: now,
				}
				return tx.Create(&newSetting).Error
			}
			return result.Error
		}

		updates := map[string]any{
			"value":      valueStr,
			"updated_by": updater,
			"updated_at": now,
		}

		return tx.Model(&user.SystemSetting{}).
			Where("id = ?", setting.ID).
			Updates(updates).Error
	})
}

func (r *systemSettingRepository) EnsureBool(key string, defaultValue bool, updatedBy string) error {
	_, err := r.GetByKey(key)
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return r.SetBool(key, defaultValue, updatedBy)
}

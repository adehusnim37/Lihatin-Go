package userrepo

import (
	"errors"
	"time"

	"github.com/adehusnim37/lihatin-go/dto"
	apperrors "github.com/adehusnim37/lihatin-go/internal/pkg/errors"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/models/user"
	"gorm.io/gorm"
)

type UserPremiumKeyRepository struct {
	db *gorm.DB
}

func NewUserPremiumKeyRepository(db *gorm.DB) *UserPremiumKeyRepository {
	return &UserPremiumKeyRepository{db: db}
}

/**
 * Create a new premium key
 */
func (r *UserPremiumKeyRepository) CreateUserPremiumKey(data *user.PremiumKey) error {
	if data == nil {
		return apperrors.ErrUserInvalidInput
	}

	if err := r.db.Create(data).Error; err != nil {
		logger.Logger.Error("Failed to create premium key", "error", err)
		return apperrors.ErrUserCreationFailed
	}

	return nil
}

/**
 * Create many premium keys
 */
func (r *UserPremiumKeyRepository) CreateManyUserPremiumKeys(data []user.PremiumKey) ([]user.PremiumKey, error) {
	if len(data) == 0 {
		return nil, nil
	}

	created := append([]user.PremiumKey(nil), data...)
	if err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&created).Error; err != nil {
			logger.Logger.Error("Failed to create bulk premium keys", "error", err, "count", len(created))
			return apperrors.ErrUserCreationFailed
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return created, nil
}

/**
 * Get a premium key by code
 */
func (r *UserPremiumKeyRepository) GetUserPremiumKeyByCode(code string) (*user.PremiumKey, error) {
	var userPremiumKey user.PremiumKey
	if err := r.db.Where("code = ?", code).First(&userPremiumKey).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.ErrUserNotFound
		}
		return nil, apperrors.ErrUserFindFailed
	}
	return &userPremiumKey, nil
}

/**
 * Redeem a premium key
 */
func (r *UserPremiumKeyRepository) RedeemPremiumCode(code string, userID string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var premiumKey user.PremiumKey
		if err := tx.Where("code = ?", code).First(&premiumKey).Error; err != nil {
			logger.Logger.Error("Failed to find premium key", "error", err)
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apperrors.ErrPremiumCodeNotFound
			}
			return apperrors.ErrUserFindFailed
		}

		if premiumKey.ValidUntil != nil && premiumKey.ValidUntil.Before(time.Now()) {
			logger.Logger.Error("Premium key expired", "error", "")
			return apperrors.ErrPremiumCodeExpired
		}

		if premiumKey.LimitUsage != nil && premiumKey.UsageCount >= *premiumKey.LimitUsage {
			logger.Logger.Error("Premium key limit reached", "error", "")
			return apperrors.ErrPremiumCodeLimitReached
		}

		var count int64
		if err := tx.Model(&user.PremiumKeyUsage{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return apperrors.ErrPremiumCodeAlreadyUsed
		}

		usage := user.PremiumKeyUsage{
			PremiumKeyID: premiumKey.ID,
			UserID:       userID,
		}
		if err := tx.Create(&usage).Error; err != nil {
			return apperrors.ErrPremiumCodeRedeemFailed
		}

		if err := tx.Model(&premiumKey).Update("usage_count", gorm.Expr("usage_count + 1")).Error; err != nil {
			return apperrors.ErrPremiumCodeRedeemFailed
		}

		return nil
	})
}

/**
 * Get user premium key's all list
 */
func (r *UserPremiumKeyRepository) GetUserPremiumKeyList(page, limit int, sort, orderBy string) (*dto.PaginatedPremiumKeyResponse, error) {
	var userPremiumKeys []user.PremiumKey
	var totalCount int64

	logger.Logger.Info("Get user premium key list", "page", page, "limit", limit, "sort", sort, "orderBy", orderBy)

	if err := r.db.Model(&user.PremiumKey{}).Count(&totalCount).Error; err != nil {
		return nil, apperrors.ErrPremiumCodeCountFailed
	}

	offset := (page - 1) * limit
	orderClause := sort + " " + orderBy
	if sort == "" {
		orderClause = "created_at desc"
	}

	if err := r.db.
		Preload("Usage").
		Offset(offset).
		Limit(limit).
		Order(orderClause).
		Find(&userPremiumKeys).Error; err != nil {
		logger.Logger.Error("Failed to find user premium key list", "error", err)
		return nil, apperrors.ErrUserFindFailed
	}

	items := make([]dto.GeneratePremiumCodeResponse, len(userPremiumKeys))
	for i, key := range userPremiumKeys {

		keyUsage := make([]dto.PremiumKeyUsageResponse, len(key.Usage))
		for j, usage := range key.Usage {
			keyUsage[j] = dto.PremiumKeyUsageResponse{
				ID:           usage.ID,
				PremiumKeyID: usage.PremiumKeyID,
				UserID:       usage.UserID,
				CreatedAt:    usage.CreatedAt,
				UpdatedAt:    usage.UpdatedAt,
			}
		}
		items[i] = dto.GeneratePremiumCodeResponse{
			ID:         key.ID,
			SecretCode: key.Code,
			ValidUntil: key.ValidUntil,
			LimitUsage: key.LimitUsage,
			UsageCount: key.UsageCount,
			CreatedAt:  key.CreatedAt,
			UpdatedAt:  key.UpdatedAt,
			KeyUsage:   keyUsage,
		}
	}

	return &dto.PaginatedPremiumKeyResponse{
		Keys:       items,
		Page:       page,
		Limit:      limit,
		Total:      int(totalCount),
		TotalPages: int(totalCount / int64(limit)),
		Sort:       sort,
		OrderBy:    orderBy,
	}, nil
}

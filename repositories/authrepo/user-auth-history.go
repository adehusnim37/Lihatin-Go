package authrepo

import (
	"errors"
	"time"

	dto "github.com/adehusnim37/lihatin-go/dto"
	apperrors "github.com/adehusnim37/lihatin-go/internal/pkg/errors"
	logger "github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/models/user"
	"gorm.io/gorm"
)

type HistoryUserRepository struct {
	db *gorm.DB
}

// NewHistoryUserRepository creates a new HistoryUser repository
func NewHistoryUserRepository(db *gorm.DB) *HistoryUserRepository {
	return &HistoryUserRepository{db: db}
}

// GetHistoryUsers returns paginated user history with proper DTO
func (r *HistoryUserRepository) GetHistoryUsers(page, limit int, sort, orderBy string) (*dto.UserHistoryPagination, error) {
	var history []user.HistoryUser
	var totalCount int64

	logger.Logger.Info("Get user history list", "page", page, "limit", limit, "sort", sort, "orderBy", orderBy)

	// Count total records
	if err := r.db.Model(&user.HistoryUser{}).Count(&totalCount).Error; err != nil {
		logger.Logger.Error("Failed to count user history", "error", err)
		return nil, apperrors.ErrUserHistoryCountFailed
	}

	// Calculate offset
	offset := (page - 1) * limit

	// Build order clause
	orderClause := sort + " " + orderBy
	if sort == "" {
		orderClause = "changed_at desc"
	}

	// Fetch paginated data
	if err := r.db.
		Offset(offset).
		Limit(limit).
		Order(orderClause).
		Find(&history).Error; err != nil {
		logger.Logger.Error("Failed to find user history list", "error", err)
		return nil, apperrors.ErrUserFindFailed
	}

	// Map to DTO
	items := make([]dto.UserHistoryResponse, len(history))
	for i, h := range history {
		var revokeExpires string
		if h.RevokeExpires != nil {
			revokeExpires = h.RevokeExpires.Format(time.RFC3339)
		}

		var deletedAt string
		if h.DeletedAt.Valid {
			deletedAt = h.DeletedAt.Time.Format(time.RFC3339)
		}

		changedBy := ""
		if h.ChangedBy != nil {
			changedBy = *h.ChangedBy
		}

		ipAddress := ""
		if h.IPAddress != nil {
			ipAddress = *h.IPAddress
		}

		userAgent := ""
		if h.UserAgent != nil {
			userAgent = *h.UserAgent
		}

		items[i] = dto.UserHistoryResponse{
			ID:            int(h.ID),
			UserID:        h.UserID,
			ActionType:    string(h.ActionType),
			OldValue:      string(h.OldValue),
			NewValue:      string(h.NewValue),
			Reason:        h.Reason,
			ChangedBy:     changedBy,
			RevokeToken:   h.RevokeToken,
			RevokeExpires: revokeExpires,
			IPAddress:     ipAddress,
			UserAgent:     userAgent,
			ChangedAt:     h.ChangedAt.Format(time.RFC3339),
			DeletedAt:     deletedAt,
		}
	}

	// Return pagination DTO
	return &dto.UserHistoryPagination{
		Data:       items,
		Page:       page,
		Limit:      limit,
		Sort:       sort,
		OrderBy:    orderBy,
		TotalCount: totalCount,
	}, nil
}

// GetHistoryByID returns a specific history record by ID
func (r *HistoryUserRepository) GetHistoryByID(id int, userID string) (*dto.UserHistoryResponse, error) {
	var history user.HistoryUser

	if err := r.db.First(&history, id).Error; err != nil {
		return nil, apperrors.ErrUserHistoryFindFailed
	}

	if history.UserID != userID {
		return nil, apperrors.ErrUserHistoryFindFailed
	}

	var revokeExpires string
	if history.RevokeExpires != nil {
		revokeExpires = history.RevokeExpires.Format(time.RFC3339)
	}

	var deletedAt string
	if history.DeletedAt.Valid {
		deletedAt = history.DeletedAt.Time.Format(time.RFC3339)
	}

	changedBy := ""
	if history.ChangedBy != nil {
		changedBy = *history.ChangedBy
	}

	ipAddress := ""
	if history.IPAddress != nil {
		ipAddress = *history.IPAddress
	}

	userAgent := ""
	if history.UserAgent != nil {
		userAgent = *history.UserAgent
	}

	return &dto.UserHistoryResponse{
		ID:            int(history.ID),
		UserID:        history.UserID,
		ActionType:    string(history.ActionType),
		OldValue:      string(history.OldValue),
		NewValue:      string(history.NewValue),
		Reason:        history.Reason,
		ChangedBy:     changedBy,
		RevokeToken:   history.RevokeToken,
		RevokeExpires: revokeExpires,
		IPAddress:     ipAddress,
		UserAgent:     userAgent,
		ChangedAt:     history.ChangedAt.Format(time.RFC3339),
		DeletedAt:     deletedAt,
	}, nil
}

// GetEmailChangeHistory returns email change history for a user
func (r *HistoryUserRepository) GetEmailChangeHistory(userID string, days int) ([]user.HistoryUser, error) {
	var history []user.HistoryUser

	cutoffDate := time.Now().Add(-time.Duration(days) * 24 * time.Hour)

	err := r.db.Where(
		"user_id = ? AND action_type IN (?, ?, ?) AND changed_at > ?",
		userID,
		user.ActionEmailChange,
		"email_change_revoked",
		"email_verification",
		cutoffDate,
	).Order("changed_at DESC").Find(&history).Error

	if err != nil {
		return nil, apperrors.ErrUserHistoryFindFailed
	}

	return history, nil
}

// CreateHistoryUser creates a new history record
func (r *HistoryUserRepository) CreateHistoryUser(history *user.HistoryUser) error {
	if err := r.db.Create(history).Error; err != nil {
		return apperrors.ErrUserHistoryCreateFailed
	}
	return nil
}

// CheckEmailChangeEligibility checks if user is eligible to change email
// Returns error if user has changed email or revoked in last 30 days
func (r *HistoryUserRepository) CheckEmailChangeEligibility(userID string) (eligible bool, daysRemaining int, err error) {
	thirtyDaysAgo := time.Now().Add(-30 * 24 * time.Hour)

	var recentHistory user.HistoryUser
	err = r.db.Where(
		"user_id = ? AND action_type IN (?, ?) AND changed_at > ?",
		userID,
		user.ActionEmailChange,
		user.ActionEmailChangeRevoked,
		thirtyDaysAgo,
	).Order("changed_at DESC").First(&recentHistory).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// No recent history, user is eligible
			return true, 0, nil
		}
		return false, 0, apperrors.ErrUserHistoryFindFailed
	}

	// Calculate days remaining until eligible
	daysSinceChange := int(time.Since(recentHistory.ChangedAt).Hours() / 24)
	daysRemaining = 30 - daysSinceChange

	if daysRemaining <= 0 {
		return true, 0, nil
	}

	return false, daysRemaining, nil
}

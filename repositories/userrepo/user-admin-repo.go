package userrepo

import (
	"errors"
	"strings"
	"time"

	apperrors "github.com/adehusnim37/lihatin-go/internal/pkg/errors"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/models/user"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)


// UserAdminRepository defines the methods for admin-related user operations
type UserAdminRepository interface {
	GetAllUsersWithPagination(limit, offset int) ([]user.User, int64, error)
	LockUser(userID, reason string) error
	UnlockUser(userID, reason string) error
	RevokePremiumAccess(userID, reason, revokeType, changedBy, changedByRole string) (*user.User, error)
	ReactivatePremiumAccess(userID, reason, changedBy, changedByRole string, overridePermanent bool) (*user.User, error)
	GetPremiumStatusEvents(userID string, limit int) ([]user.PremiumStatusEvent, error)
	IsUserLocked(userID string) (bool, error)
	DeleteUserPermanent(userID string) error
}

type userAdminRepository struct {
	db *gorm.DB
}

// NewUserAdminRepository creates a new instance of UserAdminRepository
func NewUserAdminRepository(db *gorm.DB) UserAdminRepository {
	return &userAdminRepository{
		db: db,
	}
}

// GetAllUsersWithPagination retrieves all users with pagination (admin only)
func (uar *userAdminRepository) GetAllUsersWithPagination(limit, offset int) ([]user.User, int64, error) {
	var users []user.User
	var totalCount int64

	// Get total count
	countResult := uar.db.Model(&user.User{}).Where("deleted_at IS NULL").Count(&totalCount)
	if countResult.Error != nil {
		logger.Logger.Error("Error getting total user count", "error", countResult.Error)
		return nil, 0, apperrors.ErrUserDatabaseError
	}

	// Get users with pagination
	result := uar.db.Where("deleted_at IS NULL").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&users)

	if result.Error != nil {
		logger.Logger.Error("Error getting paginated users", "error", result.Error)
		return nil, 0, apperrors.ErrUserDatabaseError
	}

	logger.Logger.Info("Retrieved paginated users", "count", len(users), "total", totalCount)
	return users, totalCount, nil
}

// LockUser locks a user account with a reason
func (uar *userAdminRepository) LockUser(userID, reason string) error {
	now := time.Now()
	updates := map[string]interface{}{
		"is_locked":     true,
		"locked_at":     &now,
		"locked_reason": reason,
		"updated_at":    now,
	}

	result := uar.db.Model(&user.User{}).Where("id = ? AND deleted_at IS NULL", userID).Updates(updates)
	if result.Error != nil {
		logger.Logger.Error("Failed to lock user", "user_id", userID, "error", result.Error)
		return apperrors.ErrUserLockFailed
	}

	if result.RowsAffected == 0 {
		return apperrors.ErrUserNotFound
	}

	logger.Logger.Info("User locked successfully", "user_id", userID, "reason", reason)
	return nil
}

// UnlockUser unlocks a user account
func (uar *userAdminRepository) UnlockUser(userID, reason string) error {
	now := time.Now()
	unlockReason := "Account unlocked"
	if reason != "" {
		unlockReason = reason
	}

	updates := map[string]interface{}{
		"is_locked":     false,
		"locked_at":     nil,
		"locked_reason": unlockReason,
		"updated_at":    now,
	}

	result := uar.db.Model(&user.User{}).Where("id = ? AND deleted_at IS NULL", userID).Updates(updates)
	if result.Error != nil {
		logger.Logger.Error("Failed to unlock user", "user_id", userID, "error", result.Error)
		return apperrors.ErrUserUnlockFailed
	}

	if result.RowsAffected == 0 {
		return apperrors.ErrUserNotFound
	}

	logger.Logger.Info("User unlocked successfully", "user_id", userID, "reason", unlockReason)
	return nil
}

// RevokePremiumAccess revokes premium access, demotes role to user, and writes audit event.
func (uar *userAdminRepository) RevokePremiumAccess(userID, reason, revokeType, changedBy, changedByRole string) (*user.User, error) {
	normalizedRevokeType, err := normalizeRevokeType(revokeType)
	if err != nil {
		return nil, err
	}

	normalizedRole := normalizeRole(changedByRole)
	changedBy = strings.TrimSpace(changedBy)
	reason = strings.TrimSpace(reason)

	var updatedUser *user.User
	err = uar.db.Transaction(func(tx *gorm.DB) error {
		var target user.User
		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND deleted_at IS NULL", userID).
			First(&target).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apperrors.ErrUserNotFound
			}
			logger.Logger.Error("Failed to select user for premium revoke", "user_id", userID, "error", err)
			return apperrors.ErrUserDatabaseError
		}

		currentStatus := resolveCurrentPremiumStatus(target.PremiumStatus)
		if currentStatus == string(user.PremiumStatusRevoked) {
			return apperrors.ErrPremiumAlreadyRevoked
		}

		oldRole := normalizeRole(target.Role)
		now := time.Now()
		changedByPtr := nullableString(changedBy)

		updates := map[string]any{
			"is_premium":                 false,
			"premium_status":             string(user.PremiumStatusRevoked),
			"premium_revoke_type":        normalizedRevokeType,
			"premium_revoked_at":         &now,
			"premium_revoked_by":         changedByPtr,
			"premium_revoked_reason":     reason,
			"premium_reactivated_at":     nil,
			"premium_reactivated_by":     nil,
			"premium_reactivated_reason": "",
			"updated_at":                 now,
		}

		if err := tx.Model(&user.User{}).
			Where("id = ? AND deleted_at IS NULL", userID).
			Updates(updates).Error; err != nil {
			logger.Logger.Error("Failed to update user for premium revoke", "user_id", userID, "error", err)
			return apperrors.ErrUserUpdateFailed
		}

		event := user.PremiumStatusEvent{
			UserID:      userID,
			Action:      user.PremiumStatusEventActionRevoke,
			OldStatus:   currentStatus,
			NewStatus:   string(user.PremiumStatusRevoked),
			OldRole:     oldRole,
			NewRole:     "user",
			RevokeType:  normalizedRevokeType,
			Reason:      reason,
			ChangedBy:   changedByPtr,
			ChangedRole: normalizedRole,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := tx.Create(&event).Error; err != nil {
			logger.Logger.Error("Failed to create premium revoke event", "user_id", userID, "error", err)
			return apperrors.ErrUserHistoryCreateFailed
		}

		var refreshed user.User
		if err := tx.Where("id = ? AND deleted_at IS NULL", userID).First(&refreshed).Error; err != nil {
			logger.Logger.Error("Failed to reload user after premium revoke", "user_id", userID, "error", err)
			return apperrors.ErrUserDatabaseError
		}
		updatedUser = &refreshed
		return nil
	})

	if err != nil {
		return nil, err
	}

	return updatedUser, nil
}

// ReactivatePremiumAccess restores premium status and writes audit event.
func (uar *userAdminRepository) ReactivatePremiumAccess(userID, reason, changedBy, changedByRole string, overridePermanent bool) (*user.User, error) {
	normalizedRole := normalizeRole(changedByRole)
	changedBy = strings.TrimSpace(changedBy)
	reason = strings.TrimSpace(reason)

	var updatedUser *user.User
	err := uar.db.Transaction(func(tx *gorm.DB) error {
		var target user.User
		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND deleted_at IS NULL", userID).
			First(&target).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apperrors.ErrUserNotFound
			}
			logger.Logger.Error("Failed to select user for premium reactivate", "user_id", userID, "error", err)
			return apperrors.ErrUserDatabaseError
		}

		currentStatus := resolveCurrentPremiumStatus(target.PremiumStatus)
		if currentStatus != string(user.PremiumStatusRevoked) {
			return apperrors.ErrPremiumNotRevoked
		}

		targetRevokeType := normalizeRevokeTypeOrDefault(target.PremiumRevokeType)
		if targetRevokeType == string(user.PremiumRevokeTypePermanent) {
			if normalizedRole != "admin" {
				return apperrors.ErrPermanentRevokeCannotReactivate
			}
			if !overridePermanent {
				return apperrors.ErrPermanentRevokeOverrideRequired
			}
		}

		oldRole := normalizeRole(target.Role)
		now := time.Now()
		changedByPtr := nullableString(changedBy)

		updates := map[string]any{
			"is_premium":                 true,
			"premium_status":             string(user.PremiumStatusActive),
			"premium_reactivated_at":     &now,
			"premium_reactivated_by":     changedByPtr,
			"premium_reactivated_reason": reason,
			"updated_at":                 now,
			"premium_revoked_at":         nil,
			"premium_revoked_by":         nil,
			"premium_revoked_reason":     "",
			"premium_revoke_type":        nil,
		}

		if err := tx.Model(&user.User{}).
			Where("id = ? AND deleted_at IS NULL", userID).
			Updates(updates).Error; err != nil {
			logger.Logger.Error("Failed to update user for premium reactivate", "user_id", userID, "error", err)
			return apperrors.ErrUserUpdateFailed
		}

		event := user.PremiumStatusEvent{
			UserID:      userID,
			Action:      user.PremiumStatusEventActionReactivate,
			OldStatus:   string(user.PremiumStatusRevoked),
			NewStatus:   string(user.PremiumStatusActive),
			OldRole:     oldRole,
			NewRole:     oldRole,
			RevokeType:  targetRevokeType,
			Reason:      reason,
			ChangedBy:   changedByPtr,
			ChangedRole: normalizedRole,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := tx.Create(&event).Error; err != nil {
			logger.Logger.Error("Failed to create premium reactivate event", "user_id", userID, "error", err)
			return apperrors.ErrUserHistoryCreateFailed
		}

		var refreshed user.User
		if err := tx.Where("id = ? AND deleted_at IS NULL", userID).First(&refreshed).Error; err != nil {
			logger.Logger.Error("Failed to reload user after premium reactivation", "user_id", userID, "error", err)
			return apperrors.ErrUserDatabaseError
		}
		updatedUser = &refreshed
		return nil
	})

	if err != nil {
		return nil, err
	}

	return updatedUser, nil
}

// GetPremiumStatusEvents returns latest premium status events for specific user.
func (uar *userAdminRepository) GetPremiumStatusEvents(userID string, limit int) ([]user.PremiumStatusEvent, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	var events []user.PremiumStatusEvent
	result := uar.db.
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&events)

	if result.Error != nil {
		logger.Logger.Error("Failed to get premium status events", "user_id", userID, "error", result.Error)
		return nil, apperrors.ErrUserDatabaseError
	}

	return events, nil
}

// IsUserLocked checks if a user account is locked
func (uar *userAdminRepository) IsUserLocked(userID string) (bool, error) {
	var user user.User
	result := uar.db.Select("is_locked").Where("id = ? AND deleted_at IS NULL", userID).First(&user)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return false, apperrors.ErrUserNotFound
		}
		logger.Logger.Error("Error checking user lock status", "user_id", userID, "error", result.Error)
		return false, apperrors.ErrUserDatabaseError
	}

	return user.IsLocked, nil
}

// DeleteUserPermanent permanently deletes a user from the database (admin only)
func (uar *userAdminRepository) DeleteUserPermanent(userID string) error {
	var user user.User

	result := uar.db.Where("id = ? AND deleted_at IS NULL", userID).First(&user)
	if result.Error != nil {
		logger.Logger.Error("Failed to find user for permanent deletion", "user_id", userID, "error", result.Error)
		return apperrors.ErrUserNotFound
	}

	result = uar.db.Unscoped().Delete(&user, "id = ?", userID)
	if result.Error != nil {
		logger.Logger.Error("Failed to permanently delete user", "user_id", userID, "error", result.Error)
		return apperrors.ErrUserDeleteFailed
	}

	if result.RowsAffected == 0 {
		return apperrors.ErrUserNotFound
	}

	logger.Logger.Info("User permanently deleted", "user_id", userID)
	return nil
}

func normalizeRevokeType(raw string) (string, error) {
	normalized := normalizeRevokeTypeOrDefault(raw)
	if normalized == string(user.PremiumRevokeTypeTemporary) || normalized == string(user.PremiumRevokeTypePermanent) {
		return normalized, nil
	}
	return "", apperrors.ErrInvalidPremiumRevokeType
}

func normalizeRevokeTypeOrDefault(raw string) string {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if normalized == "" {
		return string(user.PremiumRevokeTypeTemporary)
	}
	return normalized
}

func resolveCurrentPremiumStatus(raw string) string {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if normalized == "" {
		return string(user.PremiumStatusActive)
	}
	return normalized
}

func normalizeRole(raw string) string {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if normalized == "" {
		return "user"
	}
	return normalized
}

func nullableString(raw string) *string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

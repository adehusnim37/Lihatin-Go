package userrepo

import (
	"errors"
	"time"

	apperrors "github.com/adehusnim37/lihatin-go/internal/pkg/errors"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/models/user"
	"gorm.io/gorm"
)

// UserAdminRepository defines the methods for admin-related user operations
type UserAdminRepository interface {
	GetAllUsersWithPagination(limit, offset int) ([]user.User, int64, error)
	LockUser(userID, reason string) error
	UnlockUser(userID, reason string) error
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

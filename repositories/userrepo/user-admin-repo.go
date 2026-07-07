package userrepo

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/adehusnim37/lihatin-go/dto"
	apperrors "github.com/adehusnim37/lihatin-go/internal/pkg/errors"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/models/user"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// UserAdminRepository defines the methods for admin-related user operations
type UserAdminRepository interface {
	GetAllUsersWithPagination(limit, offset int) ([]user.User, int64, error)
	GetUserDetailByID(userID string) (*dto.AdminUserDetailResponse, error)
	LockUser(userID, reason, changedBy string) error
	UnlockUser(userID, reason, changedBy string) error
	RevokePremiumAccess(userID, reason, revokeType, changedBy, changedByRole string) (*user.User, error)
	ReactivatePremiumAccess(userID, reason, changedBy, changedByRole string, overridePermanent bool) (*user.User, error)
	GetPremiumStatusEvents(userID string, limit int) ([]user.PremiumStatusEvent, error)
	IsUserLocked(userID string) (bool, error)
	UpdateUserByAdmin(id string, updateUser dto.AdminUpdateUserRequest) error
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

// GetUserDetailByID retrieves detailed user profile and related admin context.
func (uar *userAdminRepository) GetUserDetailByID(userID string) (*dto.AdminUserDetailResponse, error) {
	var target user.User
	if err := uar.db.Where("id = ? AND deleted_at IS NULL", userID).First(&target).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.ErrUserNotFound
		}
		logger.Logger.Error("Failed to find user detail", "user_id", userID, "error", err)
		return nil, apperrors.ErrUserDatabaseError
	}

	var (
		authMethods    []user.AuthMethod
		recentHistory  []user.HistoryUser
		recentAttempts []user.LoginAttempt
		stats          dto.AdminUserDetailStatsResponse
		userAuth       *user.UserAuth
	)

	var authRecord user.UserAuth
	authErr := uar.db.Where("user_id = ? AND deleted_at IS NULL", userID).First(&authRecord).Error
	switch {
	case authErr == nil:
		userAuth = &authRecord
	case errors.Is(authErr, gorm.ErrRecordNotFound):
		userAuth = nil
	default:
		logger.Logger.Error("Failed to find user_auth detail", "user_id", userID, "error", authErr)
		return nil, apperrors.ErrUserDatabaseError
	}

	if userAuth != nil {
		if err := uar.db.
			Where("user_auth_id = ? AND deleted_at IS NULL", userAuth.ID).
			Order("created_at DESC").
			Find(&authMethods).Error; err != nil {
			logger.Logger.Error("Failed to find auth methods", "user_id", userID, "error", err)
			return nil, apperrors.ErrUserDatabaseError
		}
	}

	since24h := time.Now().Add(-24 * time.Hour)
	since7d := time.Now().Add(-7 * 24 * time.Hour)

	if err := uar.db.Model(&user.APIKey{}).
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Count(&stats.APIKeysTotal).Error; err != nil {
		return nil, apperrors.ErrUserDatabaseError
	}
	if err := uar.db.Model(&user.APIKey{}).
		Where("user_id = ? AND is_active = ? AND deleted_at IS NULL", userID, true).
		Count(&stats.APIKeysActive).Error; err != nil {
		return nil, apperrors.ErrUserDatabaseError
	}
	if err := uar.db.Model(&user.HistoryUser{}).
		Where("user_id = ?", userID).
		Count(&stats.HistoryEventsTotal).Error; err != nil {
		return nil, apperrors.ErrUserDatabaseError
	}
	if err := uar.db.Model(&user.PremiumKeyUsage{}).
		Where("user_id = ?", userID).
		Count(&stats.PremiumKeyUsageTotal).Error; err != nil {
		return nil, apperrors.ErrUserDatabaseError
	}
	if err := uar.db.Model(&user.PremiumStatusEvent{}).
		Where("user_id = ?", userID).
		Count(&stats.PremiumStatusEventsTotal).Error; err != nil {
		return nil, apperrors.ErrUserDatabaseError
	}
	if err := uar.db.Model(&user.LoginAttempt{}).
		Where("(email_or_username = ? OR email_or_username = ?) AND created_at >= ? AND deleted_at IS NULL", target.Username, target.Email, since24h).
		Count(&stats.LoginAttempts24h).Error; err != nil {
		return nil, apperrors.ErrUserDatabaseError
	}
	if err := uar.db.Model(&user.LoginAttempt{}).
		Where("(email_or_username = ? OR email_or_username = ?) AND created_at >= ? AND deleted_at IS NULL", target.Username, target.Email, since7d).
		Count(&stats.LoginAttempts7d).Error; err != nil {
		return nil, apperrors.ErrUserDatabaseError
	}

	if err := uar.db.
		Where("user_id = ?", userID).
		Order("changed_at DESC").
		Limit(10).
		Find(&recentHistory).Error; err != nil {
		return nil, apperrors.ErrUserDatabaseError
	}

	if err := uar.db.
		Where("(email_or_username = ? OR email_or_username = ?) AND deleted_at IS NULL", target.Username, target.Email).
		Order("created_at DESC").
		Limit(10).
		Find(&recentAttempts).Error; err != nil {
		return nil, apperrors.ErrUserDatabaseError
	}

	resp := &dto.AdminUserDetailResponse{
		ID:                       target.ID,
		Username:                 target.Username,
		FirstName:                target.FirstName,
		LastName:                 target.LastName,
		Email:                    target.Email,
		Avatar:                   target.Avatar,
		CreatedAt:                target.CreatedAt,
		UpdatedAt:                target.UpdatedAt,
		DeletedAt:                target.DeletedAt,
		UsernameChanged:          target.UsernameChanged,
		IsPremium:                target.IsPremium,
		IsLocked:                 target.IsLocked,
		LockedAt:                 target.LockedAt,
		LockedReason:             target.LockedReason,
		Role:                     target.Role,
		PremiumStatus:            resolveCurrentPremiumStatus(target.PremiumStatus),
		PremiumRevokeType:        normalizeRevokeTypeOrDefault(target.PremiumRevokeType),
		PremiumRevokedAt:         target.PremiumRevokedAt,
		PremiumRevokedBy:         target.PremiumRevokedBy,
		PremiumRevokedReason:     target.PremiumRevokedReason,
		PremiumReactivatedAt:     target.PremiumReactivatedAt,
		PremiumReactivatedBy:     target.PremiumReactivatedBy,
		PremiumReactivatedReason: target.PremiumReactivatedReason,
		UserAuth:                 toAdminUserAuthDetail(userAuth),
		AuthMethods:              toAdminAuthMethodDetails(authMethods),
		Stats:                    stats,
		RecentHistory:            toAdminRecentHistory(recentHistory),
		RecentLoginAttempts:      toAdminRecentLoginAttempts(recentAttempts),
	}

	return resp, nil
}

// LockUser locks a user account with a reason
func (uar *userAdminRepository) LockUser(userID, reason, changedBy string) error {
	now := time.Now()
	tx := uar.db.Begin()
	if tx.Error != nil {
		logger.Logger.Error("Failed to start lock user transaction", "user_id", userID, "error", tx.Error)
		return apperrors.ErrUserLockFailed
	}

	var target user.User
	if err := tx.Select("id", "is_locked", "locked_at", "locked_reason").Where("id = ? AND deleted_at IS NULL", userID).First(&target).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperrors.ErrUserNotFound
		}
		logger.Logger.Error("Failed to find user before lock", "user_id", userID, "error", err)
		return apperrors.ErrUserLockFailed
	}

	updates := map[string]any{
		"is_locked":     true,
		"locked_at":     &now,
		"locked_reason": reason,
		"updated_at":    now,
	}

	if err := tx.Model(&user.User{}).Where("id = ? AND deleted_at IS NULL", userID).Updates(updates).Error; err != nil {
		tx.Rollback()
		logger.Logger.Error("Failed to lock user", "user_id", userID, "error", err)
		return apperrors.ErrUserLockFailed
	}

	oldValueJSON, _ := json.Marshal(map[string]any{
		"is_locked":     target.IsLocked,
		"locked_at":     target.LockedAt,
		"locked_reason": target.LockedReason,
	})
	newValueJSON, _ := json.Marshal(map[string]any{
		"is_locked":     true,
		"locked_at":     now,
		"locked_reason": reason,
	})

	history := user.HistoryUser{
		UserID:     userID,
		ActionType: user.ActionAccountLock,
		OldValue:   datatypes.JSON(oldValueJSON),
		NewValue:   datatypes.JSON(newValueJSON),
		Reason:     reason,
		ChangedAt:  now,
	}
	if actor := strings.TrimSpace(changedBy); actor != "" {
		history.ChangedBy = &actor
	}

	if err := tx.Create(&history).Error; err != nil {
		tx.Rollback()
		logger.Logger.Error("Failed to write lock history", "user_id", userID, "error", err)
		return apperrors.ErrUserHistoryCreateFailed
	}

	if err := tx.Commit().Error; err != nil {
		logger.Logger.Error("Failed to commit lock user transaction", "user_id", userID, "error", err)
		return apperrors.ErrUserLockFailed
	}

	logger.Logger.Info("User locked successfully", "user_id", userID, "reason", reason)
	return nil
}

// UnlockUser unlocks a user account
func (uar *userAdminRepository) UnlockUser(userID, reason, changedBy string) error {
	now := time.Now()
	unlockReason := "Account unlocked"
	if reason != "" {
		unlockReason = reason
	}
	tx := uar.db.Begin()
	if tx.Error != nil {
		logger.Logger.Error("Failed to start unlock user transaction", "user_id", userID, "error", tx.Error)
		return apperrors.ErrUserUnlockFailed
	}

	var target user.User
	if err := tx.Select("id", "is_locked", "locked_at", "locked_reason").Where("id = ? AND deleted_at IS NULL", userID).First(&target).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperrors.ErrUserNotFound
		}
		logger.Logger.Error("Failed to find user before unlock", "user_id", userID, "error", err)
		return apperrors.ErrUserUnlockFailed
	}

	updates := map[string]any{
		"is_locked":     false,
		"locked_at":     nil,
		"locked_reason": "",
		"updated_at":    now,
	}

	if err := tx.Model(&user.User{}).Where("id = ? AND deleted_at IS NULL", userID).Updates(updates).Error; err != nil {
		tx.Rollback()
		logger.Logger.Error("Failed to unlock user", "user_id", userID, "error", err)
		return apperrors.ErrUserUnlockFailed
	}

	oldValueJSON, _ := json.Marshal(map[string]any{
		"is_locked":     target.IsLocked,
		"locked_at":     target.LockedAt,
		"locked_reason": target.LockedReason,
	})
	newValueJSON, _ := json.Marshal(map[string]any{
		"is_locked":     false,
		"locked_at":     nil,
		"locked_reason": "",
	})

	history := user.HistoryUser{
		UserID:     userID,
		ActionType: user.ActionAccountUnlock,
		OldValue:   datatypes.JSON(oldValueJSON),
		NewValue:   datatypes.JSON(newValueJSON),
		Reason:     unlockReason,
		ChangedAt:  now,
	}
	if actor := strings.TrimSpace(changedBy); actor != "" {
		history.ChangedBy = &actor
	}

	if err := tx.Create(&history).Error; err != nil {
		tx.Rollback()
		logger.Logger.Error("Failed to write unlock history", "user_id", userID, "error", err)
		return apperrors.ErrUserHistoryCreateFailed
	}

	if err := tx.Commit().Error; err != nil {
		logger.Logger.Error("Failed to commit unlock user transaction", "user_id", userID, "error", err)
		return apperrors.ErrUserUnlockFailed
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
			"premium_revoke_type":        nil,
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

func (uar *userAdminRepository) UpdateUserByAdmin(id string, updateUser dto.AdminUpdateUserRequest) error {
	var currentUser user.User
	if err := uar.db.Where("id = ? AND deleted_at IS NULL", id).First(&currentUser).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperrors.ErrUserNotFound
		}
		logger.Logger.Error("Error getting current user for admin update", "user_id", id, "error", err)
		return apperrors.ErrUserFindFailed.WithError(err)
	}

	updates := map[string]interface{}{
		"updated_at": time.Now(),
	}
	updatedFields := 0

	if currentUser.Role == "admin" || currentUser.Role == "super_admin" {
		return apperrors.ErrUserRoleUpdateNotAllowed
	}

	if updateUser.FirstName != nil {
		firstName := strings.TrimSpace(*updateUser.FirstName)
		currentUser.FirstName = firstName
		updates["first_name"] = firstName
		updatedFields++
	}
	if updateUser.LastName != nil {
		lastName := strings.TrimSpace(*updateUser.LastName)
		currentUser.LastName = lastName
		updates["last_name"] = lastName
		updatedFields++
	}
	if updateUser.Username != nil {
		username := strings.TrimSpace(*updateUser.Username)
		currentUser.Username = username
		updates["username"] = username
		updatedFields++
	}
	if updateUser.Email != nil {
		email := strings.ToLower(strings.TrimSpace(*updateUser.Email))
		currentUser.Email = email
		updates["email"] = email
		updatedFields++
	}
	if updateUser.Role != nil {
		// disallow setting role to admin or super_admin via this path
		rawRole := strings.TrimSpace(*updateUser.Role)
		if rawRole == "admin" || rawRole == "super_admin" {
			return apperrors.ErrUserRoleUpdateNotAllowed
		}
		role := strings.ToLower(rawRole)
		currentUser.Role = role
		updates["role"] = role
		updatedFields++
	}

	if updatedFields == 0 {
		logger.Logger.Info("Admin update user skipped due to no changes", "user_id", id)
		return nil
	}

	result := uar.db.Model(&user.User{}).Where("id = ? AND deleted_at IS NULL", id).Updates(updates)
	if result.Error != nil {
		logger.Logger.Error("Failed admin user update", "user_id", id, "error", result.Error)
		errorText := strings.ToLower(result.Error.Error())
		if strings.Contains(errorText, "duplicate entry") {
			if strings.Contains(errorText, "users.email") {
				return apperrors.ErrUserEmailExists.WithError(result.Error)
			}
			if strings.Contains(errorText, "users.username") {
				return apperrors.ErrUserUsernameExists.WithError(result.Error)
			}
			return apperrors.ErrUserDuplicateEntry.WithError(result.Error)
		}
		return apperrors.ErrUserUpdateFailed.WithError(result.Error)
	}

	if result.RowsAffected == 0 {
		return apperrors.ErrUserNotFound
	}

	logger.Logger.Info("Admin user updated successfully", "user_id", id, "fields_updated", updatedFields)
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

	if user.Role == "admin" || user.Role == "super_admin" {
		logger.Logger.Warn("Attempted to permanently delete an admin user", "user_id", userID)
		return apperrors.ErrUserRoleUpdateNotAllowed
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

func toAdminUserAuthDetail(auth *user.UserAuth) *dto.AdminUserAuthDetailResponse {
	if auth == nil {
		return nil
	}
	return &dto.AdminUserAuthDetailResponse{
		ID:                  auth.ID,
		UserID:              auth.UserID,
		IsEmailVerified:     auth.IsEmailVerified,
		PasswordChangedAt:   auth.PasswordChangedAt,
		LastEmailSendAt:     auth.LastEmailSendAt,
		DeviceID:            auth.DeviceID,
		LastIP:              auth.LastIP,
		LastLoginAt:         auth.LastLoginAt,
		LastLogoutAt:        auth.LastLogoutAt,
		FailedLoginAttempts: auth.FailedLoginAttempts,
		LockoutUntil:        auth.LockoutUntil,
		IsActive:            auth.IsActive,
		IsTOTPEnabled:       auth.IsTOTPEnabled,
		CreatedAt:           auth.CreatedAt,
		UpdatedAt:           auth.UpdatedAt,
		DeletedAt:           auth.DeletedAt,
	}
}

func toAdminAuthMethodDetails(methods []user.AuthMethod) []dto.AdminAuthMethodDetailResponse {
	if len(methods) == 0 {
		return []dto.AdminAuthMethodDetailResponse{}
	}
	out := make([]dto.AdminAuthMethodDetailResponse, 0, len(methods))
	for _, method := range methods {
		out = append(out, dto.AdminAuthMethodDetailResponse{
			ID:             method.ID,
			UserAuthID:     method.UserAuthID,
			Type:           string(method.Type),
			IsEnabled:      method.IsEnabled,
			IsVerified:     method.IsVerified,
			VerifiedAt:     method.VerifiedAt,
			LastUsedAt:     method.LastUsedAt,
			FriendlyName:   method.FriendlyName,
			ProviderUserID: method.ProviderUserID,
			DisabledAt:     method.DisabledAt,
			CreatedAt:      method.CreatedAt,
			UpdatedAt:      method.UpdatedAt,
			DeletedAt:      method.DeletedAt,
		})
	}
	return out
}

func toAdminRecentHistory(items []user.HistoryUser) []dto.AdminUserRecentHistoryResponse {
	if len(items) == 0 {
		return []dto.AdminUserRecentHistoryResponse{}
	}
	out := make([]dto.AdminUserRecentHistoryResponse, 0, len(items))
	for _, item := range items {
		out = append(out, dto.AdminUserRecentHistoryResponse{
			ID:         item.ID,
			ActionType: string(item.ActionType),
			Reason:     item.Reason,
			ChangedBy:  item.ChangedBy,
			IPAddress:  item.IPAddress,
			UserAgent:  item.UserAgent,
			ChangedAt:  item.ChangedAt,
		})
	}
	return out
}

func toAdminRecentLoginAttempts(items []user.LoginAttempt) []dto.AdminUserRecentLoginAttemptResponse {
	if len(items) == 0 {
		return []dto.AdminUserRecentLoginAttemptResponse{}
	}
	out := make([]dto.AdminUserRecentLoginAttemptResponse, 0, len(items))
	for _, item := range items {
		out = append(out, dto.AdminUserRecentLoginAttemptResponse{
			ID:              item.ID,
			EmailOrUsername: item.EmailOrUsername,
			IPAddress:       item.IPAddress,
			UserAgent:       item.UserAgent,
			Success:         item.Success,
			FailReason:      item.FailReason,
			CreatedAt:       item.CreatedAt,
		})
	}
	return out
}

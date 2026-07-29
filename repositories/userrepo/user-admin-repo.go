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
	GetAllUsersWithPagination(limit, offset int, filters AdminUserListFilters) ([]user.User, int64, error)
	GetNonPremiumUserEmailsWithPagination(limit, offset int, search, sort, orderBy string) ([]dto.AdminUserEmailResponse, int64, error)
	GetUserDetailByID(userID string) (*dto.AdminUserDetailResponse, error)
	LockUser(userID, reason, changedBy string) error
	UnlockUser(userID, reason, changedBy string) error
	RevokePremiumAccess(userID, reason, revokeType, changedBy, changedByRole string) (*user.User, error)
	ReactivatePremiumAccess(userID, reason, changedBy, changedByRole string, overridePermanent bool) (*user.User, error)
	GetPremiumAccessEvents(userID string, limit int) ([]user.PremiumAccessEvent, error)
	IsUserLocked(userID string) (bool, error)
	UpdateUserByAdmin(id string, updateUser dto.AdminUpdateUserRequest) error
	DeleteUserPermanent(userID string) error
}

type AdminUserListFilters struct {
	Search        string
	Role          string
	PremiumAccessStatus string
	LockStatus    string
	Sort          string
	OrderBy       string
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
func (uar *userAdminRepository) GetAllUsersWithPagination(limit, offset int, filters AdminUserListFilters) ([]user.User, int64, error) {
	var users []user.User
	var totalCount int64

	query := uar.db.Model(&user.User{}).
		Joins("LEFT JOIN user_auth ua ON ua.user_id = users.id AND ua.deleted_at IS NULL").
		Joins("LEFT JOIN user_premium_access pa ON pa.user_id = users.id").
		Where("users.deleted_at IS NULL")

	if filters.Search != "" {
		searchPattern := "%" + strings.ToLower(filters.Search) + "%"
		query = query.Where(
			"(LOWER(username) LIKE ? OR LOWER(email) LIKE ? OR LOWER(first_name) LIKE ? OR LOWER(last_name) LIKE ? OR LOWER(CONCAT(first_name, ' ', last_name)) LIKE ?)",
			searchPattern,
			searchPattern,
			searchPattern,
			searchPattern,
			searchPattern,
		)
	}
	if filters.Role != "" {
		query = query.Where("users.role = ?", filters.Role)
	}
	switch filters.PremiumAccessStatus {
	case "premium":
		query = query.Where("pa.status = ? AND (pa.expires_at IS NULL OR pa.expires_at > ?)", user.PremiumAccessStatusActive, time.Now())
	case "free":
		query = query.Where("pa.user_id IS NULL OR (pa.status = ? AND pa.expires_at IS NOT NULL AND pa.expires_at <= ?)", user.PremiumAccessStatusActive, time.Now())
	case "revoked":
		query = query.Where("pa.status = ?", user.PremiumAccessStatusRevoked)
	}
	switch filters.LockStatus {
	case "locked":
		query = query.Where("ua.account_status = ?", user.AccountStatusLocked)
	case "unlocked":
		query = query.Where("ua.account_status <> ? OR ua.account_status IS NULL", user.AccountStatusLocked)
	}

	if err := query.Count(&totalCount).Error; err != nil {
		logger.Logger.Error("Error getting total user count", "error", err)
		return nil, 0, apperrors.ErrUserDatabaseError
	}

	sortColumn := filters.Sort
	if sortColumn == "" {
		sortColumn = "created_at"
	}
	sortDirection := strings.ToUpper(filters.OrderBy)
	if sortDirection != "ASC" {
		sortDirection = "DESC"
	}

	result := query.
		Preload("UserAuth.AuthMethods").
		Preload("PremiumAccess").
		Order("users." + sortColumn + " " + sortDirection).
		Order("users.id ASC").
		Limit(limit).
		Offset(offset).
		Find(&users)

	if result.Error != nil {
		logger.Logger.Error("Error getting paginated users", "error", result.Error)
		return nil, 0, apperrors.ErrUserDatabaseError
	}
	for i := range users {
		users[i].HydrateDerivedState()
	}

	logger.Logger.Info(
		"Retrieved paginated users",
		"count", len(users),
		"total", totalCount,
		"has_search", filters.Search != "",
		"role", filters.Role,
		"premium_access_status", filters.PremiumAccessStatus,
		"lock_status", filters.LockStatus,
	)
	return users, totalCount, nil
}

// GetNonPremiumUserEmailsWithPagination returns compact, searchable recipient
// options. The premium filter is intentionally enforced here so callers cannot
// accidentally expose premium users as eligible recipients.
func (uar *userAdminRepository) GetNonPremiumUserEmailsWithPagination(limit, offset int, search, sort, orderBy string) ([]dto.AdminUserEmailResponse, int64, error) {
	users := make([]dto.AdminUserEmailResponse, 0)
	var totalCount int64

	query := uar.db.Model(&user.User{}).
		Joins("LEFT JOIN user_auth ua ON ua.user_id = users.id AND ua.deleted_at IS NULL").
		Joins("LEFT JOIN user_premium_access pa ON pa.user_id = users.id").
		Where("users.deleted_at IS NULL").
		Where("ua.account_status = ?", user.AccountStatusActive).
		Where("pa.user_id IS NULL OR (pa.status = ? AND pa.expires_at IS NOT NULL AND pa.expires_at <= ?)", user.PremiumAccessStatusActive, time.Now())

	if search != "" {
		searchPattern := "%" + strings.ToLower(search) + "%"
		query = query.Where(
			"(LOWER(username) LIKE ? OR LOWER(email) LIKE ?)",
			searchPattern,
			searchPattern,
		)
	}

	if err := query.Count(&totalCount).Error; err != nil {
		logger.Logger.Error("Error getting non-premium recipient count", "error", err)
		return nil, 0, apperrors.ErrUserDatabaseError
	}

	sortColumn := "created_at"
	if sort == "updated_at" {
		sortColumn = "updated_at"
	}
	sortDirection := "DESC"
	if strings.EqualFold(orderBy, "asc") {
		sortDirection = "ASC"
	}

	result := query.
		Select("users.id, users.username, users.email").
		Order("users." + sortColumn + " " + sortDirection).
		Order("users.id ASC").
		Limit(limit).
		Offset(offset).
		Scan(&users)

	if result.Error != nil {
		logger.Logger.Error("Error getting non-premium recipients", "error", result.Error)
		return nil, 0, apperrors.ErrUserDatabaseError
	}

	logger.Logger.Info("Retrieved non-premium recipient options", "count", len(users), "total", totalCount)
	return users, totalCount, nil
}

// GetUserDetailByID retrieves detailed user profile and related admin context.
func (uar *userAdminRepository) GetUserDetailByID(userID string) (*dto.AdminUserDetailResponse, error) {
	var target user.User
	if err := uar.db.
		Preload("UserAuth.AuthMethods").
		Preload("PremiumAccess").
		Where("id = ? AND deleted_at IS NULL", userID).
		First(&target).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.ErrUserNotFound
		}
		logger.Logger.Error("Failed to find user detail", "user_id", userID, "error", err)
		return nil, apperrors.ErrUserDatabaseError
	}
	target.HydrateDerivedState()

	var (
		authMethods    []user.AuthMethod
		recentHistory  []user.HistoryUser
		recentAttempts []user.LoginAttempt
		stats          dto.AdminUserDetailStatsResponse
		userAuth       *user.UserAuth
	)

	var authRecord user.UserAuth
	authErr := uar.db.Preload("AuthMethods").Where("user_id = ? AND deleted_at IS NULL", userID).First(&authRecord).Error
	switch {
	case authErr == nil:
		authRecord.HydrateDerivedState()
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
	if err := uar.db.Model(&user.PremiumAccessEvent{}).
		Where("user_id = ?", userID).
		Count(&stats.PremiumAccessEventsTotal).Error; err != nil {
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
		ID:                     target.ID,
		Username:               target.Username,
		FirstName:              target.FirstName,
		LastName:               target.LastName,
		Email:                  target.Email,
		Avatar:                 target.Avatar,
		CreatedAt:              target.CreatedAt,
		UpdatedAt:              target.UpdatedAt,
		DeletedAt:              target.DeletedAt,
		UsernameChanged:        target.UsernameChanged,
		AccountStatus:          accountStatusValue(target.UserAuth),
		AccountStatusChangedAt: accountStatusChangedAt(target.UserAuth),
		AccountStatusReason:    accountStatusReason(target.UserAuth),
		Role:                   target.Role,
		PremiumAccess:          dto.NewPremiumAccessResponse(target.PremiumAccess),
		UserAuth:               toAdminUserAuthDetail(userAuth),
		AuthMethods:            toAdminAuthMethodDetails(authMethods),
		Stats:                  stats,
		RecentHistory:          toAdminRecentHistory(recentHistory),
		RecentLoginAttempts:    toAdminRecentLoginAttempts(recentAttempts),
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

	var target user.UserAuth
	if err := tx.Where("user_id = ? AND deleted_at IS NULL", userID).First(&target).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperrors.ErrUserNotFound
		}
		logger.Logger.Error("Failed to find user before lock", "user_id", userID, "error", err)
		return apperrors.ErrUserLockFailed
	}

	updates := map[string]any{
		"account_status":    user.AccountStatusLocked,
		"status_changed_at": &now,
		"status_reason":     reason,
		"status_changed_by": nullableString(changedBy),
		"updated_at":        now,
	}

	if err := tx.Model(&user.UserAuth{}).Where("user_id = ? AND deleted_at IS NULL", userID).Updates(updates).Error; err != nil {
		tx.Rollback()
		logger.Logger.Error("Failed to lock user", "user_id", userID, "error", err)
		return apperrors.ErrUserLockFailed
	}

	oldValueJSON, _ := json.Marshal(map[string]any{
		"account_status":    target.AccountStatus,
		"status_changed_at": target.StatusChangedAt,
		"status_reason":     target.StatusReason,
	})
	newValueJSON, _ := json.Marshal(map[string]any{
		"account_status":    user.AccountStatusLocked,
		"status_changed_at": now,
		"status_reason":     reason,
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

	var target user.UserAuth
	if err := tx.Where("user_id = ? AND deleted_at IS NULL", userID).First(&target).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperrors.ErrUserNotFound
		}
		logger.Logger.Error("Failed to find user before unlock", "user_id", userID, "error", err)
		return apperrors.ErrUserUnlockFailed
	}

	updates := map[string]any{
		"account_status":    user.AccountStatusActive,
		"status_changed_at": &now,
		"status_reason":     "",
		"status_changed_by": nullableString(changedBy),
		"updated_at":        now,
	}

	if err := tx.Model(&user.UserAuth{}).Where("user_id = ? AND deleted_at IS NULL", userID).Updates(updates).Error; err != nil {
		tx.Rollback()
		logger.Logger.Error("Failed to unlock user", "user_id", userID, "error", err)
		return apperrors.ErrUserUnlockFailed
	}

	oldValueJSON, _ := json.Marshal(map[string]any{
		"account_status":    target.AccountStatus,
		"status_changed_at": target.StatusChangedAt,
		"status_reason":     target.StatusReason,
	})
	newValueJSON, _ := json.Marshal(map[string]any{
		"account_status":    user.AccountStatusActive,
		"status_changed_at": now,
		"status_reason":     "",
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

// RevokePremiumAccess revokes the entitlement without changing authorization
// role, then writes an audit event.
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

		var access user.PremiumAccess
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ?", userID).
			First(&access).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apperrors.ErrPremiumAccessNotFound
			}
			return apperrors.ErrUserDatabaseError
		}
		if access.Status == user.PremiumAccessStatusRevoked {
			return apperrors.ErrPremiumAlreadyRevoked
		}

		now := time.Now()
		changedByPtr := nullableString(changedBy)

		updates := map[string]any{
			"status":            user.PremiumAccessStatusRevoked,
			"revoke_type":       normalizedRevokeType,
			"status_changed_at": &now,
			"status_changed_by": changedByPtr,
			"status_reason":     reason,
			"updated_at":        now,
		}

		if err := tx.Model(&user.PremiumAccess{}).
			Where("user_id = ?", userID).
			Updates(updates).Error; err != nil {
			logger.Logger.Error("Failed to update user for premium revoke", "user_id", userID, "error", err)
			return apperrors.ErrUserUpdateFailed
		}

		event := user.PremiumAccessEvent{
			UserID:     userID,
			Action:     user.PremiumAccessEventActionRevoke,
			OldStatus:  string(access.Status),
			NewStatus:  string(user.PremiumAccessStatusRevoked),
			RevokeType: normalizedRevokeType,
			Reason:     reason,
			ActorID:    changedByPtr,
			ActorRole:  normalizedRole,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		if err := tx.Create(&event).Error; err != nil {
			logger.Logger.Error("Failed to create premium revoke event", "user_id", userID, "error", err)
			return apperrors.ErrUserHistoryCreateFailed
		}

		var refreshed user.User
		if err := tx.Preload("UserAuth.AuthMethods").Preload("PremiumAccess").
			Where("id = ? AND deleted_at IS NULL", userID).First(&refreshed).Error; err != nil {
			logger.Logger.Error("Failed to reload user after premium revoke", "user_id", userID, "error", err)
			return apperrors.ErrUserDatabaseError
		}
		refreshed.HydrateDerivedState()
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

		var access user.PremiumAccess
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ?", userID).
			First(&access).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apperrors.ErrPremiumAccessNotFound
			}
			return apperrors.ErrUserDatabaseError
		}
		if access.Status != user.PremiumAccessStatusRevoked {
			return apperrors.ErrPremiumNotRevoked
		}

		targetRevokeType := normalizeRevokeTypeOrDefault(string(access.RevokeType))
		if targetRevokeType == string(user.PremiumAccessRevokeTypePermanent) {
			if normalizedRole != "admin" {
				return apperrors.ErrPermanentRevokeCannotReactivate
			}
			if !overridePermanent {
				return apperrors.ErrPermanentRevokeOverrideRequired
			}
		}

		now := time.Now()
		changedByPtr := nullableString(changedBy)

		updates := map[string]any{
			"status":            user.PremiumAccessStatusActive,
			"revoke_type":       nil,
			"status_changed_at": &now,
			"status_changed_by": changedByPtr,
			"status_reason":     reason,
			"updated_at":        now,
		}

		if err := tx.Model(&user.PremiumAccess{}).
			Where("user_id = ?", userID).
			Updates(updates).Error; err != nil {
			logger.Logger.Error("Failed to update user for premium reactivate", "user_id", userID, "error", err)
			return apperrors.ErrUserUpdateFailed
		}

		event := user.PremiumAccessEvent{
			UserID:     userID,
			Action:     user.PremiumAccessEventActionReactivate,
			OldStatus:  string(user.PremiumAccessStatusRevoked),
			NewStatus:  string(user.PremiumAccessStatusActive),
			RevokeType: targetRevokeType,
			Reason:     reason,
			ActorID:    changedByPtr,
			ActorRole:  normalizedRole,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		if err := tx.Create(&event).Error; err != nil {
			logger.Logger.Error("Failed to create premium reactivate event", "user_id", userID, "error", err)
			return apperrors.ErrUserHistoryCreateFailed
		}

		var refreshed user.User
		if err := tx.Preload("UserAuth.AuthMethods").Preload("PremiumAccess").
			Where("id = ? AND deleted_at IS NULL", userID).First(&refreshed).Error; err != nil {
			logger.Logger.Error("Failed to reload user after premium reactivation", "user_id", userID, "error", err)
			return apperrors.ErrUserDatabaseError
		}
		refreshed.HydrateDerivedState()
		updatedUser = &refreshed
		return nil
	})

	if err != nil {
		return nil, err
	}

	return updatedUser, nil
}

// GetPremiumAccessEvents returns latest premium access events for a user.
func (uar *userAdminRepository) GetPremiumAccessEvents(userID string, limit int) ([]user.PremiumAccessEvent, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	var events []user.PremiumAccessEvent
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
		return apperrors.ErrUserUpdateNotAllowed
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
			return apperrors.ErrUserUpdateNotAllowed
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
	var auth user.UserAuth
	result := uar.db.Select("account_status").
		Where("user_id = ? AND deleted_at IS NULL", userID).
		First(&auth)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return false, apperrors.ErrUserNotFound
		}
		logger.Logger.Error("Error checking user lock status", "user_id", userID, "error", result.Error)
		return false, apperrors.ErrUserDatabaseError
	}

	return auth.AccountStatus == user.AccountStatusLocked, nil
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
		return apperrors.ErrUserUpdateNotAllowed
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
	if normalized == string(user.PremiumAccessRevokeTypeTemporary) || normalized == string(user.PremiumAccessRevokeTypePermanent) {
		return normalized, nil
	}
	return "", apperrors.ErrInvalidPremiumAccessRevokeType
}

func normalizeRevokeTypeOrDefault(raw string) string {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if normalized == "" {
		return string(user.PremiumAccessRevokeTypeTemporary)
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
		LoginBlockedUntil:   auth.LoginBlockedUntil,
		AccountStatus:       string(auth.AccountStatus),
		TOTPEnabled:         auth.HasEnabledTOTP(),
		CreatedAt:           auth.CreatedAt,
		UpdatedAt:           auth.UpdatedAt,
		DeletedAt:           auth.DeletedAt,
	}
}

func accountStatusValue(auth *user.UserAuth) string {
	if auth == nil {
		return ""
	}
	return string(auth.AccountStatus)
}

func accountStatusChangedAt(auth *user.UserAuth) *time.Time {
	if auth == nil {
		return nil
	}
	return auth.StatusChangedAt
}

func accountStatusReason(auth *user.UserAuth) string {
	if auth == nil {
		return ""
	}
	return auth.StatusReason
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

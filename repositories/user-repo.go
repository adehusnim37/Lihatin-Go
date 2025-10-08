package repositories

import (
	"errors"
	"fmt"
	"time"

	"github.com/adehusnim37/lihatin-go/models/user"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserRepository defines the methods for user-related database operations
type UserRepository interface {
	GetAllUsers() ([]user.User, error)
	GetUserByID(id string) (*user.User, error)
	GetUserByEmailOrUsername(input string) (*user.User, error)
	GetUserForLogin(emailOrUsername string) (*user.User, error)
	CheckPremiumByUsernameOrEmail(inputs string) (*user.User, error)
	CreateUser(user *user.User) error
	UpdateUser(id string, user *user.UpdateUser) error
	DeleteUserPermanent(id string) error
	// Admin methods
	GetAllUsersWithPagination(limit, offset int) ([]user.User, int64, error)
	LockUser(userID, reason string) error
	UnlockUser(userID, reason string) error
	IsUserLocked(userID string) (bool, error)
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{
		db: db,
	}
}

func (ur *userRepository) GetAllUsers() ([]user.User, error) {
	var users []user.User

	result := ur.db.Where("deleted_at IS NULL").Find(&users)
	if result.Error != nil {
		utils.Logger.Error("Failed to get all users", "error", result.Error)
		return nil, utils.ErrUserDatabaseError
	}

	utils.Logger.Info("Successfully retrieved users", "count", len(users))
	return users, nil
}

func (ur *userRepository) GetUserByID(id string) (*user.User, error) {
	utils.Logger.Info("Getting user by ID", "user_id", id)

	var user user.User
	result := ur.db.Where("id = ? AND deleted_at IS NULL", id).First(&user)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			utils.Logger.Warn("User not found", "user_id", id)
			return nil, utils.ErrUserNotFound
		}
		utils.Logger.Error("Database error while getting user", "user_id", id, "error", result.Error)
		return nil, utils.ErrUserDatabaseError
	}

	utils.Logger.Info("User found successfully", "user_id", user.ID, "username", user.Username)
	return &user, nil
}

func (ur *userRepository) CheckPremiumByUsernameOrEmail(inputs string) (*user.User, error) {
	var user user.User
	result := ur.db.Select("is_premium").Where("(username = ? OR email = ?) AND deleted_at IS NULL", inputs, inputs).First(&user)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, utils.ErrUserNotFound
		}
		utils.Logger.Error("Database error while checking premium status", "input", inputs, "error", result.Error)
		return nil, utils.ErrUserDatabaseError
	}

	return &user, nil
}

func (ur *userRepository) GetUserByEmailOrUsername(input string) (*user.User, error) {
	var user user.User
	result := ur.db.Select("id, email, username, is_locked").Where("(email = ? OR username = ?) AND deleted_at IS NULL AND is_locked = false", input, input).First(&user)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, utils.ErrUserNotFound
		}
		utils.Logger.Error("Database error while getting user by email/username", "input", input, "error", result.Error)
		return nil, utils.ErrUserDatabaseError
	}

	return &user, nil
}

func (ur *userRepository) CreateUser(user *user.User) error {
	// Hash the password before storing
	hashedPassword, err := utils.HashPassword(user.Password)
	if err != nil {
		utils.Logger.Error("Error hashing password", "error", err)
		return utils.ErrUserPasswordHashFailed
	}

	// Generate UUID if not provided
	if user.ID == "" {
		newUUID, err := uuid.NewV7()
		if err != nil {
			utils.Logger.Error("Error generating UUID", "error", err)
			return utils.ErrUserCreationFailed
		}
		user.ID = newUUID.String()
	}

	// Set hashed password and timestamps
	user.Password = hashedPassword
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	// Create user using GORM
	result := ur.db.Create(user)
	if result.Error != nil {
		utils.Logger.Error("Failed to create user", "error", result.Error, "email", user.Email)
		// Check for duplicate entry errors
		if fmt.Sprintf("%v", result.Error) == "Error 1062 (23000): Duplicate entry" ||
			result.Error.Error() == "UNIQUE constraint failed: users.email" ||
			result.Error.Error() == "UNIQUE constraint failed: users.username" {
			return utils.ErrUserDuplicateEntry
		}
		return utils.ErrUserCreationFailed
	}

	utils.Logger.Info("User created successfully", "user_id", user.ID, "email", user.Email)
	return nil
}

func (ur *userRepository) UpdateUser(id string, updateUser *user.UpdateUser) error {
	// First, get the current user data to compare
	currentUser, err := ur.GetUserByID(id)
	if err != nil {
		utils.Logger.Error("Error getting current user for update", "user_id", id, "error", err)
		return err
	}

	// Prepare updates map
	updates := make(map[string]interface{})

	// Check each field and only update if different
	if updateUser.FirstName != "" && updateUser.FirstName != currentUser.FirstName {
		updates["first_name"] = updateUser.FirstName
	}

	if updateUser.LastName != "" && updateUser.LastName != currentUser.LastName {
		updates["last_name"] = updateUser.LastName
	}

	// Only update email if it's different from current email
	if updateUser.Email != "" && updateUser.Email != currentUser.Email {
		// Check if the new email already exists for another user
		existingUser, err := ur.GetUserByEmailOrUsername(updateUser.Email)
		if err == nil && existingUser != nil && existingUser.ID != id {
			// Email already exists for another user
			return utils.ErrUserEmailExists
		}
		updates["email"] = updateUser.Email
	}

	if updateUser.Avatar != "" && updateUser.Avatar != currentUser.Avatar {
		updates["avatar"] = updateUser.Avatar
	}

	// Handle password separately since it needs hashing
	if updateUser.Password != "" {
		hashedPassword, err := utils.HashPassword(updateUser.Password)
		if err != nil {
			utils.Logger.Error("Error hashing password during update", "user_id", id, "error", err)
			return utils.ErrUserPasswordHashFailed
		}
		updates["password"] = hashedPassword
	}

	// If no fields to update, return early
	if len(updates) == 0 {
		utils.Logger.Info("No fields to update for user", "user_id", id)
		return nil
	}

	// Always update the updated_at timestamp when there are changes
	updates["updated_at"] = time.Now()

	// Perform the update using GORM
	result := ur.db.Model(&user.User{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		utils.Logger.Error("Failed to update user", "user_id", id, "error", result.Error)
		return utils.ErrUserUpdateFailed
	}

	utils.Logger.Info("User updated successfully", "user_id", id, "fields_updated", len(updates))
	return nil
}

func (ur *userRepository) GetUserForLogin(emailOrUsername string) (*user.User, error) {
	utils.Logger.Info("Attempting user login", "email_or_username", emailOrUsername)

	var user user.User
	result := ur.db.Where("(email = ? OR username = ?) AND deleted_at IS NULL", emailOrUsername, emailOrUsername).First(&user)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			utils.Logger.Warn("No user found for login", "email_or_username", emailOrUsername)
			return nil, utils.ErrUserNotFound
		}
		utils.Logger.Error("Database error during login attempt", "email_or_username", emailOrUsername, "error", result.Error)
		return nil, utils.ErrUserDatabaseError
	}

	utils.Logger.Info("User found for login", "user_id", user.ID, "username", user.Username)
	return &user, nil
}

func (ur *userRepository) DeleteUserPermanent(id string) error {
	result := ur.db.Unscoped().Delete(&user.User{}, "id = ?", id)
	if result.Error != nil {
		utils.Logger.Error("Failed to permanently delete user", "user_id", id, "error", result.Error)
		return utils.ErrUserDeleteFailed
	}

	if result.RowsAffected == 0 {
		return utils.ErrUserNotFound
	}

	utils.Logger.Info("User permanently deleted", "user_id", id)
	return nil
}

// GetAllUsersWithPagination retrieves all users with pagination (admin only)
func (ur *userRepository) GetAllUsersWithPagination(limit, offset int) ([]user.User, int64, error) {
	var users []user.User
	var totalCount int64

	// Get total count
	countResult := ur.db.Model(&user.User{}).Where("deleted_at IS NULL").Count(&totalCount)
	if countResult.Error != nil {
		utils.Logger.Error("Error getting total user count", "error", countResult.Error)
		return nil, 0, utils.ErrUserDatabaseError
	}

	// Get users with pagination
	result := ur.db.Where("deleted_at IS NULL").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&users)

	if result.Error != nil {
		utils.Logger.Error("Error getting paginated users", "error", result.Error)
		return nil, 0, utils.ErrUserDatabaseError
	}

	utils.Logger.Info("Retrieved paginated users", "count", len(users), "total", totalCount)
	return users, totalCount, nil
}

// LockUser locks a user account with a reason
func (ur *userRepository) LockUser(userID, reason string) error {
	now := time.Now()
	updates := map[string]interface{}{
		"is_locked":     true,
		"locked_at":     &now,
		"locked_reason": reason,
		"updated_at":    now,
	}

	result := ur.db.Model(&user.User{}).Where("id = ? AND deleted_at IS NULL", userID).Updates(updates)
	if result.Error != nil {
		utils.Logger.Error("Failed to lock user", "user_id", userID, "error", result.Error)
		return utils.ErrUserLockFailed
	}

	if result.RowsAffected == 0 {
		return utils.ErrUserNotFound
	}

	utils.Logger.Info("User locked successfully", "user_id", userID, "reason", reason)
	return nil
}

// UnlockUser unlocks a user account
func (ur *userRepository) UnlockUser(userID, reason string) error {
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

	result := ur.db.Model(&user.User{}).Where("id = ? AND deleted_at IS NULL", userID).Updates(updates)
	if result.Error != nil {
		utils.Logger.Error("Failed to unlock user", "user_id", userID, "error", result.Error)
		return utils.ErrUserUnlockFailed
	}

	if result.RowsAffected == 0 {
		return utils.ErrUserNotFound
	}

	utils.Logger.Info("User unlocked successfully", "user_id", userID, "reason", unlockReason)
	return nil
}

// IsUserLocked checks if a user account is locked
func (ur *userRepository) IsUserLocked(userID string) (bool, error) {
	var user user.User
	result := ur.db.Select("is_locked").Where("id = ? AND deleted_at IS NULL", userID).First(&user)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return false, utils.ErrUserNotFound
		}
		utils.Logger.Error("Error checking user lock status", "user_id", userID, "error", result.Error)
		return false, utils.ErrUserDatabaseError
	}

	return user.IsLocked, nil
}

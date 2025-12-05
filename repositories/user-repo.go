package repositories

import (
	"errors"
	"fmt"
	"time"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/models/user"
	"github.com/adehusnim37/lihatin-go/internal/pkg/auth"
	apperrors "github.com/adehusnim37/lihatin-go/internal/pkg/errors"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserRepository defines the methods for user-related database operations
type UserRepository interface {
	GetAllUsers() ([]user.User, error)
	GetUserByID(id string) (*user.User, error)
	GetUserByEmailOrUsername(input string) (*user.User, error)
	CheckPremiumByUsernameOrEmail(inputs string) (*user.User, error)
	CreateUser(user *user.User) error
	UpdateUser(id string, user dto.UpdateProfileRequest) error
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
		logger.Logger.Error("Failed to get all users", "error", result.Error)
		return nil, apperrors.ErrUserDatabaseError
	}

	logger.Logger.Info("Successfully retrieved users", "count", len(users))
	return users, nil
}

func (ur *userRepository) GetUserByID(id string) (*user.User, error) {
	logger.Logger.Info("Getting user by ID", "user_id", id)

	var user user.User
	result := ur.db.Where("id = ? AND deleted_at IS NULL", id).First(&user)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			logger.Logger.Warn("User not found", "user_id", id)
			return nil, apperrors.ErrUserNotFound
		}
		logger.Logger.Error("Database error while getting user", "user_id", id, "error", result.Error)
		return nil, apperrors.ErrUserDatabaseError
	}

	logger.Logger.Info("User found successfully", "user_id", user.ID, "username", user.Username)
	return &user, nil
}

func (ur *userRepository) CheckPremiumByUsernameOrEmail(inputs string) (*user.User, error) {
	var user user.User
	result := ur.db.Select("is_premium").Where("(username = ? OR email = ?) AND deleted_at IS NULL", inputs, inputs).First(&user)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, apperrors.ErrUserNotFound
		}
		logger.Logger.Error("Database error while checking premium status", "input", inputs, "error", result.Error)
		return nil, apperrors.ErrUserDatabaseError
	}

	return &user, nil
}

func (ur *userRepository) GetUserByEmailOrUsername(input string) (*user.User, error) {
	var user user.User
	result := ur.db.Select("id, email, username, is_locked").Where("(email = ? OR username = ?) AND deleted_at IS NULL AND is_locked = false", input, input).First(&user)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, apperrors.ErrUserNotFound
		}
		logger.Logger.Error("Database error while getting user by email/username", "input", input, "error", result.Error)
		return nil, apperrors.ErrUserDatabaseError
	}

	return &user, nil
}

func (ur *userRepository) CreateUser(user *user.User) error {
	// Hash the password before storing
	hashedPassword, err := auth.HashPassword(user.Password)
	if err != nil {
		logger.Logger.Error("Error hashing password", "error", err)
		return apperrors.ErrUserPasswordHashFailed
	}

	// Generate UUID if not provided
	if user.ID == "" {
		newUUID, err := uuid.NewV7()
		if err != nil {
			logger.Logger.Error("Error generating UUID", "error", err)
			return apperrors.ErrUserCreationFailed
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
		logger.Logger.Error("Failed to create user", "error", result.Error, "email", user.Email)
		// Check for duplicate entry errors
		if fmt.Sprintf("%v", result.Error) == "Error 1062 (23000): Duplicate entry" ||
			result.Error.Error() == "UNIQUE constraint failed: users.email" ||
			result.Error.Error() == "UNIQUE constraint failed: users.username" {
			return apperrors.ErrUserDuplicateEntry
		}
		return apperrors.ErrUserCreationFailed
	}

	logger.Logger.Info("User created successfully", "user_id", user.ID, "email", user.Email)
	return nil
}

func (ur *userRepository) UpdateUser(id string, updateUser dto.UpdateProfileRequest) error {
	// First, get the current user data to compare
	currentUser, err := ur.GetUserByID(id)
	if err != nil {
		logger.Logger.Error("Error getting current user for update", "user_id", id, "error", err)
		return apperrors.ErrUserNotFound
	}

	if updateUser.FirstName != nil {
		currentUser.FirstName = *updateUser.FirstName
	}
	if updateUser.LastName != nil {
		currentUser.LastName = *updateUser.LastName
	}
	if currentUser.UsernameChanged {
		logger.Logger.Warn("Username change attempt denied", "user_id", id, "current_username", currentUser.Username)
		return apperrors.ErrUsernameChangeNotAllowed
	}
	if updateUser.Username != nil {
		checkUsername := ur.db.Where("username = ? AND id != ? AND deleted_at IS NULL", currentUser.Username, id).First(&user.User{})
		if checkUsername.Error == nil {
			logger.Logger.Warn("Username already taken", "username", currentUser.Username)
			return apperrors.ErrUserDuplicateEntry
		}
		currentUser.Username = *updateUser.Username
	}

	currentUser.UsernameChanged = currentUser.UsernameChanged || (updateUser.Username != nil)

	// Perform the update using GORM
	result := ur.db.Model(&user.User{}).Where("id = ?", id).Updates(currentUser)
	if result.Error != nil {
		logger.Logger.Error("Failed to update user", "user_id", id, "error", result.Error)
		return apperrors.ErrUserUpdateFailed
	}

	updatedFields := 0
	if updateUser.FirstName != nil {
		updatedFields++
	}
	if updateUser.LastName != nil {
		updatedFields++
	}
	if updateUser.Username != nil {
		updatedFields++
	}

	logger.Logger.Info("User updated successfully", "user_id", id, "fields_updated", updatedFields)
	return nil
}

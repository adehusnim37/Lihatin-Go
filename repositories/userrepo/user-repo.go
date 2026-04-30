package userrepo

import (
	"errors"
	"fmt"
	"time"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/internal/pkg/auth"
	apperrors "github.com/adehusnim37/lihatin-go/internal/pkg/errors"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/models/user"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserRepository defines the methods for user-related database operations
type UserRepository interface {
	GetAllUsers() ([]user.User, error)
	GetUserByID(id string) (*user.User, error)
	GetUserByEmail(email string) (*user.User, error)
	GetUserByEmailOrUsername(input string) (*user.User, error)
	CheckPremiumByUsernameOrEmail(inputs string) (bool, error)
	CreateUser(user *user.User) error
	UpdateUser(id string, user dto.UpdateProfileRequest) error
	SetPremium(id string, isPremium bool) error
	CheckUsernameChangeEligibility(userID string) error
	ChangeUsername(userID string, newUsername string) (string, error)
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{
		db: db,
	}
}

func userFindError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return apperrors.ErrUserNotFound.WithError(err)
	}
	return apperrors.ErrUserFindFailed.WithError(err)
}

func (ur *userRepository) GetAllUsers() ([]user.User, error) {
	var users []user.User

	result := ur.db.Where("deleted_at IS NULL").Find(&users)
	if result.Error != nil {
		logger.Logger.Error("Failed to get all users", "error", result.Error)
		return nil, apperrors.ErrUserFindFailed.WithError(result.Error)
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
		}
		logger.Logger.Error("Database error while getting user", "user_id", id, "error", result.Error)
		return nil, userFindError(result.Error)
	}

	logger.Logger.Info("User found successfully", "user_id", user.ID, "username", user.Username)
	return &user, nil
}

func (ur *userRepository) CheckPremiumByUsernameOrEmail(inputs string) (bool, error) {
	var user user.User
	result := ur.db.Select("is_premium").Where("(username = ? OR email = ?) AND deleted_at IS NULL", inputs, inputs).First(&user)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return false, apperrors.ErrUserNotFound.WithError(result.Error)
		}
		logger.Logger.Error("Database error while checking premium status", "input", inputs, "error", result.Error)
		return false, apperrors.ErrUserFindFailed.WithError(result.Error)
	}

	return user.IsPremium, nil
}

func (ur *userRepository) GetUserByEmail(email string) (*user.User, error) {
	var user user.User
	result := ur.db.Where("LOWER(email) = LOWER(?) AND deleted_at IS NULL", email).First(&user)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, apperrors.ErrUserNotFound.WithError(result.Error)
		}
		logger.Logger.Error("Database error while getting user by email", "email", email, "error", result.Error)
		return nil, apperrors.ErrUserFindFailed.WithError(result.Error)
	}

	return &user, nil
}

func (ur *userRepository) GetUserByEmailOrUsername(input string) (*user.User, error) {
	var user user.User
	result := ur.db.Select("id, email, username, is_locked").Where("(email = ? OR username = ?) AND deleted_at IS NULL AND is_locked = false", input, input).First(&user)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, apperrors.ErrUserNotFound.WithError(result.Error)
		}
		logger.Logger.Error("Database error while getting user by email/username", "input", input, "error", result.Error)
		return nil, apperrors.ErrUserFindFailed.WithError(result.Error)
	}

	return &user, nil
}

func (ur *userRepository) CheckUsernameChangeEligibility(userID string) error {
	var user user.User
	result := ur.db.Where("id = ? AND deleted_at IS NULL", userID).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return apperrors.ErrUserNotFound.WithError(result.Error)
		}
		logger.Logger.Error("Database error while checking username change eligibility", "user_id", userID, "error", result.Error)
		return apperrors.ErrUserFindFailed.WithError(result.Error)
	}

	if user.UsernameChanged {
		return apperrors.ErrUsernameChangeNotAllowed
	}

	return nil
}

func (ur *userRepository) ChangeUsername(userID string, newUsername string) (string, error) {
	var existingUser user.User
	var taken int64

	result := ur.db.Where("id = ? AND deleted_at IS NULL", userID).First(&existingUser)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return "", apperrors.ErrUserNotFound.WithError(result.Error)
		}
		logger.Logger.Error("Database error while checking username change eligibility", "user_id", userID, "error", result.Error)
		return "", apperrors.ErrUserFindFailed.WithError(result.Error)
	}

	if existingUser.UsernameChanged {
		return "", apperrors.ErrUsernameChangeNotAllowed
	}

	if newUsername == existingUser.Username {
		return "", apperrors.ErrUserUsernameExists
	}

	err := ur.db.Model(&user.User{}).
		Where("username = ? AND id <> ? AND deleted_at IS NULL", newUsername, userID).
		Count(&taken).Error
	if err != nil {
		logger.Logger.Error("Database error while checking username availability", "username", newUsername, "error", err)
		return "", apperrors.ErrUserFindFailed.WithError(err)
	}

	if taken > 0 {
		return "", apperrors.ErrUserUsernameExists
	}

	oldUsername := existingUser.Username
	existingUser.Username = newUsername
	existingUser.UsernameChanged = true
	existingUser.UpdatedAt = time.Now()

	if err := ur.db.Save(&existingUser).Error; err != nil {
		logger.Logger.Error("Database error while changing username", "user_id", userID, "error", err)
		return "", apperrors.ErrUserUpdateFailed.WithError(err)
	}

	return oldUsername, nil
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
			return apperrors.ErrUserDuplicateEntry.WithError(result.Error)
		}
		return apperrors.ErrUserCreationFailed.WithError(result.Error)
	}

	logger.Logger.Info("User created successfully", "user_id", user.ID, "email", user.Email)
	return nil
}

func (ur *userRepository) UpdateUser(id string, updateUser dto.UpdateProfileRequest) error {
	// First, get the current user data to compare
	currentUser, err := ur.GetUserByID(id)
	if err != nil {
		logger.Logger.Error("Error getting current user for update", "user_id", id, "error", err)
		return err
	}

	if updateUser.FirstName != nil {
		currentUser.FirstName = *updateUser.FirstName
	}
	if updateUser.LastName != nil {
		currentUser.LastName = *updateUser.LastName
	}
	if updateUser.Avatar != nil {
		currentUser.Avatar = *updateUser.Avatar
	}

	// Perform the update using GORM
	result := ur.db.Model(&user.User{}).Where("id = ?", id).Updates(currentUser)
	if result.Error != nil {
		logger.Logger.Error("Failed to update user", "user_id", id, "error", result.Error)
		return apperrors.ErrUserUpdateFailed.WithError(result.Error)
	}

	updatedFields := 0
	if updateUser.FirstName != nil {
		updatedFields++
	}
	if updateUser.LastName != nil {
		updatedFields++
	}
	if updateUser.Avatar != nil {
		updatedFields++
	}

	logger.Logger.Info("User updated successfully", "user_id", id, "fields_updated", updatedFields)
	return nil
}

func (ur *userRepository) SetPremium(id string, isPremium bool) error {
	result := ur.db.Model(&user.User{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("is_premium", isPremium)

	if result.Error != nil {
		logger.Logger.Error("Failed to update premium status", "user_id", id, "error", result.Error)
		return apperrors.ErrUserUpdateFailed.WithError(result.Error)
	}

	if result.RowsAffected == 0 {
		return apperrors.ErrUserNotFound
	}

	logger.Logger.Info("Premium status updated", "user_id", id, "is_premium", isPremium)
	return nil
}

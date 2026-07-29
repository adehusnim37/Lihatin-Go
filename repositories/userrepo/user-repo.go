package userrepo

import (
	"errors"
	"fmt"
	"time"

	"github.com/adehusnim37/lihatin-go/dto"
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
	HasActivePremiumAccessByIdentifier(identifier string) (bool, error)
	CreateUser(account *user.User, accountAuth *user.UserAuth, notificationPreference *user.NotificationPreference, premiumAccess *user.PremiumAccess) error
	UpdateUser(id string, user dto.UpdateProfileRequest) error
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
		logger.Logger.Warn("User not found", "error", err)
		return apperrors.ErrUserNotFound.WithError(err)
	}
	logger.Logger.Error("Database error while finding user", "error", err)
	return apperrors.ErrUserFindFailed.WithError(err)
}

func (ur *userRepository) GetAllUsers() ([]user.User, error) {
	var users []user.User

	result := ur.withAccountState(ur.db).Where("users.deleted_at IS NULL").Find(&users)
	if result.Error != nil {
		logger.Logger.Error("Failed to get all users", "error", result.Error)
		return nil, apperrors.ErrUserFindFailed.WithError(result.Error)
	}
	for i := range users {
		users[i].HydrateDerivedState()
	}

	logger.Logger.Info("Successfully retrieved users", "count", len(users))
	return users, nil
}

func (ur *userRepository) GetUserByID(id string) (*user.User, error) {
	logger.Logger.Info("Getting user by ID", "user_id", id)

	var user user.User
	result := ur.withAccountState(ur.db).Where("users.id = ? AND users.deleted_at IS NULL", id).First(&user)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			logger.Logger.Warn("User not found", "user_id", id)
		}
		logger.Logger.Error("Database error while getting user", "user_id", id, "error", result.Error)
		return nil, userFindError(result.Error)
	}
	user.HydrateDerivedState()

	logger.Logger.Info("User found successfully", "user_id", user.ID, "username", user.Username)
	return &user, nil
}

func (ur *userRepository) HasActivePremiumAccessByIdentifier(identifier string) (bool, error) {
	var account user.User
	result := ur.withAccountState(ur.db).
		Where("(users.username = ? OR users.email = ?) AND users.deleted_at IS NULL", identifier, identifier).
		First(&account)

	if result.Error != nil {
		return false, userFindError(result.Error)
	}
	account.HydrateDerivedState()
	return account.HasPremiumAccessAt(time.Now()), nil
}

func (ur *userRepository) GetUserByEmail(email string) (*user.User, error) {
	var user user.User
	result := ur.withAccountState(ur.db).Where("LOWER(users.email) = LOWER(?) AND users.deleted_at IS NULL", email).First(&user)

	if result.Error != nil {
		return nil, userFindError(result.Error)
	}
	user.HydrateDerivedState()

	return &user, nil
}

func (ur *userRepository) GetUserByEmailOrUsername(input string) (*user.User, error) {
	var user user.User
	result := ur.withAccountState(ur.db).
		Where("(users.email = ? OR users.username = ?) AND users.deleted_at IS NULL", input, input).
		First(&user)

	if result.Error != nil {
		return nil, userFindError(result.Error)
	}
	user.HydrateDerivedState()

	return &user, nil
}

func (ur *userRepository) CheckUsernameChangeEligibility(userID string) error {
	var user user.User
	result := ur.db.Where("id = ? AND deleted_at IS NULL", userID).First(&user)
	if result.Error != nil {
		return userFindError(result.Error)
	}

	if user.Role == "admin" {
		return apperrors.ErrUserUpdateNotAllowed
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
		return "", userFindError(result.Error)
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

func (ur *userRepository) CreateUser(account *user.User, accountAuth *user.UserAuth, notificationPreference *user.NotificationPreference, premiumAccess *user.PremiumAccess) error {
	// Generate UUID if not provided
	if account.ID == "" {
		newUUID, err := uuid.NewV7()
		if err != nil {
			logger.Logger.Error("Error generating UUID", "error", err)
			return apperrors.ErrUserCreationFailed
		}
		account.ID = newUUID.String()
	}

	now := time.Now()
	account.CreatedAt = now
	account.UpdatedAt = now

	if accountAuth == nil {
		return apperrors.ErrUserCreationFailed
	}
	accountAuth.UserID = account.ID
	accountAuth.CreatedAt = now
	accountAuth.UpdatedAt = now
	if accountAuth.AccountStatus == "" {
		accountAuth.AccountStatus = user.AccountStatusActive
	}

	if notificationPreference != nil {
		notificationPreference.UserID = account.ID
		notificationPreference.CreatedAt = now
		notificationPreference.UpdatedAt = now

		if notificationPreference.WeeklySummaryEmail {
			notificationPreference.WeeklySummaryOptInAt = &now
			notificationPreference.WeeklySummaryOptOutAt = nil
		} else {
			notificationPreference.WeeklySummaryOptInAt = nil
			notificationPreference.WeeklySummaryOptOutAt = &now
		}

		if notificationPreference.PromotionalEmail {
			notificationPreference.PromotionalOptInAt = &now
			notificationPreference.PromotionalOptOutAt = nil
		} else {
			notificationPreference.PromotionalOptInAt = nil
			notificationPreference.PromotionalOptOutAt = &now
		}
	}

	if premiumAccess != nil {
		premiumAccess.UserID = account.ID
		if premiumAccess.Status == "" {
			premiumAccess.Status = user.PremiumAccessStatusActive
		}
		if premiumAccess.Tier == "" {
			premiumAccess.Tier = "premium"
		}
		if premiumAccess.Source == "" {
			premiumAccess.Source = "system"
		}
		if premiumAccess.GrantedAt.IsZero() {
			premiumAccess.GrantedAt = now
		}
		premiumAccess.CreatedAt = now
		premiumAccess.UpdatedAt = now
	}

	err := ur.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(account).Error; err != nil {
			return err
		}
		if err := tx.Create(accountAuth).Error; err != nil {
			return err
		}
		if notificationPreference != nil {
			if err := tx.Create(notificationPreference).Error; err != nil {
				return err
			}
		}
		if premiumAccess != nil {
			if err := tx.Create(premiumAccess).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		logger.Logger.Error("Failed to create user", "error", err, "email", account.Email)
		// Check for duplicate entry errors
		if fmt.Sprintf("%v", err) == "Error 1062 (23000): Duplicate entry" ||
			err.Error() == "UNIQUE constraint failed: users.email" ||
			err.Error() == "UNIQUE constraint failed: users.username" {
			return apperrors.ErrUserDuplicateEntry.WithError(err)
		}
		return apperrors.ErrUserCreationFailed.WithError(err)
	}

	account.UserAuth = accountAuth
	account.PremiumAccess = premiumAccess
	account.HydrateDerivedState()
	logger.Logger.Info("User created successfully", "user_id", account.ID, "email", account.Email)
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

func (ur *userRepository) withAccountState(db *gorm.DB) *gorm.DB {
	return db.
		Preload("UserAuth.AuthMethods").
		Preload("PremiumAccess")
}

package authrepo

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/adehusnim37/lihatin-go/internal/pkg/errors"
	"github.com/adehusnim37/lihatin-go/models/user"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AuthMethodRepository handles AuthMethod database operations
type AuthMethodRepository struct {
	db *gorm.DB
}

// NewAuthMethodRepository creates a new AuthMethod repository
func NewAuthMethodRepository(db *gorm.DB) *AuthMethodRepository {
	return &AuthMethodRepository{db: db}
}

// CreateAuthMethod creates a new auth method
func (r *AuthMethodRepository) CreateAuthMethod(authMethod *user.AuthMethod) error {
	if err := r.db.Create(authMethod).Error; err != nil {
		return errors.ErrAuthMethodCreateFailed
	}
	return nil
}

// GetAuthMethodByID retrieves auth method by ID
func (r *AuthMethodRepository) GetAuthMethodByID(id string) (*user.AuthMethod, error) {
	var authMethod user.AuthMethod
	if err := r.db.First(&authMethod, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, sql.ErrNoRows
		}
		return nil, errors.ErrAuthMethodFindFailed
	}
	return &authMethod, nil
}

// GetAuthMethodsByUserAuthID retrieves all auth methods for a user
func (r *AuthMethodRepository) GetAuthMethodsByUserAuthID(userAuthID string) ([]user.AuthMethod, error) {
	var authMethods []user.AuthMethod
	if err := r.db.Where("user_auth_id = ?", userAuthID).Find(&authMethods).Error; err != nil {
		return nil, errors.ErrAuthMethodFindFailed
	}
	return authMethods, nil
}

// GetAuthMethodByType retrieves auth method by user and type
func (r *AuthMethodRepository) GetAuthMethodByType(userAuthID string, methodType user.AuthMethodType) (*user.AuthMethod, error) {
	var authMethod user.AuthMethod
	if err := r.db.Where("user_auth_id = ? AND type = ?", userAuthID, methodType).First(&authMethod).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, sql.ErrNoRows
		}
		return nil, errors.ErrAuthMethodFindFailed
	}
	return &authMethod, nil
}

// UpdateAuthMethod updates an auth method
func (r *AuthMethodRepository) UpdateAuthMethod(authMethod *user.AuthMethod) error {
	if err := r.db.Save(authMethod).Error; err != nil {
		return errors.ErrAuthMethodUpdateFailed
	}
	return nil
}

// EnableAuthMethod enables an auth method
func (r *AuthMethodRepository) EnableAuthMethod(id string) error {
	err := r.db.Model(&user.AuthMethod{}).
		Where("id = ?", id).
		Update("is_enabled", true).Error

	if err != nil {
		return errors.ErrAuthMethodUpdateFailed
	}
	return nil
}

// DisableAuthMethod disables an auth method
func (r *AuthMethodRepository) DisableAuthMethod(id string) error {
	now := time.Now()
	if err := r.db.Model(&user.AuthMethod{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"is_enabled":  false,
			"disabled_at": &now,
		}).Error; err != nil {
		return errors.ErrAuthMethodUpdateFailed
	}

	if err := r.db.Model(&user.UserAuth{}).
		Where("id = (SELECT user_auth_id FROM auth_methods WHERE id = ?)", id).
		Update("is_totp_enabled", false).Error; err != nil {
		return errors.ErrUserAuthUpdateFailed
	}

	return nil
}

// VerifyAuthMethod marks an auth method as verified
func (r *AuthMethodRepository) VerifyAuthMethod(id string) error {
	now := time.Now()
	if err := r.db.Model(&user.AuthMethod{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"is_verified": true,
			"verified_at": &now,
		}).Error; err != nil {
		return errors.ErrAuthMethodUpdateFailed
	}

	if err := r.db.Model(&user.UserAuth{}).
		Where("id = (SELECT user_auth_id FROM auth_methods WHERE id = ?)", id).
		Update("is_totp_enabled", true).Error; err != nil {
		return errors.ErrUserAuthUpdateFailed
	}

	return nil
}

// UpdateLastUsed updates the last used timestamp for an auth method
func (r *AuthMethodRepository) UpdateLastUsed(id string) error {
	now := time.Now()
	err := r.db.Model(&user.AuthMethod{}).
		Where("id = ?", id).
		Update("last_used_at", &now).Error

	if err != nil {
		return errors.ErrAuthMethodUpdateFailed
	}
	return nil
}

// UpdateTOTPLastUsed updates the last used timestamp for TOTP auth method by userAuthID
func (r *AuthMethodRepository) UpdateTOTPLastUsed(userAuthID string) error {
	now := time.Now()
	err := r.db.Model(&user.AuthMethod{}).
		Where("user_auth_id = ? AND type = ? AND is_enabled = ?", userAuthID, user.AuthMethodTypeTOTP, true).
		Update("last_used_at", &now).Error

	if err != nil {
		return errors.ErrAuthMethodUpdateFailed
	}
	return nil
}

// DeleteAuthMethod soft deletes an auth method
func (r *AuthMethodRepository) DeleteAuthMethod(id string) error {
	err := r.db.Where("id = ?", id).Delete(&user.AuthMethod{}).Error
	if err != nil {
		return errors.ErrAuthMethodDeleteFailed
	}
	return nil
}

// GetEnabledAuthMethods retrieves all enabled auth methods for a user
func (r *AuthMethodRepository) GetEnabledAuthMethods(userAuthID string) ([]user.AuthMethod, error) {
	var authMethods []user.AuthMethod
	if err := r.db.Where("user_auth_id = ? AND is_enabled = ?", userAuthID, true).Find(&authMethods).Error; err != nil {
		return nil, errors.ErrAuthMethodFindFailed
	}
	return authMethods, nil
}

// GetVerifiedAuthMethods retrieves all verified auth methods for a user
func (r *AuthMethodRepository) GetVerifiedAuthMethods(userAuthID string) ([]user.AuthMethod, error) {
	var authMethods []user.AuthMethod
	if err := r.db.Where("user_auth_id = ? AND is_verified = ?", userAuthID, true).Find(&authMethods).Error; err != nil {
		return nil, errors.ErrAuthMethodFindFailed
	}
	return authMethods, nil
}

// HasTOTPEnabled checks if user has TOTP enabled
func (r *AuthMethodRepository) HasTOTPEnabled(userAuthID string) (bool, error) {
	var count int64
	err := r.db.Model(&user.AuthMethod{}).
		Where("user_auth_id = ? AND type = ? AND is_enabled = ? AND is_verified = ?",
			userAuthID, user.AuthMethodTypeTOTP, true, true).
		Count(&count).Error

	if err != nil {
		return false, errors.ErrAuthMethodFindFailed
	}

	return count > 0, nil
}

// GetTOTPSecret retrieves TOTP secret for a user
func (r *AuthMethodRepository) GetTOTPSecret(userAuthID string) (string, error) {
	var authMethod user.AuthMethod
	err := r.db.Where("user_auth_id = ? AND type = ? AND is_enabled = ?",
		userAuthID, user.AuthMethodTypeTOTP, true).First(&authMethod).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", sql.ErrNoRows
		}
		return "", errors.ErrAuthMethodFindFailed
	}

	return authMethod.Secret, nil
}

// GetRecoveryCodes retrieves recovery codes for a user's TOTP
func (r *AuthMethodRepository) GetRecoveryCodes(userAuthID string) ([]string, error) {
	var authMethod user.AuthMethod
	err := r.db.Where("user_auth_id = ? AND type = ? AND is_enabled = ?",
		userAuthID, user.AuthMethodTypeTOTP, true).First(&authMethod).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("TOTP not enabled for user")
		}
		return nil, fmt.Errorf("failed to get recovery codes: %w", err)
	}

	// Deserialize recovery codes from JSON
	var recoveryCodes []string
	if err := json.Unmarshal(authMethod.RecoveryCodes, &recoveryCodes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal recovery codes: %w", err)
	}

	return recoveryCodes, nil
}

// UpdateRecoveryCodes updates recovery codes for TOTP
func (r *AuthMethodRepository) UpdateRecoveryCodes(userAuthID string, recoveryCodes []string) error {
	// Serialize recovery codes to JSON
	recoveryCodesJSON, err := json.Marshal(recoveryCodes)
	if err != nil {
		return fmt.Errorf("failed to marshal recovery codes: %w", err)
	}

	err = r.db.Model(&user.AuthMethod{}).
		Where("user_auth_id = ? AND type = ?", userAuthID, user.AuthMethodTypeTOTP).
		Update("recovery_codes", recoveryCodesJSON).Error

	if err != nil {
		return fmt.Errorf("failed to update recovery codes: %w", err)
	}
	return nil
}

// SetupTOTP creates or updates TOTP auth method
func (r *AuthMethodRepository) SetupTOTP(userAuthID, encryptedSecret string, recoveryCodes []string, friendlyName string) error {
	// Serialize recovery codes to JSON
	recoveryCodesJSON, err := json.Marshal(recoveryCodes)
	if err != nil {
		return fmt.Errorf("failed to marshal recovery codes: %w", err)
	}

	// Check if TOTP already exists
	var existing user.AuthMethod
	err = r.db.Where("user_auth_id = ? AND type = ?", userAuthID, user.AuthMethodTypeTOTP).First(&existing).Error

	uuidV7, _ := uuid.NewV7()

	if err == gorm.ErrRecordNotFound {

		// Create new TOTP method
		authMethod := &user.AuthMethod{
			ID:            uuidV7.String(),
			UserAuthID:    userAuthID,
			Type:          user.AuthMethodTypeTOTP,
			Secret:        encryptedSecret,
			RecoveryCodes: recoveryCodesJSON,
			FriendlyName:  friendlyName,
			IsEnabled:     true,
			IsVerified:    false,
		}
		return r.CreateAuthMethod(authMethod)
	} else if err != nil {
		return fmt.Errorf("failed to check existing TOTP: %w", err)
	}

	// Update existing TOTP method
	existing.Secret = encryptedSecret
	existing.RecoveryCodes = recoveryCodesJSON
	existing.FriendlyName = friendlyName
	existing.IsEnabled = true
	existing.IsVerified = false

	return r.UpdateAuthMethod(&existing)
}

// GetAuthMethodStats retrieves authentication method statistics
func (r *AuthMethodRepository) GetAuthMethodStats() (map[string]interface{}, error) {
	stats := make(map[string]any)

	// Count by type
	var results []struct {
		Type  user.AuthMethodType
		Count int64
	}

	err := r.db.Model(&user.AuthMethod{}).
		Select("type, COUNT(*) as count").
		Where("is_enabled = ?", true).
		Group("type").
		Scan(&results).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get auth method stats: %w", err)
	}

	typeStats := make(map[string]int64)
	for _, result := range results {
		typeStats[string(result.Type)] = result.Count
	}
	stats["by_type"] = typeStats

	// Total enabled methods
	var totalEnabled int64
	if err := r.db.Model(&user.AuthMethod{}).Where("is_enabled = ?", true).Count(&totalEnabled).Error; err != nil {
		return nil, fmt.Errorf("failed to count enabled methods: %w", err)
	}
	stats["total_enabled"] = totalEnabled

	// Total verified methods
	var totalVerified int64
	if err := r.db.Model(&user.AuthMethod{}).Where("is_verified = ?", true).Count(&totalVerified).Error; err != nil {
		return nil, fmt.Errorf("failed to count verified methods: %w", err)
	}
	stats["total_verified"] = totalVerified

	return stats, nil
}

// GetUserAuthMethodsSummary retrieves summary of auth methods for a user
func (r *AuthMethodRepository) GetUserAuthMethodsSummary(userAuthID string) (map[string]any, error) {
	var authMethods []user.AuthMethod
	if err := r.db.Where("user_auth_id = ?", userAuthID).Find(&authMethods).Error; err != nil {
		return nil, fmt.Errorf("failed to get user auth methods: %w", err)
	}

	summary := make(map[string]any)
	summary["total_methods"] = len(authMethods)

	enabledCount := 0
	verifiedCount := 0
	methodTypes := make([]string, 0)

	for _, method := range authMethods {
		if method.IsEnabled {
			enabledCount++
		}
		if method.IsVerified {
			verifiedCount++
		}
		methodTypes = append(methodTypes, string(method.Type))
	}

	summary["enabled_count"] = enabledCount
	summary["verified_count"] = verifiedCount
	summary["method_types"] = methodTypes

	return summary, nil
}

// CleanupExpiredMethods removes auth methods that haven't been verified within a timeframe
func (r *AuthMethodRepository) CleanupExpiredMethods(olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan)

	err := r.db.Where("is_verified = ? AND created_at < ?", false, cutoff).
		Delete(&user.AuthMethod{}).Error

	if err != nil {
		return fmt.Errorf("failed to cleanup expired methods: %w", err)
	}

	return nil
}

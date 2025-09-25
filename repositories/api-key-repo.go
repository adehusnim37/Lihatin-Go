package repositories

import (
	"errors"
	"fmt"
	"time"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/models/user"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// APIKeyRepository handles API key database operations
type APIKeyRepository struct {
	db *gorm.DB
}

// NewAPIKeyRepository creates a new API key repository
func NewAPIKeyRepository(db *gorm.DB) *APIKeyRepository {
	return &APIKeyRepository{db: db}
}

// CreateAPIKey creates a new API key using improved atomic operations
func (r *APIKeyRepository) CreateAPIKey(userID, name string, expiresAt *time.Time, permissions []string) (*user.APIKey, string, error) {
	// Set default permissions if none provided
	if len(permissions) == 0 {
		permissions = []string{"read", "write", "delete", "update"}
	}

	// Generate API key pair first (fail fast if generation fails)
	keyID, secretKey, secretKeyHash, keyPreview, err := utils.GenerateAPIKeyPair("")
	if err != nil {
		utils.Logger.Error("Failed to generate API key pair", "error", err.Error())
		return nil, "", utils.ErrAPIKeyCreateFailed
	}

	utils.Logger.Info("Starting API key creation process",
		"user_id", userID,
		"key_name", name,
		"key_preview", keyPreview,
		"permissions", permissions,
	)

	// Use atomic transaction for all database operations
	var apiKey user.APIKey
	err = r.db.Transaction(func(tx *gorm.DB) error {
		// 1. Validate user exists, is active, and verified (single query)
		var userAuth user.UserAuth
		if err := tx.Where("user_id = ? AND is_email_verified = ? AND is_active = ?",
			userID, true, true).First(&userAuth).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				utils.Logger.Warn("User not found or not eligible for API key creation",
					"user_id", userID)
				return utils.ErrUserNotFound
			}
			utils.Logger.Error("Failed to validate user for API key creation",
				"user_id", userID, "error", err.Error())
			return utils.ErrAPIKeyFailedFetching
		}

		// 2. Check for duplicate name and count limit in one query
		var existingKeys []user.APIKey
		if err := tx.Where("user_id = ? AND is_active = ? AND deleted_at IS NULL", userID, true).
			Find(&existingKeys).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			utils.Logger.Error("Failed to fetch existing API keys",
				"user_id", userID, "error", err.Error())
			return utils.ErrAPIKeyFailedFetching
		}

		// Check for duplicate name and count limit
		for _, key := range existingKeys {
			if key.Name == name {
				utils.Logger.Warn("API key name already exists",
					"user_id", userID, "name", name)
				return utils.ErrAPIKeyNameExists
			}
		}

		if len(existingKeys) >= 3 {
			utils.Logger.Warn("API key limit reached",
				"user_id", userID, "current_count", len(existingKeys))
			return utils.ErrAPIKeyLimitReached
		}

		// 3. Create the new API key with explicit timestamp handling
		now := time.Now()
		apiKey = user.APIKey{
			ID:          uuid.New().String(),
			UserID:      userID,
			Name:        name,
			Key:         keyID,
			KeyHash:     secretKeyHash,
			ExpiresAt:   expiresAt,
			IsActive:    true,
			Permissions: user.PermissionsList(permissions),
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		// Create using GORM with explicit handling
		if err := tx.Create(&apiKey).Error; err != nil {
			utils.Logger.Error("Failed to create API key in database",
				"user_id", userID,
				"key_name", name,
				"error", err.Error(),
				"permissions", permissions,
			)
			return utils.ErrAPIKeyCreateFailed
		}

		utils.Logger.Info("API key created successfully in transaction",
			"user_id", userID,
			"key_id", apiKey.ID,
			"key_name", name,
			"key_preview", keyPreview,
		)

		return nil
	})

	// Handle transaction errors
	if err != nil {
		return nil, "", err
	}

	// Log final success
	utils.Logger.Info("API key creation completed successfully",
		"user_id", userID,
		"key_id", apiKey.ID,
		"key_name", name,
		"key_preview", keyPreview,
	)

	// Return the full secret key (this is the only time it will be shown)
	fullAPIKey := keyID + "." + secretKey
	return &apiKey, fullAPIKey, nil
}

// GetAPIKeysByUserID retrieves all API keys for a user
func (r *APIKeyRepository) GetAPIKeysByUserID(userID string) ([]user.APIKey, error) {
	var apiKeys []user.APIKey
	var user user.User

	// Start a transaction
	tx := r.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Check if the user exists
	if err := tx.Where("id = ? AND deleted_at IS NULL", userID).First(&user).Error; err != nil {
		tx.Rollback()
		return nil, utils.ErrUserNotFound
	}

	// Build the query for fetching API keys
	q := tx.Where("deleted_at IS NULL")
	if user.Role != "admin" {
		q = q.Where("user_id = ?", userID)
	}

	// Fetch the API keys
	if err := q.Find(&apiKeys).Error; err != nil {
		tx.Rollback()
		return nil, utils.ErrAPIKeyNotFound
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return nil, utils.ErrAPIKeyFailedFetching
	}

	return apiKeys, nil
}

// GetAPIKeyByID retrieves an API key by ID
func (r *APIKeyRepository) GetAPIKeyByID(id dto.APIKeyIDRequest, userID string) (dto.APIKeyResponse, error) {
	var apiKey user.APIKey
	var user user.User

	// Start a transaction
	tx := r.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Check if the user exists
	if err := tx.Where("id = ? AND deleted_at IS NULL", userID).First(&user).Error; err != nil {
		tx.Rollback()
		return dto.APIKeyResponse{}, utils.ErrUserNotFound
	}

	// Build the query for fetching API keys
	q := tx.Where("deleted_at IS NULL")
	if user.Role != "admin" {
		q = q.Where("user_id = ?", userID)
	}

	// Fetch the API keys
	if err := q.First(&apiKey).Error; err != nil {
		tx.Rollback()
		return dto.APIKeyResponse{}, utils.ErrAPIKeyNotFound
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return dto.APIKeyResponse{}, utils.ErrAPIKeyFailedFetching
	}

	return dto.APIKeyResponse{
		ID:          apiKey.ID,
		Name:        apiKey.Name,
		KeyPreview:  utils.GetKeyPreview(apiKey.Key),
		LastUsedAt:  apiKey.LastUsedAt,
		ExpiresAt:   apiKey.ExpiresAt,
		IsActive:    apiKey.IsActive,
		Permissions: []string(apiKey.Permissions),
		CreatedAt:   apiKey.CreatedAt,
	}, nil
}

// ValidateAPIKey validates an API key and returns the associated user
func (r *APIKeyRepository) ValidateAPIKey(fullAPIKey string) (*user.User, *user.APIKey, error) {
	utils.Logger.Info("Validating API key", "key_preview", utils.GetKeyPreview(fullAPIKey))

	// Parse the full API key (format: keyID.secretKey)
	keyParts := utils.SplitAPIKey(fullAPIKey)
	if len(keyParts) != 2 {
		utils.Logger.Warn("Invalid API key format - missing separator", "key_preview", utils.GetKeyPreview(fullAPIKey))
		return nil, nil, fmt.Errorf("invalid API key format")
	}

	keyID := keyParts[0]
	secretKey := keyParts[1]

	// Validate key ID format
	if !utils.ValidateAPIKeyIDFormat(keyID) {
		utils.Logger.Warn("Invalid API key ID format", "key_id", keyID)
		return nil, nil, fmt.Errorf("invalid API key ID format")
	}

	// Find API key by key ID
	var apiKey user.APIKey
	if err := r.db.Where("`key` = ? AND is_active = ? AND deleted_at IS NULL AND (expires_at IS NULL OR expires_at > ?)",
		keyID, true, time.Now()).First(&apiKey).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.Logger.Warn("API key not found", "key_id", keyID)
			return nil, nil, fmt.Errorf("invalid API key")
		}
		utils.Logger.Error("Database error while validating API key", "error", err.Error())
		return nil, nil, fmt.Errorf("failed to validate API key: %w", err)
	}

	// Validate the secret key against stored hash
	if !utils.ValidateAPISecretKey(secretKey, apiKey.KeyHash) {
		utils.Logger.Warn("Invalid secret key for API key", "key_id", keyID, "user_id", apiKey.UserID)
		return nil, nil, fmt.Errorf("invalid API key")
	}

	// Get the associated user
	var user user.User
	if err := r.db.First(&user, "id = ?", apiKey.UserID).Error; err != nil {
		utils.Logger.Error("User not found for valid API key", "user_id", apiKey.UserID, "key_id", keyID)
		return nil, nil, fmt.Errorf("user not found")
	}

	// Update last used timestamp
	go func() {
		if err := r.db.Model(&apiKey).Update("last_used_at", time.Now()).Error; err != nil {
			utils.Logger.Error("Failed to update last_used_at for API key", "key_id", keyID, "error", err.Error())
		}
	}()

	utils.Logger.Info("API key validated successfully",
		"user_id", user.ID,
		"key_id", keyID,
		"key_name", apiKey.Name,
	)

	return &user, &apiKey, nil
}

// repositories/api-key-repo.go - Add these methods

// APIKeyCheckPermissions checks if API key has ANY of the required permissions
func (r *APIKeyRepository) APIKeyCheckPermissions(apiKeyID string, requiredPermissions []string) (bool, error) {
    var apiKey user.APIKey
    
    // Get API key from database
    if err := r.db.Where("id = ? AND is_active = ? AND deleted_at IS NULL", apiKeyID, true).
        First(&apiKey).Error; err != nil {
        if err == gorm.ErrRecordNotFound {
            utils.Logger.Warn("API key not found during permission check", "key_id", apiKeyID)
            return false, utils.ErrAPIKeyNotFound
        }
        utils.Logger.Error("Database error while checking API key permissions", 
            "key_id", apiKeyID, "error", err.Error())
        return false, fmt.Errorf("failed to check API key permissions: %w", err)
    }

    // Check if API key has ANY of the required permissions
    keyPermissions := []string(apiKey.Permissions)
    for _, requiredPerm := range requiredPermissions {
        for _, perm := range keyPermissions {
            if perm == requiredPerm || perm == "*" {
                utils.Logger.Info("API key has required permission (ANY match)",
                    "key_id", apiKeyID,
                    "key_name", apiKey.Name,
                    "required_permissions", requiredPermissions,
                    "matched_permission", perm,
                )
                return true, nil
            }
        }
    }

    utils.Logger.Warn("API key lacks required permissions (ANY match)",
        "key_id", apiKeyID,
        "key_name", apiKey.Name,
        "required_permissions", requiredPermissions,
        "available_permissions", keyPermissions,
    )
    return false, nil
}

// APIKeyCheckAllPermissions checks if API key has ALL required permissions
func (r *APIKeyRepository) APIKeyCheckAllPermissions(apiKeyID string, requiredPermissions []string) (bool, error) {
    var apiKey user.APIKey
    
    // Get API key from database
    if err := r.db.Where("id = ? AND is_active = ? AND deleted_at IS NULL", apiKeyID, true).
        First(&apiKey).Error; err != nil {
        if err == gorm.ErrRecordNotFound {
            utils.Logger.Warn("API key not found during permission check", "key_id", apiKeyID)
            return false, utils.ErrAPIKeyNotFound
        }
        utils.Logger.Error("Database error while checking API key permissions", 
            "key_id", apiKeyID, "error", err.Error())
        return false, fmt.Errorf("failed to check API key permissions: %w", err)
    }

    // Check if API key has ALL required permissions
    keyPermissions := []string(apiKey.Permissions)
    
    for _, requiredPerm := range requiredPermissions {
        hasPermission := false
        for _, perm := range keyPermissions {
            if perm == requiredPerm || perm == "*" {
                hasPermission = true
                break
            }
        }
        if !hasPermission {
            utils.Logger.Warn("API key missing required permission (ALL match)",
                "key_id", apiKeyID,
                "key_name", apiKey.Name,
                "missing_permission", requiredPerm,
                "required_permissions", requiredPermissions,
                "available_permissions", keyPermissions,
            )
            return false, nil
        }
    }

    utils.Logger.Info("API key has all required permissions",
        "key_id", apiKeyID,
        "key_name", apiKey.Name,
        "required_permissions", requiredPermissions,
        "available_permissions", keyPermissions,
    )
    return true, nil
}

// Enhanced permission validation with full API key (for direct validation)
func (r *APIKeyRepository) ValidateAPIKeyWithPermissions(fullAPIKey string, requiredPermissions []string, requireAll bool) (*user.User, *user.APIKey, bool, error) {
    // First validate the API key and get user/key info
    user, apiKeyRecord, err := r.ValidateAPIKey(fullAPIKey)
    if err != nil {
        utils.Logger.Warn("Invalid API key during permission check", 
            "key_preview", utils.GetKeyPreview(fullAPIKey),
            "error", err.Error())
        return nil, nil, false, fmt.Errorf("invalid API key: %w", err)
    }

    // Check permissions using repository methods
    var hasPermission bool
    if requireAll {
        hasPermission, err = r.APIKeyCheckAllPermissions(apiKeyRecord.ID, requiredPermissions)
    } else {
        hasPermission, err = r.APIKeyCheckPermissions(apiKeyRecord.ID, requiredPermissions)
    }

    if err != nil {
        return user, apiKeyRecord, false, err
    }

    return user, apiKeyRecord, hasPermission, nil
}

// UpdateAPIKey updates an API key with proper validation and type handling
func (r *APIKeyRepository) UpdateAPIKey(keyID dto.APIKeyIDRequest , userID string, req dto.UpdateAPIKeyRequest) (*user.APIKey, error) {
	var apiKey user.APIKey

	err := r.db.Transaction(func(tx *gorm.DB) error {
		// 1. Check if API key exists and belongs to user (or user is admin)
		var userModel user.User
		if err := tx.Where("id = ? AND deleted_at IS NULL", userID).First(&userModel).Error; err != nil {
			return utils.ErrUserNotFound
		}

		// Build query based on user role
		query := tx.Where("id = ? AND is_active = ? AND deleted_at IS NULL", keyID.ID, true)
		if userModel.Role != "admin" {
			query = query.Where("user_id = ?", userID)
		}

		if err := query.First(&apiKey).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return utils.ErrAPIKeyNotFound
			}
			return fmt.Errorf("failed to fetch API key: %w", err)
		}

		// 2. Prepare update map with only non-nil fields
		updates := make(map[string]any)

		if req.Name != nil {
			// Check for duplicate name (exclude current key)
			var existingKey user.APIKey
			if err := tx.Where("user_id = ? AND name = ? AND id != ? AND deleted_at IS NULL",
				apiKey.UserID, *req.Name, keyID).First(&existingKey).Error; err == nil {
				return utils.ErrAPIKeyNameExists
			}
			updates["name"] = *req.Name
		}

		if req.ExpiresAt != nil {
			// Validate expiration date is in the future
			if req.ExpiresAt.Before(time.Now()) {
				return fmt.Errorf("expiration date must be in the future")
			}
			updates["expires_at"] = *req.ExpiresAt
		}

		if req.Permissions != nil {
			// Validate permissions
			validPermissions := []string{"read", "write", "delete", "update"}
			for _, perm := range req.Permissions {
				valid := false
				for _, validPerm := range validPermissions {
					if perm == validPerm {
						valid = true
						break
					}
				}
				if !valid {
					return fmt.Errorf("invalid permission: %s", perm)
				}
			}
			updates["permissions"] = user.PermissionsList(req.Permissions)
		}

		if req.IsActive != nil {
			updates["is_active"] = *req.IsActive
		}

		// Always update timestamp
		updates["updated_at"] = time.Now()

		// 3. Perform the update
		if len(updates) > 1 { // More than just updated_at
			result := tx.Model(&apiKey).Updates(updates)
			if result.Error != nil {
				return fmt.Errorf("failed to update API key: %w", result.Error)
			}

			// Reload the updated model
			if err := tx.Where("id = ?", keyID).First(&apiKey).Error; err != nil {
				return fmt.Errorf("failed to reload updated API key: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	utils.Logger.Info("API key updated successfully",
		"user_id", userID,
		"key_id", keyID,
		"key_name", apiKey.Name,
	)

	return &apiKey, nil
}

// RevokeAPIKey soft deletes an API key with proper user validation and transaction handling
func (r *APIKeyRepository) RevokeAPIKey(id dto.APIKeyIDRequest, UserID string) error {
	var apiKey user.APIKey
	var user user.User

	// Start a transaction
	tx := r.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Check if the user exists
	if err := tx.Where("id = ? AND deleted_at IS NULL", UserID).First(&user).Error; err != nil {
		tx.Rollback()
		return utils.ErrUserNotFound
	}

	// Build the query for fetching API keys
	q := tx.Where("id = ? AND deleted_at IS NULL", id.ID)
	if user.Role != "admin" {
		q = q.Where("user_id = ?", UserID)
	}

	// Fetch the API key
	if err := q.First(&apiKey).Error; err != nil {
		tx.Rollback()
		return utils.ErrAPIKeyNotFound
	}

	// Update the API key to set is_active to false and mark as deleted
	if err := tx.Model(&apiKey).Updates(map[string]any{
		"is_active":  false,
		"deleted_at": time.Now(),
	}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to revoke API key: %w", err)
	}
	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return utils.ErrAPIKeyFailedFetching
	}
	utils.Logger.Info("API key revoked successfully",
		"user_id", UserID,
		"key_id", id,
		"key_name", apiKey.Name,
	)
	return nil
}

// DeleteExpiredAPIKeys removes expired API keys
func (r *APIKeyRepository) DeleteExpiredAPIKeys(id dto.APIKeyIDRequest) error {
	err := r.db.Where("id = ? AND expires_at IS NOT NULL AND expires_at < ?", id, time.Now()).Delete(&user.APIKey{}).Error
	if err != nil {
		return fmt.Errorf("failed to delete expired API keys: %w", err)
	}
	return nil
}

// GetAPIKeyStats returns API key statistics
func (r *APIKeyRepository) GetAPIKeyStats(userID string) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total API keys for user
	var total int64
	if err := r.db.Model(&user.APIKey{}).Where("user_id = ? AND deleted_at IS NULL", userID).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count total API keys: %w", err)
	}
	stats["total"] = total

	// Active API keys
	var active int64
	if err := r.db.Model(&user.APIKey{}).Where("user_id = ? AND is_active = ? AND deleted_at IS NULL", userID, true).Count(&active).Error; err != nil {
		return nil, fmt.Errorf("failed to count active API keys: %w", err)
	}
	stats["active"] = active

	// Expired API keys
	var expired int64
	if err := r.db.Model(&user.APIKey{}).Where("user_id = ? AND expires_at IS NOT NULL AND expires_at < ? AND deleted_at IS NULL",
		userID, time.Now()).Count(&expired).Error; err != nil {
		return nil, fmt.Errorf("failed to count expired API keys: %w", err)
	}
	stats["expired"] = expired

	return stats, nil
}

// CreateAPIKeyWithCustomPrefix creates a new API key with custom prefix
func (r *APIKeyRepository) CreateAPIKeyWithCustomPrefix(userID, name, prefix string, expiresAt *time.Time, permissions []string) (*user.APIKey, string, error) {
	// Generate API key pair with custom prefix
	keyID, secretKey, secretKeyHash, keyPreview, err := utils.GenerateAPIKeyPair(prefix)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate API key pair: %w", err)
	}

	utils.Logger.Info("Creating custom prefix API key",
		"user_id", userID,
		"key_name", name,
		"key_preview", keyPreview,
		"prefix", prefix,
	)

	apiKey := &user.APIKey{
		ID:          uuid.New().String(),
		UserID:      userID,
		Name:        name,
		Key:         keyID,
		KeyHash:     secretKeyHash,
		ExpiresAt:   expiresAt,
		IsActive:    true,
		Permissions: user.PermissionsList(permissions),
	}

	if err := r.db.Create(apiKey).Error; err != nil {
		utils.Logger.Error("Failed to create custom prefix API key",
			"user_id", userID,
			"key_name", name,
			"prefix", prefix,
			"error", err.Error(),
		)
		return nil, "", fmt.Errorf("failed to create API key: %w", err)
	}

	utils.Logger.Info("Custom prefix API key created successfully",
		"user_id", userID,
		"key_id", apiKey.ID,
		"key_name", name,
		"key_preview", keyPreview,
		"prefix", prefix,
	)

	// Return the full secret key (keyID.secretKey format)
	fullAPIKey := keyID + "." + secretKey
	return apiKey, fullAPIKey, nil
}

// RegenerateAPIKey regenerates an existing API key (creates new key, updates record)
func (r *APIKeyRepository) RegenerateAPIKey(keyID string, userID string) (*user.APIKey, string, error) {
	// First, get the existing API key
	var existingKey user.APIKey
	
	if err := r.db.Where("id = ? AND user_id = ? AND is_active = ? AND deleted_at IS NULL", keyID, userID, true).First(&existingKey).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, "", utils.ErrAPIKeyNotFound
		}
		return nil, "", utils.ErrAPIKeyInactive
	}

	// Generate new API key pair with same prefix
	newKeyID, newSecretKey, newSecretKeyHash, newKeyPreview, err := utils.GenerateAPIKeyPair("")
	if err != nil {
		return nil, "", utils.ErrAPIKeyCreateFailed
	}

	utils.Logger.Info("Regenerating API key",
		"user_id", userID,
		"old_key_id", existingKey.ID,
		"new_key_preview", newKeyPreview,
	)

	// Update the existing record with new key data using direct GORM updates
	updates := map[string]interface{}{
		"key":          newKeyID,
		"key_hash":     newSecretKeyHash,
		"updated_at":   time.Now(),
		"last_used_at": nil, // Reset last used timestamp
		"last_ip_used": nil, // Reset last IP used
	}

	if err := r.db.Model(&existingKey).Where("id = ?", keyID).Updates(updates).Error; err != nil {
		utils.Logger.Error("Failed to update API key with new values",
			"key_id", keyID,
			"user_id", userID,
			"error", err.Error(),
		)
		return nil, "", utils.ErrAPIKeyUpdateFailed
	}

	// Get updated API key
	var updatedKey user.APIKey
	if err := r.db.Where("id = ?", keyID).First(&updatedKey).Error; err != nil {
		return nil, "", utils.ErrAPIKeyFailedFetching
	}

	utils.Logger.Info("API key regenerated successfully",
		"user_id", userID,
		"key_id", keyID,
		"new_key_preview", newKeyPreview,
	)

	// Return the full new secret key
	fullAPIKey := newKeyID + "." + newSecretKey
	return &updatedKey, fullAPIKey, nil
}

// Helper function to extract prefix from existing key ID
func extractPrefixFromKey(keyID string) string {
	// Find the last underscore to separate prefix from random part
	lastUnderscore := -1
	for i := len(keyID) - 1; i >= 0; i-- {
		if keyID[i] == '_' {
			lastUnderscore = i
			break
		}
	}

	if lastUnderscore == -1 {
		return utils.DefaultKeyPrefix // Fallback to default prefix
	}

	return keyID[:lastUnderscore]
}

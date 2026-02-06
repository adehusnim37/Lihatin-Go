package repositories

import (
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/internal/pkg/auth"
	apperrors "github.com/adehusnim37/lihatin-go/internal/pkg/errors"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/models/logging"
	"github.com/adehusnim37/lihatin-go/models/user"
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
func (r *APIKeyRepository) CreateAPIKey(userID string, req dto.CreateAPIKeyRequest) (*user.APIKey, string, error) {
	// Set default permissions if none provided
	if len(req.Permissions) == 0 {
		req.Permissions = []string{"read", "write", "delete", "update"}
	}

	// Generate API key pair first (fail fast if generation fails)
	keyID, secretKey, secretKeyHash, keyPreview, err := auth.GenerateAPIKeyPair("")
	if err != nil {
		logger.Logger.Error("Failed to generate API key pair", "error", err.Error())
		return nil, "", apperrors.ErrAPIKeyCreateFailed
	}

	logger.Logger.Info("Starting API key creation process",
		"user_id", userID,
		"key_name", req.Name,
		"key_preview", keyPreview,
		"permissions", req.Permissions,
	)

	// Use atomic transaction for all database operations
	var apiKey user.APIKey
	err = r.db.Transaction(func(tx *gorm.DB) error {
		// 1. Validate user exists, is active, and verified (single query)
		var userAuth user.UserAuth
		if err := tx.Where("user_id = ? AND is_email_verified = ? AND is_active = ?",
			userID, true, true).First(&userAuth).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				logger.Logger.Warn("User not found or not eligible for API key creation",
					"user_id", userID)
				return apperrors.ErrUserNotFound
			}
			logger.Logger.Error("Failed to validate user for API key creation",
				"user_id", userID, "error", err.Error())
			return apperrors.ErrAPIKeyFailedFetching
		}

		// 2. Check for duplicate name and count limit in one query
		var existingKeys []user.APIKey
		if err := tx.Where("user_id = ? AND is_active = ? AND deleted_at IS NULL", userID, true).
			Find(&existingKeys).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Logger.Error("Failed to fetch existing API keys",
				"user_id", userID, "error", err.Error())
			return apperrors.ErrAPIKeyFailedFetching
		}

		// Check for duplicate name and count limit
		for _, key := range existingKeys {
			if key.Name == req.Name {
				logger.Logger.Warn("API key name already exists",
					"user_id", userID, "name", req.Name)
				return apperrors.ErrAPIKeyNameExists
			}
		}

		if len(existingKeys) >= 3 {
			logger.Logger.Warn("API key limit reached",
				"user_id", userID, "current_count", len(existingKeys))
			return apperrors.ErrAPIKeyLimitReached
		}

		// Prepare IP lists
		var allowedIPs user.IPList = user.IPList{} // Initialize as IPList, not []string
		var blockedIPs user.IPList = user.IPList{} // Initialize as IPList, not []string

		if len(req.AllowedIPs) > 0 {
			allowedIPs = user.IPList(req.AllowedIPs) // Convert slice to IPList
			logger.Logger.Debug("Setting allowed IPs", "allowed_ips", allowedIPs)
		}

		if len(req.BlockedIPs) > 0 {
			blockedIPs = user.IPList(req.BlockedIPs) // Convert slice to IPList
			logger.Logger.Debug("Setting blocked IPs", "blocked_ips", blockedIPs)
		}

		// 3. Create the new API key with explicit timestamp handling
		now := time.Now()
		apiKey = user.APIKey{
			ID:          uuid.New().String(),
			UserID:      userID,
			Name:        req.Name,
			Key:         keyID,
			KeyHash:     secretKeyHash,
			ExpiresAt:   req.ExpiresAt,
			IsActive:    true,
			BlockedIPs:  blockedIPs,
			AllowedIPs:  allowedIPs,
			LimitUsage:  req.LimitUsage,
			UsageCount:  0,
			Permissions: user.PermissionsList(req.Permissions),
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		// Create using GORM with explicit handling
		if err := tx.Create(&apiKey).Error; err != nil {
			logger.Logger.Error("Failed to create API key in database",
				"user_id", userID,
				"key_name", req.Name,
				"error", err.Error(),
				"permissions", req.Permissions,
				"blocked_ips", req.BlockedIPs,
				"allowed_ips", req.AllowedIPs,
			)
			return apperrors.ErrAPIKeyCreateFailed
		}

		logger.Logger.Info("API key created successfully in transaction",
			"user_id", userID,
			"key_id", apiKey.ID,
			"key_name", req.Name,
			"key_preview", keyPreview,
		)

		return nil
	})

	// Handle transaction errors
	if err != nil {
		return nil, "", err
	}

	// Log final success
	logger.Logger.Info("API key creation completed successfully",
		"user_id", userID,
		"key_id", apiKey.ID,
		"key_name", req.Name,
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
		return nil, apperrors.ErrUserNotFound
	}

	// Build the query for fetching API keys
	q := tx.Where("deleted_at IS NULL")
	if user.Role != "admin" {
		q = q.Where("user_id = ?", userID)
	}

	// Fetch the API keys
	if err := q.Find(&apiKeys).Error; err != nil {
		tx.Rollback()
		return nil, apperrors.ErrAPIKeyNotFound
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return nil, apperrors.ErrAPIKeyFailedFetching
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
		return dto.APIKeyResponse{}, apperrors.ErrUserNotFound
	}

	// Build the query for fetching API keys
	q := tx.Where("deleted_at IS NULL")
	if user.Role != "admin" {
		q = q.Where("user_id = ?", userID)
	}

	// Fetch the API keys
	if err := q.First(&apiKey).Error; err != nil {
		tx.Rollback()
		return dto.APIKeyResponse{}, apperrors.ErrAPIKeyNotFound
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return dto.APIKeyResponse{}, apperrors.ErrAPIKeyFailedFetching
	}

	return dto.APIKeyResponse{
		ID:          apiKey.ID,
		Name:        apiKey.Name,
		KeyPreview:  auth.GetKeyPreview(apiKey.Key),
		LastUsedAt:  apiKey.LastUsedAt,
		ExpiresAt:   apiKey.ExpiresAt,
		IsActive:    apiKey.IsActive,
		Permissions: []string(apiKey.Permissions),
		CreatedAt:   apiKey.CreatedAt,
	}, nil
}

// ValidateAPIKey validates an API key and returns the associated user
func (r *APIKeyRepository) ValidateAPIKey(fullAPIKey string, ip string) (*user.User, *user.APIKey, error) {
	logger.Logger.Info("Validating API key", "key_preview", auth.GetKeyPreview(fullAPIKey))

	// Parse the full API key (format: keyID.secretKey)
	keyParts := auth.SplitAPIKey(fullAPIKey)
	if len(keyParts) != 2 {
		logger.Logger.Warn("Invalid API key format - missing separator", "key_preview", auth.GetKeyPreview(fullAPIKey))
		return nil, nil, apperrors.ErrAPIKeyInvalidFormat
	}

	keyID := keyParts[0]
	secretKey := keyParts[1]

	// Validate key ID format
	if !auth.ValidateAPIKeyIDFormat(keyID) {
		logger.Logger.Warn("Invalid API key ID format", "key_id", keyID)
		return nil, nil, apperrors.ErrAPIKeyIDFailedFormat
	}

	// Find API key by key ID
	var apiKey user.APIKey
	if err := r.db.Where("`key` = ? AND is_active = ? AND deleted_at IS NULL AND (expires_at IS NULL OR expires_at > ?)",
		keyID, true, time.Now()).First(&apiKey).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Logger.Warn("API key not found", "key_id", keyID)
			return nil, nil, apperrors.ErrAPIKeyNotFound
		}
		logger.Logger.Error("Database error while validating API key", "error", err.Error())
		return nil, nil, fmt.Errorf("failed to validate API key: %w", err)
	}

	// validate usage limit
	if apiKey.LimitUsage != nil && apiKey.UsageCount >= *apiKey.LimitUsage {
		logger.Logger.Warn("API key usage limit reached", "key_id", keyID)
		return nil, nil, apperrors.ErrAPIKeyRateLimitExceeded
	}

	// validate IP restrictions
	if len(apiKey.AllowedIPs) > 0 {
		allowed := slices.Contains(apiKey.AllowedIPs, ip)
		if !allowed {
			logger.Logger.Warn("API key access from disallowed IP", "key_id", keyID, "ip", ip)
			return nil, nil, apperrors.ErrAPIKeyIPNotAllowed
		}
	}

	// check blocked IPs
	if slices.Contains(apiKey.BlockedIPs, ip) {
		logger.Logger.Warn("API key access from blocked IP", "key_id", keyID, "ip", ip)
		return nil, nil, apperrors.ErrAPIKeyIPNotAllowed
	}

	// Validate the secret key against stored hash
	if !auth.ValidateAPISecretKey(secretKey, apiKey.KeyHash) {
		logger.Logger.Warn("Invalid secret key for API key", "key_id", keyID, "user_id", apiKey.UserID)
		return nil, nil, apperrors.ErrAPIKeyInvalidFormat
	}

	// Get the associated user
	var user user.User
	if err := r.db.First(&user, "id = ?", apiKey.UserID).Error; err != nil {
		logger.Logger.Error("User not found for valid API key", "user_id", apiKey.UserID, "key_id", keyID)
		return nil, nil, apperrors.ErrUserNotFound
	}

	// Update last used timestamp and increment usage count asynchronously and get last IP Used
	go func() {
		if err := r.db.Model(&apiKey).Updates(map[string]interface{}{
			"last_used_at": time.Now(),
			"usage_count":  gorm.Expr("usage_count + ?", 1),
			"last_ip_used": ip,
		}).Error; err != nil {
			logger.Logger.Error("Failed to update usage stats for API key", "key_id", keyID, "error", err.Error())
		}
	}()

	logger.Logger.Info("API key validated successfully",
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
			logger.Logger.Warn("API key not found during permission check", "key_id", apiKeyID)
			return false, apperrors.ErrAPIKeyNotFound
		}
		logger.Logger.Error("Database error while checking API key permissions",
			"key_id", apiKeyID, "error", err.Error())
		return false, fmt.Errorf("failed to check API key permissions: %w", err)
	}

	// Check if API key has ANY of the required permissions
	keyPermissions := []string(apiKey.Permissions)
	for _, requiredPerm := range requiredPermissions {
		for _, perm := range keyPermissions {
			if perm == requiredPerm || perm == "*" {
				logger.Logger.Info("API key has required permission (ANY match)",
					"key_id", apiKeyID,
					"key_name", apiKey.Name,
					"required_permissions", requiredPermissions,
					"matched_permission", perm,
				)
				return true, nil
			}
		}
	}

	logger.Logger.Warn("API key lacks required permissions (ANY match)",
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
			logger.Logger.Warn("API key not found during permission check", "key_id", apiKeyID)
			return false, apperrors.ErrAPIKeyNotFound
		}
		logger.Logger.Error("Database error while checking API key permissions",
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
			logger.Logger.Warn("API key missing required permission (ALL match)",
				"key_id", apiKeyID,
				"key_name", apiKey.Name,
				"missing_permission", requiredPerm,
				"required_permissions", requiredPermissions,
				"available_permissions", keyPermissions,
			)
			return false, nil
		}
	}

	logger.Logger.Info("API key has all required permissions",
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
	user, apiKeyRecord, err := r.ValidateAPIKey(fullAPIKey, "")
	if err != nil {
		logger.Logger.Warn("Invalid API key during permission check",
			"key_preview", auth.GetKeyPreview(fullAPIKey),
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
func (r *APIKeyRepository) UpdateAPIKey(keyID dto.APIKeyIDRequest, userID string, req dto.UpdateAPIKeyRequest) (*user.APIKey, error) {
	var apiKey user.APIKey

	err := r.db.Transaction(func(tx *gorm.DB) error {
		// 1. Check if API key exists and belongs to user (or user is admin)
		var userModel user.User
		if err := tx.Where("id = ? AND deleted_at IS NULL", userID).First(&userModel).Error; err != nil {
			return apperrors.ErrUserNotFound
		}

		// Build query based on user role
		query := tx.Where("id = ? AND is_active = ? AND deleted_at IS NULL", keyID.ID, true)
		if userModel.Role != "admin" {
			query = query.Where("user_id = ?", userID)
		}

		if err := query.First(&apiKey).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return apperrors.ErrAPIKeyNotFound
			}
			return fmt.Errorf("failed to fetch API key: %w", err)
		}

		// 2. Prepare update map with only non-nil fields
		updates := make(map[string]any)

		if req.Name != nil {
			// Check for duplicate name (exclude current key)
			var existingKey user.APIKey
			if err := tx.Where("user_id = ? AND name = ? AND id != ? AND deleted_at IS NULL",
				apiKey.UserID, *req.Name, keyID.ID).First(&existingKey).Error; err == nil {
				return apperrors.ErrAPIKeyNameExists
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

		if req.LimitUsage != nil {
			if *req.LimitUsage < 0 {
				return fmt.Errorf("limit_usage must be non-negative or null for unlimited")
			}
			updates["limit_usage"] = req.LimitUsage
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
			if err := tx.Where("id = ?", keyID.ID).First(&apiKey).Error; err != nil {
				return fmt.Errorf("failed to reload updated API key: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	logger.Logger.Info("API key updated successfully",
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
		return apperrors.ErrUserNotFound
	}

	// Build the query for fetching API keys
	q := tx.Where("id = ? AND deleted_at IS NULL", id.ID)
	if user.Role != "admin" {
		q = q.Where("user_id = ?", UserID)
	}

	// Fetch the API key
	if err := q.First(&apiKey).Error; err != nil {
		tx.Rollback()
		return apperrors.ErrAPIKeyNotFound
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
		return apperrors.ErrAPIKeyFailedFetching
	}
	logger.Logger.Info("API key revoked successfully",
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
// GetAPIKeyStats returns API key statistics
func (r *APIKeyRepository) GetAPIKeyStats(userID string) (*dto.APIKeyStatsResponse, error) {
	stats := &dto.APIKeyStatsResponse{}

	// Basic counts
	query := r.db.Model(&user.APIKey{}).Where("user_id = ? AND deleted_at IS NULL", userID)

	if err := query.Count(&stats.TotalKeys).Error; err != nil {
		return nil, fmt.Errorf("failed to count total API keys: %w", err)
	}

	if err := query.Where("is_active = ?", true).Count(&stats.ActiveKeys).Error; err != nil {
		return nil, fmt.Errorf("failed to count active API keys: %w", err)
	}

	if err := r.db.Model(&user.APIKey{}).Where("user_id = ? AND expires_at IS NOT NULL AND expires_at < ? AND deleted_at IS NULL",
		userID, time.Now()).Count(&stats.ExpiredKeys).Error; err != nil {
		return nil, fmt.Errorf("failed to count expired API keys: %w", err)
	}

	// Total Usage
	var totalUsage sql.NullInt64
	if err := query.Session(&gorm.Session{}).Select("SUM(usage_count)").Scan(&totalUsage).Error; err != nil {
		return nil, fmt.Errorf("failed to get total usage: %w", err)
	}
	if totalUsage.Valid {
		stats.TotalUsage = totalUsage.Int64
	}

	// Most Used Key
	var mostUsed user.APIKey
	if err := query.Session(&gorm.Session{}).Order("usage_count DESC").First(&mostUsed).Error; err == nil {
		stats.MostUsedKey = &dto.APIKeyUsage{
			Name:       mostUsed.Name,
			UsageCount: mostUsed.UsageCount,
		}
	} else if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to get most used key: %w", err)
	}

	// Last Used Key (based on last_used_at)
	var lastUsed user.APIKey
	if err := query.Session(&gorm.Session{}).Where("last_used_at IS NOT NULL").Order("last_used_at DESC").First(&lastUsed).Error; err == nil {
		stats.LastUsedKey = &dto.APIKeyLastUsed{
			Name:       lastUsed.Name,
			LastUsedAt: lastUsed.LastUsedAt,
		}
	} else if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to get last used key: %w", err)
	}

	return stats, nil
}

// CreateAPIKeyWithCustomPrefix creates a new API key with custom prefix
func (r *APIKeyRepository) CreateAPIKeyWithCustomPrefix(userID, name, prefix string, expiresAt *time.Time, permissions []string) (*user.APIKey, string, error) {
	// Generate API key pair with custom prefix
	keyID, secretKey, secretKeyHash, keyPreview, err := auth.GenerateAPIKeyPair(prefix)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate API key pair: %w", err)
	}

	logger.Logger.Info("Creating custom prefix API key",
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
		logger.Logger.Error("Failed to create custom prefix API key",
			"user_id", userID,
			"key_name", name,
			"prefix", prefix,
			"error", err.Error(),
		)
		return nil, "", fmt.Errorf("failed to create API key: %w", err)
	}

	logger.Logger.Info("Custom prefix API key created successfully",
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
			return nil, "", apperrors.ErrAPIKeyNotFound
		}
		return nil, "", apperrors.ErrAPIKeyInactive
	}

	// Generate new API key pair with same prefix
	newKeyID, newSecretKey, newSecretKeyHash, newKeyPreview, err := auth.GenerateAPIKeyPair("")
	if err != nil {
		return nil, "", apperrors.ErrAPIKeyCreateFailed
	}

	logger.Logger.Info("Regenerating API key",
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
		logger.Logger.Error("Failed to update API key with new values",
			"key_id", keyID,
			"user_id", userID,
			"error", err.Error(),
		)
		return nil, "", apperrors.ErrAPIKeyUpdateFailed
	}

	// Get updated API key
	var updatedKey user.APIKey
	if err := r.db.Where("id = ?", keyID).First(&updatedKey).Error; err != nil {
		return nil, "", apperrors.ErrAPIKeyFailedFetching
	}

	logger.Logger.Info("API key regenerated successfully",
		"user_id", userID,
		"key_id", keyID,
		"new_key_preview", newKeyPreview,
	)

	// Return the full new secret key
	fullAPIKey := newKeyID + "." + newSecretKey
	return &updatedKey, fullAPIKey, nil
}

/*
ActivateAPIKey activates a deactivated API key
*/
func (r *APIKeyRepository) ActivateAPIKey(keyID dto.APIKeyIDRequest, userID string, userRole string) (*user.APIKey, error) {
	var apiKey user.APIKey

	// Start a transaction
	tx := r.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Build the query for fetching API keys
	q := tx.Where("id = ? AND deleted_at IS NULL", keyID.ID)
	if userRole != "admin" {
		q = q.Where("user_id = ?", userID)
	}

	// Fetch the API key
	if err := q.First(&apiKey).Error; err != nil {
		tx.Rollback()
		return nil, apperrors.ErrAPIKeyNotFound
	}

	// Update the API key to set is_active to true
	if err := tx.Model(&apiKey).Updates(map[string]any{
		"is_active":  true,
		"updated_at": time.Now(),
	}).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to activate API key: %w", err)
	}
	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return nil, apperrors.ErrAPIKeyFailedFetching
	}
	logger.Logger.Info("API key activated successfully",
		"user_id", userID,
		"key_id", keyID,
		"key_name", apiKey.Name,
	)
	return &apiKey, nil
}

/*
DeactivateAPIKey deactivates an active API key
*/
func (r *APIKeyRepository) DeactivateAPIKey(keyID dto.APIKeyIDRequest, userID string, userRole string) (*user.APIKey, error) {
	var apiKey user.APIKey

	// Start a transaction
	if userRole != "admin" {
		// Ensure the user owns the key if not admin
		if err := r.db.Where("id = ? AND user_id = ? AND deleted_at IS NULL", keyID.ID, userID).First(&apiKey).Error; err != nil {
			return nil, apperrors.ErrAPIKeyNotFound
		}
	} else {
		// Admin can access any key
		if err := r.db.Where("id = ? AND deleted_at IS NULL", keyID.ID).First(&apiKey).Error; err != nil {
			return nil, apperrors.ErrAPIKeyNotFound
		}
	}
	if !apiKey.IsActive {
		return nil, apperrors.ErrAPIKeyInactive
	}
	if userRole != "admin" && apiKey.UserID != userID {
		return nil, apperrors.ErrAPIKeyUnauthorized
	}
	// Update the API key to set is_active to false
	if err := r.db.Model(&apiKey).Updates(map[string]any{
		"is_active":  false,
		"updated_at": time.Now(),
	}).Error; err != nil {
		return nil, fmt.Errorf("failed to deactivate API key: %w", err)
	}

	logger.Logger.Info("API key deactivated successfully",
		"user_id", userID,
		"key_id", keyID,
		"key_name", apiKey.Name,
	)
	return &apiKey, nil
}

/*
APIKeyUsageHistory retrieves usage history for an API key with pagination and sorting
*/
func (r *APIKeyRepository) APIKeyUsageHistory(keyID dto.APIKeyIDRequest, userID, userRole string, page, limit int, sort, orderBy string) (*user.APIKey, []logging.ActivityLog, int64, error) {
	var apiKeys user.APIKey
	var activityLogs []logging.ActivityLog
	var totalCount int64

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}
	offset := (page - 1) * limit
	orderClause := sort + " " + orderBy

	// Start a transaction
	err := r.db.Transaction(func(tx *gorm.DB) error {
		query := tx.Where("id = ? AND deleted_at IS NULL", keyID.ID)
		if userRole != "admin" {
			query = query.Where("user_id = ?", userID)
		}
		if err := query.First(&apiKeys).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return apperrors.ErrAPIKeyNotFound
			}
			return apperrors.ErrAPIKeyFailedFetching
		}

		logger.Logger.Info("Fetching API key usage history",
			"user_id", userID,
			"key_id", keyID,
			"key", apiKeys.Key,
			"key_name", apiKeys.Name,
			"page", page,
			"limit", limit,
		)

		// Count total activity logs for pagination
		if err := tx.Model(&logging.ActivityLog{}).
			Where("api_key = ?", apiKeys.Key).
			Count(&totalCount).Error; err != nil {
			return apperrors.ErrActivityLogNotFound
		}

		logger.Logger.Info("Total activity logs found",
			"user_id", userID,
			"key_id", keyID,
			"key", apiKeys.Key,
			"key_name", apiKeys.Name,
			"total_logs", totalCount,
		)

		if err := tx.Where("api_key = ? AND deleted_at IS NULL", apiKeys.Key).
			Offset(offset).
			Limit(limit).
			Order(orderClause).
			Find(&activityLogs).Error; err != nil {
			return apperrors.ErrActivityLogNotFound
		}

		return nil
	})

	if err != nil {
		return nil, nil, 0, err
	}

	logger.Logger.Info("Fetched API key usage history",
		"user_id", userID,
		"key_id", keyID,
		"key_name", apiKeys.Name,
		"page", page,
		"limit", limit,
		"total_logs", totalCount,
	)

	return &apiKeys, activityLogs, totalCount, nil

}

/*
Helper function to extract prefix from existing key ID
*/
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
		return auth.DefaultKeyPrefix // Fallback to default prefix
	}

	return keyID[:lastUnderscore]
}

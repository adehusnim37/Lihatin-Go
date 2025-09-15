package repositories

import (
	"fmt"
	"time"

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

// CreateAPIKey creates a new API key using the improved GenerateAPIKeyPair method
func (r *APIKeyRepository) CreateAPIKey(userID, name string, expiresAt *time.Time, permissions []string) (*user.APIKey, string, error) {
	// Generate API key pair with structured format
	keyID, secretKey, secretKeyHash, keyPreview, err := utils.GenerateAPIKeyPair("")
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate API key pair: %w", err)
	}

	utils.Logger.Info("Creating new API key",
		"user_id", userID,
		"key_name", name,
		"key_preview", keyPreview,
	)

	apiKey := &user.APIKey{
		ID:          uuid.New().String(),
		UserID:      userID,
		Name:        name,
		Key:         keyID,         // Store the key ID (safe to display)
		KeyHash:     secretKeyHash, // Store the hashed secret key
		ExpiresAt:   expiresAt,
		IsActive:    true,
		Permissions: permissions,
	}

	if err := r.db.Create(apiKey).Error; err != nil {
		utils.Logger.Error("Failed to create API key in database",
			"user_id", userID,
			"key_name", name,
			"error", err.Error(),
		)
		return nil, "", fmt.Errorf("failed to create API key: %w", err)
	}

	utils.Logger.Info("API key created successfully",
		"user_id", userID,
		"key_id", apiKey.ID,
		"key_name", name,
		"key_preview", keyPreview,
	)

	// Return the full secret key (this is the only time it will be shown)
	fullAPIKey := keyID + "." + secretKey
	return apiKey, fullAPIKey, nil
}

// GetAPIKeysByUserID retrieves all API keys for a user
func (r *APIKeyRepository) GetAPIKeysByUserID(userID string) ([]user.APIKey, error) {
	var apiKeys []user.APIKey
	if err := r.db.Where("user_id = ? AND deleted_at IS NULL", userID).Find(&apiKeys).Error; err != nil {
		return nil, fmt.Errorf("failed to get API keys: %w", err)
	}
	return apiKeys, nil
}

// GetAPIKeyByID retrieves an API key by ID
func (r *APIKeyRepository) GetAPIKeyByID(id string) (*user.APIKey, error) {
	var apiKey user.APIKey
	if err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&apiKey).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("API key not found")
		}
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}
	return &apiKey, nil
}

// ValidateAPIKey validates an API key and returns the associated user
func (r *APIKeyRepository) ValidateAPIKey(fullAPIKey string) (*user.User, *user.APIKey, error) {
	utils.Logger.Info("Validating API key", "key_preview", utils.GetKeyPreview(fullAPIKey))

	// Parse the full API key (format: keyID.secretKey)
	keyParts := splitAPIKey(fullAPIKey)
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
	if err := r.db.Where("key = ? AND is_active = ? AND deleted_at IS NULL AND (expires_at IS NULL OR expires_at > ?)",
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

// Helper function to split API key into keyID and secretKey
func splitAPIKey(fullAPIKey string) []string {
	// Look for the first dot after the prefix to split keyID from secretKey
	prefixEnd := -1
	for i := 0; i < len(fullAPIKey); i++ {
		if fullAPIKey[i] == '_' {
			// Find the end of prefix (after last underscore in prefix)
			for j := i + 1; j < len(fullAPIKey); j++ {
				if fullAPIKey[j] == '.' {
					prefixEnd = j
					break
				}
			}
			break
		}
	}

	if prefixEnd == -1 {
		return []string{fullAPIKey} // No separator found
	}

	return []string{fullAPIKey[:prefixEnd], fullAPIKey[prefixEnd+1:]}
}

// UpdateAPIKey updates an API key
func (r *APIKeyRepository) UpdateAPIKey(id string, updates map[string]interface{}) error {
	result := r.db.Model(&user.APIKey{}).Where("id = ? AND deleted_at IS NULL", id).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to update API key: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("API key not found")
	}
	return nil
}

// RevokeAPIKey soft deletes an API key
func (r *APIKeyRepository) RevokeAPIKey(id string) error {
	result := r.db.Where("id = ?", id).Delete(&user.APIKey{})
	if result.Error != nil {
		return fmt.Errorf("failed to revoke API key: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("API key not found")
	}
	return nil
}

// DeleteExpiredAPIKeys removes expired API keys
func (r *APIKeyRepository) DeleteExpiredAPIKeys() error {
	err := r.db.Where("expires_at IS NOT NULL AND expires_at < ?", time.Now()).Delete(&user.APIKey{}).Error
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
		Permissions: permissions,
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
	if err := r.db.Where("id = ? AND user_id = ? AND deleted_at IS NULL", keyID, userID).First(&existingKey).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, "", fmt.Errorf("API key not found")
		}
		return nil, "", fmt.Errorf("failed to get API key: %w", err)
	}

	// Extract prefix from existing key
	existingPrefix := extractPrefixFromKey(existingKey.Key)

	// Generate new API key pair with same prefix
	newKeyID, newSecretKey, newSecretKeyHash, newKeyPreview, err := utils.GenerateAPIKeyPair(existingPrefix)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate new API key pair: %w", err)
	}

	utils.Logger.Info("Regenerating API key",
		"user_id", userID,
		"old_key_id", existingKey.ID,
		"new_key_preview", newKeyPreview,
	)

	// Update the existing record with new key data
	updates := map[string]interface{}{
		"key":          newKeyID,
		"key_hash":     newSecretKeyHash,
		"updated_at":   time.Now(),
		"last_used_at": nil, // Reset last used timestamp
	}

	if err := r.UpdateAPIKey(keyID, updates); err != nil {
		utils.Logger.Error("Failed to update API key with new values",
			"key_id", keyID,
			"user_id", userID,
			"error", err.Error(),
		)
		return nil, "", fmt.Errorf("failed to update API key: %w", err)
	}

	// Get updated API key
	var updatedKey user.APIKey
	if err := r.db.Where("id = ?", keyID).First(&updatedKey).Error; err != nil {
		return nil, "", fmt.Errorf("failed to get updated API key: %w", err)
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

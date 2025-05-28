package repositories

import (
	"fmt"
	"time"

	"github.com/adehusnim37/lihatin-go/models"
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

// CreateAPIKey creates a new API key
func (r *APIKeyRepository) CreateAPIKey(userID, name string, expiresAt *time.Time, permissions []string) (*models.APIKey, string, error) {
	// Generate API key
	keyString, err := utils.GenerateAPIKey()
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate API key: %w", err)
	}

	// Hash the API key for storage
	keyHash, err := utils.HashPassword(keyString)
	if err != nil {
		return nil, "", fmt.Errorf("failed to hash API key: %w", err)
	}

	apiKey := &models.APIKey{
		ID:          uuid.New().String(),
		UserID:      userID,
		Name:        name,
		Key:         keyString[:8] + "..." + keyString[len(keyString)-4:], // Store preview only
		KeyHash:     keyHash,
		ExpiresAt:   expiresAt,
		IsActive:    true,
		Permissions: permissions,
	}

	if err := r.db.Create(apiKey).Error; err != nil {
		return nil, "", fmt.Errorf("failed to create API key: %w", err)
	}

	return apiKey, keyString, nil
}

// GetAPIKeysByUserID retrieves all API keys for a user
func (r *APIKeyRepository) GetAPIKeysByUserID(userID string) ([]models.APIKey, error) {
	var apiKeys []models.APIKey
	if err := r.db.Where("user_id = ? AND deleted_at IS NULL", userID).Find(&apiKeys).Error; err != nil {
		return nil, fmt.Errorf("failed to get API keys: %w", err)
	}
	return apiKeys, nil
}

// GetAPIKeyByID retrieves an API key by ID
func (r *APIKeyRepository) GetAPIKeyByID(id string) (*models.APIKey, error) {
	var apiKey models.APIKey
	if err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&apiKey).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("API key not found")
		}
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}
	return &apiKey, nil
}

// ValidateAPIKey validates an API key and returns the associated user
func (r *APIKeyRepository) ValidateAPIKey(keyString string) (*models.User, *models.APIKey, error) {
	var apiKeys []models.APIKey
	
	// Get all active API keys (we need to check hash for each)
	if err := r.db.Where("is_active = ? AND deleted_at IS NULL AND (expires_at IS NULL OR expires_at > ?)", 
		true, time.Now()).Find(&apiKeys).Error; err != nil {
		return nil, nil, fmt.Errorf("failed to query API keys: %w", err)
	}

	// Check each API key hash
	for _, apiKey := range apiKeys {
		if err := utils.CheckPassword(apiKey.KeyHash, keyString); err == nil {
			// Found matching key, get the user
			var user models.User
			if err := r.db.First(&user, "id = ?", apiKey.UserID).Error; err != nil {
				continue // Skip if user not found
			}

			// Update last used timestamp
			r.db.Model(&apiKey).Update("last_used_at", time.Now())

			return &user, &apiKey, nil
		}
	}

	return nil, nil, fmt.Errorf("invalid API key")
}

// UpdateAPIKey updates an API key
func (r *APIKeyRepository) UpdateAPIKey(id string, updates map[string]interface{}) error {
	result := r.db.Model(&models.APIKey{}).Where("id = ? AND deleted_at IS NULL", id).Updates(updates)
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
	result := r.db.Where("id = ?", id).Delete(&models.APIKey{})
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
	err := r.db.Where("expires_at IS NOT NULL AND expires_at < ?", time.Now()).Delete(&models.APIKey{}).Error
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
	if err := r.db.Model(&models.APIKey{}).Where("user_id = ? AND deleted_at IS NULL", userID).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count total API keys: %w", err)
	}
	stats["total"] = total

	// Active API keys
	var active int64
	if err := r.db.Model(&models.APIKey{}).Where("user_id = ? AND is_active = ? AND deleted_at IS NULL", userID, true).Count(&active).Error; err != nil {
		return nil, fmt.Errorf("failed to count active API keys: %w", err)
	}
	stats["active"] = active

	// Expired API keys
	var expired int64
	if err := r.db.Model(&models.APIKey{}).Where("user_id = ? AND expires_at IS NOT NULL AND expires_at < ? AND deleted_at IS NULL", 
		userID, time.Now()).Count(&expired).Error; err != nil {
		return nil, fmt.Errorf("failed to count expired API keys: %w", err)
	}
	stats["expired"] = expired

	return stats, nil
}

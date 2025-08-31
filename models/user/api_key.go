package user

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"
)

// APIKey represents an API key for user authentication
type APIKey struct {
	ID          string     `json:"id" gorm:"primaryKey"`
	UserID      string     `json:"user_id" gorm:"size:191;not null;index"`
	Name        string     `json:"name" gorm:"size:100;not null" validate:"required,min=3,max=100"`
	Key         string     `json:"key" gorm:"uniqueIndex;size:255;not null"`
	KeyHash     string     `json:"-" gorm:"column:key_hash;size:255;not null"` // Store hashed version
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	IsActive    bool       `json:"is_active" gorm:"default:true"`
	Permissions []string   `json:"permissions" gorm:"type:json"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty" gorm:"index"`

	// Associations
	User *User `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

// TableName specifies the table name for GORM
func (APIKey) TableName() string {
	return "api_keys"
}

// KeyPreview returns the first 8 characters of the key followed by "..."
func (a *APIKey) KeyPreview() string {
	if len(a.Key) <= 8 {
		return a.Key + "..."
	}
	return a.Key[:8] + "..."
}

const (
	// DefaultKeyPrefix is a generic prefix for API keys.
	DefaultKeyPrefix = "sk_lh" // "sk" for Secret Key
	// KeyByteLength determines the length of the random part of the key.
	KeyByteLength = 32
	// SecretByteLength determines the length of the random part of the secret.
	SecretByteLength = 64
)

// GenerateAPIKeyPair creates a new API Key ID (Client ID part) and a Secret Key.
// The Key ID is returned directly.
// The Secret Key is returned directly ONCE for the user to copy.
// The HASH of the Secret Key is what you should store.
//
// Returns:
//
//	keyID (string): The full API Key ID (e.g., "sk_live_xxxxxxxxxxxx"). To be stored and used by the client.
//	secretKey (string): The plaintext secret key. SHOW THIS TO THE USER ONCE. DO NOT STORE.
//	secretKeyHash (string): The SHA256 hash of the secret key. STORE THIS in APIClientKey.SecretKeyHash.
//	keyPreview (string): Last few characters of keyID for display.
//	error: Any error during generation.
func GenerateAPIKeyPair(prefix string) (keyID, secretKey, secretKeyHash, keyPreview string, err error) {
	if prefix == "" {
		prefix = DefaultKeyPrefix
	}

	// Generate Key ID part
	keyIDBytes := make([]byte, KeyByteLength)
	if _, err = rand.Read(keyIDBytes); err != nil {
		return "", "", "", "", fmt.Errorf("failed to generate key ID bytes: %w", err)
	}
	keyID = prefix + base64.URLEncoding.EncodeToString(keyIDBytes)
	// Make it a bit shorter and more URL-friendly if needed, or ensure length constraints
	// For simplicity, using the full base64 encoded string here.

	if len(keyID) > 4 { // Ensure there are enough characters for a preview
		keyPreview = keyID[len(keyID)-4:]
	} else {
		keyPreview = keyID
	}

	// Generate Secret Key part
	secretBytes := make([]byte, SecretByteLength)
	if _, err = rand.Read(secretBytes); err != nil {
		return "", "", "", "", fmt.Errorf("failed to generate secret key bytes: %w", err)
	}
	secretKey = base64.URLEncoding.EncodeToString(secretBytes) // This is the raw secret

	// Hash the Secret Key
	hash := sha256.Sum256([]byte(secretKey))
	secretKeyHash = fmt.Sprintf("%x", hash[:]) // Hex encode the hash

	return keyID, secretKey, secretKeyHash, keyPreview, nil
}

// ValidateAPISecretKey compares a provided plaintext secret key against a stored hash.
func ValidateAPISecretKey(providedSecretKey, storedSecretKeyHash string) bool {
	hash := sha256.Sum256([]byte(providedSecretKey))
	providedSecretKeyHash := fmt.Sprintf("%x", hash[:])
	return providedSecretKeyHash == storedSecretKeyHash
}

// APIKeyRequest represents the request to create a new API key
type APIKeyRequest struct {
	Name        string     `json:"name" validate:"required,min=3,max=100"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	Permissions []string   `json:"permissions,omitempty"`
}

type UpdateAPIKeyRequest struct {
	Name        *string    `json:"name,omitempty" validate:"omitempty,min=3,max=100"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	Permissions []string   `json:"permissions,omitempty"`
	IsActive    *bool      `json:"is_active,omitempty"`
}

// APIKeyResponse represents the API key response (without sensitive data)
type APIKeyResponse struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	KeyPreview  string     `json:"key_preview"` // Only first 8 characters + "..."
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	IsActive    bool       `json:"is_active"`
	Permissions []string   `json:"permissions"`
	CreatedAt   time.Time  `json:"created_at"`
}

// CreateAPIKeyResponse represents the response when creating a new API key
type CreateAPIKeyResponse struct {
	APIKey *APIKeyResponse `json:"api_key"`
	Key    string          `json:"key"` // Only shown once during creation
}

// PaginatedAPIKeysResponse represents paginated API keys
type PaginatedAPIKeysResponse struct {
	APIKeys    []APIKeyResponse `json:"api_keys"`
	TotalCount int64            `json:"total_count"`
	Page       int              `json:"page"`
	Limit      int              `json:"limit"`
	TotalPages int              `json:"total_pages"`
}

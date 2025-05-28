package models

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"
)

// APIClient represents an external application or service that is authorized
// to access your API. It can be associated with a User who manages it.
type APIClient struct {
	ID     string `json:"id"`                                      // Primary Key (e.g., UUID)
	UserID string `json:"user_id"`                                 // Foreign Key to User.ID - the user who owns/manages this client
	User   *User  `json:"user,omitempty" gorm:"foreignKey:UserID"` // Optional: for eager loading

	Name        string     `json:"name" validate:"required,min=3,max=100"` // User-friendly name for the client (e.g., "Partner X Integration", "Mobile App Backend")
	Description string     `json:"description,omitempty" validate:"max=500"`
	IsEnabled   bool       `json:"is_enabled" default:"true"`                      // Whether this client and its keys are active
	RateLimit   int        `json:"rate_limit,omitempty"`                           // Optional: custom rate limit (requests per period)
	AllowedIPs  []string   `json:"allowed_ips,omitempty"`                          // Optional: restrict API access to specific IP addresses (store as JSON array or comma-separated string)
	WebhookURL  string     `json:"webhook_url,omitempty" validate:"omitempty,url"` // Optional: for sending notifications to the client
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`                         // Timestamp of the last successful API call by any key of this client
	RevokedAt   *time.Time `json:"revoked_at,omitempty"`                           // If the entire client access is revoked
	Scopes      []string   `json:"scopes,omitempty"`                               // List of permissions/scopes granted to this client (e.g., "read:products", "write:orders")

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" gorm:"index"` // For soft deletes
}

// APIClientKey holds the actual credentials for an APIClient.
// An APIClient can have multiple keys (e.g., for rotation).
type APIClientKey struct {
	ID          string     `json:"id"`                              // Primary Key (e.g., UUID) - This is often the "Client ID" or "Access Key ID" part
	APIClientID string     `json:"api_client_id"`                   // Foreign Key to APIClient.ID
	APIClient   *APIClient `json:"-" gorm:"foreignKey:APIClientID"` // Optional: for eager loading

	// The SecretKeyHash stores the HASH of the secret key.
	// The actual secret key is only shown to the user ONCE upon creation.
	// You should NOT store the plaintext secret key.
	SecretKeyHash string `json:"-"` // Hashed secret key

	// Prefix can be used to identify the key type or for easier recognition (e.g., "sk_live_", "pk_test_")
	// It's part of the visible key ID, not the secret.
	Prefix string `json:"prefix,omitempty" validate:"max=10"`

	// The KeyPreview is the last few characters of the actual key ID (not the secret),
	// useful for display purposes so users can identify keys without exposing the full ID.
	KeyPreview string `json:"key_preview,omitempty" validate:"max=10"`

	IsEnabled  bool       `json:"is_enabled" default:"true"` // Whether this specific key is active
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`      // Optional: for keys that automatically expire
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`    // Timestamp of the last successful API call with this specific key
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`      // If this specific key is revoked

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" gorm:"index"` // For soft deletes
}

// --- Helper functions for API Key Generation (Conceptual) ---

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

// Note:
// - The `User` struct is assumed to be the one defined previously.
// - `gorm:"..."` tags are examples for GORM.
// - You'll need robust procedures for generating, displaying (once), storing hashes, and validating these keys.
// - The helper functions `GenerateAPIKeyPair` and `ValidateAPISecretKey` are conceptual and provide a basic idea.
//   In a production system, consider using established libraries or patterns for key generation and management.
//   Ensure your random generation is cryptographically secure.

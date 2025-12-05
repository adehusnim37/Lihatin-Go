package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

// GenerateSecureToken generates a secure random token of specified byte length
func GenerateSecureToken(byteLength int) (string, error) {
	bytes := make([]byte, byteLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// GenerateVerificationToken generates a secure random token for email verification
func GenerateVerificationToken() (string, error) {
	// Generate 32 bytes of random data (256 bits)
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Convert to hex string (64 characters)
	return hex.EncodeToString(bytes), nil
}

// GeneratePasswordResetToken generates a secure random token for password reset
func GeneratePasswordResetToken() (string, error) {
	// Generate 32 bytes of random data (256 bits)
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Convert to hex string (64 characters)
	return hex.EncodeToString(bytes), nil
}

func GeneratePasscodeResetCode() (string, error) {
	// Generate 4 bytes of random data
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Convert to hex string (64 characters)
	return hex.EncodeToString(bytes), nil
}

// GenerateRecoveryCode generates a recovery code for 2FA backup
func GenerateRecoveryCode() (string, error) {
	// Generate 8 bytes of random data
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Convert to hex and format as XXXX-XXXX
	hexStr := hex.EncodeToString(bytes)
	return fmt.Sprintf("%s-%s", hexStr[:8], hexStr[8:]), nil
}

// GenerateAPIKey generates a secure API key (DEPRECATED - use GenerateAPIKeyPair)
func GenerateAPIKey() (string, error) {
	// Generate 24 bytes of random data
	bytes := make([]byte, 24)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Convert to hex string (48 characters)
	return hex.EncodeToString(bytes), nil
}

// API Key constants
const (
	// DefaultKeyPrefix is a generic prefix for API keys.
	DefaultKeyPrefix = "sk_lh" // "sk" for Secret Key, "lh" for Lihatin
	// KeyByteLength determines the length of the random part of the key.
	KeyByteLength = 24
	// SecretByteLength determines the length of the random part of the secret.
	SecretByteLength = 32
)

/*
GenerateAPIKeyPair creates a new API key ID and secret key pair.
The key ID is prefixed (default "sk_lh") and the secret key is a random string.
The secret key is hashed for secure storage.

Returns:
- keyID: The generated API key ID (e.g., "sk_lh_AbCdEf123...")
- secretKey: The generated secret key (to be shown only once)
- secretKeyHash: The SHA-256 hash of the secret key (for storage)
- keyPreview: A preview of the key ID (first 8 chars + "..." + last 4 chars)
- error: Any error encountered during generation
*/
func GenerateAPIKeyPair(prefix string) (keyID, secretKey, secretKeyHash, keyPreview string, err error) {
	if prefix == "" {
		prefix = DefaultKeyPrefix
	}

	// Generate Key ID part (shorter, URL-friendly)
	keyIDBytes := make([]byte, KeyByteLength)
	if _, err = rand.Read(keyIDBytes); err != nil {
		return "", "", "", "", fmt.Errorf("failed to generate key ID bytes: %w", err)
	}

	// Create keyID with prefix and base64 URL encoding (no padding)
	keyIDRandom := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(keyIDBytes)
	keyID = prefix + "_" + keyIDRandom

	// Generate Secret Key part (longer for security)
	secretBytes := make([]byte, SecretByteLength)
	if _, err = rand.Read(secretBytes); err != nil {
		return "", "", "", "", fmt.Errorf("failed to generate secret key bytes: %w", err)
	}
	secretKey = base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(secretBytes)

	// Hash the Secret Key for secure storage
	hash := sha256.Sum256([]byte(secretKey))
	secretKeyHash = fmt.Sprintf("%x", hash[:]) // Hex encode the hash

	// Create user-friendly preview (first 8 + "..." + last 4)
	if len(keyID) > 12 {
		keyPreview = keyID[:8] + "..." + keyID[len(keyID)-4:]
	} else {
		keyPreview = keyID + "..."
	}

	return keyID, secretKey, secretKeyHash, keyPreview, nil
}

// ValidateAPISecretKey compares a provided plaintext secret key against a stored hash.
func ValidateAPISecretKey(providedSecretKey, storedSecretKeyHash string) bool {
	hash := sha256.Sum256([]byte(providedSecretKey))
	providedSecretKeyHash := fmt.Sprintf("%x", hash[:])
	return providedSecretKeyHash == storedSecretKeyHash
}

func ValidateAPIKeyFormat(key string) bool {
	if len(key) != 48 {
		return false
	}
	for _, c := range key {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

// ValidateAPIKeyIDFormat validates the format of an API key ID
func ValidateAPIKeyIDFormat(keyID string) bool {
	// Must start with a valid prefix
	if !hasValidPrefix(keyID) {
		return false
	}

	// Must have minimum length (prefix + underscore + random part)
	if len(keyID) < 10 {
		return false
	}

	// Must contain underscore after prefix
	parts := splitKeyID(keyID)
	return len(parts) == 2 && len(parts[1]) > 0
}

// GetKeyPreview generates a preview for any API key ID
func GetKeyPreview(keyID string) string {
	if len(keyID) <= 12 {
		return keyID + "..."
	}
	return keyID[:8] + "..." + keyID[len(keyID)-4:]
}

// Helper functions
func hasValidPrefix(keyID string) bool {
	validPrefixes := []string{"sk_lh", "sk_test", "sk_live"}
	for _, prefix := range validPrefixes {
		if len(keyID) >= len(prefix) && keyID[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

func splitKeyID(keyID string) []string {
	for i := 0; i < len(keyID); i++ {
		if keyID[i] == '_' {
			// Find the last underscore to split prefix from random part
			for j := len(keyID) - 1; j > i; j-- {
				if keyID[j] == '_' {
					return []string{keyID[:j], keyID[j+1:]}
				}
			}
			return []string{keyID[:i], keyID[i+1:]}
		}
	}
	return []string{keyID}
}

// SplitAPIKey is a helper function to split API key into keyID and secretKey
func SplitAPIKey(fullAPIKey string) []string {
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

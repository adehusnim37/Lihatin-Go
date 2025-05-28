package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

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

// GenerateAPIKey generates a secure API key
func GenerateAPIKey() (string, error) {
	// Generate 24 bytes of random data
	bytes := make([]byte, 24)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Convert to hex string (48 characters)
	return hex.EncodeToString(bytes), nil
}

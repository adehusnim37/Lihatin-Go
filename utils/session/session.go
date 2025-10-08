package session

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/adehusnim37/lihatin-go/utils"
)

// SessionMetadata is defined in metadata.go

// GenerateSessionID creates a simple cryptographically secure random session ID
// This ID is used as a Redis key, with metadata stored as the value
// Format: sess_<64_hex_chars> (32 random bytes)
// Note: userID, purpose, etc are ignored in this simple version - metadata is stored in Redis
func GenerateSessionID(userID, purpose, ipAddress, userAgent string, expireDuration time.Duration) (string, error) {
	// Generate 32 bytes of cryptographically secure random data
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate random session ID: %w", err)
	}

	// Convert to hex string (64 characters)
	sessionID := fmt.Sprintf("sess_%s", hex.EncodeToString(randomBytes))

	utils.Logger.Info("Generated new session ID",
		"session_preview", utils.GetKeyPreview(sessionID),
		"user_id", userID,
		"purpose", purpose,
	)

	return sessionID, nil
}

// ValidateSessionID validates the format of a session ID
// Only checks if the session ID has correct format (sess_ prefix + 64 hex chars)
// Does NOT validate expiration or check Redis - use Manager.Get() for full validation
func ValidateSessionID(sessionID string) (*SessionMetadata, bool) {
	// Check prefix
	if !strings.HasPrefix(sessionID, "sess_") {
		return nil, false
	}

	// Remove prefix and check length
	hexPart := strings.TrimPrefix(sessionID, "sess_")

	// Should be exactly 64 hex characters (32 bytes)
	if len(hexPart) != 64 {
		return nil, false
	}

	// Verify all characters are valid hex
	_, err := hex.DecodeString(hexPart)
	if err != nil {
		return nil, false
	}

	// Format is valid, but we can't extract metadata without Redis lookup
	// Return nil metadata to indicate we need to fetch from Redis
	return nil, true
}

// IsValidSessionFormat checks if session ID has valid format
// Quick validation without Redis lookup
func IsValidSessionFormat(sessionID string) bool {
	if !strings.HasPrefix(sessionID, "sess_") {
		return false
	}

	hexPart := strings.TrimPrefix(sessionID, "sess_")
	if len(hexPart) != 64 {
		return false
	}

	_, err := hex.DecodeString(hexPart)
	return err == nil
}

package session

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/adehusnim37/lihatin-go/utils"
)

// SessionMetadata is defined in metadata.go

// GenerateSessionID creates a sophisticated session ID with metadata and signature
func GenerateSessionID(userID, purpose, ipAddress, userAgent string, expireDuration time.Duration) (string, error) {
	now := time.Now()

	// Create session metadata
	metadata := SessionMetadata{
		UserID:    userID,
		Purpose:   purpose,
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(expireDuration).Unix(),
		IPAddress: ipAddress,
		UserAgent: userAgent,
	}

	// Serialize metadata to JSON
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return "", fmt.Errorf("failed to serialize metadata: %w", err)
	}

	// Generate 16 bytes of random data for entropy
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Create timestamp bytes (8 bytes)
	timestampBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(timestampBytes, uint64(now.Unix()))

	// Combine: timestamp + random + metadata
	payload := make([]byte, 0, 8+16+len(metadataBytes))
	payload = append(payload, timestampBytes...)
	payload = append(payload, randomBytes...)
	payload = append(payload, metadataBytes...)

	// Create HMAC signature using secret key
	secret := getSessionSecret()
	signature := generateHMAC(payload, secret)

	// Final format: base64(timestamp + random + metadata) + "." + base64(signature)
	payloadEncoded := base64.URLEncoding.EncodeToString(payload)
	signatureEncoded := base64.URLEncoding.EncodeToString(signature)

	sessionID := fmt.Sprintf("sess_%s.%s", payloadEncoded, signatureEncoded)

	return sessionID, nil
}

// ValidateSessionID validates and parses a session ID
func ValidateSessionID(sessionID string) (*SessionMetadata, bool) {
	// Check prefix
	if !strings.HasPrefix(sessionID, "sess_") {
		return nil, false
	}

	// Remove prefix
	sessionID = strings.TrimPrefix(sessionID, "sess_")

	// Split payload and signature
	parts := strings.Split(sessionID, ".")
	if len(parts) != 2 {
		return nil, false
	}

	payloadEncoded := parts[0]
	signatureEncoded := parts[1]

	// Decode payload
	payload, err := base64.URLEncoding.DecodeString(payloadEncoded)
	if err != nil {
		return nil, false
	}

	// Decode signature
	providedSignature, err := base64.URLEncoding.DecodeString(signatureEncoded)
	if err != nil {
		return nil, false
	}

	// Verify HMAC signature
	secret := getSessionSecret()
	expectedSignature := generateHMAC(payload, secret)

	if !hmac.Equal(providedSignature, expectedSignature) {
		return nil, false // Signature mismatch - possible tampering
	}

	// Parse payload
	if len(payload) < 24 { // 8 bytes timestamp + 16 bytes random
		return nil, false
	}

	// Extract timestamp (first 8 bytes)
	timestampBytes := payload[:8]
	timestamp := int64(binary.BigEndian.Uint64(timestampBytes))

	// Extract metadata (after timestamp + random bytes)
	metadataBytes := payload[24:]

	// Parse metadata JSON
	var metadata SessionMetadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return nil, false
	}

	// Validate timestamp matches metadata
	if timestamp != metadata.IssuedAt {
		return nil, false
	}

	// Check expiration
	if time.Now().Unix() > metadata.ExpiresAt {
		return nil, false // Session expired
	}

	return &metadata, true
}

// RevokeSession invalidates a session (placeholder for future database integration)
func RevokeSession(sessionID string) error {
	// TODO: Add to revoked sessions database/cache
	// For now, just validate the session exists
	_, isValid := ValidateSessionID(sessionID)
	if !isValid {
		return fmt.Errorf("invalid session ID")
	}

	// Log revocation
	if metadata, err := ExtractSessionInfo(sessionID); err == nil {
		utils.Logger.Info("Session revoked",
			"user_id", metadata.UserID,
			"purpose", metadata.Purpose,
			"session_preview", getSessionPreview(sessionID),
		)
	}

	return nil
}

// RefreshSession creates a new session with extended expiration
func RefreshSession(sessionID string, newExpireDuration time.Duration) (string, error) {
	// Extract current session info
	metadata, err := ExtractSessionInfo(sessionID)
	if err != nil {
		return "", fmt.Errorf("failed to extract session info: %w", err)
	}

	// Validate current session
	if _, isValid := ValidateSessionID(sessionID); !isValid {
		return "", fmt.Errorf("invalid or expired session")
	}

	// Create new session with same metadata but new expiration
	return GenerateSessionID(
		metadata.UserID,
		metadata.Purpose,
		metadata.IPAddress,
		metadata.UserAgent,
		newExpireDuration,
	)
}

// GenerateSimpleSession creates a basic session without metadata (for backward compatibility)
func GenerateSimpleSession(userID string, expireDuration time.Duration) (string, error) {
	return GenerateSessionID(userID, "simple", "", "", expireDuration)
}

// Helper function to get session secret from environment
func getSessionSecret() string {
	secret := os.Getenv("SESSION_SECRET")
	if secret == "" {
		// Fallback to JWT secret if SESSION_SECRET not set
		secret = os.Getenv("JWT_SECRET")
		if secret == "" {
			panic("Neither SESSION_SECRET nor JWT_SECRET environment variable is set")
		}
	}
	return secret
}

// generateHMAC creates HMAC-SHA256 signature
func generateHMAC(data []byte, secret string) []byte {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(data)
	return h.Sum(nil)
}

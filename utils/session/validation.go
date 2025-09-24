package session

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/adehusnim37/lihatin-go/utils"
)

// ValidateSessionForUser checks if session belongs to specific user with additional validations
func ValidateSessionForUser(sessionID, expectedUserID, currentIP, currentUserAgent string) (*SessionMetadata, error) {
	metadata, isValid := ValidateSessionID(sessionID)
	if !isValid {
		return nil, fmt.Errorf("invalid session ID")
	}

	// Check user ID
	if metadata.UserID != expectedUserID {
		return nil, fmt.Errorf("session belongs to different user")
	}

	// Optional: Check IP consistency (for security)
	if metadata.IPAddress != "" && currentIP != "" && metadata.IPAddress != currentIP {
		utils.Logger.Warn("IP address changed during session",
			"user_id", expectedUserID,
			"original_ip", metadata.IPAddress,
			"current_ip", currentIP,
		)
		// Could return error for strict security, or just log for monitoring
	}

	// Optional: Check User-Agent consistency
	if metadata.UserAgent != "" && currentUserAgent != "" && metadata.UserAgent != currentUserAgent {
		utils.Logger.Warn("User-Agent changed during session",
			"user_id", expectedUserID,
			"original_ua", metadata.UserAgent,
			"current_ua", currentUserAgent,
		)
	}

	return metadata, nil
}

// ValidateSessionFormat checks if sessionID has correct format without full validation
func ValidateSessionFormat(sessionID string) bool {
	if !strings.HasPrefix(sessionID, "sess_") {
		return false
	}

	sessionID = strings.TrimPrefix(sessionID, "sess_")
	parts := strings.Split(sessionID, ".")

	// Should have exactly 2 parts (payload.signature)
	if len(parts) != 2 {
		return false
	}

	// Both parts should be valid base64
	if _, err := base64.URLEncoding.DecodeString(parts[0]); err != nil {
		return false
	}

	if _, err := base64.URLEncoding.DecodeString(parts[1]); err != nil {
		return false
	}

	return true
}

// ValidateSessionStrict performs strict validation with IP and User-Agent enforcement
func ValidateSessionStrict(sessionID, expectedUserID, currentIP, currentUserAgent string) (*SessionMetadata, error) {
	metadata, isValid := ValidateSessionID(sessionID)
	if !isValid {
		return nil, fmt.Errorf("invalid session ID")
	}

	// Check user ID
	if metadata.UserID != expectedUserID {
		return nil, fmt.Errorf("session belongs to different user")
	}

	// Strict IP validation - return error if IP changed
	if metadata.IPAddress != "" && currentIP != "" && metadata.IPAddress != currentIP {
		return nil, fmt.Errorf("IP address changed from %s to %s", metadata.IPAddress, currentIP)
	}

	// Strict User-Agent validation - return error if UA changed
	if metadata.UserAgent != "" && currentUserAgent != "" && metadata.UserAgent != currentUserAgent {
		return nil, fmt.Errorf("user agent changed")
	}

	return metadata, nil
}

// ValidateSessionPurpose checks if session has specific purpose
func ValidateSessionPurpose(sessionID string, expectedPurposes ...string) (*SessionMetadata, error) {
	metadata, isValid := ValidateSessionID(sessionID)
	if !isValid {
		return nil, fmt.Errorf("invalid session ID")
	}

	// Check if session purpose matches any of expected purposes
	for _, expectedPurpose := range expectedPurposes {
		if metadata.Purpose == expectedPurpose {
			return metadata, nil
		}
	}

	return nil, fmt.Errorf("session purpose '%s' not in allowed purposes: %v",
		metadata.Purpose, expectedPurposes)
}

// ValidateSessionNotExpired checks if session is not expired (lightweight validation)
func ValidateSessionNotExpired(sessionID string) error {
	if IsSessionExpired(sessionID) {
		return fmt.Errorf("session expired")
	}
	return nil
}

// ValidateMultipleSessions validates multiple session IDs and returns valid ones
func ValidateMultipleSessions(sessionIDs []string) map[string]*SessionMetadata {
	validSessions := make(map[string]*SessionMetadata)

	for _, sessionID := range sessionIDs {
		if metadata, isValid := ValidateSessionID(sessionID); isValid {
			validSessions[sessionID] = metadata
		}
	}

	return validSessions
}

// ValidateSessionMinimumTimeLeft ensures session has minimum time remaining
func ValidateSessionMinimumTimeLeft(sessionID string, minimumDuration time.Duration) error {
	timeLeft, err := GetSessionTimeLeft(sessionID)
	if err != nil {
		return err
	}

	if timeLeft < minimumDuration {
		return fmt.Errorf("session expires too soon: %v remaining, need at least %v",
			timeLeft, minimumDuration)
	}

	return nil
}

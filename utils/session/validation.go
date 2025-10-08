package session

import (
	"context"
	"fmt"

	"github.com/adehusnim37/lihatin-go/utils"
)

// ValidateSessionForUser validates session from Redis and checks user/IP/UA
// This function requires the Manager to be initialized
func ValidateSessionForUser(sessionID, expectedUserID, currentIP, currentUserAgent string) (*SessionMetadata, error) {
	// First check format
	if !IsValidSessionFormat(sessionID) {
		return nil, fmt.Errorf("invalid session ID format")
	}

	// Get session from Redis (this validates expiration too)
	manager := GetManager()
	if manager == nil {
		return nil, fmt.Errorf("session manager not initialized")
	}

	sess, err := manager.Get(context.Background(), sessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found or expired: %w", err)
	}

	// Convert Session to SessionMetadata
	metadata := &SessionMetadata{
		UserID:    sess.UserID,
		Purpose:   sess.Purpose,
		IssuedAt:  sess.CreatedAt.Unix(),
		ExpiresAt: sess.ExpiresAt.Unix(),
		IPAddress: sess.IPAddress,
		UserAgent: sess.UserAgent,
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

// ValidateSessionStrict performs strict validation with IP and User-Agent enforcement
func ValidateSessionStrict(sessionID, expectedUserID, currentIP, currentUserAgent string) (*SessionMetadata, error) {
	if !IsValidSessionFormat(sessionID) {
		return nil, fmt.Errorf("invalid session ID format")
	}

	manager := GetManager()
	if manager == nil {
		return nil, fmt.Errorf("session manager not initialized")
	}

	sess, err := manager.Get(context.Background(), sessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found or expired: %w", err)
	}

	metadata := &SessionMetadata{
		UserID:    sess.UserID,
		Purpose:   sess.Purpose,
		IssuedAt:  sess.CreatedAt.Unix(),
		ExpiresAt: sess.ExpiresAt.Unix(),
		IPAddress: sess.IPAddress,
		UserAgent: sess.UserAgent,
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
	if !IsValidSessionFormat(sessionID) {
		return nil, fmt.Errorf("invalid session ID format")
	}

	manager := GetManager()
	if manager == nil {
		return nil, fmt.Errorf("session manager not initialized")
	}

	sess, err := manager.Get(context.Background(), sessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found or expired: %w", err)
	}

	metadata := &SessionMetadata{
		UserID:    sess.UserID,
		Purpose:   sess.Purpose,
		IssuedAt:  sess.CreatedAt.Unix(),
		ExpiresAt: sess.ExpiresAt.Unix(),
		IPAddress: sess.IPAddress,
		UserAgent: sess.UserAgent,
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

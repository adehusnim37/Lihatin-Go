package session

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// SessionMetadata contains session information
type SessionMetadata struct {
	UserID    string `json:"user_id"`
	Purpose   string `json:"purpose"`   // "login", "registration", "api", etc
	IssuedAt  int64  `json:"issued_at"` // Unix timestamp
	ExpiresAt int64  `json:"expires_at"`
	IPAddress string `json:"ip_address,omitempty"`
	UserAgent string `json:"user_agent,omitempty"`
}

// IsExpired checks if session is expired
func (s *SessionMetadata) IsExpired() bool {
	return time.Now().Unix() > s.ExpiresAt
}

// TimeLeft returns remaining session time
func (s *SessionMetadata) TimeLeft() time.Duration {
	expiresAt := time.Unix(s.ExpiresAt, 0)
	remaining := time.Until(expiresAt)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// Age returns how long ago the session was created
func (s *SessionMetadata) Age() time.Duration {
	issuedAt := time.Unix(s.IssuedAt, 0)
	return time.Since(issuedAt)
}

// IsValidFor checks if session is valid for a specific duration
func (s *SessionMetadata) IsValidFor(duration time.Duration) bool {
	return s.TimeLeft() >= duration
}

// String returns a string representation of session metadata
func (s *SessionMetadata) String() string {
	return fmt.Sprintf("Session{UserID: %s, Purpose: %s, Age: %v, TimeLeft: %v}",
		s.UserID, s.Purpose, s.Age(), s.TimeLeft())
}

// ExtractSessionInfo extracts session information without full validation (for debugging/logging)
func ExtractSessionInfo(sessionID string) (*SessionMetadata, error) {
	if !strings.HasPrefix(sessionID, "sess_") {
		return nil, fmt.Errorf("invalid session format")
	}

	sessionID = strings.TrimPrefix(sessionID, "sess_")
	parts := strings.Split(sessionID, ".")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid session structure")
	}

	payload, err := base64.URLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("failed to decode payload")
	}

	if len(payload) < 24 {
		return nil, fmt.Errorf("payload too short")
	}

	metadataBytes := payload[24:]
	var metadata SessionMetadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata")
	}

	return &metadata, nil
}

// IsSessionExpired checks if session is expired without full validation
func IsSessionExpired(sessionID string) bool {
	metadata, err := ExtractSessionInfo(sessionID)
	if err != nil {
		return true
	}
	return metadata.IsExpired()
}

// GetSessionTimeLeft returns remaining time for session
func GetSessionTimeLeft(sessionID string) (time.Duration, error) {
	metadata, err := ExtractSessionInfo(sessionID)
	if err != nil {
		return 0, err
	}

	remaining := metadata.TimeLeft()
	if remaining == 0 {
		return 0, fmt.Errorf("session expired")
	}

	return remaining, nil
}

// GetSessionUserID extracts user ID from session without full validation
func GetSessionUserID(sessionID string) (string, error) {
	metadata, err := ExtractSessionInfo(sessionID)
	if err != nil {
		return "", err
	}
	return metadata.UserID, nil
}

// GetSessionPurpose extracts purpose from session without full validation
func GetSessionPurpose(sessionID string) (string, error) {
	metadata, err := ExtractSessionInfo(sessionID)
	if err != nil {
		return "", err
	}
	return metadata.Purpose, nil
}

// GetSessionAge returns how long ago the session was created
func GetSessionAge(sessionID string) (time.Duration, error) {
	metadata, err := ExtractSessionInfo(sessionID)
	if err != nil {
		return 0, err
	}
	return metadata.Age(), nil
}

// getSessionPreview returns a preview of session ID for safe logging
func getSessionPreview(sessionID string) string {
	if len(sessionID) <= 12 {
		return sessionID
	}
	return sessionID[:8] + "..." + sessionID[len(sessionID)-4:]
}

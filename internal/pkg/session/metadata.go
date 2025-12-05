package session

import (
	"context"
	"fmt"
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

// Note: Since session ID is now a simple random key,
// all metadata must be fetched from Redis using the Manager.
// The functions below require the Manager to be initialized.

// GetSessionInfo fetches session information from Redis
func GetSessionInfo(sessionID string) (*SessionMetadata, error) {
	manager := GetManager()
	if manager == nil {
		return nil, fmt.Errorf("session manager not initialized")
	}

	session, err := manager.Get(context.Background(), sessionID)
	if err != nil {
		return nil, err
	}

	// Convert Session to SessionMetadata
	return &SessionMetadata{
		UserID:    session.UserID,
		Purpose:   session.Purpose,
		IssuedAt:  session.CreatedAt.Unix(),
		ExpiresAt: session.ExpiresAt.Unix(),
		IPAddress: session.IPAddress,
		UserAgent: session.UserAgent,
	}, nil
}

// IsSessionExpired checks if session is expired by checking Redis
func IsSessionExpired(sessionID string) bool {
	metadata, err := GetSessionInfo(sessionID)
	if err != nil {
		return true
	}

	return metadata.IsExpired()
}

// GetSessionTimeLeft returns remaining time for session from Redis
func GetSessionTimeLeft(sessionID string) (time.Duration, error) {
	metadata, err := GetSessionInfo(sessionID)
	if err != nil {
		return 0, err
	}

	remaining := metadata.TimeLeft()
	if remaining == 0 {
		return 0, fmt.Errorf("session expired")
	}

	return remaining, nil
}

// GetSessionUserID extracts user ID from session via Redis
func GetSessionUserID(sessionID string) (string, error) {
	metadata, err := GetSessionInfo(sessionID)
	if err != nil {
		return "", err
	}
	return metadata.UserID, nil
}

// GetSessionPurpose extracts purpose from session via Redis
func GetSessionPurpose(sessionID string) (string, error) {
	metadata, err := GetSessionInfo(sessionID)
	if err != nil {
		return "", err
	}
	return metadata.Purpose, nil
}

// GetSessionAge returns how long ago the session was created
func GetSessionAge(sessionID string) (time.Duration, error) {
	metadata, err := GetSessionInfo(sessionID)
	if err != nil {
		return 0, err
	}
	return metadata.Age(), nil
}

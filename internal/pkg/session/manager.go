package session

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/redis/go-redis/v9"
)

var (
	// ErrSessionNotFound indicates session was not found in store
	ErrSessionNotFound = errors.New("session not found")

	// ErrSessionExpired indicates session has expired
	ErrSessionExpired = errors.New("session expired")

	// ErrInvalidSession indicates session is malformed or invalid
	ErrInvalidSession = errors.New("invalid session")

	// globalManager is the singleton session manager instance
	globalManager *Manager
)

// Session represents a user session with metadata
type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Purpose   string    `json:"purpose"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
	DeviceID  string    `json:"device_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	LastSeen  time.Time `json:"last_seen"`
}

// Manager handles session lifecycle with Redis backend
type Manager struct {
	store      *RedisStore
	defaultTTL time.Duration
}

// NewManager creates a new session manager with Redis store
func NewManager(redisAddr, redisPassword string, redisDB int, defaultTTL time.Duration) (*Manager, error) {
	store, err := NewRedisStore(redisAddr, redisPassword, redisDB)
	if err != nil {
		return nil, fmt.Errorf("failed to create Redis store: %w", err)
	}

	return &Manager{
		store:      store,
		defaultTTL: defaultTTL,
	}, nil
}

// Create creates a new session and stores it in Redis
func (m *Manager) Create(ctx context.Context, userID, purpose, ipAddress, userAgent, deviceID string) (*Session, error) {
	// Generate secure random session ID
	sessionID, err := m.generateSecureID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session ID: %w", err)
	}

	now := time.Now()
	session := &Session{
		ID:        sessionID,
		UserID:    userID,
		Purpose:   purpose,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		DeviceID:  deviceID,
		CreatedAt: now,
		ExpiresAt: now.Add(m.defaultTTL),
		LastSeen:  now,
	}

	// Store in Redis
	if err := m.store.Set(ctx, sessionID, session, m.defaultTTL); err != nil {
		return nil, fmt.Errorf("failed to store session: %w", err)
	}

	logger.Logger.Info("Session created",
		"session_id", getSessionPreview(sessionID),
		"user_id", userID,
		"purpose", purpose,
		"ip", ipAddress,
		"expires_at", session.ExpiresAt,
	)

	return session, nil
}

// Get retrieves a session from Redis
func (m *Manager) Get(ctx context.Context, sessionID string) (*Session, error) {
	session, err := m.store.Get(ctx, sessionID)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Check if session has expired
	if time.Now().After(session.ExpiresAt) {
		// Delete expired session
		_ = m.store.Delete(ctx, sessionID)
		return nil, ErrSessionExpired
	}

	return session, nil
}

// Refresh extends the session TTL and updates last seen time
func (m *Manager) Refresh(ctx context.Context, sessionID string) error {
	// Get current session
	session, err := m.Get(ctx, sessionID)
	if err != nil {
		return err
	}

	// Update last seen and expiration
	session.LastSeen = time.Now()
	session.ExpiresAt = time.Now().Add(m.defaultTTL)

	// Save updated session
	if err := m.store.Set(ctx, sessionID, session, m.defaultTTL); err != nil {
		return fmt.Errorf("failed to refresh session: %w", err)
	}

	logger.Logger.Debug("Session refreshed",
		"session_id", getSessionPreview(sessionID),
		"user_id", session.UserID,
		"new_expires_at", session.ExpiresAt,
	)

	return nil
}

// Delete removes a session from Redis
func (m *Manager) Delete(ctx context.Context, sessionID string) error {
	// Get session for logging before deletion
	session, _ := m.store.Get(ctx, sessionID)

	if err := m.store.Delete(ctx, sessionID); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	if session != nil {
		logger.Logger.Info("Session deleted",
			"session_id", getSessionPreview(sessionID),
			"user_id", session.UserID,
		)
	}

	return nil
}

// DeleteAllUserSessions removes all sessions for a specific user
func (m *Manager) DeleteAllUserSessions(ctx context.Context, userID string) error {
	if err := m.store.DeleteByUserID(ctx, userID); err != nil {
		return fmt.Errorf("failed to delete user sessions: %w", err)
	}

	logger.Logger.Info("All user sessions deleted", "user_id", userID)
	return nil
}

// Validate checks if a session is valid
func (m *Manager) Validate(ctx context.Context, sessionID string) (*Session, error) {
	session, err := m.Get(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// Additional validation can be added here
	// e.g., IP address validation, device fingerprint, etc.

	return session, nil
}

// GetActiveSessionCount returns the number of active sessions for a user
func (m *Manager) GetActiveSessionCount(ctx context.Context, userID string) (int, error) {
	count, err := m.store.GetActiveSessionCount(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to get session count: %w", err)
	}
	return count, nil
}

// GetTTL returns the remaining time until session expires
func (m *Manager) GetTTL(ctx context.Context, sessionID string) (time.Duration, error) {
	ttl, err := m.store.GetTTL(ctx, sessionID)
	if err != nil {
		return 0, fmt.Errorf("failed to get session TTL: %w", err)
	}
	return ttl, nil
}

// Close closes the session manager and Redis connection
func (m *Manager) Close() error {
	return m.store.Close()
}

// generateSecureID generates a cryptographically secure random session ID
// Format: sess_<64_hex_chars> (32 random bytes in hex)
func (m *Manager) generateSecureID() (string, error) {
	// This delegates to the session.GenerateSessionID with dummy parameters
	// since the new GenerateSessionID only needs to generate random hex
	// We'll use the metadata later when we store in Redis
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}

	// Convert to hex and add prefix
	return fmt.Sprintf("sess_%x", randomBytes), nil
}

// SetManager sets the global session manager instance
func SetManager(manager *Manager) {
	globalManager = manager
}

// GetManager returns the global session manager instance
func GetManager() *Manager {
	return globalManager
}

// GetRedisClient returns the Redis client from the store
func (m *Manager) GetRedisClient() *redis.Client {
	return m.store.GetClient()
}

// getSessionPreview returns first 16 chars of session ID for logging (security)
func getSessionPreview(sessionID string) string {
	if len(sessionID) > 16 {
		return sessionID[:16] + "..."
	}
	return sessionID
}

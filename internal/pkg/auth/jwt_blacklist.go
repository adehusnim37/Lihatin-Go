package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	blacklistPrefix    = "jwt_blacklist:"
	refreshTokenPrefix = "refresh_token:"
)

// JWTBlacklistManager handles JWT blacklist operations in Redis
type JWTBlacklistManager struct {
	redisClient *redis.Client
}

// NewJWTBlacklistManager creates a new blacklist manager
func NewJWTBlacklistManager(redisClient *redis.Client) *JWTBlacklistManager {
	return &JWTBlacklistManager{
		redisClient: redisClient,
	}
}

// BlacklistJWT adds a JWT ID (jti) to the blacklist with TTL matching token expiration
func (m *JWTBlacklistManager) BlacklistJWT(ctx context.Context, jti string, expiresAt time.Time) error {
	key := blacklistPrefix + jti
	ttl := time.Until(expiresAt)

	// Only blacklist if token hasn't expired yet
	if ttl <= 0 {
		return nil // Token already expired, no need to blacklist
	}

	// Set the blacklist entry with TTL
	err := m.redisClient.Set(ctx, key, "1", ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to blacklist JWT: %w", err)
	}

	return nil
}

// IsJWTBlacklisted checks if a JWT ID is blacklisted
func (m *JWTBlacklistManager) IsJWTBlacklisted(ctx context.Context, jti string) (bool, error) {
	key := blacklistPrefix + jti

	exists, err := m.redisClient.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check JWT blacklist: %w", err)
	}

	return exists > 0, nil
}

// RefreshTokenManager handles refresh token operations in Redis
type RefreshTokenManager struct {
	redisClient *redis.Client
}

// NewRefreshTokenManager creates a new refresh token manager
func NewRefreshTokenManager(redisClient *redis.Client) *RefreshTokenManager {
	return &RefreshTokenManager{
		redisClient: redisClient,
	}
}

// RefreshTokenData contains refresh token metadata
type RefreshTokenData struct {
	UserID    string
	SessionID string
	DeviceID  string
	LastIP    string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// StoreRefreshToken stores a refresh token in Redis
func (m *RefreshTokenManager) StoreRefreshToken(ctx context.Context, refreshToken string, data RefreshTokenData) error {
	key := refreshTokenPrefix + refreshToken
	ttl := time.Until(data.ExpiresAt)

	if ttl <= 0 {
		return fmt.Errorf("refresh token already expired")
	}

	// Store as hash for structured data
	err := m.redisClient.HSet(ctx, key, map[string]interface{}{
		"user_id":    data.UserID,
		"session_id": data.SessionID,
		"device_id":  data.DeviceID,
		"last_ip":    data.LastIP,
		"created_at": data.CreatedAt.Unix(),
		"expires_at": data.ExpiresAt.Unix(),
	}).Err()

	if err != nil {
		return fmt.Errorf("failed to store refresh token: %w", err)
	}

	// Set expiration
	err = m.redisClient.Expire(ctx, key, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to set refresh token expiration: %w", err)
	}

	return nil
}

// GetRefreshToken retrieves refresh token data from Redis
func (m *RefreshTokenManager) GetRefreshToken(ctx context.Context, refreshToken string) (*RefreshTokenData, error) {
	key := refreshTokenPrefix + refreshToken

	// Get all fields
	result, err := m.redisClient.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("refresh token not found or expired")
	}

	// Parse timestamps
	createdAt := time.Unix(parseInt64(result["created_at"]), 0)
	expiresAt := time.Unix(parseInt64(result["expires_at"]), 0)

	return &RefreshTokenData{
		UserID:    result["user_id"],
		SessionID: result["session_id"],
		DeviceID:  result["device_id"],
		LastIP:    result["last_ip"],
		CreatedAt: createdAt,
		ExpiresAt: expiresAt,
	}, nil
}

// DeleteRefreshToken removes a refresh token from Redis
func (m *RefreshTokenManager) DeleteRefreshToken(ctx context.Context, refreshToken string) error {
	key := refreshTokenPrefix + refreshToken
	err := m.redisClient.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete refresh token: %w", err)
	}
	return nil
}

// DeleteAllUserRefreshTokens removes all refresh tokens for a user (useful for logout from all devices)
func (m *RefreshTokenManager) DeleteAllUserRefreshTokens(ctx context.Context, userID string) error {
	// Scan for all refresh tokens
	iter := m.redisClient.Scan(ctx, 0, refreshTokenPrefix+"*", 0).Iterator()

	for iter.Next(ctx) {
		key := iter.Val()

		// Get user_id from this refresh token
		tokenUserID, err := m.redisClient.HGet(ctx, key, "user_id").Result()
		if err != nil {
			continue
		}

		// Delete if it belongs to the user
		if tokenUserID == userID {
			m.redisClient.Del(ctx, key)
		}
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("failed to scan refresh tokens: %w", err)
	}

	return nil
}

// Helper function to parse int64 from string
func parseInt64(s string) int64 {
	var result int64
	fmt.Sscanf(s, "%d", &result)
	return result
}

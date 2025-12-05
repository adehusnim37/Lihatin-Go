package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/redis/go-redis/v9"
)

const (
	// PendingAuthPrefix is the Redis key prefix for pending auth tokens
	PendingAuthPrefix = "pending_auth:"
	// PendingAuthTTL is how long a pending auth token is valid (5 minutes)
	PendingAuthTTL = 5 * time.Minute
)

// pendingAuthRedisClient holds the Redis client for pending auth operations
var pendingAuthRedisClient *redis.Client

// InitPendingAuthRedis initializes the Redis client for pending auth tokens
func InitPendingAuthRedis(client *redis.Client) {
	pendingAuthRedisClient = client
}

// GeneratePendingAuthToken creates a temporary token for TOTP verification
// This token is stored in Redis and must be verified within PendingAuthTTL
func GeneratePendingAuthToken(ctx context.Context, userID string) (string, error) {
	if pendingAuthRedisClient == nil {
		return "", fmt.Errorf("pending auth redis client not initialized")
	}

	// Generate random token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	token := hex.EncodeToString(tokenBytes)

	// Store in Redis with userID as value
	key := PendingAuthPrefix + token
	err := pendingAuthRedisClient.Set(ctx, key, userID, PendingAuthTTL).Err()
	if err != nil {
		return "", fmt.Errorf("failed to store pending auth token: %w", err)
	}

	logger.Logger.Info("Pending auth token created",
		"user_id", userID,
		"token_preview", GetKeyPreview(token),
		"expires_in", PendingAuthTTL.String(),
	)

	return token, nil
}

// ValidatePendingAuthToken validates a pending auth token and returns the userID
// The token is deleted after successful validation (one-time use)
func ValidatePendingAuthToken(ctx context.Context, token string) (string, error) {
	if pendingAuthRedisClient == nil {
		return "", fmt.Errorf("pending auth redis client not initialized")
	}

	key := PendingAuthPrefix + token

	// Get and delete in one operation (atomic)
	userID, err := pendingAuthRedisClient.GetDel(ctx, key).Result()
	if err != nil {
		logger.Logger.Warn("Invalid or expired pending auth token",
			"token_preview", GetKeyPreview(token),
			"error", err.Error(),
		)
		return "", fmt.Errorf("invalid or expired pending auth token")
	}

	logger.Logger.Info("Pending auth token validated",
		"user_id", userID,
		"token_preview", GetKeyPreview(token),
	)

	return userID, nil
}

// DeletePendingAuthToken manually deletes a pending auth token (for cleanup)
func DeletePendingAuthToken(ctx context.Context, token string) error {
	if pendingAuthRedisClient == nil {
		return fmt.Errorf("pending auth redis client not initialized")
	}

	key := PendingAuthPrefix + token
	return pendingAuthRedisClient.Del(ctx, key).Err()
}

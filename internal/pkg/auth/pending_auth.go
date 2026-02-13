package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	apperrors "github.com/adehusnim37/lihatin-go/internal/pkg/errors"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/redis/go-redis/v9"
)

const (
	// PendingAuthPrefix is the Redis key prefix for pending auth tokens
	PendingAuthPrefix = "pending_auth:"
	// PendingAuthAttemptsPrefix is the Redis key prefix for pending auth attempts
	PendingAuthAttemptsPrefix = "pending_auth_attempts:"
	// PendingAuthTTL is how long a pending auth token is valid (5 minutes)
	PendingAuthTTL = 5 * time.Minute
	// PendingAuthMaxAttempts is the maximum number of TOTP attempts allowed
	PendingAuthMaxAttempts = 5
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
		return "", apperrors.ErrServerInternalPendingAuth
	}

	// Generate random token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		logger.Logger.Error("Failed to generate random token for pending auth",
			"error", err.Error(),
		)
		return "", apperrors.ErrCreatePendingAuthFailed
	}
	token := hex.EncodeToString(tokenBytes)

	// Store in Redis with userID as value
	key := PendingAuthPrefix + token
	err := pendingAuthRedisClient.Set(ctx, key, userID, PendingAuthTTL).Err()
	if err != nil {
		logger.Logger.Error("Failed to store pending auth token in Redis",
			"error", err.Error(),
		)
		return "", apperrors.ErrCreatePendingAuthFailed
	}

	logger.Logger.Info("Pending auth token created",
		"user_id", userID,
		"token_preview", GetKeyPreview(token),
		"expires_in", PendingAuthTTL.String(),
	)

	return token, nil
}

// ValidatePendingAuthToken validates a pending auth token and returns the userID
// The token remains valid until it expires, is consumed, or max attempts is exceeded.
func ValidatePendingAuthToken(ctx context.Context, token string) (string, error) {
	if pendingAuthRedisClient == nil {
		return "", apperrors.ErrServerInternalPendingAuth
	}

	key := PendingAuthPrefix + token

	userID, err := pendingAuthRedisClient.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			logger.Logger.Warn("Invalid or expired pending auth token",
				"token_preview", GetKeyPreview(token),
			)
			return "", apperrors.ErrPendingAuthNotFound
		}

		logger.Logger.Warn("Invalid or expired pending auth token",
			"token_preview", GetKeyPreview(token),
			"error", err.Error(),
		)
		return "", apperrors.ErrServerInternalPendingAuth
	}

	logger.Logger.Info("Pending auth token validated",
		"user_id", userID,
		"token_preview", GetKeyPreview(token),
	)

	return userID, nil
}

// IncrementPendingAuthAttempts increments failed attempts for a pending auth token.
// Returns remaining attempts. If attempts are exhausted, token is revoked.
func IncrementPendingAuthAttempts(ctx context.Context, token string) (int, error) {
	if pendingAuthRedisClient == nil {
		return 0, apperrors.ErrServerInternalPendingAuth
	}

	pendingKey := PendingAuthPrefix + token
	attemptsKey := PendingAuthAttemptsPrefix + token

	userID, err := pendingAuthRedisClient.Get(ctx, pendingKey).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			logger.Logger.Warn("Cannot increment attempts for invalid pending auth token",
				"token_preview", GetKeyPreview(token),
			)
			return 0, apperrors.ErrPendingAuthNotFound
		}

		logger.Logger.Warn("Cannot increment attempts for invalid pending auth token",
			"token_preview", GetKeyPreview(token),
			"error", err.Error(),
		)
		return 0, apperrors.ErrServerInternalPendingAuth
	}

	attempts, err := pendingAuthRedisClient.Incr(ctx, attemptsKey).Result()
	if err != nil {
		logger.Logger.Error("Failed to increment pending auth attempts",
			"user_id", userID,
			"token_preview", GetKeyPreview(token),
			"error", err.Error(),
		)
		return 0, apperrors.ErrServerInternalPendingAuth
	}

	ttl, err := pendingAuthRedisClient.TTL(ctx, pendingKey).Result()
	if err != nil {
		logger.Logger.Warn("Failed to get pending auth token TTL for attempts key",
			"user_id", userID,
			"token_preview", GetKeyPreview(token),
			"error", err.Error(),
		)
	} else if ttl > 0 {
		if err := pendingAuthRedisClient.Expire(ctx, attemptsKey, ttl).Err(); err != nil {
			logger.Logger.Warn("Failed to sync attempts TTL with pending auth token",
				"user_id", userID,
				"token_preview", GetKeyPreview(token),
				"error", err.Error(),
			)
		}
	}

	remainingAttempts := PendingAuthMaxAttempts - int(attempts)
	if remainingAttempts <= 0 {
		if err := DeletePendingAuthToken(ctx, token); err != nil {
			logger.Logger.Error("Failed to revoke pending auth token after attempts exceeded",
				"user_id", userID,
				"token_preview", GetKeyPreview(token),
				"error", err.Error(),
			)
			return 0, err
		}

		logger.Logger.Warn("Pending auth attempts exceeded",
			"user_id", userID,
			"token_preview", GetKeyPreview(token),
			"max_attempts", PendingAuthMaxAttempts,
		)
		return 0, apperrors.ErrPendingAuthAttemptsExceeded
	}

	logger.Logger.Warn("Pending auth attempt recorded",
		"user_id", userID,
		"token_preview", GetKeyPreview(token),
		"attempt", attempts,
		"max_attempts", PendingAuthMaxAttempts,
		"remaining_attempts", remainingAttempts,
	)

	return remainingAttempts, nil
}

// ConsumePendingAuthToken removes pending auth token after successful TOTP verification.
func ConsumePendingAuthToken(ctx context.Context, token string) error {
	return DeletePendingAuthToken(ctx, token)
}

// DeletePendingAuthToken manually deletes a pending auth token and its attempts counter.
func DeletePendingAuthToken(ctx context.Context, token string) error {
	if pendingAuthRedisClient == nil {
		return apperrors.ErrServerInternalPendingAuth
	}

	pendingKey := PendingAuthPrefix + token
	attemptsKey := PendingAuthAttemptsPrefix + token

	err := pendingAuthRedisClient.Del(ctx, pendingKey, attemptsKey).Err()
	if err != nil {
		logger.Logger.Error("Failed to delete pending auth token from Redis",
			"token_preview", GetKeyPreview(token),
			"error", err.Error(),
		)
		return apperrors.ErrPendingAuthDeleteFailed
	}

	logger.Logger.Info("Pending auth token deleted",
		"token_preview", GetKeyPreview(token),
	)

	return nil
}

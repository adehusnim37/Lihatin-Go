package session

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStore manages sessions in Redis
type RedisStore struct {
	client *redis.Client
	prefix string // Key prefix for namespacing
}

// NewRedisStore creates a new Redis session store
func NewRedisStore(addr, password string, db int) (*RedisStore, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 5,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisStore{
		client: client,
		prefix: "session:",
	}, nil
}

// Set stores a session in Redis with TTL
func (s *RedisStore) Set(ctx context.Context, sessionID string, data *Session, ttl time.Duration) error {
	key := s.prefix + sessionID

	// Serialize session data to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	// Store in Redis with expiration
	if err := s.client.Set(ctx, key, jsonData, ttl).Err(); err != nil {
		return fmt.Errorf("failed to set session in Redis: %w", err)
	}

	return nil
}

// Get retrieves a session from Redis
func (s *RedisStore) Get(ctx context.Context, sessionID string) (*Session, error) {
	key := s.prefix + sessionID

	// Get from Redis
	data, err := s.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session from Redis: %w", err)
	}

	// Deserialize JSON to Session
	var session Session
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &session, nil
}

// Delete removes a session from Redis
func (s *RedisStore) Delete(ctx context.Context, sessionID string) error {
	key := s.prefix + sessionID

	if err := s.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete session from Redis: %w", err)
	}

	return nil
}

// Refresh extends the TTL of a session
func (s *RedisStore) Refresh(ctx context.Context, sessionID string, ttl time.Duration) error {
	key := s.prefix + sessionID

	// Check if key exists
	exists, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("failed to check session existence: %w", err)
	}
	if exists == 0 {
		return ErrSessionNotFound
	}

	// Extend TTL
	if err := s.client.Expire(ctx, key, ttl).Err(); err != nil {
		return fmt.Errorf("failed to refresh session TTL: %w", err)
	}

	return nil
}

// DeleteByUserID deletes all sessions for a specific user
func (s *RedisStore) DeleteByUserID(ctx context.Context, userID string) error {
	pattern := s.prefix + "*"

	// Scan for all session keys
	iter := s.client.Scan(ctx, 0, pattern, 0).Iterator()
	var keysToDelete []string

	for iter.Next(ctx) {
		key := iter.Val()

		// Get session data
		data, err := s.client.Get(ctx, key).Result()
		if err != nil {
			continue
		}

		// Check if session belongs to user
		var session Session
		if err := json.Unmarshal([]byte(data), &session); err != nil {
			continue
		}

		if session.UserID == userID {
			keysToDelete = append(keysToDelete, key)
		}
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("failed to scan sessions: %w", err)
	}

	// Delete all matching sessions
	if len(keysToDelete) > 0 {
		if err := s.client.Del(ctx, keysToDelete...).Err(); err != nil {
			return fmt.Errorf("failed to delete user sessions: %w", err)
		}
	}

	return nil
}

// GetTTL returns the remaining TTL for a session
func (s *RedisStore) GetTTL(ctx context.Context, sessionID string) (time.Duration, error) {
	key := s.prefix + sessionID

	ttl, err := s.client.TTL(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get session TTL: %w", err)
	}

	if ttl < 0 {
		return 0, ErrSessionExpired
	}

	return ttl, nil
}

// Close closes the Redis connection
func (s *RedisStore) Close() error {
	return s.client.Close()
}

// GetActiveSessionCount returns the number of active sessions for a user
func (s *RedisStore) GetActiveSessionCount(ctx context.Context, userID string) (int, error) {
	pattern := s.prefix + "*"
	count := 0

	iter := s.client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		data, err := s.client.Get(ctx, key).Result()
		if err != nil {
			continue
		}

		var session Session
		if err := json.Unmarshal([]byte(data), &session); err != nil {
			continue
		}

		if session.UserID == userID {
			count++
		}
	}

	if err := iter.Err(); err != nil {
		return 0, fmt.Errorf("failed to count user sessions: %w", err)
	}

	return count, nil
}

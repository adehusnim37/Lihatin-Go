package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/adehusnim37/lihatin-go/internal/pkg/auth"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const (
	apiKeyAuthRequestsPerMinute = 120
	apiKeyAuthFailuresPerWindow = 10
	apiKeyAuthFailureWindow     = 15 * time.Minute
	maxAPIKeyHeaderLength       = 256
)

// INCR and its expiration must be one operation: a crash between two Redis
// commands would otherwise leave a permanent counter.
var incrementAPIKeyAuthCounter = redis.NewScript(`
local count = redis.call('INCR', KEYS[1])
if count == 1 then redis.call('PEXPIRE', KEYS[1], ARGV[1]) end
return count
`)

type apiKeyAuthThrottle struct {
	redis *redis.Client
}

func newAPIKeyAuthThrottle() (*apiKeyAuthThrottle, error) {
	manager := GetSessionManager()
	if manager == nil || manager.GetRedisClient() == nil {
		return nil, errors.New("API key authentication limiter is unavailable")
	}
	return &apiKeyAuthThrottle{redis: manager.GetRedisClient()}, nil
}

func apiKeyAuthCounterKey(kind, ip, apiKey string) string {
	identity := ip
	if apiKey != "" {
		parts := auth.SplitAPIKey(apiKey)
		keyID := "invalid"
		if len(apiKey) <= maxAPIKeyHeaderLength && len(parts) == 2 && auth.ValidateAPIKeyIDFormat(parts[0]) {
			keyID = parts[0]
		}
		identity += ":" + keyID
	}
	sum := sha256.Sum256([]byte(identity))
	return "rate_limit:api_key_auth:" + kind + ":" + hex.EncodeToString(sum[:])
}

func (t *apiKeyAuthThrottle) increment(ctx context.Context, key string, window time.Duration) (int64, error) {
	return incrementAPIKeyAuthCounter.Run(ctx, t.redis, []string{key}, window.Milliseconds()).Int64()
}

func (t *apiKeyAuthThrottle) check(ctx context.Context, ip, apiKey string) (bool, time.Duration, error) {
	requestKey := apiKeyAuthCounterKey("requests", ip, "")
	count, err := t.increment(ctx, requestKey, time.Minute)
	if err != nil {
		return false, 0, err
	}
	if count > apiKeyAuthRequestsPerMinute {
		return false, t.retryAfter(ctx, requestKey), nil
	}

	failureKey := apiKeyAuthCounterKey("failures", ip, apiKey)
	failures, err := t.redis.Get(ctx, failureKey).Int64()
	if err != nil && err != redis.Nil {
		return false, 0, err
	}
	if failures >= apiKeyAuthFailuresPerWindow {
		return false, t.retryAfter(ctx, failureKey), nil
	}
	return true, 0, nil
}

func (t *apiKeyAuthThrottle) recordFailure(ctx context.Context, ip, apiKey string) error {
	_, err := t.increment(ctx, apiKeyAuthCounterKey("failures", ip, apiKey), apiKeyAuthFailureWindow)
	return err
}

func (t *apiKeyAuthThrottle) retryAfter(ctx context.Context, key string) time.Duration {
	ttl, err := t.redis.PTTL(ctx, key).Result()
	if err != nil || ttl <= 0 {
		return time.Second
	}
	return ttl
}

func rejectAPIKeyAuthThrottle(c *gin.Context, retryAfter time.Duration) {
	seconds := int64((retryAfter + time.Second - 1) / time.Second)
	c.Header("Retry-After", strconv.FormatInt(seconds, 10))
	c.JSON(http.StatusTooManyRequests, common.APIResponse{
		Success: false,
		Message: "Too many API key authentication attempts",
		Error:   map[string]string{"api_key": "Please wait before trying again"},
	})
	c.Abort()
}

func rejectAPIKeyAuthLimiterUnavailable(c *gin.Context, err error) {
	logger.Logger.Error("API key authentication limiter unavailable", "error", err)
	c.JSON(http.StatusServiceUnavailable, common.APIResponse{
		Success: false,
		Message: "API key authentication temporarily unavailable",
		Error:   map[string]string{"api_key": "Please try again later"},
	})
	c.Abort()
}

package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
)

const (
	defaultSecondFactorLimitPerUserIP = 8
	defaultSecondFactorLimitPerIP     = 60
	defaultSecondFactorWindowMinutes  = 15
)

// EnforceSecondFactorRiskLimit applies Redis-backed throttling for second-factor verification.
// It enforces both per-user+IP and per-IP limits to reduce credential stuffing and OTP brute force.
func EnforceSecondFactorRiskLimit(ctx context.Context, action, userID, clientIP string) (blocked bool, retryAfterSeconds int, err error) {
	if pendingAuthRedisClient == nil {
		return false, 0, ErrEmailOTPServiceUnavailable
	}

	action = strings.TrimSpace(action)
	if action == "" {
		action = "generic"
	}

	userID = strings.TrimSpace(userID)
	if userID == "" {
		userID = "unknown"
	}

	clientIP = strings.TrimSpace(clientIP)
	if clientIP == "" {
		clientIP = "unknown"
	}

	windowMinutes := config.GetEnvAsInt(config.EnvAuthSecondFactorWindowMinutes, defaultSecondFactorWindowMinutes)
	if windowMinutes <= 0 {
		windowMinutes = defaultSecondFactorWindowMinutes
	}
	window := time.Duration(windowMinutes) * time.Minute

	userIPLimit := config.GetEnvAsInt(config.EnvAuthSecondFactorLimitPerUserIP, defaultSecondFactorLimitPerUserIP)
	if userIPLimit <= 0 {
		userIPLimit = defaultSecondFactorLimitPerUserIP
	}
	ipLimit := config.GetEnvAsInt(config.EnvAuthSecondFactorLimitPerIP, defaultSecondFactorLimitPerIP)
	if ipLimit <= 0 {
		ipLimit = defaultSecondFactorLimitPerIP
	}

	perUserIPKey := fmt.Sprintf("2fa_risk:%s:user:%s:ip:%s", action, userID, clientIP)
	perIPKey := fmt.Sprintf("2fa_risk:%s:ip:%s", action, clientIP)

	userBlocked, userRetry, err := incrementAndCheckLimit(ctx, perUserIPKey, userIPLimit, window)
	if err != nil {
		return false, 0, err
	}

	ipBlocked, ipRetry, err := incrementAndCheckLimit(ctx, perIPKey, ipLimit, window)
	if err != nil {
		return false, 0, err
	}

	if userBlocked || ipBlocked {
		if ipRetry > userRetry {
			return true, ipRetry, nil
		}
		return true, userRetry, nil
	}

	return false, 0, nil
}

func incrementAndCheckLimit(ctx context.Context, key string, limit int, window time.Duration) (blocked bool, retryAfterSeconds int, err error) {
	count, err := pendingAuthRedisClient.Incr(ctx, key).Result()
	if err != nil {
		return false, 0, err
	}

	if count == 1 {
		if err := pendingAuthRedisClient.Expire(ctx, key, window).Err(); err != nil {
			return false, 0, err
		}
	}

	if count <= int64(limit) {
		return false, 0, nil
	}

	ttl, err := pendingAuthRedisClient.TTL(ctx, key).Result()
	if err != nil {
		return true, 0, nil
	}

	if ttl <= 0 {
		return true, 1, nil
	}

	retryAfter := int(ttl.Seconds())
	if retryAfter < 1 {
		retryAfter = 1
	}

	return true, retryAfter, nil
}

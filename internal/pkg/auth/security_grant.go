package auth

import (
	"context"
	"github.com/redis/go-redis/v9"
)

// ConsumeSecurityGrant ties a recent verification to one action in the same account session.
func ConsumeSecurityGrant(ctx context.Context, client *redis.Client, userID, sessionID, token string) bool {
	if client == nil || len(token) != 64 || userID == "" || sessionID == "" {
		return false
	}
	value, err := client.GetDel(ctx, "security_grant:"+token).Result()
	return err == nil && (value == userID+":"+sessionID || value == userID+":"+sessionID+":mfa")
}

// ConsumeMFASecurityGrant rejects password-only and legacy proofs for admin recovery.
func ConsumeMFASecurityGrant(ctx context.Context, client *redis.Client, userID, sessionID, token string) bool {
	if client == nil || len(token) != 64 || userID == "" || sessionID == "" {
		return false
	}
	value, err := client.GetDel(ctx, "security_grant:"+token).Result()
	return err == nil && value == userID+":"+sessionID+":mfa"
}

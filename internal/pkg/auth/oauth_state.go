package auth

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	googleOAuthStatePrefix = "oauth_google_state:"
	GoogleOAuthStateTTL    = 10 * time.Minute
)

var (
	ErrOAuthStateServiceUnavailable = errors.New("oauth state service unavailable")
	ErrOAuthStateNotFound           = errors.New("oauth state not found")
)

type GoogleOAuthIntent string

const (
	GoogleOAuthIntentLogin  GoogleOAuthIntent = "login"
	GoogleOAuthIntentSignup GoogleOAuthIntent = "signup"
)

type GoogleOAuthStatePayload struct {
	Intent    GoogleOAuthIntent `json:"intent"`
	CreatedAt int64             `json:"created_at"`
}

func googleOAuthStateKey(state string) string {
	return googleOAuthStatePrefix + state
}

func normalizeGoogleOAuthIntent(intent GoogleOAuthIntent) GoogleOAuthIntent {
	switch intent {
	case GoogleOAuthIntentSignup:
		return GoogleOAuthIntentSignup
	default:
		return GoogleOAuthIntentLogin
	}
}

func StoreGoogleOAuthState(ctx context.Context, state string, intent GoogleOAuthIntent) error {
	if pendingAuthRedisClient == nil {
		return ErrOAuthStateServiceUnavailable
	}

	payload := GoogleOAuthStatePayload{
		Intent:    normalizeGoogleOAuthIntent(intent),
		CreatedAt: time.Now().Unix(),
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return pendingAuthRedisClient.Set(ctx, googleOAuthStateKey(state), encoded, GoogleOAuthStateTTL).Err()
}

func GenerateAndStoreGoogleOAuthState(ctx context.Context, intent GoogleOAuthIntent) (string, error) {
	state, err := GenerateSecureToken(24)
	if err != nil {
		return "", err
	}

	if err := StoreGoogleOAuthState(ctx, state, intent); err != nil {
		return "", err
	}

	return state, nil
}

func ConsumeGoogleOAuthState(ctx context.Context, state string) (*GoogleOAuthStatePayload, error) {
	if pendingAuthRedisClient == nil {
		return nil, ErrOAuthStateServiceUnavailable
	}

	key := googleOAuthStateKey(state)
	raw, err := pendingAuthRedisClient.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrOAuthStateNotFound
		}
		return nil, err
	}

	if err := pendingAuthRedisClient.Del(ctx, key).Err(); err != nil {
		return nil, err
	}

	var payload GoogleOAuthStatePayload
	if unmarshalErr := json.Unmarshal([]byte(raw), &payload); unmarshalErr != nil {
		return &GoogleOAuthStatePayload{
			Intent: GoogleOAuthIntentLogin,
		}, nil
	}

	payload.Intent = normalizeGoogleOAuthIntent(payload.Intent)
	return &payload, nil
}

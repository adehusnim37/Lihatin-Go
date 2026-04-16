package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/redis/go-redis/v9"
)

const (
	EmailOTPCodeLength          = 6
	EmailOTPChallengeTTL        = 10 * time.Minute
	EmailOTPMaxAttempts         = 5
	SignupCompletionTokenTTL    = 20 * time.Minute
	emailOTPChallengePrefix     = "email_otp_challenge:"
	signupCompletionTokenPrefix = "signup_completion_token:"
	signupCompletionEmailPrefix = "signup_completion_email:"
)

type EmailOTPPurpose string

const (
	EmailOTPPurposeSignup EmailOTPPurpose = "signup"
	EmailOTPPurposeLogin  EmailOTPPurpose = "login"
)

var (
	ErrEmailOTPServiceUnavailable   = errors.New("email otp service unavailable")
	ErrEmailOTPChallengeNotFound    = errors.New("email otp challenge not found")
	ErrEmailOTPChallengeExpired     = errors.New("email otp challenge expired")
	ErrEmailOTPInvalidCode          = errors.New("invalid email otp code")
	ErrEmailOTPAttemptsExceeded     = errors.New("email otp attempts exceeded")
	ErrEmailOTPPurposeMismatch      = errors.New("email otp purpose mismatch")
	ErrSignupCompletionTokenInvalid = errors.New("signup completion token invalid")
)

type EmailOTPCooldownError struct {
	RemainingSeconds int
}

func (e *EmailOTPCooldownError) Error() string {
	return fmt.Sprintf("email otp resend cooldown active: %ds", e.RemainingSeconds)
}

type EmailOTPInvalidCodeError struct {
	RemainingAttempts int
}

func (e *EmailOTPInvalidCodeError) Error() string {
	return fmt.Sprintf("invalid email otp code, remaining attempts: %d", e.RemainingAttempts)
}

type EmailOTPChallenge struct {
	Purpose     EmailOTPPurpose `json:"purpose"`
	Email       string          `json:"email"`
	UserID      string          `json:"user_id,omitempty"`
	CodeHash    string          `json:"code_hash"`
	ExpiresAt   int64           `json:"expires_at"`
	LastSentAt  int64           `json:"last_sent_at"`
	ResendCount int             `json:"resend_count"`
	Attempts    int             `json:"attempts"`
}

func emailOTPClient() (*redis.Client, error) {
	if pendingAuthRedisClient == nil {
		return nil, ErrEmailOTPServiceUnavailable
	}
	return pendingAuthRedisClient, nil
}

func emailOTPChallengeKey(token string) string {
	return emailOTPChallengePrefix + token
}

func signupCompletionTokenKey(token string) string {
	return signupCompletionTokenPrefix + token
}

func normalizeEmailForKey(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func signupCompletionEmailKey(email string) string {
	return signupCompletionEmailPrefix + normalizeEmailForKey(email)
}

func HashEmailOTPCode(code string) string {
	hash := sha256.Sum256([]byte(code))
	return hex.EncodeToString(hash[:])
}

func GenerateEmailOTPCode() (string, error) {
	max := big.NewInt(1000000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func GenerateEmailOTPChallenge(
	ctx context.Context,
	purpose EmailOTPPurpose,
	email string,
	userID string,
) (token string, code string, challenge *EmailOTPChallenge, err error) {
	client, err := emailOTPClient()
	if err != nil {
		return "", "", nil, err
	}

	token, err = GenerateSecureToken(24)
	if err != nil {
		return "", "", nil, err
	}

	code, err = GenerateEmailOTPCode()
	if err != nil {
		return "", "", nil, err
	}

	now := time.Now()
	challenge = &EmailOTPChallenge{
		Purpose:     purpose,
		Email:       email,
		UserID:      userID,
		CodeHash:    HashEmailOTPCode(code),
		ExpiresAt:   now.Add(EmailOTPChallengeTTL).Unix(),
		LastSentAt:  now.Unix(),
		ResendCount: 0,
		Attempts:    0,
	}

	if err := saveEmailOTPChallenge(ctx, client, token, challenge); err != nil {
		return "", "", nil, err
	}

	return token, code, challenge, nil
}

func GetEmailOTPChallenge(ctx context.Context, token string) (*EmailOTPChallenge, error) {
	client, err := emailOTPClient()
	if err != nil {
		return nil, err
	}

	raw, err := client.Get(ctx, emailOTPChallengeKey(token)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrEmailOTPChallengeNotFound
		}
		return nil, err
	}

	var challenge EmailOTPChallenge
	if err := json.Unmarshal([]byte(raw), &challenge); err != nil {
		return nil, err
	}

	if time.Now().After(time.Unix(challenge.ExpiresAt, 0)) {
		_ = DeleteEmailOTPChallenge(ctx, token)
		return nil, ErrEmailOTPChallengeExpired
	}

	return &challenge, nil
}

func SaveEmailOTPChallenge(ctx context.Context, token string, challenge *EmailOTPChallenge) error {
	client, err := emailOTPClient()
	if err != nil {
		return err
	}
	return saveEmailOTPChallenge(ctx, client, token, challenge)
}

func saveEmailOTPChallenge(ctx context.Context, client *redis.Client, token string, challenge *EmailOTPChallenge) error {
	payload, err := json.Marshal(challenge)
	if err != nil {
		return err
	}

	ttl := time.Until(time.Unix(challenge.ExpiresAt, 0))
	if ttl <= 0 {
		ttl = time.Second
	}

	return client.Set(ctx, emailOTPChallengeKey(token), payload, ttl).Err()
}

func DeleteEmailOTPChallenge(ctx context.Context, token string) error {
	client, err := emailOTPClient()
	if err != nil {
		return err
	}
	return client.Del(ctx, emailOTPChallengeKey(token)).Err()
}

func getEmailOTPResendCooldown(resendCount int) time.Duration {
	if resendCount >= 1 {
		return 5 * time.Minute
	}
	return 1 * time.Minute
}

func GetEmailOTPResendRemainingSeconds(challenge *EmailOTPChallenge, now time.Time) int {
	lastSent := time.Unix(challenge.LastSentAt, 0)
	cooldown := getEmailOTPResendCooldown(challenge.ResendCount)
	elapsed := now.Sub(lastSent)
	if elapsed >= cooldown {
		return 0
	}
	remaining := int((cooldown - elapsed).Seconds())
	if remaining < 1 {
		return 1
	}
	return remaining
}

func ResendEmailOTPChallenge(ctx context.Context, token string) (challenge *EmailOTPChallenge, code string, err error) {
	challenge, err = GetEmailOTPChallenge(ctx, token)
	if err != nil {
		return nil, "", err
	}

	remaining := GetEmailOTPResendRemainingSeconds(challenge, time.Now())
	if remaining > 0 {
		return nil, "", &EmailOTPCooldownError{RemainingSeconds: remaining}
	}

	code, err = GenerateEmailOTPCode()
	if err != nil {
		return nil, "", err
	}

	challenge.CodeHash = HashEmailOTPCode(code)
	challenge.LastSentAt = time.Now().Unix()
	challenge.ResendCount++
	challenge.Attempts = 0

	if err := SaveEmailOTPChallenge(ctx, token, challenge); err != nil {
		return nil, "", err
	}

	return challenge, code, nil
}

func VerifyEmailOTPChallenge(ctx context.Context, token string, otpCode string, purpose EmailOTPPurpose) (*EmailOTPChallenge, error) {
	challenge, err := GetEmailOTPChallenge(ctx, token)
	if err != nil {
		return nil, err
	}

	if challenge.Purpose != purpose {
		return nil, ErrEmailOTPPurposeMismatch
	}

	hash := HashEmailOTPCode(otpCode)
	if subtle.ConstantTimeCompare([]byte(challenge.CodeHash), []byte(hash)) != 1 {
		challenge.Attempts++
		if challenge.Attempts >= EmailOTPMaxAttempts {
			_ = DeleteEmailOTPChallenge(ctx, token)
			return nil, ErrEmailOTPAttemptsExceeded
		}

		if err := SaveEmailOTPChallenge(ctx, token, challenge); err != nil {
			logger.Logger.Warn("Failed to persist failed email OTP attempt",
				"error", err.Error(),
			)
		}

		remainingAttempts := EmailOTPMaxAttempts - challenge.Attempts
		return nil, &EmailOTPInvalidCodeError{RemainingAttempts: remainingAttempts}
	}

	if err := DeleteEmailOTPChallenge(ctx, token); err != nil {
		logger.Logger.Warn("Failed to consume email OTP challenge",
			"error", err.Error(),
		)
	}

	return challenge, nil
}

func CreateSignupCompletionToken(ctx context.Context, email string) (string, error) {
	client, err := emailOTPClient()
	if err != nil {
		return "", err
	}

	normalizedEmail := normalizeEmailForKey(email)
	existingToken, err := client.Get(ctx, signupCompletionEmailKey(normalizedEmail)).Result()
	switch {
	case err == nil:
		if existingToken != "" {
			_ = client.Del(ctx, signupCompletionTokenKey(existingToken)).Err()
		}
	case errors.Is(err, redis.Nil):
		// No existing pending signup token.
	default:
		return "", err
	}

	token, err := GenerateSecureToken(24)
	if err != nil {
		return "", err
	}

	pipe := client.TxPipeline()
	pipe.Set(ctx, signupCompletionTokenKey(token), email, SignupCompletionTokenTTL)
	pipe.Set(ctx, signupCompletionEmailKey(normalizedEmail), token, SignupCompletionTokenTTL)

	if _, err := pipe.Exec(ctx); err != nil {
		return "", err
	}

	return token, nil
}

func GetSignupCompletionTokenEmail(ctx context.Context, token string) (string, error) {
	client, err := emailOTPClient()
	if err != nil {
		return "", err
	}

	key := signupCompletionTokenKey(token)
	email, err := client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", ErrSignupCompletionTokenInvalid
		}
		return "", err
	}

	return email, nil
}

func GetSignupCompletionTokenByEmail(ctx context.Context, email string) (string, error) {
	client, err := emailOTPClient()
	if err != nil {
		return "", err
	}

	key := signupCompletionEmailKey(email)
	token, err := client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", ErrSignupCompletionTokenInvalid
		}
		return "", err
	}

	_, err = client.Get(ctx, signupCompletionTokenKey(token)).Result()
	if err == nil {
		return token, nil
	}
	if errors.Is(err, redis.Nil) {
		_ = client.Del(ctx, key).Err()
		return "", ErrSignupCompletionTokenInvalid
	}

	return "", err
}

func DeleteSignupCompletionToken(ctx context.Context, token string) error {
	client, err := emailOTPClient()
	if err != nil {
		return err
	}

	tokenKey := signupCompletionTokenKey(token)
	email, getErr := client.Get(ctx, tokenKey).Result()
	if getErr != nil && !errors.Is(getErr, redis.Nil) {
		return getErr
	}

	pipe := client.TxPipeline()
	pipe.Del(ctx, tokenKey)
	if getErr == nil && email != "" {
		pipe.Del(ctx, signupCompletionEmailKey(email))
	}

	_, err = pipe.Exec(ctx)
	return err
}

func ConsumeSignupCompletionToken(ctx context.Context, token string) (string, error) {
	email, err := GetSignupCompletionTokenEmail(ctx, token)
	if err != nil {
		return "", err
	}

	if err := DeleteSignupCompletionToken(ctx, token); err != nil {
		logger.Logger.Warn("Failed to consume signup completion token",
			"token_preview", GetKeyPreview(token),
			"error", err.Error(),
		)
	}

	return email, nil
}

func CooldownSecondsForNextResend(challenge *EmailOTPChallenge) int {
	if challenge == nil {
		return 60
	}
	return int(getEmailOTPResendCooldown(challenge.ResendCount).Seconds())
}

func ParseOTPCode(input string) string {
	if len(input) == 0 {
		return ""
	}
	if len(input) > EmailOTPCodeLength {
		input = input[:EmailOTPCodeLength]
	}
	_, err := strconv.Atoi(input)
	if err != nil {
		return ""
	}
	if len(input) != EmailOTPCodeLength {
		return ""
	}
	return input
}

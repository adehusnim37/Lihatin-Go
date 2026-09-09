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
	SupportAccessTokenTTL       = 24 * time.Hour
	emailOTPChallengePrefix     = "email_otp_challenge:"
	signupCompletionTokenPrefix = "signup_completion_token:"
	signupCompletionEmailPrefix = "signup_completion_email:"
	supportAccessTokenPrefix    = "support_access_token:"
	supportAccessTicketPrefix   = "support_access_ticket:"
)

type EmailOTPPurpose string

const (
	EmailOTPPurposeSignup        EmailOTPPurpose = "signup"
	EmailOTPPurposeLogin         EmailOTPPurpose = "login"
	EmailOTPPurposeSupportAccess EmailOTPPurpose = "support_access"
)

var (
	ErrEmailOTPServiceUnavailable   = errors.New("email otp service unavailable")
	ErrEmailOTPChallengeNotFound    = errors.New("email otp challenge not found")
	ErrEmailOTPChallengeExpired     = errors.New("email otp challenge expired")
	ErrEmailOTPInvalidCode          = errors.New("invalid email otp code")
	ErrEmailOTPAttemptsExceeded     = errors.New("email otp attempts exceeded")
	ErrEmailOTPPurposeMismatch      = errors.New("email otp purpose mismatch")
	ErrSignupCompletionTokenInvalid = errors.New("signup completion token invalid")
	ErrSupportAccessTokenInvalid    = errors.New("support access token invalid")
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
	IsDecoy     bool            `json:"is_decoy,omitempty"`
	CodeHash    string          `json:"code_hash"`
	ExpiresAt   int64           `json:"expires_at"`
	LastSentAt  int64           `json:"last_sent_at"`
	ResendCount int             `json:"resend_count"`
	Attempts    int             `json:"attempts"`
}

type SupportAccessTokenPayload struct {
	TicketID      string `json:"ticket_id"`
	TicketCode    string `json:"ticket_code"`
	Email         string `json:"email"`
	AccessVersion string `json:"access_version"`
	ExpiresAt     int64  `json:"expires_at"`
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

func supportAccessTokenKey(token string) string {
	return supportAccessTokenPrefix + token
}

func supportAccessTicketKey(ticketID string) string {
	return supportAccessTicketPrefix + strings.TrimSpace(ticketID)
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
	client, err := emailOTPClient()
	if err != nil {
		return nil, err
	}

	key := emailOTPChallengeKey(strings.TrimSpace(token))
	providedHash := HashEmailOTPCode(otpCode)
	var verified *EmailOTPChallenge

	// WATCH makes verification and challenge consumption atomic. Concurrent
	// requests cannot both redeem the same OTP or lose failed-attempt updates.
	for attempt := 0; attempt < 8; attempt++ {
		err = client.Watch(ctx, func(tx *redis.Tx) error {
			raw, getErr := tx.Get(ctx, key).Result()
			if getErr != nil {
				if errors.Is(getErr, redis.Nil) {
					return ErrEmailOTPChallengeNotFound
				}
				return getErr
			}

			var challenge EmailOTPChallenge
			if unmarshalErr := json.Unmarshal([]byte(raw), &challenge); unmarshalErr != nil {
				return unmarshalErr
			}
			if time.Now().After(time.Unix(challenge.ExpiresAt, 0)) {
				pipe := tx.TxPipeline()
				pipe.Del(ctx, key)
				if _, execErr := pipe.Exec(ctx); execErr != nil {
					return execErr
				}
				return ErrEmailOTPChallengeExpired
			}
			if challenge.Purpose != purpose {
				return ErrEmailOTPPurposeMismatch
			}

			pipe := tx.TxPipeline()
			if subtle.ConstantTimeCompare([]byte(challenge.CodeHash), []byte(providedHash)) != 1 {
				challenge.Attempts++
				if challenge.Attempts >= EmailOTPMaxAttempts {
					pipe.Del(ctx, key)
					if _, execErr := pipe.Exec(ctx); execErr != nil {
						return execErr
					}
					return ErrEmailOTPAttemptsExceeded
				}

				payload, marshalErr := json.Marshal(&challenge)
				if marshalErr != nil {
					return marshalErr
				}
				ttl := time.Until(time.Unix(challenge.ExpiresAt, 0))
				if ttl <= 0 {
					ttl = time.Second
				}
				pipe.Set(ctx, key, payload, ttl)
				if _, execErr := pipe.Exec(ctx); execErr != nil {
					return execErr
				}
				return &EmailOTPInvalidCodeError{RemainingAttempts: EmailOTPMaxAttempts - challenge.Attempts}
			}

			pipe.Del(ctx, key)
			if _, execErr := pipe.Exec(ctx); execErr != nil {
				return execErr
			}
			verified = &challenge
			return nil
		}, key)
		if !errors.Is(err, redis.TxFailedErr) {
			return verified, err
		}
	}

	return nil, redis.TxFailedErr
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

func CreateSupportAccessToken(ctx context.Context, ticketID, ticketCode, email, accessVersion string) (string, *SupportAccessTokenPayload, error) {
	client, err := emailOTPClient()
	if err != nil {
		return "", nil, err
	}

	token, err := GenerateSecureToken(24)
	if err != nil {
		return "", nil, err
	}

	payload := &SupportAccessTokenPayload{
		TicketID:      strings.TrimSpace(ticketID),
		TicketCode:    strings.TrimSpace(ticketCode),
		Email:         normalizeEmailForKey(email),
		AccessVersion: strings.TrimSpace(accessVersion),
		ExpiresAt:     time.Now().Add(SupportAccessTokenTTL).Unix(),
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return "", nil, err
	}

	pipe := client.TxPipeline()
	pipe.Set(ctx, supportAccessTokenKey(token), raw, SupportAccessTokenTTL)
	pipe.SAdd(ctx, supportAccessTicketKey(payload.TicketID), token)
	pipe.Expire(ctx, supportAccessTicketKey(payload.TicketID), SupportAccessTokenTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return "", nil, err
	}

	return token, payload, nil
}

func GetSupportAccessToken(ctx context.Context, token string) (*SupportAccessTokenPayload, error) {
	client, err := emailOTPClient()
	if err != nil {
		return nil, err
	}

	raw, err := client.Get(ctx, supportAccessTokenKey(strings.TrimSpace(token))).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrSupportAccessTokenInvalid
		}
		return nil, err
	}

	var payload SupportAccessTokenPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, err
	}

	if payload.ExpiresAt <= time.Now().Unix() {
		_ = client.Del(ctx, supportAccessTokenKey(strings.TrimSpace(token))).Err()
		return nil, ErrSupportAccessTokenInvalid
	}

	return &payload, nil
}

// DeleteSupportAccessToken removes one browser/API session and its ticket index entry.
func DeleteSupportAccessToken(ctx context.Context, token string) error {
	client, err := emailOTPClient()
	if err != nil {
		return err
	}

	normalizedToken := strings.TrimSpace(token)
	if normalizedToken == "" {
		return nil
	}

	payload, payloadErr := GetSupportAccessToken(ctx, normalizedToken)
	pipe := client.TxPipeline()
	pipe.Del(ctx, supportAccessTokenKey(normalizedToken))
	if payloadErr == nil && payload != nil && strings.TrimSpace(payload.TicketID) != "" {
		pipe.SRem(ctx, supportAccessTicketKey(payload.TicketID), normalizedToken)
	}
	_, err = pipe.Exec(ctx)
	return err
}

// RevokeSupportAccessTokensForTicket invalidates every outstanding public
// support session when a ticket crosses the closed/resolved boundary.
func RevokeSupportAccessTokensForTicket(ctx context.Context, ticketID string) error {
	client, err := emailOTPClient()
	if err != nil {
		return err
	}

	indexKey := supportAccessTicketKey(ticketID)
	tokens, err := client.SMembers(ctx, indexKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return err
	}

	keys := make([]string, 0, len(tokens)+1)
	for _, token := range tokens {
		if normalized := strings.TrimSpace(token); normalized != "" {
			keys = append(keys, supportAccessTokenKey(normalized))
		}
	}
	keys = append(keys, indexKey)
	return client.Del(ctx, keys...).Err()
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

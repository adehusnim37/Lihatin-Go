package premium

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/blake2s"
)

const (
	codeVersion  byte = 1
	payloadSize       = 10 // version(1) + expiry_unix(4) + nonce(5)
	tagSize           = 10 // 80-bit tag
	totalRawSize      = payloadSize + tagSize

	redeemedPrefix = "premium:redeemed:"
)

var (
	ErrSecretKeyMissing = errors.New("premium code secret key is missing")
	ErrCodeFormat       = errors.New("invalid secret code format")
	ErrCodeSignature    = errors.New("invalid secret code signature")
	ErrCodeExpired      = errors.New("secret code is expired")
	ErrCodeUsed         = errors.New("secret code has already been used")
	ErrRedisUnavailable = errors.New("redis client is unavailable")
)

var base32Encoding = base32.StdEncoding.WithPadding(base32.NoPadding)

func loadSecretKey() ([]byte, error) {
	secret := strings.TrimSpace(config.GetEnvOrDefault(config.EnvPremiumCodeSecret, ""))
	if secret == "" {
		return nil, ErrSecretKeyMissing
	}

	var secretMaterial []byte
	if decoded, err := base64.StdEncoding.DecodeString(secret); err == nil && len(decoded) > 0 {
		secretMaterial = decoded
	} else {
		secretMaterial = []byte(secret)
	}

	if len(secretMaterial) < 16 {
		return nil, fmt.Errorf("%w: minimum 16 chars", ErrSecretKeyMissing)
	}

	derived := sha256.Sum256(secretMaterial)
	return derived[:], nil
}

func normalizeCode(code string) string {
	code = strings.ToUpper(code)
	code = strings.ReplaceAll(code, "-", "")
	code = strings.ReplaceAll(code, " ", "")
	return code
}

func computeTag(secret, payload []byte) ([]byte, error) {
	mac, err := blake2s.New256(secret)
	if err != nil {
		return nil, err
	}
	_, _ = mac.Write(payload)
	sum := mac.Sum(nil)
	return sum[:tagSize], nil
}

func BuildSecretCode(validUntil time.Time) (string, error) {
	secret, err := loadSecretKey()
	if err != nil {
		return "", err
	}

	payload := make([]byte, payloadSize)
	payload[0] = codeVersion
	binary.BigEndian.PutUint32(payload[1:5], uint32(validUntil.Unix()))
	if _, err := rand.Read(payload[5:]); err != nil {
		return "", err
	}

	tag, err := computeTag(secret, payload)
	if err != nil {
		return "", err
	}

	raw := make([]byte, 0, totalRawSize)
	raw = append(raw, payload...)
	raw = append(raw, tag...)
	return base32Encoding.EncodeToString(raw), nil
}

func verifyCode(code string, now time.Time) (time.Time, string, error) {
	secret, err := loadSecretKey()
	if err != nil {
		return time.Time{}, "", err
	}

	normalized := normalizeCode(code)
	raw, err := base32Encoding.DecodeString(normalized)
	if err != nil || len(raw) != totalRawSize {
		return time.Time{}, "", ErrCodeFormat
	}

	payload := raw[:payloadSize]
	receivedTag := raw[payloadSize:]

	if payload[0] != codeVersion {
		return time.Time{}, "", ErrCodeFormat
	}

	expectedTag, err := computeTag(secret, payload)
	if err != nil {
		return time.Time{}, "", err
	}
	if subtle.ConstantTimeCompare(receivedTag, expectedTag) != 1 {
		return time.Time{}, "", ErrCodeSignature
	}

	expiry := time.Unix(int64(binary.BigEndian.Uint32(payload[1:5])), 0).UTC()
	if now.UTC().After(expiry) {
		return time.Time{}, "", ErrCodeExpired
	}

	digest := sha256.Sum256([]byte(normalized))
	return expiry, hex.EncodeToString(digest[:]), nil
}

func RedeemOneTimeCode(ctx context.Context, redisClient *redis.Client, code, owner string, now time.Time) (string, time.Time, error) {
	if redisClient == nil {
		return "", time.Time{}, ErrRedisUnavailable
	}

	expiry, digest, err := verifyCode(code, now)
	if err != nil {
		return "", time.Time{}, err
	}

	ttl := time.Until(expiry) + (30 * 24 * time.Hour)
	if ttl < time.Hour {
		ttl = time.Hour
	}

	key := redeemedPrefix + digest
	ok, err := redisClient.SetNX(ctx, key, owner, ttl).Result()
	if err != nil {
		return "", time.Time{}, err
	}
	if !ok {
		return "", time.Time{}, ErrCodeUsed
	}

	return key, expiry, nil
}

func MarkRedeemedOwner(ctx context.Context, redisClient *redis.Client, key, owner string) error {
	if redisClient == nil || key == "" {
		return nil
	}

	ttl, err := redisClient.TTL(ctx, key).Result()
	if err != nil {
		return err
	}
	if ttl <= 0 {
		ttl = time.Hour
	}

	return redisClient.Set(ctx, key, owner, ttl).Err()
}

func ReleaseReservation(ctx context.Context, redisClient *redis.Client, key string) error {
	if redisClient == nil || key == "" {
		return nil
	}
	return redisClient.Del(ctx, key).Err()
}

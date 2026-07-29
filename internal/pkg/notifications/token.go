package notifications

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"

	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
)

var errInvalidUnsubscribeToken = errors.New("invalid unsubscribe token")

func IsSupportedCategory(category string) bool {
	return category == "weekly_summary" || category == "promotional"
}

// GenerateUnsubscribeToken creates a stateless, signed token that contains no
// email address and remains valid until JWT_SECRET is rotated.
func GenerateUnsubscribeToken(userID, category string) (string, error) {
	userID = strings.TrimSpace(userID)
	category = strings.TrimSpace(category)
	if userID == "" || !IsSupportedCategory(category) {
		return "", errInvalidUnsubscribeToken
	}

	userPart := base64.RawURLEncoding.EncodeToString([]byte(userID))
	categoryPart := base64.RawURLEncoding.EncodeToString([]byte(category))
	signature := sign(userID, category)

	return userPart + "." + categoryPart + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func VerifyUnsubscribeToken(token string) (userID, category string, err error) {
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 3 {
		return "", "", errInvalidUnsubscribeToken
	}

	userBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", "", errInvalidUnsubscribeToken
	}
	categoryBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", "", errInvalidUnsubscribeToken
	}
	providedSignature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return "", "", errInvalidUnsubscribeToken
	}

	userID = string(userBytes)
	category = string(categoryBytes)
	if strings.TrimSpace(userID) == "" || !IsSupportedCategory(category) {
		return "", "", errInvalidUnsubscribeToken
	}

	expectedSignature := sign(userID, category)
	if !hmac.Equal(providedSignature, expectedSignature) {
		return "", "", errInvalidUnsubscribeToken
	}

	return userID, category, nil
}

func sign(userID, category string) []byte {
	secret := []byte(config.GetRequiredEnv(config.EnvJWTSecret))
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(userID))
	_, _ = mac.Write([]byte{0})
	_, _ = mac.Write([]byte(category))
	return mac.Sum(nil)
}

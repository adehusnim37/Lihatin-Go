package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base32"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// TOTPConfig holds TOTP configuration
type TOTPConfig struct {
	Issuer      string
	AccountName string
	Secret      string
}

// GenerateTOTPSecret generates a new TOTP secret
func GenerateTOTPSecret(issuer, accountName string) (*TOTPConfig, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: accountName,
		SecretSize:  32,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate TOTP secret: %w", err)
	}

	return &TOTPConfig{
		Issuer:      issuer,
		AccountName: accountName,
		Secret:      key.Secret(),
	}, nil
}

// GenerateQRCodeURL generates a QR code URL for TOTP setup
func GenerateQRCodeURL(config *TOTPConfig) string {
	// Generate OTP Auth URL
	v := url.Values{}
	v.Set("secret", config.Secret)
	v.Set("issuer", config.Issuer)

	return fmt.Sprintf("otpauth://totp/%s:%s?%s",
		url.QueryEscape(config.Issuer),
		url.QueryEscape(config.AccountName),
		v.Encode())
}

// ValidateTOTPCode validates a TOTP code against a secret
func ValidateTOTPCode(secret, code string) bool {
	return totp.Validate(code, secret)
}

// ValidateTOTPCodeWithWindow validates a TOTP code with time window tolerance
func ValidateTOTPCodeWithWindow(secret, code string, window uint) bool {
	opts := totp.ValidateOpts{
		Period:    30,
		Skew:      window,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	}

	valid, err := totp.ValidateCustom(code, secret, time.Now(), opts)
	return err == nil && valid
}

// GenerateRecoveryCodes generates backup recovery codes
func GenerateRecoveryCodes(count int) ([]string, error) {
	codes := make([]string, count)

	for i := 0; i < count; i++ {
		code, err := generateRandomCode(8)
		if err != nil {
			return nil, fmt.Errorf("failed to generate recovery code: %w", err)
		}
		codes[i] = code
	}

	return codes, nil
}

// ValidateRecoveryCode validates a recovery code against stored hashed codes
func ValidateRecoveryCode(inputCode string, hashedCodes []string) bool {
	inputHash, err := HashPassword(inputCode)
	if err != nil {
		return false
	}

	for _, hashedCode := range hashedCodes {
		if subtle.ConstantTimeCompare([]byte(inputHash), []byte(hashedCode)) == 1 {
			return true
		}
	}

	return false
}

// HashRecoveryCodes hashes recovery codes for storage
func HashRecoveryCodes(codes []string) []string {
	hashedCodes := make([]string, len(codes))

	for i, code := range codes {
		hash, err := HashPassword(code)
		if err != nil {
			// Log error but continue - in production you might want to handle this differently
			continue
		}
		hashedCodes[i] = hash
	}

	return hashedCodes
}

// generateRandomCode generates a random alphanumeric code
func generateRandomCode(length int) (string, error) {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)

	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	for i := range b {
		b[i] = charset[b[i]%byte(len(charset))]
	}

	// Format as groups of 4 characters
	code := string(b)
	if length == 8 {
		return code[:4] + "-" + code[4:], nil
	}

	return code, nil
}

// EncryptTOTPSecret encrypts TOTP secret for database storage
func EncryptTOTPSecret(secret string) (string, error) {
	// For simplicity, using base32 encoding
	// In production, use proper encryption like AES
	encoded := base32.StdEncoding.EncodeToString([]byte(secret))
	return encoded, nil
}

// DecryptTOTPSecret decrypts TOTP secret from database
func DecryptTOTPSecret(encryptedSecret string) (string, error) {
	// For simplicity, using base32 decoding
	// In production, use proper decryption like AES
	decoded, err := base32.StdEncoding.DecodeString(encryptedSecret)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt TOTP secret: %w", err)
	}
	return string(decoded), nil
}

// RemoveRecoveryCode removes a used recovery code from the slice
func RemoveRecoveryCode(codes []string, usedCode string) []string {
	usedHash, err := HashPassword(usedCode)
	if err != nil {
		return codes // Return original codes if hashing fails
	}

	var updatedCodes []string

	for _, hashedCode := range codes {
		if subtle.ConstantTimeCompare([]byte(usedHash), []byte(hashedCode)) != 1 {
			updatedCodes = append(updatedCodes, hashedCode)
		}
	}

	return updatedCodes
}

// FormatTOTPSecret formats a secret for display (shows first 4 and last 4 chars)
func FormatTOTPSecret(secret string) string {
	if len(secret) <= 8 {
		return strings.Repeat("*", len(secret))
	}
	return secret[:4] + strings.Repeat("*", len(secret)-8) + secret[len(secret)-4:]
}

// GetCurrentTOTPCode generates current TOTP code for testing/display
func GetCurrentTOTPCode(secret string) (string, error) {
	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		return "", fmt.Errorf("failed to generate current TOTP code: %w", err)
	}
	return code, nil
}

// EstimateTimeUntilNextCode estimates seconds until next TOTP code
func EstimateTimeUntilNextCode() int {
	return 30 - int(time.Now().Unix()%30)
}

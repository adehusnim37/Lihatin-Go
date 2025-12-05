package auth

import (
	"context"
	"errors"
	"time"

	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

var jwtSecret = []byte(config.GetRequiredEnv(config.EnvJWTSecret))

// JWTClaims represents the JWT claims structure
type JWTClaims struct {
	UserID     string `json:"user_id"`
	SessionID  string `json:"session_id,omitempty"`
	DeviceID   string `json:"device_id,omitempty"`
	LastIP     string `json:"last_ip,omitempty"`
	Username   string `json:"username"`
	Email      string `json:"email"`
	Role       string `json:"role"`
	IsPremium  bool   `json:"is_premium"`
	IsVerified bool   `json:"is_verified"`
	jwt.RegisteredClaims
}

// GenerateJWT creates a new JWT token for a user with unique JTI for blacklist support
func GenerateJWT(userID, session_id, device_id, last_ip, username, email, role string, isPremium, isVerified bool) (string, error) {
	// Generate unique JTI (JWT ID) for blacklist tracking
	jti, err := GenerateSecureToken(32)
	if err != nil {
		return "", err
	}

	claims := JWTClaims{
		UserID:     userID,
		SessionID:  session_id,
		DeviceID:   device_id,
		LastIP:     last_ip,
		Username:   username,
		Email:      email,
		Role:       role,
		IsPremium:  isPremium,
		IsVerified: isVerified,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,                                                                                                         // JWT ID for blacklist
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(config.GetEnvAsInt(config.EnvJWTExpired, 24)) * time.Hour)), // 48 hours
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "lihatin-go",
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// GenerateRefreshToken creates a refresh token and stores it in Redis
func GenerateRefreshToken(ctx context.Context, redisClient *redis.Client, userID, sessionID, deviceID, lastIP string) (string, error) {
	// Generate unique refresh token (128 char hex string)
	refreshToken, err := GenerateSecureToken(64)
	if err != nil {
		return "", err
	}

	// Store in Redis with metadata
	refreshTokenManager := NewRefreshTokenManager(redisClient)
	expiresAt := time.Now().Add(7 * 24 * time.Hour) // 7 days

	err = refreshTokenManager.StoreRefreshToken(ctx, refreshToken, RefreshTokenData{
		UserID:    userID,
		SessionID: sessionID,
		DeviceID:  deviceID,
		LastIP:    lastIP,
		CreatedAt: time.Now(),
		ExpiresAt: expiresAt,
	})

	if err != nil {
		return "", err
	}

	return refreshToken, nil
}

// ValidateJWT validates and parses a JWT token
func ValidateJWT(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// ValidateRefreshToken validates a refresh token
func ValidateRefreshToken(tokenString string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return jwtSecret, nil
	})

	if err != nil {
		return "", err
	}

	if claims, ok := token.Claims.(*jwt.RegisteredClaims); ok && token.Valid {
		return claims.Subject, nil
	}

	return "", errors.New("invalid refresh token")
}

// ExtractTokenFromHeader extracts JWT token from Authorization header
func ExtractTokenFromHeader(authHeader string) string {
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		return authHeader[7:]
	}
	return ""
}

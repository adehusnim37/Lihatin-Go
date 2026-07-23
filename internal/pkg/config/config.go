// Package config provides configuration utilities for environment variables.
// This package centralizes all environment variable handling with safe loading
// and error handling for .env files.
package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// envLoaded tracks if .env file has been loaded to avoid multiple loads
var envLoaded bool

// loadEnvOnce loads the .env file only once per application run
func loadEnvOnce() {
	if !envLoaded {
		err := godotenv.Load()
		if err != nil {
			// Log warning but don't fail - environment variables might be set elsewhere
			log.Printf("Warning: Could not load .env file: %v", err)
		}
		envLoaded = true
	}
}

// GetEnvOrDefault returns environment variable value or default
// This function safely loads .env file and handles cases where it might not exist
func GetEnvOrDefault(key, defaultValue string) string {
	loadEnvOnce()

	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetRequiredEnv returns environment variable value or panics if not set
func GetRequiredEnv(key string) string {
	loadEnvOnce()

	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("%s environment variable is required but not set", key)
	}
	return value
}

// GetEnvAsInt returns environment variable as integer or default
func GetEnvAsInt(key string, defaultValue int) int {
	loadEnvOnce()

	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		log.Printf("Warning: Could not parse %s as integer, using default: %v", key, err)
		return defaultValue
	}

	return value
}

// GetEnvAsBool returns environment variable as boolean or default
func GetEnvAsBool(key string, defaultValue bool) bool {
	loadEnvOnce()

	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		log.Printf("Warning: Could not parse %s as boolean, using default: %v", key, err)
		return defaultValue
	}

	return value
}

// Configuration constants for commonly used environment variables
const (
	EnvDatabaseURL         = "DATABASE_URL"
	EnvSMTPHost            = "SMTP_HOST"
	EnvSMTPPort            = "SMTP_PORT"
	EnvSMTPUser            = "SMTP_USER"
	EnvSMTPPass            = "SMTP_PASS"
	EnvFromEmail           = "FROM_EMAIL"
	EnvFromName            = "FROM_NAME"
	EnvJWTSecret           = "JWT_SECRET"
	EnvAppPort             = "APP_PORT"
	EnvJWTExpired          = "JWT_EXPIRED"
	EnvRefreshTokenExpired = "REFRESH_TOKEN_EXPIRED"
	EnvIPGeoAPIKey         = "IP_GEOLOCATION_API_KEY"
	EnvFrontendURL         = "FRONTEND_URL"
	EnvBackendURL          = "BACKEND_URL"
	EnvRedisAddr           = "REDIS_ADDR"
	EnvRedisPassword       = "REDIS_PASSWORD"
	EnvRedisDB             = "REDIS_DB"
	EnvSessionSecret       = "SESSION_SECRET"
	EnvSessionTTL          = "SESSION_TTL_HOURS"
	EnvAllowedOrigins      = "ALLOWED_ORIGINS"
	EnvDomain              = "DOMAIN"
	Env                    = "ENV"

	// Security
	EnvCSRFSecret                     = "CSRF_SECRET"
	EnvAuthCookieSameSite             = "AUTH_COOKIE_SAME_SITE"
	EnvRateLimit                      = "RATE_LIMIT"
	EnvPremiumCodeSecret              = "PREMIUM_CODE_SECRET"
	EnvAuthEnforceTOTPForPrivileged   = "AUTH_ENFORCE_TOTP_FOR_PRIVILEGED"
	EnvAuthSecondFactorLimitPerUserIP = "AUTH_SECOND_FACTOR_LIMIT_PER_USER_IP"
	EnvAuthSecondFactorLimitPerIP     = "AUTH_SECOND_FACTOR_LIMIT_PER_IP"
	EnvAuthSecondFactorWindowMinutes  = "AUTH_SECOND_FACTOR_WINDOW_MINUTES"

	// Email
	EnvEmailVerificationExpiry = "EXPIRE_EMAIL_VERIFICATION_TOKEN_HOURS"

	// Object Storage (S3-compatible)
	// #nosec G101 -- These are validation messages, not credentials.
	EnvOSSEndpoint      = "OSS_ENDPOINT"
	EnvOSSRegion        = "OSS_REGION"
	EnvOSSAccessKey     = "OSS_ACCESS_KEY"
	EnvOSSSecretKey     = "OSS_SECRET_KEY"
	EnvOSSBucket        = "OSS_BUCKET"
	EnvOSSPathStyle     = "OSS_PATH_STYLE"
	EnvOSSPublicBaseURL = "OSS_PUBLIC_BASE_URL"

	// OAuth (Google)
	// #nosec G101 -- These are validation messages, not credentials.
	EnvGoogleOAuthAuthorizeEndpoint = "GOOGLE_OAUTH_AUTHORIZE_ENDPOINT"
	EnvGoogleOAuthTokenEndpoint     = "GOOGLE_OAUTH_TOKEN_ENDPOINT"
	EnvGoogleOAuthTokenInfoEndpoint = "GOOGLE_OAUTH_TOKEN_INFO_ENDPOINT"
	EnvGoogleOAuthClientID          = "GOOGLE_OAUTH_CLIENT_ID"
	EnvGoogleOAuthClientSecret      = "GOOGLE_OAUTH_CLIENT_SECRET"
	EnvGoogleOAuthRedirectURI       = "GOOGLE_OAUTH_REDIRECT_URI"
	EnvGoogleOAuthScopes            = "GOOGLE_OAUTH_SCOPES"

	// Disposable email policy
	EnvDisposableEmailBlockListURL = "DISPOSABLE_EMAIL_BLOCK_LIST_URL"

	// Support + captcha
	EnvTurnstileSecretKey = "TURNSTILE_SECRET_KEY"
	EnvTurnstileSiteKey   = "TURNSTILE_SITE_KEY"
	EnvSupportAlertEmails = "SUPPORT_ALERT_EMAILS"
)

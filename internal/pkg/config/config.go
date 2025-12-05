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
	EnvCSRFSecret = "CSRF_SECRET"
	EnvRateLimit  = "RATE_LIMIT"

	// Email
	EnvEmailVerificationExpiry = "EXPIRE_EMAIL_VERIFICATION_TOKEN_HOURS"
)

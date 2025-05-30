package middleware

import (
	"net/http"
	"strings"

	"github.com/adehusnim37/lihatin-go/models"
	"github.com/adehusnim37/lihatin-go/repositories"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/gin-gonic/gin"
)

// AuthMiddleware provides JWT authentication middleware
func AuthMiddleware(userRepo repositories.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, models.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Authorization header required",
				Error:   map[string]string{"auth": "Missing authorization header"},
			})
			c.Abort()
			return
		}

		token := utils.ExtractTokenFromHeader(authHeader)
		if token == "" {
			c.JSON(http.StatusUnauthorized, models.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Invalid authorization header format",
				Error:   map[string]string{"auth": "Authorization header must be 'Bearer <token>'"},
			})
			c.Abort()
			return
		}

		claims, err := utils.ValidateJWT(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, models.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Invalid or expired token",
				Error:   map[string]string{"auth": "Please login again"},
			})
			c.Abort()
			return
		}

		// Optionally verify user still exists and is active
		user, err := userRepo.GetUserByID(claims.UserID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, models.APIResponse{
				Success: false,
				Data:    nil,
				Message: "User not found",
				Error:   map[string]string{"auth": "Invalid user"},
			})
			c.Abort()
			return
		}

		// Set user information in context for use by handlers
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("email", claims.Email)
		c.Set("role", claims.Role)
		c.Set("is_premium", claims.IsPremium)
		c.Set("is_verified", claims.IsVerified)
		c.Set("user", user)

		c.Next()
	}
}

// RequireEmailVerification middleware ensures user has verified their email
func RequireEmailVerification() gin.HandlerFunc {
	return func(c *gin.Context) {
		isVerified, exists := c.Get("is_verified")
		if !exists || !isVerified.(bool) {
			c.JSON(http.StatusForbidden, models.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Email verification required",
				Error:   map[string]string{"verification": "Please verify your email address to access this resource"},
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// RequirePremium middleware ensures user has premium access
func RequirePremium() gin.HandlerFunc {
	return func(c *gin.Context) {
		isPremium, exists := c.Get("is_premium")
		if !exists || !isPremium.(bool) {
			c.JSON(http.StatusForbidden, models.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Premium access required",
				Error:   map[string]string{"premium": "This feature requires premium access"},
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// RequireRole middleware ensures user has specific role
func RequireRole(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists || role.(string) != requiredRole {
			c.JSON(http.StatusForbidden, models.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Insufficient permissions",
				Error:   map[string]string{"role": "You don't have permission to access this resource"},
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// OptionalAuth middleware that extracts user info if token is present but doesn't require it
func OptionalAuth(userRepo repositories.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		if authHeader != "" {
			token := utils.ExtractTokenFromHeader(authHeader)
			if token != "" {
				claims, err := utils.ValidateJWT(token)
				if err == nil {
					// Set user information in context if token is valid
					c.Set("user_id", claims.UserID)
					c.Set("username", claims.Username)
					c.Set("email", claims.Email)
					c.Set("role", claims.Role)
					c.Set("is_premium", claims.IsPremium)
					c.Set("is_verified", claims.IsVerified)

					// Optionally get full user object
					if user, err := userRepo.GetUserByID(claims.UserID); err == nil {
						c.Set("user", user)
					}
				}
			}
		}

		c.Next()
	}
}

// RateLimitMiddleware provides basic rate limiting
func RateLimitMiddleware() gin.HandlerFunc {
	// Simple in-memory rate limiting
	// In production, use Redis or similar
	requestCounts := make(map[string]int)

	return func(c *gin.Context) {
		clientIP := c.ClientIP()

		// Reset counter every minute (simplified)
		if requestCounts[clientIP] > 100 { // 100 requests per minute
			c.JSON(http.StatusTooManyRequests, models.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Rate limit exceeded",
				Error:   map[string]string{"rate_limit": "Too many requests, please try again later"},
			})
			c.Abort()
			return
		}

		requestCounts[clientIP]++
		c.Next()
	}
}

// SecurityHeaders middleware adds security headers
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Header("Content-Security-Policy", "default-src 'self'")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Next()
	}
}

// CORSMiddleware handles CORS headers
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// In production, maintain a whitelist of allowed origins
		allowedOrigins := []string{
			"http://localhost:3000",
			"http://localhost:3001",
			"http://localhost:8080",
			"https://yourdomain.com",
		}

		for _, allowedOrigin := range allowedOrigins {
			if origin == allowedOrigin {
				c.Header("Access-Control-Allow-Origin", origin)
				break
			}
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Requested-With")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// IPWhitelistMiddleware restricts access to whitelisted IPs
func IPWhitelistMiddleware(whitelist []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()

		for _, allowedIP := range whitelist {
			if clientIP == allowedIP || allowedIP == "*" {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Access denied",
			Error:   map[string]string{"ip": "Your IP address is not allowed"},
		})
		c.Abort()
	}
}

// APIKeyMiddleware validates API keys for service-to-service communication
func APIKeyMiddleware(userRepo repositories.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, models.APIResponse{
				Success: false,
				Data:    nil,
				Message: "API key required",
				Error:   map[string]string{"api_key": "Missing X-API-Key header"},
			})
			c.Abort()
			return
		}

		// Validate API key (implementation depends on your API key storage)
		// This is a placeholder - implement based on your UserAPIClient model
		valid := validateAPIKey(apiKey, userRepo)
		if !valid {
			c.JSON(http.StatusUnauthorized, models.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Invalid API key",
				Error:   map[string]string{"api_key": "The provided API key is invalid or expired"},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// validateAPIKey validates an API key against the database
func validateAPIKey(apiKey string, userRepo repositories.UserRepository) bool {
	// TODO: Implement API key validation based on UserAPIClient model
	// This should check if the API key exists, is active, and not expired
	return false // Placeholder
}

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := generateRequestID()
		c.Header("X-Request-ID", requestID)
		c.Set("request_id", requestID)
		c.Next()
	}
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	// Simple implementation - in production use UUID or similar
	token, _ := utils.GeneratePasswordResetToken()
	return strings.ReplaceAll(token, "-", "")[:16]
}

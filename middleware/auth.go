package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/adehusnim37/lihatin-go/repositories"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/adehusnim37/lihatin-go/utils/session"
	"github.com/gin-gonic/gin"
)

// AuthMiddleware provides JWT authentication middleware
func AuthMiddleware(userRepo repositories.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Authorization header required.",
				Error:   map[string]string{"auth": "Missing authorization header. Login required."},
			})
			c.Abort()
			return
		}

		token := utils.ExtractTokenFromHeader(authHeader)
		if token == "" {
			c.JSON(http.StatusUnauthorized, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Invalid authorization header format. Please provide a valid token.",
				Error:   map[string]string{"auth": "Authorization header must be 'Bearer <token>'"},
			})
			c.Abort()
			return
		}

		claims, err := utils.ValidateJWT(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, common.APIResponse{
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
			c.JSON(http.StatusUnauthorized, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "User not found",
				Error:   map[string]string{"auth": "Invalid user"},
			})
			c.Abort()
			return
		}

		if claims.SessionID != "" {
			// Basic validation
			_, isValid := session.ValidateSessionID(claims.SessionID)
			if !isValid {
				c.JSON(http.StatusUnauthorized, common.APIResponse{
					Success: false,
					Data:    nil,
					Message: "Invalid session",
					Error:   map[string]string{"session": "Invalid session"},
				})
				c.Abort()
				return
			}

			// Advanced validation with user/IP/UA check
			validMetadata, err := session.ValidateSessionForUser(
				claims.SessionID,
				claims.UserID,
				c.ClientIP(),
				c.GetHeader("User-Agent"),
			)

			if err != nil {
				utils.Logger.Warn("Session validation failed",
					"user_id", claims.UserID,
					"error", err.Error(),
				)
				c.JSON(http.StatusUnauthorized, common.APIResponse{
					Success: false,
					Data:    nil,
					Message: "Session validation failed",
					Error:   map[string]string{"session": "Session validation failed"},
				})
				c.Abort()
				return
			}

			// Log successful session validation
			utils.Logger.Info("Session validated successfully",
				"user_id", claims.UserID,
				"session_preview", utils.GetKeyPreview(claims.SessionID),
				"purpose", validMetadata.Purpose,
			)

			// Add session metadata to context
			c.Set("session_metadata", validMetadata)
			c.Set("session_purpose", validMetadata.Purpose)
			c.Set("session_issued_at", time.Unix(validMetadata.IssuedAt, 0))
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
			c.JSON(http.StatusForbidden, common.APIResponse{
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
			c.JSON(http.StatusForbidden, common.APIResponse{
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
		utils.Logger.Info("Checking user role", "role", role, "required", requiredRole)
		if !exists || role.(string) != requiredRole {
			c.JSON(http.StatusForbidden, common.APIResponse{
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
					c.Set("is_authenticated", true)

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
func RateLimitMiddleware(count int) gin.HandlerFunc {
	// Simple in-memory rate limiting
	// In production, use Redis or similar
	requestCounts := make(map[string]int)

	return func(c *gin.Context) {
		clientIP := c.ClientIP()

		// Reset counter every minute (simplified)
		if requestCounts[clientIP] > count { // 100 requests per minute
			c.JSON(http.StatusTooManyRequests, common.APIResponse{
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

		c.JSON(http.StatusForbidden, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Access denied",
			Error:   map[string]string{"ip": "Your IP address is not allowed"},
		})
		c.Abort()
	}
}

// APIKeyMiddleware validates API keys for service-to-service communication
func APIKeyMiddleware(apiKeyRepo *repositories.APIKeyRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get API key from request header
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "API key required",
				Error:   map[string]string{"api_key": "Missing X-API-Key header"},
			})
			c.Abort()
			return
		}

		utils.Logger.Info("Received API key for validation",
			"key_preview", utils.GetKeyPreview(apiKey),
			"client_ip", c.ClientIP(),
			"user_agent", c.GetHeader("User-Agent"),
			"path", c.Request.URL.Path,
			"method", c.Request.Method,
			"query", c.Request.URL.RawQuery,
		)

		ip := c.ClientIP()

		// Validate API key using the repository
		user, apiKeyRecord, err := apiKeyRepo.ValidateAPIKey(apiKey, ip)
		if err != nil {
			utils.Logger.Warn("API key validation failed",
				"key_preview", utils.GetKeyPreview(apiKey),
				"error", err.Error(),
			)
			utils.HandleError(c, err, nil)
			c.Abort()
			return
		}

		// Set user and API key information in context for use by handlers
		c.Set("user_id", user.ID)
		c.Set("username", user.Username)
		c.Set("email", user.Email)
		c.Set("role", user.Role)
		c.Set("is_premium", user.IsPremium)
		c.Set("user", user)
		c.Set("api_key", apiKeyRecord)
		c.Set("api_key_id", apiKeyRecord.ID)
		c.Set("api_key_name", apiKeyRecord.Name)
		c.Set("api_key_permissions", apiKeyRecord.Permissions)
		c.Set("is_api_authenticated", true)

		utils.Logger.Info("API key authentication successful",
			"user_id", user.ID,
			"api_key_id", apiKeyRecord.ID,
			"api_key_name", apiKeyRecord.Name,
		)

		c.Next()
	}
}

// ✅ NEW: Multiple permissions middleware
func CheckPermissionAPIKey(authRepo *repositories.AuthRepository, requiredPermissions []string, requireAll bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		// First ensure API key authentication has run
		apiKeyID, exists := c.Get("api_key_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "API key authentication required",
				Error:   map[string]string{"auth": "API key must be authenticated first"},
			})
			c.Abort()
			return
		}

		// Check permissions using repository
		var hasPermission bool
		var err error

		keyIDStr := apiKeyID.(string)
		if requireAll {
			hasPermission, err = authRepo.GetAPIKeyRepository().APIKeyCheckAllPermissions(keyIDStr, requiredPermissions)
		} else {
			hasPermission, err = authRepo.GetAPIKeyRepository().APIKeyCheckPermissions(keyIDStr, requiredPermissions)
		}

		if err != nil {
			utils.Logger.Error("Error checking API key permissions from database",
				"key_id", keyIDStr,
				"error", err.Error(),
			)
			c.JSON(http.StatusInternalServerError, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Permission check failed",
				Error:   map[string]string{"permission": "Failed to verify API key permissions"},
			})
			c.Abort()
			return
		}

		if !hasPermission {
			apiKeyName, _ := c.Get("api_key_name")
			utils.Logger.Warn("API key lacks required permissions (database check)",
				"api_key_name", apiKeyName,
				"key_id", keyIDStr,
				"required_permissions", requiredPermissions,
				"require_all", requireAll,
			)

			permissionType := "any of"
			if requireAll {
				permissionType = "all of"
			}

			c.JSON(http.StatusForbidden, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Insufficient API key permissions.",
				Error: map[string]string{
					"permission": fmt.Sprintf("API key requires %s these permissions: %v", permissionType, requiredPermissions),
				},
			})
			c.Abort()
			return
		}

		// Log successful permission check
		utils.Logger.Info("API key permission check passed (database)",
			"key_id", keyIDStr,
			"required_permissions", requiredPermissions,
			"require_all", requireAll,
		)

		c.Next()
	}
}

// AuthRepositoryAPIKeyMiddleware validates API keys using AuthRepository
func AuthRepositoryAPIKeyMiddleware(authRepo *repositories.AuthRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "API key required",
				Error:   map[string]string{"api_key": "Missing X-API-Key header"},
			})
			c.Abort()
			return
		}

		// Extract client IP
		ip := c.ClientIP()

		// Validate API key using the auth repository
		user, apiKeyRecord, err := authRepo.GetAPIKeyRepository().ValidateAPIKey(apiKey, ip)
		if err != nil {
			utils.Logger.Warn("API key validation failed",
				"key_preview", utils.GetKeyPreview(apiKey),
				"error", err.Error(),
			)
			utils.HandleError(c, err, nil)
			c.Abort()
			return
		}

		// Set user and API key information in context for use by handlers
		c.Set("user_id", user.ID)
		c.Set("username", user.Username)
		c.Set("email", user.Email)
		c.Set("role", user.Role)
		c.Set("is_premium", user.IsPremium)
		c.Set("user", user)
		c.Set("api_key", apiKeyRecord)
		c.Set("api_key_id", apiKeyRecord.ID)
		c.Set("api_key_name", apiKeyRecord.Name)
		c.Set("api_key_permissions", apiKeyRecord.Permissions)
		c.Set("is_api_authenticated", true)

		utils.Logger.Info("API key authentication successful",
			"user_id", user.ID,
			"api_key_id", apiKeyRecord.ID,
			"api_key_name", apiKeyRecord.Name,
		)

		c.Next()
	}
}

// ApiKeyAuthMiddleware is an alias for backward compatibility (deprecated)
func ApiKeyAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "API key middleware not properly initialized",
			Error:   map[string]string{"config": "Please use APIKeyMiddleware or AuthRepositoryAPIKeyMiddleware instead"},
		})
		c.Abort()
	}
}

// APIKeyPermissionMiddleware checks if the API key has specific permissions
func APIKeyPermissionMiddleware(requiredPermission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		permissions, exists := c.Get("api_key_permissions")
		if !exists {
			c.JSON(http.StatusForbidden, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "API key permissions not found",
				Error:   map[string]string{"permission": "Unable to verify API key permissions"},
			})
			c.Abort()
			return
		}

		permissionList, ok := permissions.([]string)
		if !ok {
			c.JSON(http.StatusForbidden, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Invalid permission format",
				Error:   map[string]string{"permission": "Unable to parse API key permissions"},
			})
			c.Abort()
			return
		}

		// Check if the required permission exists in the API key permissions
		hasPermission := false
		for _, perm := range permissionList {
			if perm == requiredPermission || perm == "*" { // "*" means all permissions
				hasPermission = true
				break
			}
		}

		if !hasPermission {
			apiKeyName, _ := c.Get("api_key_name")
			utils.Logger.Warn("API key missing required permission",
				"api_key_name", apiKeyName,
				"required_permission", requiredPermission,
				"available_permissions", permissionList,
			)
			c.JSON(http.StatusForbidden, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Insufficient API key permissions",
				Error:   map[string]string{"permission": fmt.Sprintf("API key does not have '%s' permission", requiredPermission)},
			})
			c.Abort()
			return
		}

		c.Next()
	}
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

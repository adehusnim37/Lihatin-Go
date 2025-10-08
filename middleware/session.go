package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/adehusnim37/lihatin-go/utils/session"
	"github.com/gin-gonic/gin"
)

// InitSessionManager initializes the global session manager
func InitSessionManager() error {
	redisAddr := utils.GetEnvOrDefault(utils.EnvRedisAddr, "localhost:6379")
	redisPassword := utils.GetEnvOrDefault(utils.EnvRedisPassword, "")
	redisDB := utils.GetEnvAsInt(utils.EnvRedisDB, 0)
	sessionTTLHours := utils.GetEnvAsInt(utils.EnvSessionTTL, 48) // Default 48 hours

	manager, err := session.NewManager(
		redisAddr,
		redisPassword,
		redisDB,
		time.Duration(sessionTTLHours)*time.Hour,
	)
	if err != nil {
		return err
	}

	// Set as global manager for use in validation functions
	session.SetManager(manager)

	utils.Logger.Info("Session manager initialized",
		"redis_addr", redisAddr,
		"session_ttl_hours", sessionTTLHours,
	)

	return nil
}

// GetSessionManager returns the initialized session manager
func GetSessionManager() *session.Manager {
	return session.GetManager()
}

// SessionMiddleware validates and refreshes sessions from Redis
func SessionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get session ID from header or cookie
		sessionID := getSessionIDFromRequest(c)
		if sessionID == "" {
			c.JSON(http.StatusUnauthorized, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Session required",
				Error:   map[string]string{"session": "No session ID provided"},
			})
			c.Abort()
			return
		}

		ctx := context.Background()
		manager := GetSessionManager()

		// Validate session
		sess, err := manager.Validate(ctx, sessionID)
		if err != nil {
			if err == session.ErrSessionExpired {
				c.JSON(http.StatusUnauthorized, common.APIResponse{
					Success: false,
					Data:    nil,
					Message: "Session expired",
					Error:   map[string]string{"session": "Your session has expired, please login again"},
				})
			} else {
				c.JSON(http.StatusUnauthorized, common.APIResponse{
					Success: false,
					Data:    nil,
					Message: "Invalid session",
					Error:   map[string]string{"session": "Session is invalid or not found"},
				})
			}
			c.Abort()
			return
		}

		// Refresh session TTL on each request
		if err := manager.Refresh(ctx, sessionID); err != nil {
			utils.Logger.Warn("Failed to refresh session", "error", err)
		}

		// Set session data in context
		c.Set("session_id", sessionID)
		c.Set("session", sess)
		c.Set("user_id", sess.UserID)

		c.Next()
	}
}

// OptionalSessionMiddleware validates session if present, but doesn't require it
func OptionalSessionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := getSessionIDFromRequest(c)
		if sessionID == "" {
			c.Next()
			return
		}

		ctx := context.Background()
		manager := GetSessionManager()
		sess, err := manager.Validate(ctx, sessionID)
		if err == nil {
			// Refresh session
			_ = manager.Refresh(ctx, sessionID)

			// Set session data in context
			c.Set("session_id", sessionID)
			c.Set("session", sess)
			c.Set("user_id", sess.UserID)
		}

		c.Next()
	}
}

// CreateSession creates a new session and returns the session ID
func CreateSession(ctx context.Context, userID, purpose, ipAddress, userAgent, deviceID string) (string, error) {
	manager := GetSessionManager()
	sess, err := manager.Create(ctx, userID, purpose, ipAddress, userAgent, deviceID)
	if err != nil {
		return "", err
	}
	return sess.ID, nil
}

// DeleteSession removes a session
func DeleteSession(ctx context.Context, sessionID string) error {
	manager := GetSessionManager()
	return manager.Delete(ctx, sessionID)
}

// DeleteAllUserSessions removes all sessions for a user
func DeleteAllUserSessions(ctx context.Context, userID string) error {
	manager := GetSessionManager()
	return manager.DeleteAllUserSessions(ctx, userID)
}

// GetSession retrieves session data
func GetSession(ctx context.Context, sessionID string) (*session.Session, error) {
	manager := GetSessionManager()
	return manager.Get(ctx, sessionID)
}

// GetActiveSessionCount returns the number of active sessions for a user
func GetActiveSessionCount(ctx context.Context, userID string) (int, error) {
	manager := GetSessionManager()
	return manager.GetActiveSessionCount(ctx, userID)
}

// getSessionIDFromRequest extracts session ID from Authorization header or cookie
func getSessionIDFromRequest(c *gin.Context) string {
	// Try Authorization header first (format: "Session <session_id>")
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Session") {
			return parts[1]
		}
	}

	// Try X-Session-ID header
	if sessionID := c.GetHeader("X-Session-ID"); sessionID != "" {
		return sessionID
	}

	// Try cookie
	if sessionID, err := c.Cookie("session_id"); err == nil {
		return sessionID
	}

	return ""
}

// SetSessionCookie sets the session cookie in response
func SetSessionCookie(c *gin.Context, sessionID string, maxAge int) {
	c.SetCookie(
		"session_id", // name
		sessionID,    // value
		maxAge,       // maxAge (seconds)
		"/",          // path
		"",           // domain (empty = current domain)
		false,        // secure (set to true in production with HTTPS)
		true,         // httpOnly
	)
}

// ClearSessionCookie removes the session cookie
func ClearSessionCookie(c *gin.Context) {
	c.SetCookie(
		"session_id",
		"",
		-1,
		"/",
		"",
		false,
		true,
	)
}

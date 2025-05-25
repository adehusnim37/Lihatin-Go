package middleware

import (
	"fmt"
	_ "net/http"
	"time"

	"github.com/adehusnim37/lihatin-go/models"
	"github.com/adehusnim37/lihatin-go/repositories"
	"github.com/gin-gonic/gin"
)

// ActivityLogger middleware for logging user activities
func ActivityLogger(loggerRepo *repositories.LoggerRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		startTime := time.Now()

		// Process request
		c.Next()

		// Get response status
		statusCode := c.Writer.Status()

		// Extract browser and system information
		userAgent := c.Request.UserAgent()
		browserInfo := parseUserAgent(userAgent)

		// Get username from context (if user is authenticated)
		username, exists := c.Get("username")
		if !exists {
			username = "anonymous"
		}

		// Get path and method
		path := c.Request.URL.Path
		method := c.Request.Method

		// Determine action and level based on method and status code
		action := determineAction(method, path)
		level := determineLevel(statusCode)

		// Create log message
		message := fmt.Sprintf("%s %s - %d", method, path, statusCode)

		// Create log entry
		log := &models.LoggerUser{
			Level:       level,
			Message:     message,
			Username:    fmt.Sprintf("%v", username),
			Timestamp:   startTime.Format(time.RFC3339),
			IPAddress:   c.ClientIP(),
			UserAgent:   userAgent,
			BrowserInfo: browserInfo,
			Action:      action,
			Route:       path,
			Method:      method,
			StatusCode:  statusCode,
		}

		// Save log asynchronously to not block the response
		go func(logEntry *models.LoggerUser) {
			err := loggerRepo.CreateLog(logEntry)
			if err != nil {
				// Just print error since we're in a goroutine
				fmt.Printf("Failed to save log: %v\n", err)
			}
		}(log)
	}
}

// Helper functions

// parseUserAgent extracts browser and OS information from user agent string
func parseUserAgent(userAgent string) string {
	// This is a simplified version. For production, you might want to use
	// a library like github.com/mssola/user_agent
	return userAgent
}

// determineAction determines what action the user is performing based on HTTP method and path
func determineAction(method, path string) string {
	switch method {
	case "GET":
		return "View"
	case "POST":
		return "Create"
	case "PUT", "PATCH":
		return "Update"
	case "DELETE":
		return "Delete"
	default:
		return "Other"
	}
}

// determineLevel determines log level based on status code
func determineLevel(statusCode int) string {
	if statusCode >= 500 {
		return "ERROR"
	} else if statusCode >= 400 {
		return "WARNING"
	} else {
		return "INFO"
	}
}

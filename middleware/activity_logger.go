package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/adehusnim37/lihatin-go/models/logging"
	"github.com/adehusnim37/lihatin-go/repositories"
	"github.com/gin-gonic/gin"
)

type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)                  // copy to buffer
	return w.ResponseWriter.Write(b) // write out normally
}

func (w bodyLogWriter) WriteString(s string) (int, error) {
	w.body.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}
	
// ActivityLogger middleware for logging user activities
func ActivityLogger(loggerRepo *repositories.LoggerRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		startTime := time.Now()

		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw

		// Capture request body for POST, PUT, PATCH requests
		var requestBody string
		if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH" {
			bodyBytes, err := captureRequestBody(c)
			if err == nil {
				requestBody = sanitizeRequestBody(string(bodyBytes))
			}
		}

		// Capture query parameters - ensure valid JSON for database constraint
		queryParams := captureQueryParams(c)
		if queryParams == "" {
			queryParams = "{}" // Empty JSON object to satisfy database constraint
		}

		// Capture route parameters - ensure valid JSON for database constraint
		routeParams := captureRouteParams(c)
		if routeParams == "" {
			routeParams = "{}" // Empty JSON object to satisfy database constraint
		}

		// Process request
		c.Next()

		// Calculate response time
		responseTime := time.Since(startTime).Milliseconds()

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

		// Capture context locals (all context values) - ensure valid JSON for database constraint
		contextLocals := captureContextLocals(c)
		if contextLocals == "" {
			contextLocals = "{}" // Empty JSON object to satisfy database constraint
		}

		// Get path and method
		path := c.Request.URL.Path
		method := c.Request.Method

		// Determine action and level based on method and status code
		action := determineAction(method, path)
		level := determineLevel(statusCode)

		// Create log message
		message := fmt.Sprintf("%s %s - %d (%dms)", method, path, statusCode, responseTime)

		// Create response body - ensure valid JSON for database constraint
		responseBody := blw.body.String()
		if responseBody == "" {
			responseBody = "{}" // Empty JSON object to satisfy database constraint
		}

		// Ensure requestBody is valid JSON for database constraint
		if requestBody == "" {
			requestBody = "{}" // Empty JSON object to satisfy database constraint
		}

		// Create log entry
		log := &logging.ActivityLog{
			Level:         level,
			Message:       message,
			Username:      fmt.Sprintf("%v", username),
			Timestamp:     startTime,
			IPAddress:     c.ClientIP(),
			UserAgent:     userAgent,
			BrowserInfo:   browserInfo,
			Action:        action,
			Route:         path,
			Method:        method,
			StatusCode:    statusCode,
			RequestBody:   requestBody,
			QueryParams:   queryParams,
			RouteParams:   routeParams,
			ContextLocals: contextLocals,
			ResponseTime:  responseTime,
			ResponseBody:  responseBody,
		}

		// Save log asynchronously to not block the response
		go func(logEntry *logging.ActivityLog) {
			err := loggerRepo.CreateLog(logEntry)
			if err != nil {
				// Just print error since we're in a goroutine
				fmt.Printf("Failed to save log: %v\n", err)
			}
		}(log)
	}
}

// Helper functions

// captureRequestBody safely captures the request body without consuming it
func captureRequestBody(c *gin.Context) ([]byte, error) {
	if c.Request.Body == nil {
		return nil, nil
	}

	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, err
	}

	// Restore the request body for subsequent handlers
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	return bodyBytes, nil
}

// sanitizeRequestBody removes sensitive information from request body
func sanitizeRequestBody(body string) string {
	// Limit body size to prevent huge logs
	const maxBodySize = 1000
	if len(body) > maxBodySize {
		body = body[:maxBodySize] + "... [truncated]"
	}

	// Try to parse as JSON and remove sensitive fields
	var jsonData map[string]interface{}
	if err := json.Unmarshal([]byte(body), &jsonData); err == nil {
		// Remove sensitive fields
		sensitiveFields := []string{"password", "token", "secret", "key", "auth", "authorization"}
		for _, field := range sensitiveFields {
			for key := range jsonData {
				if strings.Contains(strings.ToLower(key), field) {
					jsonData[key] = "[REDACTED]"
				}
			}
		}

		// Convert back to JSON string
		if sanitizedBytes, err := json.Marshal(jsonData); err == nil {
			return string(sanitizedBytes)
		}
	}

	return body
}

// captureQueryParams captures all query parameters as JSON string
func captureQueryParams(c *gin.Context) string {
	if len(c.Request.URL.RawQuery) == 0 {
		return "{}" // Return empty JSON object instead of empty string
	}

	queryMap := make(map[string]interface{})
	for key, values := range c.Request.URL.Query() {
		if len(values) == 1 {
			queryMap[key] = values[0]
		} else {
			queryMap[key] = values
		}
	}

	if jsonBytes, err := json.Marshal(queryMap); err == nil {
		return string(jsonBytes)
	}

	return "{}" // Return empty JSON object on error
}

// captureRouteParams captures all route parameters as JSON string
func captureRouteParams(c *gin.Context) string {
	params := c.Params
	if len(params) == 0 {
		return "{}" // Return empty JSON object instead of empty string
	}

	paramsMap := make(map[string]string)
	for _, param := range params {
		paramsMap[param.Key] = param.Value
	}

	if jsonBytes, err := json.Marshal(paramsMap); err == nil {
		return string(jsonBytes)
	}

	return "{}" // Return empty JSON object on error
}

// captureContextLocals captures important context values (excluding sensitive data)
func captureContextLocals(c *gin.Context) string {
	locals := make(map[string]interface{})

	// Get common context keys that might be interesting to log
	contextKeys := []string{"user_id", "session_id", "request_id", "tenant_id", "role", "permissions"}

	for _, key := range contextKeys {
		if value, exists := c.Get(key); exists {
			locals[key] = value
		}
	}

	// Add any custom headers that might be relevant
	if c.GetHeader("X-Request-ID") != "" {
		locals["x_request_id"] = c.GetHeader("X-Request-ID")
	}
	if c.GetHeader("X-Forwarded-For") != "" {
		locals["x_forwarded_for"] = c.GetHeader("X-Forwarded-For")
	}

	if len(locals) == 0 {
		return "{}" // Return empty JSON object instead of empty string
	}

	if jsonBytes, err := json.Marshal(locals); err == nil {
		return string(jsonBytes)
	}

	return "{}" // Return empty JSON object on error
}

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

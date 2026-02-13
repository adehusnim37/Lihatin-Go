package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/adehusnim37/lihatin-go/internal/pkg/auth"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/models/logging"
	"github.com/adehusnim37/lihatin-go/repositories/loggerrepo"
	"github.com/gin-gonic/gin"
	"github.com/mssola/useragent"
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
func ActivityLogger(loggerRepo *loggerrepo.LoggerRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip logging for certain paths to prevent recursive logging or large responses
		path := c.Request.URL.Path
		skipPaths := []string{"/v1/logs", "/v1/health", "/v1/csrf-token", "/v1/auth/me"}
		for _, skip := range skipPaths {
			if strings.HasPrefix(path, skip) {
				c.Next()
				return
			}
		}

		// Start timer
		startTime := time.Now()

		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw

		// Capture request body for POST, PUT, PATCH requests
		var requestBody string
		if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH" || c.Request.Method == "DELETE" {
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
		ua := useragent.New(c.Request.UserAgent())
		userAgent := ua.UA()
		browserInfo := parseUserAgent(userAgent)

		// Get username from context (if user is authenticated)
		username, exists := c.Get("username")
		if !exists {
			username = "anonymous user"
		}

		// Get user_id from context and handle type conversion properly
		var userID *string
		if userIDValue, exists := c.Get("user_id"); exists {
			if userIDStr, ok := userIDValue.(string); ok {
				userID = &userIDStr // Convert string to *string
			}
		}

		// Capture context locals (all context values) - ensure valid JSON for database constraint
		contextLocals := captureContextLocals(c)
		if contextLocals == "" {
			contextLocals = "{}" // Empty JSON object to satisfy database constraint
		}

		// Get path and method
		path = c.Request.URL.Path
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

		headersInfo := extractHeadersInfo(c)
		apiKeyValue := extractAPIKey(c)
		apiKeyID := extractIDApiKey(apiKeyValue)

		// Convert API key to pointer if not empty

		// Create log entry
		log := &logging.ActivityLog{
			Level:          level,
			Message:        message,
			Username:       fmt.Sprintf("%v", username),
			Timestamp:      startTime,
			IPAddress:      c.ClientIP(),
			UserID:         userID,
			RequestHeaders: headersInfo,
			APIKey:         apiKeyID,
			UserAgent:      userAgent,
			BrowserInfo:    browserInfo,
			Action:         action,
			Route:          path,
			Method:         method,
			StatusCode:     statusCode,
			RequestBody:    requestBody,
			QueryParams:    queryParams,
			RouteParams:    routeParams,
			ContextLocals:  contextLocals,
			ResponseTime:   responseTime,
			ResponseBody:   responseBody,
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
	ua := useragent.New(userAgent)
	name, version := ua.Browser()
	os := ua.OS()

	return fmt.Sprintf("%s %s on %s", name, version, os)
}

func GetBrowser(userAgent string) string {
	ua := useragent.New(userAgent)
	name, version := ua.Browser()

	return fmt.Sprintf("%s %s", name, version)
}

func GetOS(userAgent string) string {
	ua := useragent.New(userAgent)
	os := ua.OS()

	return fmt.Sprintf("%s", os)
}

func GetDevice(userAgent string) string {
	ua := useragent.New(userAgent)
	uaLower := strings.ToLower(userAgent)

	// Check for specific API clients / HTTP tools
	if strings.Contains(uaLower, "postman") ||
		strings.Contains(uaLower, "insomnia") ||
		strings.Contains(uaLower, "curl") ||
		strings.Contains(uaLower, "httpie") ||
		strings.Contains(uaLower, "axios") ||
		strings.Contains(uaLower, "go-http-client") {
		return "API"
	}

	// Check for iPad specifically (OS or UA string)
	if ua.OS() == "iPadOS" || strings.Contains(uaLower, "ipad") {
		return "iPad"
	}

	if ua.Mobile() {
		return "Mobile"
	}

	if ua.Bot() {
		return "Bot"
	}

	return "Desktop"
}

// extract HeadersInfo extracts relevant headers information
func extractHeadersInfo(c *gin.Context) string {
	headersMap := make(map[string]string)

	for key, values := range c.Request.Header {
		if len(values) > 0 {
			headersMap[key] = values[0] // ambil value pertama
		}
	}

	// Ubah map jadi string JSON
	headersJSON, err := json.Marshal(headersMap)
	if err != nil {
		fmt.Println("Error encoding headers:", err)
		return ""
	}

	return string(headersJSON)
}

// extract X-API-Key from headers if present
func extractAPIKey(c *gin.Context) string {
	apiKey := c.GetHeader("X-API-Key")
	if apiKey == "" {
		return ""
	}
	return apiKey
}

// extractIDApiKey extracts the ID part from a full API key
func extractIDApiKey(apikey string) *string {
	keyParts := auth.SplitAPIKey(apikey)
	if len(keyParts) != 2 {
		logger.Logger.Warn("Invalid API key format - missing separator or the APIKey is not defined", "key_preview", auth.GetKeyPreview(apikey))
		return nil
	}

	keyID := keyParts[0]
	return &keyID
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
	} else if statusCode >= 100 && statusCode < 200 {
		return "DEBUG"
	} else if statusCode >= 200 && statusCode < 300 {
		return "SUCCESS"
	} else {
		return "INFO"
	}
}

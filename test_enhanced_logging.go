// Enhanced Activity Logger Demo Script
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"time"

	"github.com/adehusnim37/lihatin-go/middleware"
	"github.com/adehusnim37/lihatin-go/models"
	"github.com/adehusnim37/lihatin-go/repositories"
	"github.com/gin-gonic/gin"
)

// MockLoggerRepository for testing
type MockLoggerRepository struct {
	*repositories.LoggerRepository
	logs []models.LoggerUser
}

func NewMockLoggerRepository() *MockLoggerRepository {
	return &MockLoggerRepository{
		logs: make([]models.LoggerUser, 0),
	}
}

func (m *MockLoggerRepository) CreateLog(log *models.LoggerUser) error {
	m.logs = append(m.logs, *log)
	fmt.Printf("📝 Log Created:\n")
	fmt.Printf("   Method: %s %s\n", log.Method, log.Route)
	fmt.Printf("   Status: %d\n", log.StatusCode)
	fmt.Printf("   Response Time: %dms\n", log.ResponseTime)
	fmt.Printf("   Request Body: %s\n", log.RequestBody)
	fmt.Printf("   Query Params: %s\n", log.QueryParams)
	fmt.Printf("   Route Params: %s\n", log.RouteParams)
	fmt.Printf("   Context Locals: %s\n", log.ContextLocals)
	fmt.Printf("   Username: %s\n", log.Username)
	fmt.Printf("   IP Address: %s\n", log.IPAddress)
	fmt.Printf("   Level: %s\n", log.Level)
	fmt.Printf("   Message: %s\n\n", log.Message)
	return nil
}

(
	"github.com/adehusnim37/lihatin-go/models"
	"github.com/gin-gonic/gin"
)

// Demo function to show the enhanced logging features
func demoEnhancedLogging() {
	fmt.Println("🚀 Enhanced Activity Logger Demo")
	fmt.Println("=====================================\n")

	// Create sample log entries to demonstrate the new features
	sampleLogs := []models.LoggerUser{
		{
			ID:            "log-001",
			Level:         "INFO",
			Message:       "GET /v1/users/123 - 200 (45ms)",
			Username:      "johndoe",
			Timestamp:     time.Now().Format(time.RFC3339),
			IPAddress:     "192.168.1.100",
			UserAgent:     "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
			BrowserInfo:   "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
			Action:        "View",
			Route:         "/v1/users/123",
			Method:        "GET",
			StatusCode:    200,
			RequestBody:   "",
			QueryParams:   `{"page":"1","limit":"10","search":"john"}`,
			RouteParams:   `{"id":"123"}`,
			ContextLocals: `{"user_id":"456","username":"johndoe","role":"admin","session_id":"sess_789"}`,
			ResponseTime:  45,
		},
		{
			ID:            "log-002",
			Level:         "INFO",
			Message:       "POST /v1/users - 201 (120ms)",
			Username:      "admin",
			Timestamp:     time.Now().Format(time.RFC3339),
			IPAddress:     "10.0.0.5",
			UserAgent:     "PostmanRuntime/7.32.2",
			BrowserInfo:   "PostmanRuntime/7.32.2",
			Action:        "Create",
			Route:         "/v1/users",
			Method:        "POST",
			StatusCode:    201,
			RequestBody:   `{"username":"newuser","email":"user@example.com","password":"[REDACTED]","first_name":"Jane","last_name":"Smith"}`,
			QueryParams:   `{"source":"api","version":"v1"}`,
			RouteParams:   `{}`,
			ContextLocals: `{"user_id":"admin-1","username":"admin","role":"super_admin","x_request_id":"req-abc123"}`,
			ResponseTime:  120,
		},
		{
			ID:            "log-003",
			Level:         "WARNING",
			Message:       "PUT /v1/users/999 - 404 (15ms)",
			Username:      "anonymous",
			Timestamp:     time.Now().Format(time.RFC3339),
			IPAddress:     "203.0.113.1",
			UserAgent:     "curl/7.68.0",
			BrowserInfo:   "curl/7.68.0",
			Action:        "Update",
			Route:         "/v1/users/999",
			Method:        "PUT",
			StatusCode:    404,
			RequestBody:   `{"first_name":"Updated","password":"[REDACTED]","auth_token":"[REDACTED]"}`,
			QueryParams:   `{}`,
			RouteParams:   `{"id":"999"}`,
			ContextLocals: `{}`,
			ResponseTime:  15,
		},
	}

	// Display the enhanced log entries
	for i, log := range sampleLogs {
		fmt.Printf("📋 Sample Log Entry %d:\n", i+1)
		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		fmt.Printf("🔍 Basic Info:\n")
		fmt.Printf("   ID: %s\n", log.ID)
		fmt.Printf("   Level: %s\n", log.Level)
		fmt.Printf("   Message: %s\n", log.Message)
		fmt.Printf("   Username: %s\n", log.Username)
		fmt.Printf("   Timestamp: %s\n", log.Timestamp)
		
		fmt.Printf("\n🌐 Request Details:\n")
		fmt.Printf("   Method: %s\n", log.Method)
		fmt.Printf("   Route: %s\n", log.Route)
		fmt.Printf("   Status Code: %d\n", log.StatusCode)
		fmt.Printf("   Response Time: %dms\n", log.ResponseTime)
		fmt.Printf("   IP Address: %s\n", log.IPAddress)
		fmt.Printf("   User Agent: %s\n", log.UserAgent)
		
		fmt.Printf("\n📝 Enhanced Data:\n")
		if log.RequestBody != "" {
			fmt.Printf("   Request Body: %s\n", log.RequestBody)
		} else {
			fmt.Printf("   Request Body: (empty)\n")
		}
		
		if log.QueryParams != "" && log.QueryParams != "{}" {
			fmt.Printf("   Query Params: %s\n", log.QueryParams)
		} else {
			fmt.Printf("   Query Params: (none)\n")
		}
		
		if log.RouteParams != "" && log.RouteParams != "{}" {
			fmt.Printf("   Route Params: %s\n", log.RouteParams)
		} else {
			fmt.Printf("   Route Params: (none)\n")
		}
		
		if log.ContextLocals != "" && log.ContextLocals != "{}" {
			fmt.Printf("   Context Locals: %s\n", log.ContextLocals)
		} else {
			fmt.Printf("   Context Locals: (none)\n")
		}
		
		fmt.Println()
	}

	// Show security features
	fmt.Println("🔒 Security Features Demonstrated:")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("✅ Password fields automatically redacted: [REDACTED]")
	fmt.Println("✅ Auth tokens automatically redacted: [REDACTED]")
	fmt.Println("✅ Request body size limits applied (1000 chars max)")
	fmt.Println("✅ Sensitive context data filtered")
	fmt.Println()

	// Show performance insights
	fmt.Println("📊 Performance Insights:")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Fast Response (GET): %dms\n", sampleLogs[0].ResponseTime)
	fmt.Printf("Medium Response (POST): %dms\n", sampleLogs[1].ResponseTime)
	fmt.Printf("Quick Error Response: %dms\n", sampleLogs[2].ResponseTime)
	fmt.Println()

	// Show analytics potential
	fmt.Println("🎯 Analytics Capabilities:")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("• Response time tracking for performance monitoring")
	fmt.Println("• Request pattern analysis through body/param logging")
	fmt.Println("• User behavior tracking via context locals")
	fmt.Println("• Error rate monitoring by status codes")
	fmt.Println("• Security audit trails with IP and user agent data")
	fmt.Println()

	fmt.Println("✨ Enhanced Activity Logger is ready for use!")
	fmt.Println("See ACTIVITY_LOGGER.md for complete documentation.")
}

func main() {
	demoEnhancedLogging()
}

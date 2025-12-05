package middleware

import (
    "bytes"
    "encoding/json"
    "io"
    "net/http"

    "github.com/adehusnim37/lihatin-go/repositories"
    "github.com/adehusnim37/lihatin-go/internal/pkg/logger"
    "github.com/gin-gonic/gin"
)

// RecordLoginAttempt middleware records login attempts
func RecordLoginAttempt(loginAttemptRepo *repositories.LoginAttemptRepository) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. Capture request body BEFORE handler consumes it
        var requestBody []byte
        var emailOrUsername string
        
        if c.Request.Body != nil {
            requestBody, _ = io.ReadAll(c.Request.Body)
            // Restore body so handler can read it
            c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
            
            // Extract email_or_username from request
            var payload map[string]interface{}
            if err := json.Unmarshal(requestBody, &payload); err == nil {
                if val, ok := payload["email_or_username"].(string); ok {
                    emailOrUsername = val
                }
            }
        }

        if emailOrUsername == "" {
            emailOrUsername = "unknown"
        }

        // 2. Wrap response writer to capture response body
        blw := &bodyLogWriter{
            body:           bytes.NewBufferString(""),
            ResponseWriter: c.Writer,
        }
        c.Writer = blw

        // 3. Execute the actual login handler
        c.Next()

        // 4. After handler execution, collect data
        status := c.Writer.Status()
        success := status == http.StatusOK
        ipAddress := c.ClientIP()
        userAgent := c.GetHeader("User-Agent")

        // 5. Get failure reason from response body
        failReason := ""
        if !success {
            failReason = blw.body.String()
            
            // Try to extract error message from JSON response
            var responseBody map[string]interface{}
            if err := json.Unmarshal(blw.body.Bytes(), &responseBody); err == nil {
                if errMap, ok := responseBody["error"].(map[string]interface{}); ok {
                    if authErr, ok := errMap["auth"].(string); ok {
                        failReason = authErr
                    }
                } else if msg, ok := responseBody["message"].(string); ok {
                    failReason = msg
                }
            }
        } else {
            failReason = "Login successful"
        }

        // 6. Record the login attempt
        if err := loginAttemptRepo.RecordLoginAttempt(
            ipAddress,
            userAgent,
            success,
            failReason,
            emailOrUsername,
        ); err != nil {
            logger.Logger.Error("Failed to record login attempt", "error", err.Error())
        }

        // 7. Log the attempt
        logger.Logger.Info("Login attempt recorded",
            "email_or_username", emailOrUsername,
            "ip_address", ipAddress,
            "success", success,
            "status", status,
            "fail_reason", failReason,
        )
    }
}
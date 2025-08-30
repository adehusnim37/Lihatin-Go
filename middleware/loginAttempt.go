package middleware

import (
	"bytes"

	"github.com/adehusnim37/lihatin-go/repositories"
	"github.com/gin-gonic/gin"
)

// RecordLoginAttempt is a middleware that records login attempts
func RecordLoginAttempt(loginAttemptRepo *repositories.LoginAttemptRepository) gin.HandlerFunc {
	return func(c *gin.Context) {

		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw
		// Get user information from the context
		userID := c.GetString("user_id")
		ipAddress := c.ClientIP()
		userAgent := c.GetHeader("User-Agent")
		// Check if the login attempt was successful
		success := func() bool {
			return c.Writer.Status() == 200
		}()
		failreason := func() string {
			if success {
				return "All good"
			}
			return blw.body.String()
		}()

		// Record the login attempt
		if err := loginAttemptRepo.RecordLoginAttempt(userID, ipAddress, userAgent, success, failreason); err != nil {
			c.Error(err)
		}
	}
}

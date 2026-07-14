package auth

import (
	"net"
	"net/url"
	"strings"

	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
	"github.com/gin-gonic/gin"
)

// CORS middleware
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get allowed origins from environment or use default
		allowedOrigins := config.GetEnvOrDefault(config.EnvAllowedOrigins, "http://localhost:3000,http://localhost:3001")
		origins := strings.Split(allowedOrigins, ",")

		origin := c.Request.Header.Get("Origin")

		if origin != "" {
			if isExactAllowedOrigin(origin, origins) || (isDevelopmentEnv() && isLocalhostOrigin(origin)) {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			}
		}

		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-API-Key, X-Support-Access-Token")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH, HEAD")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func isExactAllowedOrigin(origin string, allowedOrigins []string) bool {
	for _, allowedOrigin := range allowedOrigins {
		if strings.TrimSpace(allowedOrigin) == origin {
			return true
		}
	}
	return false
}

func isDevelopmentEnv() bool {
	env := strings.ToLower(strings.TrimSpace(config.GetEnvOrDefault(config.Env, "development")))
	return env == "development" || env == "dev" || env == "local"
}

func isLocalhostOrigin(origin string) bool {
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}

	host := parsed.Hostname()
	if host == "" {
		// Fallback for non-standard origins if hostname extraction fails.
		if h, _, splitErr := net.SplitHostPort(parsed.Host); splitErr == nil {
			host = h
		} else {
			host = parsed.Host
		}
	}

	host = strings.Trim(host, "[]")
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

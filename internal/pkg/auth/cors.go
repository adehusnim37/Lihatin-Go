package auth

import (
	"errors"
	"net"
	"net/http"
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
		originAllowed := origin != "" &&
			(isExactAllowedOrigin(origin, origins) || (isDevelopmentEnv() && isLocalhostOrigin(origin)))

		c.Writer.Header().Add("Vary", "Origin")
		if originAllowed {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-API-Key, X-Support-Access-Token")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH, HEAD")
			c.Writer.Header().Set("Access-Control-Max-Age", "86400")
		}

		if c.Request.Method == "OPTIONS" {
			if origin != "" && !originAllowed {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func isExactAllowedOrigin(origin string, allowedOrigins []string) bool {
	candidate, err := canonicalCORSOrigin(origin)
	if err != nil {
		return false
	}

	for _, allowedOrigin := range allowedOrigins {
		allowed, err := canonicalCORSOrigin(allowedOrigin)
		if err == nil && allowed == candidate {
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
	normalized, err := canonicalCORSOrigin(origin)
	if err != nil {
		return false
	}

	parsed, err := url.Parse(normalized)
	if err != nil {
		return false
	}

	host := parsed.Hostname()
	host = strings.Trim(host, "[]")
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func canonicalCORSOrigin(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil {
		return "", errInvalidCORSOrigin
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", errInvalidCORSOrigin
	}
	if (parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errInvalidCORSOrigin
	}

	hostname := strings.ToLower(parsed.Hostname())
	if hostname == "" {
		return "", errInvalidCORSOrigin
	}
	port := parsed.Port()
	if (scheme == "http" && port == "80") || (scheme == "https" && port == "443") {
		port = ""
	}

	host := hostname
	if strings.Contains(hostname, ":") {
		host = "[" + hostname + "]"
	}
	if port != "" {
		host = net.JoinHostPort(hostname, port)
	}

	return (&url.URL{Scheme: scheme, Host: host}).String(), nil
}

var errInvalidCORSOrigin = errors.New("invalid CORS origin")

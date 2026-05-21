package middleware

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/gin-gonic/gin"
)

// BlockSensitivePaths rejects common scanner payloads targeting dotfiles and path traversal.
func BlockSensitivePaths() gin.HandlerFunc {
	return func(c *gin.Context) {
		rawPath := c.Request.URL.EscapedPath()
		if rawPath == "" {
			rawPath = c.Request.URL.Path
		}

		path := strings.ToLower(rawPath)
		if decoded, err := url.PathUnescape(path); err == nil {
			path = decoded
		}
		path = strings.ReplaceAll(path, "\\", "/")

		hiddenSegment := strings.Contains(path, "/.") && !strings.HasPrefix(path, "/.well-known/")
		traversalAttempt := strings.Contains(path, "../") || strings.Contains(path, "/..")
		sensitiveTarget := strings.Contains(path, "/.git") ||
			strings.Contains(path, "/.env") ||
			strings.Contains(path, "/.svn") ||
			strings.Contains(path, "/.hg")

		if hiddenSegment || traversalAttempt || sensitiveTarget {
			c.JSON(http.StatusNotFound, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Route not found",
				Error:   map[string]string{"route": "resource not found"},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

package middleware

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/gin-gonic/gin"
)

// AdminAuth middleware checks if the user has admin privileges
func AdminAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user_id from context (should be set by JWT middleware)
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Authentication required",
				Error:   map[string]string{"auth": "Please authenticate to access this resource"},
			})
			c.Abort()
			return
		}

		// Get user role from context
		role, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusForbidden, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Access denied",
				Error:   map[string]string{"auth": "Insufficient permissions"},
			})
			c.Abort()
			return
		}

		// Check if user has admin role
		roleStr, ok := role.(string)
		if !ok || (roleStr != "admin" && roleStr != "super_admin") {
			c.JSON(http.StatusForbidden, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Admin access required",
				Error:   map[string]string{"auth": "This endpoint requires administrator privileges"},
			})
			c.Abort()
			return
		}

		// Store admin info in context for logging
		c.Set("is_admin", true)
		c.Set("admin_id", userID)

		c.Next()
	}
}

// SuperAdminAuth middleware checks if the user has super admin privileges
func SuperAdminAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user_id from context
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Authentication required",
				Error:   map[string]string{"auth": "Please authenticate to access this resource"},
			})
			c.Abort()
			return
		}

		// Get user role from context
		role, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusForbidden, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Access denied",
				Error:   map[string]string{"auth": "Insufficient permissions"},
			})
			c.Abort()
			return
		}

		// Check if user has super admin role
		roleStr, ok := role.(string)
		if !ok || roleStr != "super_admin" {
			c.JSON(http.StatusForbidden, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Super admin access required",
				Error:   map[string]string{"auth": "This endpoint requires super administrator privileges"},
			})
			c.Abort()
			return
		}

		// Store super admin info in context for logging
		c.Set("is_super_admin", true)
		c.Set("admin_id", userID)

		c.Next()
	}
}

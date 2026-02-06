package loginattempts

import (
	"strings"

	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/gin-gonic/gin"
)

// GetTopFailedIPs returns the top IP addresses with failed login attempts
func (c *Controller) GetTopFailedIPs(ctx *gin.Context) {
	logger.Logger.Info("GetTopFailedIPs called")

	// Only admins can access this endpoint
	isAdmin := ctx.GetBool("is_admin")
	if !isAdmin {
		httputil.SendErrorResponse(ctx, 403, "ADMIN_ONLY", "This endpoint requires admin privileges", "", nil)
		return
	}

	limit := 10
	topIPs, err := c.repo.GetLoginAttemptRepository().GetTopFailedIPs(limit)
	if err != nil {
		httputil.HandleError(ctx, err, nil)
		return
	}

	httputil.SendOKResponse(ctx, map[string]any{
		"top_failed_ips": topIPs,
		"limit":          limit,
	}, "Successfully retrieved top failed IPs")
}

// GetAttemptsByHour returns login attempts grouped by hour
func (c *Controller) GetAttemptsByHour(ctx *gin.Context) {
	logger.Logger.Info("GetAttemptsByHour called")

	days := 7 // Default 7 days

	isAdmin := ctx.GetBool("is_admin")
	var emailOrUsername string

	if !isAdmin {
		// Non-admin users can only view their own data
		if email, exists := ctx.Get("email"); exists {
			emailOrUsername = email.(string)
		} else {
			httputil.SendErrorResponse(ctx, 401, "AUTH_REQUIRED", "Authentication required", "", nil)
			return
		}
	}

	attempts, err := c.repo.GetLoginAttemptRepository().GetAttemptsByHour(days, emailOrUsername)
	if err != nil {
		httputil.HandleError(ctx, err, nil)
		return
	}

	httputil.SendOKResponse(ctx, map[string]any{
		"attempts_by_hour": attempts,
		"days":             days,
	}, "Successfully retrieved attempts by hour")
}

// GetSuspiciousActivity returns potentially suspicious login activity
func (c *Controller) GetSuspiciousActivity(ctx *gin.Context) {
	logger.Logger.Info("GetSuspiciousActivity called")

	// Only admins can access this endpoint
	isAdmin := ctx.GetBool("is_admin")
	if !isAdmin {
		httputil.SendErrorResponse(ctx, 403, "ADMIN_ONLY", "This endpoint requires admin privileges", "", nil)
		return
	}

	suspicious, err := c.repo.GetLoginAttemptRepository().GetSuspiciousActivity()
	if err != nil {
		httputil.HandleError(ctx, err, nil)
		return
	}

	httputil.SendOKResponse(ctx, map[string]any{
		"suspicious_activity": suspicious,
	}, "Successfully retrieved suspicious activity")
}

// GetRecentActivity returns recent login activity summary
func (c *Controller) GetRecentActivity(ctx *gin.Context) {
	logger.Logger.Info("GetRecentActivity called")

	role := ctx.GetString("role")
	isAdmin := strings.EqualFold(role, "admin")
	var emailOrUsername string

	if !isAdmin {
		// Non-admin users can only view their own data
		if email := ctx.GetString("username"); email != "" {
			emailOrUsername = email
		} else {
			httputil.SendErrorResponse(ctx, 401, "AUTH_REQUIRED", "Authentication required", "", nil)
			return
		}
	}

	hours := 24 // Last 24 hours
	summary, err := c.repo.GetLoginAttemptRepository().GetRecentActivitySummary(hours, emailOrUsername)
	if err != nil {
		httputil.HandleError(ctx, err, nil)
		return
	}

	httputil.SendOKResponse(ctx, summary, "Successfully retrieved recent activity")
}

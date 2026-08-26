package auth

import (
	"github.com/adehusnim37/lihatin-go/internal/pkg/auth"
	"github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/middleware"
	"github.com/gin-gonic/gin"
)

// revokeDeviceRequest specifies which device fingerprint to revoke.
type revokeDeviceRequest struct {
	DeviceID string `json:"device_id" validate:"required,min=3,max=2048"`
}

// RevokeDevice invalidates every session belonging to a given device
// fingerprint for the signed-in user.
//
// Security model: the device fingerprint is treated as untrusted metadata and
// is only used to SELECT which sessions to revoke. A malicious caller can only
// revoke sessions for their OWN user_id (scoped server-side), never another
// user's, and cannot escalate privileges. Deleting the Redis session instantly
// invalidates that session's access + refresh tokens via the middleware's
// ValidateSessionForUser check, exactly like logout.
func (c *Controller) RevokeDevice(ctx *gin.Context) {
	userID := ctx.GetString("user_id")
	if userID == "" {
		http.SendErrorResponse(ctx, 401, "Authentication required", "UNAUTHORIZED", "Please login to continue", nil)
		return
	}

	var req revokeDeviceRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		http.SendErrorResponse(ctx, 400, "Invalid request body", "INVALID_BODY", err.Error(), nil)
		return
	}
	if req.DeviceID == "" {
		http.SendErrorResponse(ctx, 400, "DEVICE_ID_REQUIRED", "Device ID is required", "device_id")
		return
	}

	// Load the user's active sessions and pick those matching the device.
	sessions, err := middleware.ListUserSessions(ctx.Request.Context(), userID)
	if err != nil {
		logger.Logger.Error("Failed to list sessions for revoke",
			"user_id", userID,
			"error", err.Error(),
		)
		http.SendErrorResponse(ctx, 500, "Failed to revoke device", "REVOKE_FAILED", "Could not list sessions", nil)
		return
	}

	revoked := 0
	refreshManager := auth.NewRefreshTokenManager(middleware.GetSessionManager().GetRedisClient())

	for _, sess := range sessions {
		if sess.DeviceID != req.DeviceID {
			continue
		}

		// 1) Drop the refresh token(s) for this session so they can't mint a
		//    new access token.
		if err := refreshManager.DeleteSessionRefreshTokens(ctx.Request.Context(), sess.ID); err != nil {
			logger.Logger.Warn("Failed to delete session refresh tokens",
				"session_id", sess.ID,
				"error", err.Error(),
			)
		}

		// 2) Delete the Redis session. Next request on this session will 401 in
		//    the middleware because the session can no longer be validated.
		if err := middleware.DeleteSession(ctx.Request.Context(), sess.ID); err != nil {
			logger.Logger.Warn("Failed to delete session during device revoke",
				"session_id", sess.ID,
				"error", err.Error(),
			)
			continue
		}

		revoked++
	}

	// Always report success; if no session matched it simply means the device
	// has no more active sessions (idempotent, like logout).
	http.SendOKResponse(ctx, map[string]interface{}{
		"revoked_sessions": revoked,
		"device_id":        req.DeviceID,
	}, "Device sessions revoked")
}

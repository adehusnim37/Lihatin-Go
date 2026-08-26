package auth

import (
	nethttp "net/http"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/internal/pkg/auth"
	"github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/middleware"
	"github.com/gin-gonic/gin"
)

// httpErr is a small convenience wrapper around the uniform error response.
func httpErr(ctx *gin.Context, status int, code, message string) {
	http.SendErrorResponse(ctx, status, code, message, ctx.GetString("user_id"))
}

// getSessionManagerClient returns the Redis client shared by auth subsystems.
func getSessionManagerClient() *auth.RefreshTokenManager {
	return auth.NewRefreshTokenManager(middleware.GetSessionManager().GetRedisClient())
}

// ListSessions returns the signed-in user's active sessions, marking the one
// corresponding to the current request as `is_current: true`.
func (c *Controller) ListSessions(ctx *gin.Context) {
	userID := ctx.GetString("user_id")
	currentSessionID := ctx.GetString("session_id")
	if userID == "" {
		httpErr(ctx, nethttp.StatusUnauthorized, "UNAUTHORIZED", "Please login to continue")
		return
	}

	sessions, err := middleware.ListUserSessions(ctx.Request.Context(), userID)
	if err != nil {
		logger.Logger.Error("Failed to list user sessions",
			"user_id", userID,
			"error", err.Error(),
		)
		httpErr(ctx, nethttp.StatusInternalServerError, "LIST_SESSIONS_FAILED", "Could not list sessions")
		return
	}

	items := make([]dto.ActiveSessionResponse, 0, len(sessions))
	for _, s := range sessions {
		items = append(items, dto.ActiveSessionResponse{
			SessionID: s.ID,
			IsCurrent: s.ID == currentSessionID,
			DeviceID:  s.DeviceID,
			IPAddress: s.IPAddress,
			UserAgent: s.UserAgent,
			CreatedAt: s.CreatedAt,
			LastSeen:  s.LastSeen,
			ExpiresAt: s.ExpiresAt,
		})
	}

	http.SendOKResponse(ctx, dto.SessionsListResponse{
		Total:    len(items),
		Sessions: items,
	}, "Active sessions retrieved")
}

// RevokeSession revokes a single session, optionally the current one.
func (c *Controller) RevokeSession(ctx *gin.Context) {
	userID := ctx.GetString("user_id")
	currentSessionID := ctx.GetString("session_id")
	if userID == "" {
		httpErr(ctx, nethttp.StatusUnauthorized, "UNAUTHORIZED", "Please login to continue")
		return
	}

	var req struct {
		SessionID string `json:"session_id" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil || req.SessionID == "" {
		httpErr(ctx, nethttp.StatusBadRequest, "SESSION_ID_REQUIRED", "session_id is required")
		return
	}

	// Security: only allow revoking sessions that actually belong to this user.
	sessions, err := middleware.ListUserSessions(ctx.Request.Context(), userID)
	if err != nil {
		httpErr(ctx, nethttp.StatusInternalServerError, "REVOKE_FAILED", "Could not list sessions")
		return
	}

	targetFound := false
	for _, s := range sessions {
		if s.ID == req.SessionID {
			targetFound = true
			// Drop the refresh token(s) so they can't mint new access tokens.
			_ = getSessionManagerClient().DeleteSessionRefreshTokens(ctx.Request.Context(), s.ID)
			// Delete the Redis session; next request on it will 401.
			if err := middleware.DeleteSession(ctx.Request.Context(), s.ID); err != nil {
				logger.Logger.Warn("Failed to delete session during revoke",
					"session_id", s.ID,
					"error", err.Error(),
				)
			}
			break
		}
	}

	if !targetFound {
		httpErr(ctx, nethttp.StatusNotFound, "SESSION_NOT_FOUND", "Active session not found")
		return
	}

	// If the current session was revoked (e.g. user "removed" the device they're
	// logged in on), clear cookies so the browser signs out cleanly.
	if req.SessionID == currentSessionID {
		middleware.ClearAuthCookies(ctx)
	}

	http.SendOKResponse(ctx, map[string]interface{}{
		"revoked_session": req.SessionID,
		"was_current":     req.SessionID == currentSessionID,
	}, "Session revoked")
}

// RevokeAllSessions revokes every session for the user and clears cookies.
func (c *Controller) RevokeAllSessions(ctx *gin.Context) {
	userID := ctx.GetString("user_id")
	if userID == "" {
		httpErr(ctx, nethttp.StatusUnauthorized, "UNAUTHORIZED", "Please login to continue")
		return
	}

	// 1) Delete all Redis sessions.
	if err := middleware.DeleteAllUserSessions(ctx.Request.Context(), userID); err != nil {
		logger.Logger.Error("Failed to delete all user sessions",
			"user_id", userID,
			"error", err.Error(),
		)
		httpErr(ctx, nethttp.StatusInternalServerError, "REVOKE_ALL_FAILED", "Could not revoke all sessions")
		return
	}

	// 2) Drop all refresh tokens so they can't mint new access tokens.
	_ = getSessionManagerClient().DeleteAllUserRefreshTokens(ctx.Request.Context(), userID)

	// 3) Update the account logout timestamp like logout does.
	if err := c.repo.GetUserAuthRepository().Logout(userID); err != nil {
		logger.Logger.Warn("Failed to update last_logout during revoke-all",
			"user_id", userID,
			"error", err.Error(),
		)
	}

	// 4) Clear cookies immediately (this is the revoke-all action on the
	//    current device, so the browser must sign out).
	middleware.ClearAuthCookies(ctx)

	http.SendOKResponse(ctx, map[string]interface{}{
		"revoked_all": true,
	}, "All sessions revoked")
}

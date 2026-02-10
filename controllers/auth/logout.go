package auth

import (
	"context"
	nethttp "net/http"

	"github.com/adehusnim37/lihatin-go/internal/pkg/auth"
	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
	"github.com/adehusnim37/lihatin-go/internal/pkg/errors"
	"github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/middleware"
	"github.com/gin-gonic/gin"
)

func (c *Controller) Logout(ctx *gin.Context) {
	sessionID := ctx.GetString("session_id")
	userID := ctx.GetString("user_id")

	// Get JWT token from cookie or Authorization header
	var token string
	cookieToken, err := ctx.Cookie("access_token")
	if err == nil && cookieToken != "" {
		token = cookieToken
	} else {
		authHeader := ctx.GetHeader("Authorization")
		token = auth.ExtractTokenFromHeader(authHeader)
	}

	// Get JWT claims to extract jti and expiration
	claims, err := auth.ValidateJWT(token)
	if err != nil {
		logger.Logger.Warn("Failed to extract JWT claims during logout",
			"user_id", userID,
			"error", err.Error(),
		)
		// Continue with logout even if JWT parsing fails
	}

	if err := c.repo.GetUserAuthRepository().Logout(userID); err != nil {
		logger.Logger.Error("Failed to logout user",
			"user_id", userID,
			"session_id", sessionID,
			"error", err.Error(),
		)
		http.HandleError(ctx, err, userID)
		return
	}

	// Get Redis client from session manager
	sessionManager := middleware.GetSessionManager()
	redisClient := sessionManager.GetRedisClient()

	// Blacklist JWT if we have valid claims
	if claims != nil && claims.ID != "" {
		blacklistManager := auth.NewJWTBlacklistManager(redisClient)
		if err := blacklistManager.BlacklistJWT(context.Background(), claims.ID, claims.ExpiresAt.Time); err != nil {
			logger.Logger.Error("Failed to blacklist JWT",
				"user_id", userID,
				"jti", claims.ID,
				"error", err.Error(),
			)
			// Don't fail logout if blacklist fails
		} else {
			logger.Logger.Info("JWT blacklisted successfully",
				"user_id", userID,
				"jti", auth.GetKeyPreview(claims.ID),
			)
		}
	}

	// Delete all refresh tokens for this user
	refreshTokenManager := auth.NewRefreshTokenManager(redisClient)
	if err := refreshTokenManager.DeleteAllUserRefreshTokens(context.Background(), userID); err != nil {
		logger.Logger.Error("Failed to delete refresh tokens",
			"user_id", userID,
			"error", err.Error(),
		)
		// Don't fail logout if refresh token deletion fails
	} else {
		logger.Logger.Info("All refresh tokens deleted",
			"user_id", userID,
		)
	}

	// Invalidate session in Redis
	if err := middleware.DeleteSession(ctx, sessionID); err != nil {
		logger.Logger.Error("Error invalidating user session", "user_id", userID, "session_id", sessionID, "error", err)
		http.SendErrorResponse(ctx, 500, "Failed to invalidate session", "ERR_SESSION_INVALIDATION_FAILED", errors.ErrSessionInvalidationFailed.Error(), nil)
		return
	}

	// Clear HTTP-Only cookies (XSS protection)
	// Detect if running on HTTPS
	isSecure := ctx.GetHeader("X-Forwarded-Proto") == "https" ||
		ctx.GetHeader("X-Forwarded-Ssl") == "on" ||
		ctx.Request.TLS != nil

	// Get domain and set cookie domain (must match original cookie)
	domain := config.GetEnvOrDefault(config.EnvDomain, "localhost")

	// For production with subdomains, use root domain with dot prefix
	if domain != "localhost" && domain != "127.0.0.1" {
		domain = "." + domain
	}

	// Clear access_token cookie
	accessTokenCookie := &nethttp.Cookie{
		Name:     "access_token",
		Value:    "",
		Path:     "/",
		Domain:   domain,
		MaxAge:   -1,
		Secure:   isSecure,
		HttpOnly: true,
		SameSite: nethttp.SameSiteNoneMode,
	}

	// Clear refresh_token cookie
	refreshTokenCookie := &nethttp.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		Domain:   domain,
		MaxAge:   -1,
		Secure:   isSecure,
		HttpOnly: true,
		SameSite: nethttp.SameSiteNoneMode,
	}

	// Apply cookies to response
	nethttp.SetCookie(ctx.Writer, accessTokenCookie)
	nethttp.SetCookie(ctx.Writer, refreshTokenCookie)

	logger.Logger.Info("HTTP-Only cookies cleared",
		"user_id", userID,
		"is_secure", isSecure,
		"domain", domain,
	)

	http.SendOKResponse(ctx, nil, "Logout successful")
}

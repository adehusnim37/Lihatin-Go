package auth

import (
	"context"

	"github.com/adehusnim37/lihatin-go/middleware"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/gin-gonic/gin"
)

func (c *Controller) Logout(ctx *gin.Context) {
	sessionID := ctx.GetString("session_id")
	userID := ctx.GetString("user_id")

	// Get JWT claims to extract jti and expiration
	authHeader := ctx.GetHeader("Authorization")
	token := utils.ExtractTokenFromHeader(authHeader)
	claims, err := utils.ValidateJWT(token)
	if err != nil {
		utils.Logger.Warn("Failed to extract JWT claims during logout",
			"user_id", userID,
			"error", err.Error(),
		)
		// Continue with logout even if JWT parsing fails
	}

	if err := c.repo.GetUserAuthRepository().Logout(userID); err != nil {
		utils.Logger.Error("Failed to logout user",
			"user_id", userID,
			"session_id", sessionID,
			"error", err.Error(),
		)
		utils.HandleError(ctx, err, userID)
		return
	}

	// Get Redis client from session manager
	sessionManager := middleware.GetSessionManager()
	redisClient := sessionManager.GetRedisClient()

	// Blacklist JWT if we have valid claims
	if claims != nil && claims.ID != "" {
		blacklistManager := utils.NewJWTBlacklistManager(redisClient)
		if err := blacklistManager.BlacklistJWT(context.Background(), claims.ID, claims.ExpiresAt.Time); err != nil {
			utils.Logger.Error("Failed to blacklist JWT",
				"user_id", userID,
				"jti", claims.ID,
				"error", err.Error(),
			)
			// Don't fail logout if blacklist fails
		} else {
			utils.Logger.Info("JWT blacklisted successfully",
				"user_id", userID,
				"jti", utils.GetKeyPreview(claims.ID),
			)
		}
	}

	// Delete all refresh tokens for this user
	refreshTokenManager := utils.NewRefreshTokenManager(redisClient)
	if err := refreshTokenManager.DeleteAllUserRefreshTokens(context.Background(), userID); err != nil {
		utils.Logger.Error("Failed to delete refresh tokens",
			"user_id", userID,
			"error", err.Error(),
		)
		// Don't fail logout if refresh token deletion fails
	} else {
		utils.Logger.Info("All refresh tokens deleted",
			"user_id", userID,
		)
	}

	// Invalidate session in Redis
	if err := middleware.DeleteSession(ctx, sessionID); err != nil {
		utils.Logger.Error("Error invalidating user session", "user_id", userID, "session_id", sessionID, "error", err)
		utils.SendErrorResponse(ctx, 500, "Failed to invalidate session", "ERR_SESSION_INVALIDATION_FAILED", utils.ErrSessionInvalidationFailed.Error(), nil)
		return
	}

	utils.SendOKResponse(ctx, nil, "Logout successful")
}

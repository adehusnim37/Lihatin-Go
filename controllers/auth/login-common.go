package auth

import (
	"context"
	"net/http"
	"time"

	"github.com/adehusnim37/lihatin-go/dto"
	pkgauth "github.com/adehusnim37/lihatin-go/internal/pkg/auth"
	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/ip"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/middleware"
	"github.com/adehusnim37/lihatin-go/models/user"
	"github.com/gin-gonic/gin"
)

func buildUserProfile(u *user.User) dto.UserProfile {
	return dto.UserProfile{
		ID:        u.ID,
		Username:  u.Username,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Email:     u.Email,
		Avatar:    u.Avatar,
		Role:      u.Role,
		IsPremium: u.IsPremium,
		CreatedAt: u.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func buildUserAuthResponse(ua *user.UserAuth) dto.UserAuthResponse {
	return dto.UserAuthResponse{
		ID:              ua.ID,
		UserID:          ua.UserID,
		IsEmailVerified: ua.IsEmailVerified,
		IsTOTPEnabled:   ua.IsTOTPEnabled,
		LastLoginAt:     formatTimeOrEmpty(ua.LastLoginAt),
	}
}

func (c *Controller) completeLogin(ctx *gin.Context, u *user.User, ua *user.UserAuth, message string) error {
	deviceID, lastIP := ip.GetDeviceAndIPInfo(ctx)

	sessionID, err := middleware.CreateSession(
		context.Background(),
		u.ID,
		"login",
		ctx.ClientIP(),
		ctx.GetHeader("User-Agent"),
		*deviceID,
	)
	if err != nil {
		logger.Logger.Error("Failed to create session in Redis",
			"user_id", u.ID,
			"error", err.Error(),
		)
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "SESSION_CREATION_FAILED", "Failed to create session", "server")
		return err
	}

	token, err := pkgauth.GenerateJWT(u.ID, sessionID, *deviceID, *lastIP, u.Username, u.Email, u.Role, u.IsPremium, ua.IsEmailVerified)
	if err != nil {
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "JWT_GENERATION_FAILED", "Failed to generate authentication token", "auth")
		return err
	}

	sessionManager := middleware.GetSessionManager()
	refreshToken, err := pkgauth.GenerateRefreshToken(
		context.Background(),
		sessionManager.GetRedisClient(),
		u.ID,
		sessionID,
		*deviceID,
		*lastIP,
	)
	if err != nil {
		logger.Logger.Error("Failed to generate refresh token",
			"user_id", u.ID,
			"error", err.Error(),
		)
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "REFRESH_TOKEN_GENERATION_FAILED", "Failed to generate refresh token", "auth")
		return err
	}

	if err := c.repo.GetUserAuthRepository().UpdateLastLogin(u.ID, *deviceID, *lastIP); err != nil {
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "LAST_LOGIN_UPDATE_FAILED", "Failed to update last login", "auth")
		return err
	}

	go func() {
		userAgent := ctx.GetHeader("User-Agent")
		clientIP := ctx.ClientIP()
		c.emailService.SendLoginAlertEmail(u.Email, u.Username, clientIP, userAgent)
	}()

	cookieSettings := pkgauth.ResolveAuthCookieSettings(ctx)

	accessTokenCookie := &http.Cookie{
		Name:     "access_token",
		Value:    token,
		Path:     "/",
		Domain:   cookieSettings.Domain,
		MaxAge:   config.GetEnvAsInt(config.EnvJWTExpired, 24) * 3600,
		Secure:   cookieSettings.Secure,
		HttpOnly: true,
		SameSite: cookieSettings.SameSite,
	}

	refreshTokenCookie := &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     "/",
		Domain:   cookieSettings.Domain,
		MaxAge:   config.GetEnvAsInt(config.EnvRefreshTokenExpired, 168) * 3600,
		Secure:   cookieSettings.Secure,
		HttpOnly: true,
		SameSite: cookieSettings.SameSite,
	}

	http.SetCookie(ctx.Writer, accessTokenCookie)
	http.SetCookie(ctx.Writer, refreshTokenCookie)

	logger.Logger.Info("Authentication cookies set successfully",
		"user_id", u.ID,
		"secure", cookieSettings.Secure,
		"domain", cookieSettings.Domain,
		"same_site", cookieSettings.SameSiteLabel,
		"access_token_max_age_hours", config.GetEnvAsInt(config.EnvJWTExpired, 24),
		"refresh_token_max_age_hours", config.GetEnvAsInt(config.EnvRefreshTokenExpired, 168),
	)

	responseData := dto.LoginResponse{
		User: buildUserProfile(u),
		Auth: buildUserAuthResponse(ua),
	}

	if message == "" {
		message = "Login successful"
	}
	if ua.LastLoginAt == nil {
		now := time.Now()
		ua.LastLoginAt = &now
	}
	httputil.SendOKResponse(ctx, responseData, message)
	return nil
}

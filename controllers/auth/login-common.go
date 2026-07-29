package auth

import (
	"context"
	"errors"
	"net/http"

	"github.com/adehusnim37/lihatin-go/dto"
	pkgauth "github.com/adehusnim37/lihatin-go/internal/pkg/auth"
	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/ip"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/middleware"
	"github.com/adehusnim37/lihatin-go/models/user"
	"github.com/adehusnim37/lihatin-go/repositories/authrepo"
	"github.com/gin-gonic/gin"
)

func buildUserProfile(u *user.User) dto.UserProfile {
	return dto.UserProfile{
		ID:            u.ID,
		Username:      u.Username,
		FirstName:     u.FirstName,
		LastName:      u.LastName,
		Email:         u.Email,
		Avatar:        u.Avatar,
		Role:          u.Role,
		PremiumAccess: dto.NewPremiumAccessResponse(u.PremiumAccess),
		CreatedAt:     u.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func buildUserAuthResponse(ua *user.UserAuth) dto.UserAuthResponse {
	return dto.UserAuthResponse{
		ID:              ua.ID,
		UserID:          ua.UserID,
		IsEmailVerified: ua.IsEmailVerified,
		TOTPEnabled:     ua.HasEnabledTOTP(),
		AccountStatus:   string(ua.AccountStatus),
		LastLoginAt:     formatTimeOrEmpty(ua.LastLoginAt),
	}
}

// CompleteLogin is the single finalization path for every completed
// authentication method. Challenges and refreshes must not call this method.
func (c *Controller) CompleteLogin(ctx *gin.Context, u *user.User, ua *user.UserAuth, method user.LoginMethod, message string) error {
	cookieSettings := pkgauth.ResolveAuthCookieSettings(ctx)
	if cookieSettings.RejectInsecureRequest {
		httputil.SendErrorResponse(ctx, http.StatusForbidden, "INSECURE_TRANSPORT", "HTTPS is required in production", "auth")
		return errors.New("insecure transport is not allowed in production")
	}

	deviceID, lastIP := ip.GetDeviceAndIPInfo(ctx)
	requestContext := ctx.Request.Context()

	sessionID, err := middleware.CreateSession(
		requestContext,
		u.ID,
		"login",
		*lastIP,
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
	loginCompleted := false
	defer func() {
		if !loginCompleted {
			if cleanupErr := middleware.DeleteSession(context.Background(), sessionID); cleanupErr != nil {
				logger.Logger.Warn("Failed to clean up incomplete login session",
					"user_id", u.ID,
					"error", cleanupErr.Error(),
				)
			}
		}
	}()

	token, err := pkgauth.GenerateJWT(u.ID, sessionID, *deviceID, *lastIP, u.Username, u.Email, u.Role, ua.IsEmailVerified)
	if err != nil {
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "JWT_GENERATION_FAILED", "Failed to generate authentication token", "auth")
		return err
	}

	sessionManager := middleware.GetSessionManager()
	refreshToken, err := pkgauth.GenerateRefreshToken(
		requestContext,
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

	loginEvent, previousLogin, err := c.repo.GetLoginEventRepository().RecordSuccessfulLogin(authrepo.SuccessfulLogin{
		UserID:    u.ID,
		SessionID: sessionID,
		Method:    method,
		DeviceID:  *deviceID,
		IPAddress: *lastIP,
		UserAgent: ctx.GetHeader("User-Agent"),
	})
	if err != nil {
		refreshTokenManager := pkgauth.NewRefreshTokenManager(sessionManager.GetRedisClient())
		if cleanupErr := refreshTokenManager.DeleteRefreshToken(context.Background(), refreshToken); cleanupErr != nil {
			logger.Logger.Warn("Failed to clean up refresh token after login persistence failure",
				"user_id", u.ID,
				"error", cleanupErr.Error(),
			)
		}
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "LAST_LOGIN_UPDATE_FAILED", "Failed to update last login", "auth")
		return err
	}
	ua.LastLoginAt = &loginEvent.AuthenticatedAt

	userAgent := ctx.GetHeader("User-Agent")
	clientIP := *lastIP
	go func() {
		c.emailService.SendLoginAlertEmail(u.Email, u.Username, clientIP, userAgent)
	}()

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

	authResponse := buildUserAuthResponse(ua)
	authResponse.PreviousLogin = dto.NewLoginEventResponse(previousLogin)
	responseData := dto.LoginResponse{
		User: buildUserProfile(u),
		Auth: authResponse,
	}

	if message == "" {
		message = "Login successful"
	}
	loginCompleted = true
	httputil.SendOKResponse(ctx, responseData, message)
	return nil
}

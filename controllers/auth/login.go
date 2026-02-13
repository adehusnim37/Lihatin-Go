package auth

import (
	"context"
	"net/http"
	"time"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/internal/pkg/auth"
	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/ip"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	"github.com/adehusnim37/lihatin-go/middleware"
	"github.com/gin-gonic/gin"
)

// Login authenticates a user with email/username and password
func (c *Controller) Login(ctx *gin.Context) {
	var loginReq dto.LoginRequest

	// Bind and validate the request body
	if err := ctx.ShouldBindJSON(&loginReq); err != nil {
		validator.SendValidationError(ctx, err, &loginReq)
		return
	}

	// Get user and auth data from database
	user, userAuth, err := c.repo.GetUserAuthRepository().GetUserForLogin(loginReq.EmailOrUsername)
	if err != nil {
		// Increment failed login attempt regardless of whether user exists
		if user != nil {
			c.repo.GetUserAuthRepository().IncrementFailedLogin(user.ID)
		}
		httputil.HandleError(ctx, err, nil)
		return
	}

	// Check if account is locked
	isLocked, err := c.repo.GetUserAuthRepository().IsAccountLocked(user.ID)
	if err != nil {
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "LOGIN_FAILED", "An error occurred during login", "auth")
		return
	}

	if isLocked {
		httputil.SendErrorResponse(ctx, http.StatusForbidden, "ACCOUNT_LOCKED", "Your account is locked. Please try again later.", "auth")
		return
	}

	// Check if account is active
	if !userAuth.IsActive {
		httputil.SendErrorResponse(ctx, http.StatusForbidden, "ACCOUNT_DEACTIVATED", "Your account has been deactivated. Please contact support.", "auth")
		return
	}

	if !userAuth.IsEmailVerified {
		httputil.SendErrorResponse(ctx, http.StatusForbidden, "EMAIL_NOT_VERIFIED", "Your email address is not verified. Please verify your email to proceed.", "email")
		return
	}

	// Check password
	if err := auth.CheckPassword(userAuth.PasswordHash, loginReq.Password); err != nil {
		// Increment failed login attempts
		c.repo.GetUserAuthRepository().IncrementFailedLogin(user.ID)

		httputil.SendErrorResponse(ctx, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid email/username or password", "auth")
		return
	}

	// Check if TOTP is enabled for the user
	hasTOTP, err := c.repo.GetAuthMethodRepository().HasTOTPEnabled(userAuth.ID)
	if err != nil {
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "LOGIN_FAILED", "An error occurred during login", "auth")
		return
	}

	// If TOTP is enabled, DO NOT issue JWT tokens yet
	// Return a pending auth token that must be verified with TOTP
	if hasTOTP {
		// Generate pending auth token and store in Redis (5 min expiry)
		pendingToken, err := auth.GeneratePendingAuthToken(context.Background(), user.ID)
		if err != nil {
			logger.Logger.Error("Failed to generate pending auth token",
				"user_id", user.ID,
				"error", err.Error(),
			)
			httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "PENDING_TOKEN_GENERATION_FAILED", "Failed to generate authentication token", "auth")
			return
		}

		// Return pending response - NO JWT tokens!
		pendingResponse := dto.PendingTOTPResponse{
			RequiresTOTP:     true,
			PendingAuthToken: pendingToken,
			User: dto.UserProfile{
				ID:        user.ID,
				Username:  user.Username,
				FirstName: user.FirstName,
				LastName:  user.LastName,
				Email:     user.Email,
				Avatar:    user.Avatar,
				IsPremium: user.IsPremium,
				CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			},
		}

		logger.Logger.Info("TOTP verification required for login",
			"user_id", user.ID,
			"pending_token_preview", auth.GetKeyPreview(pendingToken),
		)

		httputil.SendOKResponse(ctx, pendingResponse, "Password verified. Please complete two-factor authentication.")
		return
	}

	// NO TOTP - proceed with normal login flow
	// Get device and IP info
	deviceID, lastIP := ip.GetDeviceAndIPInfo(ctx)

	// Create session in Redis using middleware helper
	sessionID, err := middleware.CreateSession(
		context.Background(),
		user.ID,
		"login",
		ctx.ClientIP(),
		ctx.GetHeader("User-Agent"),
		*deviceID,
	)
	if err != nil {
		logger.Logger.Error("Failed to create session in Redis",
			"user_id", user.ID,
			"error", err.Error(),
		)
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "SESSION_CREATION_FAILED", "Failed to create session", "server")
		return
	}

	logger.Logger.Info("Session created successfully",
		"user_id", user.ID,
		"session_preview", auth.GetKeyPreview(sessionID),
		"device_id", *deviceID,
	)

	// Generate JWT token
	role := user.Role
	token, err := auth.GenerateJWT(user.ID, sessionID, *deviceID, *lastIP, user.Username, user.Email, role, user.IsPremium, userAuth.IsEmailVerified)
	if err != nil {
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "JWT_GENERATION_FAILED", "Failed to generate authentication token", "auth")
		return
	}

	// Generate refresh token and store in Redis
	sessionManager := middleware.GetSessionManager()
	refreshToken, err := auth.GenerateRefreshToken(
		context.Background(),
		sessionManager.GetRedisClient(),
		user.ID,
		sessionID,
		*deviceID,
		*lastIP,
	)
	if err != nil {
		logger.Logger.Error("Failed to generate refresh token",
			"user_id", user.ID,
			"error", err.Error(),
		)
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "REFRESH_TOKEN_GENERATION_FAILED", "Failed to generate refresh token", "auth")
		return
	}

	// Update last login and reset failed attempts
	if err := c.repo.GetUserAuthRepository().UpdateLastLogin(user.ID, *deviceID, *lastIP); err != nil {
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "LAST_LOGIN_UPDATE_FAILED", "Failed to update last login", "auth")
		return
	}

	// Send login alert email (async)
	go func() {
		userAgent := ctx.GetHeader("User-Agent")
		clientIP := ctx.ClientIP()
		c.emailService.SendLoginAlertEmail(user.Email, user.FirstName, clientIP, userAgent)
	}()

	cookieSettings := auth.ResolveAuthCookieSettings(ctx)

	// Set Access Token Cookie with proper SameSite for CORS
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

	// Set Refresh Token Cookie with proper SameSite for CORS
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

	// Apply cookies to response
	http.SetCookie(ctx.Writer, accessTokenCookie)
	http.SetCookie(ctx.Writer, refreshTokenCookie)

	// Log successful cookie setting (but not the actual token values)
	logger.Logger.Info("Authentication cookies set successfully",
		"user_id", user.ID,
		"secure", cookieSettings.Secure,
		"domain", cookieSettings.Domain,
		"same_site", cookieSettings.SameSiteLabel,
		"access_token_max_age_hours", config.GetEnvAsInt(config.EnvJWTExpired, 24),
		"refresh_token_max_age_hours", config.GetEnvAsInt(config.EnvRefreshTokenExpired, 168),
	)

	// Prepare response data
	responseData := dto.LoginResponse{
		User: dto.UserProfile{
			ID:        user.ID,
			Username:  user.Username,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			Email:     user.Email,
			Avatar:    user.Avatar,
			IsPremium: user.IsPremium,
			CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		},
		Auth: dto.UserAuthResponse{
			ID:              userAuth.ID,
			UserID:          userAuth.UserID,
			IsEmailVerified: userAuth.IsEmailVerified,
			IsTOTPEnabled:   userAuth.IsTOTPEnabled,
			LastLoginAt:     formatTimeOrEmpty(userAuth.LastLoginAt),
		},
	}

	httputil.SendOKResponse(ctx, responseData, "Login successful")
}

// Helper function to safely format time pointer
func formatTimeOrEmpty(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02T15:04:05Z07:00")
}

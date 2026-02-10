package totp

import (
	"context"
	"net/http"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/internal/pkg/auth"
	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/ip"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	"github.com/adehusnim37/lihatin-go/middleware"
	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/gin-gonic/gin"
)

// VerifyTOTPLogin verifies TOTP code during login and issues JWT tokens
// This is the ONLY way to get JWT tokens when TOTP is enabled
func (c *Controller) VerifyTOTPLogin(ctx *gin.Context) {
	var req dto.VerifyTOTPLoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		validator.SendValidationError(ctx, err, &req)
		return
	}

	// Validate pending auth token and get userID
	userID, err := auth.ValidatePendingAuthToken(context.Background(), req.PendingAuthToken)
	if err != nil {
		logger.Logger.Warn("Invalid pending auth token for TOTP login",
			"error", err.Error(),
		)
		ctx.JSON(http.StatusUnauthorized, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Authentication expired",
			Error:   map[string]string{"auth": "Your authentication session has expired. Please login again."},
		})
		return
	}

	// Get user and auth info
	user, err := c.repo.GetUserRepository().GetUserByID(userID)
	if err != nil {
		logger.Logger.Error("Failed to get user for TOTP login",
			"user_id", userID,
			"error", err.Error(),
		)
		httputil.HandleError(ctx, err, userID)
		return
	}

	userAuth, err := c.repo.GetUserAuthRepository().GetUserAuthByUserID(userID)
	if err != nil {
		logger.Logger.Error("Failed to get user auth for TOTP login",
			"user_id", userID,
			"error", err.Error(),
		)
		httputil.HandleError(ctx, err, userID)
		return
	}

	// Get TOTP secret
	encryptedSecret, err := c.repo.GetAuthMethodRepository().GetTOTPSecret(userAuth.ID)
	if err != nil {
		logger.Logger.Error("Failed to get TOTP secret for login",
			"user_id", userID,
			"error", err.Error(),
		)
		httputil.HandleError(ctx, err, userID)
		return
	}

	secret, err := auth.DecryptTOTPSecret(encryptedSecret)
	if err != nil {
		logger.Logger.Error("Failed to decrypt TOTP secret for login",
			"user_id", userID,
			"error", err.Error(),
		)
		httputil.HandleError(ctx, err, userID)
		return
	}

	// Validate TOTP code
	if !auth.ValidateTOTPCodeWithWindow(secret, req.TOTPCode, 1) {
		logger.Logger.Warn("Invalid TOTP code during login",
			"user_id", userID,
		)
		httputil.SendErrorResponse(ctx, http.StatusBadRequest, "INVALID_TOTP", "Invalid verification code", "totp")
		return
	}

	// TOTP verified! Now issue JWT tokens (same as normal login)
	deviceID, lastIP := ip.GetDeviceAndIPInfo(ctx)

	// Create session in Redis
	sessionID, err := middleware.CreateSession(
		context.Background(),
		user.ID,
		"login",
		ctx.ClientIP(),
		ctx.GetHeader("User-Agent"),
		*deviceID,
	)
	if err != nil {
		logger.Logger.Error("Failed to create session after TOTP verification",
			"user_id", user.ID,
			"error", err.Error(),
		)
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Message: "Failed to create session",
			Error:   map[string]string{"server": "Session creation failed"},
		})
		return
	}

	logger.Logger.Info("Session created after TOTP verification",
		"user_id", user.ID,
		"session_preview", auth.GetKeyPreview(sessionID),
		"device_id", *deviceID,
	)

	// Generate JWT token
	token, err := auth.GenerateJWT(user.ID, sessionID, *deviceID, *lastIP, user.Username, user.Email, user.Role, user.IsPremium, userAuth.IsEmailVerified)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Login failed",
			Error:   map[string]string{"error": "Failed to generate authentication token"},
		})
		return
	}

	// Generate refresh token
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
		logger.Logger.Error("Failed to generate refresh token after TOTP",
			"user_id", user.ID,
			"error", err.Error(),
		)
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Login failed",
			Error:   map[string]string{"error": "Failed to generate refresh token"},
		})
		return
	}

	// Update last login
	if err := c.repo.GetUserAuthRepository().UpdateLastLogin(user.ID, *deviceID, *lastIP); err != nil {
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Login failed",
			Error:   map[string]string{"error": "Failed to update last login"},
		})
		return
	}

	// Update TOTP auth method last_used_at
	if err := c.repo.GetAuthMethodRepository().UpdateTOTPLastUsed(userAuth.ID); err != nil {
		logger.Logger.Warn("Failed to update TOTP last_used_at",
			"user_id", user.ID,
			"error", err.Error(),
		)
		// Don't fail login for this
	}

	// Send login alert email (async)
	go func() {
		userAgent := ctx.GetHeader("User-Agent")
		clientIP := ctx.ClientIP()
		c.emailService.SendLoginAlertEmail(user.Email, user.FirstName, clientIP, userAgent)
	}()

	// Set cookies with proper SameSite for CORS
	isSecure := ctx.Request.TLS != nil || ctx.GetHeader("X-Forwarded-Proto") == "https"

	// Get domain and set cookie domain
	domain := config.GetEnvOrDefault(config.EnvDomain, "localhost")

	// For production with subdomains, use root domain with dot prefix
	if domain != "localhost" && domain != "127.0.0.1" {
		domain = "." + domain
	}

	// Set Access Token Cookie
	accessTokenCookie := &http.Cookie{
		Name:     "access_token",
		Value:    token,
		Path:     "/",
		Domain:   domain,
		MaxAge:   config.GetEnvAsInt(config.EnvJWTExpired, 24) * 3600,
		Secure:   isSecure,
		HttpOnly: true,
		SameSite: http.SameSiteNoneMode,
	}

	// Set Refresh Token Cookie
	refreshTokenCookie := &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     "/",
		Domain:   domain,
		MaxAge:   config.GetEnvAsInt(config.EnvRefreshTokenExpired, 168) * 3600,
		Secure:   isSecure,
		HttpOnly: true,
		SameSite: http.SameSiteNoneMode,
	}

	// Apply cookies to response
	http.SetCookie(ctx.Writer, accessTokenCookie)
	http.SetCookie(ctx.Writer, refreshTokenCookie)

	logger.Logger.Info("TOTP login successful, tokens issued",
		"user_id", user.ID,
		"secure", isSecure,
		"domain", domain,
		"same_site", "None",
	)

	// Prepare response
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
			LastLoginAt:     userAuth.LastLoginAt.Format("2006-01-02T15:04:05Z07:00"),
		},
	}

	httputil.SendOKResponse(ctx, responseData, "Two-factor authentication successful. Welcome back!")
}

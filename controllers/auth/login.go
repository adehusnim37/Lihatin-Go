package auth

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/middleware"
	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/adehusnim37/lihatin-go/internal/pkg/auth"
	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	"github.com/adehusnim37/lihatin-go/internal/pkg/ip"
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

		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusUnauthorized, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Invalid credentials",
				Error:   map[string]string{"auth": "Invalid email/username or password"},
			})
			return
		}

		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Login failed",
			Error:   map[string]string{"error": "An error occurred during login, please try again later"},
		})
		return
	}

	// Check if account is locked
	isLocked, err := c.repo.GetUserAuthRepository().IsAccountLocked(user.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Login failed",
			Error:   map[string]string{"error": "An error occurred during login"},
		})
		return
	}

	if isLocked {
		ctx.JSON(http.StatusTooManyRequests, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Account temporarily locked",
			Error:   map[string]string{"auth": "Too many failed login attempts. Please try again later."},
		})
		return
	}

	// Check if account is locked
	if user.IsLocked {
		ctx.JSON(http.StatusForbidden, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Account locked",
			Error:   map[string]string{"auth": "Your account has been locked. Please contact support."},
		})
		return
	}

	// Check if account is active
	if !userAuth.IsActive {
		ctx.JSON(http.StatusForbidden, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Account deactivated",
			Error:   map[string]string{"auth": "Your account has been deactivated. Please contact support."},
		})
		return
	}

	// Check password
	if err := auth.CheckPassword(userAuth.PasswordHash, loginReq.Password); err != nil {
		// Increment failed login attempts
		c.repo.GetUserAuthRepository().IncrementFailedLogin(user.ID)

		ctx.JSON(http.StatusUnauthorized, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid credentials",
			Error:   map[string]string{"auth": "Invalid email/username or password"},
		})
		return
	}

	// Check if TOTP is enabled for the user
	hasTOTP, err := c.repo.GetAuthMethodRepository().HasTOTPEnabled(userAuth.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Login failed",
			Error:   map[string]string{"error": "An error occurred during login"},
		})
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
			ctx.JSON(http.StatusInternalServerError, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Login failed",
				Error:   map[string]string{"error": "Failed to generate authentication token"},
			})
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
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Message: "Failed to create session",
			Error:   map[string]string{"server": "Session creation failed"},
		})
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
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Login failed",
			Error:   map[string]string{"error": "Failed to generate authentication token"},
		})
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
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Login failed",
			Error:   map[string]string{"error": "Failed to generate refresh token"},
		})
		return
	}

	// Update last login and reset failed attempts
	if err := c.repo.GetUserAuthRepository().UpdateLastLogin(user.ID, *deviceID, *lastIP); err != nil {
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Login failed",
			Error:   map[string]string{"error": "Failed to update last login"},
		})
		return
	}

	// Send login alert email (async)
	go func() {
		userAgent := ctx.GetHeader("User-Agent")
		clientIP := ctx.ClientIP()
		c.emailService.SendLoginAlertEmail(user.Email, user.FirstName, clientIP, userAgent)
	}()

	// Determine if we're in production (HTTPS) or development (HTTP)
	isSecure := ctx.Request.TLS != nil || ctx.GetHeader("X-Forwarded-Proto") == "https"

	// Set Access Token Cookie (HttpOnly, Secure in prod, Short-lived)
	// Domain set to localhost for cross-port development (3000 <-> 8080)
	ctx.SetCookie(
		"access_token", // Name
		token,          // Value
		config.GetEnvAsInt(config.EnvJWTExpired, 24)*3600, // MaxAge in seconds (default 24 hours)
		"/", // Path
		config.GetEnvOrDefault(config.EnvDomain, "localhost"), // Domain (localhost for dev, change to actual domain in prod)
		isSecure, // Secure (true in production/HTTPS)
		true,     // HttpOnly (MUST be true for security)
	)

	// Set Refresh Token Cookie (HttpOnly, Secure in prod, Long-lived)
	ctx.SetCookie(
		"refresh_token", // Name
		refreshToken,    // Value
		config.GetEnvAsInt(config.EnvRefreshTokenExpired, 168)*3600, // MaxAge in seconds (default 7 days)
		"/", // Path
		config.GetEnvOrDefault(config.EnvDomain, "localhost"), // Domain (localhost for dev, change to actual domain in prod)
		isSecure, // Secure (true in production/HTTPS)
		true,     // HttpOnly (MUST be true for security)
	)

	// Log successful cookie setting (but not the actual token values)
	logger.Logger.Info("Authentication cookies set successfully",
		"user_id", user.ID,
		"secure", isSecure,
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
			LastLoginAt:     userAuth.LastLoginAt.Format("2006-01-02T15:04:05Z07:00"),
		},
	}

	httputil.SendOKResponse(ctx, responseData, "Login successful")
}

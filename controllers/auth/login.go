package auth

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/middleware"
	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/adehusnim37/lihatin-go/utils"
	clientip "github.com/adehusnim37/lihatin-go/utils/clientip"
	"github.com/gin-gonic/gin"
)

// Login authenticates a user with email/username and password
func (c *Controller) Login(ctx *gin.Context) {
	var loginReq dto.LoginRequest

	// Bind and validate the request body
	if err := ctx.ShouldBindJSON(&loginReq); err != nil {
		utils.SendValidationError(ctx, err, &loginReq)
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
	if err := utils.CheckPassword(userAuth.PasswordHash, loginReq.Password); err != nil {
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

	// Get device and IP info
	deviceID, lastIP := clientip.GetDeviceAndIPInfo(ctx)

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
		utils.Logger.Error("Failed to create session in Redis",
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

	utils.Logger.Info("Session created successfully",
		"user_id", user.ID,
		"session_preview", utils.GetKeyPreview(sessionID),
		"device_id", *deviceID,
	)

	// Generate JWT token
	role := user.Role
	token, err := utils.GenerateJWT(user.ID, sessionID, *deviceID, *lastIP, user.Username, user.Email, role, user.IsPremium, userAuth.IsEmailVerified)
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
	refreshToken, err := utils.GenerateRefreshToken(
		context.Background(),
		sessionManager.GetRedisClient(),
		user.ID,
		sessionID,
		*deviceID,
		*lastIP,
	)
	if err != nil {
		utils.Logger.Error("Failed to generate refresh token",
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
	if err := c.repo.GetUserAuthRepository().UpdateLastLogin(user.ID); err != nil {
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

	ctx.SetCookie(
		"access_token", // Nama Cookie
		token,          // Nilai Token
		2000,           // MaxAge (dalam detik)
		"/",            // Path: Tersedia untuk seluruh aplikasi
		"",             // Domain: Kosong berarti domain saat ini
		false,          // Secure: True if using HTTPS
		true,           // HttpOnly: MUST be TRUE to prevent JS from reading
	)

	ctx.SetCookie(
		"refresh_token", // Nama Cookie
		refreshToken,    // Nilai Token
		604800,          // MaxAge (dalam detik) - 7 days
		"/",             // Path: Tersedia untuk seluruh aplikasi
		"",              // Domain: Kosong berarti domain saat ini
		false,           // Secure: True if using HTTPS
		true,            // HttpOnly: MUST be TRUE to prevent JS from reading
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

	message := "Login successful"
	if hasTOTP {
		message = "Login successful. Please complete two-factor authentication."
	}

	utils.SendOKResponse(ctx, responseData, message)
}

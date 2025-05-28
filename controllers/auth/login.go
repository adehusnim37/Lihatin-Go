package auth

import (
	"database/sql"
	"net/http"

	"github.com/adehusnim37/lihatin-go/models"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// Login authenticates a user with email/username and password
func (c *Controller) Login(ctx *gin.Context) {
	var loginReq models.LoginRequest

	// Bind and validate the request body
	if err := ctx.ShouldBindJSON(&loginReq); err != nil {
		ctx.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid input",
			Error:   map[string]string{"input": "Invalid input format or missing fields, please check your request body"},
		})
		return
	}

	// Validate the login request
	if err := c.Validate.Struct(loginReq); err != nil {
		errorMap := make(map[string]string)
		for _, err := range err.(validator.ValidationErrors) {
			field := err.Field()
			switch err.Tag() {
			case "required":
				errorMap[field] = field + " is required"
			case "min":
				errorMap[field] = field + " must be at least " + err.Param() + " characters"
			case "max":
				errorMap[field] = field + " must not exceed " + err.Param() + " characters"
			default:
				errorMap[field] = "Invalid value for " + field
			}
		}

		ctx.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Validation failed",
			Error:   errorMap,
		})
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
			ctx.JSON(http.StatusUnauthorized, models.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Invalid credentials",
				Error:   map[string]string{"auth": "Invalid email/username or password"},
			})
			return
		}

		ctx.JSON(http.StatusInternalServerError, models.APIResponse{
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
		ctx.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Login failed",
			Error:   map[string]string{"error": "An error occurred during login"},
		})
		return
	}

	if isLocked {
		ctx.JSON(http.StatusTooManyRequests, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Account temporarily locked",
			Error:   map[string]string{"auth": "Too many failed login attempts. Please try again later."},
		})
		return
	}

	// Check if account is active
	if !userAuth.IsActive {
		ctx.JSON(http.StatusForbidden, models.APIResponse{
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
		
		ctx.JSON(http.StatusUnauthorized, models.APIResponse{
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
		ctx.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Login failed",
			Error:   map[string]string{"error": "An error occurred during login"},
		})
		return
	}

	// Generate JWT token
	role := map[bool]string{true: "premium", false: "regular"}[user.IsPremium]
	token, err := utils.GenerateJWT(user.ID, user.Username, user.Email, role, user.IsPremium, userAuth.IsEmailVerified)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Login failed",
			Error:   map[string]string{"error": "Failed to generate authentication token"},
		})
		return
	}

	// Generate refresh token
	refreshToken, err := utils.GenerateRefreshToken(user.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Login failed",
			Error:   map[string]string{"error": "Failed to generate refresh token"},
		})
		return
	}

	// Update last login and reset failed attempts
	if err := c.repo.GetUserAuthRepository().UpdateLastLogin(user.ID); err != nil {
		// Log error but don't fail the login
		// logger.Error("Failed to update last login", err)
	}

	// Create user profile (without sensitive information)
	userProfile := models.UserProfile{
		ID:        user.ID,
		Username:  user.Username,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Email:     user.Email,
		Avatar:    user.Avatar,
		IsPremium: user.IsPremium,
		CreatedAt: user.CreatedAt,
	}

	// Set user context for logging
	ctx.Set("username", user.Username)
	ctx.Set("user_id", user.ID)
	ctx.Set("role", role)

	// Send login alert email (async)
	go func() {
		userAgent := ctx.GetHeader("User-Agent")
		clientIP := ctx.ClientIP()
		c.emailService.SendLoginAlertEmail(user.Email, user.FirstName, clientIP, userAgent)
	}()

	// Prepare response data
	responseData := map[string]interface{}{
		"user":          userProfile,
		"token":         token,
		"refresh_token": refreshToken,
		"requires_2fa":  hasTOTP,
		"is_verified":   userAuth.IsEmailVerified,
	}

	if hasTOTP {
		responseData["message"] = "Login successful. Please complete two-factor authentication."
	} else {
		responseData["message"] = "Login successful"
	}

	ctx.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    responseData,
		Message: "Login successful",
		Error:   nil,
	})
}

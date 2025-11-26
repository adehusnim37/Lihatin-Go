package auth

import (
	"context"
	"net/http"

	"github.com/adehusnim37/lihatin-go/controllers"
	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/middleware"
	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/adehusnim37/lihatin-go/repositories"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/adehusnim37/lihatin-go/utils/mail"
	"github.com/gin-gonic/gin"
)

// Controller provides all handlers for authentication operations
type Controller struct {
	*controllers.BaseController
	repo         *repositories.AuthRepository
	emailService *mail.EmailService
}

// NewAuthController creates a new auth controller instance
func NewAuthController(base *controllers.BaseController) *Controller {
	if base.GormDB == nil {
		panic("GormDB is required for AuthController")
	}

	authRepo := repositories.NewAuthRepository(base.GormDB)
	emailService := mail.NewEmailService()

	return &Controller{
		BaseController: base,
		repo:           authRepo,
		emailService:   emailService,
	}
}



// GetPremiumStatus returns premium subscription status
func (c *Controller) GetPremiumStatus(ctx *gin.Context) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Authentication required",
			Error:   map[string]string{"auth": "Please login to access this feature"},
		})
		return
	}

	// Get user information
	user, err := c.repo.GetUserRepository().GetUserByID(userID.(string))
	if err != nil {
		ctx.JSON(http.StatusNotFound, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "User not found",
			Error:   map[string]string{"user": "User not found"},
		})
		return
	}

	// Create premium status response
	premiumStatus := map[string]interface{}{
		"user_id":           user.ID,
		"is_premium":        user.IsPremium,
		"premium_since":     nil, // TODO: Add premium_since field to user model
		"subscription_type": map[bool]string{true: "premium", false: "free"}[user.IsPremium],
		"features": map[string]interface{}{
			"unlimited_uploads":  user.IsPremium,
			"priority_support":   user.IsPremium,
			"advanced_analytics": user.IsPremium,
			"custom_branding":    user.IsPremium,
		},
	}

	ctx.JSON(http.StatusOK, common.APIResponse{
		Success: true,
		Data:    premiumStatus,
		Message: "Premium status retrieved successfully",
		Error:   nil,
	})
}

// LockUser locks a user account (admin only)
func (c *Controller) LockUser(ctx *gin.Context) {
	userID := ctx.Param("user_id")
	if userID == "" {
		ctx.JSON(http.StatusBadRequest, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "User ID is required",
			Error:   map[string]string{"user_id": "User ID parameter is required"},
		})
		return
	}

	var req dto.AdminLockUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.SendValidationError(ctx, err, &req)
		return
	}

	// Validate request
	if err := ctx.ShouldBindUri(&req); err != nil {
		utils.SendValidationError(ctx, err, &req)
		return
	}

	// Check if user exists
	user, err := c.repo.GetUserRepository().GetUserByID(userID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "User not found",
			Error:   map[string]string{"user_id": "User with this ID does not exist"},
		})
		return
	}

	// Check if user is already locked
	if user.IsLocked {
		ctx.JSON(http.StatusConflict, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "User already locked",
			Error:   map[string]string{"status": "User account is already locked"},
		})
		return
	}

	// Lock the user
	if err := c.repo.GetUserAdminRepository().LockUser(userID, req.Reason); err != nil {
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to lock user",
			Error:   map[string]string{"error": "Failed to lock user account, please try again later"},
		})
		return
	}

	ctx.JSON(http.StatusOK, common.APIResponse{
		Success: true,
		Data:    nil,
		Message: "User account locked successfully",
		Error:   nil,
	})
}

// UnlockUser unlocks a user account (admin only)
func (c *Controller) UnlockUser(ctx *gin.Context) {
	userID := ctx.Param("user_id")
	if userID == "" {
		ctx.JSON(http.StatusBadRequest, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "User ID is required",
			Error:   map[string]string{"user_id": "User ID parameter is required"},
		})
		return
	}

	var req dto.AdminUnlockUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		// Allow empty body for unlock requests
		req = dto.AdminUnlockUserRequest{}
	}

	// Validate request if reason is provided
	if req.Reason != "" {
		if err := ctx.ShouldBindUri(&req); err != nil {
			utils.SendValidationError(ctx, err, &req)
			return
		}
	}

	// Check if user exists
	user, err := c.repo.GetUserRepository().GetUserByID(userID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "User not found",
			Error:   map[string]string{"user_id": "User with this ID does not exist"},
		})
		return
	}

	// Check if user is actually locked
	if !user.IsLocked {
		ctx.JSON(http.StatusConflict, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "User not locked",
			Error:   map[string]string{"status": "User account is not currently locked"},
		})
		return
	}

	// Unlock the user
	if err := c.repo.GetUserAdminRepository().UnlockUser(userID, req.Reason); err != nil {
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to unlock user",
			Error:   map[string]string{"error": "Failed to unlock user account, please try again later"},
		})
		return
	}

	ctx.JSON(http.StatusOK, common.APIResponse{
		Success: true,
		Data:    nil,
		Message: "User account unlocked successfully",
		Error:   nil,
	})
}

// RefreshToken generates new access token from refresh token (Redis-based)
// 🔐 SECURITY: Reads refresh_token from HTTP-Only cookie (not JSON body) to prevent XSS attacks
// Returns new tokens via Set-Cookie headers only (never in response body)
func (c *Controller) RefreshToken(ctx *gin.Context) {
	// Read refresh token from HTTP-Only cookie (secure approach)
	refreshTokenCookie, err := ctx.Cookie("refresh_token")
	if err != nil || refreshTokenCookie == "" {
		ctx.JSON(http.StatusUnauthorized, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Refresh token missing",
			Error:   map[string]string{"refresh_token": "Please login again"},
		})
		return
	}

	// Get Redis client
	sessionManager := middleware.GetSessionManager()
	redisClient := sessionManager.GetRedisClient()

	// Validate refresh token from Redis
	refreshTokenManager := utils.NewRefreshTokenManager(redisClient)
	tokenData, err := refreshTokenManager.GetRefreshToken(context.Background(), refreshTokenCookie)
	if err != nil {
		utils.Logger.Warn("Invalid refresh token",
			"error", err.Error(),
		)
		ctx.JSON(http.StatusUnauthorized, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid or expired refresh token",
			Error:   map[string]string{"refresh_token": "Please login again"},
		})
		return
	}

	// Get user details
	user, err := c.repo.GetUserRepository().GetUserByID(tokenData.UserID)
	if err != nil {
		utils.Logger.Error("Failed to get user for refresh token",
			"user_id", tokenData.UserID,
			"error", err.Error(),
		)
		ctx.JSON(http.StatusUnauthorized, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "User not found",
			Error:   map[string]string{"user": "Invalid user"},
		})
		return
	}

	// Get user auth info
	userAuth, err := c.repo.GetUserAuthRepository().GetUserAuthByUserID(user.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to get user authentication info",
			Error:   map[string]string{"error": "Internal server error"},
		})
		return
	}

	// Generate new JWT token with same session
	role := user.Role
	newToken, err := utils.GenerateJWT(
		user.ID,
		tokenData.SessionID,
		tokenData.DeviceID,
		tokenData.LastIP,
		user.Username,
		user.Email,
		role,
		user.IsPremium,
		userAuth.IsEmailVerified,
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to generate new token",
			Error:   map[string]string{"error": "Token generation failed"},
		})
		return
	}

	// Rotate refresh token (delete old, create new) - security best practice
	err = refreshTokenManager.DeleteRefreshToken(context.Background(), refreshTokenCookie)
	if err != nil {
		utils.Logger.Warn("Failed to delete old refresh token",
			"user_id", tokenData.UserID,
			"error", err.Error(),
		)
	}

	// Generate new refresh token
	newRefreshToken, err := utils.GenerateRefreshToken(
		context.Background(),
		redisClient,
		user.ID,
		tokenData.SessionID,
		tokenData.DeviceID,
		tokenData.LastIP,
	)
	if err != nil {
		utils.Logger.Error("Failed to generate new refresh token",
			"user_id", user.ID,
			"error", err.Error(),
		)
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to generate refresh token",
			Error:   map[string]string{"error": "Refresh token generation failed"},
		})
		return
	}

	utils.Logger.Info("Token refreshed successfully",
		"user_id", user.ID,
		"session_id", utils.GetKeyPreview(tokenData.SessionID),
	)

	// ✅ Set tokens as HTTP-Only cookies (XSS protection)
	// Detect if running on HTTPS (check standard headers)
	isSecure := ctx.GetHeader("X-Forwarded-Proto") == "https" ||
		ctx.GetHeader("X-Forwarded-Ssl") == "on" ||
		ctx.Request.TLS != nil

	// Access token cookie (short-lived: default 48 hours)
	ctx.SetCookie(
		"access_token", // name
		newToken,       // value
		48*60*60,       // maxAge in seconds (48 hours)
		"/",            // path
		utils.GetEnvOrDefault(utils.EnvDomain, "localhost"), // domain (localhost for dev, change to actual domain in prod)
		isSecure, // secure (HTTPS only in production)
		true,     // httpOnly (prevent JavaScript access)
	)

	// Refresh token cookie (long-lived: default 168 hours = 7 days)
	ctx.SetCookie(
		"refresh_token", // name
		newRefreshToken, // value
		168*60*60,       // maxAge in seconds (168 hours = 7 days)
		"/",             // path
		utils.GetEnvOrDefault(utils.EnvDomain, "localhost"), // domain (localhost for dev, change to actual domain in prod)
		isSecure, // secure
		true,     // httpOnly
	)

	utils.Logger.Info("Tokens rotated and set as HTTP-Only cookies",
		"user_id", user.ID,
		"is_secure", isSecure,
	)

	// ⚠️ DO NOT return tokens in response body (security requirement)
	// Tokens are only accessible via HTTP-Only cookies
	ctx.JSON(http.StatusOK, common.APIResponse{
		Success: true,
		Data:    nil, // No token in response body for security
		Message: "Token refreshed successfully",
		Error:   nil,
	})
}

// GetCurrentUser returns the current authenticated user's information
// 🔐 Used by frontend to check authentication status via cookie validation
func (c *Controller) GetCurrentUser(ctx *gin.Context) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Authentication required",
			Error:   map[string]string{"auth": "Please login to access this feature"},
		})
		return
	}

	// Get user details
	user, err := c.repo.GetUserRepository().GetUserByID(userID.(string))
	if err != nil {
		utils.Logger.Error("Failed to get user for /me endpoint",
			"user_id", userID,
			"error", err.Error(),
		)
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to get user information",
			Error:   map[string]string{"error": "Internal server error"},
		})
		return
	}

	// Get user auth info
	userAuth, err := c.repo.GetUserAuthRepository().GetUserAuthByUserID(user.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to get user authentication info",
			Error:   map[string]string{"error": "Internal server error"},
		})
		return
	}

	// Convert to DTO
	userProfile := dto.UserProfile{
		ID:        user.ID,
		Username:  user.Username,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Email:     user.Email,
		Avatar:    user.Avatar,
		IsPremium: user.IsPremium,
		CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
	}

	authResponse := dto.UserAuthResponse{
		ID:              userAuth.ID,
		IsEmailVerified: userAuth.IsEmailVerified,
		IsTOTPEnabled:   userAuth.IsTOTPEnabled,
	}

	ctx.JSON(http.StatusOK, common.APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"user": userProfile,
			"auth": authResponse,
		},
		Message: "User information retrieved successfully",
		Error:   nil,
	})
}

// ChangePassword changes user password (requires current password)
func (c *Controller) ChangePassword(ctx *gin.Context) {
	userID := ctx.GetString("user_id")

	var req struct {
		CurrentPassword string `json:"current_password" validate:"required"`
		NewPassword     string `json:"new_password" validate:"required,min=8,max=50,pwdcomplex"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid input",
			Error:   map[string]string{"input": "Invalid input format"},
		})
		return
	}

	if err := c.Validate.Struct(req); err != nil {
		ctx.JSON(http.StatusBadRequest, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Validation failed",
			Error:   map[string]string{"password": "New password must be 8-50 characters with letters, numbers, and symbols"},
		})
		return
	}

	// Get user auth data
	userAuth, err := c.repo.GetUserAuthRepository().GetUserAuthByUserID(userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to get user data",
			Error:   map[string]string{"error": "Internal server error"},
		})
		return
	}

	// Verify current password
	if err := utils.CheckPassword(userAuth.PasswordHash, req.CurrentPassword); err != nil {
		ctx.JSON(http.StatusUnauthorized, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid current password",
			Error:   map[string]string{"password": "Current password is incorrect"},
		})
		return
	}

	// Hash new password
	hashedPassword, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Password change failed",
			Error:   map[string]string{"error": "Failed to process new password"},
		})
		return
	}

	// Update password
	if err := c.repo.GetUserAuthRepository().UpdatePassword(userID, hashedPassword); err != nil {
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Password change failed",
			Error:   map[string]string{"error": "Failed to update password"},
		})
		return
	}

	ctx.JSON(http.StatusOK, common.APIResponse{
		Success: true,
		Data:    nil,
		Message: "Password changed successfully",
		Error:   nil,
	})
}

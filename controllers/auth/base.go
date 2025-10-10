package auth

import (
	"context"
	"net/http"
	"strconv"

	"github.com/adehusnim37/lihatin-go/controllers"
	"github.com/adehusnim37/lihatin-go/middleware"
	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/adehusnim37/lihatin-go/models/user"
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

// GetRecoveryCodes returns TOTP recovery codes
func (c *Controller) GetRecoveryCodes(ctx *gin.Context) {
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

	// Get user auth record first
	userAuth, err := c.repo.GetUserAuthRepository().GetUserAuthByUserID(userID.(string))
	if err != nil {
		ctx.JSON(http.StatusNotFound, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "User authentication data not found",
			Error:   map[string]string{"auth": "Authentication data not found"},
		})
		return
	}

	// Get TOTP auth method
	authMethod, err := c.repo.GetAuthMethodRepository().GetAuthMethodByType(userAuth.ID, user.AuthMethodTypeTOTP)
	if err != nil {
		ctx.JSON(http.StatusNotFound, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "No TOTP setup found",
			Error:   map[string]string{"totp": "TOTP is not configured for this account"},
		})
		return
	}

	if !authMethod.IsEnabled {
		ctx.JSON(http.StatusBadRequest, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "TOTP not enabled",
			Error:   map[string]string{"totp": "TOTP must be enabled to view recovery codes"},
		})
		return
	}

	ctx.JSON(http.StatusOK, common.APIResponse{
		Success: true,
		Data: gin.H{
			"recovery_codes": authMethod.RecoveryCodes,
			"codes_count":    len(authMethod.RecoveryCodes),
		},
		Message: "Recovery codes retrieved successfully",
		Error:   nil,
	})
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

// GetAllUsers returns all users (admin only)
func (c *Controller) GetAllUsers(ctx *gin.Context) {
	// Parse pagination parameters
	page := 1
	limit := 20

	if pageStr := ctx.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if limitStr := ctx.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	offset := (page - 1) * limit

	// Get users with pagination
	users, totalCount, err := c.repo.GetUserRepository().GetAllUsersWithPagination(limit, offset)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to retrieve users",
			Error:   map[string]string{"error": "Failed to retrieve users, please try again later"},
		})
		return
	}

	// Convert to admin response format (remove passwords)
	adminUsers := make([]user.AdminUserResponse, len(users))
	for i, u := range users {
		adminUsers[i] = user.AdminUserResponse{
			ID:           u.ID,
			Username:     u.Username,
			FirstName:    u.FirstName,
			LastName:     u.LastName,
			Email:        u.Email,
			CreatedAt:    u.CreatedAt,
			UpdatedAt:    u.UpdatedAt,
			IsPremium:    u.IsPremium,
			IsLocked:     u.IsLocked,
			LockedAt:     u.LockedAt,
			LockedReason: u.LockedReason,
			Role:         u.Role,
		}
	}

	totalPages := int((totalCount + int64(limit) - 1) / int64(limit))

	response := user.PaginatedUsersResponse{
		Users:      adminUsers,
		TotalCount: totalCount,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}

	ctx.JSON(http.StatusOK, common.APIResponse{
		Success: true,
		Data:    response,
		Message: "Users retrieved successfully",
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

	var req user.AdminLockUserRequest
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
	if err := c.repo.GetUserRepository().LockUser(userID, req.Reason); err != nil {
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

	var req user.AdminUnlockUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		// Allow empty body for unlock requests
		req = user.AdminUnlockUserRequest{}
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
	if err := c.repo.GetUserRepository().UnlockUser(userID, req.Reason); err != nil {
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

// GetLoginAttempts returns login attempt history (admin only)
func (c *Controller) GetLoginAttempts(ctx *gin.Context) {
	// Parse pagination parameters
	page := 1
	limit := 50

	if pageStr := ctx.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if limitStr := ctx.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 500 {
			limit = l
		}
	}

	offset := (page - 1) * limit

	// Get login attempts
	attempts, totalCount, err := c.repo.GetLoginAttemptRepository().GetAllLoginAttempts(limit, offset)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to retrieve login attempts",
			Error:   map[string]string{"error": "Failed to retrieve login attempts, please try again later"},
		})
		return
	}

	totalPages := int((totalCount + int64(limit) - 1) / int64(limit))

	response := map[string]interface{}{
		"attempts":    attempts,
		"total_count": totalCount,
		"page":        page,
		"limit":       limit,
		"total_pages": totalPages,
	}

	ctx.JSON(http.StatusOK, common.APIResponse{
		Success: true,
		Data:    response,
		Message: "Login attempts retrieved successfully",
		Error:   nil,
	})
}

// ResetPassword resets password with token
func (c *Controller) ResetPassword(ctx *gin.Context) {
	var req struct {
		Token    string `json:"token" validate:"required"`
		Password string `json:"password" validate:"required,min=8,max=50,pwdcomplex"`
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

	//if err := c.Validate.Struct(req); err != nil {
	//	ctx.JSON(http.StatusBadRequest, common.APIResponse{
	//		Success: false,
	//		Data:    nil,
	//		Message: "Validation failed",
	//		Error:   map[string]string{"password": "Password must be 8-50 characters with letters, numbers, and symbols"},
	//	})
	//	return
	//}

	// Validate reset token
	_, err := c.repo.GetUserAuthRepository().ValidatePasswordResetToken(req.Token)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Password reset failed",
			Error:   map[string]string{"token": "Invalid or expired reset token"},
		})
		return
	}

	// Hash new password
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Password reset failed",
			Error:   map[string]string{"error": "Failed to process new password"},
		})
		return
	}

	// Reset password
	if err := c.repo.GetUserAuthRepository().ResetPassword(req.Token, hashedPassword); err != nil {
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Password reset failed",
			Error:   map[string]string{"error": "Failed to update password"},
		})
		return
	}

	ctx.JSON(http.StatusOK, common.APIResponse{
		Success: true,
		Data:    nil,
		Message: "Password reset successfully",
		Error:   nil,
	})
}

// RefreshToken generates new access token from refresh token (Redis-based)
func (c *Controller) RefreshToken(ctx *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" validate:"required"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid input",
			Error:   map[string]string{"input": "Refresh token is required"},
		})
		return
	}

	// Get Redis client
	sessionManager := middleware.GetSessionManager()
	redisClient := sessionManager.GetRedisClient()

	// Validate refresh token from Redis
	refreshTokenManager := utils.NewRefreshTokenManager(redisClient)
	tokenData, err := refreshTokenManager.GetRefreshToken(context.Background(), req.RefreshToken)
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
	err = refreshTokenManager.DeleteRefreshToken(context.Background(), req.RefreshToken)
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

	// Return new tokens
	ctx.JSON(http.StatusOK, common.APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"token":         newToken,
			"refresh_token": newRefreshToken,
		},
		Message: "Token refreshed successfully",
		Error:   nil,
	})
}

// ChangePassword changes user password (requires current password)
func (c *Controller) ChangePassword(ctx *gin.Context) {
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
	userAuth, err := c.repo.GetUserAuthRepository().GetUserAuthByUserID(userID.(string))
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
	if err := c.repo.GetUserAuthRepository().UpdatePassword(userID.(string), hashedPassword); err != nil {
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

package auth

import (
	"net/http"
	"strconv"
	"time"

	"github.com/adehusnim37/lihatin-go/controllers"
	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/adehusnim37/lihatin-go/models/user"
	"github.com/adehusnim37/lihatin-go/repositories"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/gin-gonic/gin"
)

// Controller provides all handlers for authentication operations
type Controller struct {
	*controllers.BaseController
	repo         *repositories.AuthRepository
	emailService *utils.EmailService
}

// NewAuthController creates a new auth controller instance
func NewAuthController(base *controllers.BaseController) *Controller {
	if base.GormDB == nil {
		panic("GormDB is required for AuthController")
	}

	authRepo := repositories.NewAuthRepository(base.GormDB)
	emailService := utils.NewEmailService()

	return &Controller{
		BaseController: base,
		repo:           authRepo,
		emailService:   emailService,
	}
}

// Register creates a new user account
func (c *Controller) Register(ctx *gin.Context) {
	var req dto.RegisterRequest

	// Bind and validate request
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid input",
			Error:   map[string]string{"input": "Invalid input format or missing fields"},
		})
		return
	}

	// Validate the request
	if err := c.Validate.Struct(req); err != nil {
		ctx.JSON(http.StatusBadRequest, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Validation failed",
			Error:   map[string]string{"validation": err.Error()},
		})
		return
	}

	// Check if user already exists
	existingUser, _ := c.repo.GetUserRepository().GetUserByEmailOrUsername(req.Email)
	if existingUser != nil {
		ctx.JSON(http.StatusConflict, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Registration failed",
			Error:   map[string]string{"email": "User with this email already exists"},
		})
		return
	}

	// Hash password
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Registration failed",
			Error:   map[string]string{"server": "Failed to process password"},
		})
		return
	}

	// Create user
	newUser := &user.User{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
		Username:  req.Username,
		Password:  hashedPassword,
		IsPremium: false,
	}

	if err := c.repo.GetUserRepository().CreateUser(newUser); err != nil {
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Registration failed",
			Error:   map[string]string{"server": "Failed to create user account"},
		})
		return
	}

	token, err := utils.GenerateVerificationToken()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Registration failed",
			Error:   map[string]string{"server": "Failed to generate verification token"},
		})
		return
	}

	jwtExpiredHours, err := strconv.Atoi(utils.GetRequiredEnv("JWT_EXPIRED"))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid JWT_EXPIRED environment variable",
			Error:   map[string]string{"server": "Failed to parse JWT_EXPIRED"},
		})
		return
	}
	expirationTime := time.Now().Add(time.Duration(jwtExpiredHours) * time.Hour)

	// Create user auth record for email verification
	userAuth := &user.UserAuth{
		UserID:                          newUser.ID,
		PasswordHash:                    hashedPassword, // Use the same hashed password
		IsEmailVerified:                 false,
		EmailVerificationToken:          token,
		EmailVerificationTokenExpiresAt: &expirationTime,
		IsActive:                        true,
	}

	if err := c.repo.GetUserAuthRepository().CreateUserAuth(userAuth); err != nil {
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Registration failed",
			Error:   map[string]string{"server": "Failed to create user authentication record"},
		})
		return
	}

	// Send verification email (optional, continue even if fails)
	_ = c.emailService.SendVerificationEmail(newUser.Email, newUser.FirstName, token)

	ctx.JSON(http.StatusCreated, common.APIResponse{
		Success: true,
		Data: gin.H{
			"user_id": newUser.ID,
			"email":   newUser.Email,
		},
		Message: "Registration successful. Please check your email for verification.",
		Error:   nil,
	})
}

// ResendVerification is implemented in email-auth.go

// GetProfile returns user profile information
func (c *Controller) GetProfile(ctx *gin.Context) {
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

	// Get user auth information
	userAuth, err := c.repo.GetUserAuthRepository().GetUserAuthByUserID(user.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to get user authentication data",
			Error:   map[string]string{"error": "Internal server error"},
		})
		return
	}

	// Create profile response
	profile := map[string]interface{}{
		"id":                user.ID,
		"username":          user.Username,
		"first_name":        user.FirstName,
		"last_name":         user.LastName,
		"email":             user.Email,
		"avatar":            user.Avatar,
		"is_premium":        user.IsPremium,
		"created_at":        user.CreatedAt,
		"is_email_verified": userAuth.IsEmailVerified,
		"has_totp_enabled":  userAuth.IsTOTPEnabled,
		"last_login_at":     userAuth.LastLoginAt,
	}

	ctx.JSON(http.StatusOK, common.APIResponse{
		Success: true,
		Data:    profile,
		Message: "Profile retrieved successfully",
		Error:   nil,
	})
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
		ctx.JSON(http.StatusBadRequest, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid input",
			Error:   map[string]string{"input": "Invalid input format or missing fields"},
		})
		return
	}

	// Validate request
	if err := c.Validate.Struct(req); err != nil {
		ctx.JSON(http.StatusBadRequest, common.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Validation failed",
			Error:   map[string]string{"reason": "Lock reason is required and must be between 10-500 characters"},
		})
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
		if err := c.Validate.Struct(req); err != nil {
			ctx.JSON(http.StatusBadRequest, common.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Validation failed",
				Error:   map[string]string{"reason": "Unlock reason must not exceed 500 characters"},
			})
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

package auth

import (
	"net/http"
	"strconv"
	"time"

	"github.com/adehusnim37/lihatin-go/controllers"
	"github.com/adehusnim37/lihatin-go/models"
	"github.com/adehusnim37/lihatin-go/repositories"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
	var req models.RegisterRequest

	// Bind and validate request
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid input",
			Error:   map[string]string{"input": "Invalid input format or missing fields"},
		})
		return
	}

	// Validate the request
	if err := c.Validate.Struct(req); err != nil {
		ctx.JSON(http.StatusBadRequest, models.APIResponse{
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
		ctx.JSON(http.StatusConflict, models.APIResponse{
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
		ctx.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Registration failed",
			Error:   map[string]string{"server": "Failed to process password"},
		})
		return
	}

	// Create user
	user := &models.User{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
		Username:  req.Username,
		Password:  hashedPassword,
		IsPremium: false,
	}

	if err := c.repo.GetUserRepository().CreateUser(user); err != nil {
		ctx.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Registration failed",
			Error:   map[string]string{"server": "Failed to create user account"},
		})
		return
	}

	// Create user auth record for email verification
	userAuth := &models.UserAuth{
		ID:                              uuid.New().String(), // Generate proper ID
		UserID:                          user.ID,
		PasswordHash:                    hashedPassword, // Use the same hashed password
		IsEmailVerified:                 false,
		EmailVerificationToken:          "verification-token", // TODO: Generate proper token
		IsActive:                        true,
	}
	
	if err := c.repo.GetUserAuthRepository().CreateUserAuth(userAuth); err != nil {
		// Log error but don't fail registration
		// User can verify email later
	}

	// Send verification email (optional, continue even if fails)
	_ = c.emailService.SendVerificationEmail(user.Email, user.FirstName, "verification-token")

	ctx.JSON(http.StatusCreated, models.APIResponse{
		Success: true,
		Data: gin.H{
			"user_id": user.ID,
			"email":   user.Email,
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
		ctx.JSON(http.StatusUnauthorized, models.APIResponse{
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
		ctx.JSON(http.StatusNotFound, models.APIResponse{
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
		ctx.JSON(http.StatusInternalServerError, models.APIResponse{
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

	ctx.JSON(http.StatusOK, models.APIResponse{
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
		ctx.JSON(http.StatusUnauthorized, models.APIResponse{
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
		ctx.JSON(http.StatusNotFound, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "User authentication data not found",
			Error:   map[string]string{"auth": "Authentication data not found"},
		})
		return
	}

	// Get TOTP auth method
	authMethod, err := c.repo.GetAuthMethodRepository().GetAuthMethodByType(userAuth.ID, models.AuthMethodTypeTOTP)
	if err != nil {
		ctx.JSON(http.StatusNotFound, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "No TOTP setup found",
			Error:   map[string]string{"totp": "TOTP is not configured for this account"},
		})
		return
	}

	if !authMethod.IsEnabled {
		ctx.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "TOTP not enabled",
			Error:   map[string]string{"totp": "TOTP must be enabled to view recovery codes"},
		})
		return
	}

	ctx.JSON(http.StatusOK, models.APIResponse{
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
		ctx.JSON(http.StatusUnauthorized, models.APIResponse{
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
		ctx.JSON(http.StatusNotFound, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "User not found",
			Error:   map[string]string{"user": "User not found"},
		})
		return
	}

	// Create premium status response
	premiumStatus := map[string]interface{}{
		"user_id":          user.ID,
		"is_premium":       user.IsPremium,
		"premium_since":    nil, // TODO: Add premium_since field to user model
		"subscription_type": map[bool]string{true: "premium", false: "free"}[user.IsPremium],
		"features": map[string]interface{}{
			"unlimited_uploads": user.IsPremium,
			"priority_support": user.IsPremium,
			"advanced_analytics": user.IsPremium,
			"custom_branding": user.IsPremium,
		},
	}

	ctx.JSON(http.StatusOK, models.APIResponse{
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
		ctx.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to retrieve users",
			Error:   map[string]string{"error": "Failed to retrieve users, please try again later"},
		})
		return
	}
	
	// Convert to admin response format (remove passwords)
	adminUsers := make([]models.AdminUserResponse, len(users))
	for i, user := range users {
		adminUsers[i] = models.AdminUserResponse{
			ID:           user.ID,
			Username:     user.Username,
			FirstName:    user.FirstName,
			LastName:     user.LastName,
			Email:        user.Email,
			CreatedAt:    user.CreatedAt,
			UpdatedAt:    user.UpdatedAt,
			IsPremium:    user.IsPremium,
			IsLocked:     user.IsLocked,
			LockedAt:     user.LockedAt,
			LockedReason: user.LockedReason,
			Role:         user.Role,
		}
	}
	
	totalPages := int((totalCount + int64(limit) - 1) / int64(limit))
	
	response := models.PaginatedUsersResponse{
		Users:      adminUsers,
		TotalCount: totalCount,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}
	
	ctx.JSON(http.StatusOK, models.APIResponse{
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
		ctx.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "User ID is required",
			Error:   map[string]string{"user_id": "User ID parameter is required"},
		})
		return
	}
	
	var req models.AdminLockUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid input",
			Error:   map[string]string{"input": "Invalid input format or missing fields"},
		})
		return
	}
	
	// Validate request
	if err := c.Validate.Struct(req); err != nil {
		ctx.JSON(http.StatusBadRequest, models.APIResponse{
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
		ctx.JSON(http.StatusNotFound, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "User not found",
			Error:   map[string]string{"user_id": "User with this ID does not exist"},
		})
		return
	}
	
	// Check if user is already locked
	if user.IsLocked {
		ctx.JSON(http.StatusConflict, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "User already locked",
			Error:   map[string]string{"status": "User account is already locked"},
		})
		return
	}
	
	// Lock the user
	if err := c.repo.GetUserRepository().LockUser(userID, req.Reason); err != nil {
		ctx.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to lock user",
			Error:   map[string]string{"error": "Failed to lock user account, please try again later"},
		})
		return
	}
	
	ctx.JSON(http.StatusOK, models.APIResponse{
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
		ctx.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "User ID is required",
			Error:   map[string]string{"user_id": "User ID parameter is required"},
		})
		return
	}
	
	var req models.AdminUnlockUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		// Allow empty body for unlock requests
		req = models.AdminUnlockUserRequest{}
	}
	
	// Validate request if reason is provided
	if req.Reason != "" {
		if err := c.Validate.Struct(req); err != nil {
			ctx.JSON(http.StatusBadRequest, models.APIResponse{
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
		ctx.JSON(http.StatusNotFound, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "User not found",
			Error:   map[string]string{"user_id": "User with this ID does not exist"},
		})
		return
	}
	
	// Check if user is actually locked
	if !user.IsLocked {
		ctx.JSON(http.StatusConflict, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "User not locked",
			Error:   map[string]string{"status": "User account is not currently locked"},
		})
		return
	}
	
	// Unlock the user
	if err := c.repo.GetUserRepository().UnlockUser(userID, req.Reason); err != nil {
		ctx.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to unlock user",
			Error:   map[string]string{"error": "Failed to unlock user account, please try again later"},
		})
		return
	}
	
	ctx.JSON(http.StatusOK, models.APIResponse{
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
		ctx.JSON(http.StatusInternalServerError, models.APIResponse{
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
	
	ctx.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    response,
		Message: "Login attempts retrieved successfully",
		Error:   nil,
	})
}

// GetAPIKeys returns user's API keys
func (c *Controller) GetAPIKeys(ctx *gin.Context) {
	// Get user ID from JWT token context
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Authentication required",
			Error:   map[string]string{"auth": "Please authenticate to access this resource"},
		})
		return
	}
	
	// Get user's API keys
	apiKeys, err := c.repo.GetAPIKeyRepository().GetAPIKeysByUserID(userID.(string))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to retrieve API keys",
			Error:   map[string]string{"error": "Failed to retrieve API keys, please try again later"},
		})
		return
	}
	
	// Convert to response format (hide sensitive data)
	responses := make([]models.APIKeyResponse, len(apiKeys))
	for i, key := range apiKeys {
		responses[i] = models.APIKeyResponse{
			ID:          key.ID,
			Name:        key.Name,
			KeyPreview:  key.KeyPreview(), // Use method to get preview
			LastUsedAt:  key.LastUsedAt,
			ExpiresAt:   key.ExpiresAt,
			IsActive:    key.IsActive,
			Permissions: key.Permissions,
			CreatedAt:   key.CreatedAt,
		}
	}
	
	ctx.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    responses,
		Message: "API keys retrieved successfully",
		Error:   nil,
	})
}

// CreateAPIKey creates a new API key
func (c *Controller) CreateAPIKey(ctx *gin.Context) {
	// Get user ID from JWT token context
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Authentication required",
			Error:   map[string]string{"auth": "Please authenticate to access this resource"},
		})
		return
	}
	
	var req models.APIKeyRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid input",
			Error:   map[string]string{"input": "Invalid input format or missing fields"},
		})
		return
	}
	
	// Validate request
	if err := c.Validate.Struct(req); err != nil {
		ctx.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Validation failed",
			Error:   map[string]string{"name": "API key name is required and must be between 3-100 characters"},
		})
		return
	}
	
	// Create API key
	apiKey, keyString, err := c.repo.GetAPIKeyRepository().CreateAPIKey(
		userID.(string), 
		req.Name, 
		req.ExpiresAt, 
		req.Permissions,
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to create API key",
			Error:   map[string]string{"error": "Failed to create API key, please try again later"},
		})
		return
	}
	
	// Return the key with the full key string (this is the only time it's shown)
	response := map[string]interface{}{
		"id":          apiKey.ID,
		"name":        apiKey.Name,
		"key":         keyString, // Full key shown only once
		"expires_at":  apiKey.ExpiresAt,
		"permissions": apiKey.Permissions,
		"created_at":  apiKey.CreatedAt,
		"warning":     "This is the only time the full API key will be shown. Please save it securely.",
	}
	
	ctx.JSON(http.StatusCreated, models.APIResponse{
		Success: true,
		Data:    response,
		Message: "API key created successfully",
		Error:   nil,
	})
}

// RevokeAPIKey revokes an API key
func (c *Controller) RevokeAPIKey(ctx *gin.Context) {
	// Get user ID from JWT token context
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Authentication required",
			Error:   map[string]string{"auth": "Please authenticate to access this resource"},
		})
		return
	}
	
	keyID := ctx.Param("key_id")
	if keyID == "" {
		ctx.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "API key ID is required",
			Error:   map[string]string{"key_id": "API key ID parameter is required"},
		})
		return
	}
	
	// Check if API key exists and belongs to the user
	apiKey, err := c.repo.GetAPIKeyRepository().GetAPIKeyByID(keyID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "API key not found",
			Error:   map[string]string{"key_id": "API key with this ID does not exist"},
		})
		return
	}
	
	// Check ownership
	if apiKey.UserID != userID.(string) {
		ctx.JSON(http.StatusForbidden, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Access denied",
			Error:   map[string]string{"auth": "You can only revoke your own API keys"},
		})
		return
	}
	
	// Revoke the API key
	if err := c.repo.GetAPIKeyRepository().RevokeAPIKey(keyID); err != nil {
		ctx.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to revoke API key",
			Error:   map[string]string{"error": "Failed to revoke API key, please try again later"},
		})
		return
	}
	
	ctx.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    nil,
		Message: "API key revoked successfully",
		Error:   nil,
	})
}

// UpdateAPIKey updates an API key
func (c *Controller) UpdateAPIKey(ctx *gin.Context) {
	// Get user ID from context
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Unauthorized",
			Error:   map[string]string{"auth": "User not authenticated"},
		})
		return
	}

	// Get API key ID from URL
	keyID := ctx.Param("id")
	if keyID == "" {
		ctx.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid request",
			Error:   map[string]string{"id": "API key ID is required"},
		})
		return
	}

	// Bind and validate request
	var req models.UpdateAPIKeyRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Invalid input",
			Error:   map[string]string{"input": "Invalid input format or missing fields"},
		})
		return
	}

	// Check if API key exists and belongs to user
	apiKey, err := c.repo.APIKeyRepo.GetAPIKeyByID(keyID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "API key not found",
			Error:   map[string]string{"api_key": "API key not found or access denied"},
		})
		return
	}

	// Verify ownership
	if apiKey.UserID != userID.(string) {
		ctx.JSON(http.StatusForbidden, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Access denied",
			Error:   map[string]string{"api_key": "You can only update your own API keys"},
		})
		return
	}

	// Prepare updates map
	updates := make(map[string]interface{})
	
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	
	if req.ExpiresAt != nil {
		// Validate expiration date is in the future
		if req.ExpiresAt.Before(time.Now()) {
			ctx.JSON(http.StatusBadRequest, models.APIResponse{
				Success: false,
				Data:    nil,
				Message: "Invalid expiration date",
				Error:   map[string]string{"expires_at": "Expiration date must be in the future"},
			})
			return
		}
		updates["expires_at"] = *req.ExpiresAt
	}
	
	if req.Permissions != nil {
		// Validate permissions (you can add custom validation logic here)
		validPermissions := []string{"read", "write", "admin"}
		for _, perm := range req.Permissions {
			valid := false
			for _, validPerm := range validPermissions {
				if perm == validPerm {
					valid = true
					break
				}
			}
			if !valid {
				ctx.JSON(http.StatusBadRequest, models.APIResponse{
					Success: false,
					Data:    nil,
					Message: "Invalid permission",
					Error:   map[string]string{"permissions": "Invalid permission: " + perm},
				})
				return
			}
		}
		updates["permissions"] = req.Permissions
	}
	
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	// Always update the updated_at timestamp
	updates["updated_at"] = time.Now()

	// Update the API key
	if err := c.repo.APIKeyRepo.UpdateAPIKey(keyID, updates); err != nil {
		ctx.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "Failed to update API key",
			Error:   map[string]string{"database": "Could not update API key"},
		})
		return
	}

	// Fetch the updated API key to return
	updatedKey, err := c.repo.APIKeyRepo.GetAPIKeyByID(keyID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Data:    nil,
			Message: "API key updated but failed to fetch updated data",
			Error:   map[string]string{"database": "Could not fetch updated API key"},
		})
		return
	}

	// Convert to response model
	response := models.APIKeyResponse{
		ID:          updatedKey.ID,
		Name:        updatedKey.Name,
		KeyPreview:  updatedKey.KeyPreview(),
		LastUsedAt:  updatedKey.LastUsedAt,
		ExpiresAt:   updatedKey.ExpiresAt,
		IsActive:    updatedKey.IsActive,
		Permissions: updatedKey.Permissions,
		CreatedAt:   updatedKey.CreatedAt,
	}

	ctx.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    response,
		Message: "API key updated successfully",
		Error:   nil,
	})
}

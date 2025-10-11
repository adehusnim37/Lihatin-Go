package auth

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/middleware"
	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/adehusnim37/lihatin-go/models/user"
	"github.com/adehusnim37/lihatin-go/utils"
	clientip "github.com/adehusnim37/lihatin-go/utils/clientip"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Register creates a new user account
func (c *Controller) Register(ctx *gin.Context) {
	var req dto.RegisterRequest

	// Bind and validate request
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.SendValidationError(ctx, err, &req)
		return
	}

	// Check if user already exists
	existingUser, _ := c.repo.GetUserRepository().GetUserByEmailOrUsername(req.Email)
	if existingUser != nil {
		utils.SendErrorResponse(
			ctx,
			http.StatusBadRequest,
			"Registration failed",
			"Email is already registered",
			"EMAIL_ALREADY_REGISTERED",
			map[string]string{"email": "Email is already registered"},
		)
		return
	}

	// Hash password
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		utils.SendErrorResponse(
			ctx,
			http.StatusInternalServerError,
			"Registration failed",
			"Failed to process password",
			"PASSWORD_HASHING_FAILED",
			map[string]string{"server": "Failed to process password"},
		)
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
		utils.SendErrorResponse(
			ctx,
			http.StatusInternalServerError,
			"Registration failed",
			"Failed to create user account",
			"USER_CREATION_FAILED",
			map[string]string{"server": "Failed to create user account"},
		)
		return
	}

	token, err := utils.GenerateVerificationToken()
	if err != nil {
		utils.SendErrorResponse(
			ctx,
			http.StatusInternalServerError,
			"Registration failed",
			"Failed to generate verification token",
			"VERIFICATION_TOKEN_GENERATION_FAILED",
			map[string]string{"server": "Failed to generate verification token"},
		)
		return
	}

	jwtExpiredHours, err := strconv.Atoi(utils.GetRequiredEnv("JWT_EXPIRED"))
	if err != nil {
		utils.SendErrorResponse(
			ctx,
			http.StatusInternalServerError,
			"Registration failed",
			"Failed to parse JWT_EXPIRED",
			"JWT_EXPIRED_PARSING_FAILED",
			map[string]string{"server": "Failed to parse JWT_EXPIRED"},
		)
		return
	}
	expirationTime := time.Now().Add(time.Duration(jwtExpiredHours) * time.Hour)

	// Get device and IP info
	deviceID, lastIP := clientip.GetDeviceAndIPInfo(ctx)

	// Create session in Redis
	sessionID, err := middleware.CreateSession(
		context.Background(),
		newUser.ID,
		"registration",
		ctx.ClientIP(),
		ctx.GetHeader("User-Agent"),
		*deviceID,
	)

	if err != nil {
		utils.Logger.Error("Failed to create registration session in Redis",
			"user_id", newUser.ID,
			"error", err.Error(),
		)
		utils.SendErrorResponse(
			ctx,
			http.StatusInternalServerError,
			"Registration failed",
			"Failed to create session",
			"SESSION_CREATION_FAILED",
			map[string]string{"server": "Session creation failed"},
		)
		return
	}

	utils.Logger.Info("Registration session created",
		"user_id", newUser.ID,
		"session_preview", utils.GetKeyPreview(sessionID),
	)

	uuidV7, err := uuid.NewV7()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, common.APIResponse{
			Success: false,
			Message: "Failed to create user authentication record",
			Error:   map[string]string{"server": "Failed to generate UUID"},
		})
		return
	}
	userAuth := &user.UserAuth{
		ID:                              uuidV7.String(),
		DeviceID:                        deviceID,
		LastIP:                          lastIP,
		UserID:                          newUser.ID,
		PasswordHash:                    hashedPassword, // Use the same hashed password
		IsEmailVerified:                 false,
		EmailVerificationToken:          token,
		EmailVerificationTokenExpiresAt: &expirationTime,
		EmailVerificationSource:         user.EmailSourceSignup,
		IsActive:                        true,
	}

	if err := c.repo.GetUserAuthRepository().CreateUserAuth(userAuth); err != nil {
		utils.HandleError(ctx, err, newUser.ID)
		return
	}

	// Send verification email (optional, continue even if fails)
	_ = c.emailService.SendVerificationEmail(newUser.Email, newUser.FirstName, token)

	utils.SendCreatedResponse(ctx, map[string]any{
		"user_id": newUser.ID,
		"email":   newUser.Email,
	}, "Registration successful. Please check your email for verification.")
}

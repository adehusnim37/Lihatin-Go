package auth

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/internal/pkg/auth"
	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/ip"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/internal/pkg/premium"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	"github.com/adehusnim37/lihatin-go/middleware"
	"github.com/adehusnim37/lihatin-go/models/user"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Register creates a new user account
func (c *Controller) Register(ctx *gin.Context) {
	var req dto.RegisterRequest
	var premiumReservationKey string

	// Bind and validate request
	if err := ctx.ShouldBindJSON(&req); err != nil {
		validator.SendValidationError(ctx, err, &req)
		return
	}

	// Check if user already exists
	existingUser, _ := c.repo.GetUserRepository().GetUserByEmailOrUsername(req.Email)
	if existingUser != nil {
		httputil.SendErrorResponse(
			ctx,
			http.StatusBadRequest,
			"Registration failed",
			"Email is already registered",
			"EMAIL_ALREADY_REGISTERED",
			map[string]string{"email": "Email is already registered"},
		)
		return
	}

	if req.SecretCode != "" {
		manager := middleware.GetSessionManager()
		if manager == nil || manager.GetRedisClient() == nil {
			httputil.SendErrorResponse(
				ctx,
				http.StatusInternalServerError,
				"PREMIUM_CODE_REDEEM_FAILED",
				"Premium code service is unavailable",
				"secret_code",
			)
			return
		}

		key, _, err := premium.RedeemOneTimeCode(
			context.Background(),
			manager.GetRedisClient(),
			req.SecretCode,
			"pending:"+req.Email,
			time.Now(),
		)
		if err != nil {
			switch {
			case errors.Is(err, premium.ErrCodeFormat), errors.Is(err, premium.ErrCodeSignature):
				httputil.SendErrorResponse(ctx, http.StatusBadRequest, "PREMIUM_CODE_INVALID", "Secret code is invalid", "secret_code")
				return
			case errors.Is(err, premium.ErrCodeExpired):
				httputil.SendErrorResponse(ctx, http.StatusBadRequest, "PREMIUM_CODE_EXPIRED", "Secret code is expired", "secret_code")
				return
			case errors.Is(err, premium.ErrCodeUsed):
				httputil.SendErrorResponse(ctx, http.StatusConflict, "PREMIUM_CODE_ALREADY_USED", "Secret code has already been used", "secret_code")
				return
			case errors.Is(err, premium.ErrSecretKeyMissing):
				httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "PREMIUM_CODE_CONFIG_INVALID", "Premium code is not configured", "secret_code")
				return
			default:
				httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "PREMIUM_CODE_REDEEM_FAILED", "Failed to redeem premium code", "secret_code")
				return
			}
		}

		premiumReservationKey = key
	}

	// Create user
	newUser := &user.User{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
		Username:  req.Username,
	}
	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "PASSWORD_HASHING_FAILED", "Failed to process password", "password")
		return
	}
	authID, err := uuid.NewV7()
	if err != nil {
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "AUTH_RECORD_CREATION_FAILED", "Failed to create user authentication record", "auth")
		return
	}
	userAuth := &user.UserAuth{
		ID:              authID.String(),
		PasswordHash:    hashedPassword,
		AccountStatus:   user.AccountStatusActive,
		IsEmailVerified: false,
	}

	notificationPreferences := &user.NotificationPreference{
		PromotionalEmail:   req.OptInPromotionalEmails,
		WeeklySummaryEmail: req.OptInWeeklySummaryEmails,
		ConsentSource:      req.ConsentSource,
	}

	if err := c.repo.GetUserRepository().CreateUser(newUser, userAuth, notificationPreferences, nil); err != nil {
		if premiumReservationKey != "" {
			manager := middleware.GetSessionManager()
			if manager != nil {
				_ = premium.ReleaseReservation(context.Background(), manager.GetRedisClient(), premiumReservationKey)
			}
		}
		httputil.SendErrorResponse(
			ctx,
			http.StatusInternalServerError,
			"Registration failed",
			"Failed to create user account",
			"USER_CREATION_FAILED",
			map[string]string{"server": "Failed to create user account"},
		)
		return
	}

	if premiumReservationKey != "" {
		manager := middleware.GetSessionManager()
		if manager != nil {
			_ = premium.MarkRedeemedOwner(context.Background(), manager.GetRedisClient(), premiumReservationKey, newUser.ID)
		}

		if err := c.redeemPremiumCodeInDB(req.SecretCode, newUser.ID); err != nil {
			logger.Logger.Error("Failed to persist premium code redemption during register",
				"user_id", newUser.ID,
				"error", err.Error(),
			)
			httputil.HandleError(ctx, err, nil)
			return
		}
	}

	token, err := auth.GenerateVerificationToken()
	if err != nil {
		httputil.SendErrorResponse(
			ctx,
			http.StatusInternalServerError,
			"Registration failed",
			"Failed to generate verification token",
			"VERIFICATION_TOKEN_GENERATION_FAILED",
			map[string]string{"server": "Failed to generate verification token"},
		)
		return
	}

	// Use EMAIL_VERIFICATION_EXPIRY instead of JWT_EXPIRED for email tokens
	emailVerificationHours := config.GetEnvAsInt(config.EnvEmailVerificationExpiry, 24)
	expirationTime := time.Now().Add(time.Duration(emailVerificationHours) * time.Hour)

	// Get device and IP info
	deviceID, lastIP := ip.GetDeviceAndIPInfo(ctx)

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
		logger.Logger.Error("Failed to create registration session in Redis",
			"user_id", newUser.ID,
			"error", err.Error(),
		)
		httputil.SendErrorResponse(
			ctx,
			http.StatusInternalServerError,
			"Registration failed",
			"Failed to create session",
			"SESSION_CREATION_FAILED",
			map[string]string{"server": "Session creation failed"},
		)
		return
	}

	logger.Logger.Info("Registration session created",
		"user_id", newUser.ID,
		"session_preview", auth.GetKeyPreview(sessionID),
	)

	userAuth.DeviceID = deviceID
	userAuth.LastIP = lastIP
	userAuth.EmailVerificationToken = token
	userAuth.EmailVerificationTokenExpiresAt = &expirationTime
	userAuth.EmailVerificationSource = user.EmailSourceSignup

	if err := c.repo.GetUserAuthRepository().UpdateUserAuth(userAuth); err != nil {
		httputil.HandleError(ctx, err, newUser.ID)
		return
	}

	// Send verification email (optional, continue even if fails)
	_ = c.emailService.SendVerificationEmail(newUser.Email, newUser.Username, token)

	httputil.SendCreatedResponse(ctx, map[string]any{
		"user_id": newUser.ID,
		"email":   newUser.Email,
	}, "Registration successful. Please check your email for verification.")
}

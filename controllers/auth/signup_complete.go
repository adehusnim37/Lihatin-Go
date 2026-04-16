package auth

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/internal/pkg/auth"
	apperrors "github.com/adehusnim37/lihatin-go/internal/pkg/errors"
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

// SignupComplete creates account after signup OTP verification.
func (c *Controller) SignupComplete(ctx *gin.Context) {
	var req dto.SignupCompleteRequest
	var premiumReservationKey string
	isPremiumFromCode := false

	if err := ctx.ShouldBindJSON(&req); err != nil {
		validator.SendValidationError(ctx, err, &req)
		return
	}

	email, err := auth.GetSignupCompletionTokenEmail(ctx.Request.Context(), req.SignupToken)
	if err != nil {
		httputil.SendErrorResponse(ctx, http.StatusUnauthorized, "SIGNUP_TOKEN_INVALID", "Signup session expired or invalid. Please restart signup.", "signup_token")
		return
	}

	existingEmail, _ := c.repo.GetUserRepository().GetUserByEmailOrUsername(email)
	if existingEmail != nil {
		httputil.SendErrorResponse(ctx, http.StatusConflict, "EMAIL_ALREADY_REGISTERED", "Email is already registered", "email")
		return
	}

	existingUsername, _ := c.repo.GetUserRepository().GetUserByEmailOrUsername(req.Username)
	if existingUsername != nil {
		httputil.SendErrorResponse(ctx, http.StatusConflict, "USERNAME_ALREADY_USED", "Username is already used", "username")
		return
	}

	if req.SecretCode != "" {
		manager := middleware.GetSessionManager()
		if manager == nil || manager.GetRedisClient() == nil {
			httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "PREMIUM_CODE_REDEEM_FAILED", "Premium code service is unavailable", "secret_code")
			return
		}

		key, _, err := premium.RedeemOneTimeCode(
			context.Background(),
			manager.GetRedisClient(),
			req.SecretCode,
			"pending:"+email,
			time.Now(),
		)
		if err != nil {
			switch {
			case errors.Is(err, premium.ErrCodeFormat), errors.Is(err, premium.ErrCodeSignature):
				httputil.SendErrorResponse(ctx, http.StatusBadRequest, "PREMIUM_CODE_INVALID", "Secret code is invalid", "secret_code")
			case errors.Is(err, premium.ErrCodeExpired):
				httputil.SendErrorResponse(ctx, http.StatusBadRequest, "PREMIUM_CODE_EXPIRED", "Secret code is expired", "secret_code")
			case errors.Is(err, premium.ErrCodeUsed):
				httputil.SendErrorResponse(ctx, http.StatusConflict, "PREMIUM_CODE_ALREADY_USED", "Secret code has already been used", "secret_code")
			case errors.Is(err, premium.ErrSecretKeyMissing):
				httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "PREMIUM_CODE_CONFIG_INVALID", "Premium code is not configured", "secret_code")
			default:
				httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "PREMIUM_CODE_REDEEM_FAILED", "Failed to redeem premium code", "secret_code")
			}
			return
		}

		premiumReservationKey = key
		isPremiumFromCode = true
	}

	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "PASSWORD_HASHING_FAILED", "Failed to process password", "password")
		return
	}

	newUser := &user.User{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     email,
		Username:  req.Username,
		Password:  req.Password,
		IsPremium: isPremiumFromCode,
	}

	if err := c.repo.GetUserRepository().CreateUser(newUser); err != nil {
		if premiumReservationKey != "" {
			manager := middleware.GetSessionManager()
			if manager != nil {
				_ = premium.ReleaseReservation(context.Background(), manager.GetRedisClient(), premiumReservationKey)
			}
		}

		if errors.Is(err, apperrors.ErrUserDuplicateEntry) {
			httputil.SendErrorResponse(ctx, http.StatusConflict, "USER_ALREADY_EXISTS", "Email or username is already registered", "user")
			return
		}

		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "USER_CREATION_FAILED", "Failed to create user account", "user")
		return
	}

	if premiumReservationKey != "" {
		manager := middleware.GetSessionManager()
		if manager != nil {
			_ = premium.MarkRedeemedOwner(context.Background(), manager.GetRedisClient(), premiumReservationKey, newUser.ID)
		}
	}

	deviceID, lastIP := ip.GetDeviceAndIPInfo(ctx)
	uuidV7, err := uuid.NewV7()
	if err != nil {
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "AUTH_RECORD_CREATION_FAILED", "Failed to create user authentication record", "auth")
		return
	}

	userAuth := &user.UserAuth{
		ID:              uuidV7.String(),
		DeviceID:        deviceID,
		LastIP:          lastIP,
		UserID:          newUser.ID,
		PasswordHash:    hashedPassword,
		IsEmailVerified: true,
		IsActive:        true,
	}

	if err := c.repo.GetUserAuthRepository().CreateUserAuth(userAuth); err != nil {
		httputil.HandleError(ctx, err, newUser.ID)
		return
	}

	if err := auth.DeleteSignupCompletionToken(ctx.Request.Context(), req.SignupToken); err != nil {
		logger.Logger.Warn("Failed to delete signup completion token",
			"user_id", newUser.ID,
			"error", err.Error(),
		)
	}

	_ = c.emailService.SendSuccessEmailVerification(newUser.Email, newUser.Username)

	httputil.SendCreatedResponse(ctx, map[string]any{
		"user_id": newUser.ID,
		"email":   newUser.Email,
	}, "Signup completed successfully. You can now log in.")
}

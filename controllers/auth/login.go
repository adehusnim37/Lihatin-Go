package auth

import (
	"context"
	"net/http"
	"time"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/internal/pkg/auth"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	"github.com/gin-gonic/gin"
)

// Login authenticates a user with email and password,
// then enforces second factor (TOTP or email OTP).
func (c *Controller) Login(ctx *gin.Context) {
	var loginReq dto.LoginRequest

	if err := ctx.ShouldBindJSON(&loginReq); err != nil {
		validator.SendValidationError(ctx, err, &loginReq)
		return
	}

	user, userAuth, err := c.repo.GetUserAuthRepository().GetUserForLogin(loginReq.EmailOrUsername)
	if err != nil {
		if user != nil {
			c.repo.GetUserAuthRepository().IncrementFailedLogin(user.ID)
		}
		httputil.HandleError(ctx, err, nil)
		return
	}

	isLocked, err := c.repo.GetUserAuthRepository().IsAccountLocked(user.ID)
	if err != nil {
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "LOGIN_FAILED", "An error occurred during login", "auth")
		return
	}

	if isLocked {
		httputil.SendErrorResponse(ctx, http.StatusForbidden, "ACCOUNT_LOCKED", "Your account is locked. Please try again later.", "auth")
		return
	}

	if !userAuth.IsActive {
		httputil.SendErrorResponse(ctx, http.StatusForbidden, "ACCOUNT_DEACTIVATED", "Your account has been deactivated. Please contact support.", "auth")
		return
	}

	if err := auth.CheckPassword(userAuth.PasswordHash, loginReq.Password); err != nil {
		c.repo.GetUserAuthRepository().IncrementFailedLogin(user.ID)
		httputil.SendErrorResponse(ctx, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid email/username or password", "auth")
		return
	}

	hasTOTP, err := c.repo.GetAuthMethodRepository().HasTOTPEnabled(userAuth.ID)
	if err != nil {
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "LOGIN_FAILED", "An error occurred during login", "auth")
		return
	}

	if hasTOTP {
		pendingToken, err := auth.GeneratePendingAuthToken(context.Background(), user.ID)
		if err != nil {
			logger.Logger.Error("Failed to generate pending auth token",
				"user_id", user.ID,
				"error", err.Error(),
			)
			httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "PENDING_TOKEN_GENERATION_FAILED", "Failed to generate authentication token", "auth")
			return
		}

		pendingResponse := dto.PendingTOTPResponse{
			RequiresTOTP:     true,
			PendingAuthToken: pendingToken,
			User:             buildUserProfile(user),
		}

		logger.Logger.Info("TOTP verification required for login",
			"user_id", user.ID,
			"pending_token_preview", auth.GetKeyPreview(pendingToken),
		)

		httputil.SendOKResponse(ctx, pendingResponse, "Password verified. Please complete two-factor authentication.")
		return
	}

	challengeToken, otpCode, challenge, err := auth.GenerateEmailOTPChallenge(
		ctx.Request.Context(),
		auth.EmailOTPPurposeLogin,
		user.Email,
		user.ID,
	)
	if err != nil {
		logger.Logger.Error("Failed to generate login email OTP challenge",
			"user_id", user.ID,
			"error", err.Error(),
		)
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "EMAIL_OTP_CHALLENGE_FAILED", "Failed to start email OTP verification", "auth")
		return
	}

	if err := c.emailService.SendEmailOTP(user.Email, user.Username, "login", otpCode); err != nil {
		_ = auth.DeleteEmailOTPChallenge(ctx.Request.Context(), challengeToken)
		logger.Logger.Error("Failed to send login email OTP",
			"user_id", user.ID,
			"email", user.Email,
			"error", err.Error(),
		)
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "EMAIL_SEND_FAILED", "Failed to send verification code", "email")
		return
	}

	pendingResponse := dto.PendingEmailOTPResponse{
		RequiresEmailOTP: true,
		ChallengeToken:   challengeToken,
		CooldownSeconds:  auth.CooldownSecondsForNextResend(challenge),
		Email:            user.Email,
		User:             buildUserProfile(user),
	}

	httputil.SendOKResponse(ctx, pendingResponse, "Password verified. Please enter the verification code sent to your email.")
}

// Helper function to safely format time pointer
func formatTimeOrEmpty(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02T15:04:05Z07:00")
}

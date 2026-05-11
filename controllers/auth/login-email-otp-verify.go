package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/internal/pkg/auth"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	"github.com/gin-gonic/gin"
)

// VerifyLoginEmailOTP verifies login OTP and issues login cookies.
func (c *Controller) VerifyLoginEmailOTP(ctx *gin.Context) {
	var req dto.VerifyEmailOTPLoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		validator.SendValidationError(ctx, err, &req)
		return
	}

	otpCode := auth.ParseOTPCode(req.OTPCode)
	if otpCode == "" {
		httputil.SendErrorResponse(ctx, http.StatusBadRequest, "INVALID_OTP", "Verification code must be a 6-digit number", "otp_code")
		return
	}

	challenge, err := auth.GetEmailOTPChallenge(ctx.Request.Context(), req.ChallengeToken)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrEmailOTPChallengeNotFound), errors.Is(err, auth.ErrEmailOTPChallengeExpired):
			httputil.SendErrorResponse(ctx, http.StatusUnauthorized, "CHALLENGE_EXPIRED", "Verification session expired. Please login again.", "challenge_token")
			return
		default:
			logger.Logger.Error("Failed to get login email OTP challenge", "error", err.Error())
			httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "OTP_VERIFICATION_FAILED", "Failed to verify code", "otp_code")
			return
		}
	}

	if challenge.Purpose != auth.EmailOTPPurposeLogin || challenge.UserID == "" {
		httputil.SendErrorResponse(ctx, http.StatusBadRequest, "INVALID_CHALLENGE", "Invalid login verification session", "challenge_token")
		return
	}

	user, err := c.repo.GetUserRepository().GetUserByID(challenge.UserID)
	if err != nil {
		httputil.HandleError(ctx, err, challenge.UserID)
		return
	}

	userAuth, err := c.repo.GetUserAuthRepository().GetUserAuthByUserID(challenge.UserID)
	if err != nil {
		httputil.HandleError(ctx, err, challenge.UserID)
		return
	}

	if user.IsLocked {
		httputil.SendErrorResponse(ctx, http.StatusForbidden, "USER_LOCKED", "Your account has been locked. Please contact support.", "auth")
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

	if !userAuth.IsEmailVerified {
		httputil.SendErrorResponse(ctx, http.StatusForbidden, "EMAIL_NOT_VERIFIED", "Your email is not verified. Please verify your email first.", "email")
		return
	}

	_, err = auth.VerifyEmailOTPChallenge(ctx.Request.Context(), req.ChallengeToken, otpCode, auth.EmailOTPPurposeLogin)
	if err != nil {
		var invalidCodeErr *auth.EmailOTPInvalidCodeError
		switch {
		case errors.As(err, &invalidCodeErr):
			httputil.SendErrorResponse(
				ctx,
				http.StatusBadRequest,
				"INVALID_OTP",
				fmt.Sprintf("Invalid verification code. %d attempt(s) remaining.", invalidCodeErr.RemainingAttempts),
				"otp_code",
			)
			return
		case errors.Is(err, auth.ErrEmailOTPAttemptsExceeded):
			httputil.SendErrorResponse(ctx, http.StatusTooManyRequests, "OTP_ATTEMPTS_EXCEEDED", "Too many invalid attempts. Please login again.", "otp_code")
			return
		case errors.Is(err, auth.ErrEmailOTPChallengeNotFound), errors.Is(err, auth.ErrEmailOTPChallengeExpired):
			httputil.SendErrorResponse(ctx, http.StatusUnauthorized, "CHALLENGE_EXPIRED", "Verification session expired. Please login again.", "challenge_token")
			return
		case errors.Is(err, auth.ErrEmailOTPPurposeMismatch):
			httputil.SendErrorResponse(ctx, http.StatusBadRequest, "INVALID_CHALLENGE", "Invalid login verification session", "challenge_token")
			return
		default:
			logger.Logger.Error("Failed to verify login email OTP", "error", err.Error())
			httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "OTP_VERIFICATION_FAILED", "Failed to verify code", "otp_code")
			return
		}
	}

	// If TOTP was enabled after OTP challenge issued, enforce TOTP as final factor.
	hasTOTP, err := c.repo.GetAuthMethodRepository().HasTOTPEnabled(userAuth.ID)
	if err != nil {
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "LOGIN_FAILED", "An error occurred during login", "auth")
		return
	}
	if hasTOTP {
		pendingToken, err := auth.GeneratePendingAuthToken(context.Background(), user.ID)
		if err != nil {
			logger.Logger.Error("Failed to generate pending auth token after email OTP",
				"user_id", user.ID,
				"error", err.Error(),
			)
			httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "PENDING_TOKEN_GENERATION_FAILED", "Failed to continue authentication", "auth")
			return
		}

		pendingResponse := dto.PendingTOTPResponse{
			RequiresTOTP:     true,
			PendingAuthToken: pendingToken,
			User:             buildUserProfile(user),
		}
		httputil.SendOKResponse(ctx, pendingResponse, "Please complete two-factor authentication.")
		return
	}

	if err := c.completeLogin(ctx, user, userAuth, "Login successful"); err != nil {
		return
	}
}

package auth

import (
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

// SignupVerifyOTP validates signup OTP and returns one-time signup token.
func (c *Controller) SignupVerifyOTP(ctx *gin.Context) {
	var req dto.SignupVerifyOTPRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		validator.SendValidationError(ctx, err, &req)
		return
	}

	otpCode := auth.ParseOTPCode(req.OTPCode)
	if otpCode == "" {
		httputil.SendErrorResponse(ctx, http.StatusBadRequest, "INVALID_OTP", "Verification code must be a 6-digit number", "otp_code")
		return
	}

	challenge, err := auth.VerifyEmailOTPChallenge(ctx.Request.Context(), req.ChallengeToken, otpCode, auth.EmailOTPPurposeSignup)
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
			httputil.SendErrorResponse(ctx, http.StatusTooManyRequests, "OTP_ATTEMPTS_EXCEEDED", "Too many invalid attempts. Please restart signup.", "otp_code")
			return
		case errors.Is(err, auth.ErrEmailOTPChallengeNotFound), errors.Is(err, auth.ErrEmailOTPChallengeExpired):
			httputil.SendErrorResponse(ctx, http.StatusUnauthorized, "CHALLENGE_EXPIRED", "Signup verification session expired. Please restart signup.", "challenge_token")
			return
		case errors.Is(err, auth.ErrEmailOTPPurposeMismatch):
			httputil.SendErrorResponse(ctx, http.StatusBadRequest, "INVALID_CHALLENGE", "Invalid signup verification session", "challenge_token")
			return
		default:
			logger.Logger.Error("Failed to verify signup OTP", "error", err.Error())
			httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "OTP_VERIFICATION_FAILED", "Failed to verify code", "otp_code")
			return
		}
	}

	signupToken, err := auth.CreateSignupCompletionToken(ctx.Request.Context(), challenge.Email)
	if err != nil {
		logger.Logger.Error("Failed to create signup completion token",
			"email", challenge.Email,
			"error", err.Error(),
		)
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "SIGNUP_TOKEN_FAILED", "Failed to continue signup", "challenge_token")
		return
	}

	httputil.SendOKResponse(ctx, dto.SignupVerifyOTPResponse{SignupToken: signupToken}, "Verification successful. Continue to complete your profile.")
}

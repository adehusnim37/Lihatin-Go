package auth

import (
	"errors"
	"net/http"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/internal/pkg/auth"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	"github.com/gin-gonic/gin"
)

// ResendLoginEmailOTP resends OTP for login email challenge.
func (c *Controller) ResendLoginEmailOTP(ctx *gin.Context) {
	var req dto.ResendEmailOTPLoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		validator.SendValidationError(ctx, err, &req)
		return
	}

	challenge, otpCode, err := auth.ResendEmailOTPChallenge(ctx.Request.Context(), req.ChallengeToken)
	if err != nil {
		var cooldownErr *auth.EmailOTPCooldownError
		switch {
		case errors.As(err, &cooldownErr):
			httputil.SendOKResponse(ctx, dto.ResendOTPResponse{
				CooldownRemainingSeconds: cooldownErr.RemainingSeconds,
			}, "Please wait before requesting another code")
			return
		case errors.Is(err, auth.ErrEmailOTPChallengeNotFound), errors.Is(err, auth.ErrEmailOTPChallengeExpired):
			httputil.SendErrorResponse(ctx, http.StatusUnauthorized, "CHALLENGE_EXPIRED", "Verification session expired. Please login again.", "challenge_token")
			return
		default:
			logger.Logger.Error("Failed to resend login email OTP",
				"error", err.Error(),
			)
			httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "LOGIN_OTP_RESEND_FAILED", "Failed to resend verification code", "challenge_token")
			return
		}
	}

	if challenge.Purpose != auth.EmailOTPPurposeLogin {
		httputil.SendErrorResponse(ctx, http.StatusBadRequest, "INVALID_CHALLENGE", "Invalid login verification session", "challenge_token")
		return
	}

	userName := ""
	if challenge.UserID != "" {
		if u, err := c.repo.GetUserRepository().GetUserByID(challenge.UserID); err == nil && u != nil {
			userName = u.Username
		}
	}

	if err := c.emailService.SendEmailOTP(challenge.Email, userName, "login", otpCode); err != nil {
		logger.Logger.Error("Failed to send resent login OTP",
			"email", challenge.Email,
			"error", err.Error(),
		)
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "EMAIL_SEND_FAILED", "Failed to send verification code", "email")
		return
	}

	httputil.SendOKResponse(ctx, dto.ResendOTPResponse{
		CooldownSeconds: auth.CooldownSecondsForNextResend(challenge),
	}, "Verification code has been resent")
}

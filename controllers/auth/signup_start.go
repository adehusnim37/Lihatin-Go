package auth

import (
	"errors"
	"net/http"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/internal/pkg/auth"
	"github.com/adehusnim37/lihatin-go/internal/pkg/disposable"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	"github.com/gin-gonic/gin"
)

// SignupStart starts email-first signup and sends OTP code.
func (c *Controller) SignupStart(ctx *gin.Context) {
	var req dto.SignupStartRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		validator.SendValidationError(ctx, err, &req)
		return
	}

	if policy := disposable.Global(); policy != nil {
		blocked, err := policy.ShouldBlockEmail(ctx.Request.Context(), req.Email)
		if err != nil {
			logger.Logger.Warn("Disposable email policy check failed for signup",
				"email", req.Email,
				"error", err.Error(),
			)
		}
		if blocked {
			httputil.SendErrorResponse(
				ctx,
				http.StatusBadRequest,
				"DISPOSABLE_EMAIL_BLOCKED",
				"Disposable email addresses are not allowed. Please use a permanent email address.",
				"email",
			)
			return
		}
	}

	existingUser, _ := c.repo.GetUserRepository().GetUserByEmailOrUsername(req.Email)
	if existingUser != nil {
		httputil.SendErrorResponse(
			ctx,
			http.StatusBadRequest,
			"EMAIL_ALREADY_REGISTERED",
			"Email is already registered",
			"email",
		)
		return
	}

	signupToken, err := auth.GetSignupCompletionTokenByEmail(ctx.Request.Context(), req.Email)
	switch {
	case err == nil && signupToken != "":
		httputil.SendOKResponse(ctx, dto.SignupStartResponse{
			SignupToken:               signupToken,
			RequiresProfileCompletion: true,
		}, "Signup already verified. Continue completing your profile.")
		return
	case err != nil && !errors.Is(err, auth.ErrSignupCompletionTokenInvalid):
		logger.Logger.Warn("Failed to check existing signup completion token",
			"email", req.Email,
			"error", err.Error(),
		)
	}

	challengeToken, otpCode, challenge, err := auth.GenerateEmailOTPChallenge(
		ctx.Request.Context(),
		auth.EmailOTPPurposeSignup,
		req.Email,
		"",
	)
	if err != nil {
		logger.Logger.Error("Failed to create signup OTP challenge",
			"email", req.Email,
			"error", err.Error(),
		)
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "SIGNUP_OTP_FAILED", "Failed to start signup verification", "email")
		return
	}

	if err := c.emailService.SendEmailOTP(req.Email, "", "signup", otpCode); err != nil {
		_ = auth.DeleteEmailOTPChallenge(ctx.Request.Context(), challengeToken)
		logger.Logger.Error("Failed to send signup OTP",
			"email", req.Email,
			"error", err.Error(),
		)
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "EMAIL_SEND_FAILED", "Failed to send verification code", "email")
		return
	}

	httputil.SendOKResponse(ctx, dto.SignupStartResponse{
		ChallengeToken:  challengeToken,
		CooldownSeconds: auth.CooldownSecondsForNextResend(challenge),
	}, "Verification code has been sent to your email")
}

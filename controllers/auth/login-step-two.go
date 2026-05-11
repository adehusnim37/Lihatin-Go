package auth

import (
	"context"
	"net/http"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/internal/pkg/auth"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/models/user"
	"github.com/gin-gonic/gin"
)

func (c *Controller) requireSecondFactor(ctx *gin.Context, usr *user.User, userAuth *user.UserAuth, verifiedMessage string) error {
	hasTOTP, err := c.repo.GetAuthMethodRepository().HasTOTPEnabled(userAuth.ID)
	if err != nil {
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "LOGIN_FAILED", "An error occurred during login", "auth")
		return err
	}

	if hasTOTP {
		pendingToken, err := auth.GeneratePendingAuthToken(context.Background(), usr.ID)
		if err != nil {
			logger.Logger.Error("Failed to generate pending auth token",
				"user_id", usr.ID,
				"error", err.Error(),
			)
			httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "PENDING_TOKEN_GENERATION_FAILED", "Failed to generate authentication token", "auth")
			return err
		}

		pendingResponse := dto.PendingTOTPResponse{
			RequiresTOTP:     true,
			PendingAuthToken: pendingToken,
			User:             buildUserProfile(usr),
		}

		logger.Logger.Info("TOTP verification required for login",
			"user_id", usr.ID,
			"pending_token_preview", auth.GetKeyPreview(pendingToken),
		)

		httputil.SendOKResponse(ctx, pendingResponse, verifiedMessage+". Please complete two-factor authentication.")
		return nil
	}

	challengeToken, otpCode, challenge, err := auth.GenerateEmailOTPChallenge(
		ctx.Request.Context(),
		auth.EmailOTPPurposeLogin,
		usr.Email,
		usr.ID,
	)
	if err != nil {
		logger.Logger.Error("Failed to generate login email OTP challenge",
			"user_id", usr.ID,
			"error", err.Error(),
		)
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "EMAIL_OTP_CHALLENGE_FAILED", "Failed to start email OTP verification", "auth")
		return err
	}

	if err := c.emailService.SendEmailOTP(usr.Email, usr.Username, "login", otpCode); err != nil {
		_ = auth.DeleteEmailOTPChallenge(ctx.Request.Context(), challengeToken)
		logger.Logger.Error("Failed to send login email OTP",
			"user_id", usr.ID,
			"email", usr.Email,
			"error", err.Error(),
		)
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "EMAIL_SEND_FAILED", "Failed to send verification code", "email")
		return err
	}

	pendingResponse := dto.PendingEmailOTPResponse{
		RequiresEmailOTP: true,
		ChallengeToken:   challengeToken,
		CooldownSeconds:  auth.CooldownSecondsForNextResend(challenge),
		Email:            usr.Email,
		User:             buildUserProfile(usr),
	}

	httputil.SendOKResponse(ctx, pendingResponse, verifiedMessage+". Please enter the verification code sent to your email.")
	return nil
}

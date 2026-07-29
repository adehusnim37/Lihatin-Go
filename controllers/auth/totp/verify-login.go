package totp

import (
	"context"
	"fmt"
	"net/http"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/internal/pkg/auth"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	usermodel "github.com/adehusnim37/lihatin-go/models/user"
	"github.com/gin-gonic/gin"
)

// VerifyTOTPLogin verifies TOTP code during login and issues JWT tokens
// This is the ONLY way to get JWT tokens when TOTP is enabled
func (c *Controller) VerifyTOTPLogin(ctx *gin.Context) {
	var req dto.VerifyTOTPLoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		validator.SendValidationError(ctx, err, &req)
		return
	}

	// Validate pending auth token and get userID
	userID, err := auth.ValidatePendingAuthToken(context.Background(), req.PendingAuthToken)
	if err != nil {
		logger.Logger.Warn("Invalid pending auth token for TOTP login",
			"error", err.Error(),
		)
		httputil.SendErrorResponse(ctx, http.StatusUnauthorized, "INVALID_AUTH_TOKEN", "Invalid authentication token", "auth")
		return
	}

	blocked, retryAfterSeconds, err := auth.EnforceSecondFactorRiskLimit(ctx.Request.Context(), "totp_verify", userID, ctx.ClientIP())
	if err != nil {
		logger.Logger.Warn("Second-factor risk guard unavailable for TOTP login verification",
			"user_id", userID,
			"ip", ctx.ClientIP(),
			"error", err.Error(),
		)
	} else if blocked {
		httputil.SendErrorResponse(ctx, http.StatusTooManyRequests, "SECOND_FACTOR_RATE_LIMITED", fmt.Sprintf("Too many verification attempts. Please wait %d seconds and try again.", retryAfterSeconds), "totp")
		return
	}

	// Get user and auth info
	user, err := c.repo.GetUserRepository().GetUserByID(userID)
	if err != nil {
		logger.Logger.Error("Failed to get user for TOTP login",
			"user_id", userID,
			"error", err.Error(),
		)
		httputil.HandleError(ctx, err, userID)
		return
	}

	userAuth, err := c.repo.GetUserAuthRepository().GetUserAuthByUserID(userID)
	if err != nil {
		logger.Logger.Error("Failed to get user auth for TOTP login",
			"user_id", userID,
			"error", err.Error(),
		)
		httputil.HandleError(ctx, err, userID)
		return
	}

	if user.IsAccountLocked() {
		httputil.SendErrorResponse(ctx, http.StatusForbidden, "USER_LOCKED", "Your account has been locked. Please contact support.", "auth")
		return
	}

	temporarilyBlocked, err := c.repo.GetUserAuthRepository().IsLoginTemporarilyBlocked(userID)
	if err != nil {
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "LOGIN_FAILED", "An error occurred during login", "auth")
		return
	}
	if temporarilyBlocked {
		httputil.SendErrorResponse(ctx, http.StatusTooManyRequests, "LOGIN_TEMPORARILY_BLOCKED", "Too many failed login attempts. Please try again later.", "auth")
		return
	}

	if userAuth.AccountStatus != usermodel.AccountStatusActive {
		httputil.SendErrorResponse(ctx, http.StatusForbidden, "ACCOUNT_DEACTIVATED", "Your account has been deactivated. Please contact support.", "auth")
		return
	}

	if !userAuth.IsEmailVerified {
		httputil.SendErrorResponse(ctx, http.StatusForbidden, "EMAIL_NOT_VERIFIED", "Your email is not verified. Please verify your email first.", "email")
		return
	}

	// Get TOTP secret
	encryptedSecret, err := c.repo.GetAuthMethodRepository().GetTOTPSecret(userAuth.ID)
	if err != nil {
		logger.Logger.Error("Failed to get TOTP secret for login",
			"user_id", userID,
			"error", err.Error(),
		)
		httputil.HandleError(ctx, err, userID)
		return
	}

	secret, err := auth.DecryptTOTPSecret(encryptedSecret)
	if err != nil {
		logger.Logger.Error("Failed to decrypt TOTP secret for login",
			"user_id", userID,
			"error", err.Error(),
		)
		httputil.HandleError(ctx, err, userID)
		return
	}

	// Validate TOTP code
	if !auth.ValidateTOTPCodeWithWindow(secret, req.TOTPCode, 1) {
		remainingAttempts, attemptErr := auth.IncrementPendingAuthAttempts(context.Background(), req.PendingAuthToken)
		if attemptErr != nil {
			logger.Logger.Warn("Failed to record pending auth attempt",
				"user_id", userID,
				"error", attemptErr.Error(),
			)
			httputil.HandleError(ctx, attemptErr, userID)
			return
		}

		logger.Logger.Warn("Invalid TOTP code during login",
			"user_id", userID,
			"remaining_attempts", remainingAttempts,
		)
		httputil.SendErrorResponse(
			ctx,
			http.StatusBadRequest,
			"INVALID_TOTP",
			fmt.Sprintf("Invalid verification code. %d attempt(s) remaining.", remainingAttempts),
			"totp",
		)
		return
	}

	// Consume pending auth token only after successful TOTP validation.
	if err := auth.ConsumePendingAuthToken(context.Background(), req.PendingAuthToken); err != nil {
		logger.Logger.Warn("Failed to consume pending auth token after successful TOTP",
			"user_id", userID,
			"error", err.Error(),
		)
	}

	// Update TOTP auth method last_used_at
	if err := c.repo.GetAuthMethodRepository().UpdateTOTPLastUsed(userAuth.ID); err != nil {
		logger.Logger.Warn("Failed to update TOTP last_used_at",
			"user_id", user.ID,
			"error", err.Error(),
		)
		// Don't fail login for this
	}

	if err := c.loginController.CompleteLogin(
		ctx,
		user,
		userAuth,
		usermodel.LoginMethodTOTP,
		"Two-factor authentication successful. Welcome back!",
	); err != nil {
		return
	}
}

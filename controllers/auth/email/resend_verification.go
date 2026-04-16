package email

import (
	"errors"
	"math"
	"net/http"
	"time"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/internal/pkg/auth"
	apperrors "github.com/adehusnim37/lihatin-go/internal/pkg/errors"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	"github.com/adehusnim37/lihatin-go/models/user"
	"github.com/gin-gonic/gin"
)

// ResendVerificationEmail resends verification email to user
func (c *Controller) ResendVerificationEmail(ctx *gin.Context) {
	var req dto.ResendVerificationRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		validator.SendValidationError(ctx, err, &req)
		return
	}

	emailOrUsername, err := decodeIdentifier(req.Identifier)
	if err != nil {
		httputil.SendErrorResponse(ctx, http.StatusBadRequest, "INVALID_IDENTIFIER", "Invalid identifier format", "identifier")
		return
	}

	genericMessage := "If account exists and is not yet verified, we have sent a new verification link to the registered email."
	genericData := map[string]any{"message": genericMessage}

	userAccount, userAuth, err := c.repo.GetUserAuthRepository().GetUserForLogin(emailOrUsername)
	if err != nil {
		if errors.Is(err, apperrors.ErrUserNotFound) {
			logger.Logger.Info("Resend verification requested for unknown identifier",
				"identifier", emailOrUsername,
			)
			httputil.SendOKResponse(ctx, genericData, genericMessage)
			return
		}

		logger.Logger.Error("Failed to get user auth info",
			"identifier", emailOrUsername,
			"error", err.Error(),
		)
		httputil.HandleError(ctx, err, emailOrUsername)
		return
	}
	stage := getResendStage(ctx, userAccount.Email)

	// Do not reveal account existence/verification state in response
	if userAuth.IsEmailVerified || !userAuth.IsActive {
		logger.Logger.Info("Resend verification skipped due to account state",
			"user_id", userAccount.ID,
			"is_verified", userAuth.IsEmailVerified,
			"is_active", userAuth.IsActive,
		)
		httputil.SendOKResponse(ctx, genericData, genericMessage)
		return
	}

	if userAuth.LastEmailSendAt != nil {
		cooldownDuration := getResendCooldownDuration(stage)
		timeSinceLastSend := time.Since(*userAuth.LastEmailSendAt)
		if timeSinceLastSend < cooldownDuration {
			remainingSeconds := int(math.Ceil((cooldownDuration - timeSinceLastSend).Seconds()))
			if remainingSeconds < 1 {
				remainingSeconds = 1
			}
			logger.Logger.Warn("Resend verification email attempted too soon",
				"user_id", userAccount.ID,
				"email", userAccount.Email,
				"seconds_since_last_send", timeSinceLastSend.Seconds(),
				"required_cooldown_seconds", cooldownDuration.Seconds(),
			)
			genericData["cooldown_remaining_seconds"] = remainingSeconds
			httputil.SendOKResponse(ctx, genericData, genericMessage)
			return
		}
	}

	// Generate new verification token
	token, err := auth.GenerateVerificationToken()
	if err != nil {
		logger.Logger.Error("Failed to generate verification token",
			"user_id", userAccount.ID,
			"error", err.Error(),
		)
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "TOKEN_GENERATION_FAILED", "Failed to generate verification token", "token", userAccount.ID)
		return
	}

	// Save token to database
	if err := c.repo.GetUserAuthRepository().SetEmailVerificationToken(userAccount.ID, token, user.EmailSourceResend); err != nil {
		logger.Logger.Error("Failed to save verification token",
			"user_id", userAccount.ID,
			"error", err.Error(),
		)
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "TOKEN_SAVE_FAILED", "Failed to save verification token", "database", userAccount.ID)
		return
	}

	// Send verification email
	if err := c.emailService.SendVerificationEmail(userAccount.Email, userAccount.Username, token); err != nil {
		logger.Logger.Error("Failed to resend verification email",
			"user_id", userAccount.ID,
			"email", userAccount.Email,
			"error", err.Error(),
		)
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "EMAIL_SEND_FAILED", "Failed to resend verification email", "email", userAccount.ID)
		return
	}

	logger.Logger.Info("Verification email resent successfully",
		"user_id", userAccount.ID,
		"email", userAccount.Email,
	)

	newStage := stage + 1
	setResendStage(ctx, userAccount.Email, newStage)
	genericData["cooldown_seconds"] = int(getResendCooldownDuration(newStage).Seconds())
	httputil.SendOKResponse(ctx, genericData, genericMessage)
}

func getResendCooldownDuration(stage int) time.Duration {
	if stage >= 2 {
		return 5 * time.Minute
	}
	return 1 * time.Minute
}

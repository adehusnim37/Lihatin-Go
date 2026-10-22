package support

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/internal/pkg/auth"
	apperrors "github.com/adehusnim37/lihatin-go/internal/pkg/errors"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	"github.com/adehusnim37/lihatin-go/middleware"
	supportmodel "github.com/adehusnim37/lihatin-go/models/support"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const recoveryOTPCooldown = time.Minute

func recoveryChallengeKey(ticketID string) string { return "support_recovery_otp:" + ticketID }
func recoveryCooldownKey(ticketID string) string  { return "support_recovery_otp_cooldown:" + ticketID }

// deleteRecoveryKey only removes this operation's lease/challenge reference.
func deleteRecoveryKey(ctx context.Context, client *redis.Client, key, value string) {
	const script = `if redis.call("GET", KEYS[1]) == ARGV[1] then return redis.call("DEL", KEYS[1]) end return 0`
	if err := client.Eval(ctx, script, []string{key}, value).Err(); err != nil {
		logger.Logger.Warn("Failed to clear recovery verification state", "error", err.Error())
	}
}

func (c *Controller) RequestAdminRecoveryOTP(ctx *gin.Context) {
	ctx.Header("Cache-Control", "no-store")
	actorID := strings.TrimSpace(ctx.GetString("user_id"))
	if actorID == "" || (ctx.GetString("role") != "admin" && ctx.GetString("role") != "super_admin") {
		httputil.SendErrorResponse(ctx, http.StatusForbidden, "ADMIN_REQUIRED", "Administrator permission is required", "auth")
		return
	}
	c.sendRecoveryOTP(ctx, strings.TrimSpace(ctx.Param("id")), actorID)
}

func (c *Controller) RequestPublicRecoveryOTP(ctx *gin.Context) {
	ticket, _, ok := c.authorizePublicConversation(ctx)
	if !ok {
		return
	}
	c.sendRecoveryOTP(ctx, ticket.ID, "")
}
func (c *Controller) RequestUserRecoveryOTP(ctx *gin.Context) {
	ticket, ok := c.resolveOwnedTicket(ctx, strings.TrimSpace(ctx.Param("id")))
	if !ok {
		return
	}
	if ticket.UserID == nil || *ticket.UserID != ctx.GetString("user_id") {
		httputil.SendErrorResponse(ctx, http.StatusForbidden, "SUPPORT_ACCESS_DENIED", "This recovery belongs to another account", "ticket")
		return
	}
	c.sendRecoveryOTP(ctx, ticket.ID, "")
}

func (c *Controller) sendRecoveryOTP(ctx *gin.Context, ticketID, actorID string) {
	ctx.Header("Cache-Control", "no-store")
	ticket, target, err := c.repo.RecoveryVerificationTarget(ctx.Request.Context(), ticketID, actorID)
	if err != nil {
		c.handleAppError(ctx, err)
		return
	}
	if actorID == "" && ticket.RecoveryVerificationRequestedAt == nil {
		httputil.SendErrorResponse(ctx, http.StatusForbidden, "RECOVERY_REVIEW_REQUIRED", "Support must request recovery verification first", "ticket")
		return
	}
	now := time.Now().UTC()
	if ticket.RequesterVerifiedAt != nil && ticket.RequesterVerifiedAt.After(now.Add(-24*time.Hour)) {
		httputil.SendErrorResponse(ctx, http.StatusConflict, "RECOVERY_ALREADY_VERIFIED", "Your recovery email is already verified. Admin review is still required", "ticket")
		return
	}
	manager := middleware.GetSessionManager()
	if manager == nil {
		httputil.SendErrorResponse(ctx, http.StatusServiceUnavailable, "SUPPORT_ACCESS_UNAVAILABLE", "Recovery verification is temporarily unavailable", "ticket")
		return
	}
	client := manager.GetRedisClient()
	lease, err := auth.GenerateSecureToken(16)
	if err != nil {
		c.handleAppError(ctx, err)
		return
	}
	acquired, err := client.SetNX(ctx.Request.Context(), recoveryCooldownKey(ticket.ID), lease, recoveryOTPCooldown).Result()
	if err != nil {
		httputil.SendErrorResponse(ctx, http.StatusServiceUnavailable, "SUPPORT_ACCESS_UNAVAILABLE", "Recovery verification is temporarily unavailable", "ticket")
		return
	}
	if !acquired {
		ctx.Header("Retry-After", "60")
		httputil.SendErrorResponse(ctx, http.StatusTooManyRequests, "RECOVERY_OTP_COOLDOWN", "Wait one minute before requesting another recovery code", "ticket")
		return
	}
	sent := false
	defer func() {
		if !sent {
			deleteRecoveryKey(context.Background(), client, recoveryCooldownKey(ticket.ID), lease)
		}
	}()
	blocked, err := c.enforceSupportAccessRateLimit(ctx.Request.Context(), "recovery_otp_send", ticket.TicketCode, target.Email, 3, time.Hour)
	if err != nil {
		httputil.SendErrorResponse(ctx, http.StatusServiceUnavailable, "SUPPORT_ACCESS_UNAVAILABLE", "Recovery verification is temporarily unavailable", "ticket")
		return
	}
	if blocked {
		httputil.SendErrorResponse(ctx, http.StatusTooManyRequests, "RECOVERY_OTP_RATE_LIMIT", "The recovery email limit is reached. Try again in an hour", "ticket")
		return
	}
	token, code, challenge, err := auth.GenerateEmailOTPChallenge(ctx.Request.Context(), auth.EmailOTPPurposeSupportRecovery, target.Email, ticket.ID)
	if err != nil {
		c.handleAppError(ctx, err)
		return
	}
	expires := time.Unix(challenge.ExpiresAt, 0).UTC()
	cleanup := func() {
		_ = auth.DeleteEmailOTPChallenge(context.Background(), token)
		deleteRecoveryKey(context.Background(), client, recoveryChallengeKey(ticket.ID), token)
		if err := c.repo.CancelRecoveryVerificationRequest(context.Background(), ticket.ID, token); err != nil {
			logger.Logger.Warn("Failed to cancel recovery verification", "ticket_id", ticket.ID, "error", err.Error())
		}
	}
	if err := c.repo.RecordRecoveryVerificationRequest(ctx.Request.Context(), ticket.ID, actorID, token, now, expires); err != nil {
		cleanup()
		c.handleAppError(ctx, err)
		return
	}
	if err := client.Set(ctx.Request.Context(), recoveryChallengeKey(ticket.ID), token, auth.EmailOTPChallengeTTL).Err(); err != nil {
		cleanup()
		httputil.SendErrorResponse(ctx, http.StatusServiceUnavailable, "SUPPORT_ACCESS_UNAVAILABLE", "Recovery verification is temporarily unavailable", "ticket")
		return
	}
	if err := c.emailSvc.SendSupportRecoveryOTPEmail(target.Email, ticket.TicketCode, code); err != nil {
		cleanup()
		logger.Logger.Warn("Failed to send recovery verification email", "ticket_id", ticket.ID, "error", err.Error())
		httputil.SendErrorResponse(ctx, http.StatusBadGateway, "RECOVERY_EMAIL_FAILED", "Recovery email could not be sent. Try again", "email")
		return
	}
	sent = true
	// No OTP, challenge token, or recovery link is returned to the administrator.
	httputil.SendOKResponse(ctx, gin.H{"requested_at": now, "expires_at": expires, "cooldown_seconds": 60}, "Recovery code emailed. The requester must enter it in their protected ticket conversation.")
}

func (c *Controller) VerifyPublicRecoveryOTP(ctx *gin.Context) {
	ticket, _, ok := c.authorizePublicConversation(ctx)
	if !ok {
		return
	}
	c.verifyRecoveryOTP(ctx, ticket)
}
func (c *Controller) VerifyUserRecoveryOTP(ctx *gin.Context) {
	ticket, ok := c.resolveOwnedTicket(ctx, strings.TrimSpace(ctx.Param("id")))
	if !ok {
		return
	}
	if ticket.UserID == nil || *ticket.UserID != ctx.GetString("user_id") {
		httputil.SendErrorResponse(ctx, http.StatusForbidden, "SUPPORT_ACCESS_DENIED", "This recovery belongs to another account", "ticket")
		return
	}
	c.verifyRecoveryOTP(ctx, ticket)
}
func (c *Controller) verifyRecoveryOTP(ctx *gin.Context, ticket *supportmodel.SupportTicket) {
	ctx.Header("Cache-Control", "no-store")
	var req dto.SupportVerifyRecoveryOTPRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		validator.SendValidationError(ctx, err, &req)
		return
	}
	if _, _, err := c.repo.RecoveryVerificationTarget(ctx.Request.Context(), ticket.ID, ""); err != nil {
		c.handleAppError(ctx, err)
		return
	}
	manager := middleware.GetSessionManager()
	if manager == nil {
		httputil.SendErrorResponse(ctx, http.StatusServiceUnavailable, "SUPPORT_ACCESS_UNAVAILABLE", "Recovery verification is temporarily unavailable", "ticket")
		return
	}
	blocked, err := c.enforceSupportAccessRateLimit(ctx.Request.Context(), "recovery_otp_verify", ticket.TicketCode, ticket.Email, 10, 10*time.Minute)
	if err != nil {
		httputil.SendErrorResponse(ctx, http.StatusServiceUnavailable, "SUPPORT_ACCESS_UNAVAILABLE", "Recovery verification is temporarily unavailable", "ticket")
		return
	}
	if blocked {
		httputil.SendErrorResponse(ctx, http.StatusTooManyRequests, "RECOVERY_OTP_RATE_LIMIT", "Too many recovery verification attempts. Wait before trying again", "otp_code")
		return
	}
	client := manager.GetRedisClient()
	token, err := client.Get(ctx.Request.Context(), recoveryChallengeKey(ticket.ID)).Result()
	if errors.Is(err, redis.Nil) {
		httputil.SendErrorResponse(ctx, http.StatusGone, "RECOVERY_OTP_EXPIRED", "No active recovery code. Request a new code in this ticket", "otp_code")
		return
	}
	if err != nil {
		httputil.SendErrorResponse(ctx, http.StatusServiceUnavailable, "SUPPORT_ACCESS_UNAVAILABLE", "Recovery verification is temporarily unavailable", "ticket")
		return
	}
	challenge, err := auth.VerifyEmailOTPChallenge(ctx.Request.Context(), token, req.OTPCode, auth.EmailOTPPurposeSupportRecovery)
	if err != nil {
		var invalid *auth.EmailOTPInvalidCodeError
		switch {
		case errors.As(err, &invalid):
			httputil.SendErrorResponse(ctx, http.StatusBadRequest, "RECOVERY_OTP_INVALID", "Incorrect recovery code. Check the latest email and try again", "otp_code")
		case errors.Is(err, auth.ErrEmailOTPAttemptsExceeded):
			deleteRecoveryKey(ctx.Request.Context(), client, recoveryChallengeKey(ticket.ID), token)
			httputil.SendErrorResponse(ctx, http.StatusTooManyRequests, "RECOVERY_OTP_ATTEMPTS_EXCEEDED", "Too many incorrect codes. Request a new code in this ticket", "otp_code")
		case errors.Is(err, auth.ErrEmailOTPChallengeExpired), errors.Is(err, auth.ErrEmailOTPChallengeNotFound):
			httputil.SendErrorResponse(ctx, http.StatusGone, "RECOVERY_OTP_EXPIRED", "This code expired or was already used. Request a new code in this ticket", "otp_code")
		default:
			c.handleAppError(ctx, apperrors.NewAppError("RECOVERY_VERIFICATION_FAILED", "Recovery verification could not be completed", http.StatusServiceUnavailable, "otp_code"))
		}
		return
	}
	if challenge.UserID != ticket.ID || !strings.EqualFold(challenge.Email, ticket.Email) {
		httputil.SendErrorResponse(ctx, http.StatusForbidden, "SUPPORT_ACCESS_DENIED", "This recovery code does not belong to this ticket", "otp_code")
		return
	}
	verifiedAt, err := c.repo.ConfirmRecoveryVerification(ctx.Request.Context(), ticket.ID, challenge.Email, token)
	if err != nil {
		c.handleAppError(ctx, err)
		return
	}
	deleteRecoveryKey(ctx.Request.Context(), client, recoveryChallengeKey(ticket.ID), token)
	httputil.SendOKResponse(ctx, dto.SupportRecoveryVerificationStatus{RequestedAt: ticket.RecoveryVerificationRequestedAt, VerifiedAt: verifiedAt}, "Email verified for recovery. Support still needs to review and approve the reset.")
}

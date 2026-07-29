package premium

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/adehusnim37/lihatin-go/dto"
	apperrors "github.com/adehusnim37/lihatin-go/internal/pkg/errors"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	"github.com/adehusnim37/lihatin-go/models/user"
	"github.com/gin-gonic/gin"
)

func (c *Controller) SendPremiumCodeEmail(ctx *gin.Context) {
	var uri dto.PremiumCodeIDRequest
	if err := ctx.ShouldBindUri(&uri); err != nil {
		validator.SendValidationError(ctx, err, &uri)
		return
	}

	var req dto.SendPremiumCodeEmailRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		validator.SendValidationError(ctx, err, &req)
		return
	}

	targetUserID := strings.TrimSpace(req.UserID)
	targetEmail := strings.TrimSpace(req.RecipientEmail)
	targetName := strings.TrimSpace(req.RecipientName)

	if targetUserID == "" && targetEmail == "" {
		httputil.SendErrorResponse(
			ctx,
			http.StatusBadRequest,
			"RECIPIENT_REQUIRED",
			"Either user_id or recipient_email must be provided",
			"recipient_email",
		)
		return
	}
	if targetUserID != "" && targetEmail != "" {
		httputil.SendErrorResponse(
			ctx,
			http.StatusBadRequest,
			"AMBIGUOUS_RECIPIENT",
			"Provide either user_id or recipient_email, not both",
			"recipient",
		)
		return
	}

	premiumKey, err := c.premiumRepo.GetUserPremiumKeyByID(uri.ID)
	if err != nil {
		httputil.HandleError(ctx, err, uri.ID)
		return
	}

	var targetUser *user.User
	if targetUserID != "" {
		targetUser, err = c.userRepo.GetUserByID(targetUserID)
		if err != nil {
			httputil.HandleError(ctx, err, targetUserID)
			return
		}
	} else {
		targetUser, err = c.userRepo.GetUserByEmail(targetEmail)
		if err != nil && !isUserNotFoundError(err) {
			httputil.HandleError(ctx, err, targetEmail)
			return
		}
	}

	if targetUser != nil {
		if targetUser.IsAccountLocked() {
			httputil.SendErrorResponse(
				ctx,
				http.StatusForbidden,
				"USER_ACCOUNT_LOCKED",
				"User account is locked",
				"user_id",
			)
			return
		}
		if targetUser.PremiumAccess != nil && targetUser.PremiumAccess.Status == user.PremiumAccessStatusRevoked {
			httputil.SendErrorResponse(
				ctx,
				http.StatusForbidden,
				"USER_PREMIUM_REVOKED",
				"User's premium access has been revoked",
				"user_id",
			)
			return
		}
		if targetUser.HasPremiumAccessAt(time.Now()) {
			httputil.SendErrorResponse(
				ctx,
				http.StatusBadRequest,
				"USER_ALREADY_PREMIUM",
				"User already has premium access",
				"user_id",
			)
			return
		}

		// A user_id always resolves to the account's own email address. This
		// prevents callers from pairing an eligible user ID with another email.
		targetEmail = strings.TrimSpace(targetUser.Email)
		if targetName == "" {
			targetName = strings.TrimSpace(targetUser.FirstName + " " + targetUser.LastName)
			if targetName == "" {
				targetName = strings.TrimSpace(targetUser.Username)
			}
		}
	}

	if targetEmail == "" {
		httputil.SendErrorResponse(
			ctx,
			http.StatusBadRequest,
			"RECIPIENT_EMAIL_REQUIRED",
			"Recipient email is required",
			"recipient_email",
		)
		return
	}

	if premiumKey.ValidUntil != nil && premiumKey.ValidUntil.Before(time.Now()) {
		httputil.SendErrorResponse(
			ctx,
			http.StatusBadRequest,
			"PREMIUM_CODE_EXPIRED",
			"Premium code has expired",
			"premium_code",
		)
		return
	}

	if premiumKey.LimitUsage != nil && premiumKey.UsageCount >= *premiumKey.LimitUsage {
		httputil.SendErrorResponse(
			ctx,
			http.StatusBadRequest,
			"PREMIUM_CODE_LIMIT_REACHED",
			"Premium code usage limit reached",
			"premium_code",
		)
		return
	}

	if err := c.emailSvc.SendPremiumCodeDeliveryEmail(
		targetEmail,
		targetName,
		premiumKey.Code,
		premiumKey.ValidUntil,
		premiumKey.LimitUsage,
		req.Note,
	); err != nil {
		httputil.SendErrorResponse(
			ctx,
			http.StatusInternalServerError,
			"PREMIUM_CODE_EMAIL_SEND_FAILED",
			"Failed to send premium code email",
			"recipient_email",
		)
		return
	}

	response := dto.SendPremiumCodeEmailResponse{
		PremiumKeyID:    premiumKey.ID,
		RecipientEmail:  targetEmail,
		RecipientName:   targetName,
		DeliveredSecret: premiumKey.Code,
		SentAt:          time.Now(),
	}
	httputil.SendOKResponse(ctx, response, "Premium code email sent successfully")
}

func isUserNotFoundError(err error) bool {
	var appErr *apperrors.AppError
	return errors.As(err, &appErr) && appErr.Code == apperrors.ErrUserNotFound.Code
}

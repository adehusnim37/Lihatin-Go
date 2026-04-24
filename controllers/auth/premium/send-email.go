package premium

import (
	"net/http"
	"strings"
	"time"

	"github.com/adehusnim37/lihatin-go/dto"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
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

	premiumKey, err := c.premiumRepo.GetUserPremiumKeyByID(uri.ID)
	if err != nil {
		httputil.HandleError(ctx, err, uri.ID)
		return
	}

	if targetEmail == "" && targetUserID != "" {
		userData, userErr := c.userRepo.GetUserByID(targetUserID)
		if userErr != nil {
			httputil.HandleError(ctx, userErr, targetUserID)
			return
		}
		targetEmail = strings.TrimSpace(userData.Email)
		if targetName == "" {
			targetName = strings.TrimSpace(userData.FirstName + " " + userData.LastName)
			if targetName == "" {
				targetName = strings.TrimSpace(userData.Username)
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

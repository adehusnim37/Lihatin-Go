package support

import (
	apperrors "github.com/adehusnim37/lihatin-go/internal/pkg/errors"
	"net/http"
	"strings"
	"time"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/internal/pkg/auth"
	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
	"github.com/adehusnim37/lihatin-go/internal/pkg/disposable"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	supportmodel "github.com/adehusnim37/lihatin-go/models/support"
	"github.com/adehusnim37/lihatin-go/models/user"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (c *Controller) CreateTicket(ctx *gin.Context) {
	var req dto.CreateSupportTicketRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		validator.SendValidationError(ctx, err, &req)
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Subject = strings.TrimSpace(req.Subject)
	req.Description = strings.TrimSpace(req.Description)
	req.Category = normalizeTicketCategory(req.Category)
	senderEmail := req.Email

	if policy := disposable.Global(); policy != nil {
		blocked, err := policy.ShouldBlockEmail(ctx.Request.Context(), senderEmail)
		if err != nil {
			logger.Logger.Warn("Disposable email policy check failed for support ticket",
				"email", senderEmail,
				"error", err.Error(),
			)
		}
		if blocked {
			httputil.HandleError(
				ctx,
				apperrors.NewAppError("DISPOSABLE_EMAIL_BLOCKED", "Disposable email addresses are not allowed. Please use a permanent email address.", http.StatusBadRequest, "email"),
				nil,
			)
			return
		}
	}

	todayCount, err := c.repo.CountTicketsByEmailToday(req.Email)
	if err != nil {
		c.handleAppErrorAs(ctx, err, "email")
		return
	}
	if todayCount >= 3 {
		httputil.HandleError(ctx, apperrors.NewAppError("SUPPORT_EMAIL_RATE_LIMIT", "You can only submit 3 tickets per email each day", http.StatusTooManyRequests, "email"), nil)
		return
	}

	captchaOK, err := c.verifyCaptcha(strings.TrimSpace(req.CaptchaToken), ctx.ClientIP())
	if err != nil {
		logger.Logger.Warn("Captcha validation error", "error", err.Error(), "ip", ctx.ClientIP(), "email", req.Email)
		httputil.HandleError(ctx, apperrors.NewAppError("CAPTCHA_VERIFICATION_FAILED", "Captcha verification failed", http.StatusBadRequest, "captcha_token"), nil)
		return
	}
	if !captchaOK {
		httputil.HandleError(ctx, apperrors.NewAppError("CAPTCHA_VERIFICATION_FAILED", "Captcha verification failed", http.StatusBadRequest, "captcha_token"), nil)
		return
	}

	ticketCode, err := c.generateTicketCode()
	if err != nil {
		logger.Logger.Error("Failed generating support ticket code", "error", err.Error())
		httputil.HandleError(ctx, apperrors.NewAppError("TICKET_CODE_GENERATION_FAILED", "Failed to create support ticket", http.StatusInternalServerError, "ticket"), nil)
		return
	}

	ticketID := uuid.NewString()
	now := time.Now()
	accessCode, err := auth.GenerateSecureToken(24)
	if err != nil {
		logger.Logger.Error("Failed generating support public access code", "error", err.Error())
		httputil.HandleError(ctx, apperrors.NewAppError("TICKET_CREATE_FAILED", "Failed to create support ticket", http.StatusInternalServerError, "ticket"), nil)
		return
	}

	var linkedUserID *string
	var linkedUserPtr *user.User
	if linked, err := c.authRepo.GetUserRepository().GetUserByEmail(req.Email); err == nil && linked != nil {
		linkedUserID = &linked.ID
		linkedUserPtr = linked
	}

	ticket := supportmodel.SupportTicket{
		ID:                   ticketID,
		TicketCode:           ticketCode,
		Email:                req.Email,
		Category:             req.Category,
		Subject:              req.Subject,
		Description:          req.Description,
		Status:               string(supportmodel.TicketStatusOpen),
		Priority:             "normal",
		UserID:               linkedUserID,
		PublicAccessCodeHash: hashSupportAccessCode(accessCode),
		IPAddress:            ctx.ClientIP(),
		UserAgent:            strings.TrimSpace(ctx.GetHeader("User-Agent")),
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	if err := c.repo.CreateTicket(&ticket); err != nil {
		logger.Logger.Error("Failed creating support ticket", "error", err.Error(), "email", req.Email)
		c.handleAppError(ctx, err)
		return
	}

	initialMessage := supportmodel.SupportMessage{
		ID:          uuid.NewString(),
		TicketID:    ticket.ID,
		SenderType:  string(supportmodel.SupportMessageSenderPublic),
		SenderEmail: &senderEmail,
		Body:        req.Description,
		IsInternal:  false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := c.repo.CreateMessage(&initialMessage); err != nil {
		logger.Logger.Warn("Failed creating initial support message", "ticket_code", ticket.TicketCode, "error", err.Error())
	}

	frontendURL := strings.TrimRight(config.GetEnvOrDefault(config.EnvFrontendURL, "http://localhost:3000"), "/")
	go func(t supportmodel.SupportTicket, linked *user.User) {
		displayName := "there"
		if linked != nil {
			fullName := strings.TrimSpace(linked.FirstName + " " + linked.LastName)
			if fullName != "" {
				displayName = fullName
			} else if strings.TrimSpace(linked.Username) != "" {
				displayName = strings.TrimSpace(linked.Username)
			}
		}

		if sendErr := c.emailSvc.SendSupportTicketConfirmationEmail(
			t.Email,
			displayName,
			t.TicketCode,
			buildCategoryLabel(t.Category),
			accessCode,
			frontendURL,
			t.CreatedAt,
		); sendErr != nil {
			logger.Logger.Error("Failed sending support confirmation email", "ticket_code", t.TicketCode, "error", sendErr.Error())
		}

		if sendErr := c.emailSvc.SendSupportTicketAdminAlertEmail(t.TicketCode, t.Email, t.Subject, t.Category, frontendURL); sendErr != nil {
			logger.Logger.Warn("Failed sending support admin alert email", "ticket_code", t.TicketCode, "error", sendErr.Error())
		}
	}(ticket, linkedUserPtr)

	httputil.SendCreatedResponse(ctx, dto.CreateSupportTicketResponse{
		TicketCode: ticketCode,
	}, "Support ticket submitted successfully")
}

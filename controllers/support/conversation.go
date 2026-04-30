package support

import (
	"errors"
	"fmt"
	apperrors "github.com/adehusnim37/lihatin-go/internal/pkg/errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/internal/pkg/auth"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	supportmodel "github.com/adehusnim37/lihatin-go/models/support"
	"github.com/adehusnim37/lihatin-go/repositories/supportrepo"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	supportAccessTokenHeader        = "X-Support-Access-Token"
	maxSupportMessageBodyLength     = 5000
	maxSupportAttachmentsPerMessage = 5
	maxSupportAttachmentSizeBytes   = 10 * 1024 * 1024
)

func (c *Controller) RequestAccessOTP(ctx *gin.Context) {
	var req dto.SupportRequestAccessOTPRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		validator.SendValidationError(ctx, err, &req)
		return
	}

	req.Ticket = strings.ToUpper(strings.TrimSpace(req.Ticket))
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	captchaOK, err := c.verifyCaptcha(strings.TrimSpace(req.CaptchaToken), ctx.ClientIP())
	if err != nil {
		logger.Logger.Warn("Support access OTP captcha validation error", "error", err.Error(), "ip", ctx.ClientIP(), "email", req.Email)
		httputil.HandleError(ctx, apperrors.NewAppError("CAPTCHA_VERIFICATION_FAILED", "Captcha verification failed", http.StatusBadRequest, "captcha_token"), nil)
		return
	}
	if !captchaOK {
		httputil.HandleError(ctx, apperrors.NewAppError("CAPTCHA_VERIFICATION_FAILED", "Captcha verification failed", http.StatusBadRequest, "captcha_token"), nil)
		return
	}

	status, err := c.repo.GetStatusByCode(req.Ticket)
	if err != nil {
		c.handleAppError(ctx, err)
		return
	}
	if status.Status == "resolved" || status.Status == "closed" {
		httputil.HandleError(ctx, apperrors.NewAppError("TICKET_CLOSED", "Ticket is already "+status.Status, http.StatusForbidden, "ticket"), nil)
		return
	}

	ticket, err := c.repo.GetTicketByCodeAndEmail(req.Ticket, req.Email)
	if err != nil {
		c.handleAppError(ctx, err)
		return
	}
	if ticket == nil {
		httputil.HandleError(ctx, apperrors.NewAppError("TICKET_NOT_FOUND", "Ticket not found for provided email", http.StatusNotFound, "ticket"), nil)
		return
	}

	blocked, rateLimitErr := c.enforceSupportAccessRateLimit(
		ctx.Request.Context(),
		"request_otp",
		ticket.TicketCode,
		req.Email,
		supportAccessOTPRequestLimit,
		supportAccessOTPRequestWindow,
	)
	if rateLimitErr != nil {
		logger.Logger.Warn("Support access OTP rate limit check failed", "error", rateLimitErr.Error(), "ticket_code", ticket.TicketCode)
	}
	if blocked {
		httputil.HandleError(ctx, apperrors.NewAppError("SUPPORT_ACCESS_OTP_RATE_LIMIT", "Too many verification code requests for this ticket. Please try again later.", http.StatusTooManyRequests, "ticket"), nil)
		return
	}

	challengeToken, otpCode, challenge, err := auth.GenerateEmailOTPChallenge(
		ctx.Request.Context(),
		auth.EmailOTPPurposeSupportAccess,
		req.Email,
		ticket.ID,
	)
	if err != nil {
		httputil.HandleError(ctx, apperrors.NewAppError("SUPPORT_ACCESS_OTP_FAILED", "Failed to create verification challenge", http.StatusInternalServerError, "ticket"), nil)
		return
	}

	if err := c.emailSvc.SendSupportAccessOTPEmail(req.Email, ticket.TicketCode, otpCode); err != nil {
		_ = auth.DeleteEmailOTPChallenge(ctx.Request.Context(), challengeToken)
		httputil.HandleError(ctx, apperrors.NewAppError("SUPPORT_ACCESS_OTP_EMAIL_FAILED", "Failed to send verification code", http.StatusInternalServerError, "email"), nil)
		return
	}

	httputil.SendOKResponse(ctx, dto.SupportOTPChallengeResponse{
		ChallengeToken:  challengeToken,
		CooldownSeconds: auth.CooldownSecondsForNextResend(challenge),
	}, "Support access verification code sent")
}

func (c *Controller) ResendAccessOTP(ctx *gin.Context) {
	var req dto.SupportResendAccessOTPRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		validator.SendValidationError(ctx, err, &req)
		return
	}

	captchaOK, err := c.verifyCaptcha(strings.TrimSpace(req.CaptchaToken), ctx.ClientIP())
	if err != nil {
		logger.Logger.Warn("Support resend OTP captcha validation error", "error", err.Error(), "ip", ctx.ClientIP())
		httputil.HandleError(ctx, apperrors.NewAppError("CAPTCHA_VERIFICATION_FAILED", "Captcha verification failed", http.StatusBadRequest, "captcha_token"), nil)
		return
	}
	if !captchaOK {
		httputil.HandleError(ctx, apperrors.NewAppError("CAPTCHA_VERIFICATION_FAILED", "Captcha verification failed", http.StatusBadRequest, "captcha_token"), nil)
		return
	}

	challenge, err := auth.GetEmailOTPChallenge(ctx.Request.Context(), req.ChallengeToken)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrEmailOTPChallengeNotFound), errors.Is(err, auth.ErrEmailOTPChallengeExpired):
			httputil.HandleError(ctx, apperrors.NewAppError("SUPPORT_ACCESS_CHALLENGE_EXPIRED", "Verification session expired. Please request a new code.", http.StatusGone, "challenge_token"), nil)
			return
		default:
			httputil.HandleError(ctx, apperrors.NewAppError("SUPPORT_ACCESS_OTP_RESEND_FAILED", "Failed to resend verification code", http.StatusInternalServerError, "challenge_token"), nil)
			return
		}
	}

	if challenge.Purpose != auth.EmailOTPPurposeSupportAccess {
		httputil.HandleError(ctx, apperrors.NewAppError("SUPPORT_ACCESS_CHALLENGE_INVALID", "Invalid verification challenge", http.StatusBadRequest, "challenge_token"), nil)
		return
	}

	ticket, err := c.repo.GetTicketByID(strings.TrimSpace(challenge.UserID))
	if err != nil || ticket == nil {
		httputil.HandleError(ctx, apperrors.NewAppError("TICKET_NOT_FOUND", "Ticket not found", http.StatusNotFound, "challenge_token"), nil)
		return
	}
	
	if ticket.Status == "resolved" || ticket.Status == "closed" {
		httputil.HandleError(ctx, apperrors.NewAppError("TICKET_CLOSED", "Ticket is already closed/resolved", http.StatusForbidden, "challenge_token"), nil)
		return
	}

	ticketCode := ticket.TicketCode

	blocked, rateLimitErr := c.enforceSupportAccessRateLimit(
		ctx.Request.Context(),
		"resend_otp",
		ticketCode,
		challenge.Email,
		supportAccessOTPResendLimit,
		supportAccessOTPResendWindow,
	)
	if rateLimitErr != nil {
		logger.Logger.Warn("Support resend OTP rate limit check failed", "error", rateLimitErr.Error(), "ticket_code", ticketCode)
	}
	if blocked {
		httputil.HandleError(ctx, apperrors.NewAppError("SUPPORT_ACCESS_OTP_RESEND_RATE_LIMIT", "Too many resend attempts for this ticket. Please try again later.", http.StatusTooManyRequests, "challenge_token"), nil)
		return
	}

	challenge, otpCode, err := auth.ResendEmailOTPChallenge(ctx.Request.Context(), req.ChallengeToken)
	if err != nil {
		var cooldownErr *auth.EmailOTPCooldownError
		switch {
		case errors.As(err, &cooldownErr):
			httputil.SendOKResponse(ctx, dto.SupportOTPChallengeResponse{
				ChallengeToken:  req.ChallengeToken,
				CooldownSeconds: cooldownErr.RemainingSeconds,
			}, "Please wait before requesting another code")
			return
		case errors.Is(err, auth.ErrEmailOTPChallengeNotFound), errors.Is(err, auth.ErrEmailOTPChallengeExpired):
			httputil.HandleError(ctx, apperrors.NewAppError("SUPPORT_ACCESS_CHALLENGE_EXPIRED", "Verification session expired. Please request a new code.", http.StatusGone, "challenge_token"), nil)
			return
		default:
			httputil.HandleError(ctx, apperrors.NewAppError("SUPPORT_ACCESS_OTP_RESEND_FAILED", "Failed to resend verification code", http.StatusInternalServerError, "challenge_token"), nil)
			return
		}
	}

	if err := c.emailSvc.SendSupportAccessOTPEmail(challenge.Email, ticketCode, otpCode); err != nil {
		httputil.HandleError(ctx, apperrors.NewAppError("SUPPORT_ACCESS_OTP_EMAIL_FAILED", "Failed to resend verification code", http.StatusInternalServerError, "email"), nil)
		return
	}

	httputil.SendOKResponse(ctx, dto.SupportOTPChallengeResponse{
		ChallengeToken:  req.ChallengeToken,
		CooldownSeconds: auth.CooldownSecondsForNextResend(challenge),
	}, "Support access verification code sent")
}

func (c *Controller) VerifyAccessOTP(ctx *gin.Context) {
	var req dto.SupportVerifyAccessOTPRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		validator.SendValidationError(ctx, err, &req)
		return
	}

	otpCode := auth.ParseOTPCode(strings.TrimSpace(req.OTPCode))
	if otpCode == "" {
		httputil.HandleError(ctx, apperrors.NewAppError("INVALID_OTP", "Verification code must be a 6-digit number", http.StatusBadRequest, "otp_code"), nil)
		return
	}

	challenge, err := auth.VerifyEmailOTPChallenge(ctx.Request.Context(), req.ChallengeToken, otpCode, auth.EmailOTPPurposeSupportAccess)
	if err != nil {
		var invalidCodeErr *auth.EmailOTPInvalidCodeError
		switch {
		case errors.As(err, &invalidCodeErr):
			httputil.SendValidationErrorResponse(ctx, "Validation failed", map[string]string{
				"otp_code": fmt.Sprintf("Invalid code. Remaining attempts: %d", invalidCodeErr.RemainingAttempts),
			})
			return
		case errors.Is(err, auth.ErrEmailOTPAttemptsExceeded):
			httputil.HandleError(ctx, apperrors.NewAppError("OTP_ATTEMPTS_EXCEEDED", "Too many invalid attempts. Request a new code.", http.StatusTooManyRequests, "otp_code"), nil)
			return
		case errors.Is(err, auth.ErrEmailOTPChallengeNotFound), errors.Is(err, auth.ErrEmailOTPChallengeExpired):
			httputil.HandleError(ctx, apperrors.NewAppError("SUPPORT_ACCESS_CHALLENGE_EXPIRED", "Verification session expired. Request a new code.", http.StatusGone, "challenge_token"), nil)
			return
		default:
			httputil.HandleError(ctx, apperrors.NewAppError("OTP_VERIFICATION_FAILED", "Failed to verify code", http.StatusInternalServerError, "otp_code"), nil)
			return
		}
	}

	ticketID := strings.TrimSpace(challenge.UserID)
	if ticketID == "" {
		httputil.HandleError(ctx, apperrors.NewAppError("SUPPORT_ACCESS_CHALLENGE_INVALID", "Invalid support verification challenge", http.StatusBadRequest, "challenge_token"), nil)
		return
	}

	ticket, err := c.repo.GetTicketByID(ticketID)
	if err != nil {
		c.handleAppError(ctx, err)
		return
	}
	if ticket == nil {
		httputil.HandleError(ctx, apperrors.NewAppError("TICKET_NOT_FOUND", "Ticket not found", http.StatusNotFound, "ticket"), nil)
		return
	}

	if !strings.EqualFold(strings.TrimSpace(ticket.Email), strings.TrimSpace(challenge.Email)) {
		httputil.HandleError(ctx, apperrors.NewAppError("SUPPORT_ACCESS_DENIED", "Ticket ownership verification failed", http.StatusUnauthorized, "email"), nil)
		return
	}

	accessToken, _, err := auth.CreateSupportAccessToken(ctx.Request.Context(), ticket.ID, ticket.TicketCode, ticket.Email)
	if err != nil {
		httputil.HandleError(ctx, apperrors.NewAppError("SUPPORT_ACCESS_TOKEN_FAILED", "Failed to create support access token", http.StatusInternalServerError, "ticket"), nil)
		return
	}

	httputil.SendOKResponse(ctx, dto.SupportAccessResponse{
		AccessToken:      accessToken,
		ExpiresInSeconds: int(auth.SupportAccessTokenTTL.Seconds()),
		Ticket:           c.toTrackResponse(ticket),
	}, "Support ticket access granted")
}

func (c *Controller) VerifyAccessCode(ctx *gin.Context) {
	var req dto.SupportVerifyAccessCodeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		validator.SendValidationError(ctx, err, &req)
		return
	}

	req.Ticket = strings.ToUpper(strings.TrimSpace(req.Ticket))
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Code = strings.TrimSpace(req.Code)

	ticket, err := c.repo.GetTicketByCodeAndEmail(req.Ticket, req.Email)
	if err != nil {
		c.handleAppError(ctx, err)
		return
	}
	if ticket == nil {
		httputil.HandleError(ctx, apperrors.NewAppError("TICKET_NOT_FOUND", "Ticket not found for provided email", http.StatusNotFound, "ticket"), nil)
		return
	}

	if hashSupportAccessCode(req.Code) != strings.TrimSpace(ticket.PublicAccessCodeHash) {
		httputil.HandleError(ctx, apperrors.NewAppError("SUPPORT_ACCESS_DENIED", "Invalid support access code", http.StatusUnauthorized, "code"), nil)
		return
	}

	accessToken, _, err := auth.CreateSupportAccessToken(ctx.Request.Context(), ticket.ID, ticket.TicketCode, ticket.Email)
	if err != nil {
		httputil.HandleError(ctx, apperrors.NewAppError("SUPPORT_ACCESS_TOKEN_FAILED", "Failed to create support access token", http.StatusInternalServerError, "ticket"), nil)
		return
	}

	httputil.SendOKResponse(ctx, dto.SupportAccessResponse{
		AccessToken:      accessToken,
		ExpiresInSeconds: int(auth.SupportAccessTokenTTL.Seconds()),
		Ticket:           c.toTrackResponse(ticket),
	}, "Support ticket access granted")
}

func (c *Controller) ListPublicConversation(ctx *gin.Context) {
	ticket, _, ok := c.authorizePublicConversation(ctx)
	if !ok {
		return
	}

	messages, err := c.repo.ListMessagesByTicketID(ticket.ID, supportrepo.MessageListFilters{IncludeInternal: false})
	if err != nil {
		c.handleAppError(ctx, err)
		return
	}

	httputil.SendOKResponse(ctx, c.toConversationResponse(ticket, messages), "Support conversation loaded")
}

func (c *Controller) SendPublicMessage(ctx *gin.Context) {
	ticket, email, ok := c.authorizePublicConversation(ctx)
	if !ok {
		return
	}

	body := strings.TrimSpace(ctx.PostForm("body"))
	if len(body) > maxSupportMessageBodyLength {
		httputil.HandleError(ctx, apperrors.NewAppError("SUPPORT_MESSAGE_TOO_LONG", "Message must be less than or equal to 5000 characters", http.StatusBadRequest, "body"), nil)
		return
	}

	messageID := uuid.NewString()
	attachments, err := c.collectSupportAttachments(ctx, ticket.ID, messageID)
	if err != nil {
		httputil.SendErrorResponse(ctx, http.StatusBadRequest, "SUPPORT_ATTACHMENT_INVALID", err.Error(), "attachments")
		return
	}

	if body == "" && len(attachments) == 0 {
		httputil.HandleError(ctx, apperrors.NewAppError("SUPPORT_MESSAGE_REQUIRED", "Message body or attachment is required", http.StatusBadRequest, "body"), nil)
		return
	}
	if body == "" {
		body = "Attachment uploaded"
	}

	now := time.Now()
	senderEmail := email
	message := supportmodel.SupportMessage{
		ID:          messageID,
		TicketID:    ticket.ID,
		SenderType:  string(supportmodel.SupportMessageSenderPublic),
		SenderEmail: &senderEmail,
		Body:        body,
		IsInternal:  false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	ticket, err = c.repo.GetStatusByCode(ticket.TicketCode)
	if err != nil {
		c.handleAppError(ctx, err)
		return
	}
	if ticket.Status == "resolved" || ticket.Status == "closed" {
		httputil.HandleError(ctx, apperrors.NewAppError("TICKET_CLOSED", "Ticket is already "+ticket.Status+", cant send message", http.StatusBadRequest, "ticket"), nil)
		return
	}

	if err := c.repo.CreateMessageWithAttachments(&message, attachments); err != nil {
		c.cleanupUploadedAttachments(ctx, attachments)
		c.handleAppError(ctx, err)
		return
	}

	_ = c.repo.MarkTicketAsActiveByReply(ticket.ID)
	go func(ticketCode string, fromEmail string, preview string) {
		if err := c.emailSvc.SendSupportTicketMessageToAdminEmail(
			ticketCode,
			fromEmail,
			"Public",
			preview,
			c.frontendURL(),
		); err != nil {
			// Best effort notification only.
		}
	}(ticket.TicketCode, senderEmail, supportMessagePreview(body))
	message.Attachments = attachments
	httputil.SendCreatedResponse(ctx, c.toMessageResponse(message), "Message sent")
}

func (c *Controller) ListUserTickets(ctx *gin.Context) {
	userID := strings.TrimSpace(ctx.GetString("user_id"))
	email := strings.ToLower(strings.TrimSpace(ctx.GetString("email")))
	if userID == "" || email == "" {
		httputil.HandleError(ctx, apperrors.NewAppError("AUTH_REQUIRED", "Authentication required", http.StatusUnauthorized, "auth"), nil)
		return
	}

	page := parsePositiveInt(ctx.DefaultQuery("page", "1"), 1)
	limit := parsePositiveInt(ctx.DefaultQuery("limit", "20"), 20)
	if limit > 100 {
		limit = 100
	}

	items, total, err := c.repo.ListTicketsForUser(userID, email, supportrepo.Pagination{Page: page, Limit: limit})
	if err != nil {
		c.handleAppError(ctx, err)
		return
	}

	result := make([]dto.AdminSupportTicketItem, 0, len(items))
	for _, item := range items {
		result = append(result, dto.AdminSupportTicketItem{
			ID:         item.ID,
			TicketCode: item.TicketCode,
			Email:      item.Email,
			Category:   item.Category,
			Subject:    item.Subject,
			Status:     item.Status,
			Priority:   item.Priority,
			UserID:     item.UserID,
			CreatedAt:  item.CreatedAt,
			ResolvedAt: item.ResolvedAt,
		})
	}

	totalPages := int((total + int64(limit) - 1) / int64(limit))
	if totalPages == 0 {
		totalPages = 1
	}

	httputil.SendOKResponse(ctx, dto.UserListSupportTicketsResponse{
		Items:      result,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, "Support tickets retrieved successfully")
}

func (c *Controller) ListUserConversation(ctx *gin.Context) {
	ticket, ok := c.resolveOwnedTicket(ctx, strings.TrimSpace(ctx.Param("id")))
	if !ok {
		return
	}

	messages, err := c.repo.ListMessagesByTicketID(ticket.ID, supportrepo.MessageListFilters{IncludeInternal: false})
	if err != nil {
		c.handleAppError(ctx, err)
		return
	}

	httputil.SendOKResponse(ctx, c.toConversationResponse(ticket, messages), "Support conversation loaded")
}

func (c *Controller) SendUserMessage(ctx *gin.Context) {
	ticket, ok := c.resolveOwnedTicket(ctx, strings.TrimSpace(ctx.Param("id")))
	if !ok {
		return
	}

	body := strings.TrimSpace(ctx.PostForm("body"))
	if len(body) > maxSupportMessageBodyLength {
		httputil.HandleError(ctx, apperrors.NewAppError("SUPPORT_MESSAGE_TOO_LONG", "Message must be less than or equal to 5000 characters", http.StatusBadRequest, "body"), nil)
		return
	}

	messageID := uuid.NewString()
	attachments, err := c.collectSupportAttachments(ctx, ticket.ID, messageID)
	if err != nil {
		httputil.SendErrorResponse(ctx, http.StatusBadRequest, "SUPPORT_ATTACHMENT_INVALID", err.Error(), "attachments")
		return
	}

	if body == "" && len(attachments) == 0 {
		httputil.HandleError(ctx, apperrors.NewAppError("SUPPORT_MESSAGE_REQUIRED", "Message body or attachment is required", http.StatusBadRequest, "body"), nil)
		return
	}
	if body == "" {
		body = "Attachment uploaded"
	}

	now := time.Now()
	userID := strings.TrimSpace(ctx.GetString("user_id"))
	senderUserID := userID
	senderEmail := strings.ToLower(strings.TrimSpace(ctx.GetString("email")))
	message := supportmodel.SupportMessage{
		ID:           messageID,
		TicketID:     ticket.ID,
		SenderType:   string(supportmodel.SupportMessageSenderUser),
		SenderUserID: &senderUserID,
		SenderEmail:  &senderEmail,
		Body:         body,
		IsInternal:   false,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	ticket, err = c.repo.GetStatusByCode(ticket.TicketCode)
	if err != nil {
		c.handleAppError(ctx, err)
		return
	}
	if ticket.Status == "resolved" || ticket.Status == "closed" {
		httputil.HandleError(ctx, apperrors.NewAppError("TICKET_CLOSED", "Ticket is already "+ticket.Status+", cant send message", http.StatusBadRequest, "ticket"), nil)
		return
	}

	if err := c.repo.CreateMessageWithAttachments(&message, attachments); err != nil {
		c.cleanupUploadedAttachments(ctx, attachments)
		c.handleAppError(ctx, err)
		return
	}

	_ = c.repo.MarkTicketAsActiveByReply(ticket.ID)
	go func(ticketCode string, fromEmail string, preview string) {
		if err := c.emailSvc.SendSupportTicketMessageToAdminEmail(
			ticketCode,
			fromEmail,
			"User",
			preview,
			c.frontendURL(),
		); err != nil {
			// Best effort notification only.
		}
	}(ticket.TicketCode, senderEmail, supportMessagePreview(body))
	message.Attachments = attachments
	httputil.SendCreatedResponse(ctx, c.toMessageResponse(message), "Message sent")
}

func (c *Controller) DownloadUserAttachment(ctx *gin.Context) {
	attachmentID := strings.TrimSpace(ctx.Param("attachmentID"))
	if attachmentID == "" {
		httputil.HandleError(ctx, apperrors.NewAppError("ATTACHMENT_ID_REQUIRED", "Attachment ID is required", http.StatusBadRequest, "attachment"), nil)
		return
	}

	attachment, err := c.repo.GetAttachmentByID(attachmentID)
	if err != nil {
		c.handleAppError(ctx, err)
		return
	}
	if attachment == nil {
		httputil.HandleError(ctx, apperrors.NewAppError("ATTACHMENT_NOT_FOUND", "Attachment not found", http.StatusNotFound, "attachment"), nil)
		return
	}

	ticket, ok := c.resolveOwnedTicket(ctx, attachment.TicketID)
	if !ok || ticket == nil {
		return
	}

	sendAttachment(ctx, attachment)
}

func (c *Controller) ListAdminConversation(ctx *gin.Context) {
	ticketID := strings.TrimSpace(ctx.Param("id"))
	if ticketID == "" {
		httputil.HandleError(ctx, apperrors.NewAppError("TICKET_ID_REQUIRED", "Ticket ID is required", http.StatusBadRequest, "ticket"), nil)
		return
	}

	ticket, err := c.repo.GetTicketByID(ticketID)
	if err != nil {
		c.handleAppError(ctx, err)
		return
	}
	if ticket == nil {
		httputil.HandleError(ctx, apperrors.NewAppError("TICKET_NOT_FOUND", "Support ticket not found", http.StatusNotFound, "ticket"), nil)
		return
	}

	messages, err := c.repo.ListMessagesByTicketID(ticket.ID, supportrepo.MessageListFilters{IncludeInternal: true})
	if err != nil {
		c.handleAppError(ctx, err)
		return
	}

	httputil.SendOKResponse(ctx, c.toConversationResponse(ticket, messages), "Support conversation loaded")
}

func (c *Controller) SendAdminMessage(ctx *gin.Context) {
	ticketID := strings.TrimSpace(ctx.Param("id"))
	if ticketID == "" {
		httputil.HandleError(ctx, apperrors.NewAppError("TICKET_ID_REQUIRED", "Ticket ID is required", http.StatusBadRequest, "ticket"), nil)
		return
	}

	ticket, err := c.repo.GetTicketByID(ticketID)
	if err != nil {
		c.handleAppError(ctx, err)
		return
	}
	if ticket == nil {
		httputil.HandleError(ctx, apperrors.NewAppError("TICKET_NOT_FOUND", "Support ticket not found", http.StatusNotFound, "ticket"), nil)
		return
	}

	body := strings.TrimSpace(ctx.PostForm("body"))
	if len(body) > maxSupportMessageBodyLength {
		httputil.HandleError(ctx, apperrors.NewAppError("SUPPORT_MESSAGE_TOO_LONG", "Message must be less than or equal to 5000 characters", http.StatusBadRequest, "body"), nil)
		return
	}

	ticket, err = c.repo.GetStatusByCode(ticket.TicketCode)
	if err != nil {
		c.handleAppError(ctx, err)
		return
	}
	if ticket.Status == "resolved" || ticket.Status == "closed" {
		httputil.HandleError(ctx, apperrors.NewAppError("TICKET_CLOSED", "Ticket is already "+ticket.Status+", cant send message", http.StatusBadRequest, "ticket"), nil)
		return
	}

	messageID := uuid.NewString()
	attachments, err := c.collectSupportAttachments(ctx, ticket.ID, messageID)
	if err != nil {
		httputil.SendErrorResponse(ctx, http.StatusBadRequest, "SUPPORT_ATTACHMENT_INVALID", err.Error(), "attachments")
		return
	}

	if body == "" && len(attachments) == 0 {
		httputil.HandleError(ctx, apperrors.NewAppError("SUPPORT_MESSAGE_REQUIRED", "Message body or attachment is required", http.StatusBadRequest, "body"), nil)
		return
	}
	if body == "" {
		body = "Attachment uploaded"
	}

	isInternal := strings.EqualFold(strings.TrimSpace(ctx.PostForm("is_internal")), "true")
	now := time.Now()
	adminID := strings.TrimSpace(ctx.GetString("user_id"))
	adminEmail := strings.ToLower(strings.TrimSpace(ctx.GetString("email")))
	message := supportmodel.SupportMessage{
		ID:           messageID,
		TicketID:     ticket.ID,
		SenderType:   string(supportmodel.SupportMessageSenderAdmin),
		SenderUserID: &adminID,
		SenderEmail:  &adminEmail,
		Body:         body,
		IsInternal:   isInternal,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := c.repo.CreateMessageWithAttachments(&message, attachments); err != nil {
		c.cleanupUploadedAttachments(ctx, attachments)
		c.handleAppError(ctx, err)
		return
	}

	if !isInternal {
		_ = c.repo.MarkTicketAsActiveByReply(ticket.ID)
		go func(toEmail string, ticketCode string, preview string) {
			if err := c.emailSvc.SendSupportTicketMessageToRequesterEmail(
				toEmail,
				ticketCode,
				"Support Team",
				preview,
				c.frontendURL(),
			); err != nil {
				// Best effort notification only.
			}
		}(ticket.Email, ticket.TicketCode, supportMessagePreview(body))
	}
	message.Attachments = attachments
	httputil.SendCreatedResponse(ctx, c.toMessageResponse(message), "Message sent")
}

func (c *Controller) DownloadAdminAttachment(ctx *gin.Context) {
	attachmentID := strings.TrimSpace(ctx.Param("attachmentID"))
	if attachmentID == "" {
		httputil.HandleError(ctx, apperrors.NewAppError("ATTACHMENT_ID_REQUIRED", "Attachment ID is required", http.StatusBadRequest, "attachment"), nil)
		return
	}

	attachment, err := c.repo.GetAttachmentByID(attachmentID)
	if err != nil {
		c.handleAppError(ctx, err)
		return
	}
	if attachment == nil {
		httputil.HandleError(ctx, apperrors.NewAppError("ATTACHMENT_NOT_FOUND", "Attachment not found", http.StatusNotFound, "attachment"), nil)
		return
	}

	sendAttachment(ctx, attachment)
}

func (c *Controller) DownloadPublicAttachment(ctx *gin.Context) {
	ticket, _, ok := c.authorizePublicConversation(ctx)
	if !ok {
		return
	}

	attachmentID := strings.TrimSpace(ctx.Param("attachmentID"))
	if attachmentID == "" {
		httputil.HandleError(ctx, apperrors.NewAppError("ATTACHMENT_ID_REQUIRED", "Attachment ID is required", http.StatusBadRequest, "attachment"), nil)
		return
	}

	attachment, err := c.repo.GetAttachmentByID(attachmentID)
	if err != nil {
		c.handleAppError(ctx, err)
		return
	}
	if attachment == nil {
		httputil.HandleError(ctx, apperrors.NewAppError("ATTACHMENT_NOT_FOUND", "Attachment not found", http.StatusNotFound, "attachment"), nil)
		return
	}

	if strings.TrimSpace(attachment.TicketID) != strings.TrimSpace(ticket.ID) {
		httputil.HandleError(ctx, apperrors.NewAppError("SUPPORT_ATTACHMENT_ACCESS_DENIED", "Attachment does not belong to ticket", http.StatusForbidden, "attachment"), nil)
		return
	}

	sendAttachment(ctx, attachment)
}

func (c *Controller) authorizePublicConversation(ctx *gin.Context) (*supportmodel.SupportTicket, string, bool) {
	ticketCode := strings.ToUpper(strings.TrimSpace(ctx.Param("ticketCode")))
	if ticketCode == "" {
		ticketCode = strings.ToUpper(strings.TrimSpace(ctx.Query("ticket")))
	}
	email := strings.ToLower(strings.TrimSpace(ctx.Query("email")))
	if email == "" {
		email = strings.ToLower(strings.TrimSpace(ctx.PostForm("email")))
	}

	accessToken := strings.TrimSpace(ctx.GetHeader(supportAccessTokenHeader))
	if accessToken == "" {
		accessToken = strings.TrimSpace(ctx.Query("access_token"))
	}
	if accessToken == "" {
		accessToken = strings.TrimSpace(ctx.PostForm("access_token"))
	}

	if ticketCode == "" || email == "" {
		httputil.HandleError(ctx, apperrors.NewAppError("SUPPORT_ACCESS_REQUIRED", "Ticket and email are required", http.StatusBadRequest, "ticket"), nil)
		return nil, "", false
	}
	if accessToken == "" {
		httputil.HandleError(ctx, apperrors.NewAppError("SUPPORT_ACCESS_REQUIRED", "Support access token is required", http.StatusUnauthorized, "access_token"), nil)
		return nil, "", false
	}

	ticket, err := c.repo.GetTicketByCodeAndEmail(ticketCode, email)
	if err != nil {
		c.handleAppError(ctx, err)
		return nil, "", false
	}
	if ticket == nil {
		httputil.HandleError(ctx, apperrors.NewAppError("TICKET_NOT_FOUND", "Ticket not found for provided email", http.StatusNotFound, "ticket"), nil)
		return nil, "", false
	}

	tokenPayload, err := auth.GetSupportAccessToken(ctx.Request.Context(), accessToken)
	if err != nil || tokenPayload == nil {
		httputil.HandleError(ctx, apperrors.NewAppError("SUPPORT_ACCESS_INVALID", "Support access token is invalid or expired", http.StatusUnauthorized, "access_token"), nil)
		return nil, "", false
	}

	if strings.TrimSpace(tokenPayload.TicketID) != strings.TrimSpace(ticket.ID) ||
		!strings.EqualFold(strings.TrimSpace(tokenPayload.TicketCode), strings.TrimSpace(ticket.TicketCode)) ||
		!strings.EqualFold(strings.TrimSpace(tokenPayload.Email), strings.TrimSpace(email)) {
		httputil.HandleError(ctx, apperrors.NewAppError("SUPPORT_ACCESS_DENIED", "Support access token does not match this ticket", http.StatusForbidden, "access_token"), nil)
		return nil, "", false
	}

	return ticket, email, true
}

func (c *Controller) resolveOwnedTicket(ctx *gin.Context, ticketID string) (*supportmodel.SupportTicket, bool) {
	if strings.TrimSpace(ticketID) == "" {
		httputil.HandleError(ctx, apperrors.NewAppError("TICKET_ID_REQUIRED", "Ticket ID is required", http.StatusBadRequest, "ticket"), nil)
		return nil, false
	}

	ticket, err := c.repo.GetTicketByID(ticketID)
	if err != nil {
		c.handleAppError(ctx, err)
		return nil, false
	}
	if ticket == nil {
		httputil.HandleError(ctx, apperrors.NewAppError("TICKET_NOT_FOUND", "Support ticket not found", http.StatusNotFound, "ticket"), nil)
		return nil, false
	}

	userID := strings.TrimSpace(ctx.GetString("user_id"))
	email := strings.ToLower(strings.TrimSpace(ctx.GetString("email")))
	isOwnerByID := ticket.UserID != nil && strings.TrimSpace(*ticket.UserID) != "" && strings.TrimSpace(*ticket.UserID) == userID
	isOwnerByEmail := email != "" && strings.EqualFold(strings.TrimSpace(ticket.Email), email)

	if !isOwnerByID && !isOwnerByEmail {
		httputil.HandleError(ctx, apperrors.NewAppError("SUPPORT_TICKET_ACCESS_DENIED", "You do not have access to this ticket", http.StatusForbidden, "ticket"), nil)
		return nil, false
	}

	return ticket, true
}

func (c *Controller) toTrackResponse(ticket *supportmodel.SupportTicket) dto.TrackSupportTicketResponse {
	if ticket == nil {
		return dto.TrackSupportTicketResponse{}
	}
	return dto.TrackSupportTicketResponse{
		TicketCode: ticket.TicketCode,
		Category:   ticket.Category,
		Subject:    ticket.Subject,
		Status:     ticket.Status,
		CreatedAt:  ticket.CreatedAt,
		ResolvedAt: ticket.ResolvedAt,
	}
}

func (c *Controller) toConversationResponse(ticket *supportmodel.SupportTicket, messages []supportmodel.SupportMessage) dto.SupportConversationResponse {
	if len(messages) == 0 && strings.TrimSpace(ticket.Description) != "" {
		email := strings.ToLower(strings.TrimSpace(ticket.Email))
		messages = append(messages, supportmodel.SupportMessage{
			ID:          "legacy-" + ticket.ID,
			TicketID:    ticket.ID,
			SenderType:  string(supportmodel.SupportMessageSenderPublic),
			SenderEmail: &email,
			Body:        ticket.Description,
			IsInternal:  false,
			CreatedAt:   ticket.CreatedAt,
			UpdatedAt:   ticket.CreatedAt,
		})
	}

	mapped := make([]dto.SupportMessageResponse, 0, len(messages))
	for _, message := range messages {
		mapped = append(mapped, c.toMessageResponse(message))
	}

	return dto.SupportConversationResponse{
		TicketCode: ticket.TicketCode,
		TicketID:   ticket.ID,
		Category:   ticket.Category,
		Subject:    ticket.Subject,
		Status:     ticket.Status,
		CreatedAt:  ticket.CreatedAt,
		UpdatedAt:  ticket.UpdatedAt,
		Messages:   mapped,
	}
}

func (c *Controller) toMessageResponse(message supportmodel.SupportMessage) dto.SupportMessageResponse {
	attachments := make([]dto.SupportAttachmentResponse, 0, len(message.Attachments))
	for _, attachment := range message.Attachments {
		attachments = append(attachments, dto.SupportAttachmentResponse{
			ID:          attachment.ID,
			FileName:    attachment.FileName,
			ContentType: attachment.ContentType,
			SizeBytes:   attachment.SizeBytes,
			CreatedAt:   attachment.CreatedAt,
		})
	}

	return dto.SupportMessageResponse{
		ID:           message.ID,
		TicketID:     message.TicketID,
		SenderType:   message.SenderType,
		SenderUserID: message.SenderUserID,
		SenderEmail:  message.SenderEmail,
		Body:         message.Body,
		IsInternal:   message.IsInternal,
		CreatedAt:    message.CreatedAt,
		UpdatedAt:    message.UpdatedAt,
		Attachments:  attachments,
	}
}

func (c *Controller) collectSupportAttachments(ctx *gin.Context, ticketID, messageID string) ([]supportmodel.SupportAttachment, error) {
	if !strings.EqualFold(ctx.ContentType(), "multipart/form-data") {
		return nil, nil
	}

	form, err := ctx.MultipartForm()
	if err != nil {
		return nil, fmt.Errorf("failed to parse multipart upload")
	}

	files := form.File["attachments"]
	if len(files) == 0 {
		files = form.File["files"]
	}
	if len(files) == 0 {
		return nil, nil
	}
	if len(files) > maxSupportAttachmentsPerMessage {
		return nil, fmt.Errorf("maximum %d attachments per message", maxSupportAttachmentsPerMessage)
	}
	if c.attachmentStore == nil {
		return nil, fmt.Errorf("attachment upload storage is not configured")
	}

	now := time.Now()
	attachments := make([]supportmodel.SupportAttachment, 0, len(files))
	for _, header := range files {
		if header == nil {
			continue
		}
		if header.Size > maxSupportAttachmentSizeBytes {
			return nil, fmt.Errorf("each file must be <= %d MB", maxSupportAttachmentSizeBytes/(1024*1024))
		}

		opened, err := header.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to read attachment")
		}

		fileData, err := io.ReadAll(io.LimitReader(opened, maxSupportAttachmentSizeBytes+1))
		_ = opened.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to process attachment")
		}
		if int64(len(fileData)) > maxSupportAttachmentSizeBytes {
			return nil, fmt.Errorf("each file must be <= %d MB", maxSupportAttachmentSizeBytes/(1024*1024))
		}
		if len(fileData) == 0 {
			return nil, fmt.Errorf("attachment file cannot be empty")
		}

		contentType := strings.TrimSpace(header.Header.Get("Content-Type"))
		if contentType == "" {
			contentType = http.DetectContentType(fileData)
		}
		if len(contentType) > 100 {
			contentType = contentType[:100]
		}

		fileName := sanitizeSupportFileName(header.Filename)
		objectURL, objectKey, uploadErr := c.attachmentStore.UploadAttachment(
			ctx.Request.Context(),
			ticketID,
			messageID,
			fileName,
			contentType,
			fileData,
		)
		if uploadErr != nil {
			logger.Logger.Error(
				"Failed uploading support attachment to object storage",
				"ticket_id", ticketID,
				"message_id", messageID,
				"file_name", fileName,
				"content_type", contentType,
				"size_bytes", len(fileData),
				"error", uploadErr.Error(),
			)
			return nil, fmt.Errorf("failed to upload attachment")
		}

		attachments = append(attachments, supportmodel.SupportAttachment{
			ID:          uuid.NewString(),
			TicketID:    ticketID,
			MessageID:   messageID,
			FileName:    fileName,
			ContentType: contentType,
			SizeBytes:   int64(len(fileData)),
			ObjectKey:   objectKey,
			ObjectURL:   objectURL,
			CreatedAt:   now,
			UpdatedAt:   now,
		})
	}

	return attachments, nil
}

func (c *Controller) cleanupUploadedAttachments(ctx *gin.Context, attachments []supportmodel.SupportAttachment) {
	if c == nil || c.attachmentStore == nil || len(attachments) == 0 {
		return
	}

	for _, attachment := range attachments {
		if err := c.attachmentStore.DeleteAttachment(ctx.Request.Context(), attachment.ObjectKey); err != nil {
			logger.Logger.Error(
				"Failed deleting orphaned support attachment from object storage",
				"attachment_id", attachment.ID,
				"ticket_id", attachment.TicketID,
				"message_id", attachment.MessageID,
				"object_key", attachment.ObjectKey,
				"error", err.Error(),
			)
		}
	}
}

func sanitizeSupportFileName(raw string) string {
	base := filepath.Base(strings.TrimSpace(raw))
	base = strings.ReplaceAll(base, "\"", "")
	if base == "" || base == "." || base == string(filepath.Separator) {
		base = "attachment"
	}
	if len(base) > 180 {
		base = base[:180]
	}
	return base
}

func sendAttachment(ctx *gin.Context, attachment *supportmodel.SupportAttachment) {
	objectURL := strings.TrimSpace(attachment.ObjectURL)
	if objectURL == "" {
		httputil.HandleError(ctx, apperrors.NewAppError("ATTACHMENT_URL_NOT_AVAILABLE", "Attachment file URL is not available", http.StatusNotFound, "attachment"), nil)
		return
	}
	ctx.Redirect(http.StatusTemporaryRedirect, objectURL)
}

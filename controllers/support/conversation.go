package support

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	apperrors "github.com/adehusnim37/lihatin-go/internal/pkg/errors"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/internal/pkg/auth"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/identifier"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	supportmodel "github.com/adehusnim37/lihatin-go/models/support"
	"github.com/adehusnim37/lihatin-go/repositories/supportrepo"
	"github.com/gin-gonic/gin"
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
		logger.Logger.Warn("Support access OTP captcha validation error", "error", err.Error(), "ip", ctx.ClientIP())
		httputil.HandleError(ctx, apperrors.NewAppError("CAPTCHA_VERIFICATION_FAILED", "Captcha verification failed", http.StatusBadRequest, "captcha_token"), nil)
		return
	}
	if !captchaOK {
		httputil.HandleError(ctx, apperrors.NewAppError("CAPTCHA_VERIFICATION_FAILED", "Captcha verification failed", http.StatusBadRequest, "captcha_token"), nil)
		return
	}

	blocked, rateLimitErr := c.enforceSupportAccessRateLimit(
		ctx.Request.Context(),
		"request_otp",
		req.Ticket,
		req.Email,
		supportAccessOTPRequestLimit,
		supportAccessOTPRequestWindow,
	)
	if rateLimitErr != nil {
		httputil.HandleError(ctx, apperrors.NewAppError("SUPPORT_ACCESS_UNAVAILABLE", "Support verification is temporarily unavailable", http.StatusServiceUnavailable, "ticket"), nil)
		return
	}
	if blocked {
		httputil.HandleError(ctx, apperrors.NewAppError("SUPPORT_ACCESS_OTP_RATE_LIMIT", "Too many verification code requests for this ticket. Please try again later.", http.StatusTooManyRequests, "ticket"), nil)
		return
	}

	ticket, ticketErr := c.repo.GetTicketByCodeAndEmail(req.Ticket, req.Email)
	isDecoy := ticketErr != nil || ticket == nil || isPublicTicketClosed(ticket.Status)
	ticketID := "support-access-decoy"
	if !isDecoy {
		ticketID = ticket.ID
	}

	challengeToken, otpCode, challenge, err := auth.GenerateEmailOTPChallenge(
		ctx.Request.Context(),
		auth.EmailOTPPurposeSupportAccess,
		req.Email,
		ticketID,
	)
	if err != nil {
		httputil.HandleError(ctx, apperrors.NewAppError("SUPPORT_ACCESS_OTP_FAILED", "Failed to create verification challenge", http.StatusInternalServerError, "ticket"), nil)
		return
	}
	if isDecoy {
		challenge.IsDecoy = true
		if err := auth.SaveEmailOTPChallenge(ctx.Request.Context(), challengeToken, challenge); err != nil {
			_ = auth.DeleteEmailOTPChallenge(ctx.Request.Context(), challengeToken)
			httputil.HandleError(ctx, apperrors.NewAppError("SUPPORT_ACCESS_OTP_FAILED", "Failed to create verification challenge", http.StatusInternalServerError, "ticket"), nil)
			return
		}
	}

	response := dto.SupportOTPChallengeResponse{
		ChallengeToken:  challengeToken,
		CooldownSeconds: auth.CooldownSecondsForNextResend(challenge),
	}
	if !isDecoy {
		go func(email, ticketCode, code, token string) {
			if sendErr := c.emailSvc.SendSupportAccessOTPEmail(email, ticketCode, code); sendErr != nil {
				logger.Logger.Error("Failed sending support access OTP email", "ticket_code", ticketCode, "error", sendErr.Error())
				_ = auth.DeleteEmailOTPChallenge(context.Background(), token)
			}
		}(req.Email, ticket.TicketCode, otpCode, challengeToken)
	}

	httputil.SendSuccessResponse(ctx, http.StatusAccepted, response, "If the ticket details match, a verification email will be sent.")
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

	ticketCode := strings.TrimSpace(challenge.UserID)
	var ticket *supportmodel.SupportTicket
	if !challenge.IsDecoy {
		ticket, err = c.repo.GetTicketByID(ticketCode)
		if err != nil || ticket == nil {
			httputil.HandleError(ctx, apperrors.NewAppError("TICKET_NOT_FOUND", "Ticket not found", http.StatusNotFound, "challenge_token"), nil)
			return
		}
		if isPublicTicketClosed(ticket.Status) {
			httputil.HandleError(ctx, apperrors.NewAppError("TICKET_CLOSED", "Ticket is already closed/resolved", http.StatusForbidden, "challenge_token"), nil)
			return
		}
		ticketCode = ticket.TicketCode
	}

	blocked, rateLimitErr := c.enforceSupportAccessRateLimit(
		ctx.Request.Context(),
		"resend_otp",
		ticketCode,
		challenge.Email,
		supportAccessOTPResendLimit,
		supportAccessOTPResendWindow,
	)
	if rateLimitErr != nil {
		httputil.HandleError(ctx, apperrors.NewAppError("SUPPORT_ACCESS_UNAVAILABLE", "Support verification is temporarily unavailable", http.StatusServiceUnavailable, "challenge_token"), nil)
		return
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

	if !challenge.IsDecoy {
		if err := c.emailSvc.SendSupportAccessOTPEmail(challenge.Email, ticketCode, otpCode); err != nil {
			httputil.HandleError(ctx, apperrors.NewAppError("SUPPORT_ACCESS_OTP_EMAIL_FAILED", "Failed to resend verification code", http.StatusInternalServerError, "email"), nil)
			return
		}
	}

	httputil.SendOKResponse(ctx, dto.SupportOTPChallengeResponse{
		ChallengeToken:  req.ChallengeToken,
		CooldownSeconds: auth.CooldownSecondsForNextResend(challenge),
	}, "If the ticket details match, a verification email will be sent.")
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
	if isPublicTicketClosed(ticket.Status) {
		httputil.HandleError(ctx, apperrors.NewAppError("TICKET_CLOSED", "Ticket is closed", http.StatusForbidden, "ticket"), nil)
		return
	}

	accessToken, _, err := auth.CreateSupportAccessToken(ctx.Request.Context(), ticket.ID, ticket.TicketCode, ticket.Email, ticket.PublicAccessCodeHash)
	if err != nil {
		httputil.HandleError(ctx, apperrors.NewAppError("SUPPORT_ACCESS_TOKEN_FAILED", "Failed to create support access token", http.StatusInternalServerError, "ticket"), nil)
		return
	}
	if err := setSupportAccessCookie(ctx, accessToken, ticket.TicketCode); err != nil {
		_ = auth.DeleteSupportAccessToken(ctx.Request.Context(), accessToken)
		httputil.HandleError(ctx, apperrors.NewAppError("SUPPORT_ACCESS_TOKEN_FAILED", "Failed to create support session", http.StatusInternalServerError, "ticket"), nil)
		return
	}

	httputil.SendOKResponse(ctx, dto.SupportAccessResponse{
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

	blocked, rateLimitErr := c.enforceSupportAccessRateLimit(
		ctx.Request.Context(),
		"verify_code",
		req.Ticket,
		req.Email,
		supportAccessCodeVerifyLimit,
		supportAccessCodeVerifyWindow,
	)
	if rateLimitErr != nil {
		httputil.HandleError(ctx, apperrors.NewAppError("SUPPORT_ACCESS_UNAVAILABLE", "Support verification is temporarily unavailable", http.StatusServiceUnavailable, "ticket"), nil)
		return
	}
	if blocked {
		httputil.HandleError(ctx, apperrors.NewAppError("SUPPORT_ACCESS_RATE_LIMIT", "Too many verification attempts. Please try again later.", http.StatusTooManyRequests, "ticket"), nil)
		return
	}

	ticket, lookupErr := c.repo.GetTicketByCodeAndEmail(req.Ticket, req.Email)
	expectedHash := hashSupportAccessCode("invalid support access details")
	validTicket := lookupErr == nil && ticket != nil && !isPublicTicketClosed(ticket.Status)
	if validTicket {
		expectedHash = strings.TrimSpace(ticket.PublicAccessCodeHash)
	}
	codeMatches := subtle.ConstantTimeCompare(
		[]byte(hashSupportAccessCode(req.Code)),
		[]byte(expectedHash),
	) == 1
	if !validTicket || !codeMatches {
		httputil.HandleError(ctx, apperrors.NewAppError("SUPPORT_ACCESS_DENIED", "Invalid support access details", http.StatusUnauthorized, "code"), nil)
		return
	}

	accessToken, _, err := auth.CreateSupportAccessToken(ctx.Request.Context(), ticket.ID, ticket.TicketCode, ticket.Email, ticket.PublicAccessCodeHash)
	if err != nil {
		httputil.HandleError(ctx, apperrors.NewAppError("SUPPORT_ACCESS_TOKEN_FAILED", "Failed to create support access token", http.StatusInternalServerError, "ticket"), nil)
		return
	}
	if err := setSupportAccessCookie(ctx, accessToken, ticket.TicketCode); err != nil {
		_ = auth.DeleteSupportAccessToken(ctx.Request.Context(), accessToken)
		httputil.HandleError(ctx, apperrors.NewAppError("SUPPORT_ACCESS_TOKEN_FAILED", "Failed to create support session", http.StatusInternalServerError, "ticket"), nil)
		return
	}

	httputil.SendOKResponse(ctx, dto.SupportAccessResponse{
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

	if isPublicTicketClosed(ticket.Status) {
		httputil.HandleError(ctx, apperrors.NewAppError("TICKET_CLOSED", "Ticket is already "+ticket.Status+", cant send message", http.StatusBadRequest, "ticket"), nil)
		return
	}

	messageID := identifier.NewUUIDV7()
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
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	ticket, err = c.repo.GetStatusByCode(ticket.TicketCode)
	if err != nil {
		c.cleanupUploadedAttachments(ctx, attachments)
		c.handleAppError(ctx, err)
		return
	}
	if isPublicTicketClosed(ticket.Status) {
		c.cleanupUploadedAttachments(ctx, attachments)
		httputil.HandleError(ctx, apperrors.NewAppError("TICKET_CLOSED", "Ticket is already "+ticket.Status+", cant send message", http.StatusBadRequest, "ticket"), nil)
		return
	}

	if err := c.repo.CreateMessageWithAttachmentsIfActive(&message, attachments); err != nil {
		c.cleanupUploadedAttachments(ctx, attachments)
		if errors.Is(err, supportrepo.ErrSupportTicketClosed) {
			httputil.HandleError(ctx, apperrors.NewAppError("TICKET_CLOSED", "Ticket is closed", http.StatusConflict, "ticket"), nil)
			return
		}
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

	if isPublicTicketClosed(ticket.Status) {
		httputil.HandleError(ctx, apperrors.NewAppError("TICKET_CLOSED", "Ticket is already "+ticket.Status+", cant send message", http.StatusBadRequest, "ticket"), nil)
		return
	}

	messageID := identifier.NewUUIDV7()
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
		c.cleanupUploadedAttachments(ctx, attachments)
		c.handleAppError(ctx, err)
		return
	}
	if isPublicTicketClosed(ticket.Status) {
		c.cleanupUploadedAttachments(ctx, attachments)
		httputil.HandleError(ctx, apperrors.NewAppError("TICKET_CLOSED", "Ticket is already "+ticket.Status+", cant send message", http.StatusBadRequest, "ticket"), nil)
		return
	}

	if err := c.repo.CreateMessageWithAttachmentsIfActive(&message, attachments); err != nil {
		c.cleanupUploadedAttachments(ctx, attachments)
		if errors.Is(err, supportrepo.ErrSupportTicketClosed) {
			httputil.HandleError(ctx, apperrors.NewAppError("TICKET_CLOSED", "Ticket is closed", http.StatusConflict, "ticket"), nil)
			return
		}
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

	c.sendAttachment(ctx, attachment)
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
	if isPublicTicketClosed(ticket.Status) {
		httputil.HandleError(ctx, apperrors.NewAppError("TICKET_CLOSED", "Ticket is already "+ticket.Status+", cant send message", http.StatusBadRequest, "ticket"), nil)
		return
	}

	messageID := identifier.NewUUIDV7()
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

	if err := c.repo.CreateMessageWithAttachmentsIfActive(&message, attachments); err != nil {
		c.cleanupUploadedAttachments(ctx, attachments)
		if errors.Is(err, supportrepo.ErrSupportTicketClosed) {
			httputil.HandleError(ctx, apperrors.NewAppError("TICKET_CLOSED", "Ticket is closed", http.StatusConflict, "ticket"), nil)
			return
		}
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

	c.sendAttachment(ctx, attachment)
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

	c.sendAttachment(ctx, attachment)
}

func (c *Controller) authorizePublicConversation(ctx *gin.Context) (*supportmodel.SupportTicket, string, bool) {
	ticketCode := strings.ToUpper(strings.TrimSpace(ctx.Param("ticketCode")))
	accessToken := supportAccessTokenFromRequest(ctx)

	if ticketCode == "" {
		httputil.HandleError(ctx, apperrors.NewAppError("SUPPORT_ACCESS_REQUIRED", "Ticket is required", http.StatusBadRequest, "ticket"), nil)
		return nil, "", false
	}
	if accessToken == "" {
		httputil.HandleError(ctx, apperrors.NewAppError("SUPPORT_ACCESS_REQUIRED", "Support access token is required", http.StatusUnauthorized, "access_token"), nil)
		return nil, "", false
	}

	tokenPayload, err := auth.GetSupportAccessToken(ctx.Request.Context(), accessToken)
	if err != nil || tokenPayload == nil {
		clearSupportAccessCookie(ctx, ticketCode)
		httputil.HandleError(ctx, apperrors.NewAppError("SUPPORT_ACCESS_INVALID", "Support access token is invalid or expired", http.StatusUnauthorized, "access_token"), nil)
		return nil, "", false
	}

	ticket, err := c.repo.GetTicketByCode(ticketCode)
	if err != nil || ticket == nil {
		clearSupportAccessCookie(ctx, ticketCode)
		httputil.HandleError(ctx, apperrors.NewAppError("SUPPORT_ACCESS_DENIED", "Support access token does not match this ticket", http.StatusForbidden, "access_token"), nil)
		return nil, "", false
	}

	if !supportAccessPayloadMatchesTicket(tokenPayload, ticket) {
		clearSupportAccessCookie(ctx, ticketCode)
		httputil.HandleError(ctx, apperrors.NewAppError("SUPPORT_ACCESS_DENIED", "Support access token does not match this ticket", http.StatusForbidden, "access_token"), nil)
		return nil, "", false
	}
	if isPublicTicketClosed(ticket.Status) {
		_ = auth.DeleteSupportAccessToken(ctx.Request.Context(), accessToken)
		clearSupportAccessCookie(ctx, ticketCode)
		httputil.HandleError(ctx, apperrors.NewAppError("TICKET_CLOSED", "Ticket is closed", http.StatusForbidden, "ticket"), nil)
		return nil, "", false
	}

	return ticket, strings.ToLower(strings.TrimSpace(tokenPayload.Email)), true
}

func supportAccessPayloadMatchesTicket(payload *auth.SupportAccessTokenPayload, ticket *supportmodel.SupportTicket) bool {
	if payload == nil || ticket == nil {
		return false
	}
	return strings.TrimSpace(payload.TicketID) == strings.TrimSpace(ticket.ID) &&
		strings.EqualFold(strings.TrimSpace(payload.TicketCode), strings.TrimSpace(ticket.TicketCode)) &&
		strings.EqualFold(strings.TrimSpace(payload.Email), strings.TrimSpace(ticket.Email)) &&
		subtle.ConstantTimeCompare(
			[]byte(strings.TrimSpace(payload.AccessVersion)),
			[]byte(strings.TrimSpace(ticket.PublicAccessCodeHash)),
		) == 1
}

func (c *Controller) RevokePublicAccess(ctx *gin.Context) {
	ticketCode := strings.ToUpper(strings.TrimSpace(ctx.Param("ticketCode")))
	accessToken := supportAccessTokenFromRequest(ctx)
	if accessToken != "" {
		if err := auth.DeleteSupportAccessToken(ctx.Request.Context(), accessToken); err != nil {
			logger.Logger.Warn("Failed revoking support access token", "error", err.Error())
		}
	}
	clearSupportAccessCookie(ctx, ticketCode)
	httputil.SendNoContentResponse(ctx, "Support session ended")
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
			CreatedAt:   ticket.CreatedAt,
			UpdatedAt:   ticket.UpdatedAt,
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
	failAfterPartialUpload := func(err error) ([]supportmodel.SupportAttachment, error) {
		c.cleanupUploadedAttachments(ctx, attachments)
		return nil, err
	}
	for _, header := range files {
		if header == nil {
			continue
		}
		if header.Size > maxSupportAttachmentSizeBytes {
			return failAfterPartialUpload(fmt.Errorf("each file must be <= %d MB", maxSupportAttachmentSizeBytes/(1024*1024)))
		}

		opened, err := header.Open()
		if err != nil {
			return failAfterPartialUpload(fmt.Errorf("failed to read attachment"))
		}

		fileData, err := io.ReadAll(io.LimitReader(opened, maxSupportAttachmentSizeBytes+1))
		_ = opened.Close()
		if err != nil {
			return failAfterPartialUpload(fmt.Errorf("failed to process attachment"))
		}
		if int64(len(fileData)) > maxSupportAttachmentSizeBytes {
			return failAfterPartialUpload(fmt.Errorf("each file must be <= %d MB", maxSupportAttachmentSizeBytes/(1024*1024)))
		}
		if len(fileData) == 0 {
			return failAfterPartialUpload(fmt.Errorf("attachment file cannot be empty"))
		}

		contentType := strings.TrimSpace(header.Header.Get("Content-Type"))
		if contentType == "" {
			contentType = http.DetectContentType(fileData)
		}
		if len(contentType) > 100 {
			contentType = contentType[:100]
		}

		fileName := sanitizeSupportFileName(header.Filename)
		objectKey, uploadErr := c.attachmentStore.UploadAttachment(
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
			return failAfterPartialUpload(fmt.Errorf("failed to upload attachment"))
		}

		attachments = append(attachments, supportmodel.SupportAttachment{
			ID:          identifier.NewUUIDV7(),
			TicketID:    ticketID,
			MessageID:   messageID,
			FileName:    fileName,
			ContentType: contentType,
			SizeBytes:   int64(len(fileData)),
			ObjectKey:   objectKey,
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

	cleanupBase := context.Background()
	if ctx != nil && ctx.Request != nil {
		cleanupBase = context.WithoutCancel(ctx.Request.Context())
	}
	cleanupContext, cancel := context.WithTimeout(cleanupBase, 15*time.Second)
	defer cancel()

	for _, attachment := range attachments {
		if err := c.attachmentStore.DeleteAttachment(cleanupContext, attachment.ObjectKey); err != nil {
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

func (c *Controller) sendAttachment(ctx *gin.Context, attachment *supportmodel.SupportAttachment) {
	if c == nil || c.attachmentStore == nil || strings.TrimSpace(attachment.ObjectKey) == "" {
		httputil.HandleError(ctx, apperrors.NewAppError("ATTACHMENT_NOT_AVAILABLE", "Attachment file is not available", http.StatusNotFound, "attachment"), nil)
		return
	}

	body, contentType, contentLength, err := c.attachmentStore.OpenAttachment(ctx.Request.Context(), attachment.ObjectKey)
	if err != nil {
		logger.Logger.Error("Failed opening support attachment", "attachment_id", attachment.ID, "error", err.Error())
		httputil.HandleError(ctx, apperrors.NewAppError("ATTACHMENT_DOWNLOAD_FAILED", "Failed to download attachment", http.StatusBadGateway, "attachment"), nil)
		return
	}
	defer body.Close()

	if strings.TrimSpace(contentType) == "" {
		contentType = "application/octet-stream"
	}
	ctx.Header("Cache-Control", "private, no-store")
	ctx.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": attachment.FileName}))
	ctx.DataFromReader(http.StatusOK, contentLength, contentType, body, nil)
}

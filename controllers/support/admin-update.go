package support

import (
	"fmt"
	"net/http"
	"strings"

	apperrors "github.com/adehusnim37/lihatin-go/internal/pkg/errors"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/internal/pkg/auth"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	supportmodel "github.com/adehusnim37/lihatin-go/models/support"
	"github.com/adehusnim37/lihatin-go/models/user"
	"github.com/gin-gonic/gin"
)

func (c *Controller) UpdateTicket(ctx *gin.Context) {
	id := strings.TrimSpace(ctx.Param("id"))
	if id == "" {
		httputil.HandleError(ctx, apperrors.NewAppError("TICKET_ID_REQUIRED", "Ticket ID is required", http.StatusBadRequest, "id"), nil)
		return
	}

	var req dto.AdminUpdateSupportTicketRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		validator.SendValidationError(ctx, err, &req)
		return
	}

	ticket, err := c.repo.GetTicketByID(id)
	if err != nil {
		c.handleAppErrorAs(ctx, err, "id")
		return
	}
	if ticket == nil {
		httputil.HandleError(ctx, apperrors.NewAppError("TICKET_NOT_FOUND", "Support ticket not found", http.StatusNotFound, "id"), nil)
		return
	}

	action := strings.TrimSpace(req.Action)
	actionResult := ""
	adminID := strings.TrimSpace(ctx.GetString("user_id"))
	if action != "" {
		actionResult, err = c.applyAdminAction(action, ticket, adminID)
		if err != nil {
			httputil.SendErrorResponse(ctx, http.StatusBadRequest, "SUPPORT_ACTION_FAILED", err.Error(), "action")
			return
		}
	}

	var resolvedBy *string
	if req.Status == string(supportmodel.TicketStatusResolved) || req.Status == string(supportmodel.TicketStatusClosed) {
		if adminID != "" {
			resolvedBy = &adminID
		}
	}

	if err := c.repo.UpdateTicketStatus(id, req.Status, req.Priority, req.AdminNotes, resolvedBy); err != nil {
		c.handleAppErrorAs(ctx, err, "status")
		return
	}

	updatedTicket, err := c.repo.GetTicketByID(id)
	if err != nil {
		c.handleAppError(ctx, err)
		return
	}
	if updatedTicket == nil {
		httputil.HandleError(ctx, apperrors.NewAppError("TICKET_UPDATE_FAILED", "Ticket updated but failed to fetch latest state", http.StatusInternalServerError, "ticket"), nil)
		return
	}

	shouldSendResolvedEmail := ticket.ResolvedAt == nil && (updatedTicket.Status == string(supportmodel.TicketStatusResolved) || updatedTicket.Status == string(supportmodel.TicketStatusClosed))
	if shouldSendResolvedEmail {
		go c.sendResolvedTicketEmail(updatedTicket)
	} else {
		go c.sendTicketUpdatedEmail(updatedTicket, actionResult)
	}

	httputil.SendOKResponse(ctx, dto.AdminUpdateSupportTicketResponse{
		TicketCode:    updatedTicket.TicketCode,
		Status:        updatedTicket.Status,
		AdminNotes:    updatedTicket.AdminNotes,
		ResolvedAt:    updatedTicket.ResolvedAt,
		ActionApplied: actionResult,
	}, "Support ticket updated successfully")
}

func (c *Controller) applyAdminAction(action string, ticket *supportmodel.SupportTicket, actorID string) (string, error) {
	normalizedAction := strings.ToLower(strings.TrimSpace(action))
	if normalizedAction == "" || normalizedAction == "manual_response" {
		return normalizedAction, nil
	}

	targetUser, targetAuth, err := c.resolveTargetUser(ticket)
	if err != nil {
		return "", err
	}

	switch normalizedAction {
	case "unlock_user":
		reason := fmt.Sprintf("Unlocked from support ticket %s", ticket.TicketCode)
		if err := c.authRepo.GetUserAdminRepository().UnlockUser(targetUser.ID, reason, actorID); err != nil {
			return "", fmt.Errorf("failed to unlock user account")
		}
		if err := c.authRepo.GetUserAuthRepository().UnlockAccount(targetUser.ID); err != nil {
			return "", fmt.Errorf("failed to clear login lockout")
		}
		return "unlock_user", nil

	case "activate_user":
		if targetAuth.IsActive {
			return "activate_user", nil
		}
		if err := c.authRepo.GetUserAuthRepository().ActivateAccount(targetUser.ID); err != nil {
			return "", fmt.Errorf("failed to activate account")
		}
		return "activate_user", nil

	case "resend_verification":
		if targetAuth.IsEmailVerified {
			return "", fmt.Errorf("email already verified")
		}

		token, err := auth.GenerateVerificationToken()
		if err != nil {
			return "", fmt.Errorf("failed to generate verification token")
		}

		if err := c.authRepo.GetUserAuthRepository().SetEmailVerificationToken(targetUser.ID, token, user.EmailSourceResend); err != nil {
			return "", fmt.Errorf("failed to store verification token")
		}

		if err := c.emailSvc.SendVerificationEmail(targetUser.Email, targetUser.Username, token); err != nil {
			return "", fmt.Errorf("failed to resend verification email")
		}
		return "resend_verification", nil
	default:
		return "", fmt.Errorf("unsupported action")
	}
}

func (c *Controller) resolveTargetUser(ticket *supportmodel.SupportTicket) (*user.User, *user.UserAuth, error) {
	if ticket == nil {
		return nil, nil, fmt.Errorf("ticket not found")
	}

	if ticket.UserID != nil && strings.TrimSpace(*ticket.UserID) != "" {
		targetUser, err := c.authRepo.GetUserRepository().GetUserByID(strings.TrimSpace(*ticket.UserID))
		if err == nil && targetUser != nil {
			targetAuth, authErr := c.authRepo.GetUserAuthRepository().GetUserAuthByUserID(targetUser.ID)
			if authErr != nil {
				return nil, nil, fmt.Errorf("failed to read user auth record")
			}
			return targetUser, targetAuth, nil
		}
	}

	if strings.TrimSpace(ticket.Email) == "" {
		return nil, nil, fmt.Errorf("ticket has no linked email")
	}

	targetUser, targetAuth, err := c.authRepo.GetUserAuthRepository().GetUserForLogin(ticket.Email)
	if err != nil || targetUser == nil || targetAuth == nil {
		return nil, nil, fmt.Errorf("no user account found for ticket email")
	}

	return targetUser, targetAuth, nil
}

func (c *Controller) sendResolvedTicketEmail(ticket *supportmodel.SupportTicket) {
	if ticket == nil {
		return
	}
	if strings.TrimSpace(ticket.Email) == "" {
		return
	}

	resolutionMessage := "Your support request has been resolved. Please try logging in again."
	if ticket.AdminNotes != nil && strings.TrimSpace(*ticket.AdminNotes) != "" {
		resolutionMessage = strings.TrimSpace(*ticket.AdminNotes)
	}

	if err := c.emailSvc.SendSupportTicketResolvedEmail(
		ticket.Email,
		ticket.TicketCode,
		resolutionMessage,
	); err != nil {
		logger.Logger.Error("Failed sending support resolved email", "ticket_code", ticket.TicketCode, "error", err.Error())
	}
}

func (c *Controller) sendTicketUpdatedEmail(ticket *supportmodel.SupportTicket, actionResult string) {
	if ticket == nil {
		return
	}
	if strings.TrimSpace(ticket.Email) == "" {
		return
	}

	accessCode, err := auth.GenerateSecureToken(24)
	if err != nil {
		logger.Logger.Error("Failed generating support access code for update email", "ticket_code", ticket.TicketCode, "error", err.Error())
		return
	}

	if err := c.repo.UpdatePublicAccessCodeHash(ticket.ID, hashSupportAccessCode(accessCode)); err != nil {
		logger.Logger.Error("Failed updating support access code hash", "ticket_code", ticket.TicketCode, "error", err.Error())
		return
	}

	adminSummary := ""
	if ticket.AdminNotes != nil && strings.TrimSpace(*ticket.AdminNotes) != "" {
		adminSummary = strings.TrimSpace(*ticket.AdminNotes)
	}
	if strings.TrimSpace(actionResult) != "" {
		actionText := "Action applied: " + strings.TrimSpace(actionResult)
		if adminSummary == "" {
			adminSummary = actionText
		} else {
			adminSummary = adminSummary + "\n\n" + actionText
		}
	}

	if err := c.emailSvc.SendSupportTicketUpdatedEmail(
		ticket.Email,
		ticket.TicketCode,
		ticket.Status,
		adminSummary,
		accessCode,
		c.frontendURL(),
	); err != nil {
		logger.Logger.Error("Failed sending support updated email", "ticket_code", ticket.TicketCode, "error", err.Error())
	}
}

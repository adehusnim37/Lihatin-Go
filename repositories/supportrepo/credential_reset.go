package supportrepo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	apperrors "github.com/adehusnim37/lihatin-go/internal/pkg/errors"
	"github.com/adehusnim37/lihatin-go/internal/pkg/identifier"
	appvalidator "github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	supportmodel "github.com/adehusnim37/lihatin-go/models/support"
	"github.com/adehusnim37/lihatin-go/models/user"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CredentialResetApproval struct {
	ActorID, TicketID, TicketCode, Method, Reason, IPAddress string
	Approved                                                 bool
}

func resetDenied(message string) error {
	return apperrors.NewAppError("SUPPORT_RESET_DENIED", message, 403, "approval")
}

// ApproveCredentialReset validates the target from the stored ticket, never from
// client-supplied account identifiers. Row locks serialize concurrent approvals.
// Redis revocation must succeed before the database can commit the reset.
func (r *SupportTicketRepository) ApproveCredentialReset(ctx context.Context, approval CredentialResetApproval, revoke func(context.Context, string) error) (*supportmodel.CredentialReset, error) {
	reasonLength := utf8.RuneCountInString(strings.TrimSpace(approval.Reason))
	if !approval.Approved || reasonLength < 20 || reasonLength > 1000 {
		return nil, resetDenied("Explicit approval and an ownership verification reason (20–1000 characters) are required")
	}
	if !appvalidator.IsMeaningfulText(approval.Reason) {
		return nil, resetDenied("Provide a meaningful explanation of the ownership verification and approval")
	}
	if approval.Method != "totp" && approval.Method != "passkeys" && approval.Method != "all" {
		return nil, resetDenied("Choose TOTP, passkeys, or both")
	}
	var result supportmodel.CredentialReset
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var actor user.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND deleted_at IS NULL", approval.ActorID).First(&actor).Error; err != nil {
			return resetDenied("Administrator account is unavailable")
		}
		if actor.Role != "admin" && actor.Role != "super_admin" {
			return resetDenied("Administrator permission is required")
		}
		var actorAuth user.UserAuth
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ? AND deleted_at IS NULL", actor.ID).First(&actorAuth).Error; err != nil {
			return resetDenied("Administrator account is unavailable")
		}
		if !actorAuth.IsEmailVerified || actorAuth.AccountStatus != user.AccountStatusActive {
			return resetDenied("An active, verified administrator is required")
		}

		var ticket supportmodel.SupportTicket
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", approval.TicketID).First(&ticket).Error; err != nil {
			return supportTicketNotFound(err)
		}
		if ticket.Category != string(supportmodel.TicketCategoryLost2FA) || ticket.Status != string(supportmodel.TicketStatusInProgress) {
			return resetDenied("Only a Lost 2FA ticket under review (In Progress) can be approved")
		}
		if ticket.TicketCode != approval.TicketCode {
			return resetDenied("Ticket confirmation does not match")
		}
		now := time.Now().UTC()
		if ticket.RequesterVerifiedAt == nil || ticket.RequesterVerifiedAt.Before(now.Add(-24*time.Hour)) || ticket.RequesterVerifiedAt.After(now) {
			return resetDenied("The requester must verify this ticket by email OTP within the last 24 hours")
		}
		if ticket.UserID == nil || *ticket.UserID == "" || *ticket.UserID == actor.ID {
			return resetDenied("This account cannot be reset through support")
		}
		var existing int64
		if err := tx.Model(&supportmodel.CredentialReset{}).Where("ticket_id = ?", ticket.ID).Count(&existing).Error; err != nil {
			return err
		}
		if existing != 0 {
			return resetDenied("This ticket already has an approved credential reset")
		}

		var target user.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND deleted_at IS NULL", *ticket.UserID).First(&target).Error; err != nil {
			return resetDenied("Linked account is unavailable")
		}
		if target.Role != "user" || !strings.EqualFold(target.Email, ticket.Email) {
			return resetDenied("Only the linked regular user with the same email can be reset; privileged accounts are excluded")
		}
		var account user.UserAuth
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ? AND deleted_at IS NULL", target.ID).First(&account).Error; err != nil {
			return err
		}
		if !account.IsEmailVerified || account.AccountStatus != user.AccountStatusActive {
			return resetDenied("The linked account must be active and email verified")
		}
		resetTOTP := approval.Method == "totp" || approval.Method == "all"
		resetPasskeys := approval.Method == "passkeys" || approval.Method == "all"
		var totpCount, passkeyCount int64
		if err := tx.Model(&user.AuthMethod{}).Where("user_auth_id = ? AND type = ? AND deleted_at IS NULL", account.ID, user.AuthMethodTypeTOTP).Count(&totpCount).Error; err != nil {
			return err
		}
		if err := tx.Model(&user.Passkey{}).Where("user_id = ?", target.ID).Count(&passkeyCount).Error; err != nil {
			return err
		}
		if (!resetTOTP || totpCount == 0) && (!resetPasskeys || passkeyCount == 0) {
			return resetDenied("The selected credentials are not configured")
		}
		if resetPasskeys && passkeyCount > 0 && account.PasswordHash == "" {
			var alternatives int64
			if err := tx.Model(&user.AuthMethod{}).Where("user_auth_id = ? AND type = ? AND is_enabled = ? AND is_verified = ? AND deleted_at IS NULL", account.ID, user.AuthMethodTypeOAuthGoogle, true, true).Count(&alternatives).Error; err != nil {
				return err
			}
			if alternatives == 0 {
				return resetDenied("The user must have a password or verified Google sign-in before all passkeys can be removed")
			}
		}
		result = supportmodel.CredentialReset{TicketID: ticket.ID, TicketCode: ticket.TicketCode, UserID: target.ID, ApprovedBy: actor.ID, Method: approval.Method, Reason: strings.TrimSpace(approval.Reason), IPAddress: approval.IPAddress, CreatedAt: now, Email: target.Email, Username: target.Username, NextNotificationAt: now}
		if err := tx.Create(&result).Error; err != nil {
			return err
		}
		if resetTOTP {
			// Hard deletion removes encrypted secrets and every recovery code as well.
			if err := tx.Where("user_auth_id = ? AND type = ?", account.ID, user.AuthMethodTypeTOTP).Delete(&user.AuthMethod{}).Error; err != nil {
				return err
			}
		}
		if resetPasskeys {
			if err := tx.Where("user_id = ?", target.ID).Delete(&user.Passkey{}).Error; err != nil {
				return err
			}
			if err := tx.Where("user_auth_id = ? AND type = ?", account.ID, user.AuthMethodTypeFIDO2).Delete(&user.AuthMethod{}).Error; err != nil {
				return err
			}
		}
		// Invalidate outstanding password-reset links as part of recovery, too.
		if err := tx.Model(&account).Updates(map[string]any{"password_reset_token": "", "password_reset_token_expires_at": nil, "last_logout_at": now}).Error; err != nil {
			return err
		}
		if err := tx.Model(&ticket).Updates(map[string]any{"status": string(supportmodel.TicketStatusResolved), "resolved_at": now, "resolved_by": actor.ID, "public_access_code_hash": "", "updated_at": now}).Error; err != nil {
			return err
		}
		note := supportmodel.SupportMessage{ID: identifier.NewUUIDV7(), TicketID: ticket.ID, SenderType: "system", SenderUserID: &actor.ID, IsInternal: true, Body: fmt.Sprintf("Approved %s reset. Ownership verification: %s", approval.Method, result.Reason), CreatedAt: now, UpdatedAt: now}
		if err := tx.Create(&note).Error; err != nil {
			return err
		}
		if revoke == nil {
			return errors.New("session revocation is unavailable")
		}
		return revoke(ctx, target.ID)
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *SupportTicketRepository) MarkRequesterVerified(ctx context.Context, ticketID string) error {
	return r.db.WithContext(ctx).Model(&supportmodel.SupportTicket{}).Where("id = ?", ticketID).Update("requester_verified_at", time.Now().UTC()).Error
}

func (r *SupportTicketRepository) GetCredentialReset(ticketID string) (*supportmodel.CredentialReset, error) {
	var item supportmodel.CredentialReset
	err := r.db.Where("ticket_id = ?", ticketID).First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &item, err
}

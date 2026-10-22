package supportrepo

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	"github.com/adehusnim37/lihatin-go/internal/pkg/identifier"
	supportmodel "github.com/adehusnim37/lihatin-go/models/support"
	"github.com/adehusnim37/lihatin-go/models/user"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// RecoveryVerificationHash binds the database request to its latest Redis challenge.
// The challenge itself and OTP are never exposed in a ticket response.
func RecoveryVerificationHash(token string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(token)))
}

func recoveryVerificationTarget(db *gorm.DB, ticketID, actorID string) (*supportmodel.SupportTicket, *user.User, error) {
	if actorID != "" {
		var actor user.User
		if err := db.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND deleted_at IS NULL", actorID).First(&actor).Error; err != nil {
			return nil, nil, resetDenied("Administrator account is unavailable")
		}
		var account user.UserAuth
		if err := db.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ? AND deleted_at IS NULL", actorID).First(&account).Error; err != nil {
			return nil, nil, resetDenied("Administrator account is unavailable")
		}
		if (actor.Role != "admin" && actor.Role != "super_admin") || !account.IsEmailVerified || account.AccountStatus != user.AccountStatusActive {
			return nil, nil, resetDenied("An active, verified administrator is required")
		}
	}
	var ticket supportmodel.SupportTicket
	if err := db.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", ticketID).First(&ticket).Error; err != nil {
		return nil, nil, supportTicketNotFound(err)
	}
	if ticket.Category != "lost_2fa" || ticket.Status != "in_progress" {
		return nil, nil, resetDenied("Move a Lost 2FA ticket to In Progress before requesting recovery verification")
	}
	if ticket.UserID == nil || *ticket.UserID == "" || *ticket.UserID == actorID {
		return nil, nil, resetDenied("This account cannot be recovered through support")
	}
	var target user.User
	if err := db.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND deleted_at IS NULL", *ticket.UserID).First(&target).Error; err != nil {
		return nil, nil, resetDenied("Linked account is unavailable")
	}
	if target.Role != "user" || !strings.EqualFold(target.Email, ticket.Email) {
		return nil, nil, resetDenied("Only the linked regular user with the same email can be verified")
	}
	var account user.UserAuth
	if err := db.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ? AND deleted_at IS NULL", target.ID).First(&account).Error; err != nil {
		return nil, nil, err
	}
	if !account.IsEmailVerified || account.AccountStatus != user.AccountStatusActive {
		return nil, nil, resetDenied("The linked account must be active and email verified")
	}
	var resets int64
	if err := db.Model(&supportmodel.CredentialReset{}).Where("ticket_id = ?", ticket.ID).Count(&resets).Error; err != nil {
		return nil, nil, err
	}
	if resets != 0 {
		return nil, nil, resetDenied("This ticket already has an approved credential reset")
	}
	return &ticket, &target, nil
}

func (r *SupportTicketRepository) RecoveryVerificationTarget(ctx context.Context, ticketID, actorID string) (*supportmodel.SupportTicket, *user.User, error) {
	return recoveryVerificationTarget(r.db.WithContext(ctx), ticketID, actorID)
}

func (r *SupportTicketRepository) RecordRecoveryVerificationRequest(ctx context.Context, ticketID, actorID, token string, requestedAt, expiresAt time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		ticket, _, err := recoveryVerificationTarget(tx, ticketID, actorID)
		if err != nil {
			return err
		}
		if ticket.RequesterVerifiedAt != nil && ticket.RequesterVerifiedAt.After(requestedAt.Add(-24*time.Hour)) {
			return resetDenied("The requester email is already verified for this recovery")
		}
		if actorID == "" && ticket.RecoveryVerificationRequestedAt == nil {
			return resetDenied("An administrator must request recovery verification first")
		}
		if err := tx.Model(ticket).Updates(map[string]any{"recovery_verification_requested_at": requestedAt, "recovery_verification_expires_at": expiresAt, "recovery_verification_token_hash": RecoveryVerificationHash(token)}).Error; err != nil {
			return err
		}
		note := supportmodel.SupportMessage{ID: identifier.NewUUIDV7(), TicketID: ticket.ID, SenderType: "system", IsInternal: true, Body: "Recovery email verification requested. Credential reset still requires explicit admin approval.", CreatedAt: requestedAt, UpdatedAt: requestedAt}
		if actorID != "" {
			note.SenderUserID = &actorID
		}
		return tx.Create(&note).Error
	})
}

func (r *SupportTicketRepository) CancelRecoveryVerificationRequest(ctx context.Context, ticketID, token string) error {
	return r.db.WithContext(ctx).Model(&supportmodel.SupportTicket{}).Where("id = ? AND recovery_verification_token_hash = ?", ticketID, RecoveryVerificationHash(token)).Updates(map[string]any{"recovery_verification_token_hash": "", "recovery_verification_expires_at": nil}).Error
}

// ConfirmRecoveryVerification records mailbox proof only. It never approves or
// removes credentials. The caller must have authorized access to this exact ticket.
func (r *SupportTicketRepository) ConfirmRecoveryVerification(ctx context.Context, ticketID, email, token string) (*time.Time, error) {
	now := time.Now().UTC()
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		ticket, target, err := recoveryVerificationTarget(tx, ticketID, "")
		if err != nil {
			return err
		}
		if !strings.EqualFold(target.Email, email) || ticket.RecoveryVerificationTokenHash != RecoveryVerificationHash(token) || ticket.RecoveryVerificationExpiresAt == nil || !ticket.RecoveryVerificationExpiresAt.After(now) {
			return resetDenied("This recovery code expired or was replaced. Request a new code in this ticket")
		}
		if err := tx.Model(ticket).Updates(map[string]any{"requester_verified_at": now, "recovery_verification_token_hash": "", "recovery_verification_expires_at": nil}).Error; err != nil {
			return err
		}
		note := supportmodel.SupportMessage{ID: identifier.NewUUIDV7(), TicketID: ticket.ID, SenderType: "system", SenderUserID: &target.ID, IsInternal: true, Body: "Requester verified the recovery email. Credential reset still requires explicit admin approval.", CreatedAt: now, UpdatedAt: now}
		return tx.Create(&note).Error
	})
	if err != nil {
		return nil, err
	}
	return &now, nil
}

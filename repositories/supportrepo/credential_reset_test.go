package supportrepo

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	supportmodel "github.com/adehusnim37/lihatin-go/models/support"
	"github.com/adehusnim37/lihatin-go/models/user"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func resetFixture(t *testing.T) (*gorm.DB, *SupportTicketRepository, CredentialResetApproval) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	t.Cleanup(func() { sqlDB.Close() })
	if err := db.AutoMigrate(&user.User{}, &user.UserAuth{}, &user.AuthMethod{}, &user.Passkey{}, &supportmodel.SupportTicket{}, &supportmodel.SupportMessage{}, &supportmodel.CredentialReset{}); err != nil {
		t.Fatal(err)
	}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, u := range []user.User{{ID: "admin", Username: "admin", Email: "admin@example.com", Role: "admin"}, {ID: "target", Username: "target", Email: "target@example.com", Role: "user"}} {
		must(db.Create(&u).Error)
	}
	for _, ua := range []user.UserAuth{{ID: "admin-auth", UserID: "admin", IsEmailVerified: true, AccountStatus: user.AccountStatusActive}, {ID: "target-auth", UserID: "target", IsEmailVerified: true, AccountStatus: user.AccountStatusActive, PasswordHash: "existing-password", PasswordResetToken: "old-reset-link"}} {
		must(db.Create(&ua).Error)
	}
	targetID := "target"
	now := time.Now().UTC()
	must(db.Create(&supportmodel.SupportTicket{ID: "ticket", TicketCode: "LHTK-ABC123", Email: "target@example.com", Category: "lost_2fa", Status: "in_progress", UserID: &targetID, RequesterVerifiedAt: &now, PublicAccessCodeHash: "old-code"}).Error)
	must(db.Create(&user.AuthMethod{ID: "totp", UserAuthID: "target-auth", Type: user.AuthMethodTypeTOTP, IsEnabled: true, IsVerified: true, Secret: "encrypted-secret", RecoveryCodes: datatypes.JSON(`["hashed-code"]`)}).Error)
	must(db.Create(&user.Passkey{ID: "passkey", UserID: "target", CredentialHash: "hash", CredentialData: "credential"}).Error)
	return db, NewSupportTicketRepository(db), CredentialResetApproval{ActorID: "admin", TicketID: "ticket", TicketCode: "LHTK-ABC123", Method: "all", Approved: true, Reason: "Verified ownership using the approved recovery review procedure.", IPAddress: "127.0.0.1"}
}

func TestCredentialResetRequiresExplicitApprovalAndVerifiedLinkedRegularUser(t *testing.T) {
	cases := []struct {
		name   string
		change func(*gorm.DB, *CredentialResetApproval)
	}{
		{"not approved", func(_ *gorm.DB, a *CredentialResetApproval) { a.Approved = false }},
		{"no evidence", func(_ *gorm.DB, a *CredentialResetApproval) { a.Reason = "approved" }},
		{"wrong method", func(_ *gorm.DB, a *CredentialResetApproval) { a.Method = "password" }},
		{"wrong confirmation", func(_ *gorm.DB, a *CredentialResetApproval) { a.TicketCode = "another-ticket" }},
		{"ordinary actor", func(db *gorm.DB, _ *CredentialResetApproval) {
			db.Model(&user.User{}).Where("id = ?", "admin").Update("role", "user")
		}},
		{"locked actor", func(db *gorm.DB, _ *CredentialResetApproval) {
			db.Model(&user.UserAuth{}).Where("user_id = ?", "admin").Update("account_status", "locked")
		}},
		{"wrong category", func(db *gorm.DB, _ *CredentialResetApproval) {
			db.Model(&supportmodel.SupportTicket{}).Update("category", "other")
		}},
		{"unreviewed", func(db *gorm.DB, _ *CredentialResetApproval) {
			db.Model(&supportmodel.SupportTicket{}).Update("status", "open")
		}},
		{"closed", func(db *gorm.DB, _ *CredentialResetApproval) {
			db.Model(&supportmodel.SupportTicket{}).Update("status", "closed")
		}},
		{"email never verified", func(db *gorm.DB, _ *CredentialResetApproval) {
			db.Model(&supportmodel.SupportTicket{}).Update("requester_verified_at", nil)
		}},
		{"stale ownership", func(db *gorm.DB, _ *CredentialResetApproval) {
			db.Model(&supportmodel.SupportTicket{}).Update("requester_verified_at", time.Now().Add(-25*time.Hour))
		}},
		{"self reset", func(db *gorm.DB, _ *CredentialResetApproval) {
			db.Model(&supportmodel.SupportTicket{}).Update("user_id", "admin")
		}},
		{"admin target", func(db *gorm.DB, _ *CredentialResetApproval) {
			db.Model(&user.User{}).Where("id = ?", "target").Update("role", "admin")
		}},
		{"super admin target", func(db *gorm.DB, _ *CredentialResetApproval) {
			db.Model(&user.User{}).Where("id = ?", "target").Update("role", "super_admin")
		}},
		{"email changed", func(db *gorm.DB, _ *CredentialResetApproval) {
			db.Model(&user.User{}).Where("id = ?", "target").Update("email", "new@example.com")
		}},
		{"unlinked", func(db *gorm.DB, _ *CredentialResetApproval) {
			db.Model(&supportmodel.SupportTicket{}).Update("user_id", nil)
		}},
		{"locked target", func(db *gorm.DB, _ *CredentialResetApproval) {
			db.Model(&user.UserAuth{}).Where("user_id = ?", "target").Update("account_status", "locked")
		}},
		{"last sign in", func(db *gorm.DB, _ *CredentialResetApproval) {
			db.Model(&user.UserAuth{}).Where("user_id = ?", "target").Update("password_hash", "")
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db, repo, a := resetFixture(t)
			// Restrict fixture mutations to this one stored ticket.
			tc.change(db.Session(&gorm.Session{AllowGlobalUpdate: true}), &a)
			revoked := false
			_, err := repo.ApproveCredentialReset(context.Background(), a, func(context.Context, string) error { revoked = true; return nil })
			if err == nil || revoked {
				t.Fatalf("unauthorized reset: err=%v revoked=%v", err, revoked)
			}
			var credentials, resets int64
			db.Model(&user.Passkey{}).Count(&credentials)
			db.Model(&supportmodel.CredentialReset{}).Count(&resets)
			if credentials != 1 || resets != 0 {
				t.Fatalf("denial changed state: credentials=%d resets=%d", credentials, resets)
			}
		})
	}
}

func TestCredentialResetAtomicSingleUseAndMethodScoped(t *testing.T) {
	for _, method := range []string{"totp", "passkeys", "all"} {
		t.Run(method, func(t *testing.T) {
			db, repo, a := resetFixture(t)
			a.Method = method
			revoked := 0
			result, err := repo.ApproveCredentialReset(context.Background(), a, func(_ context.Context, id string) error {
				if id != "target" {
					t.Fatal("wrong account revoked")
				}
				revoked++
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			if result.Email != "target@example.com" || result.ApprovedBy != "admin" || result.NotificationSentAt != nil {
				t.Fatalf("wrong audit/outbox: %#v", result)
			}
			var totp, keys, notes int64
			db.Model(&user.AuthMethod{}).Where("type = ?", "totp").Count(&totp)
			db.Model(&user.Passkey{}).Count(&keys)
			db.Model(&supportmodel.SupportMessage{}).Where("is_internal = ?", true).Count(&notes)
			expectedTOTP, expectedKeys := int64(0), int64(0)
			if method == "totp" {
				expectedKeys = 1
			}
			if method == "passkeys" {
				expectedTOTP = 1
			}
			if totp != expectedTOTP || keys != expectedKeys || notes != 1 || revoked != 1 {
				t.Fatalf("incorrect reset: totp=%d keys=%d notes=%d revoked=%d", totp, keys, notes, revoked)
			}
			var ticket supportmodel.SupportTicket
			db.First(&ticket)
			if ticket.Status != "resolved" || ticket.ResolvedBy == nil || *ticket.ResolvedBy != "admin" || ticket.PublicAccessCodeHash != "" {
				t.Fatalf("approval not persisted: %#v", ticket)
			}
			var account user.UserAuth
			db.Where("user_id = ?", "target").First(&account)
			if account.PasswordHash != "existing-password" || account.PasswordResetToken != "" {
				t.Fatal("password/reset link state incorrect")
			}
			// Reopening a ticket never grants another reset, even after new credentials.
			db.Model(&ticket).Update("status", "in_progress")
			if _, err := repo.ApproveCredentialReset(context.Background(), a, func(context.Context, string) error { return nil }); err == nil {
				t.Fatal("approval replay accepted")
			}
		})
	}
}

func TestCredentialResetRollsBackWhenRevocationFails(t *testing.T) {
	db, repo, a := resetFixture(t)
	_, err := repo.ApproveCredentialReset(context.Background(), a, func(context.Context, string) error { return errors.New("Redis unavailable") })
	if err == nil {
		t.Fatal("reset succeeded without revocation")
	}
	var totp, keys, resets, notes int64
	db.Model(&user.AuthMethod{}).Count(&totp)
	db.Model(&user.Passkey{}).Count(&keys)
	db.Model(&supportmodel.CredentialReset{}).Count(&resets)
	db.Model(&supportmodel.SupportMessage{}).Count(&notes)
	if totp != 1 || keys != 1 || resets != 0 || notes != 0 {
		t.Fatal("failed recovery was not rolled back")
	}
}

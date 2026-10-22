package jobs

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	supportmodel "github.com/adehusnim37/lihatin-go/models/support"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type resetTestSender struct {
	calls int
	fail  bool
}

func (s *resetTestSender) SendSupportCredentialResetEmail(to, name, ticket, method string, _ time.Time) error {
	s.calls++
	if to != "verified@example.com" || ticket != "LHTK-ABC123" || method != "totp" {
		return errors.New("incorrect notification destination or content")
	}
	if s.fail {
		return errors.New("SMTP unavailable")
	}
	return nil
}
func TestRecoveryOutboxRetriesAndDoesNotResendDeliveredNotifications(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()
	if err := db.AutoMigrate(&supportmodel.CredentialReset{}); err != nil {
		t.Fatal(err)
	}
	item := supportmodel.CredentialReset{TicketID: "ticket", TicketCode: "LHTK-ABC123", UserID: "target", ApprovedBy: "admin", Method: "totp", Email: "verified@example.com", Reason: "Verified recovery", NextNotificationAt: time.Now().UTC().Add(-time.Minute)}
	if err := db.Create(&item).Error; err != nil {
		t.Fatal(err)
	}
	sender := &resetTestSender{fail: true}
	job := &SupportResetNotificationsJob{db: db, sender: sender}
	if job.Run(context.Background()) == nil {
		t.Fatal("SMTP failure must remain visible")
	}
	var pending supportmodel.CredentialReset
	db.First(&pending)
	if pending.NotificationSentAt != nil || pending.NotificationAttempts != 1 || !pending.NextNotificationAt.After(time.Now()) {
		t.Fatalf("failed mail was lost: %#v", pending)
	}
	if err := job.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if sender.calls != 1 {
		t.Fatal("retry happened before backoff")
	}
	if err := db.Model(&item).Update("next_notification_at", time.Now().UTC().Add(-time.Minute)).Error; err != nil {
		t.Fatal(err)
	}
	sender.fail = false
	if err := job.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	db.First(&pending)
	if pending.NotificationSentAt == nil || pending.NotificationAttempts != 2 || sender.calls != 2 {
		t.Fatal("successful delivery was not recorded")
	}
	if err := job.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if sender.calls != 2 {
		t.Fatal("delivered email resent")
	}
}

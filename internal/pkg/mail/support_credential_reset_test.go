package mail

import (
	"testing"
	"time"
)

func TestRecoveryNotificationDoesNotSilentlySkipUnconfiguredSMTP(t *testing.T) {
	service := &EmailService{}
	if err := service.SendSupportCredentialResetEmail("user@example.com", "user", "LHTK-ABC123", "totp", time.Now()); err == nil {
		t.Fatal("missing SMTP must keep the security notification pending")
	}
}

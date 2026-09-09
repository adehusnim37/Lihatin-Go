package mail

import (
	"strings"
	"testing"
	"time"
)

func TestSupportConfirmationTextContainsCorrectFields(t *testing.T) {
	fixedTime := time.Date(2026, time.September, 1, 12, 34, 56, 0, time.UTC)
	trackURL, err := buildPublicSupportAccessURL("https://lihat.in", "LHTK-A+B 123")
	if err != nil {
		t.Fatalf("build support URL: %v", err)
	}
	body := renderSupportConfirmationText(
		"Ade",
		"LHTK-A+B 123",
		"Bug Report",
		"secure-access-code",
		fixedTime,
		trackURL,
	)

	for _, expected := range []string{
		"Access Code: secure-access-code",
		"Submitted At: " + fixedTime.Local().Format("2006-01-02 15:04:05"),
		"https://lihat.in/support/access?ticket=LHTK-A%2BB+123",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("confirmation body missing %q:\n%s", expected, body)
		}
	}
	for _, forbidden := range []string{"%!s(MISSING)", "email=", "code="} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("confirmation body contains forbidden value %q:\n%s", forbidden, body)
		}
	}
}

func TestSupportFrontendURLRejectsUnsafeConfiguration(t *testing.T) {
	for _, raw := range []string{
		"javascript:alert(1)",
		"https://user:password@lihat.in",
		"https://lihat.in?redirect=https://evil.example",
		"//lihat.in",
	} {
		if _, err := normalizeSupportFrontendURL(raw); err == nil {
			t.Fatalf("expected unsafe frontend URL %q to be rejected", raw)
		}
	}
}

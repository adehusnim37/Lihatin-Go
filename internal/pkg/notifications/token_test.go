package notifications

import "testing"

func TestUnsubscribeTokenRoundTrip(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-at-least-thirty-two-characters")

	token, err := GenerateUnsubscribeToken("user-123", "promotional")
	if err != nil {
		t.Fatalf("GenerateUnsubscribeToken() error = %v", err)
	}

	userID, category, err := VerifyUnsubscribeToken(token)
	if err != nil {
		t.Fatalf("VerifyUnsubscribeToken() error = %v", err)
	}
	if userID != "user-123" {
		t.Fatalf("userID = %q, want user-123", userID)
	}
	if category != "promotional" {
		t.Fatalf("category = %q, want promotional", category)
	}
}

func TestUnsubscribeTokenRejectsTampering(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-at-least-thirty-two-characters")

	token, err := GenerateUnsubscribeToken("user-123", "weekly_summary")
	if err != nil {
		t.Fatalf("GenerateUnsubscribeToken() error = %v", err)
	}

	if _, _, err := VerifyUnsubscribeToken(token + "x"); err == nil {
		t.Fatal("VerifyUnsubscribeToken() accepted a tampered token")
	}
}

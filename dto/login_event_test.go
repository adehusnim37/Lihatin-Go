package dto

import (
	"testing"
	"time"

	"github.com/adehusnim37/lihatin-go/models/user"
)

func TestNewLoginEventResponse(t *testing.T) {
	t.Parallel()

	authenticatedAt := time.Date(2026, time.July, 29, 10, 30, 0, 0, time.UTC)
	event := &user.LoginEvent{
		SessionIDHash:   "must-not-be-exposed",
		Method:          user.LoginMethodTOTP,
		DeviceID:        "device-1",
		IPAddress:       "203.0.113.10",
		UserAgent:       "test-agent",
		AuthenticatedAt: authenticatedAt,
	}

	response := NewLoginEventResponse(event)
	if response == nil {
		t.Fatal("expected login event response")
	}
	if response.Method != string(user.LoginMethodTOTP) {
		t.Fatalf("unexpected method: %q", response.Method)
	}
	if response.AuthenticatedAt != authenticatedAt {
		t.Fatalf("unexpected authentication time: %v", response.AuthenticatedAt)
	}
}

func TestNewLoginEventResponseHandlesNil(t *testing.T) {
	t.Parallel()

	if response := NewLoginEventResponse(nil); response != nil {
		t.Fatalf("expected nil response, got %#v", response)
	}
}

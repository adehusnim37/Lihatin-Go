package dto

import (
	"encoding/json"
	"testing"
	"time"
)

func TestUpdateShortLinkRequestExpirationPresence(t *testing.T) {
	t.Parallel()

	future := time.Date(2027, time.January, 2, 3, 4, 5, 0, time.UTC)
	tests := []struct {
		name          string
		payload       string
		wantExpiresAt *time.Time
		wantSet       bool
	}{
		{
			name:    "field omitted leaves expiration unchanged",
			payload: `{"title":"Updated title"}`,
		},
		{
			name:    "explicit null removes expiration",
			payload: `{"expires_at":null}`,
			wantSet: true,
		},
		{
			name:          "timestamp updates expiration",
			payload:       `{"expires_at":"2027-01-02T03:04:05Z"}`,
			wantExpiresAt: &future,
			wantSet:       true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var request UpdateShortLinkRequest
			if err := json.Unmarshal([]byte(tt.payload), &request); err != nil {
				t.Fatalf("decode update request: %v", err)
			}

			if request.ExpiresAtSet != tt.wantSet {
				t.Fatalf("ExpiresAtSet = %v, want %v", request.ExpiresAtSet, tt.wantSet)
			}
			if tt.wantExpiresAt == nil {
				if request.ExpiresAt != nil {
					t.Fatalf("ExpiresAt = %v, want nil", request.ExpiresAt)
				}
				return
			}
			if request.ExpiresAt == nil || !request.ExpiresAt.Equal(*tt.wantExpiresAt) {
				t.Fatalf("ExpiresAt = %v, want %v", request.ExpiresAt, tt.wantExpiresAt)
			}
		})
	}
}

func TestUpdateShortLinkRequestRejectsInvalidExpiration(t *testing.T) {
	t.Parallel()

	var request UpdateShortLinkRequest
	if err := json.Unmarshal([]byte(`{"expires_at":"not-a-date"}`), &request); err == nil {
		t.Fatal("expected invalid expires_at to fail decoding")
	}
}

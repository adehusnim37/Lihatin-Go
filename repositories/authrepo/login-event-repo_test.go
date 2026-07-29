package authrepo

import (
	"strings"
	"testing"
)

func TestHashSessionIDIsStableAndDoesNotExposeSession(t *testing.T) {
	t.Parallel()

	sessionID := "secret-session-id"
	first := hashSessionID(sessionID)
	second := hashSessionID(sessionID)

	if first != second {
		t.Fatalf("hashSessionID must be deterministic: %q != %q", first, second)
	}
	if len(first) != 64 {
		t.Fatalf("expected SHA-256 hex length 64, got %d", len(first))
	}
	if first == sessionID || strings.Contains(first, sessionID) {
		t.Fatal("stored session hash must not expose the raw session ID")
	}
	if first == hashSessionID("another-session-id") {
		t.Fatal("different session IDs must not produce the same hash")
	}
}

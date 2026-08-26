package identifier

import (
	"testing"

	"github.com/google/uuid"
)

func TestNewUUIDV7(t *testing.T) {
	first, err := uuid.Parse(NewUUIDV7())
	if err != nil {
		t.Fatalf("parse first UUID: %v", err)
	}
	second, err := uuid.Parse(NewUUIDV7())
	if err != nil {
		t.Fatalf("parse second UUID: %v", err)
	}

	if first.Version() != 7 || second.Version() != 7 {
		t.Fatalf("versions = %d, %d; want 7, 7", first.Version(), second.Version())
	}
	if first.Variant() != uuid.RFC4122 || second.Variant() != uuid.RFC4122 {
		t.Fatalf("UUIDs must use the RFC 4122 variant")
	}
	if first == second {
		t.Fatalf("expected unique UUID v7 values, got duplicate %s", first)
	}
}

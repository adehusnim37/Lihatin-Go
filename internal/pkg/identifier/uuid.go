package identifier

import "github.com/google/uuid"

// NewUUIDV7 returns a time-ordered UUID v7 string.
//
// It intentionally preserves google/uuid's existing New/NewString behavior of
// panicking only when the operating system's cryptographic random source fails.
func NewUUIDV7() string {
	return uuid.Must(uuid.NewV7()).String()
}

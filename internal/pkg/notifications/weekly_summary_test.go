package notifications

import (
	"testing"
	"time"
)

func TestPreviousWeekRange(t *testing.T) {
	now := time.Date(2026, time.July, 29, 12, 30, 0, 0, time.FixedZone("WIB", 7*60*60))
	start, end := PreviousWeekRange(now)

	wib := time.FixedZone("WIB", 7*60*60)
	wantStart := time.Date(2026, time.July, 20, 0, 0, 0, 0, wib)
	wantEnd := time.Date(2026, time.July, 27, 0, 0, 0, 0, wib)
	if !start.Equal(wantStart) {
		t.Fatalf("start = %s, want %s", start, wantStart)
	}
	if !end.Equal(wantEnd) {
		t.Fatalf("end = %s, want %s", end, wantEnd)
	}
}

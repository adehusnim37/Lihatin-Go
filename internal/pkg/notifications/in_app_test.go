package notifications

import (
	"context"
	"errors"
	"testing"

	"github.com/adehusnim37/lihatin-go/models/user"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestInAppAnnouncementReadIsPerAccountAndIdempotent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:in-app-announcements?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&user.InAppAnnouncementRead{}); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	initial, err := PendingInAppAnnouncements(ctx, db, "first")
	if err != nil || len(initial) != 1 || initial[0].ID != APIRateLimitAnnouncementID {
		t.Fatalf("initial announcements=%+v error=%v", initial, err)
	}
	for range 2 {
		if err := MarkInAppAnnouncementRead(ctx, db, "first", APIRateLimitAnnouncementID); err != nil {
			t.Fatal(err)
		}
	}
	first, err := PendingInAppAnnouncements(ctx, db, "first")
	if err != nil || len(first) != 0 {
		t.Fatalf("read announcement still pending=%+v error=%v", first, err)
	}
	second, err := PendingInAppAnnouncements(ctx, db, "second")
	if err != nil || len(second) != 1 {
		t.Fatalf("other account announcement=%+v error=%v", second, err)
	}
	if err := MarkInAppAnnouncementRead(ctx, db, "first", "unknown"); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("unknown announcement error=%v", err)
	}
}

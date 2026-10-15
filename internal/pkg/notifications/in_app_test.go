package notifications

import (
	"context"
	"errors"
	"testing"

	"github.com/adehusnim37/lihatin-go/models/announcements"
	"github.com/adehusnim37/lihatin-go/repositories/announcementrepo"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestInAppAnnouncementReadIsPerAccountAndIdempotent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:in-app-announcements?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&announcements.InAppAnnouncement{}, &announcements.InAppAnnouncementRead{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&announcements.InAppAnnouncement{
		ID: APIRateLimitAnnouncementID, Title: "API request limits have changed",
		Body: "API limits changed", Href: "/main/api-integrations/docs", Status: "published",
	}).Error; err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	service := NewInAppAnnouncementService(announcementrepo.NewInAppRepository(db))
	initial, err := service.Pending(ctx, "first")
	if err != nil || len(initial) != 1 || initial[0].ID != APIRateLimitAnnouncementID {
		t.Fatalf("initial announcements=%+v error=%v", initial, err)
	}
	for range 2 {
		if err := service.MarkRead(ctx, "first", APIRateLimitAnnouncementID); err != nil {
			t.Fatal(err)
		}
	}
	first, err := service.Pending(ctx, "first")
	if err != nil || len(first) != 0 {
		t.Fatalf("read announcement still pending=%+v error=%v", first, err)
	}
	second, err := service.Pending(ctx, "second")
	if err != nil || len(second) != 1 {
		t.Fatalf("other account announcement=%+v error=%v", second, err)
	}
	if err := service.MarkRead(ctx, "first", "unknown"); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("unknown announcement error=%v", err)
	}
}

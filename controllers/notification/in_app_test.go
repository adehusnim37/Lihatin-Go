package notification

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/adehusnim37/lihatin-go/controllers"
	"github.com/adehusnim37/lihatin-go/models/announcements"
	"github.com/adehusnim37/lihatin-go/repositories/announcementrepo"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCreateInAppAnnouncement(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open("file:create-in-app-announcement?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&announcements.InAppAnnouncement{}); err != nil {
		t.Fatal(err)
	}

	controller := NewController(
		controllers.NewBaseControllerWithGorm(db, validator.New()),
		announcementrepo.NewInAppRepository(db),
	)

	body := `{
		"title": "Scheduled maintenance",
		"body": "The service will be unavailable briefly tonight.",
		"href": "https://status.example.com/maintenance",
		"status": "published",
		"start_at": "2026-10-15T20:00:00Z",
		"ends_at": "2026-10-16T02:00:00Z",
		"published_at": "2026-10-15T19:00:00Z"
	}`
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/auth/admin/in-app-announcements", strings.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("user_id", "admin-test")

	controller.CreateInAppAnnouncement(ctx)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	var response struct {
		Success bool `json:"success"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if !response.Success {
		t.Fatalf("response success = false, body = %s", recorder.Body.String())
	}

	var announcement announcements.InAppAnnouncement
	if err := db.Where("title = ?", "Scheduled maintenance").First(&announcement).Error; err != nil {
		t.Fatal(err)
	}
	if announcement.CreatedBy != "admin-test" {
		t.Fatalf("created_by = %q, want %q", announcement.CreatedBy, "admin-test")
	}
	if announcement.Status != "published" {
		t.Fatalf("status = %q, want %q", announcement.Status, "published")
	}
}

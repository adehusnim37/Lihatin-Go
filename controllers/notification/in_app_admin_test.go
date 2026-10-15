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

func newAnnouncementTestController(t *testing.T, name string) (*Controller, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&announcements.InAppAnnouncement{}, &announcements.InAppAnnouncementRead{}); err != nil {
		t.Fatal(err)
	}

	return NewController(
		controllers.NewBaseControllerWithGorm(db, validator.New()),
		announcementrepo.NewInAppRepository(db),
	), db
}

func serveAnnouncement(controller *Controller, method, target, body string, handler gin.HandlerFunc) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	ctx.Request = request
	ctx.Set("user_id", "admin-test")
	handler(ctx)
	return recorder
}

func TestListInAppAnnouncementsPaginatesAndFilters(t *testing.T) {
	controller, db := newAnnouncementTestController(t, "list-in-app-announcements")

	seed := []announcements.InAppAnnouncement{
		{ID: "one", Title: "First", Body: "First body", Href: "https://example.com/1", Status: "published"},
		{ID: "two", Title: "Second", Body: "Second body", Href: "https://example.com/2", Status: "draft"},
		{ID: "three", Title: "Third", Body: "Third body", Href: "https://example.com/3", Status: "published"},
	}
	for index := range seed {
		if err := db.Create(&seed[index]).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Create(&announcements.InAppAnnouncementRead{
		ID: "read-one", AnnouncementID: "one", UserID: "user-1", IsRead: true,
	}).Error; err != nil {
		t.Fatal(err)
	}

	recorder := serveAnnouncement(controller, http.MethodGet, "/v1/auth/admin/in-app-announcements", "", controller.ListInAppAnnouncements)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	var response struct {
		Data struct {
			Announcements []struct {
				ID        string `json:"id"`
				ReadCount int64  `json:"read_count"`
			} `json:"announcements"`
			Total        int64            `json:"total"`
			TotalPages   int              `json:"total_pages"`
			StatusCounts map[string]int64 `json:"status_counts"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Data.Total != 3 || response.Data.TotalPages != 1 || len(response.Data.Announcements) != 3 {
		t.Fatalf("unexpected page payload: %+v", response.Data)
	}
	if response.Data.StatusCounts["published"] != 2 || response.Data.StatusCounts["draft"] != 1 {
		t.Fatalf("status counts = %+v", response.Data.StatusCounts)
	}
	for _, announcement := range response.Data.Announcements {
		if announcement.ID == "one" && announcement.ReadCount != 1 {
			t.Fatalf("read_count for %q = %d, want 1", announcement.ID, announcement.ReadCount)
		}
	}

	recorder = serveAnnouncement(controller, http.MethodGet, "/v1/auth/admin/in-app-announcements?status=draft&page=1&limit=1", "", controller.ListInAppAnnouncements)
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Data.Total != 1 || len(response.Data.Announcements) != 1 || response.Data.Announcements[0].ID != "two" {
		t.Fatalf("filtered page payload: %+v", response.Data)
	}
}

func TestListInAppAnnouncementsRejectsUnknownStatus(t *testing.T) {
	controller, _ := newAnnouncementTestController(t, "list-in-app-announcements-invalid")

	recorder := serveAnnouncement(controller, http.MethodGet, "/v1/auth/admin/in-app-announcements?status=unknown", "", controller.ListInAppAnnouncements)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestPublishAndArchiveInAppAnnouncement(t *testing.T) {
	controller, db := newAnnouncementTestController(t, "publish-in-app-announcement")

	if err := db.Create(&announcements.InAppAnnouncement{
		ID: "draft-one", Title: "Draft", Body: "Draft body", Href: "https://example.com/draft", Status: "draft",
	}).Error; err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/auth/admin/in-app-announcements/draft-one/publish", nil)
	ctx.Params = gin.Params{{Key: "id", Value: "draft-one"}}
	ctx.Set("user_id", "admin-test")
	controller.PublishInAppAnnouncement(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("publish status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	var stored announcements.InAppAnnouncement
	if err := db.Where("id = ?", "draft-one").First(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != "published" || stored.PublishedAt == nil {
		t.Fatalf("published announcement = %+v", stored)
	}

	recorder = httptest.NewRecorder()
	ctx, _ = gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/auth/admin/in-app-announcements/draft-one/archive", nil)
	ctx.Params = gin.Params{{Key: "id", Value: "draft-one"}}
	ctx.Set("user_id", "admin-test")
	controller.ArchiveInAppAnnouncement(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("archive status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if err := db.Where("id = ?", "draft-one").First(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != "archived" {
		t.Fatalf("archived status = %q, want %q", stored.Status, "archived")
	}
}

func TestUpdateAndDeleteInAppAnnouncement(t *testing.T) {
	controller, db := newAnnouncementTestController(t, "update-in-app-announcement")

	if err := db.Create(&announcements.InAppAnnouncement{
		ID: "edit-one", Title: "Old title", Body: "Old body", Href: "https://example.com/old", Status: "draft",
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&announcements.InAppAnnouncementRead{
		ID: "read-edit-one", AnnouncementID: "edit-one", UserID: "user-1", IsRead: true,
	}).Error; err != nil {
		t.Fatal(err)
	}

	body := `{"title":"New title","body":"New body","href":"https://example.com/new"}`
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPut, "/v1/auth/admin/in-app-announcements/edit-one", strings.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Params = gin.Params{{Key: "id", Value: "edit-one"}}
	ctx.Set("user_id", "admin-test")
	controller.UpdateInAppAnnouncement(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("update status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	var stored announcements.InAppAnnouncement
	if err := db.Where("id = ?", "edit-one").First(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Title != "New title" || stored.Body != "New body" || stored.Href != "https://example.com/new" {
		t.Fatalf("updated announcement = %+v", stored)
	}

	recorder = httptest.NewRecorder()
	ctx, _ = gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodDelete, "/v1/auth/admin/in-app-announcements/edit-one", nil)
	ctx.Params = gin.Params{{Key: "id", Value: "edit-one"}}
	ctx.Set("user_id", "admin-test")
	controller.DeleteInAppAnnouncement(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("delete status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	var total int64
	if err := db.Model(&announcements.InAppAnnouncement{}).Where("id = ?", "edit-one").Count(&total).Error; err != nil {
		t.Fatal(err)
	}
	if total != 0 {
		t.Fatalf("announcement still stored after delete, total = %d", total)
	}

	var reads int64
	if err := db.Model(&announcements.InAppAnnouncementRead{}).
		Where("announcement_id = ?", "edit-one").
		Count(&reads).Error; err != nil {
		t.Fatal(err)
	}
	if reads != 0 {
		t.Fatalf("dismissal records survived the delete, total = %d", reads)
	}
}

func TestGetInAppAnnouncementNotFound(t *testing.T) {
	controller, _ := newAnnouncementTestController(t, "get-in-app-announcement-missing")

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/auth/admin/in-app-announcements/missing", nil)
	ctx.Params = gin.Params{{Key: "id", Value: "missing"}}
	ctx.Set("user_id", "admin-test")
	controller.GetInAppAnnouncement(ctx)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}

// The admin dashboard reads `read_count` on single-record responses too, so the
// detail payload must stay shaped like the paginated list.
func TestGetInAppAnnouncementIncludesReadCount(t *testing.T) {
	controller, db := newAnnouncementTestController(t, "get-in-app-announcement-read-count")

	if err := db.Create(&announcements.InAppAnnouncement{
		ID: "detail-one", Title: "Detail", Body: "Detail body", Href: "https://example.com/detail", Status: "published",
	}).Error; err != nil {
		t.Fatal(err)
	}
	for _, userID := range []string{"user-1", "user-2"} {
		if err := db.Create(&announcements.InAppAnnouncementRead{
			ID: "read-" + userID, AnnouncementID: "detail-one", UserID: userID, IsRead: true,
		}).Error; err != nil {
			t.Fatal(err)
		}
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/auth/admin/in-app-announcements/detail-one", nil)
	ctx.Params = gin.Params{{Key: "id", Value: "detail-one"}}
	ctx.Set("user_id", "admin-test")
	controller.GetInAppAnnouncement(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	var response struct {
		Data struct {
			ID        string `json:"id"`
			ReadCount int64  `json:"read_count"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Data.ID != "detail-one" || response.Data.ReadCount != 2 {
		t.Fatalf("detail payload = %+v, want id=detail-one read_count=2", response.Data)
	}

	recorder = httptest.NewRecorder()
	ctx, _ = gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/auth/admin/in-app-announcements/detail-one/archive", nil)
	ctx.Params = gin.Params{{Key: "id", Value: "detail-one"}}
	ctx.Set("user_id", "admin-test")
	controller.ArchiveInAppAnnouncement(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("archive status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

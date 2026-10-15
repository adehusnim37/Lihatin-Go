package notification

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/adehusnim37/lihatin-go/controllers"
	"github.com/adehusnim37/lihatin-go/models/announcements"
	"github.com/adehusnim37/lihatin-go/models/user"
	"github.com/adehusnim37/lihatin-go/repositories/announcementrepo"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// newInboxTestController migrates the announcement plus the user tables the
// audience queries join against.
func newInboxTestController(t *testing.T, name string) (*Controller, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&announcements.InAppAnnouncement{},
		&announcements.InAppAnnouncementRead{},
		&user.User{},
		&user.UserAuth{},
	); err != nil {
		t.Fatal(err)
	}

	return NewController(
		controllers.NewBaseControllerWithGorm(db, validator.New()),
		announcementrepo.NewInAppRepository(db),
	), db
}

func seedReader(t *testing.T, db *gorm.DB, id, username string) {
	t.Helper()
	if err := db.Create(&user.User{
		ID: id, Username: username, FirstName: "Test", LastName: "User",
		Email: username + "@example.com", Role: "user",
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&user.UserAuth{
		ID: "auth-" + id, UserID: id, IsEmailVerified: true,
		AccountStatus: user.AccountStatusActive,
	}).Error; err != nil {
		t.Fatal(err)
	}
}

func serveAsUser(controller *Controller, method, target, userID string, handler gin.HandlerFunc) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(method, target, nil)
	ctx.Set("user_id", userID)
	handler(ctx)
	return recorder
}

func TestInboxKeepsReadAnnouncementsSoUsersCanReadAgain(t *testing.T) {
	controller, db := newInboxTestController(t, "inbox-keeps-read")

	if err := db.Create(&announcements.InAppAnnouncement{
		ID: "notice-one", Title: "Maintenance", Body: "Short downtime tonight.",
		Href: "https://example.com/status", Status: "published",
	}).Error; err != nil {
		t.Fatal(err)
	}

	// Before reading: one unread item.
	recorder := serveAsUser(controller, http.MethodGet, "/v1/notifications/in-app/inbox", "user-1", controller.ListInAppAnnouncementInbox)
	if recorder.Code != http.StatusOK {
		t.Fatalf("inbox status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	var inbox struct {
		Data struct {
			Announcements []struct {
				ID     string `json:"id"`
				IsRead bool   `json:"is_read"`
			} `json:"announcements"`
			UnreadCount int64 `json:"unread_count"`
			Total       int64 `json:"total"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &inbox); err != nil {
		t.Fatal(err)
	}
	if inbox.Data.Total != 1 || inbox.Data.UnreadCount != 1 || inbox.Data.Announcements[0].IsRead {
		t.Fatalf("unread inbox = %+v", inbox.Data)
	}

	// Dismiss it, then confirm it stays listed but flipped to read.
	recorder = serveAsUser(controller, http.MethodPost, "/v1/notifications/in-app/notice-one/read", "user-1", func(ctx *gin.Context) {
		ctx.Params = gin.Params{{Key: "id", Value: "notice-one"}}
		controller.MarkInAppAnnouncementRead(ctx)
	})
	if recorder.Code != http.StatusOK {
		t.Fatalf("mark read status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	recorder = serveAsUser(controller, http.MethodGet, "/v1/notifications/in-app/inbox", "user-1", controller.ListInAppAnnouncementInbox)
	if err := json.Unmarshal(recorder.Body.Bytes(), &inbox); err != nil {
		t.Fatal(err)
	}
	if inbox.Data.Total != 1 {
		t.Fatalf("read announcement disappeared from the inbox: %+v", inbox.Data)
	}
	if !inbox.Data.Announcements[0].IsRead {
		t.Fatalf("inbox item should be marked read: %+v", inbox.Data.Announcements[0])
	}
	if inbox.Data.UnreadCount != 0 {
		t.Fatalf("unread_count = %d, want 0", inbox.Data.UnreadCount)
	}

	// The pending feed must NOT include an already-read announcement.
	recorder = serveAsUser(controller, http.MethodGet, "/v1/notifications/in-app", "user-1", controller.PendingInAppAnnouncements)
	var pending struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &pending); err != nil {
		t.Fatal(err)
	}
	if len(pending.Data) != 0 {
		t.Fatalf("pending feed should be empty, got %+v", pending.Data)
	}

	// Reading again puts it back into the pending feed.
	recorder = serveAsUser(controller, http.MethodPost, "/v1/notifications/in-app/notice-one/unread", "user-1", func(ctx *gin.Context) {
		ctx.Params = gin.Params{{Key: "id", Value: "notice-one"}}
		controller.MarkInAppAnnouncementUnread(ctx)
	})
	if recorder.Code != http.StatusOK {
		t.Fatalf("mark unread status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	recorder = serveAsUser(controller, http.MethodGet, "/v1/notifications/in-app", "user-1", controller.PendingInAppAnnouncements)
	if err := json.Unmarshal(recorder.Body.Bytes(), &pending); err != nil {
		t.Fatal(err)
	}
	if len(pending.Data) != 1 {
		t.Fatalf("announcement should be pending again, got %+v", pending.Data)
	}
}

func TestInboxIsScopedPerAccount(t *testing.T) {
	controller, db := newInboxTestController(t, "inbox-per-account")

	if err := db.Create(&announcements.InAppAnnouncement{
		ID: "notice-one", Title: "Maintenance", Body: "Short downtime tonight.",
		Href: "https://example.com/status", Status: "published",
	}).Error; err != nil {
		t.Fatal(err)
	}

	recorder := serveAsUser(controller, http.MethodPost, "/v1/notifications/in-app/read-all", "user-1", controller.MarkAllInAppAnnouncementsRead)
	if recorder.Code != http.StatusOK {
		t.Fatalf("read-all status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	var marked struct {
		Data struct {
			MarkedCount int64 `json:"marked_count"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &marked); err != nil {
		t.Fatal(err)
	}
	if marked.Data.MarkedCount != 1 {
		t.Fatalf("marked_count = %d, want 1", marked.Data.MarkedCount)
	}

	var unread struct {
		Data struct {
			UnreadCount int64 `json:"unread_count"`
		} `json:"data"`
	}

	recorder = serveAsUser(controller, http.MethodGet, "/v1/notifications/in-app/unread-count", "user-1", controller.InAppAnnouncementUnreadCount)
	if err := json.Unmarshal(recorder.Body.Bytes(), &unread); err != nil {
		t.Fatal(err)
	}
	if unread.Data.UnreadCount != 0 {
		t.Fatalf("user-1 unread_count = %d, want 0", unread.Data.UnreadCount)
	}

	recorder = serveAsUser(controller, http.MethodGet, "/v1/notifications/in-app/unread-count", "user-2", controller.InAppAnnouncementUnreadCount)
	if err := json.Unmarshal(recorder.Body.Bytes(), &unread); err != nil {
		t.Fatal(err)
	}
	if unread.Data.UnreadCount != 1 {
		t.Fatalf("user-2 unread_count = %d, want 1", unread.Data.UnreadCount)
	}
}

func TestReadAllIsIdempotent(t *testing.T) {
	controller, db := newInboxTestController(t, "read-all-idempotent")

	if err := db.Create(&announcements.InAppAnnouncement{
		ID: "notice-one", Title: "Maintenance", Body: "Short downtime tonight.",
		Href: "https://example.com/status", Status: "published",
	}).Error; err != nil {
		t.Fatal(err)
	}

	for range 2 {
		recorder := serveAsUser(controller, http.MethodPost, "/v1/notifications/in-app/read-all", "user-1", controller.MarkAllInAppAnnouncementsRead)
		if recorder.Code != http.StatusOK {
			t.Fatalf("read-all status = %d, body = %s", recorder.Code, recorder.Body.String())
		}
	}

	var rows int64
	if err := db.Model(&announcements.InAppAnnouncementRead{}).
		Where("user_id = ? AND announcement_id = ?", "user-1", "notice-one").
		Count(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if rows != 1 {
		t.Fatalf("read rows = %d, want a single upserted row", rows)
	}
}

func TestArchivedAndExpiredAnnouncementsLeaveTheInbox(t *testing.T) {
	controller, db := newInboxTestController(t, "inbox-visibility-window")

	past := time.Now().Add(-48 * time.Hour)
	if err := db.Create(&announcements.InAppAnnouncement{
		ID: "expired", Title: "Expired", Body: "This window has closed.",
		Href: "https://example.com/old", Status: "published", EndsAt: &past,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&announcements.InAppAnnouncement{
		ID: "archived", Title: "Archived", Body: "No longer shown.",
		Href: "https://example.com/archived", Status: "archived",
	}).Error; err != nil {
		t.Fatal(err)
	}

	recorder := serveAsUser(controller, http.MethodGet, "/v1/notifications/in-app/inbox", "user-1", controller.ListInAppAnnouncementInbox)

	var inbox struct {
		Data struct {
			Total       int64 `json:"total"`
			UnreadCount int64 `json:"unread_count"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &inbox); err != nil {
		t.Fatal(err)
	}
	if inbox.Data.Total != 0 || inbox.Data.UnreadCount != 0 {
		t.Fatalf("expired/archived announcements still visible: %+v", inbox.Data)
	}
}

func TestAudienceListsReadersAndAggregates(t *testing.T) {
	controller, db := newInboxTestController(t, "audience-readers")

	if err := db.Create(&announcements.InAppAnnouncement{
		ID: "notice-one", Title: "Maintenance", Body: "Short downtime tonight.",
		Href: "https://example.com/status", Status: "published",
	}).Error; err != nil {
		t.Fatal(err)
	}
	seedReader(t, db, "user-1", "alice")
	seedReader(t, db, "user-2", "bob")
	seedReader(t, db, "user-3", "carol")

	readAt := time.Now().UTC()
	if err := db.Create(&announcements.InAppAnnouncementRead{
		ID: "read-1", AnnouncementID: "notice-one", UserID: "user-1", IsRead: true, ReadAt: &readAt,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&announcements.InAppAnnouncementRead{
		ID: "read-2", AnnouncementID: "notice-one", UserID: "user-2", IsRead: false,
	}).Error; err != nil {
		t.Fatal(err)
	}

	recorder := serveAsUser(controller, http.MethodGet, "/v1/auth/admin/in-app-announcements/notice-one/audience", "admin-test", func(ctx *gin.Context) {
		ctx.Params = gin.Params{{Key: "id", Value: "notice-one"}}
		controller.ListInAppAnnouncementAudience(ctx)
	})
	if recorder.Code != http.StatusOK {
		t.Fatalf("audience status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	var response struct {
		Data struct {
			ReadCount   int64  `json:"read_count"`
			UnreadCount int64  `json:"unread_count"`
			TotalUsers  int64  `json:"total_users"`
			Total       int64  `json:"total"`
			Status      string `json:"status"`
			Readers     []struct {
				UserID   string `json:"user_id"`
				Username string `json:"username"`
				Email    string `json:"email"`
				IsRead   bool   `json:"is_read"`
			} `json:"readers"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}

	if response.Data.Status != "read" || response.Data.Total != 1 {
		t.Fatalf("read audience = %+v", response.Data)
	}
	if len(response.Data.Readers) != 1 || response.Data.Readers[0].Username != "alice" {
		t.Fatalf("readers = %+v", response.Data.Readers)
	}
	if response.Data.Readers[0].Email != "alice@example.com" {
		t.Fatalf("reader email = %q", response.Data.Readers[0].Email)
	}
	if response.Data.ReadCount != 1 || response.Data.UnreadCount != 1 || response.Data.TotalUsers != 3 {
		t.Fatalf("aggregate = %+v", response.Data)
	}

	// status=unread flips to the account with a pending record.
	recorder = serveAsUser(controller, http.MethodGet, "/v1/auth/admin/in-app-announcements/notice-one/audience?status=unread", "admin-test", func(ctx *gin.Context) {
		ctx.Params = gin.Params{{Key: "id", Value: "notice-one"}}
		controller.ListInAppAnnouncementAudience(ctx)
	})
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Data.Status != "unread" || response.Data.Total != 1 || response.Data.Readers[0].Username != "bob" {
		t.Fatalf("unread audience = %+v", response.Data)
	}
}

func TestAudienceRejectsUnknownStatusAndMissingAnnouncement(t *testing.T) {
	controller, db := newInboxTestController(t, "audience-invalid")

	if err := db.Create(&announcements.InAppAnnouncement{
		ID: "notice-one", Title: "Maintenance", Body: "Short downtime tonight.",
		Href: "https://example.com/status", Status: "published",
	}).Error; err != nil {
		t.Fatal(err)
	}

	recorder := serveAsUser(controller, http.MethodGet, "/v1/auth/admin/in-app-announcements/notice-one/audience?status=bogus", "admin-test", func(ctx *gin.Context) {
		ctx.Params = gin.Params{{Key: "id", Value: "notice-one"}}
		controller.ListInAppAnnouncementAudience(ctx)
	})
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}

	recorder = serveAsUser(controller, http.MethodGet, "/v1/auth/admin/in-app-announcements/missing/audience", "admin-test", func(ctx *gin.Context) {
		ctx.Params = gin.Params{{Key: "id", Value: "missing"}}
		controller.ListInAppAnnouncementAudience(ctx)
	})
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", recorder.Code)
	}
}

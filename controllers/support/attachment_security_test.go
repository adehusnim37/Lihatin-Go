package support

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path"
	"strings"
	"testing"

	supportmodel "github.com/adehusnim37/lihatin-go/models/support"
	"github.com/adehusnim37/lihatin-go/repositories/supportrepo"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type attachmentTestStorage struct {
	uploads, opens int
	deleted        []string
}

func (s *attachmentTestStorage) UploadAttachment(_ context.Context, _, _, contentType string, _ []byte) (string, error) {
	s.uploads++
	return fmt.Sprintf("test-key-%d", s.uploads), nil
}
func (s *attachmentTestStorage) DeleteAttachment(_ context.Context, key string) error {
	s.deleted = append(s.deleted, key)
	return nil
}
func (s *attachmentTestStorage) OpenAttachment(context.Context, string) (io.ReadCloser, string, int64, error) {
	s.opens++
	// Untrusted object metadata must not override the server's stored type.
	return io.NopCloser(strings.NewReader("fixture")), "text/html", 7, nil
}

func TestSupportUploadRejectsDisguisedNamesAndCleansPartialUploads(t *testing.T) {
	for _, tt := range []struct {
		name             string
		files            []string
		uploads, deleted int
		valid            bool
	}{
		{"disguised executable", []string{"invoice.bat"}, 0, 0, false},
		{"mismatched image", []string{"invoice.jpg"}, 0, 0, false},
		{"canonical PDF", []string{"invoice.PDF"}, 1, 0, true},
		{"partial failure", []string{"invoice.pdf", "invoice.hta"}, 1, 1, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var payload bytes.Buffer
			form := multipart.NewWriter(&payload)
			for _, name := range tt.files {
				file, err := form.CreateFormFile("attachments", name)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := file.Write([]byte("%PDF-1.4\ninert fixture")); err != nil {
					t.Fatal(err)
				}
			}
			if err := form.Close(); err != nil {
				t.Fatal(err)
			}
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Request = httptest.NewRequest(http.MethodPost, "/messages", &payload)
			ctx.Request.Header.Set("Content-Type", form.FormDataContentType())
			store := &attachmentTestStorage{}
			controller := &Controller{attachmentStore: store}
			attachments, err := controller.collectSupportAttachments(ctx, "ticket", "message")
			if (err == nil) != tt.valid || store.uploads != tt.uploads || len(store.deleted) != tt.deleted {
				t.Fatalf("error=%v uploads=%d deleted=%v", err, store.uploads, store.deleted)
			}
			if tt.valid && (len(attachments) != 1 || attachments[0].FileName != "invoice.pdf" || attachments[0].ContentType != "application/pdf") {
				t.Fatalf("invalid metadata: %+v", attachments)
			}
		})
	}
}

func TestSupportDownloadAlwaysForcesSafeAttachment(t *testing.T) {
	for _, contentType := range []string{"application/pdf", "image/jpeg", "text/html"} {
		t.Run(contentType, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodGet, "/attachments/id?disposition=inline", nil)
			controller := &Controller{attachmentStore: &attachmentTestStorage{}}
			controller.sendAttachment(ctx, &supportmodel.SupportAttachment{ID: "id", ObjectKey: "key", FileName: "invoice.bat", ContentType: contentType})
			disposition, params, err := mime.ParseMediaType(recorder.Header().Get("Content-Disposition"))
			if err != nil || disposition != "attachment" {
				t.Fatalf("unsafe disposition: %q", recorder.Header().Get("Content-Disposition"))
			}
			wantExt, wantType := ".pdf", contentType
			if contentType == "image/jpeg" {
				wantExt = ".jpg"
			}
			if contentType == "text/html" {
				wantExt, wantType = ".bin", "application/octet-stream"
			}
			if recorder.Code != http.StatusOK || path.Ext(params["filename"]) != wantExt || recorder.Header().Get("Content-Type") != wantType {
				t.Fatalf("unsafe response: %d %v", recorder.Code, recorder.Header())
			}
			if recorder.Header().Get("X-Content-Type-Options") != "nosniff" || !strings.HasPrefix(recorder.Header().Get("Content-Security-Policy"), "sandbox;") {
				t.Fatalf("missing hardening: %v", recorder.Header())
			}
		})
	}
}

func TestSupportUserCannotDownloadInternalOrOrphanedAttachments(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	connection, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = connection.Close() })
	if err := db.AutoMigrate(&supportmodel.SupportTicket{}, &supportmodel.SupportMessage{}, &supportmodel.SupportAttachment{}); err != nil {
		t.Fatal(err)
	}
	owner := "owner"
	ticket := supportmodel.SupportTicket{ID: "ticket", TicketCode: "LHTK-ABC123", Email: "person@example.com", UserID: &owner, Status: "open"}
	if err := db.Create(&ticket).Error; err != nil {
		t.Fatal(err)
	}
	for _, message := range []supportmodel.SupportMessage{
		{ID: "internal", TicketID: ticket.ID, IsInternal: true},
		{ID: "visible", TicketID: ticket.ID},
		{ID: "other-ticket", TicketID: "different-ticket"},
	} {
		if err := db.Create(&message).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, messageID := range []string{"internal", "visible", "missing", "other-ticket"} {
		attachment := supportmodel.SupportAttachment{ID: messageID, MessageID: messageID, TicketID: ticket.ID, ObjectKey: "key", FileName: "invoice.bat", ContentType: "application/pdf"}
		if err := db.Create(&attachment).Error; err != nil {
			t.Fatal(err)
		}
	}
	store := &attachmentTestStorage{}
	controller := &Controller{repo: supportrepo.NewSupportTicketRepository(db), attachmentStore: store}
	for _, tt := range []struct {
		id, user string
		status   int
	}{
		{"internal", owner, http.StatusNotFound}, {"missing", owner, http.StatusNotFound},
		{"other-ticket", owner, http.StatusNotFound}, {"visible", "intruder", http.StatusForbidden},
		{"visible", owner, http.StatusOK},
	} {
		t.Run(tt.id+"/"+tt.user, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodGet, "/attachments/"+tt.id, nil)
			ctx.Params = gin.Params{{Key: "attachmentID", Value: tt.id}}
			ctx.Set("user_id", tt.user)
			before := store.opens
			controller.DownloadUserAttachment(ctx)
			if recorder.Code != tt.status {
				t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
			}
			if tt.status != http.StatusOK && store.opens != before {
				t.Fatal("unauthorized request accessed object storage")
			}
		})
	}
	// The privileged route still permits internal attachments.
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/admin/attachments/internal", nil)
	ctx.Params = gin.Params{{Key: "attachmentID", Value: "internal"}}
	controller.DownloadAdminAttachment(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("admin download denied: %s", recorder.Body.String())
	}
}

func TestSupportResponsesRepairLegacyNamesForBlobDownloads(t *testing.T) {
	controller := &Controller{}
	response := controller.toMessageResponse(supportmodel.SupportMessage{Attachments: []supportmodel.SupportAttachment{{FileName: "invoice.bat", ContentType: "application/pdf"}}})
	if response.Attachments[0].FileName != "invoice.pdf" {
		t.Fatalf("unsafe response name: %q", response.Attachments[0].FileName)
	}
}

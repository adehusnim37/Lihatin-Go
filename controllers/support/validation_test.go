package support

import (
	"bytes"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apperrors "github.com/adehusnim37/lihatin-go/internal/pkg/errors"
	appvalidator "github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	supportmodel "github.com/adehusnim37/lihatin-go/models/support"
	"github.com/adehusnim37/lihatin-go/repositories/supportrepo"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestValidateSupportMessageBody(t *testing.T) {
	for _, tt := range []struct{ name, body, code string }{
		{"attachment only", "", ""},
		{"technical code", "ERR_CONNECTION_REFUSED", ""},
		{"normal keyboard mention", "Saya memakai keyboard qwerty", ""},
		{"repeated text", "abc abc abc abc", "SUPPORT_MESSAGE_NOT_MEANINGFUL"},
		{"non Latin repetition", "哈哈哈哈哈哈哈哈", "SUPPORT_MESSAGE_NOT_MEANINGFUL"},
		{"length checked first", strings.Repeat("a", maxSupportMessageBodyLength+1), "SUPPORT_MESSAGE_TOO_LONG"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSupportMessageBody(tt.body)
			if tt.code == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			var appErr *apperrors.AppError
			if !errors.As(err, &appErr) || appErr.Code != tt.code || appErr.Field != "body" || appErr.StatusCode != http.StatusBadRequest {
				t.Fatalf("unexpected validation error: %v", err)
			}
		})
	}
}

func TestCreateTicketRejectsSpamBeforeSideEffects(t *testing.T) {
	// A controller without repositories proves rejection happens during binding,
	// before captcha calls, database writes, or confirmation emails.
	gin.SetMode(gin.TestMode)
	engine := binding.Validator.Engine().(*validator.Validate)
	if err := appvalidator.Register(engine); err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/support/tickets", strings.NewReader(`{"email":"person@example.com","category":"bug_report","subject":"abc abc abc abc","description":"Saya tidak bisa login sejak pagi.","captcha_token":"test-captcha-token"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	(&Controller{}).CreateTicket(ctx)
	if recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), "Subjek tidak boleh") {
		t.Fatalf("unexpected response: %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestSendUserMessageRejectsSpamBeforeAttachmentsAndPersistence(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&supportmodel.SupportTicket{}, &supportmodel.SupportMessage{}, &supportmodel.SupportAttachment{}); err != nil {
		t.Fatal(err)
	}
	userID := "user-a"
	ticket := supportmodel.SupportTicket{ID: "ticket-a", TicketCode: "LHTK-ABC123", Email: "person@example.com", UserID: &userID, Status: "open"}
	if err := db.Create(&ticket).Error; err != nil {
		t.Fatal(err)
	}
	controller := &Controller{repo: supportrepo.NewSupportTicketRepository(db)}
	for _, tt := range []struct {
		name, body string
		attachment bool
		field      string
	}{
		{"spam body", "abc abc abc abc", false, "body"},
		{"spam with attachment", "qwertyuiopasdfgh", true, "body"},
		{"empty without attachment", "", false, "body"},
		// A nil attachment store produces an attachments error only after text has
		// passed validation, proving attachment-only and valid prose remain allowed.
		{"attachment only", "", true, "attachments"},
		{"valid prose with attachment", "Saya memakai keyboard qwerty", true, "attachments"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var payload bytes.Buffer
			form := multipart.NewWriter(&payload)
			if err := form.WriteField("body", tt.body); err != nil {
				t.Fatal(err)
			}
			if tt.attachment {
				file, err := form.CreateFormFile("attachments", "report.png")
				if err != nil {
					t.Fatal(err)
				}
				if _, err := file.Write([]byte("sample")); err != nil {
					t.Fatal(err)
				}
			}
			if err := form.Close(); err != nil {
				t.Fatal(err)
			}
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodPost, "/support/tickets/ticket-a/messages", &payload)
			ctx.Request.Header.Set("Content-Type", form.FormDataContentType())
			ctx.Params = gin.Params{{Key: "id", Value: ticket.ID}}
			ctx.Set("user_id", userID)
			ctx.Set("email", ticket.Email)
			controller.SendUserMessage(ctx)
			var response struct {
				Error map[string]string `json:"error"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if recorder.Code != http.StatusBadRequest || response.Error[tt.field] == "" {
				t.Fatalf("unexpected response: %d %s", recorder.Code, recorder.Body.String())
			}
			var count int64
			if err := db.Model(&supportmodel.SupportMessage{}).Count(&count).Error; err != nil {
				t.Fatal(err)
			}
			if count != 0 {
				t.Fatalf("rejected request persisted %d messages", count)
			}
		})
	}
}

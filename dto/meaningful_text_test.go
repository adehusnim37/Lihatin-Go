package dto

import (
	"errors"
	"strings"
	"testing"

	appvalidator "github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	"github.com/go-playground/validator/v10"
)

func TestUserProseMeaningfulTextBinding(t *testing.T) {
	validate, err := appvalidator.New()
	if err != nil {
		t.Fatal(err)
	}
	validate.SetTagName("binding")
	ticket := func(subject, description string) CreateSupportTicketRequest {
		return CreateSupportTicketRequest{
			Email: "person@example.com", Category: "bug_report", Subject: subject,
			Description: description, CaptchaToken: strings.Repeat("x", 20),
		}
	}
	link := func(title, description string) CreateShortLinkRequest {
		return CreateShortLinkRequest{OriginalURL: "https://example.com", Title: title, Description: description}
	}
	text := func(s string) *string { return &s }
	tests := []struct {
		name  string
		input any
		field string
	}{
		{"ticket subject", ticket("abc abc abc abc", "Saya tidak bisa login sejak pagi."), "Subject"},
		{"ticket description", ticket("Gagal masuk akun", "qwertyuiopasdfgh"), "Description"},
		{"blank ticket subject", ticket("     ", "Saya tidak bisa login sejak pagi."), "Subject"},
		{"blank ticket description", ticket("Gagal masuk akun", strings.Repeat(" ", 12)), "Description"},
		{"legitimate ticket", ticket("Keyboard qwerty bermasalah", "Saya memakai keyboard qwerty dan tidak bisa login."), ""},
		{"technical ticket", ticket("ERR_CONNECTION_REFUSED", "The API returns ERR_CONNECTION_REFUSED on the dashboard."), ""},
		{"non Latin ticket", ticket("日本語のサポート", "ログインできません。助けてください。"), ""},
		{"create title", link("abc abc abc abc", ""), "Title"},
		{"create description", link("Dokumentasi layanan", "哈哈哈哈哈哈哈哈"), "Description"},
		{"blank optional title", link("   ", ""), "Title"},
		{"blank optional description", link("", "   "), "Description"},
		{"optional fields omitted", link("", ""), ""},
		{"update title", UpdateShortLinkRequest{Title: text("qwertyuiopasdfgh")}, "Title"},
		{"update description", UpdateShortLinkRequest{Description: text("abc abc abc abc")}, "Description"},
		{"patch fields omitted", UpdateShortLinkRequest{}, ""},
		{"legitimate patch", UpdateShortLinkRequest{Title: text("Dokumentasi API"), Description: text("Cara menggunakan layanan shortlink.")}, ""},
		{"nested single link", ShortLinkRequest{Link: &CreateShortLinkRequest{OriginalURL: "https://example.com", Description: "abc abc abc abc"}}, "Description"},
		{"nested bulk link", ShortLinkRequest{IsBulky: true, Links: []CreateShortLinkRequest{link("Dokumentasi API", ""), link("", "abc abc abc abc")}}, "Description"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.Struct(tt.input)
			if tt.field == "" {
				if err != nil {
					t.Fatalf("valid prose rejected: %v", err)
				}
				return
			}
			var fields validator.ValidationErrors
			if !errors.As(err, &fields) {
				t.Fatalf("expected field error, got %v", err)
			}
			found := false
			for _, field := range fields {
				if field.Field() == tt.field && field.Tag() == "meaningful_text" {
					found = true
				}
			}
			if !found {
				t.Fatalf("expected meaningful_text on %s, got %v", tt.field, err)
			}
		})
	}
}

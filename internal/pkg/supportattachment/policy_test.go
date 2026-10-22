package supportattachment

import (
	"path"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestValidateUploadMatchesSignatureAndExtension(t *testing.T) {
	for _, tt := range []struct{ name, payload, wantType, wantName string }{
		{"invoice.PDF", "%PDF-1.4\nfixture", "application/pdf", "invoice.pdf"},
		{"photo.JPEG", "\xff\xd8\xff\xe0fixture", "image/jpeg", "photo.jpg"},
		{"image.png", "\x89PNG\r\n\x1a\nfixture", "image/png", "image.png"},
		{"image.gif", "GIF89afixture", "image/gif", "image.gif"},
		{"image.webp", "RIFF\x00\x00\x00\x00WEBPVP8 fixture", "image/webp", "image.webp"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			name, contentType, err := ValidateUpload(tt.name, []byte(tt.payload))
			if err != nil || name != tt.wantName || contentType != tt.wantType {
				t.Fatalf("got %q, %q, %v", name, contentType, err)
			}
		})
	}
}

func TestValidateUploadRejectsExecutableAndMismatchedNames(t *testing.T) {
	for _, name := range []string{"invoice.bat", "invoice.hta", "invoice.js", "invoice.html", "invoice.pdf.exe", "invoice", "invoice.pdf.", "invoice.jpg", "invoice.pdf:payload.bat", "invoice\u202efdp.pdf", "invoice\x00.pdf"} {
		t.Run(name, func(t *testing.T) {
			if _, _, err := ValidateUpload(name, []byte("%PDF-1.4\ninert fixture")); err == nil {
				t.Fatal("unsafe name accepted")
			}
		})
	}
	if _, _, err := ValidateUpload("image.png", []byte("GIF89afixture")); err == nil {
		t.Fatal("mismatched image extension accepted")
	}
	if _, _, err := ValidateUpload("invoice.pdf", []byte("ordinary text")); err == nil {
		t.Fatal("unsupported signature accepted")
	}
}

func TestDownloadNameRepairsLegacyAndLongNames(t *testing.T) {
	for _, raw := range []string{"invoice.bat", `C:\folder\invoice.hta`, "../../invoice.js", "invoice\r\n.html", "invoice\u202efdp.exe", strings.Repeat("界", 100) + ".jpeg"} {
		name := DownloadName(raw, "application/pdf")
		if path.Ext(name) != ".pdf" || !utf8.ValidString(name) || len(name) > 180 || strings.ContainsAny(name, "/\\\r\n\u202e") {
			t.Fatalf("unsafe download name: %q", name)
		}
	}
	if got := DownloadName("CON.bat", "application/pdf"); got != "attachment.pdf" {
		t.Fatalf("reserved name: %q", got)
	}
	if got := DownloadName("invoice.html", "text/html"); got != "invoice.bin" {
		t.Fatalf("unknown type: %q", got)
	}
}

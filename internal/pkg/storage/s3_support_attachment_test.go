package storage

import (
	"strings"
	"testing"
)

func TestSupportObjectKeyUsesCanonicalDetectedExtension(t *testing.T) {
	storage := &S3SupportAttachmentStorage{}
	for _, tt := range []struct{ contentType, ext string }{
		{"application/pdf", ".pdf"}, {"image/jpeg", ".jpg"}, {"image/png", ".png"}, {"image/webp", ".webp"}, {"image/gif", ".gif"},
	} {
		key, err := storage.buildObjectKey("ticket", "message", tt.contentType)
		if err != nil || !strings.HasPrefix(key, "support/ticket/message/") || !strings.HasSuffix(key, tt.ext) {
			t.Fatalf("key = %q, error = %v", key, err)
		}
		another, err := storage.buildObjectKey("ticket", "message", tt.contentType)
		if err != nil || another == key {
			t.Fatal("object keys must be unique")
		}
	}
	if _, err := storage.buildObjectKey("ticket", "message", "text/html"); err == nil {
		t.Fatal("unsupported storage type accepted")
	}
}

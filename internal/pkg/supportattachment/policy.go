// Package supportattachment defines file types and safe names for support uploads.
// Signature checks identify types; they do not scan documents for malware.
package supportattachment

import (
	"fmt"
	"net/http"
	"path"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Extension returns the canonical extension for an allowed content type.
func Extension(contentType string) (string, bool) {
	switch contentType {
	case "application/pdf":
		return ".pdf", true
	case "image/jpeg":
		return ".jpg", true
	case "image/png":
		return ".png", true
	case "image/webp":
		return ".webp", true
	case "image/gif":
		return ".gif", true
	default:
		return "", false
	}
}

// ValidateUpload requires the client extension to match the detected file type.
// The returned name uses the canonical extension, including .jpg for .jpeg.
func ValidateUpload(raw string, data []byte) (name, contentType string, err error) {
	if !utf8.ValidString(raw) || strings.ContainsAny(raw, ":") || strings.ContainsFunc(raw, func(r rune) bool {
		return unicode.IsControl(r) || unicode.Is(unicode.Cf, r)
	}) {
		return "", "", fmt.Errorf("invalid attachment filename")
	}
	base := path.Base(strings.ReplaceAll(strings.TrimSpace(raw), "\\", "/"))
	ext := strings.ToLower(path.Ext(base))
	contentType = http.DetectContentType(data)
	canonical, ok := Extension(contentType)
	if !ok {
		return "", "", fmt.Errorf("only PDF and image files (JPG, PNG, WebP, GIF) are allowed")
	}
	if ext != canonical && !(contentType == "image/jpeg" && ext == ".jpeg") {
		return "", "", fmt.Errorf("attachment extension must match its detected file type (%s)", canonical)
	}
	return DownloadName(base, contentType), contentType, nil
}

// DownloadName also repairs unsafe legacy names without renaming stored objects.
// Unknown legacy types receive a .bin extension and must be served as octet-stream.
func DownloadName(raw, contentType string) string {
	ext, ok := Extension(contentType)
	if !ok {
		ext = ".bin"
	}
	base := path.Base(strings.ReplaceAll(strings.TrimSpace(raw), "\\", "/"))
	stem := strings.TrimSuffix(base, path.Ext(base))
	stem = strings.Map(func(r rune) rune {
		if r == utf8.RuneError || unicode.IsControl(r) || unicode.Is(unicode.Cf, r) || strings.ContainsRune(`<>:"|?*`, r) {
			return '_'
		}
		return r
	}, stem)
	stem = strings.Trim(stem, " .")
	// Windows device names remain reserved even when followed by an extension.
	first, _, _ := strings.Cut(strings.ToUpper(stem), ".")
	reserved := first == "CON" || first == "PRN" || first == "AUX" || first == "NUL" ||
		(len(first) == 4 && (strings.HasPrefix(first, "COM") || strings.HasPrefix(first, "LPT")) && first[3] >= '1' && first[3] <= '9')
	if stem == "" || reserved {
		stem = "attachment"
	}
	// Preserve UTF-8 and reserve space for the server-chosen extension.
	for len(stem) > 180-len(ext) {
		_, size := utf8.DecodeLastRuneInString(stem)
		stem = stem[:len(stem)-size]
	}
	return stem + ext
}

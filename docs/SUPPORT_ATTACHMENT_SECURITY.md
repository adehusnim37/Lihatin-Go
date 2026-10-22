# Support attachment security

## Verified findings

- Upload previously accepted an allowed signature regardless of the client filename extension. A harmless PDF-header fixture named `invoice.bat` demonstrates the bypass; no executable payload was run.
- The attachment endpoint previously honored `?disposition=inline` for PDF/JPEG.
- Non-admin downloads previously checked ticket ownership/session but did not check the parent message's internal visibility. Knowing an attachment ID must not bypass this boundary.

## Enforced policy

- Client extension must match the detected type: PDF, JPG/JPEG, PNG, WebP or GIF. MIME headers supplied by the uploader are not used as evidence of type.
- Names use a server-selected canonical extension (`.jpeg` becomes `.jpg`). Control characters, Unicode format controls and colon/ADS names are rejected on upload. Download names are bounded to 180 UTF-8 bytes and avoid Windows reserved device names.
- Object keys take their extension from the allowed detected MIME type. Client filenames do not enter the storage key API.
- Downloads always use `Content-Disposition: attachment`, `nosniff`, `Cache-Control: private, no-store` and `Content-Security-Policy: sandbox; default-src 'none'`. Inline query parameters have no effect.
- Download MIME type is restricted to the allowed database type, rather than trusting object-store metadata. Unknown legacy types use `application/octet-stream` and `.bin`.
- Existing records get a safe canonical name in both download headers and conversation JSON. This also covers public clients that download a blob using the JSON filename. Existing objects do not require renaming.
- Public/user lookups require a matching non-internal parent message and ticket. Internal, orphaned and mismatched records fail closed with 404; admin downloads remain available behind admin middleware.
- Frontend attachments are downloaded, without PDF/image preview embedded in the application.

## Limits

`http.DetectContentType` examines at most 512 bytes. This policy identifies file types; it is not antivirus scanning, full document parsing, image re-encoding, or PDF content disarm/reconstruction. A file with an allowed extension and signature can still contain malicious content. Deploying malware scanning/CDR requires a separate scanner and operational policy; none is implied by these fixes.

References: [OWASP File Upload Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/File_Upload_Cheat_Sheet.html), [Go DetectContentType](https://pkg.go.dev/net/http#DetectContentType).

## Regression checks

```sh
go test ./controllers/support ./internal/pkg/supportattachment ./internal/pkg/storage ./repositories/supportrepo
go test -race ./controllers/support ./internal/pkg/supportattachment ./internal/pkg/storage
```

Tests cover disguised executable names, signature/extension mismatches, canonical names/keys, legacy filenames, forced attachment headers, partial-upload cleanup, ownership, internal visibility and orphaned records. Object storage is faked; no external bucket or malware executable is used.

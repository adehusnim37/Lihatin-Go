# Mail package

Package `mail` menyediakan satu `EmailService` untuk mengirim email
transaksional dan notifikasi aplikasi. Semua template tetap berada dalam
package yang sama agar dapat memakai renderer dan konfigurasi bersama.

## Struktur

File dikelompokkan berdasarkan tanggung jawabnya:

- `service.go`: konfigurasi dan constructor `EmailService`.
- `smtp.go`: pengiriman SMTP dan utilitas token.
- `template_layout.go`: layout HTML dan helper template.
- `verification.go`, `success_verification_email.go`: verifikasi akun.
- `password.go`, `change_password.go`, `passcode.go`: reset dan perubahan
  kredensial.
- `totp.go`, `security.go`, `passkeys.go`: notifikasi keamanan dan 2FA.
- `change_email.go`, `change_old_email.go`, `username_changed.go`:
  perubahan identitas akun.
- `create_short.go`, `bulk.go`: email terkait short link.
- `premium_*.go`: email terkait premium access.
- `support_*.go`: konfirmasi, notifikasi, dan akses tiket support.
- `email_otp.go`: kode OTP email.
- `promotional_campaign.go`, `weekly_summary.go`: email berlangganan.

Nama file menggunakan huruf kecil dan `snake_case` agar konsisten dengan
konvensi package Go dan mudah dicari berdasarkan domainnya.

## Penggunaan

```go
emailService := mail.NewEmailService()

if err := emailService.SendVerificationEmail(
    "user@example.com",
    "John Doe",
    "token123",
); err != nil {
    // Handle the delivery error.
}
```

## Konfigurasi

Konfigurasi dibaca dari environment variable berikut:

- `SMTP_HOST` (default `smtp.gmail.com`)
- `SMTP_PORT` (default `587`)
- `SMTP_USER`
- `SMTP_PASS`
- `FROM_EMAIL` (default `noreply@lihatin.com`)
- `FROM_NAME` (default `Lihatin`)
- `FRONTEND_URL` (default `http://localhost:3000`)
- `BACKEND_URL` (default `http://localhost:8080`)
- `SUPPORT_ALERT_EMAILS` untuk daftar email admin yang menerima alert tiket

Jika kredensial SMTP belum dikonfigurasi, service tidak mengirim email dan
mencetak tujuan serta subject ke output aplikasi. Ini mempertahankan perilaku
development lokal tanpa server SMTP.

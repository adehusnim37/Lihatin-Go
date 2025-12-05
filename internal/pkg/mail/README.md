# Email Service - Modular Structure

Email service telah dipecah menjadi struktur modular untuk memudahkan maintenance dan pengembangan.

## Struktur File

### `base.go`
- **Tujuan**: Konfigurasi dasar dan struktur EmailService
- **Konten**: 
  - `EmailConfig` struct
  - `EmailTemplateType` constants
  - `EmailService` struct
  - `NewEmailService()` constructor
  - `GetConfig()` method

### `sender.go`
- **Tujuan**: Utilities untuk pengiriman email dan token validation
- **Konten**:
  - `sendEmail()` - Method internal untuk SMTP
  - `IsTokenExpired()` - Utility untuk validasi token

### `verification.go`
- **Tujuan**: Email verification dan account activation
- **Konten**:
  - `SendVerificationEmail()` - Kirim email verifikasi akun baru
  - Template HTML dan text yang elegant

### `password.go`
- **Tujuan**: Password reset dan security
- **Konten**:
  - `SendPasswordResetEmail()` - Kirim email reset password
  - Template dengan security tips dan warning

### `totp.go`
- **Tujuan**: Two-Factor Authentication setup
- **Konten**:
  - `SendTOTPSetupEmail()` - Konfirmasi setup 2FA
  - Template dengan informasi security dan recovery codes

### `security.go`
- **Tujuan**: Security alerts dan login notifications
- **Konten**:
  - `SendLoginAlertEmail()` - Alert login dari device/lokasi baru
  - Template dengan detail login dan action buttons

### `shortlink.go`
- **Tujuan**: Short link management emails
- **Konten**:
  - `SendPasscodeResetEmail()` - Reset passcode untuk protected links
  - Template dengan informasi short link dan security tips

## Cara Penggunaan

```go
// Inisialisasi service
emailService := mail.NewEmailService()

// Kirim email verification
err := emailService.SendVerificationEmail("user@example.com", "John Doe", "token123")

// Kirim password reset
err := emailService.SendPasswordResetEmail("user@example.com", "John Doe", "resettoken")

// Kirim TOTP setup confirmation
err := emailService.SendTOTPSetupEmail("user@example.com", "John Doe")

// Kirim login alert
err := emailService.SendLoginAlertEmail("user@example.com", "John Doe", "192.168.1.1", "Mozilla/5.0...")

// Kirim passcode reset untuk short link
err := emailService.SendPasscodeResetEmail("user@example.com", "short123", "John Doe", "resettoken")
```

## Keuntungan Struktur Modular

1. **Maintainability**: Setiap jenis email terpisah, mudah untuk edit
2. **Readability**: Code lebih terorganisir dan mudah dibaca
3. **Scalability**: Mudah menambah jenis email baru
4. **Testing**: Bisa test setiap modul secara terpisah
5. **Collaboration**: Developer bisa bekerja pada file yang berbeda tanpa konflik

## File Dependencies

Semua file dalam package `mail` dapat mengakses:
- `EmailService` struct (dari `base.go`)
- `sendEmail()` method (dari `sender.go`)
- All configuration dan utilities

## Environment Variables

Email service menggunakan environment variables berikut:
- `SMTP_HOST` - SMTP server host
- `SMTP_PORT` - SMTP server port  
- `SMTP_USER` - SMTP username
- `SMTP_PASS` - SMTP password
- `FROM_EMAIL` - From email address
- `FROM_NAME` - From name
- `BACKEND_URL` - Backend URL untuk links

## Template Design

Semua email templates menggunakan:
- Modern responsive HTML design
- Professional gradient headers
- Mobile-friendly layouts
- Consistent branding
- Security-focused messaging
- Clear call-to-action buttons
- Alternative text versions
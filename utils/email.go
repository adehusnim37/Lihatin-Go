package utils

import (
	"fmt"
	"net/smtp"
	"time"

	"github.com/jordan-wright/email"
)

// EmailConfig holds email configuration
type EmailConfig struct {
	SMTPHost     string
	SMTPPort     string
	SMTPUsername string
	SMTPPassword string
	FromEmail    string
	FromName     string
}

// EmailTemplate types
type EmailTemplateType string

const (
	EmailTemplateVerification  EmailTemplateType = "verification"
	EmailTemplatePasswordReset EmailTemplateType = "password_reset"
	EmailTemplateTOTPSetup     EmailTemplateType = "totp_setup"
	EmailTemplateLoginAlert    EmailTemplateType = "login_alert"
)

// EmailService handles email operations
type EmailService struct {
	config EmailConfig
}

// NewEmailService creates a new email service instance
func NewEmailService() *EmailService {
	return &EmailService{
		config: EmailConfig{
			SMTPHost:     GetEnvOrDefault(EnvSMTPHost, "smtp.gmail.com"),
			SMTPPort:     GetEnvOrDefault(EnvSMTPPort, "587"),
			SMTPUsername: GetEnvOrDefault(EnvSMTPUser, ""),
			SMTPPassword: GetEnvOrDefault(EnvSMTPPass, ""),
			FromEmail:    GetEnvOrDefault(EnvFromEmail, "noreply@lihatin.com"),
			FromName:     GetEnvOrDefault(EnvFromName, "Lihatin"),
		},
	}
}

// SendVerificationEmail sends email verification email
func (es *EmailService) SendVerificationEmail(toEmail, userName, token string) error {
	subject := "Verify Your Email Address"
	verificationURL := fmt.Sprintf("http://localhost:8880/v1/auth/verify-email?token=%s", token)

	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #4CAF50; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; background-color: #f9f9f9; }
        .button { background-color: #4CAF50; color: white; padding: 12px 24px; text-decoration: none; border-radius: 4px; display: inline-block; margin: 20px 0; }
        .footer { text-align: center; padding: 20px; font-size: 12px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Welcome to Lihatin!</h1>
        </div>
        <div class="content">
            <h2>Hello %s,</h2>
            <p>Thank you for signing up! Please verify your email address by clicking the button below:</p>
            <a href="%s" class="button">Verify Email Address</a>
            <p>Or copy and paste this link in your browser:</p>
            <p><a href="%s">%s</a></p>
            <p>This verification link will expire in 24 hours.</p>
            <p>If you didn't create an account, please ignore this email.</p>
        </div>
        <div class="footer">
            <p>&copy; 2025 Lihatin. All rights reserved.</p>
        </div>
    </div>
</body>
</html>
`, userName, verificationURL, verificationURL, verificationURL)

	textBody := fmt.Sprintf(`
Hello %s,

Thank you for signing up for Lihatin! 

Please verify your email address by visiting this link:
%s

This verification link will expire in 24 hours.

If you didn't create an account, please ignore this email.

Best regards,
The Lihatin Team
`, userName, verificationURL)

	return es.sendEmail(toEmail, subject, textBody, htmlBody)
}

// SendPasswordResetEmail sends password reset email
func (es *EmailService) SendPasswordResetEmail(toEmail, userName, token string) error {
	subject := "Reset Your Password"
	resetURL := fmt.Sprintf("http://localhost:8880/v1/auth/reset-password?token=%s", token)

	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #FF5722; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; background-color: #f9f9f9; }
        .button { background-color: #FF5722; color: white; padding: 12px 24px; text-decoration: none; border-radius: 4px; display: inline-block; margin: 20px 0; }
        .footer { text-align: center; padding: 20px; font-size: 12px; color: #666; }
        .warning { background-color: #fff3cd; border: 1px solid #ffeaa7; color: #856404; padding: 10px; border-radius: 4px; margin: 10px 0; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Password Reset Request</h1>
        </div>
        <div class="content">
            <h2>Hello %s,</h2>
            <p>We received a request to reset your password. Click the button below to reset it:</p>
            <a href="%s" class="button">Reset Password</a>
            <p>Or copy and paste this link in your browser:</p>
            <p><a href="%s">%s</a></p>
            <div class="warning">
                <strong>Important:</strong> This reset link will expire in 1 hour for security reasons.
            </div>
            <p>If you didn't request a password reset, please ignore this email or contact support if you have concerns.</p>
        </div>
        <div class="footer">
            <p>&copy; 2025 Lihatin. All rights reserved.</p>
        </div>
    </div>
</body>
</html>
`, userName, resetURL, resetURL, resetURL)

	textBody := fmt.Sprintf(`
Hello %s,

We received a request to reset your password for your Lihatin account.

Please reset your password by visiting this link:
%s

This reset link will expire in 1 hour for security reasons.

If you didn't request a password reset, please ignore this email or contact support if you have concerns.

Best regards,
The Lihatin Team
`, userName, resetURL)

	return es.sendEmail(toEmail, subject, textBody, htmlBody)
}

// SendTOTPSetupEmail sends TOTP setup confirmation email
func (es *EmailService) SendTOTPSetupEmail(toEmail, userName string) error {
	subject := "Two-Factor Authentication Enabled"

	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #2196F3; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; background-color: #f9f9f9; }
        .footer { text-align: center; padding: 20px; font-size: 12px; color: #666; }
        .success { background-color: #d4edda; border: 1px solid #c3e6cb; color: #155724; padding: 10px; border-radius: 4px; margin: 10px 0; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Two-Factor Authentication Enabled</h1>
        </div>
        <div class="content">
            <h2>Hello %s,</h2>
            <div class="success">
                <strong>Success!</strong> Two-factor authentication has been successfully enabled for your account.
            </div>
            <p>Your account is now more secure with two-factor authentication. You'll need to provide a verification code from your authenticator app when logging in.</p>
            <p><strong>Important:</strong> Make sure to save your recovery codes in a safe place. You can use them to access your account if you lose access to your authenticator app.</p>
            <p>If you didn't enable two-factor authentication, please contact support immediately.</p>
        </div>
        <div class="footer">
            <p>&copy; 2025 Lihatin. All rights reserved.</p>
        </div>
    </div>
</body>
</html>
`, userName)

	textBody := fmt.Sprintf(`
Hello %s,

Two-factor authentication has been successfully enabled for your Lihatin account.

Your account is now more secure with two-factor authentication. You'll need to provide a verification code from your authenticator app when logging in.

Important: Make sure to save your recovery codes in a safe place. You can use them to access your account if you lose access to your authenticator app.

If you didn't enable two-factor authentication, please contact support immediately.

Best regards,
The Lihatin Team
`, userName)

	return es.sendEmail(toEmail, subject, textBody, htmlBody)
}

// SendLoginAlertEmail sends login alert email
func (es *EmailService) SendLoginAlertEmail(toEmail, userName, ipAddress, userAgent string) error {
	subject := "New Login to Your Account"
	loginTime := time.Now().Format("January 2, 2006 at 3:04 PM MST")
    location := GetLocationString(ipAddress)

	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #FF9800; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; background-color: #f9f9f9; }
        .footer { text-align: center; padding: 20px; font-size: 12px; color: #666; }
        .info-box { background-color: #e3f2fd; border: 1px solid #bbdefb; padding: 15px; border-radius: 4px; margin: 15px 0; }
        .warning { background-color: #fff3cd; border: 1px solid #ffeaa7; color: #856404; padding: 10px; border-radius: 4px; margin: 10px 0; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>New Login Detected</h1>
        </div>
        <div class="content">
            <h2>Hello %s,</h2>
            <p>We detected a new login to your Lihatin account:</p>
            <div class="info-box">
                <strong>Login Details:</strong><br>
                Time: %s<br>
                Location: %s<br>
                IP Address: %s<br>
                Device/Browser: %s
            </div>
            <p>If this was you, no action is needed.</p>
            <div class="warning">
                <strong>If this wasn't you:</strong> Please change your password immediately and consider enabling two-factor authentication for better security.
            </div>
        </div>
        <div class="footer">
            <p>&copy; 2025 Lihatin. All rights reserved.</p>
        </div>
    </div>
</body>
</html>
`, userName, loginTime, location, ipAddress, userAgent)

	textBody := fmt.Sprintf(`
Hello %s,

We detected a new login to your Lihatin account:

Login Details:
Time: %s
IP Address: %s
Device/Browser: %s

If this was you, no action is needed.

If this wasn't you: Please change your password immediately and consider enabling two-factor authentication for better security.

Best regards,
The Lihatin Team
`, userName, loginTime, ipAddress, userAgent)

	return es.sendEmail(toEmail, subject, textBody, htmlBody)
}

// sendEmail sends an email using SMTP
func (es *EmailService) sendEmail(to, subject, textBody, htmlBody string) error {
	if es.config.SMTPUsername == "" || es.config.SMTPPassword == "" {
		// Skip sending email if SMTP is not configured
		fmt.Printf("Email would be sent to %s with subject: %s\n", to, subject)
		return nil
	}

	e := email.NewEmail()
	e.From = fmt.Sprintf("%s <%s>", es.config.FromName, es.config.FromEmail)
	e.To = []string{to}
	e.Subject = subject
	e.Text = []byte(textBody)
	e.HTML = []byte(htmlBody)

	auth := smtp.PlainAuth("", es.config.SMTPUsername, es.config.SMTPPassword, es.config.SMTPHost)
	addr := fmt.Sprintf("%s:%s", es.config.SMTPHost, es.config.SMTPPort)

	return e.Send(addr, auth)
}

// IsTokenExpired checks if a token timestamp has expired
func IsTokenExpired(tokenTime *time.Time, expiryHours int) bool {
	if tokenTime == nil {
		return true
	}
	return time.Now().After(tokenTime.Add(time.Duration(expiryHours) * time.Hour))
}

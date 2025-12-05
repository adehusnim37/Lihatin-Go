package mail

import (
	"fmt"

	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
)

// SendChangeEmailConfirmation sends a confirmation email for email change
func (es *EmailService) SendChangeEmailConfirmation(toEmail, userName, token string) error {
	subject := "Confirm Your Email Change - Lihatin"
	confirmationURL := fmt.Sprintf(config.GetRequiredEnv(config.EnvBackendURL)+`/v1/auth/verify-email?token=%s`, token)

	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<title>Confirm Your Email Change - Lihatin</title>
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<style>
		body { font-family: 'Segoe UI', Arial, sans-serif; background: #f8f9fa; color: #2c3e50; }
		.container { max-width: 600px; margin: 40px auto; background: #fff; border-radius: 12px; box-shadow: 0 4px 20px rgba(0,0,0,0.08); }
		.header { background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: #fff; padding: 36px 30px; text-align: center; }
		.header h1 { font-size: 26px; margin-bottom: 8px; }
		.header p { font-size: 15px; opacity: 0.9; }
		.content { padding: 36px 30px; }
		.greeting { font-size: 18px; font-weight: 600; margin-bottom: 18px; }
		.message { font-size: 16px; margin-bottom: 28px; }
		.button { display: inline-block; background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: #fff; padding: 14px 28px; border-radius: 8px; font-weight: 600; text-decoration: none; font-size: 16px; margin: 18px 0; }
		.button:hover { box-shadow: 0 6px 20px rgba(102,126,234,0.2); }
		.alt-link { background: #f8f9fa; border: 1px solid #e9ecef; border-radius: 8px; padding: 18px; margin: 22px 0; font-size: 14px; }
		.security { background: #fff3cd; border-left: 4px solid #ffc107; padding: 14px; margin: 22px 0; border-radius: 4px; font-size: 14px; }
		.footer { background: #f8f9fa; padding: 28px; text-align: center; border-top: 1px solid #e9ecef; font-size: 13px; color: #6c757d; }
		.footer a { color: #667eea; margin: 0 10px; text-decoration: none; }
		.footer a:hover { text-decoration: underline; }
		@media (max-width: 600px) { .container, .header, .content, .footer { padding: 20px; } }
	</style>
</head>
<body>
	<div class="container">
		<div class="header">
			<h1>Confirm Your Email Change</h1>
			<p>Account Security Notification</p>
		</div>
		<div class="content">
			<div class="greeting">Dear %s,</div>
			<div class="message">
				You have requested to change your email address for your Lihatin account.<br>
				To confirm this change and keep your account secure, please verify your new email address by clicking the button below.
			</div>
			<div style="text-align: center;">
				<a href="%s" class="button">Confirm Email Change</a>
			</div>
			<div class="alt-link">
				<strong>Alternative method:</strong><br>
				If the button above doesn't work, copy and paste this link into your browser:<br>
				<a href="%s">%s</a>
			</div>
			<div class="security">
				<strong>Security Notice:</strong><br>
				This link will expire in 1 hour. If you did not request this change, please ignore this email or contact our support team immediately.
			</div>
		</div>
		<div class="footer">
			Thank you for keeping your account secure.<br>
			<a href="%s/support">Support</a> | <a href="%s/privacy">Privacy Policy</a> | <a href="%s/terms">Terms</a><br>
			© 2025 Lihatin. All rights reserved.
		</div>
	</div>
</body>
</html>
`, userName, confirmationURL, confirmationURL, confirmationURL,
		config.GetRequiredEnv(config.EnvBackendURL), config.GetRequiredEnv(config.EnvBackendURL), config.GetRequiredEnv(config.EnvBackendURL))

	textBody := fmt.Sprintf(`
LIHATIN - EMAIL CHANGE CONFIRMATION

Dear %s,

You have requested to change your email address for your Lihatin account.

To confirm this change and keep your account secure, please verify your new email address by visiting the following link:

%s

SECURITY NOTICE:
- This link will expire in 24 hours.
- If you did not request this change, please ignore this email or contact our support team.

For support, visit: %s/support

Thank you for keeping your account secure,
The Lihatin Team

---
© 2025 Lihatin. All rights reserved.
This is an automated message, please do not reply directly to this email.
`, userName, confirmationURL, config.GetRequiredEnv(config.EnvBackendURL))

	return es.sendEmail(toEmail, subject, textBody, htmlBody)
}

package mail

import (
	"fmt"

	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
)

// SendChangeOldEmailNotification sends a notification to the old email address when user changes their email
func (es *EmailService) SendChangeOldEmailNotification(oldEmail, newEmail, username, revokeToken string) error {
	subject := "Lihatin - Email Change Notification"

	revokeURL := fmt.Sprintf("%s/v1/auth/revoke-email-change?token=%s", config.GetEnvOrDefault(config.EnvBackendURL, "http://localhost:8080"), revokeToken)
	dashboardURL := fmt.Sprintf("%s/dashboard", config.GetEnvOrDefault(config.EnvFrontendURL, "http://localhost:3000"))

	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<title>Lihatin - Email Change Notification</title>
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<style>
		body { font-family: 'Segoe UI', Arial, sans-serif; background: #f8f9fa; color: #2c3e50; margin: 0; padding: 0; }
		.container { max-width: 600px; margin: 40px auto; background: #fff; border-radius: 12px; box-shadow: 0 4px 20px rgba(0,0,0,0.08); overflow: hidden; }
		.header { background: linear-gradient(135deg, #f7971e 0%%, #ffd200 100%%); color: #fff; padding: 36px 30px; text-align: center; }
		.header h1 { font-size: 26px; margin-bottom: 8px; }
		.header p { font-size: 15px; opacity: 0.9; margin: 0; }
		.content { padding: 36px 30px; line-height: 1.6; }
		.greeting { font-size: 18px; font-weight: 600; margin-bottom: 18px; }
		.message { font-size: 16px; margin-bottom: 28px; }
		.button-group { text-align: center; margin: 28px 0; }
		.btn { display: inline-block; padding: 14px 26px; border-radius: 8px; font-weight: 600; text-decoration: none; font-size: 16px; margin: 0 8px; }
		.btn-yes { background: linear-gradient(135deg, #43e97b 0%%, #38f9d7 100%%); color: #fff; }
		.btn-no { background: linear-gradient(135deg, #ff6a00 0%%, #ee0979 100%%); color: #fff; }
		.btn:hover { opacity: 0.92; }
		.alert-box { background: #fff3cd; border-left: 4px solid #ffc107; padding: 14px; margin: 22px 0; border-radius: 6px; font-size: 15px; }
		.alt-link { background: #f8f9fa; border: 1px solid #e9ecef; border-radius: 8px; padding: 18px; margin: 24px 0; font-size: 14px; }
		.footer { background: #f8f9fa; padding: 28px; text-align: center; border-top: 1px solid #e9ecef; font-size: 13px; color: #6c757d; }
		.footer a { color: #f7971e; margin: 0 10px; text-decoration: none; }
		.footer a:hover { text-decoration: underline; }
		@media (max-width: 600px) { .container, .header, .content, .footer { padding: 20px; } }
	</style>
</head>
<body>
	<div class="container">
		<div class="header">
			<h1>Email Change Detected</h1>
			<p>Security Notification from Lihatin</p>
		</div>
		<div class="content">
			<div class="greeting">Hi %s,</div>
			<div class="message">
				We noticed that your account email was changed from <strong>%s</strong> to <strong>%s</strong>.<br><br>
				If you made this change yourself, no further action is needed.<br>
				If you didn’t authorize this change, you can revoke it immediately using the button below.
			</div>
			<div class="button-group">
				<a href="%s" class="btn btn-no">This Wasn't Me</a>
				<a href="%s" class="btn btn-yes">This Was Me</a>
			</div>
			<div class="alert-box">
				<strong>Security Notice:</strong><br>
				The revoke link is valid for 24 hours. If you didn’t perform this action, click “This Wasn’t Me” immediately to secure your account.
			</div>
			<div class="alt-link">
				If the buttons above don’t work, copy and paste this revoke link into your browser:<br>
				<a href="%s">%s</a>
			</div>
		</div>
		<div class="footer">
			Need help? <a href="%s/support">Support</a> | <a href="%s/privacy">Privacy Policy</a> | <a href="%s/terms">Terms</a><br>
			© 2025 Lihatin. All rights reserved.
		</div>
	</div>
</body>
</html>
`, username, oldEmail, newEmail, revokeURL, dashboardURL, revokeURL, revokeURL,
		config.GetEnvOrDefault(config.EnvFrontendURL, "http://localhost:3000"), config.GetEnvOrDefault(config.EnvFrontendURL, "http://localhost:3000"), config.GetEnvOrDefault(config.EnvFrontendURL, "http://localhost:3000"))

	textBody := fmt.Sprintf(`
LIHATIN - EMAIL CHANGE NOTIFICATION

Hi %s,

We noticed that your Lihatin account email was changed from %s to %s.

If you made this change, no further action is required.

If you DID NOT authorize this change, please click the following link to revoke and secure your account immediately:
%s

This link is valid for 24 hours.

If the link doesn't work, copy and paste it into your browser.

For support, visit: %s/support

Stay secure,
The Lihatin Team

---
© 2025 Lihatin. All rights reserved.
This is an automated message, please do not reply directly to this email.
`, username, oldEmail, newEmail, revokeURL, config.GetEnvOrDefault(config.EnvFrontendURL, "http://localhost:3000"))

	return es.sendEmail(oldEmail, subject, textBody, htmlBody)
}

package mail

import (
	"fmt"

	"github.com/adehusnim37/lihatin-go/utils"
)

// SendSuccessEmailVerification sends a confirmation email after successful email verification
func (es *EmailService) SendSuccessEmailVerification(toEmail, userName string) error {
	subject := "Your Email Has Been Verified - Lihatin"
	dashboardURL := fmt.Sprintf("%s/dashboard", utils.GetRequiredEnv(utils.EnvFrontendURL))

	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<title>Email Verified Successfully - Lihatin</title>
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<style>
		body { font-family: 'Segoe UI', Arial, sans-serif; background: #f8f9fa; color: #2c3e50; margin: 0; padding: 0; }
		.container { max-width: 600px; margin: 40px auto; background: #fff; border-radius: 12px; box-shadow: 0 4px 20px rgba(0,0,0,0.08); overflow: hidden; }
		.header { background: linear-gradient(135deg, #43e97b 0%%, #38f9d7 100%%); color: #fff; padding: 36px 30px; text-align: center; }
		.header h1 { font-size: 26px; margin-bottom: 8px; }
		.header p { font-size: 15px; opacity: 0.9; margin: 0; }
		.content { padding: 36px 30px; line-height: 1.6; }
		.greeting { font-size: 18px; font-weight: 600; margin-bottom: 18px; }
		.message { font-size: 16px; margin-bottom: 28px; }
		.button { display: inline-block; background: linear-gradient(135deg, #43e97b 0%%, #38f9d7 100%%); color: #fff; padding: 14px 28px; border-radius: 8px; font-weight: 600; text-decoration: none; font-size: 16px; margin: 18px 0; }
		.button:hover { box-shadow: 0 6px 20px rgba(67,233,123,0.3); }
		.success-box { background: #e6ffed; border-left: 4px solid #43e97b; padding: 16px; margin: 22px 0; border-radius: 6px; font-size: 15px; }
		.alt-link { background: #f8f9fa; border: 1px solid #e9ecef; border-radius: 8px; padding: 18px; margin: 24px 0; font-size: 14px; }
		.footer { background: #f8f9fa; padding: 28px; text-align: center; border-top: 1px solid #e9ecef; font-size: 13px; color: #6c757d; }
		.footer a { color: #43e97b; margin: 0 10px; text-decoration: none; }
		.footer a:hover { text-decoration: underline; }
		@media (max-width: 600px) { .container, .header, .content, .footer { padding: 20px; } }
	</style>
</head>
<body>
	<div class="container">
		<div class="header">
			<h1>Email Verified Successfully</h1>
			<p>Welcome to Lihatin!</p>
		</div>
		<div class="content">
			<div class="greeting">Hi %s,</div>
			<div class="message">
				Your email address has been <strong>successfully verified</strong>.<br>
				You can now access all features of your Lihatin account securely — from managing links to viewing detailed analytics.
			</div>
			<div style="text-align: center;">
				<a href="%s" class="button">Go to Dashboard</a>
			</div>
			<div class="success-box">
				<strong>What’s Next?</strong><br>
				- Start creating and managing your short links.<br>
				- Track analytics and optimize your performance.<br>
				- Explore new features coming soon to your account.
			</div>
			<div class="alt-link">
				If the button above doesn't work, you can manually visit:<br>
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
`, userName, dashboardURL, dashboardURL, dashboardURL,
		utils.GetRequiredEnv(utils.EnvBackendURL),
		utils.GetRequiredEnv(utils.EnvBackendURL),
		utils.GetRequiredEnv(utils.EnvBackendURL))

	textBody := fmt.Sprintf(`
LIHATIN - EMAIL VERIFIED SUCCESSFULLY

Hi %s,

Your email address has been successfully verified.
You can now access all features of your Lihatin account securely.

WHAT'S NEXT?
- Start creating and managing your short links.
- Track analytics and optimize your performance.
- Explore new features coming soon to your account.

Go to your dashboard:
%s

For support, visit: %s/support

Thank you for joining Lihatin,
The Lihatin Team

---
© 2025 Lihatin. All rights reserved.
This is an automated message, please do not reply directly to this email.
`, userName, dashboardURL, utils.GetRequiredEnv(utils.EnvBackendURL))

	return es.sendEmail(toEmail, subject, textBody, htmlBody)
}

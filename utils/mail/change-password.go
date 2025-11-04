package mail

import (
	"fmt"

	"github.com/adehusnim37/lihatin-go/utils"
)

// SuccessfulPasswordChangeEmail sends email notification for successful password change
func (es *EmailService) SuccessfulPasswordChangeEmail(toEmail, userName string) error {
	subject := "Password Successfully Changed - Lihatin"
	htmlBody := fmt.Sprintf(`
	<!DOCTYPE html>
	<html lang="en">
	<head>
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<title>Password Changed</title>
		<style>
			* { margin: 0; padding: 0; box-sizing: border-box; }
			body {
				font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
				line-height: 1.6;
				color: #2c3e50;
				background-color: #f8f9fa;
			}
			.email-container {
				max-width: 600px;
				margin: 40px auto;
				background: #ffffff;
				border-radius: 12px;
				box-shadow: 0 4px 20px rgba(0,0,0,0.08);
				overflow: hidden;
			}
			.email-header {
				background: linear-gradient(135deg, #43e97b 0%%, #38f9d7 100%%);
				color: white;
				padding: 40px 30px;
				text-align: center;
			}
			.email-header h1 {
				font-size: 28px;
				font-weight: 600;
				margin-bottom: 8px;
				letter-spacing: -0.5px;
			}
			.email-header p {
				font-size: 16px;
				opacity: 0.9;
				margin: 0;
			}
			.email-content {
				padding: 40px 30px;
				background-color: #ffffff;
			}
			.greeting {
				font-size: 18px;
				font-weight: 600;
				color: #2c3e50;
				margin-bottom: 20px;
			}
			.message {
				font-size: 16px;
				color: #555;
				line-height: 1.7;
				margin-bottom: 30px;
			}
			.success-badge {
				background: #d1fae5;
				border: 2px solid #43e97b;
				border-radius: 8px;
				padding: 20px;
				text-align: center;
				margin-bottom: 30px;
			}
			.success-badge .icon {
				font-size: 40px;
				margin-bottom: 10px;
			}
			.success-badge h2 {
				color: #065f46;
				font-size: 20px;
				margin-bottom: 8px;
			}
			.success-badge p {
				color: #047857;
				font-size: 15px;
			}
			.security-tips {
				background-color: #d4edda;
				border: 1px solid #c3e6cb;
				border-radius: 8px;
				padding: 20px;
				margin: 25px 0;
				border-left: 4px solid #28a745;
			}
			.security-tips h4 {
				color: #155724;
				margin-bottom: 12px;
				font-size: 16px;
			}
			.security-tips ul {
				color: #155724;
				margin: 0;
				padding-left: 20px;
			}
			.security-tips li {
				margin: 6px 0;
				font-size: 14px;
			}
			.email-footer {
				background-color: #f8f9fa;
				padding: 30px;
				text-align: center;
				border-top: 1px solid #e9ecef;
			}
			.footer-content {
				color: #6c757d;
				font-size: 14px;
				line-height: 1.5;
			}
			.footer-links {
				margin: 15px 0;
			}
			.footer-links a {
				color: #43e97b;
				text-decoration: none;
				margin: 0 15px;
				font-size: 14px;
			}
			.footer-links a:hover {
				text-decoration: underline;
			}
			.company-info {
				font-size: 12px;
				color: #adb5bd;
				margin-top: 15px;
			}
			@media (max-width: 600px) {
				.email-container { margin: 20px; border-radius: 8px; }
				.email-header, .email-content, .email-footer { padding: 25px 20px; }
				.email-header h1 { font-size: 24px; }
			}
		</style>
	</head>
	<body>
		<div class="email-container">
			<div class="email-header">
				<h1>Password Changed Successfully</h1>
				<p>Account Security - Lihatin</p>
			</div>
			<div class="email-content">
				<div class="success-badge">
					<div class="icon">🔒</div>
					<h2>Password Updated</h2>
					<p>Your password has been changed and your account is secure.</p>
				</div>
				<div class="greeting">Dear %s,</div>
				<div class="message">
					We wanted to let you know that your password for your Lihatin account was changed successfully. If you made this change, no further action is needed.
				</div>
				<div class="message">
					<strong>Your account is safe.</strong> If you did not change your password, please contact our support team immediately or reset your password.
				</div>
				<div class="security-tips">
					<h4>🔒 Password Security Tips</h4>
					<ul>
						<li>Use a combination of uppercase and lowercase letters, numbers, and symbols</li>
						<li>Make your password at least 8 characters long</li>
						<li>Avoid using personal information like names or birthdays</li>
						<li>Consider using a password manager for better security</li>
					</ul>
				</div>
			</div>
			<div class="email-footer">
				<div class="footer-content">
					<p>Need help? Our support team is here for you.</p>
					<div class="footer-links">
						<a href="%s/support">Contact Support</a>
						<a href="%s/security">Security Center</a>
						<a href="%s/help">Help Center</a>
					</div>
					<div class="company-info">
						© 2025 Lihatin. All rights reserved.<br>
						This is an automated security message, please do not reply directly to this email.
					</div>
				</div>
			</div>
		</div>
	</body>
	</html>
	`, userName,
		utils.GetRequiredEnv(utils.EnvBackendURL),
		utils.GetRequiredEnv(utils.EnvBackendURL),
		utils.GetRequiredEnv(utils.EnvBackendURL))

	textBody := fmt.Sprintf(`
	LIHATIN - PASSWORD CHANGED SUCCESSFULLY

	Dear %s,

	Your password for your Lihatin account was changed successfully.

	If you made this change, no further action is needed.
	If you did NOT change your password, please contact our support team immediately or reset your password.

	PASSWORD SECURITY TIPS:
	- Use a combination of uppercase and lowercase letters, numbers, and symbols
	- Make your password at least 8 characters long
	- Avoid using personal information like names or birthdays
	- Consider using a password manager for better security

	Need help? Contact our support team: %s/support

	Best regards,
	The Lihatin Security Team

	---
	© 2025 Lihatin. All rights reserved.
	This is an automated security message, please do not reply directly to this email.
	`, userName, utils.GetRequiredEnv(utils.EnvBackendURL))

	return es.sendEmail(toEmail, subject, textBody, htmlBody)
}

package mail

import (
	"fmt"

	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
)

// SendSuccessRetrieveEmail sends a reassuring email to old email after successful revoke
func (es *EmailService) SendSuccessRetrieveEmail(toEmail, userName string) error {
	subject := "Your Account is Secure - Email Change Revoked Successfully"

	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Account Security Confirmation</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { 
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            line-height: 1.6; 
            color: #2c3e50;
            background-color: #f8f9fa;
        }
        .email-container { 
            max-width: 600px; 
            margin: 40px auto; 
            background: #ffffff;
            border-radius: 12px;
            box-shadow: 0 4px 20px rgba(0, 0, 0, 0.08);
            overflow: hidden;
        }
        .email-header { 
            background: linear-gradient(135deg, #10b981 0%%, #059669 100%%);
            color: white; 
            padding: 40px 30px;
            text-align: center;
        }
        .email-header .icon {
            font-size: 48px;
            margin-bottom: 15px;
        }
        .email-header h1 {
            font-size: 26px;
            font-weight: 600;
            margin-bottom: 8px;
        }
        .email-header p {
            font-size: 15px;
            opacity: 0.95;
        }
        .email-content { 
            padding: 40px 30px;
        }
        .success-badge {
            background: #d1fae5;
            border: 2px solid #10b981;
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
        .greeting {
            font-size: 18px;
            font-weight: 600;
            color: #2c3e50;
            margin-bottom: 20px;
        }
        .message {
            font-size: 16px;
            color: #555;
            line-height: 1.8;
            margin-bottom: 25px;
        }
        .info-box {
            background: #f0f9ff;
            border-left: 4px solid #3b82f6;
            padding: 18px;
            margin: 25px 0;
            border-radius: 4px;
        }
        .info-box h4 {
            color: #1e40af;
            margin-bottom: 10px;
            font-size: 15px;
        }
        .info-box p {
            color: #1e40af;
            margin: 0;
            font-size: 14px;
            line-height: 1.6;
        }
        .security-tips {
            background: #fef3c7;
            border-left: 4px solid #f59e0b;
            padding: 18px;
            margin: 25px 0;
            border-radius: 4px;
        }
        .security-tips h4 {
            color: #92400e;
            margin-bottom: 10px;
            font-size: 15px;
        }
        .security-tips ul {
            margin: 10px 0 0 20px;
            padding: 0;
            color: #92400e;
            font-size: 14px;
        }
        .security-tips li {
            margin-bottom: 8px;
        }
        .cta-section {
            text-align: center;
            margin: 30px 0;
            padding: 25px;
            background: #f8f9fa;
            border-radius: 8px;
        }
        .cta-section p {
            font-size: 15px;
            color: #6c757d;
            margin-bottom: 15px;
        }
        .button {
            display: inline-block;
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
            color: white;
            padding: 14px 28px;
            text-decoration: none;
            border-radius: 8px;
            font-weight: 600;
            font-size: 15px;
            margin: 8px;
        }
        .button:hover {
            box-shadow: 0 6px 20px rgba(102, 126, 234, 0.3);
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
            color: #667eea;
            text-decoration: none;
            margin: 0 12px;
            font-size: 14px;
        }
        .footer-links a:hover {
            text-decoration: underline;
        }
        @media (max-width: 600px) {
            .email-container { margin: 20px; }
            .email-header, .email-content, .email-footer { padding: 25px 20px; }
        }
    </style>
</head>
<body>
    <div class="email-container">
        <div class="email-header">
            <div class="icon">✓</div>
            <h1>Your Account is Secure</h1>
            <p>Email Change Successfully Revoked</p>
        </div>
        
        <div class="email-content">
            <div class="success-badge">
                <div class="icon">🛡️</div>
                <h2>Action Completed Successfully</h2>
                <p>Your email address has been restored and your account is secure</p>
            </div>

            <div class="greeting">Dear %s,</div>
            
            <div class="message">
                We're writing to confirm that the recent email change request for your Lihatin account has been 
                <strong>successfully revoked</strong>. Your account email has been restored to this address, and 
                no changes have been made to your account settings or data.
            </div>

            <div class="info-box">
                <h4>✓ What We Did:</h4>
                <p>
                    • Immediately cancelled the unauthorized email change request<br>
                    • Restored your original email address (%s)<br>
                    • Verified that no account data was compromised<br>
                    • Your account remains fully functional and secure
                </p>
            </div>

            <div class="message">
                <strong>Your account is completely safe.</strong> All your data, settings, and shortened links 
                remain intact and unchanged. You can continue using Lihatin as normal.
            </div>

            <div class="security-tips">
                <h4>🔒 Security Recommendations:</h4>
                <ul>
                    <li>Consider updating your password for added security</li>
                    <li>Review your recent account activity</li>
                    <li>Enable two-factor authentication (2FA) if you haven't already</li>
                    <li>Be cautious of phishing emails asking for account information</li>
                </ul>
            </div>

            <div class="cta-section">
                <p>Want to secure your account further?</p>
                <a href="%s/dashboard/security" class="button">Review Security Settings</a>
                <a href="%s/support" class="button" style="background: #6c757d;">Contact Support</a>
            </div>

            <div class="message" style="margin-top: 30px; padding-top: 20px; border-top: 1px solid #e9ecef;">
                If you have any questions or concerns about your account security, our support team is here to help 24/7.
            </div>
        </div>
        
        <div class="email-footer">
            <div class="footer-content">
                <p><strong>Thank you for helping us keep your account secure.</strong></p>
                <div class="footer-links">
                    <a href="%s/support">Support Center</a>
                    <a href="%s/security">Security Info</a>
                    <a href="%s/privacy">Privacy Policy</a>
                </div>
                <div style="margin-top: 20px; font-size: 12px; color: #adb5bd;">
                    © 2025 Lihatin. All rights reserved.<br>
                    This is an automated security notification. Please do not reply to this email.
                </div>
            </div>
        </div>
    </div>
</body>
</html>
`, userName, toEmail,
		config.GetEnvOrDefault(config.EnvBackendURL, "http://localhost:8080"),
		config.GetEnvOrDefault(config.EnvBackendURL, "http://localhost:8080"),
		config.GetEnvOrDefault(config.EnvBackendURL, "http://localhost:8080"),
		config.GetEnvOrDefault(config.EnvBackendURL, "http://localhost:8080"),
		config.GetEnvOrDefault(config.EnvBackendURL, "http://localhost:8080"))

	textBody := fmt.Sprintf(`
YOUR ACCOUNT IS SECURE - EMAIL CHANGE REVOKED SUCCESSFULLY

Dear %s,

✓ ACTION COMPLETED SUCCESSFULLY

We're writing to confirm that the recent email change request for your Lihatin account has been successfully revoked. 

Your account email has been restored to: %s

WHAT WE DID:
• Immediately cancelled the unauthorized email change request
• Restored your original email address
• Verified that no account data was compromised
• Your account remains fully functional and secure

YOUR ACCOUNT IS COMPLETELY SAFE. All your data, settings, and shortened links remain intact and unchanged. You can continue using Lihatin as normal.

SECURITY RECOMMENDATIONS:
• Consider updating your password for added security
• Review your recent account activity
• Enable two-factor authentication (2FA) if you haven't already
• Be cautious of phishing emails asking for account information

To review your security settings, visit: %s/dashboard/security

If you have any questions or concerns about your account security, our support team is here to help 24/7.

Support Center: %s/support

Thank you for helping us keep your account secure.

---
© 2025 Lihatin. All rights reserved.
This is an automated security notification. Please do not reply to this email.
`, userName, toEmail,
		config.GetEnvOrDefault(config.EnvBackendURL, "http://localhost:8080"),
		config.GetEnvOrDefault(config.EnvBackendURL, "http://localhost:8080"))

	return es.sendEmail(toEmail, subject, textBody, htmlBody)
}

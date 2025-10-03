package mail

import (
	"fmt"
	"time"

	"github.com/adehusnim37/lihatin-go/utils"
)

// SendLoginAlertEmail sends login alert email
func (es *EmailService) SendLoginAlertEmail(toEmail, userName, ipAddress, userAgent string) error {
	subject := "Security Alert: New Login Detected - Lihatin"
	loginTime := time.Now().Format("January 2, 2006 at 3:04 PM MST")
	location := utils.GetLocationString(ipAddress)

	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Login Alert</title>
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
            box-shadow: 0 4px 20px rgba(0, 0, 0, 0.1);
            overflow: hidden;
        }
        .email-header { 
            background: linear-gradient(135deg, #ff9800 0%%, #f57c00 100%%);
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
            margin-bottom: 25px;
        }
        .login-details {
            background-color: #e3f2fd;
            border: 1px solid #bbdefb;
            border-radius: 12px;
            padding: 25px;
            margin: 25px 0;
            border-left: 4px solid #2196f3;
        }
        .login-details h4 {
            color: #1565c0;
            margin-bottom: 15px;
            font-size: 18px;
            display: flex;
            align-items: center;
            gap: 8px;
        }
        .detail-row {
            display: flex;
            justify-content: space-between;
            margin: 10px 0;
            padding: 8px 0;
            border-bottom: 1px solid #e3f2fd;
        }
        .detail-row:last-child {
            border-bottom: none;
        }
        .detail-label {
            font-weight: 600;
            color: #1565c0;
            min-width: 120px;
        }
        .detail-value {
            color: #333;
            flex: 1;
            text-align: right;
        }
        .security-notice {
            background-color: #d4edda;
            border: 1px solid #c3e6cb;
            border-radius: 8px;
            padding: 20px;
            margin: 25px 0;
            border-left: 4px solid #28a745;
        }
        .security-notice h4 {
            color: #155724;
            margin-bottom: 12px;
            font-size: 16px;
        }
        .security-notice p {
            color: #155724;
            margin: 0;
            font-size: 14px;
        }
        .warning-notice {
            background-color: #fff3cd;
            border: 1px solid #ffeaa7;
            border-radius: 8px;
            padding: 20px;
            margin: 25px 0;
            border-left: 4px solid #ffc107;
        }
        .warning-notice h4 {
            color: #856404;
            margin-bottom: 12px;
            font-size: 16px;
            display: flex;
            align-items: center;
            gap: 8px;
        }
        .warning-notice p {
            color: #856404;
            margin: 8px 0;
            font-size: 14px;
        }
        .action-buttons {
            text-align: center;
            margin: 30px 0;
        }
        .security-button { 
            display: inline-block;
            background: linear-gradient(135deg, #ff9800 0%%, #f57c00 100%%);
            color: white !important;
            padding: 14px 28px;
            text-decoration: none;
            border-radius: 8px;
            font-weight: 600;
            font-size: 14px;
            letter-spacing: 0.5px;
            margin: 10px;
            transition: all 0.3s ease;
            box-shadow: 0 4px 15px rgba(255, 152, 0, 0.3);
        }
        .security-button:hover {
            transform: translateY(-2px);
            box-shadow: 0 6px 20px rgba(255, 152, 0, 0.4);
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
            color: #ff9800;
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
            .detail-row { flex-direction: column; text-align: left; }
            .detail-value { text-align: left; margin-top: 5px; }
            .security-button { display: block; margin: 10px 0; }
        }
    </style>
</head>
<body>
    <div class="email-container">
        <div class="email-header">
            <h1>Security Alert</h1>
            <p>New Login Activity Detected</p>
        </div>
        
        <div class="email-content">
            <div class="greeting">Dear %s,</div>
            
            <div class="message">
                We have detected a new login to your Lihatin account. As part of our security measures, 
                we notify you whenever your account is accessed from a new location or device.
            </div>
            
            <div class="login-details">
                <h4>🔍 Login Information</h4>
                <div class="detail-row">
                    <span class="detail-label">Date & Time:</span>
                    <span class="detail-value">%s</span>
                </div>
                <div class="detail-row">
                    <span class="detail-label">Location:</span>
                    <span class="detail-value">%s</span>
                </div>
                <div class="detail-row">
                    <span class="detail-label">IP Address:</span>
                    <span class="detail-value">%s</span>
                </div>
                <div class="detail-row">
                    <span class="detail-label">Device/Browser:</span>
                    <span class="detail-value">%s</span>
                </div>
            </div>
            
            <div class="security-notice">
                <h4>✅ Was This You?</h4>
                <p>If you recognize this login activity, no further action is required. Your account remains secure.</p>
            </div>
            
            <div class="warning-notice">
                <h4>⚠️ Suspicious Activity?</h4>
                <p><strong>If this login was not authorized by you:</strong></p>
                <p>• Change your password immediately</p>
                <p>• Enable two-factor authentication for enhanced security</p>
                <p>• Review your recent account activity</p>
                <p>• Contact our security team if you notice any unauthorized changes</p>
            </div>
            
            <div class="action-buttons">
                <a href="%s/security" class="security-button">Review Security Settings</a>
                <a href="%s/support" class="security-button">Contact Support</a>
            </div>
            
            <p style="margin-top: 25px; color: #666; font-size: 14px; text-align: center;">
                <strong>Remember:</strong> Lihatin will never ask for your password via email. 
                If you receive suspicious emails, please report them to our security team.
            </p>
        </div>
        
        <div class="email-footer">
            <div class="footer-content">
                <p>Your account security is our priority.</p>
                <div class="footer-links">
                    <a href="%s/security">Security Center</a>
                    <a href="%s/help/account-security">Security Help</a>
                    <a href="%s/support">Contact Support</a>
                </div>
                <div class="company-info">
                    © 2025 Lihatin. All rights reserved.<br>
                    This is an automated security notification.
                </div>
            </div>
        </div>
    </div>
</body>
</html>
`, userName, loginTime, location, ipAddress, userAgent,
		utils.GetRequiredEnv(utils.EnvBackendURL), utils.GetRequiredEnv(utils.EnvBackendURL),
		utils.GetRequiredEnv(utils.EnvBackendURL), utils.GetRequiredEnv(utils.EnvBackendURL), utils.GetRequiredEnv(utils.EnvBackendURL))

	textBody := fmt.Sprintf(`
LIHATIN - SECURITY ALERT: NEW LOGIN DETECTED

Dear %s,

We have detected a new login to your Lihatin account.

LOGIN INFORMATION:
Date & Time: %s
Location: %s
IP Address: %s
Device/Browser: %s

WAS THIS YOU?
If you recognize this login activity, no further action is required.

SUSPICIOUS ACTIVITY?
If this login was not authorized by you:
- Change your password immediately
- Enable two-factor authentication for enhanced security
- Review your recent account activity
- Contact our security team if you notice any unauthorized changes

Security Settings: %s/security
Contact Support: %s/support

REMEMBER: Lihatin will never ask for your password via email.

Best regards,
The Lihatin Security Team

---
© 2025 Lihatin. All rights reserved.
This is an automated security notification.
`, userName, loginTime, location, ipAddress, userAgent,
		utils.GetRequiredEnv(utils.EnvBackendURL), utils.GetRequiredEnv(utils.EnvBackendURL))

	return es.sendEmail(toEmail, subject, textBody, htmlBody)
}

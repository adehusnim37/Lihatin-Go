package mail

import (
	"fmt"

	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
)

// SendInformationShortCreate sends email notification when a new short link is created
func (es *EmailService) SendInformationShortCreate(toEmail, toName, url, title, expires_at, created_at, passcode, urlOrigin, shortCode string) error {
	subject := fmt.Sprintf("Short Link Created: %s - Lihatin", title)

	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Short Link Created</title>
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
            background: linear-gradient(135deg, #4CAF50 0%%, #45a049 100%%);
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
        .link-details {
            background-color: #f8f9fa;
            border: 1px solid #e9ecef;
            border-radius: 12px;
            padding: 25px;
            margin: 25px 0;
            border-left: 4px solid #4CAF50;
        }
        .link-details h3 {
            color: #2c3e50;
            margin-bottom: 20px;
            font-size: 18px;
            display: flex;
            align-items: center;
            gap: 8px;
        }
        .detail-row {
            display: flex;
            justify-content: space-between;
            margin: 12px 0;
            padding: 10px 0;
            border-bottom: 1px solid #e9ecef;
        }
        .detail-row:last-child {
            border-bottom: none;
        }
        .detail-label {
            font-weight: 600;
            color: #495057;
            min-width: 120px;
        }
        .detail-value {
            color: #2c3e50;
            flex: 1;
            text-align: right;
            word-break: break-all;
        }
        .short-link {
            background-color: #ffffff;
            border: 2px solid #4CAF50;
            border-radius: 8px;
            padding: 15px;
            font-family: 'Courier New', monospace;
            font-size: 16px;
            font-weight: bold;
            color: #4CAF50;
            text-decoration: none;
            display: block;
            text-align: center;
            margin: 15px 0;
            transition: all 0.3s ease;
        }
        .short-link:hover {
            background-color: #f8f9fa;
            transform: translateY(-1px);
        }
        .passcode-section {
            background-color: #fff3cd;
            border: 1px solid #ffeaa7;
            border-radius: 8px;
            padding: 20px;
            margin: 25px 0;
            border-left: 4px solid #ffc107;
        }
        .passcode-section h4 {
            color: #856404;
            margin-bottom: 12px;
            font-size: 16px;
            display: flex;
            align-items: center;
            gap: 8px;
        }
        .passcode-display {
            background-color: #ffffff;
            border: 2px solid #ffc107;
            border-radius: 6px;
            padding: 12px;
            font-family: 'Courier New', monospace;
            font-size: 18px;
            font-weight: bold;
            color: #856404;
            text-align: center;
            margin: 10px 0;
            letter-spacing: 2px;
        }
        .security-notice {
            background-color: #e3f2fd;
            border: 1px solid #bbdefb;
            border-radius: 8px;
            padding: 20px;
            margin: 25px 0;
            border-left: 4px solid #2196f3;
        }
        .security-notice h4 {
            color: #1565c0;
            margin-bottom: 12px;
            font-size: 16px;
        }
        .security-notice ul {
            color: #1565c0;
            margin: 0;
            padding-left: 20px;
        }
        .security-notice li {
            margin: 8px 0;
            font-size: 14px;
        }
        .action-buttons {
            text-align: center;
            margin: 30px 0;
        }
        .action-button { 
            display: inline-block;
            background: linear-gradient(135deg, #4CAF50 0%%, #45a049 100%%);
            color: white !important;
            padding: 14px 28px;
            text-decoration: none;
            border-radius: 8px;
            font-weight: 600;
            font-size: 14px;
            letter-spacing: 0.5px;
            margin: 10px;
            transition: all 0.3s ease;
            box-shadow: 0 4px 15px rgba(76, 175, 80, 0.3);
        }
        .action-button:hover {
            transform: translateY(-2px);
            box-shadow: 0 6px 20px rgba(76, 175, 80, 0.4);
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
            color: #4CAF50;
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
            .action-button { display: block; margin: 10px 0; }
        }
    </style>
</head>
<body>
    <div class="email-container">
        <div class="email-header">
            <h1>Short Link Created Successfully</h1>
            <p>Your URL is now ready to share</p>
        </div>
        
        <div class="email-content">
            <div class="greeting">Dear %s,</div>
            
            <div class="message">
                Great news! Your short link has been created successfully and is ready to use. 
                Below you'll find all the important details about your new link.
            </div>
            
            <div class="link-details">
                <h3>🔗 Link Information</h3>
                <div class="detail-row">
                    <span class="detail-label">Title:</span>
                    <span class="detail-value">%s</span>
                </div>
                <div class="detail-row">
                    <span class="detail-label">Original URL:</span>
                    <span class="detail-value">%s</span>
                </div>
                <div class="detail-row">
                    <span class="detail-label">Short Code:</span>
                    <span class="detail-value">%s</span>
                </div>
                <div class="detail-row">
                    <span class="detail-label">Created:</span>
                    <span class="detail-value">%s</span>
                </div>
                <div class="detail-row">
                    <span class="detail-label">Expires:</span>
                    <span class="detail-value">%s</span>
                </div>
            </div>
            
            <div style="text-align: center; margin: 25px 0;">
                <p style="margin-bottom: 10px; color: #666; font-size: 14px;">Your Short Link:</p>
                <a href="%s" class="short-link">%s</a>
            </div>
            
            %s
            
            <div class="security-notice">
                <h4>🔒 Security & Usage Tips</h4>
                <ul>
                    <li>Keep your link details safe and secure</li>
                    <li>Only share your link with trusted recipients</li>
                    <li>Monitor your link analytics from your dashboard</li>
                    <li>Contact support if you notice any suspicious activity</li>
                    <li>Remember your link's expiration date to avoid interruptions</li>
                </ul>
            </div>
            
            <div class="action-buttons">
                <a href="%s/dashboard" class="action-button">View Dashboard</a>
                <a href="%s/links/manage" class="action-button">Manage Links</a>
            </div>
            
            <p style="margin-top: 25px; color: #666; font-size: 14px; text-align: center;">
                <strong>Need help?</strong> If you have any questions about your short link or need assistance, 
                our support team is ready to help you.
            </p>
        </div>
        
        <div class="email-footer">
            <div class="footer-content">
                <p>Thank you for using Lihatin - Professional URL Shortening</p>
                <div class="footer-links">
                    <a href="%s/help">Help Center</a>
                    <a href="%s/dashboard">Dashboard</a>
                    <a href="%s/support">Contact Support</a>
                </div>
                <div class="company-info">
                    © 2025 Lihatin. All rights reserved.<br>
                    This is an automated notification for your account activity.
                </div>
            </div>
        </div>
    </div>
</body>
</html>
`, toName, title, urlOrigin ,shortCode, created_at, expires_at, url, url,
		getPasscodeSection(passcode),
		config.GetRequiredEnv(config.EnvBackendURL), config.GetRequiredEnv(config.EnvBackendURL),
		config.GetRequiredEnv(config.EnvBackendURL), config.GetRequiredEnv(config.EnvBackendURL), config.GetRequiredEnv(config.EnvBackendURL))

	textBody := fmt.Sprintf(`
LIHATIN - SHORT LINK CREATED SUCCESSFULLY

Dear %s,

Great news! Your short link has been created successfully and is ready to use.

LINK INFORMATION:
Title: %s
Short Code: %s
Short Link: %s
Created: %s
Expires: %s
%s

SECURITY & USAGE TIPS:
- Keep your link details safe and secure
- Only share your link with trusted recipients
- Monitor your link analytics from your dashboard
- Contact support if you notice any suspicious activity
- Remember your link's expiration date to avoid interruptions

Dashboard: %s/dashboard
Manage Links: %s/links/manage
Support: %s/support

Thank you for using Lihatin - Professional URL Shortening

Best regards,
The Lihatin Team

---
© 2025 Lihatin. All rights reserved.
This is an automated notification for your account activity.
`, toName, title, shortCode, url, created_at, expires_at,
		getPasscodeTextSection(passcode),
		config.GetRequiredEnv(config.EnvBackendURL), config.GetRequiredEnv(config.EnvBackendURL), config.GetRequiredEnv(config.EnvBackendURL))

	return es.sendEmail(toEmail, subject, textBody, htmlBody)
}

// getPasscodeSection returns HTML for passcode section if passcode exists
func getPasscodeSection(passcode string) string {
	if passcode == "" || passcode == "-" {
		return ""
	}

	return fmt.Sprintf(`
            <div class="passcode-section">
                <h4>🔐 Protected Link Passcode</h4>
                <p style="color: #856404; margin: 8px 0; font-size: 14px;">
                    This link is protected with a passcode. Share this passcode only with authorized users:
                </p>
                <div class="passcode-display">%s</div>
                <p style="color: #856404; margin: 8px 0; font-size: 12px;">
                    <strong>Important:</strong> Keep this passcode secure. Anyone with this passcode can access your protected link.
                </p>
            </div>`, passcode)
}

// getPasscodeTextSection returns text version for passcode section if passcode exists
func getPasscodeTextSection(passcode string) string {
	if passcode == "" || passcode == "-" {
		return ""
	}

	return fmt.Sprintf(`
PROTECTED LINK PASSCODE: %s
Important: Keep this passcode secure. Anyone with this passcode can access your protected link.`, passcode)
}

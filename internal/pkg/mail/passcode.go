package mail

import (
	"fmt"

	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
)

// SendPasscodeResetEmail sends passcode reset email
func (es *EmailService) SendPasscodeResetEmail(toEmail, short, userName, token string) error {
	subject := "Short Link Passcode Reset Request - Lihatin"
	resetURL := fmt.Sprintf(config.GetRequiredEnv(config.EnvBackendURL)+"/v1/shorts/reset-passcode?token=%s", token)

	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Passcode Reset</title>
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
            background: linear-gradient(135deg, #ff5722 0%%, #d84315 100%%);
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
        .short-link-info {
            background-color: #e8f5e8;
            border: 1px solid #c3e6cb;
            border-radius: 12px;
            padding: 25px;
            margin: 25px 0;
            border-left: 4px solid #28a745;
            text-align: center;
        }
        .short-link-info h4 {
            color: #155724;
            margin-bottom: 15px;
            font-size: 18px;
            display: flex;
            align-items: center;
            justify-content: center;
            gap: 8px;
        }
        .short-link-display {
            background-color: #ffffff;
            border: 2px solid #28a745;
            border-radius: 8px;
            padding: 15px;
            font-family: 'Courier New', monospace;
            font-size: 18px;
            font-weight: bold;
            color: #155724;
            margin: 15px 0;
            word-break: break-all;
        }
        .reset-button { 
            display: inline-block;
            background: linear-gradient(135deg, #ff5722 0%%, #d84315 100%%);
            color: white !important;
            padding: 16px 32px;
            text-decoration: none;
            border-radius: 8px;
            font-weight: 600;
            font-size: 16px;
            letter-spacing: 0.5px;
            margin: 20px 0;
            transition: all 0.3s ease;
            box-shadow: 0 4px 15px rgba(255, 87, 34, 0.3);
        }
        .reset-button:hover {
            transform: translateY(-2px);
            box-shadow: 0 6px 20px rgba(255, 87, 34, 0.4);
        }
        .alternative-link {
            background-color: #f8f9fa;
            border: 1px solid #e9ecef;
            border-radius: 8px;
            padding: 20px;
            margin: 25px 0;
        }
        .alternative-link p {
            margin: 0 0 10px 0;
            font-size: 14px;
            color: #6c757d;
        }
        .alternative-link a {
            color: #ff5722;
            word-break: break-all;
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
        .security-tips {
            background-color: #e3f2fd;
            border: 1px solid #bbdefb;
            border-radius: 8px;
            padding: 20px;
            margin: 25px 0;
            border-left: 4px solid #2196f3;
        }
        .security-tips h4 {
            color: #1565c0;
            margin-bottom: 15px;
            font-size: 16px;
        }
        .security-tips ul {
            color: #1565c0;
            margin: 0;
            padding-left: 20px;
        }
        .security-tips li {
            margin: 8px 0;
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
            color: #ff5722;
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
            .reset-button { display: block; text-align: center; }
        }
    </style>
</head>
<body>
    <div class="email-container">
        <div class="email-header">
            <h1>Passcode Reset Request</h1>
            <p>Short Link Security - Lihatin</p>
        </div>
        
        <div class="email-content">
            <div class="greeting">Dear %s,</div>
            
            <div class="message">
                We have received a request to reset the passcode for one of your protected short links. 
                This security measure ensures that only authorized users can modify your link settings.
            </div>
            
            <div class="short-link-info">
                <h4>🔗 Protected Short Link</h4>
                <div class="short-link-display">%s</div>
                <p style="color: #155724; margin: 0; font-size: 14px;">
                    This link is currently protected with a passcode for enhanced security.
                </p>
            </div>
            
            <div style="text-align: center;">
                <a href="%s" class="reset-button">Reset Passcode Now</a>
            </div>
            
            <div class="alternative-link">
                <p><strong>Alternative method:</strong></p>
                <p>If the button above doesn't work, please copy and paste the following link into your browser:</p>
                <a href="%s">%s</a>
            </div>
            
            <div class="warning-notice">
                <h4>⚠️ Important Security Information</h4>
                <p><strong>Time Limit:</strong> This reset link will expire in 1 hour for security reasons.</p>
                <p><strong>One-Time Use:</strong> This link can only be used once to reset your passcode.</p>
                <p><strong>Not You?</strong> If you didn't request this reset, please ignore this email or contact our support team.</p>
            </div>
            
            <div class="security-tips">
                <h4>🔒 Passcode Security Best Practices</h4>
                <ul>
                    <li>Use a unique passcode that you haven't used elsewhere</li>
                    <li>Make it memorable but not easily guessable by others</li>
                    <li>Avoid using personal information like birthdays or names</li>
                    <li>Consider using a combination of letters and numbers</li>
                    <li>Update your passcode regularly for better security</li>
                </ul>
            </div>
            
            <p style="margin-top: 25px; color: #666; font-size: 14px; text-align: center;">
                <strong>Need help?</strong> If you're having trouble with your short links or have security concerns, 
                our support team is ready to assist you.
            </p>
        </div>
        
        <div class="email-footer">
            <div class="footer-content">
                <p>Protecting your short links is our priority.</p>
                <div class="footer-links">
                    <a href="%s/help/short-links">Short Link Help</a>
                    <a href="%s/security">Security Center</a>
                    <a href="%s/support">Contact Support</a>
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
`, userName, short, resetURL, resetURL, resetURL,
		config.GetRequiredEnv(config.EnvBackendURL), config.GetRequiredEnv(config.EnvBackendURL), config.GetRequiredEnv(config.EnvBackendURL))

	textBody := fmt.Sprintf(`
LIHATIN - SHORT LINK PASSCODE RESET REQUEST

Dear %s,

We have received a request to reset the passcode for your protected short link.

PROTECTED SHORT LINK: %s

To reset your passcode, please visit the following link:
%s

IMPORTANT SECURITY INFORMATION:
- This reset link will expire in 1 hour for security reasons
- This link can only be used once to reset your passcode
- If you didn't request this reset, please ignore this email or contact support

PASSCODE SECURITY BEST PRACTICES:
- Use a unique passcode that you haven't used elsewhere
- Make it memorable but not easily guessable by others
- Avoid using personal information like birthdays or names
- Update your passcode regularly for better security

Need help? Contact our support team: %s/support

Best regards,
The Lihatin Security Team

---
© 2025 Lihatin. All rights reserved.
This is an automated security message, please do not reply directly to this email.
`, userName, short, resetURL, config.GetRequiredEnv(config.EnvBackendURL))

	return es.sendEmail(toEmail, subject, textBody, htmlBody)
}


package mail

import (
	"fmt"

	"github.com/adehusnim37/lihatin-go/utils"
)

// SendPasswordResetEmail sends password reset email
func (es *EmailService) SendPasswordResetEmail(toEmail, userName, token string) error {
	subject := "Password Reset Request - Lihatin"
	resetURL := fmt.Sprintf(utils.GetRequiredEnv(utils.EnvFrontendURL)+`/auth/reset-password?token=%s`, token)

	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Password Reset</title>
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
            background: linear-gradient(135deg, #ff6b6b 0%%, #ee5a24 100%%);
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
        .reset-button { 
            display: inline-block;
            background: linear-gradient(135deg, #ff6b6b 0%%, #ee5a24 100%%);
            color: white !important;
            padding: 16px 32px;
            text-decoration: none;
            border-radius: 8px;
            font-weight: 600;
            font-size: 16px;
            letter-spacing: 0.5px;
            margin: 20px 0;
            transition: all 0.3s ease;
            box-shadow: 0 4px 15px rgba(255, 107, 107, 0.3);
        }
        .reset-button:hover {
            transform: translateY(-2px);
            box-shadow: 0 6px 20px rgba(255, 107, 107, 0.4);
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
        }
        .warning-notice p {
            color: #856404;
            margin: 8px 0;
            font-size: 14px;
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
            color: #ff6b6b;
            word-break: break-all;
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
            color: #ff6b6b;
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
            <h1>Password Reset Request</h1>
            <p>Account Security - Lihatin</p>
        </div>
        
        <div class="email-content">
            <div class="greeting">Dear %s,</div>
            
            <div class="message">
                We have received a request to reset the password for your Lihatin account. 
                If you initiated this request, please click the button below to create a new password.
            </div>
            
            <div style="text-align: center;">
                <a href="%s" class="reset-button">Reset My Password</a>
            </div>
            
            <div class="alternative-link">
                <p><strong>Alternative method:</strong></p>
                <p>If the button above doesn't work, please copy and paste the following link into your browser:</p>
                <a href="%s">%s</a>
            </div>
            
            <div class="warning-notice">
                <h4>⚠️ Important Security Information</h4>
                <p><strong>Time Limit:</strong> This reset link will expire in 1 hour for security reasons.</p>
                <p><strong>One-Time Use:</strong> This link can only be used once to reset your password.</p>
                <p><strong>Not You?</strong> If you didn't request this reset, please ignore this email or contact our support team immediately.</p>
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
`, userName, resetURL, resetURL, resetURL,
		utils.GetRequiredEnv(utils.EnvBackendURL), utils.GetRequiredEnv(utils.EnvBackendURL), utils.GetRequiredEnv(utils.EnvBackendURL))

	textBody := fmt.Sprintf(`
LIHATIN - PASSWORD RESET REQUEST

Dear %s,

We have received a request to reset the password for your Lihatin account.

If you initiated this request, please visit the following link to create a new password:
%s

IMPORTANT SECURITY INFORMATION:
- This reset link will expire in 1 hour for security reasons
- This link can only be used once to reset your password
- If you didn't request this reset, please ignore this email or contact support

PASSWORD SECURITY TIPS:
- Use a combination of uppercase and lowercase letters, numbers, and symbols
- Make your password at least 8 characters long
- Avoid using personal information like names or birthdays

Need help? Contact our support team: %s/support

Best regards,
The Lihatin Security Team

---
© 2025 Lihatin. All rights reserved.
This is an automated security message, please do not reply directly to this email.
`, userName, resetURL, utils.GetRequiredEnv(utils.EnvBackendURL))

	return es.sendEmail(toEmail, subject, textBody, htmlBody)
}

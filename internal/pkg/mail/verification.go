package mail

import (
	"fmt"

	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
)

// SendVerificationEmail sends email verification email
func (es *EmailService) SendVerificationEmail(toEmail, userName, token string) error {
	subject := "Email Verification Required - Lihatin"
	verificationURL := fmt.Sprintf(config.GetRequiredEnv(config.EnvBackendURL)+`/v1/auth/verify-email?token=%s`, token)

	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Email Verification</title>
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
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
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
        .verification-button { 
            display: inline-block;
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
            color: white !important;
            padding: 16px 32px;
            text-decoration: none;
            border-radius: 8px;
            font-weight: 600;
            font-size: 16px;
            letter-spacing: 0.5px;
            margin: 20px 0;
            transition: all 0.3s ease;
            box-shadow: 0 4px 15px rgba(102, 126, 234, 0.3);
        }
        .verification-button:hover {
            transform: translateY(-2px);
            box-shadow: 0 6px 20px rgba(102, 126, 234, 0.4);
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
            color: #667eea;
            word-break: break-all;
            font-size: 14px;
        }
        .security-notice {
            background-color: #fff3cd;
            border-left: 4px solid #ffc107;
            padding: 16px;
            margin: 25px 0;
            border-radius: 4px;
        }
        .security-notice h4 {
            color: #856404;
            margin-bottom: 8px;
            font-size: 14px;
        }
        .security-notice p {
            color: #856404;
            margin: 0;
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
            color: #667eea;
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
            .verification-button { display: block; text-align: center; }
        }
    </style>
</head>
<body>
    <div class="email-container">
        <div class="email-header">
            <h1>Welcome to Lihatin</h1>
            <p>Professional URL Shortening Service</p>
        </div>
        
        <div class="email-content">
            <div class="greeting">Dear %s,</div>
            
            <div class="message">
                Thank you for registering with Lihatin. To complete your account setup and start using our services, 
                please verify your email address by clicking the button below.
            </div>
            
            <div style="text-align: center;">
                <a href="%s" class="verification-button">Verify Email Address</a>
            </div>
            
            <div class="alternative-link">
                <p><strong>Alternative method:</strong></p>
                <p>If the button above doesn't work, please copy and paste the following link into your browser:</p>
                <a href="%s">%s</a>
            </div>
            
            <div class="security-notice">
                <h4>🔒 Security Information</h4>
                <p>This verification link will expire in 24 hours for security purposes. If you didn't create an account with Lihatin, please disregard this email.</p>
            </div>
        </div>
        
        <div class="email-footer">
            <div class="footer-content">
                <p>Thank you for choosing Lihatin</p>
                <div class="footer-links">
                    <a href="%s/support">Support</a>
                    <a href="%s/privacy">Privacy Policy</a>
                    <a href="%s/terms">Terms of Service</a>
                </div>
                <div class="company-info">
                    © 2025 Lihatin. All rights reserved.<br>
                    This is an automated message, please do not reply directly to this email.
                </div>
            </div>
        </div>
    </div>
</body>
</html>
`, userName, verificationURL, verificationURL, verificationURL,
		config.GetRequiredEnv(config.EnvFrontendURL), config.GetRequiredEnv(config.EnvFrontendURL), config.GetRequiredEnv(config.EnvFrontendURL))

	textBody := fmt.Sprintf(`
LIHATIN - EMAIL VERIFICATION REQUIRED

Dear %s,

Thank you for registering with Lihatin, your professional URL shortening service.

To complete your account setup and start using our services, please verify your email address by visiting the following link:

%s

SECURITY INFORMATION:
- This verification link will expire in 24 hours for security purposes
- If you didn't create an account with Lihatin, please disregard this email

For support, visit: %s/support

Best regards,
The Lihatin Team

---
© 2025 Lihatin. All rights reserved.
This is an automated message, please do not reply directly to this email.
`, userName, verificationURL, config.GetRequiredEnv(config.EnvFrontendURL))

	return es.sendEmail(toEmail, subject, textBody, htmlBody)
}

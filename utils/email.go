package utils

import (
	"fmt"
	"net/smtp"
	"time"

	"github.com/jordan-wright/email"
)

// EmailConfig holds email configuration
type EmailConfig struct {
	SMTPHost     string
	SMTPPort     string
	SMTPUsername string
	SMTPPassword string
	FromEmail    string
	FromName     string
}

// EmailTemplate types
type EmailTemplateType string

const (
	EmailTemplateVerification  EmailTemplateType = "verification"
	EmailTemplatePasswordReset EmailTemplateType = "password_reset"
	EmailTemplateTOTPSetup     EmailTemplateType = "totp_setup"
	EmailTemplateLoginAlert    EmailTemplateType = "login_alert"
)

// EmailService handles email operations
type EmailService struct {
	config EmailConfig
}

// NewEmailService creates a new email service instance
func NewEmailService() *EmailService {
	return &EmailService{
		config: EmailConfig{
			SMTPHost:     GetEnvOrDefault(EnvSMTPHost, "smtp.gmail.com"),
			SMTPPort:     GetEnvOrDefault(EnvSMTPPort, "587"),
			SMTPUsername: GetEnvOrDefault(EnvSMTPUser, ""),
			SMTPPassword: GetEnvOrDefault(EnvSMTPPass, ""),
			FromEmail:    GetEnvOrDefault(EnvFromEmail, "noreply@lihatin.com"),
			FromName:     GetEnvOrDefault(EnvFromName, "Lihatin"),
		},
	}
}

// SendVerificationEmail sends email verification email
// utils/email.go - Enhanced SendVerificationEmail

func (es *EmailService) SendVerificationEmail(toEmail, userName, token string) error {
	subject := "Email Verification Required - Lihatin"
	verificationURL := fmt.Sprintf(GetRequiredEnv(EnvBackendURL)+`/v1/auth/verify-email?token=%s`, token)

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
		GetRequiredEnv(EnvBackendURL), GetRequiredEnv(EnvBackendURL), GetRequiredEnv(EnvBackendURL))

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
`, userName, verificationURL, GetRequiredEnv(EnvBackendURL))

	return es.sendEmail(toEmail, subject, textBody, htmlBody)
}

// SendPasswordResetEmail sends password reset email
// utils/email.go - Enhanced SendPasswordResetEmail
func (es *EmailService) SendPasswordResetEmail(toEmail, userName, token string) error {
	subject := "Password Reset Request - Lihatin"
	resetURL := fmt.Sprintf(GetRequiredEnv(EnvBackendURL)+`/v1/auth/reset-password?token=%s`, token)

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
		GetRequiredEnv(EnvBackendURL), GetRequiredEnv(EnvBackendURL), GetRequiredEnv(EnvBackendURL))

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
`, userName, resetURL, GetRequiredEnv(EnvBackendURL))

	return es.sendEmail(toEmail, subject, textBody, htmlBody)
}

// SendTOTPSetupEmail sends TOTP setup confirmation email
// utils/email.go - Enhanced SendTOTPSetupEmail

func (es *EmailService) SendTOTPSetupEmail(toEmail, userName string) error {
	subject := "Two-Factor Authentication Enabled - Lihatin"

	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>2FA Enabled</title>
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
            background: linear-gradient(135deg, #20bf6b 0%%, #26de81 100%%);
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
        .success-notice {
            background-color: #d4edda;
            border: 1px solid #c3e6cb;
            border-radius: 8px;
            padding: 20px;
            margin: 25px 0;
            border-left: 4px solid #28a745;
            text-align: center;
        }
        .success-notice h3 {
            color: #155724;
            margin-bottom: 15px;
            font-size: 18px;
            display: flex;
            align-items: center;
            justify-content: center;
            gap: 10px;
        }
        .success-notice p {
            color: #155724;
            margin: 0;
            font-size: 16px;
        }
        .info-section {
            background-color: #e3f2fd;
            border: 1px solid #bbdefb;
            border-radius: 8px;
            padding: 20px;
            margin: 25px 0;
            border-left: 4px solid #2196f3;
        }
        .info-section h4 {
            color: #1565c0;
            margin-bottom: 15px;
            font-size: 16px;
        }
        .info-section ul {
            color: #1565c0;
            margin: 0;
            padding-left: 20px;
        }
        .info-section li {
            margin: 8px 0;
            font-size: 14px;
        }
        .recovery-warning {
            background-color: #fff3cd;
            border: 1px solid #ffeaa7;
            border-radius: 8px;
            padding: 20px;
            margin: 25px 0;
            border-left: 4px solid #ffc107;
        }
        .recovery-warning h4 {
            color: #856404;
            margin-bottom: 12px;
            font-size: 16px;
        }
        .recovery-warning p {
            color: #856404;
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
            color: #20bf6b;
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
            <h1>Security Enhanced</h1>
            <p>Two-Factor Authentication Activated</p>
        </div>
        
        <div class="email-content">
            <div class="greeting">Dear %s,</div>
            
            <div class="success-notice">
                <h3>🔒 Two-Factor Authentication Successfully Enabled</h3>
                <p>Your Lihatin account is now protected with an additional layer of security.</p>
            </div>
            
            <div class="info-section">
                <h4>What This Means</h4>
                <ul>
                    <li>Your account is now significantly more secure against unauthorized access</li>
                    <li>You'll need your authenticator app to generate codes when logging in</li>
                    <li>Even if someone knows your password, they can't access your account without your device</li>
                    <li>This protection applies to all login attempts from new devices</li>
                </ul>
            </div>
            
            <div class="recovery-warning">
                <h4>⚠️ Important: Save Your Recovery Codes</h4>
                <p><strong>Critical:</strong> Please ensure you have saved your recovery codes in a secure location.</p>
                <p><strong>Why?</strong> If you lose access to your authenticator app, recovery codes are the only way to regain access to your account.</p>
                <p><strong>Best Practice:</strong> Store them offline in multiple secure locations (safe, password manager, etc.)</p>
            </div>
            
            <div class="info-section">
                <h4>📱 Managing Your 2FA</h4>
                <ul>
                    <li>You can manage your 2FA settings from your account security page</li>
                    <li>Generate new recovery codes if needed</li>
                    <li>Add backup authentication methods</li>
                    <li>Review your security activity regularly</li>
                </ul>
            </div>
            
            <p style="margin-top: 30px; color: #666; font-size: 14px;">
                <strong>Didn't enable 2FA?</strong> If you didn't set up two-factor authentication, 
                please contact our security team immediately as your account may be compromised.
            </p>
        </div>
        
        <div class="email-footer">
            <div class="footer-content">
                <p>Questions about account security? We're here to help.</p>
                <div class="footer-links">
                    <a href="%s/security">Security Center</a>
                    <a href="%s/support">Contact Support</a>
                    <a href="%s/help/2fa">2FA Help</a>
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
`, userName, GetRequiredEnv(EnvBackendURL), GetRequiredEnv(EnvBackendURL), GetRequiredEnv(EnvBackendURL))

	textBody := fmt.Sprintf(`
LIHATIN - TWO-FACTOR AUTHENTICATION ENABLED

Dear %s,

SECURITY ENHANCED: Two-factor authentication has been successfully enabled for your Lihatin account.

WHAT THIS MEANS:
- Your account is now significantly more secure
- You'll need your authenticator app when logging in
- Additional protection against unauthorized access

IMPORTANT - SAVE YOUR RECOVERY CODES:
Please ensure you have saved your recovery codes in a secure location. If you lose access to your authenticator app, recovery codes are the only way to regain access to your account.

MANAGING YOUR 2FA:
You can manage your 2FA settings from your account security page at %s/security

DIDN'T ENABLE 2FA?
If you didn't set up two-factor authentication, please contact our security team immediately.

Support: %s/support
Security Center: %s/security

Best regards,
The Lihatin Security Team

---
© 2025 Lihatin. All rights reserved.
This is an automated security notification.
`, userName, GetRequiredEnv(EnvBackendURL), GetRequiredEnv(EnvBackendURL), GetRequiredEnv(EnvBackendURL))

	return es.sendEmail(toEmail, subject, textBody, htmlBody)
}

// SendLoginAlertEmail sends login alert email
func (es *EmailService) SendLoginAlertEmail(toEmail, userName, ipAddress, userAgent string) error {
	subject := "Security Alert: New Login Detected - Lihatin"
	loginTime := time.Now().Format("January 2, 2006 at 3:04 PM MST")
	location := GetLocationString(ipAddress)

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
		GetRequiredEnv(EnvBackendURL), GetRequiredEnv(EnvBackendURL),
		GetRequiredEnv(EnvBackendURL), GetRequiredEnv(EnvBackendURL), GetRequiredEnv(EnvBackendURL))

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
		GetRequiredEnv(EnvBackendURL), GetRequiredEnv(EnvBackendURL))

	return es.sendEmail(toEmail, subject, textBody, htmlBody)
}

// SendPasscodeResetEmail sends passcode reset email
func (es *EmailService) SendPasscodeResetEmail(toEmail, short, userName, token string) error {
	subject := "Short Link Passcode Reset Request - Lihatin"
	resetURL := fmt.Sprintf(GetRequiredEnv(EnvBackendURL)+"/v1/shorts/reset-passcode?token=%s", token)

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
		GetRequiredEnv(EnvBackendURL), GetRequiredEnv(EnvBackendURL), GetRequiredEnv(EnvBackendURL))

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
`, userName, short, resetURL, GetRequiredEnv(EnvBackendURL))

	return es.sendEmail(toEmail, subject, textBody, htmlBody)
}

// sendEmail sends an email using SMTP
func (es *EmailService) sendEmail(to, subject, textBody, htmlBody string) error {
	if es.config.SMTPUsername == "" || es.config.SMTPPassword == "" {
		// Skip sending email if SMTP is not configured
		fmt.Printf("Email would be sent to %s with subject: %s\n", to, subject)
		return nil
	}

	e := email.NewEmail()
	e.From = fmt.Sprintf("%s <%s>", es.config.FromName, es.config.FromEmail)
	e.To = []string{to}
	e.Subject = subject
	e.Text = []byte(textBody)
	e.HTML = []byte(htmlBody)

	auth := smtp.PlainAuth("", es.config.SMTPUsername, es.config.SMTPPassword, es.config.SMTPHost)
	addr := fmt.Sprintf("%s:%s", es.config.SMTPHost, es.config.SMTPPort)

	return e.Send(addr, auth)
}

// IsTokenExpired checks if a token timestamp has expired
func IsTokenExpired(tokenTime *time.Time, expiryHours int) bool {
	if tokenTime == nil {
		return true
	}
	return time.Now().After(tokenTime.Add(time.Duration(expiryHours) * time.Hour))
}

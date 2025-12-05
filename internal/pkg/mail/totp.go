package mail

import (
	"fmt"

	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
)

// SendTOTPSetupEmail sends TOTP setup confirmation email
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
`, userName, config.GetRequiredEnv(config.EnvBackendURL), config.GetRequiredEnv(config.EnvBackendURL), config.GetRequiredEnv(config.EnvBackendURL))

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
`, userName, config.GetRequiredEnv(config.EnvBackendURL), config.GetRequiredEnv(config.EnvBackendURL), config.GetRequiredEnv(config.EnvBackendURL))

	return es.sendEmail(toEmail, subject, textBody, htmlBody)
}

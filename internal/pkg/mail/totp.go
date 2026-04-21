package mail

import (
	"fmt"
	"time"

	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
	"github.com/adehusnim37/lihatin-go/internal/pkg/ip"
)

// SendTOTPSetupEmail sends TOTP setup confirmation email.
func (es *EmailService) SendTOTPSetupEmail(toEmail, userName string) error {
	subject := "Two-Factor Authentication Enabled - Lihatin"
	frontendURL := config.GetEnvOrDefault(config.EnvFrontendURL, "http://localhost:3000")
	securityURL := fmt.Sprintf("%s/main", frontendURL)

	htmlBody := renderEmailTemplate(emailTemplate{
		Badge:    "Security",
		Title:    "Two-factor authentication enabled",
		Subtitle: "Your account now has stronger sign-in protection.",
		Greeting: fmt.Sprintf("Hi %s,", userName),
		Intro:    "Two-factor authentication (2FA) is now active on your account.",
		Sections: []string{
			renderListSection("What this changes", []string{
				"Login now requires a code from your authenticator app",
				"Unauthorized access becomes much harder",
				"This protection applies across devices",
			}),
			renderListSection("Important", []string{
				"Store your recovery codes in a safe place",
				"If you lose your authenticator app, recovery codes are required",
			}),
		},
		Actions: []emailAction{
			{Label: "Manage Security", URL: securityURL, Variant: "primary"},
			{Label: "Get Support", URL: fmt.Sprintf("%s/support", frontendURL), Variant: "secondary"},
		},
		Notice:        "If you did not enable 2FA, contact support immediately.",
		FooterBaseURL: frontendURL,
	})

	textBody := fmt.Sprintf(`
LIHATIN - TWO-FACTOR AUTHENTICATION ENABLED

Hi %s,

Two-factor authentication has been enabled on your account.

If this was not you, contact support immediately.

Security page: %s
Support: %s/support

The Lihatin Security Team
`, userName, securityURL, frontendURL)

	return es.sendEmail(toEmail, subject, textBody, htmlBody)
}

// SendTOTPDisabledEmail sends notification when TOTP has been disabled.
func (es *EmailService) SendTOTPDisabledEmail(toEmail, userName, ipAddress, userAgent string) error {
	subject := "Security Alert: Two-Factor Authentication Disabled - Lihatin"
	frontendURL := config.GetEnvOrDefault(config.EnvFrontendURL, "http://localhost:3000")
	securityURL := fmt.Sprintf("%s/main", frontendURL)
	disabledAt := time.Now().Format("January 2, 2006 at 3:04 PM MST")
	location := ip.GetLocationString(ipAddress)

	htmlBody := renderEmailTemplate(emailTemplate{
		Badge:    "Security Alert",
		Title:    "Two-factor authentication disabled",
		Subtitle: "Your account no longer requires authenticator app code at login.",
		Greeting: fmt.Sprintf("Hi %s,", userName),
		Intro:    "We detected that two-factor authentication (2FA) was disabled for your account.",
		Details: []emailDetail{
			{Label: "Date & Time", Value: disabledAt},
			{Label: "Location", Value: location},
			{Label: "IP Address", Value: ipAddress},
			{Label: "Device/Browser", Value: userAgent},
		},
		Sections: []string{
			renderListSection("If this was not you", []string{
				"Change your password immediately",
				"Re-enable two-factor authentication",
				"Review recent account activity",
				"Contact support",
			}),
		},
		Actions: []emailAction{
			{Label: "Security Settings", URL: securityURL, Variant: "danger"},
			{Label: "Contact Support", URL: fmt.Sprintf("%s/support", frontendURL), Variant: "secondary"},
		},
		Notice:        "If you did not disable 2FA, secure your account immediately.",
		FooterBaseURL: frontendURL,
	})

	textBody := fmt.Sprintf(`
LIHATIN - SECURITY ALERT: TWO-FACTOR AUTHENTICATION DISABLED

Hi %s,

Two-factor authentication has been disabled on your account.

Event details:
- Date & Time: %s
- Location: %s
- IP Address: %s
- Device/Browser: %s

If this was not you:
- Change your password immediately
- Re-enable 2FA
- Contact support

Security page: %s
Support: %s/support

The Lihatin Security Team
`, userName, disabledAt, location, ipAddress, userAgent, securityURL, frontendURL)

	return es.sendEmail(toEmail, subject, textBody, htmlBody)
}

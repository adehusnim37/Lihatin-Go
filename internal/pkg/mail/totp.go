package mail

import (
	"fmt"

	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
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

package mail

import (
	"fmt"

	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
)

// SendChangeEmailConfirmation sends a confirmation email for email change.
func (es *EmailService) SendChangeEmailConfirmation(toEmail, userName, token string) error {
	subject := "Confirm Your Email Change - Lihatin"
	frontendURL := config.GetEnvOrDefault(config.EnvFrontendURL, "http://localhost:3000")
	backendURL := config.GetEnvOrDefault(config.EnvBackendURL, "http://localhost:8080")
	confirmationURL := fmt.Sprintf("%s/v1/auth/verify-email?token=%s", backendURL, token)

	htmlBody := renderEmailTemplate(emailTemplate{
		Badge:         "Security",
		Title:         "Confirm your new email",
		Subtitle:      "Verify this request to complete the email change.",
		Greeting:      fmt.Sprintf("Hi %s,", userName),
		Intro:         "You requested to update your account email address. Confirm this request using the button below.",
		Actions:       []emailAction{{Label: "Confirm Email Change", URL: confirmationURL, Variant: "primary"}},
		Notice:        "This confirmation link expires in 1 hour. If you did not request this change, ignore this email and secure your account.",
		FallbackURL:   confirmationURL,
		FallbackLabel: confirmationURL,
		FooterBaseURL: frontendURL,
	})

	textBody := fmt.Sprintf(`
LIHATIN - EMAIL CHANGE CONFIRMATION

Hi %s,

You requested to change your account email.
Use this link to confirm:

%s

Security note:
- This link expires in 1 hour
- If this was not you, ignore this email and secure your account

Support: %s/support

The Lihatin Team
`, userName, confirmationURL, frontendURL)

	return es.sendEmail(toEmail, subject, textBody, htmlBody)
}

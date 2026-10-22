package mail

import (
	"fmt"

	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
)

// SendChangeOldEmailNotification sends a notification to the old email address when user changes their email.
func (es *EmailService) SendChangeOldEmailNotification(oldEmail, newEmail, username, revokeToken string) error {
	subject := "Lihatin - Email Change Notification"
	frontendURL := config.GetEnvOrDefault(config.EnvFrontendURL, "http://localhost:3000")
	backendURL := config.GetEnvOrDefault(config.EnvBackendURL, "http://localhost:8080")
	revokeURL := fmt.Sprintf("%s/v1/auth/revoke-email-change?token=%s", backendURL, revokeToken)
	dashboardURL := fmt.Sprintf("%s/main", frontendURL)

	htmlBody := renderEmailTemplate(emailTemplate{
		Badge:    "Security Alert",
		Title:    "Email change detected",
		Subtitle: "Review this activity and confirm whether it was you.",
		Greeting: fmt.Sprintf("Hi %s,", username),
		Intro:    "Your account email was updated. If this action was not made by you, revoke it immediately.",
		Details: []emailDetail{
			{Label: "From", Value: oldEmail},
			{Label: "To", Value: newEmail},
		},
		Actions: []emailAction{
			{Label: "This wasn't me", URL: revokeURL, Variant: "danger"},
			{Label: "This was me", URL: dashboardURL, Variant: "secondary"},
		},
		Notice:        "The revoke link is valid for 24 hours.",
		FallbackURL:   revokeURL,
		FallbackLabel: revokeURL,
		FooterBaseURL: frontendURL,
	})

	textBody := fmt.Sprintf(`
LIHATIN - EMAIL CHANGE NOTIFICATION

Hi %s,

We noticed your account email was changed:
From: %s
To: %s

If this was not you, use this revoke link immediately:
%s

This link is valid for 24 hours.

Support: %s/support

The Lihatin Team
`, username, oldEmail, newEmail, revokeURL, frontendURL)

	return es.sendEmail(oldEmail, subject, textBody, htmlBody)
}

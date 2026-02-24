package mail

import (
	"fmt"

	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
)

// SendUsernameChangedEmail sends notification when username is changed successfully.
func (es *EmailService) SendUsernameChangedEmail(toEmail, oldUsername, newUsername string) error {
	subject := "Username Changed Successfully - Lihatin"
	frontendURL := config.GetEnvOrDefault(config.EnvFrontendURL, "http://localhost:3000")
	dashboardURL := fmt.Sprintf("%s/main", frontendURL)

	htmlBody := renderEmailTemplate(emailTemplate{
		Badge:    "Account",
		Title:    "Username updated",
		Subtitle: "Your new username is now active.",
		Greeting: fmt.Sprintf("Hi %s,", newUsername),
		Intro:    "Your username was changed successfully. We are sending this confirmation for account security.",
		Details: []emailDetail{
			{Label: "Old Username", Value: oldUsername},
			{Label: "New Username", Value: newUsername},
		},
		Sections: []string{
			renderListSection("What to do next", []string{
				"Use your new username for future login",
				"Update any saved credentials or integrations",
				"Contact support immediately if this change was not made by you",
			}),
		},
		Actions: []emailAction{
			{Label: "Go to Dashboard", URL: dashboardURL, Variant: "primary"},
			{Label: "Contact Support", URL: fmt.Sprintf("%s/support", frontendURL), Variant: "secondary"},
		},
		Notice:        "If this was not you, reset your password immediately and review account security settings.",
		FooterBaseURL: frontendURL,
	})

	textBody := fmt.Sprintf(`
LIHATIN - USERNAME CHANGED SUCCESSFULLY

Hi %s,

Your username has been changed successfully.

Old Username: %s
New Username: %s

If this was not you, reset your password and contact support immediately.

Dashboard: %s
Support: %s/support

The Lihatin Security Team
`, newUsername, oldUsername, newUsername, dashboardURL, frontendURL)

	return es.sendEmail(toEmail, subject, textBody, htmlBody)
}

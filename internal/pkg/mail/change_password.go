package mail

import (
	"fmt"

	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
)

// SuccessfulPasswordChangeEmail sends email notification for successful password change.
func (es *EmailService) SuccessfulPasswordChangeEmail(toEmail, userName string) error {
	subject := "Password Successfully Changed - Lihatin"
	frontendURL := config.GetEnvOrDefault(config.EnvFrontendURL, "http://localhost:3000")

	htmlBody := renderEmailTemplate(emailTemplate{
		Badge:    "Security",
		Title:    "Password updated",
		Subtitle: "Your account password has been changed successfully.",
		Greeting: fmt.Sprintf("Hi %s,", userName),
		Intro:    "If you made this change, no further action is needed.",
		Sections: []string{
			renderParagraphSection("Didn't do this?", "Reset your password immediately and contact support."),
			renderListSection("Keep your account secure", []string{
				"Use a strong unique password",
				"Enable two-factor authentication",
				"Review recent account activity",
			}),
		},
		Actions: []emailAction{
			{Label: "Security Settings", URL: fmt.Sprintf("%s/main", frontendURL), Variant: "primary"},
			{Label: "Contact Support", URL: fmt.Sprintf("%s/support", frontendURL), Variant: "secondary"},
		},
		FooterBaseURL: frontendURL,
	})

	textBody := fmt.Sprintf(`
LIHATIN - PASSWORD CHANGED SUCCESSFULLY

Hi %s,

Your password was changed successfully.

If this was not you:
- Reset your password immediately
- Contact support
- Review recent account activity

Support: %s/support

The Lihatin Security Team
`, userName, frontendURL)

	return es.sendEmail(toEmail, subject, textBody, htmlBody)
}

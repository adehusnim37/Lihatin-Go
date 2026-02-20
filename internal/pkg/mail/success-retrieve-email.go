package mail

import (
	"fmt"

	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
)

// SendSuccessRetrieveEmail sends a reassuring email to old email after successful revoke.
func (es *EmailService) SendSuccessRetrieveEmail(toEmail, userName string) error {
	subject := "Your Account is Secure - Email Change Revoked Successfully"
	frontendURL := config.GetEnvOrDefault(config.EnvFrontendURL, "http://localhost:3000")
	securityURL := fmt.Sprintf("%s/main", frontendURL)

	htmlBody := renderEmailTemplate(emailTemplate{
		Badge:    "Security",
		Title:    "Email change revoked",
		Subtitle: "Your original email has been restored and verified.",
		Greeting: fmt.Sprintf("Hi %s,", userName),
		Intro:    "The unauthorized email change request was successfully revoked. Your account remains secure.",
		Details: []emailDetail{
			{Label: "Restored Email", Value: toEmail},
		},
		Sections: []string{
			renderListSection("What we did", []string{
				"Cancelled the email change request",
				"Restored your original email",
				"Kept your account settings and data intact",
			}),
			renderListSection("Recommended next steps", []string{
				"Review recent account activity",
				"Update your password if needed",
				"Enable two-factor authentication",
			}),
		},
		Actions: []emailAction{
			{Label: "Review Security", URL: securityURL, Variant: "primary"},
			{Label: "Contact Support", URL: fmt.Sprintf("%s/support", frontendURL), Variant: "secondary"},
		},
		Notice:        "If you did not initiate this action, secure your account immediately.",
		FooterBaseURL: frontendURL,
	})

	textBody := fmt.Sprintf(`
YOUR ACCOUNT IS SECURE - EMAIL CHANGE REVOKED

Hi %s,

The recent email change request was revoked successfully.
Your original email was restored: %s

Recommended actions:
- Review your account activity
- Update your password if needed
- Enable two-factor authentication

Security page: %s
Support: %s/support

The Lihatin Security Team
`, userName, toEmail, securityURL, frontendURL)

	return es.sendEmail(toEmail, subject, textBody, htmlBody)
}

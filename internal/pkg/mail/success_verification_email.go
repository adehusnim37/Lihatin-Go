package mail

import (
	"fmt"

	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
)

// SendSuccessEmailVerification sends a confirmation email after successful email verification.
func (es *EmailService) SendSuccessEmailVerification(toEmail, userName string) error {
	subject := "Your Email Has Been Verified - Lihatin"
	frontendURL := config.GetEnvOrDefault(config.EnvFrontendURL, "http://localhost:3000")
	dashboardURL := fmt.Sprintf("%s/main", frontendURL)

	htmlBody := renderEmailTemplate(emailTemplate{
		Badge:         "Success",
		Title:         "Email verified",
		Subtitle:      "Your account is now fully active.",
		Greeting:      fmt.Sprintf("Hi %s,", userName),
		Intro:         "Your email address has been verified successfully. You can now use all Lihatin features.",
		Actions:       []emailAction{{Label: "Go to Dashboard", URL: dashboardURL, Variant: "primary"}},
		FallbackURL:   dashboardURL,
		FallbackLabel: dashboardURL,
		FooterBaseURL: frontendURL,
		Sections: []string{
			renderListSection("What you can do now", []string{
				"Create and manage short links",
				"Track analytics and performance",
				"Configure account security settings",
			}),
		},
	})

	textBody := fmt.Sprintf(`
LIHATIN - EMAIL VERIFIED SUCCESSFULLY

Hi %s,

Your email has been verified successfully.

Dashboard:
%s

Support: %s/support

The Lihatin Team
`, userName, dashboardURL, frontendURL)

	return es.sendEmail(toEmail, subject, textBody, htmlBody)
}

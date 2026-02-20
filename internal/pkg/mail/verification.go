package mail

import (
	"fmt"

	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
)

// SendVerificationEmail sends email verification email.
func (es *EmailService) SendVerificationEmail(toEmail, userName, token string) error {
	subject := "Email Verification Required - Lihatin"
	frontendURL := config.GetEnvOrDefault(config.EnvFrontendURL, "http://localhost:3000")
	backendURL := config.GetEnvOrDefault(config.EnvBackendURL, "http://localhost:8080")
	verificationURL := fmt.Sprintf("%s/v1/auth/verify-email?token=%s", backendURL, token)

	htmlBody := renderEmailTemplate(emailTemplate{
		Badge:         "Email Verification",
		Title:         "Verify your email address",
		Subtitle:      "Complete your account setup to start using Lihatin.",
		Greeting:      fmt.Sprintf("Hi %s,", userName),
		Intro:         "Thanks for registering. Please verify your email address using the button below.",
		Actions:       []emailAction{{Label: "Verify Email Address", URL: verificationURL, Variant: "primary"}},
		Notice:        "This verification link expires in 24 hours. If you did not create this account, you can ignore this email.",
		FallbackURL:   verificationURL,
		FallbackLabel: verificationURL,
		FooterBaseURL: frontendURL,
		Sections: []string{
			renderListSection("After verification", []string{
				"Access all account features",
				"Manage and track your short links",
				"Improve account security settings",
			}),
		},
	})

	textBody := fmt.Sprintf(`
LIHATIN - EMAIL VERIFICATION REQUIRED

Hi %s,

Thanks for registering.
Please verify your email by opening the link below:

%s

Security note:
- This link expires in 24 hours
- If you did not create this account, ignore this email

Support: %s/support

The Lihatin Team
`, userName, verificationURL, frontendURL)

	return es.sendEmail(toEmail, subject, textBody, htmlBody)
}

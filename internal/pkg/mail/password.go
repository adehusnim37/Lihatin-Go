package mail

import (
	"fmt"

	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
)

// SendPasswordResetEmail sends password reset email.
func (es *EmailService) SendPasswordResetEmail(toEmail, userName, token string) error {
	subject := "Password Reset Request - Lihatin"
	frontendURL := config.GetEnvOrDefault(config.EnvFrontendURL, "http://localhost:3000")
	resetURL := fmt.Sprintf("%s/auth/reset-password?token=%s", frontendURL, token)

	htmlBody := renderEmailTemplate(emailTemplate{
		Badge:         "Security",
		Title:         "Password reset request",
		Subtitle:      "Reset your account password securely.",
		Greeting:      fmt.Sprintf("Hi %s,", userName),
		Intro:         "We received a request to reset your password. If this was you, continue using the button below.",
		Actions:       []emailAction{{Label: "Reset Password", URL: resetURL, Variant: "danger"}},
		Notice:        "This reset link is valid for 1 hour and can only be used once.",
		FallbackURL:   resetURL,
		FallbackLabel: resetURL,
		FooterBaseURL: frontendURL,
		Sections: []string{
			renderListSection("If this wasn't you", []string{
				"Ignore this email",
				"Change your password immediately if you suspect compromise",
				"Review your recent login activity",
			}),
			renderListSection("Strong password tips", []string{
				"Use at least 8 characters",
				"Mix uppercase, lowercase, numbers, and symbols",
				"Avoid personal information",
			}),
		},
	})

	textBody := fmt.Sprintf(`
LIHATIN - PASSWORD RESET REQUEST

Hi %s,

We received a request to reset your password.
Use this link to continue:

%s

Security note:
- This link expires in 1 hour
- The link can only be used once
- If this was not you, ignore this email

Support: %s/support

The Lihatin Security Team
`, userName, resetURL, frontendURL)

	return es.sendEmail(toEmail, subject, textBody, htmlBody)
}

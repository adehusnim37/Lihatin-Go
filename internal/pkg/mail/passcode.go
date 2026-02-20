package mail

import (
	"fmt"

	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
)

// SendPasscodeResetEmail sends passcode reset email.
func (es *EmailService) SendPasscodeResetEmail(toEmail, short, userName, token string) error {
	subject := "Short Link Passcode Reset Request - Lihatin"
	frontendURL := config.GetEnvOrDefault(config.EnvFrontendURL, "http://localhost:3000")
	backendURL := config.GetEnvOrDefault(config.EnvBackendURL, "http://localhost:8080")
	resetURL := fmt.Sprintf("%s/v1/shorts/reset-passcode?token=%s", backendURL, token)

	htmlBody := renderEmailTemplate(emailTemplate{
		Badge:    "Security",
		Title:    "Passcode reset request",
		Subtitle: "Reset passcode for your protected short link.",
		Greeting: fmt.Sprintf("Hi %s,", userName),
		Intro:    "We received a request to reset the passcode for one of your protected links.",
		Details: []emailDetail{
			{Label: "Protected Link", Value: short},
		},
		Actions:       []emailAction{{Label: "Reset Passcode", URL: resetURL, Variant: "danger"}},
		Notice:        "This reset link is valid for 1 hour and can only be used once.",
		FallbackURL:   resetURL,
		FallbackLabel: resetURL,
		FooterBaseURL: frontendURL,
		Sections: []string{
			renderListSection("Security recommendations", []string{
				"Keep passcodes private",
				"Use non-obvious combinations",
				"Rotate passcodes periodically",
			}),
		},
	})

	textBody := fmt.Sprintf(`
LIHATIN - SHORT LINK PASSCODE RESET REQUEST

Hi %s,

We received a request to reset the passcode for this protected link:
%s

Use this link to continue:
%s

Security note:
- This link expires in 1 hour
- The link can only be used once
- If this was not you, ignore this email

Support: %s/support

The Lihatin Security Team
`, userName, short, resetURL, frontendURL)

	return es.sendEmail(toEmail, subject, textBody, htmlBody)
}

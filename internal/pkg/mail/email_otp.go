package mail

import (
	"fmt"
	"strings"

	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
)

// SendEmailOTP sends OTP code for signup or login verification.
func (es *EmailService) SendEmailOTP(toEmail, userName, purpose, code string) error {
	subject := "Your Verification Code - Lihatin"
	scenario := "login"
	if strings.EqualFold(purpose, "signup") {
		scenario = "signup"
		subject = "Complete Your Signup - Lihatin"
	}

	frontendURL := config.GetEnvOrDefault(config.EnvFrontendURL, "http://localhost:3000")
	greetingName := strings.TrimSpace(userName)
	if greetingName == "" {
		greetingName = "there"
	}

	htmlBody := renderEmailTemplate(emailTemplate{
		Badge:    "Security Code",
		Title:    "Your one-time verification code",
		Subtitle: "Use this code to continue securely.",
		Greeting: fmt.Sprintf("Hi %s,", greetingName),
		Intro:    fmt.Sprintf("We received a %s request. Enter this 6-digit code in the app.", scenario),
		Details: []emailDetail{
			{Label: "Verification code", Value: code},
			{Label: "Valid for", Value: "10 minutes"},
		},
		Sections: []string{
			renderListSection("Security reminders", []string{
				"Never share this code with anyone",
				"Lihatin support will never ask for this code",
				"Ignore this email if you did not initiate this request",
			}),
		},
		Notice:        "This code expires in 10 minutes and can only be used once.",
		FooterBaseURL: frontendURL,
	})

	textBody := fmt.Sprintf(`
LIHATIN - VERIFICATION CODE

Hi %s,

We received a %s request.
Use this 6-digit code:

%s

This code expires in 10 minutes.
Do not share this code with anyone.

Support: %s/support

The Lihatin Team
`, greetingName, scenario, code, frontendURL)

	return es.sendEmail(toEmail, subject, textBody, htmlBody)
}

package mail

import (
	"fmt"
	"time"

	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
	"github.com/adehusnim37/lihatin-go/internal/pkg/ip"
)

// SendLoginAlertEmail sends login alert email.
func (es *EmailService) SendLoginAlertEmail(toEmail, userName, ipAddress, userAgent string) error {
	subject := "Security Alert: New Login Detected - Lihatin"
	frontendURL := config.GetEnvOrDefault(config.EnvFrontendURL, "http://localhost:3000")
	loginTime := time.Now().Format("January 2, 2006 at 3:04 PM MST")
	location := ip.GetLocationString(ipAddress)

	htmlBody := renderEmailTemplate(emailTemplate{
		Badge:    "Security Alert",
		Title:    "New login detected",
		Subtitle: "We detected a login to your account from a device or location.",
		Greeting: fmt.Sprintf("Hi %s,", userName),
		Intro:    "Review this activity. If it was you, no action is needed.",
		Details: []emailDetail{
			{Label: "Date & Time", Value: loginTime},
			{Label: "Location", Value: location},
			{Label: "IP Address", Value: ipAddress},
			{Label: "Device/Browser", Value: userAgent},
		},
		Sections: []string{
			renderListSection("If this was not you", []string{
				"Change your password immediately",
				"Enable two-factor authentication",
				"Review account activity",
				"Contact support",
			}),
		},
		Actions: []emailAction{
			{Label: "Security Settings", URL: fmt.Sprintf("%s/main", frontendURL), Variant: "danger"},
			{Label: "Contact Support", URL: fmt.Sprintf("%s/support", frontendURL), Variant: "secondary"},
		},
		Notice:        "Lihatin will never ask for your password by email.",
		FooterBaseURL: frontendURL,
	})

	textBody := fmt.Sprintf(`
LIHATIN - SECURITY ALERT: NEW LOGIN DETECTED

Hi %s,

New login details:
- Date & Time: %s
- Location: %s
- IP Address: %s
- Device/Browser: %s

If this was not you:
- Change your password
- Enable 2FA
- Contact support

Security page: %s/main
Support: %s/support

The Lihatin Security Team
`, userName, loginTime, location, ipAddress, userAgent, frontendURL, frontendURL)

	return es.sendEmail(toEmail, subject, textBody, htmlBody)
}

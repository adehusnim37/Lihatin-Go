package mail

import (
	"fmt"
	"time"

	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
)

// SendPremiumRedeemedEmail sends notification when a premium key is redeemed.
func (es *EmailService) SendPremiumRedeemedEmail(toEmail, userName, redeemedCode string) error {
	subject := "Premium Activated Successfully - Lihatin"
	frontendURL := config.GetEnvOrDefault(config.EnvFrontendURL, "http://localhost:3000")
	dashboardURL := fmt.Sprintf("%s/main", frontendURL)

	htmlBody := renderEmailTemplate(emailTemplate{
		Badge:    "Premium",
		Title:    "Premium activated",
		Subtitle: "Your account has been upgraded successfully.",
		Greeting: fmt.Sprintf("Hi %s,", userName),
		Intro:    "Your premium key redemption is complete and premium features are now active on your account.",
		Details: []emailDetail{
			{Label: "Plan", Value: "Premium"},
			{Label: "Redeemed Key", Value: redeemedCode},
			{Label: "Redeemed At", Value: time.Now().Format("2006-01-02 15:04:05")},
			{Label: "Expires At", Value: "Never"},
		},
		Sections: []string{
			renderListSection("You now have access to", []string{
				"Advanced analytics and insights",
				"Higher Limits",
				"Priority support",
				"Premium account capabilities",
			}),
		},
		Actions: []emailAction{
			{Label: "Open Dashboard", URL: dashboardURL, Variant: "primary"},
			{Label: "Contact Support", URL: fmt.Sprintf("%s/support", frontendURL), Variant: "secondary"},
		},
		Notice:        "If you did not redeem this key, secure your account and contact support immediately.",
		FooterBaseURL: frontendURL,
	})

	textBody := fmt.Sprintf(`
LIHATIN - PREMIUM ACTIVATED

Hi %s,

Your account has been upgraded to Premium successfully.

Plan: Premium
Redeemed Key: %s

Premium features are now active on your account.

Dashboard: %s
Support: %s/support

The Lihatin Team
`, userName, redeemedCode, dashboardURL, frontendURL)

	return es.sendEmail(toEmail, subject, textBody, htmlBody)
}

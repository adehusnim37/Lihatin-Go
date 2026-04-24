package mail

import (
	"fmt"
	"strings"
	"time"

	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
)

// SendPremiumCodeDeliveryEmail sends premium secret code to target recipient.
func (es *EmailService) SendPremiumCodeDeliveryEmail(
	toEmail,
	toName,
	secretCode string,
	validUntil *time.Time,
	limitUsage *int64,
	note string,
) error {
	subject := "Your Premium Access Code - Lihatin"
	frontendURL := config.GetEnvOrDefault(config.EnvFrontendURL, "http://localhost:3000")
	redeemURL := fmt.Sprintf("%s/profile/me", frontendURL)

	recipientName := strings.TrimSpace(toName)
	if recipientName == "" {
		recipientName = "there"
	}

	validUntilLabel := "No expiration set"
	if validUntil != nil {
		validUntilLabel = validUntil.Local().Format("2006-01-02 15:04:05")
	}

	limitUsageLabel := "Unlimited"
	if limitUsage != nil {
		limitUsageLabel = fmt.Sprintf("%d", *limitUsage)
	}

	sections := []string{
		renderParagraphSection("Secret code", secretCode),
		renderParagraphSection(
			"How to redeem",
			"Login to your Lihatin account, open your profile page, then paste this secret code in Redeem Premium Code section.",
		),
		renderListSection("Important notes", []string{
			"Treat this code as confidential information.",
			"Do not share this code publicly.",
			"Contact support if you receive this by mistake.",
		}),
	}

	cleanNote := strings.TrimSpace(note)
	if cleanNote != "" {
		sections = append(sections, renderParagraphSection("Message from admin", cleanNote))
	}

	htmlBody := renderEmailTemplate(emailTemplate{
		Badge:    "Premium Access",
		Title:    "Your premium secret code is ready",
		Subtitle: "Use this code to activate premium access on your account.",
		Greeting: fmt.Sprintf("Hi %s,", recipientName),
		Intro:    "An administrator has issued a premium activation code for you. Keep this code secure and redeem it in your account.",
		Details: []emailDetail{
			{Label: "Secret Code", Value: secretCode},
			{Label: "Valid Until", Value: validUntilLabel},
			{Label: "Usage Limit", Value: limitUsageLabel},
			{Label: "Issued At", Value: time.Now().Format("2006-01-02 15:04:05")},
		},
		Sections: sections,
		Actions: []emailAction{
			{Label: "Open Account", URL: redeemURL, Variant: "primary"},
			{Label: "Contact Support", URL: fmt.Sprintf("%s/support", frontendURL), Variant: "secondary"},
		},
		Notice:        "Security reminder: Lihatin team will never ask for this code via chat or phone.",
		FallbackURL:   redeemURL,
		FallbackLabel: redeemURL,
		FooterBaseURL: frontendURL,
	})

	textBody := fmt.Sprintf(`
LIHATIN - PREMIUM ACCESS CODE

Hi %s,

An administrator has issued a premium activation code for you.

Secret Code: %s
Valid Until: %s
Usage Limit: %s

Redeem steps:
1) Login to your Lihatin account
2) Open profile page
3) Paste this secret code in Redeem Premium Code section

%s
Security reminder:
- Keep this code private
- Lihatin will never ask this code via chat/phone

Account: %s
Support: %s/support

The Lihatin Team
`, recipientName, secretCode, validUntilLabel, limitUsageLabel, renderNoteText(cleanNote), redeemURL, frontendURL)

	return es.sendEmail(toEmail, subject, textBody, htmlBody)
}

func renderNoteText(note string) string {
	if strings.TrimSpace(note) == "" {
		return ""
	}
	return fmt.Sprintf("Admin message: %s\n", note)
}

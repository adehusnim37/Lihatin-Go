package mail

import (
	"fmt"

	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
)

// SendInformationShortCreate sends email notification when a new short link is created.
func (es *EmailService) SendInformationShortCreate(toEmail, toName, url, title, expiresAt, createdAt, passcode, urlOrigin, shortCode string) error {
	subject := fmt.Sprintf("Short Link Created: %s - Lihatin", title)
	frontendURL := config.GetEnvOrDefault(config.EnvFrontendURL, "http://localhost:3000")

	htmlBody := renderEmailTemplate(emailTemplate{
		Badge:    "Links",
		Title:    "Short link created",
		Subtitle: "Your new short link is ready to use.",
		Greeting: fmt.Sprintf("Hi %s,", toName),
		Intro:    "Your link was created successfully. Here are the details:",
		Details: []emailDetail{
			{Label: "Title", Value: title},
			{Label: "Short Code", Value: shortCode},
			{Label: "Created", Value: createdAt},
			{Label: "Expires", Value: expiresAt},
		},
		Sections: []string{
			renderParagraphSection("Original URL", urlOrigin),
			fmt.Sprintf(`<div class="section"><h3>Short URL</h3><p><a href="%s">%s</a></p></div>`, url, url),
			renderPasscodeSection(passcode),
			renderListSection("Usage tips", []string{
				"Share only with intended recipients",
				"Review performance from dashboard analytics",
				"Monitor expiration to avoid broken links",
			}),
		},
		Actions: []emailAction{
			{Label: "View Dashboard", URL: fmt.Sprintf("%s/main", frontendURL), Variant: "primary"},
			{Label: "Manage Links", URL: fmt.Sprintf("%s/main/links/%s", frontendURL, shortCode), Variant: "secondary"},
		},
		FallbackURL:   url,
		FallbackLabel: url,
		FooterBaseURL: frontendURL,
	})

	textBody := fmt.Sprintf(`
LIHATIN - SHORT LINK CREATED SUCCESSFULLY

Hi %s,

Your short link has been created successfully.

Title: %s
Short Code: %s
Short URL: %s
Original URL: %s
Created: %s
Expires: %s
%s

Dashboard: %s/main
Manage Links: %s/main/links/%s
Support: %s/support

The Lihatin Team
`, toName, title, shortCode, url, urlOrigin, createdAt, expiresAt,
		renderPasscodeText(passcode),
		frontendURL, frontendURL, shortCode, frontendURL)

	return es.sendEmail(toEmail, subject, textBody, htmlBody)
}

func renderPasscodeSection(passcode string) string {
	if passcode == "" || passcode == "-" {
		return ""
	}

	return fmt.Sprintf(`<div class="section"><h3>Protected Link Passcode</h3><p><strong>%s</strong></p><p style="margin-top:8px;">Keep this passcode secure. Anyone with this passcode can access the protected link.</p></div>`, passcode)
}

func renderPasscodeText(passcode string) string {
	if passcode == "" || passcode == "-" {
		return ""
	}

	return fmt.Sprintf("\nPasscode: %s\nKeep this passcode secure.", passcode)
}

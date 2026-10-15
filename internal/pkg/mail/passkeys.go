package mail

import (
	"fmt"
	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
)

func (es *EmailService) SendPasskeyChangedEmail(email, name, action, passkey string) error {
	url := config.GetEnvOrDefault(config.EnvFrontendURL, "http://localhost:3000")
	subject := fmt.Sprintf("Security update: passkey %s - Lihat.in", action)
	text := fmt.Sprintf("Hi %s,\n\nA passkey named %q was %s on your account.\nReview your security settings: %s/profile/security\nIf this was not you, review your sessions and contact support immediately.\n", name, passkey, action, url)
	html := renderEmailTemplate(emailTemplate{Badge: "Account security", Title: fmt.Sprintf("Passkey %s", action), Greeting: fmt.Sprintf("Hi %s,", name), Intro: fmt.Sprintf("A passkey named %q was %s on your account.", passkey, action), Actions: []emailAction{{Label: "Review security", URL: url + "/profile/security", Variant: "primary"}}, Notice: "If this was not you, review your sessions and contact support immediately.", FooterBaseURL: url})
	return es.sendEmail(email, subject, text, html)
}

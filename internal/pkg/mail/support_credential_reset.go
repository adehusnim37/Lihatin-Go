package mail

import (
	"errors"
	"fmt"
	"time"

	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
)

// SendSupportCredentialResetEmail must not acknowledge a skipped notification.
func (es *EmailService) SendSupportCredentialResetEmail(to, name, ticket, method string, resetAt time.Time) error {
	if es.config.SMTPUsername == "" || es.config.SMTPPassword == "" {
		return errors.New("SMTP credentials are required for recovery security notifications")
	}
	label := "authenticator and recovery codes"
	if method == "passkeys" {
		label = "all passkeys"
	}
	if method == "all" {
		label = "authenticator, recovery codes, and all passkeys"
	}
	base := config.GetEnvOrDefault(config.EnvFrontendURL, "http://localhost:3000")
	when := resetAt.UTC().Format(time.RFC1123)
	subject := "Security alert: Support reset your sign-in protection - Lihatin"
	text := fmt.Sprintf("Hi %s,\n\nAn administrator approved recovery for ticket %s and removed your %s on %s. All existing sessions were signed out.\n\nSign in using your remaining password or Google sign-in and register a new authenticator or passkey in Security Settings. Your password was not changed.\n\nIf you did not request this, contact support immediately and change your password.\nSupport: %s/support\n\nLihatin will never ask for your password or verification codes by email.", name, ticket, label, when, base)
	html := renderEmailTemplate(emailTemplate{
		Badge: "Security Alert", Title: "Your sign-in protection was reset", Greeting: fmt.Sprintf("Hi %s,", name), Intro: "An administrator approved your account recovery request. All existing sessions were signed out.",
		Details:  []emailDetail{{Label: "Support ticket", Value: ticket}, {Label: "Removed", Value: label}, {Label: "Time (UTC)", Value: when}},
		Sections: []string{renderListSection("What to do next", []string{"Sign in with your remaining password or Google sign-in.", "Register a new authenticator or passkey in Security Settings.", "If you did not request this, contact support immediately and change your password."})},
		Actions:  []emailAction{{Label: "Contact Support", URL: base + "/support", Variant: "danger"}}, Notice: "Lihatin will never ask for your password or verification codes by email.", FooterBaseURL: base,
	})
	return es.sendEmail(to, subject, text, html)
}

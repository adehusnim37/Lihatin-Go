package mail

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
)

// SendSupportRecoveryOTPEmail links to the existing protected conversation.
// Neither a challenge token nor the OTP is placed in the URL.
func (es *EmailService) SendSupportRecoveryOTPEmail(to, ticket, code string) error {
	if es.config.SMTPUsername == "" || es.config.SMTPPassword == "" {
		return errors.New("SMTP credentials are required for recovery verification")
	}
	base, err := normalizeSupportFrontendURL(config.GetEnvOrDefault(config.EnvFrontendURL, "http://localhost:3000"))
	if err != nil {
		return err
	}
	parsed, err := url.Parse(base)
	if err != nil {
		return err
	}
	if strings.EqualFold(config.GetEnvOrDefault(config.Env, "development"), "production") && parsed.Scheme != "https" {
		return errors.New("recovery verification requires an HTTPS frontend URL")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/support/ticket/" + url.PathEscape(ticket)
	link := parsed.String()
	subject := fmt.Sprintf("Verify account recovery for %s - Lihatin", ticket)
	html := renderEmailTemplate(emailTemplate{
		Badge: "Account Recovery", Title: "Verify your recovery request", Greeting: "Hi there,",
		Intro:    "Support requested email verification for your Lost 2FA ticket. Open your protected ticket conversation and enter this code there.",
		Details:  []emailDetail{{Label: "Ticket", Value: ticket}, {Label: "Verification code", Value: code}, {Label: "Expires in", Value: "10 minutes"}},
		Sections: []string{renderListSection("Before you continue", []string{"Use your ticket access code if you need to reopen the conversation.", "Enter this verification code only in the recovery form inside your ticket.", "Never send this code to support staff or post it in a message.", "Email verification does not remove your authenticator or passkeys. An administrator must review and approve recovery separately.", "If you did not request recovery, ignore this code and contact support."})},
		Actions:  []emailAction{{Label: "Open support ticket", URL: link}}, Notice: "This code is single use. Lihatin staff will never ask you to share it.", FooterBaseURL: base,
	})
	text := fmt.Sprintf("LIHATIN - ACCOUNT RECOVERY VERIFICATION\n\nTicket: %s\nVerification code: %s\nExpires in: 10 minutes\n\nOpen your protected ticket: %s\nUse your ticket access code if needed, then enter this code in the recovery form. Never send it to staff or post it in chat.\n\nVerification does not reset credentials. Admin review and explicit approval are still required. If you did not request recovery, ignore this code and contact support.", ticket, code, link)
	return es.sendEmail(to, subject, text, html)
}

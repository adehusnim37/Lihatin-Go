package mail

import (
	"fmt"
	"strings"

	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
)

func (es *EmailService) SendSupportAccessOTPEmail(toEmail, ticketCode, otpCode string) error {
	baseURL := strings.TrimRight(config.GetEnvOrDefault(config.EnvFrontendURL, "http://localhost:3000"), "/")
	subject := fmt.Sprintf("Support Access Code for %s - Lihatin", ticketCode)

	htmlBody := renderEmailTemplate(emailTemplate{
		Badge:    "Support Access",
		Title:    "Verification code for support ticket",
		Subtitle: "Use this code to open secure ticket detail page.",
		Greeting: "Hi there,",
		Intro:    "We received a request to open your support ticket conversation.",
		Details: []emailDetail{
			{Label: "Ticket Code", Value: ticketCode},
			{Label: "OTP Code", Value: otpCode},
			{Label: "Valid for", Value: "10 minutes"},
		},
		Sections: []string{
			renderListSection("Security reminders", []string{
				"Never share this OTP with anyone",
				"Lihatin support will never ask for this OTP",
				"Ignore this email if this request is not from you",
			}),
		},
		Notice:        "This OTP expires in 10 minutes and can only be used once.",
		FooterBaseURL: baseURL,
	})

	textBody := fmt.Sprintf(`
LIHATIN - SUPPORT ACCESS OTP

Ticket Code: %s
OTP Code: %s

This code expires in 10 minutes.
Do not share this code with anyone.
`, ticketCode, otpCode)

	return es.sendEmail(toEmail, subject, textBody, htmlBody)
}

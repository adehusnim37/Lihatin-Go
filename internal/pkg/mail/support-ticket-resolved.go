package mail

import (
	"fmt"
	"strings"

	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
)

func (es *EmailService) SendSupportTicketResolvedEmail(toEmail, ticketCode, resolutionMessage string) error {
	baseURL := strings.TrimRight(config.GetEnvOrDefault(config.EnvFrontendURL, "http://localhost:3000"), "/")
	message := strings.TrimSpace(resolutionMessage)
	if message == "" {
		message = "Your support request has been resolved. Please login again."
	}

	loginURL := fmt.Sprintf("%s/auth/login", baseURL)
	trackURL := fmt.Sprintf("%s/support?ticket=%s&email=%s", baseURL, ticketCode, toEmail)

	htmlBody := renderEmailTemplate(emailTemplate{
		Badge:    "Support",
		Title:    "Support ticket resolved",
		Subtitle: "Your support request has been marked as resolved.",
		Greeting: "Hi there,",
		Intro:    "Your issue has been reviewed by Lihatin support team.",
		Details: []emailDetail{
			{Label: "Ticket Code", Value: ticketCode},
			{Label: "Status", Value: "Resolved"},
		},
		Sections: []string{
			renderParagraphSection("Resolution Message", message),
			renderParagraphSection("Next Step", "Please try signing in again. If issue persists, reply by opening a new support ticket."),
		},
		Actions: []emailAction{
			{Label: "Login Again", URL: loginURL, Variant: "primary"},
			{Label: "Track Ticket", URL: trackURL, Variant: "secondary"},
		},
		Notice:        "If your issue is still unresolved, submit another support ticket with latest details.",
		FallbackURL:   loginURL,
		FallbackLabel: loginURL,
		FooterBaseURL: baseURL,
	})

	textBody := fmt.Sprintf(`
LIHATIN - SUPPORT TICKET RESOLVED

Ticket Code: %s
Status: Resolved

Resolution message:
%s

Login again:
%s

Track ticket:
%s
`, ticketCode, message, loginURL, trackURL)

	return es.sendEmail(toEmail, fmt.Sprintf("Support Ticket %s Resolved - Lihatin", ticketCode), textBody, htmlBody)
}

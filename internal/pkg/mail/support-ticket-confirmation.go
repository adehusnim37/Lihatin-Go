package mail

import (
	"fmt"
	"strings"
	"time"

	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
)

func (es *EmailService) SendSupportTicketConfirmationEmail(
	toEmail,
	toName,
	ticketCode,
	categoryLabel,
	frontendURL string,
	createdAt time.Time,
) error {
	baseURL := strings.TrimSpace(frontendURL)
	if baseURL == "" {
		baseURL = config.GetEnvOrDefault(config.EnvFrontendURL, "http://localhost:3000")
	}
	baseURL = strings.TrimRight(baseURL, "/")

	recipientName := strings.TrimSpace(toName)
	if recipientName == "" {
		recipientName = "there"
	}

	trackURL := fmt.Sprintf("%s/support?ticket=%s&email=%s", baseURL, ticketCode, toEmail)

	htmlBody := renderEmailTemplate(emailTemplate{
		Badge:    "Support",
		Title:    "Support ticket received",
		Subtitle: "We have received your request and queued it for review.",
		Greeting: fmt.Sprintf("Hi %s,", recipientName),
		Intro:    "Thank you for contacting Lihatin support. Keep this ticket code for tracking.",
		Details: []emailDetail{
			{Label: "Ticket Code", Value: ticketCode},
			{Label: "Category", Value: categoryLabel},
			{Label: "Submitted At", Value: createdAt.Local().Format("2006-01-02 15:04:05")},
			{Label: "Estimated Response", Value: "Within 24 hours (business day)"},
		},
		Sections: []string{
			renderListSection("What to expect", []string{
				"Support team reviews your ticket details.",
				"You may be asked for additional verification information.",
				"You will receive resolution update by email.",
			}),
		},
		Actions: []emailAction{
			{Label: "Track Ticket", URL: trackURL, Variant: "primary"},
			{Label: "Login", URL: fmt.Sprintf("%s/auth/login", baseURL), Variant: "secondary"},
		},
		Notice:        "Do not share private credentials or OTP codes in support descriptions.",
		FallbackURL:   trackURL,
		FallbackLabel: trackURL,
		FooterBaseURL: baseURL,
	})

	textBody := fmt.Sprintf(`
LIHATIN - SUPPORT TICKET RECEIVED

Hi %s,

We received your support request.

Ticket Code: %s
Category: %s
Submitted At: %s
Estimated Response: Within 24 hours (business day)

Track ticket:
%s

Thanks,
Lihatin Support Team
`, recipientName, ticketCode, categoryLabel, createdAt.Local().Format("2006-01-02 15:04:05"), trackURL)

	return es.sendEmail(toEmail, fmt.Sprintf("Support Ticket %s Received - Lihatin", ticketCode), textBody, htmlBody)
}

func (es *EmailService) SendSupportTicketAdminAlertEmail(ticketCode, fromEmail, subject, category, frontendURL string) error {
	recipientsRaw := strings.TrimSpace(config.GetEnvOrDefault(config.EnvSupportAlertEmails, ""))
	if recipientsRaw == "" {
		return nil
	}

	baseURL := strings.TrimSpace(frontendURL)
	if baseURL == "" {
		baseURL = config.GetEnvOrDefault(config.EnvFrontendURL, "http://localhost:3000")
	}
	baseURL = strings.TrimRight(baseURL, "/")

	adminRecipients := strings.Split(recipientsRaw, ",")
	cleanRecipients := make([]string, 0, len(adminRecipients))
	for _, item := range adminRecipients {
		email := strings.TrimSpace(item)
		if email != "" {
			cleanRecipients = append(cleanRecipients, email)
		}
	}
	if len(cleanRecipients) == 0 {
		return nil
	}

	textBody := fmt.Sprintf(`
NEW SUPPORT TICKET

Ticket Code: %s
From: %s
Category: %s
Subject: %s

Open admin panel:
%s/main/admin/support-tickets
`, ticketCode, fromEmail, category, subject, baseURL)

	htmlBody := renderEmailTemplate(emailTemplate{
		Badge:    "Support Admin",
		Title:    "New support ticket submitted",
		Subtitle: "A new support request needs review.",
		Greeting: "Hi Team,",
		Intro:    "A new support ticket was created in Lihatin.",
		Details: []emailDetail{
			{Label: "Ticket Code", Value: ticketCode},
			{Label: "From", Value: fromEmail},
			{Label: "Category", Value: category},
			{Label: "Subject", Value: subject},
		},
		Actions: []emailAction{
			{Label: "Open Support Admin", URL: fmt.Sprintf("%s/main/admin/support-tickets", baseURL), Variant: "primary"},
		},
		FallbackURL:   fmt.Sprintf("%s/main/admin/support-tickets", baseURL),
		FallbackLabel: fmt.Sprintf("%s/main/admin/support-tickets", baseURL),
		FooterBaseURL: baseURL,
	})

	subjectLine := fmt.Sprintf("[Support] New Ticket %s", ticketCode)
	for _, recipient := range cleanRecipients {
		if err := es.sendEmail(recipient, subjectLine, textBody, htmlBody); err != nil {
			return err
		}
	}

	return nil
}

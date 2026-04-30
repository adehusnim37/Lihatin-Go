package mail

import (
	"fmt"
	"strings"

	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
)

func (es *EmailService) SendSupportTicketUpdatedEmail(toEmail, ticketCode, status, adminNotes, frontendURL string) error {
	baseURL := strings.TrimSpace(frontendURL)
	if baseURL == "" {
		baseURL = config.GetEnvOrDefault(config.EnvFrontendURL, "http://localhost:3000")
	}
	baseURL = strings.TrimRight(baseURL, "/")

	statusLabel := formatSupportStatusLabel(status)
	trackURL := fmt.Sprintf("%s/support?ticket=%s&email=%s", baseURL, ticketCode, toEmail)
	note := strings.TrimSpace(adminNotes)
	if note == "" {
		note = "Support team updated your ticket status."
	}

	htmlBody := renderEmailTemplate(emailTemplate{
		Badge:    "Support",
		Title:    "Support ticket updated",
		Subtitle: "There is a new update on your support request.",
		Greeting: "Hi there,",
		Intro:    "Your support ticket has been updated by support team.",
		Details: []emailDetail{
			{Label: "Ticket Code", Value: ticketCode},
			{Label: "Latest Status", Value: statusLabel},
		},
		Sections: []string{
			renderParagraphSection("Update Summary", note),
		},
		Actions: []emailAction{
			{Label: "Open Ticket", URL: trackURL, Variant: "primary"},
		},
		FallbackURL:   trackURL,
		FallbackLabel: trackURL,
		FooterBaseURL: baseURL,
	})

	textBody := fmt.Sprintf(`
LIHATIN - SUPPORT TICKET UPDATED

Ticket Code: %s
Status: %s

Update Summary:
%s

Open ticket:
%s
`, ticketCode, statusLabel, note, trackURL)

	return es.sendEmail(toEmail, fmt.Sprintf("Support Ticket %s Updated - Lihatin", ticketCode), textBody, htmlBody)
}

func (es *EmailService) SendSupportTicketMessageToRequesterEmail(toEmail, ticketCode, senderLabel, messagePreview, frontendURL string) error {
	baseURL := strings.TrimSpace(frontendURL)
	if baseURL == "" {
		baseURL = config.GetEnvOrDefault(config.EnvFrontendURL, "http://localhost:3000")
	}
	baseURL = strings.TrimRight(baseURL, "/")

	trackURL := fmt.Sprintf("%s/support?ticket=%s&email=%s", baseURL, ticketCode, toEmail)
	preview := strings.TrimSpace(messagePreview)
	if preview == "" {
		preview = "You have a new message on your support ticket."
	}
	if senderLabel == "" {
		senderLabel = "Support Team"
	}

	htmlBody := renderEmailTemplate(emailTemplate{
		Badge:    "Support",
		Title:    "New support message",
		Subtitle: "Support team sent a new message on your ticket.",
		Greeting: "Hi there,",
		Intro:    "There is a new message in your support conversation.",
		Details: []emailDetail{
			{Label: "Ticket Code", Value: ticketCode},
			{Label: "Sender", Value: strings.TrimSpace(senderLabel)},
		},
		Sections: []string{
			renderParagraphSection("Message Preview", preview),
		},
		Actions: []emailAction{
			{Label: "Open Conversation", URL: trackURL, Variant: "primary"},
		},
		FallbackURL:   trackURL,
		FallbackLabel: trackURL,
		FooterBaseURL: baseURL,
	})

	textBody := fmt.Sprintf(`
LIHATIN - NEW SUPPORT MESSAGE

Ticket Code: %s
Sender: %s

Message Preview:
%s

Open conversation:
%s
`, ticketCode, senderLabel, preview, trackURL)

	return es.sendEmail(toEmail, fmt.Sprintf("New Message on Support Ticket %s - Lihatin", ticketCode), textBody, htmlBody)
}

func (es *EmailService) SendSupportTicketMessageToAdminEmail(ticketCode, fromEmail, senderLabel, messagePreview, frontendURL string) error {
	recipientsRaw := strings.TrimSpace(config.GetEnvOrDefault(config.EnvSupportAlertEmails, ""))
	if recipientsRaw == "" {
		return nil
	}

	baseURL := strings.TrimSpace(frontendURL)
	if baseURL == "" {
		baseURL = config.GetEnvOrDefault(config.EnvFrontendURL, "http://localhost:3000")
	}
	baseURL = strings.TrimRight(baseURL, "/")

	adminURL := fmt.Sprintf("%s/main/admin/support-tickets", baseURL)
	preview := strings.TrimSpace(messagePreview)
	if preview == "" {
		preview = "New message posted on support ticket."
	}
	if senderLabel == "" {
		senderLabel = "User"
	}

	htmlBody := renderEmailTemplate(emailTemplate{
		Badge:    "Support Admin",
		Title:    "New support ticket message",
		Subtitle: "Customer posted a new message in support conversation.",
		Greeting: "Hi Team,",
		Intro:    "A support conversation has new activity.",
		Details: []emailDetail{
			{Label: "Ticket Code", Value: ticketCode},
			{Label: "From", Value: fromEmail},
			{Label: "Sender Type", Value: senderLabel},
		},
		Sections: []string{
			renderParagraphSection("Message Preview", preview),
		},
		Actions: []emailAction{
			{Label: "Open Support Admin", URL: adminURL, Variant: "primary"},
		},
		FallbackURL:   adminURL,
		FallbackLabel: adminURL,
		FooterBaseURL: baseURL,
	})

	textBody := fmt.Sprintf(`
NEW SUPPORT MESSAGE

Ticket Code: %s
From: %s
Sender Type: %s

Message Preview:
%s

Open support admin:
%s
`, ticketCode, fromEmail, senderLabel, preview, adminURL)

	subject := fmt.Sprintf("[Support] New Message %s", ticketCode)
	recipients := strings.Split(recipientsRaw, ",")
	for _, recipient := range recipients {
		email := strings.TrimSpace(recipient)
		if email == "" {
			continue
		}
		if err := es.sendEmail(email, subject, textBody, htmlBody); err != nil {
			return err
		}
	}

	return nil
}

func formatSupportStatusLabel(status string) string {
	value := strings.TrimSpace(strings.ToLower(status))
	if value == "" {
		return "Updated"
	}
	parts := strings.Split(value, "_")
	for i := range parts {
		if parts[i] == "" {
			continue
		}
		parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
	}
	return strings.Join(parts, " ")
}

package mail

import (
	"fmt"
	"strings"
	"time"

	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
)

// SendPremiumAccessRevokedEmail sends premium revoke notification.
func (es *EmailService) SendPremiumAccessRevokedEmail(
	toEmail,
	toName,
	reason,
	revokeType string,
	revokedAt time.Time,
) error {
	frontendURL := config.GetEnvOrDefault(config.EnvFrontendURL, "http://localhost:3000")
	recipientName := sanitizeRecipientName(toName)
	cleanReason := strings.TrimSpace(reason)
	if cleanReason == "" {
		cleanReason = "No additional reason provided by administrator."
	}

	normalizedRevokeType := strings.ToLower(strings.TrimSpace(revokeType))
	if normalizedRevokeType == "" {
		normalizedRevokeType = "temporary"
	}

	revokeTypeLabel := "Temporary"
	if normalizedRevokeType == "permanent" {
		revokeTypeLabel = "Permanent"
	}

	notice := "If this action is unexpected, contact support immediately."
	if normalizedRevokeType == "permanent" {
		notice = "This revoke is marked as permanent. Reactivation requires super administrator override."
	}

	actions := []emailAction{
		{Label: "Contact Support", URL: fmt.Sprintf("%s/support", frontendURL), Variant: "primary"},
		{Label: "Open Dashboard", URL: fmt.Sprintf("%s/main", frontendURL), Variant: "secondary"},
	}

	htmlBody := renderEmailTemplate(emailTemplate{
		Badge:    "Account Security",
		Title:    "Premium access has been revoked",
		Subtitle: "Your account remains active as regular user access.",
		Greeting: fmt.Sprintf("Hi %s,", recipientName),
		Intro:    "An administrator has revoked your premium access. Your account role and features have been adjusted based on this action.",
		Details: []emailDetail{
			{Label: "Status", Value: "Revoked"},
			{Label: "Revoke Type", Value: revokeTypeLabel},
			{Label: "Revoked At", Value: revokedAt.Local().Format("2006-01-02 15:04:05")},
			{Label: "Reason", Value: cleanReason},
		},
		Sections: []string{
			renderParagraphSection("What changed", "Premium-only features are disabled and account role is now regular user."),
			renderParagraphSection("Need assistance?", "Reach support team if you need clarification or appeal."),
		},
		Actions:       actions,
		Notice:        notice,
		FallbackURL:   fmt.Sprintf("%s/support", frontendURL),
		FallbackLabel: fmt.Sprintf("%s/support", frontendURL),
		FooterBaseURL: frontendURL,
	})

	textBody := fmt.Sprintf(`
LIHATIN - PREMIUM ACCESS REVOKED

Hi %s,

An administrator revoked your premium access.

Status: Revoked
Revoke Type: %s
Revoked At: %s
Reason: %s

What changed:
- Premium features disabled
- Account role switched to regular user

If unexpected, contact support:
%s/support

The Lihatin Team
`, recipientName, revokeTypeLabel, revokedAt.Local().Format("2006-01-02 15:04:05"), cleanReason, frontendURL)

	return es.sendEmail(toEmail, "Premium Access Revoked - Lihatin", textBody, htmlBody)
}

// SendPremiumAccessReactivatedEmail sends premium reactivation notification.
func (es *EmailService) SendPremiumAccessReactivatedEmail(
	toEmail,
	toName,
	reason string,
	reactivatedAt time.Time,
) error {
	frontendURL := config.GetEnvOrDefault(config.EnvFrontendURL, "http://localhost:3000")
	recipientName := sanitizeRecipientName(toName)
	cleanReason := strings.TrimSpace(reason)
	if cleanReason == "" {
		cleanReason = "No additional reason provided by administrator."
	}

	htmlBody := renderEmailTemplate(emailTemplate{
		Badge:    "Account Update",
		Title:    "Premium access has been reactivated",
		Subtitle: "Your premium features are now active again.",
		Greeting: fmt.Sprintf("Hi %s,", recipientName),
		Intro:    "An administrator has reactivated premium access for your account.",
		Details: []emailDetail{
			{Label: "Status", Value: "Active"},
			{Label: "Reactivated At", Value: reactivatedAt.Local().Format("2006-01-02 15:04:05")},
			{Label: "Reason", Value: cleanReason},
		},
		Sections: []string{
			renderParagraphSection("Next step", "Sign in and use premium features as usual."),
		},
		Actions: []emailAction{
			{Label: "Open Dashboard", URL: fmt.Sprintf("%s/main", frontendURL), Variant: "primary"},
			{Label: "View Profile", URL: fmt.Sprintf("%s/profile/me", frontendURL), Variant: "secondary"},
		},
		Notice:        "If you did not request this, contact support immediately.",
		FallbackURL:   fmt.Sprintf("%s/main", frontendURL),
		FallbackLabel: fmt.Sprintf("%s/main", frontendURL),
		FooterBaseURL: frontendURL,
	})

	textBody := fmt.Sprintf(`
LIHATIN - PREMIUM ACCESS REACTIVATED

Hi %s,

An administrator reactivated your premium access.

Status: Active
Reactivated At: %s
Reason: %s

Open dashboard:
%s/main

If unexpected, contact support:
%s/support

The Lihatin Team
`, recipientName, reactivatedAt.Local().Format("2006-01-02 15:04:05"), cleanReason, frontendURL, frontendURL)

	return es.sendEmail(toEmail, "Premium Access Reactivated - Lihatin", textBody, htmlBody)
}

func sanitizeRecipientName(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "there"
	}
	return trimmed
}

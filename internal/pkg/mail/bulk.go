package mail

import (
	"fmt"
	"strings"

	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
	"github.com/adehusnim37/lihatin-go/models/shortlink"
)

// SendBulkCreationSummary sends email notification for bulk short link creation.
func (es *EmailService) SendBulkCreationSummary(toEmail, toName string, links []shortlink.ShortLink, details []shortlink.ShortLinkDetail) error {
	linkCount := len(links)
	subject := fmt.Sprintf("Bulk Short Links Created: %d Links - Lihatin", linkCount)
	frontendURL := config.GetEnvOrDefault(config.EnvFrontendURL, "http://localhost:3000")

	linksTableHTML := buildLinksTableHTML(links, details)
	linksTableText := buildLinksTableText(links, details)
	protectedCount := countProtectedLinks(details)

	htmlBody := renderEmailTemplate(emailTemplate{
		Badge:    "Links",
		Title:    "Bulk links created",
		Subtitle: fmt.Sprintf("%d short links are ready.", linkCount),
		Greeting: fmt.Sprintf("Hi %s,", toName),
		Intro:    "Your bulk creation request has completed successfully.",
		Details: []emailDetail{
			{Label: "Total Links", Value: fmt.Sprintf("%d", linkCount)},
			{Label: "Protected", Value: fmt.Sprintf("%d", protectedCount)},
			{Label: "Public", Value: fmt.Sprintf("%d", linkCount-protectedCount)},
		},
		Sections: []string{
			fmt.Sprintf(`<div class="section"><h3>Created links</h3>%s</div>`, linksTableHTML),
			renderProtectedLinksWarningHTML(protectedCount),
			renderListSection("Bulk management tips", []string{
				"Review links from the dashboard",
				"Keep passcodes secure for protected links",
				"Track analytics to measure performance",
			}),
		},
		Actions: []emailAction{
			{Label: "View Dashboard", URL: fmt.Sprintf("%s/main", frontendURL), Variant: "primary"},
			{Label: "Manage Links", URL: fmt.Sprintf("%s/main/links/%s", frontendURL, links[0].ShortCode), Variant: "secondary"},
			{Label: "Analytics", URL: fmt.Sprintf("%s/main/analytics/%s", frontendURL, links[0].ShortCode), Variant: "secondary"},
		},
		Notice:        "Use dashboard filters to quickly locate links from this bulk operation.",
		FooterBaseURL: frontendURL,
	})

	textBody := fmt.Sprintf(`
LIHATIN - BULK SHORT LINKS CREATED

Hi %s,

Your bulk creation finished successfully.

Total Links: %d
Protected: %d
Public: %d

Created links:
%s
%s

Dashboard: %s/main
Manage Links: %s/main/links
Analytics: %s/main/analytics
Support: %s/support

The Lihatin Team
`,
		toName,
		linkCount,
		protectedCount,
		linkCount-protectedCount,
		linksTableText,
		renderProtectedLinksWarningText(protectedCount),
		frontendURL,
		frontendURL,
		frontendURL,
		frontendURL,
	)

	return es.sendEmail(toEmail, subject, textBody, htmlBody)
}

func countProtectedLinks(details []shortlink.ShortLinkDetail) int {
	count := 0
	for _, detail := range details {
		if detail.Passcode != 0 {
			count++
		}
	}
	return count
}

// buildLinksTableHTML creates HTML table of created links.
func buildLinksTableHTML(links []shortlink.ShortLink, details []shortlink.ShortLinkDetail) string {
	if len(links) == 0 {
		return "<p style=\"margin:0;\">No links created.</p>"
	}

	detailMap := make(map[string]shortlink.ShortLinkDetail)
	for _, detail := range details {
		detailMap[detail.ShortLinkID] = detail
	}

	backendURL := config.GetEnvOrDefault(config.EnvBackendURL, "http://localhost:8080")
	var rows strings.Builder

	for i, link := range links {
		if i >= 20 {
			rows.WriteString(fmt.Sprintf(`<tr><td colspan="4" style="padding:10px;border-top:1px solid #e5eaf2;color:#64748b;">... and %d more links</td></tr>`, len(links)-20))
			break
		}

		detail := detailMap[link.ID]
		title := link.Title
		if title == "" {
			title = "Untitled Link"
		}
		if len(title) > 40 {
			title = title[:37] + "..."
		}

		displayURL := link.OriginalURL
		if len(displayURL) > 60 {
			displayURL = displayURL[:57] + "..."
		}

		shortURL := fmt.Sprintf("%s/s/%s", backendURL, link.ShortCode)
		passcode := "None"
		if detail.Passcode != 0 {
			passcode = fmt.Sprintf("%d", detail.Passcode)
		}

		rows.WriteString(fmt.Sprintf(`<tr>
<td style="padding:10px;border-top:1px solid #e5eaf2;vertical-align:top;"><strong>%s</strong><br><span style="color:#64748b;font-size:12px;">%s</span></td>
<td style="padding:10px;border-top:1px solid #e5eaf2;vertical-align:top;"><code>%s</code><br><a href="%s" style="font-size:12px;color:#2563eb;text-decoration:none;">%s</a></td>
<td style="padding:10px;border-top:1px solid #e5eaf2;vertical-align:top;word-break:break-word;">%s</td>
<td style="padding:10px;border-top:1px solid #e5eaf2;vertical-align:top;">%s</td>
</tr>`, title, link.CreatedAt.Format("Jan 2, 15:04"), link.ShortCode, shortURL, shortURL, displayURL, passcode))
	}

	return fmt.Sprintf(`<table role="presentation" style="width:100%%;border-collapse:collapse;font-size:13px;">
<thead>
<tr>
<th style="text-align:left;padding:10px;background:#eef4ff;border-bottom:1px solid #dbe7ff;">Title</th>
<th style="text-align:left;padding:10px;background:#eef4ff;border-bottom:1px solid #dbe7ff;">Short URL</th>
<th style="text-align:left;padding:10px;background:#eef4ff;border-bottom:1px solid #dbe7ff;">Original URL</th>
<th style="text-align:left;padding:10px;background:#eef4ff;border-bottom:1px solid #dbe7ff;">Passcode</th>
</tr>
</thead>
<tbody>%s</tbody>
</table>`, rows.String())
}

// buildLinksTableText creates text version of links table.
func buildLinksTableText(links []shortlink.ShortLink, details []shortlink.ShortLinkDetail) string {
	if len(links) == 0 {
		return "No links created."
	}

	detailMap := make(map[string]shortlink.ShortLinkDetail)
	for _, detail := range details {
		detailMap[detail.ShortLinkID] = detail
	}

	backendURL := config.GetEnvOrDefault(config.EnvBackendURL, "http://localhost:8080")
	var result strings.Builder

	for i, link := range links {
		if i >= 10 {
			result.WriteString(fmt.Sprintf("\n... and %d more links\n", len(links)-10))
			break
		}

		detail := detailMap[link.ID]
		title := link.Title
		if title == "" {
			title = "Untitled Link"
		}

		passcode := "None"
		if detail.Passcode != 0 {
			passcode = fmt.Sprintf("%d", detail.Passcode)
		}

		result.WriteString(fmt.Sprintf(`
%d. %s
   Short URL: %s/s/%s
   Original: %s
   Passcode: %s
   Created: %s
`, i+1, title, backendURL, link.ShortCode, link.OriginalURL, passcode, link.CreatedAt.Format("Jan 2, 2006 15:04")))
	}

	return result.String()
}

func renderProtectedLinksWarningHTML(protectedCount int) string {
	if protectedCount == 0 {
		return ""
	}

	return fmt.Sprintf(`<div class="section"><h3>Protected links notice</h3><p>You created %d protected link(s). Keep passcodes secure and share only with authorized users.</p></div>`, protectedCount)
}

func renderProtectedLinksWarningText(protectedCount int) string {
	if protectedCount == 0 {
		return ""
	}

	return fmt.Sprintf("\nProtected links notice: You created %d protected link(s). Keep passcodes secure.", protectedCount)
}

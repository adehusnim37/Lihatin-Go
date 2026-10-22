package mail

import (
	"fmt"
	"time"
)

type WeeklySummaryEmailData struct {
	ToEmail                string
	UserName               string
	PeriodStart            time.Time
	PeriodEnd              time.Time
	LinksCreated           int64
	TotalClicks            int64
	UniqueVisitors         int64
	TopLinkTitle           string
	TopLinkShortCode       string
	TopLinkClicks          int64
	BaseURL                string
	DashboardURL           string
	PreferencesURL         string
	UnsubscribeURL         string
	OneClickUnsubscribeURL string
}

func (es *EmailService) SendWeeklySummaryEmail(data WeeklySummaryEmailData) error {
	details := []emailDetail{
		{Label: "Reporting period", Value: fmt.Sprintf("%s – %s", data.PeriodStart.Format("Jan 2"), data.PeriodEnd.Add(-time.Second).Format("Jan 2, 2006"))},
		{Label: "Links created", Value: fmt.Sprintf("%d", data.LinksCreated)},
		{Label: "Total clicks", Value: fmt.Sprintf("%d", data.TotalClicks)},
		{Label: "Unique visitors", Value: fmt.Sprintf("%d", data.UniqueVisitors)},
	}

	sections := make([]string, 0, 1)
	if data.TopLinkShortCode != "" {
		title := data.TopLinkTitle
		if title == "" {
			title = data.TopLinkShortCode
		}
		sections = append(sections, renderParagraphSection(
			"Top-performing link",
			fmt.Sprintf("%s received %d clicks.", title, data.TopLinkClicks),
		))
	} else {
		sections = append(sections, renderParagraphSection(
			"No clicks this week",
			"Share your links to start collecting performance insights.",
		))
	}

	htmlBody := renderEmailTemplate(emailTemplate{
		Badge:                "Weekly summary",
		Title:                "Your links this week",
		Subtitle:             "A quick look at your Lihatin performance.",
		Greeting:             fmt.Sprintf("Hi %s,", data.UserName),
		Intro:                "Here is your activity summary for the previous week.",
		Details:              details,
		Sections:             sections,
		Actions:              []emailAction{{Label: "View analytics", URL: data.DashboardURL, Variant: "primary"}},
		FooterBaseURL:        data.BaseURL,
		FooterPreferencesURL: data.PreferencesURL,
		FooterUnsubscribeURL: data.UnsubscribeURL,
	})

	textBody := fmt.Sprintf(`LIHATIN - WEEKLY SUMMARY

Hi %s,

Reporting period: %s - %s
Links created: %d
Total clicks: %d
Unique visitors: %d

View analytics: %s
Email preferences: %s
Unsubscribe from weekly summaries: %s
`,
		data.UserName,
		data.PeriodStart.Format("Jan 2, 2006"),
		data.PeriodEnd.Add(-time.Second).Format("Jan 2, 2006"),
		data.LinksCreated,
		data.TotalClicks,
		data.UniqueVisitors,
		data.DashboardURL,
		data.PreferencesURL,
		data.UnsubscribeURL,
	)

	headers := map[string]string{
		"List-Unsubscribe":      fmt.Sprintf("<%s>", data.OneClickUnsubscribeURL),
		"List-Unsubscribe-Post": "List-Unsubscribe=One-Click",
	}
	return es.sendEmailWithHeaders(data.ToEmail, "Your weekly Lihatin summary", textBody, htmlBody, headers)
}

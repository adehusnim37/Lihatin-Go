package mail

import (
	"fmt"
	"html"
	"strings"
	"time"
)

type emailAction struct {
	Label   string
	URL     string
	Variant string // primary | secondary | danger
}

type emailDetail struct {
	Label string
	Value string
}

type emailTemplate struct {
	Badge         string
	Title         string
	Subtitle      string
	Greeting      string
	Intro         string
	Details       []emailDetail
	Sections      []string
	Actions       []emailAction
	Notice        string
	FallbackURL   string
	FallbackLabel string
	FooterBaseURL string
}

func renderEmailTemplate(t emailTemplate) string {
	year := time.Now().Year()
	badgeHTML := ""
	if t.Badge != "" {
		badgeHTML = fmt.Sprintf(`<div class="badge">%s</div>`, html.EscapeString(t.Badge))
	}

	detailsHTML := renderDetailsSection(t.Details)
	sectionsHTML := strings.Join(t.Sections, "\n")
	actionsHTML := renderActionsSection(t.Actions)

	noticeHTML := ""
	if t.Notice != "" {
		noticeHTML = fmt.Sprintf(`<div class="notice">%s</div>`, html.EscapeString(t.Notice))
	}

	fallbackHTML := ""
	if t.FallbackURL != "" {
		linkText := t.FallbackLabel
		if linkText == "" {
			linkText = t.FallbackURL
		}
		fallbackHTML = fmt.Sprintf(`
<div class="alt-link">
  If the button does not work, copy and paste this link:<br>
  <a href="%s">%s</a>
</div>`, html.EscapeString(t.FallbackURL), html.EscapeString(linkText))
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <style>
    body { margin: 0; padding: 0; background: #f4f7fb; color: #0f172a; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Arial, sans-serif; }
    .shell { padding: 28px 16px; }
    .container { max-width: 640px; margin: 0 auto; background: #ffffff; border: 1px solid #e5eaf2; border-radius: 14px; overflow: hidden; }
    .header { padding: 28px 28px 22px; border-bottom: 1px solid #edf1f7; }
    .badge { display: inline-block; margin-bottom: 14px; background: #e9f2fc; color: #2f5f8c; border: 1px solid #d4e6f7; border-radius: 999px; font-size: 12px; font-weight: 600; padding: 6px 10px; letter-spacing: 0.2px; }
    .header h1 { margin: 0 0 8px; font-size: 24px; line-height: 1.25; font-weight: 700; color: #111827; }
    .header p { margin: 0; font-size: 14px; color: #64748b; }
    .content { padding: 24px 28px 22px; line-height: 1.6; font-size: 15px; }
    .greeting { margin: 0 0 10px; color: #111827; }
    .intro { margin: 0 0 16px; color: #334155; }
    .summary { background: #f8fafc; border: 1px solid #e5eaf2; border-radius: 10px; padding: 12px 14px; margin: 0 0 16px; }
    .summary-row { margin: 0 0 8px; }
    .summary-row:last-child { margin-bottom: 0; }
    .summary-label { display: inline-block; min-width: 120px; color: #64748b; font-size: 13px; }
    .summary-value { color: #111827; font-weight: 600; word-break: break-word; }
    .section { background: #f8fafc; border: 1px solid #e5eaf2; border-radius: 10px; padding: 12px 14px; margin: 0 0 16px; color: #334155; }
    .section h3 { margin: 0 0 8px; font-size: 14px; color: #111827; }
    .section p { margin: 0; font-size: 14px; }
    .section ul { margin: 0; padding-left: 18px; }
    .section li { margin: 4px 0; font-size: 14px; }
    .button-group { margin: 18px 0 14px; text-align: center; }
    .button-table { margin: 0 auto; border-collapse: collapse; }
    .btn-cell { padding: 0 6px; }
    .btn { display: inline-block; width: 190px; box-sizing: border-box; padding: 12px 14px; border-radius: 10px; text-decoration: none !important; text-align: center; font-size: 14px; font-weight: 600; margin: 0; -webkit-text-size-adjust: none; mso-line-height-rule: exactly; }
    .btn, .btn:visited, .btn:hover, .btn:active { text-decoration: none !important; }
    .btn-primary, .btn-primary:visited { background: #2563eb !important; border: 1px solid #2563eb !important; color: #ffffff !important; -webkit-text-fill-color: #ffffff !important; }
    .btn-secondary, .btn-secondary:visited { background: #f8fafc !important; border: 1px solid #d7e0eb !important; color: #1f2937 !important; -webkit-text-fill-color: #1f2937 !important; }
    .btn-danger, .btn-danger:visited { background: #dc2626 !important; border: 1px solid #dc2626 !important; color: #ffffff !important; -webkit-text-fill-color: #ffffff !important; }
    .notice { background: #fff7ed; border: 1px solid #fed7aa; color: #9a3412; border-radius: 10px; padding: 12px 14px; font-size: 13px; margin: 0 0 16px; }
    .alt-link { background: #f8fafc; border: 1px solid #e5eaf2; border-radius: 10px; padding: 12px 14px; font-size: 13px; color: #475569; margin: 0 0 12px; word-break: break-word; }
    .alt-link a { color: #2563eb; text-decoration: none; }
    .alt-link a:hover { text-decoration: underline; }
    .footer { border-top: 1px solid #edf1f7; padding: 18px 28px 24px; font-size: 12px; color: #64748b; text-align: center; line-height: 1.6; }
    .footer a { color: #2f5f8c; text-decoration: none; margin: 0 8px; }
    .footer a:hover { text-decoration: underline; }
    @media (max-width: 600px) {
      .shell { padding: 16px 8px; }
      .header, .content, .footer { padding-left: 16px; padding-right: 16px; }
      .btn-cell { padding: 0 4px; }
      .btn { width: 146px; font-size: 13px; padding: 10px 8px; }
    }
  </style>
</head>
<body>
  <div class="shell">
    <div class="container">
      <div class="header">
        %s
        <h1>%s</h1>
        <p>%s</p>
      </div>
      <div class="content">
        <p class="greeting">%s</p>
        <p class="intro">%s</p>
        %s
        %s
        %s
        %s
        %s
      </div>
      <div class="footer">
        Need help? <a href="%s/support">Support</a> | <a href="%s/privacy">Privacy Policy</a> | <a href="%s/terms">Terms</a><br>
        © %d Lihatin. All rights reserved.
      </div>
    </div>
  </div>
</body>
</html>`,
		badgeHTML,
		html.EscapeString(t.Title),
		html.EscapeString(t.Subtitle),
		html.EscapeString(t.Greeting),
		html.EscapeString(t.Intro),
		detailsHTML,
		sectionsHTML,
		actionsHTML,
		noticeHTML,
		fallbackHTML,
		html.EscapeString(t.FooterBaseURL),
		html.EscapeString(t.FooterBaseURL),
		html.EscapeString(t.FooterBaseURL),
		year,
	)
}

func renderDetailsSection(details []emailDetail) string {
	if len(details) == 0 {
		return ""
	}

	var rows strings.Builder
	for _, detail := range details {
		rows.WriteString(fmt.Sprintf(
			`<p class="summary-row"><span class="summary-label">%s</span><span class="summary-value">%s</span></p>`,
			html.EscapeString(detail.Label),
			html.EscapeString(detail.Value),
		))
	}

	return fmt.Sprintf(`<div class="summary">%s</div>`, rows.String())
}

func renderActionsSection(actions []emailAction) string {
	if len(actions) == 0 {
		return ""
	}

	var cells strings.Builder
	for _, action := range actions {
		variant, inlineStyle := resolveButtonVariant(action.Variant)

		cells.WriteString(fmt.Sprintf(
			`<td class="btn-cell"><a href="%s" class="btn %s" style="%s">%s</a></td>`,
			html.EscapeString(action.URL),
			variant,
			inlineStyle,
			html.EscapeString(action.Label),
		))
	}

	return fmt.Sprintf(
		`<div class="button-group"><table role="presentation" class="button-table" cellspacing="0" cellpadding="0" border="0"><tr>%s</tr></table></div>`,
		cells.String(),
	)
}

func resolveButtonVariant(variant string) (className, inlineStyle string) {
	switch variant {
	case "danger":
		return "btn-danger", "color:#ffffff !important;-webkit-text-fill-color:#ffffff !important;background:#dc2626;border:1px solid #dc2626;"
	case "secondary":
		return "btn-secondary", "color:#1f2937 !important;-webkit-text-fill-color:#1f2937 !important;background:#f8fafc;border:1px solid #d7e0eb;"
	default:
		return "btn-primary", "color:#ffffff !important;-webkit-text-fill-color:#ffffff !important;background:#2563eb;border:1px solid #2563eb;"
	}
}

func renderListSection(title string, items []string) string {
	if len(items) == 0 {
		return ""
	}

	var list strings.Builder
	for _, item := range items {
		list.WriteString(fmt.Sprintf("<li>%s</li>", html.EscapeString(item)))
	}

	return fmt.Sprintf(
		`<div class="section"><h3>%s</h3><ul>%s</ul></div>`,
		html.EscapeString(title),
		list.String(),
	)
}

func renderParagraphSection(title, content string) string {
	return fmt.Sprintf(
		`<div class="section"><h3>%s</h3><p>%s</p></div>`,
		html.EscapeString(title),
		html.EscapeString(content),
	)
}

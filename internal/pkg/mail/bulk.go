// utils/mail/bulk.go - Bulk short link creation email template

package mail

import (
    "fmt"
    "strings"

    "github.com/adehusnim37/lihatin-go/models/shortlink"
    "github.com/adehusnim37/lihatin-go/internal/pkg/config"
)

// SendBulkCreationSummary sends email notification for bulk short link creation
func (es *EmailService) SendBulkCreationSummary(toEmail, toName string, links []shortlink.ShortLink, details []shortlink.ShortLinkDetail) error {
    linkCount := len(links)
    subject := fmt.Sprintf("Bulk Short Links Created: %d Links - Lihatin", linkCount)

    // Build links table HTML
    linksTableHTML := buildLinksTableHTML(links, details)
    linksTableText := buildLinksTableText(links, details)

    // Count protected links
    protectedCount := 0
    for _, detail := range details {
        if detail.Passcode != 0 {
            protectedCount++
        }
    }

    htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Bulk Short Links Created</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { 
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            line-height: 1.6; 
            color: #2c3e50;
            background-color: #f8f9fa;
        }
        .email-container { 
            max-width: 700px; 
            margin: 40px auto; 
            background: #ffffff;
            border-radius: 12px;
            box-shadow: 0 4px 20px rgba(0, 0, 0, 0.1);
            overflow: hidden;
        }
        .email-header { 
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
            color: white; 
            padding: 40px 30px;
            text-align: center;
        }
        .email-header h1 {
            font-size: 28px;
            font-weight: 600;
            margin-bottom: 8px;
            letter-spacing: -0.5px;
        }
        .email-header p {
            font-size: 16px;
            opacity: 0.9;
            margin: 0;
        }
        .email-content { 
            padding: 40px 30px;
            background-color: #ffffff;
        }
        .greeting {
            font-size: 18px;
            font-weight: 600;
            color: #2c3e50;
            margin-bottom: 20px;
        }
        .message {
            font-size: 16px;
            color: #555;
            line-height: 1.7;
            margin-bottom: 30px;
        }
        .stats-container {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
            gap: 20px;
            margin: 25px 0;
        }
        .stat-card {
            background: linear-gradient(135deg, #f8f9fa 0%%, #e9ecef 100%%);
            border: 1px solid #dee2e6;
            border-radius: 12px;
            padding: 20px;
            text-align: center;
            border-left: 4px solid #667eea;
        }
        .stat-number {
            font-size: 32px;
            font-weight: bold;
            color: #667eea;
            display: block;
            margin-bottom: 8px;
        }
        .stat-label {
            font-size: 14px;
            color: #6c757d;
            font-weight: 500;
        }
        .links-section {
            background-color: #f8f9fa;
            border: 1px solid #e9ecef;
            border-radius: 12px;
            padding: 25px;
            margin: 25px 0;
            border-left: 4px solid #667eea;
        }
        .links-section h3 {
            color: #2c3e50;
            margin-bottom: 20px;
            font-size: 18px;
            display: flex;
            align-items: center;
            gap: 8px;
        }
        .links-table {
            width: 100%%;
            border-collapse: collapse;
            margin: 15px 0;
            background: white;
            border-radius: 8px;
            overflow: hidden;
            box-shadow: 0 2px 8px rgba(0,0,0,0.1);
        }
        .links-table th {
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
            color: white;
            padding: 12px 10px;
            text-align: left;
            font-weight: 600;
            font-size: 14px;
        }
        .links-table td {
            padding: 12px 10px;
            border-bottom: 1px solid #e9ecef;
            font-size: 13px;
            vertical-align: top;
        }
        .links-table tr:last-child td {
            border-bottom: none;
        }
        .links-table tr:nth-child(even) {
            background-color: #f8f9fa;
        }
        .short-code {
            font-family: 'Courier New', monospace;
            background-color: #e3f2fd;
            padding: 4px 8px;
            border-radius: 4px;
            color: #1565c0;
            font-weight: bold;
            font-size: 12px;
        }
        .original-url {
            max-width: 200px;
            word-break: break-all;
            color: #495057;
        }
        .passcode-badge {
            background: linear-gradient(135deg, #ffc107 0%%, #ff8f00 100%%);
            color: white;
            padding: 4px 8px;
            border-radius: 15px;
            font-size: 11px;
            font-weight: bold;
            display: inline-block;
        }
        .no-passcode {
            color: #6c757d;
            font-style: italic;
            font-size: 11px;
        }
        .security-notice {
            background-color: #fff3cd;
            border: 1px solid #ffeaa7;
            border-radius: 8px;
            padding: 20px;
            margin: 25px 0;
            border-left: 4px solid #ffc107;
        }
        .security-notice h4 {
            color: #856404;
            margin-bottom: 12px;
            font-size: 16px;
            display: flex;
            align-items: center;
            gap: 8px;
        }
        .security-notice ul {
            color: #856404;
            margin: 0;
            padding-left: 20px;
        }
        .security-notice li {
            margin: 8px 0;
            font-size: 14px;
        }
        .action-buttons {
            text-align: center;
            margin: 30px 0;
        }
        .action-button { 
            display: inline-block;
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
            color: white !important;
            padding: 14px 28px;
            text-decoration: none;
            border-radius: 8px;
            font-weight: 600;
            font-size: 14px;
            letter-spacing: 0.5px;
            margin: 10px;
            transition: all 0.3s ease;
            box-shadow: 0 4px 15px rgba(102, 126, 234, 0.3);
        }
        .action-button:hover {
            transform: translateY(-2px);
            box-shadow: 0 6px 20px rgba(102, 126, 234, 0.4);
        }
        .email-footer { 
            background-color: #f8f9fa;
            padding: 30px;
            text-align: center;
            border-top: 1px solid #e9ecef;
        }
        .footer-content {
            color: #6c757d;
            font-size: 14px;
            line-height: 1.5;
        }
        .footer-links {
            margin: 15px 0;
        }
        .footer-links a {
            color: #667eea;
            text-decoration: none;
            margin: 0 15px;
            font-size: 14px;
        }
        .footer-links a:hover {
            text-decoration: underline;
        }
        .company-info {
            font-size: 12px;
            color: #adb5bd;
            margin-top: 15px;
        }
        @media (max-width: 600px) {
            .email-container { margin: 20px; border-radius: 8px; }
            .email-header, .email-content, .email-footer { padding: 25px 20px; }
            .email-header h1 { font-size: 24px; }
            .stats-container { grid-template-columns: 1fr 1fr; }
            .action-button { display: block; margin: 10px 0; }
            .links-table { font-size: 11px; }
            .links-table th, .links-table td { padding: 8px 6px; }
            .original-url { max-width: 120px; }
        }
    </style>
</head>
<body>
    <div class="email-container">
        <div class="email-header">
            <h1>Bulk Links Created Successfully</h1>
            <p>Your %d short links are ready to share</p>
        </div>
        
        <div class="email-content">
            <div class="greeting">Dear %s,</div>
            
            <div class="message">
                Excellent! Your bulk short link creation has been completed successfully. 
                All %d links have been generated and are ready for immediate use.
            </div>
            
            <div class="stats-container">
                <div class="stat-card">
                    <span class="stat-number">%d</span>
                    <span class="stat-label">Total Links</span>
                </div>
                <div class="stat-card">
                    <span class="stat-number">%d</span>
                    <span class="stat-label">Protected</span>
                </div>
                <div class="stat-card">
                    <span class="stat-number">%d</span>
                    <span class="stat-label">Public</span>
                </div>
            </div>
            
            <div class="links-section">
                <h3>📋 Your Created Links</h3>
                %s
            </div>
            
            %s
            
            <div class="security-notice">
                <h4>🔒 Bulk Management Tips</h4>
                <ul>
                    <li>Review all links and their settings from your dashboard</li>
                    <li>Keep passcodes secure for protected links</li>
                    <li>Monitor analytics for all links from the management panel</li>
                    <li>Set up link expiration reminders if needed</li>
                    <li>Share links responsibly with intended recipients only</li>
                </ul>
            </div>
            
            <div class="action-buttons">
                <a href="%s/dashboard" class="action-button">View Dashboard</a>
                <a href="%s/links/manage" class="action-button">Manage All Links</a>
                <a href="%s/analytics" class="action-button">View Analytics</a>
            </div>
            
            <p style="margin-top: 25px; color: #666; font-size: 14px; text-align: center;">
                <strong>Need help managing your links?</strong> Our comprehensive dashboard provides 
                advanced tools for monitoring, editing, and analyzing all your short links.
            </p>
        </div>
        
        <div class="email-footer">
            <div class="footer-content">
                <p>Thank you for using Lihatin - Professional URL Shortening</p>
                <div class="footer-links">
                    <a href="%s/help">Help Center</a>
                    <a href="%s/dashboard">Dashboard</a>
                    <a href="%s/support">Contact Support</a>
                </div>
                <div class="company-info">
                    © 2025 Lihatin. All rights reserved.<br>
                    This is an automated notification for your bulk operation.
                </div>
            </div>
        </div>
    </div>
</body>
</html>
`, linkCount, toName, linkCount, linkCount, protectedCount, linkCount-protectedCount,
        linksTableHTML, 
        buildProtectedLinksWarning(protectedCount),
        config.GetRequiredEnv(config.EnvBackendURL), config.GetRequiredEnv(config.EnvBackendURL), config.GetRequiredEnv(config.EnvBackendURL),
        config.GetRequiredEnv(config.EnvBackendURL), config.GetRequiredEnv(config.EnvBackendURL), config.GetRequiredEnv(config.EnvBackendURL))

    textBody := fmt.Sprintf(`
LIHATIN - BULK SHORT LINKS CREATED SUCCESSFULLY

Dear %s,

Excellent! Your bulk short link creation has been completed successfully.

SUMMARY:
- Total Links Created: %d
- Protected Links: %d  
- Public Links: %d

YOUR CREATED LINKS:
%s

%s

BULK MANAGEMENT TIPS:
- Review all links and their settings from your dashboard
- Keep passcodes secure for protected links  
- Monitor analytics for all links from the management panel
- Set up link expiration reminders if needed
- Share links responsibly with intended recipients only

Dashboard: %s/dashboard
Manage Links: %s/links/manage
Analytics: %s/analytics
Support: %s/support

Thank you for using Lihatin - Professional URL Shortening

Best regards,
The Lihatin Team

---
© 2025 Lihatin. All rights reserved.
This is an automated notification for your bulk operation.
`, toName, linkCount, protectedCount, linkCount-protectedCount,
        linksTableText,
        buildProtectedLinksWarningText(protectedCount),
        config.GetRequiredEnv(config.EnvBackendURL), config.GetRequiredEnv(config.EnvBackendURL), 
        config.GetRequiredEnv(config.EnvBackendURL), config.GetRequiredEnv(config.EnvBackendURL))

    return es.sendEmail(toEmail, subject, textBody, htmlBody)
}

// buildLinksTableHTML creates HTML table of created links
func buildLinksTableHTML(links []shortlink.ShortLink, details []shortlink.ShortLinkDetail) string {
    if len(links) == 0 {
        return "<p>No links created.</p>"
    }

    // Create map for quick detail lookup
    detailMap := make(map[string]shortlink.ShortLinkDetail)
    for _, detail := range details {
        detailMap[detail.ShortLinkID] = detail
    }

    var tableRows strings.Builder
    backendURL := config.GetRequiredEnv(config.EnvBackendURL)

    for i, link := range links {
        if i >= 20 { // Limit display to first 20 links
            tableRows.WriteString(fmt.Sprintf(`
                <tr>
                    <td colspan="4" style="text-align: center; font-style: italic; color: #6c757d;">
                        ... and %d more links (view all in dashboard)
                    </td>
                </tr>`, len(links)-20))
            break
        }

        detail := detailMap[link.ID]
        
        // Build short URL
        shortURL := fmt.Sprintf("%s/s/%s", backendURL, link.ShortCode)
        
        // Format title
        title := link.Title
        if title == "" {
            title = "Untitled Link"
        }
        if len(title) > 30 {
            title = title[:27] + "..."
        }

        // Format original URL for display
        displayURL := link.OriginalURL
        if len(displayURL) > 40 {
            displayURL = displayURL[:37] + "..."
        }

        // Format passcode
        passcodeDisplay := `<span class="no-passcode">None</span>`
        if detail.Passcode != 0 {
            passcodeDisplay = fmt.Sprintf(`<span class="passcode-badge">%d</span>`, detail.Passcode)
        }

        tableRows.WriteString(fmt.Sprintf(`
            <tr>
                <td><strong>%s</strong><br><small style="color: #6c757d;">%s</small></td>
                <td><span class="short-code">%s</span><br><small><a href="%s" style="color: #667eea; text-decoration: none;">%s</a></small></td>
                <td class="original-url">%s</td>
                <td>%s</td>
            </tr>`, title, link.CreatedAt.Format("Jan 2, 15:04"), 
            link.ShortCode, shortURL, shortURL, displayURL, passcodeDisplay))
    }

    return fmt.Sprintf(`
        <table class="links-table">
            <thead>
                <tr>
                    <th>Title & Created</th>
                    <th>Short Code & URL</th>
                    <th>Original URL</th>
                    <th>Passcode</th>
                </tr>
            </thead>
            <tbody>%s</tbody>
        </table>`, tableRows.String())
}

// buildLinksTableText creates text version of links table
func buildLinksTableText(links []shortlink.ShortLink, details []shortlink.ShortLinkDetail) string {
    if len(links) == 0 {
        return "No links created."
    }

    // Create map for quick detail lookup
    detailMap := make(map[string]shortlink.ShortLinkDetail)
    for _, detail := range details {
        detailMap[detail.ShortLinkID] = detail
    }

    var result strings.Builder
    backendURL := config.GetRequiredEnv(config.EnvBackendURL)

    for i, link := range links {
        if i >= 10 { // Limit text version to 10 links
            result.WriteString(fmt.Sprintf("\n... and %d more links (view all in dashboard)\n", len(links)-10))
            break
        }

        detail := detailMap[link.ID]
        
        title := link.Title
        if title == "" {
            title = "Untitled Link"
        }

        shortURL := fmt.Sprintf("%s/s/%s", backendURL, link.ShortCode)
        
        passcode := "None"
        if detail.Passcode != 0 {
            passcode = fmt.Sprintf("%d", detail.Passcode)
        }

        result.WriteString(fmt.Sprintf(`
%d. %s
   Short Code: %s
   Short URL: %s
   Original: %s
   Passcode: %s
   Created: %s
`, i+1, title, link.ShortCode, shortURL, link.OriginalURL, passcode, link.CreatedAt.Format("Jan 2, 2006 15:04")))
    }

    return result.String()
}

// buildProtectedLinksWarning creates warning section for protected links
func buildProtectedLinksWarning(protectedCount int) string {
    if protectedCount == 0 {
        return ""
    }

    return fmt.Sprintf(`
            <div class="security-notice">
                <h4>🔐 Protected Links Security Notice</h4>
                <p style="color: #856404; margin: 12px 0; font-size: 14px;">
                    <strong>Important:</strong> You have %d protected link(s) with passcodes. 
                    Please keep the passcodes secure and share them only with authorized users.
                </p>
                <ul style="color: #856404; margin: 8px 0; padding-left: 20px;">
                    <li>Passcodes are required to access protected links</li>
                    <li>Store passcodes in a secure location</li>
                    <li>Don't share passcodes in unsecured communications</li>
                    <li>You can view all passcodes from your dashboard</li>
                </ul>
            </div>`, protectedCount)
}

// buildProtectedLinksWarningText creates text version of protected links warning
func buildProtectedLinksWarningText(protectedCount int) string {
    if protectedCount == 0 {
        return ""
    }

    return fmt.Sprintf(`
PROTECTED LINKS SECURITY NOTICE:
You have %d protected link(s) with passcodes. Please keep the passcodes secure and share them only with authorized users.

- Passcodes are required to access protected links
- Store passcodes in a secure location  
- Don't share passcodes in unsecured communications
- You can view all passcodes from your dashboard`, protectedCount)
}
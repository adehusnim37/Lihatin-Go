package mail

import (
	"fmt"
	"net/smtp"
	"time"

	"github.com/jordan-wright/email"
)

// sendEmail sends an email using SMTP
func (es *EmailService) sendEmail(to, subject, textBody, htmlBody string) error {
	if es.config.SMTPUsername == "" || es.config.SMTPPassword == "" {
		// Skip sending email if SMTP is not configured
		fmt.Printf("Email would be sent to %s with subject: %s\n", to, subject)
		return nil
	}

	e := email.NewEmail()
	e.From = fmt.Sprintf("%s <%s>", es.config.FromName, es.config.FromEmail)
	e.To = []string{to}
	e.Subject = subject
	e.Text = []byte(textBody)
	e.HTML = []byte(htmlBody)

	auth := smtp.PlainAuth("", es.config.SMTPUsername, es.config.SMTPPassword, es.config.SMTPHost)
	addr := fmt.Sprintf("%s:%s", es.config.SMTPHost, es.config.SMTPPort)

	return e.Send(addr, auth)
}

// IsTokenExpired checks if a token timestamp has expired
func IsTokenExpired(tokenTime *time.Time, expiryHours int) bool {
	if tokenTime == nil {
		return true
	}
	return time.Now().After(tokenTime.Add(time.Duration(expiryHours) * time.Hour))
}

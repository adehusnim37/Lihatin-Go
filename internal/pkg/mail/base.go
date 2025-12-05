package mail

import (
	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
)

// EmailConfig holds email configuration
type EmailConfig struct {
	SMTPHost     string
	SMTPPort     string
	SMTPUsername string
	SMTPPassword string
	FromEmail    string
	FromName     string
}

// EmailTemplateType represents different email template types
type EmailTemplateType string

const (
	EmailTemplateVerification  EmailTemplateType = "verification"
	EmailTemplatePasswordReset EmailTemplateType = "password_reset"
	EmailTemplateTOTPSetup     EmailTemplateType = "totp_setup"
	EmailTemplateLoginAlert    EmailTemplateType = "login_alert"
	EmailTemplatePasscodeReset EmailTemplateType = "passcode_reset"
)

// EmailService handles email operations
type EmailService struct {
	config EmailConfig
}

// NewEmailService creates a new email service instance
func NewEmailService() *EmailService {
	return &EmailService{
		config: EmailConfig{
			SMTPHost:     config.GetEnvOrDefault(config.EnvSMTPHost, "smtp.gmail.com"),
			SMTPPort:     config.GetEnvOrDefault(config.EnvSMTPPort, "587"),
			SMTPUsername: config.GetEnvOrDefault(config.EnvSMTPUser, ""),
			SMTPPassword: config.GetEnvOrDefault(config.EnvSMTPPass, ""),
			FromEmail:    config.GetEnvOrDefault(config.EnvFromEmail, "noreply@lihatin.com"),
			FromName:     config.GetEnvOrDefault(config.EnvFromName, "Lihatin"),
		},
	}
}

// GetConfig returns the email configuration
func (es *EmailService) GetConfig() EmailConfig {
	return es.config
}

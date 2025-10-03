package mail

import (
	"github.com/adehusnim37/lihatin-go/utils"
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
			SMTPHost:     utils.GetEnvOrDefault(utils.EnvSMTPHost, "smtp.gmail.com"),
			SMTPPort:     utils.GetEnvOrDefault(utils.EnvSMTPPort, "587"),
			SMTPUsername: utils.GetEnvOrDefault(utils.EnvSMTPUser, ""),
			SMTPPassword: utils.GetEnvOrDefault(utils.EnvSMTPPass, ""),
			FromEmail:    utils.GetEnvOrDefault(utils.EnvFromEmail, "noreply@lihatin.com"),
			FromName:     utils.GetEnvOrDefault(utils.EnvFromName, "Lihatin"),
		},
	}
}

// GetConfig returns the email configuration
func (es *EmailService) GetConfig() EmailConfig {
	return es.config
}

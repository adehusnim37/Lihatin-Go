package support

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/adehusnim37/lihatin-go/controllers"
	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/internal/pkg/mail"
	"github.com/adehusnim37/lihatin-go/internal/pkg/storage"
	supportmodel "github.com/adehusnim37/lihatin-go/models/support"
	"github.com/adehusnim37/lihatin-go/repositories/authrepo"
	"github.com/adehusnim37/lihatin-go/repositories/supportrepo"
)

const (
	ticketPrefix      = "LHTK-"
	ticketCodeLength  = 6
	ticketCodeCharset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
)

type Controller struct {
	*controllers.BaseController
	repo            *supportrepo.SupportTicketRepository
	authRepo        *authrepo.AuthRepository
	emailSvc        *mail.EmailService
	attachmentStore *storage.S3SupportAttachmentStorage
	httpClient      *http.Client
}

type turnstileVerifyResponse struct {
	Success    bool     `json:"success"`
	ErrorCodes []string `json:"error-codes"`
}

func NewController(base *controllers.BaseController) *Controller {
	if base == nil || base.GormDB == nil {
		panic("GormDB is required for SupportController")
	}

	attachmentStore, attachmentStoreErr := storage.NewS3SupportAttachmentStorageFromEnv()
	if attachmentStoreErr != nil {
		logger.Logger.Warn("Support attachment storage is not configured", "error", attachmentStoreErr.Error())
	}

	return &Controller{
		BaseController:  base,
		repo:            supportrepo.NewSupportTicketRepository(base.GormDB),
		authRepo:        authrepo.NewAuthRepository(base.GormDB),
		emailSvc:        mail.NewEmailService(),
		attachmentStore: attachmentStore,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Controller) generateTicketCode() (string, error) {
	for i := 0; i < 12; i++ {
		randomPart, err := randomAlphaNumeric(ticketCodeLength)
		if err != nil {
			return "", err
		}

		code := ticketPrefix + randomPart
		exists, err := c.repo.TicketCodeExists(code)
		if err != nil {
			return "", err
		}
		if !exists {
			return code, nil
		}
	}

	return "", errors.New("failed to generate unique support ticket code")
}

func randomAlphaNumeric(length int) (string, error) {
	if length <= 0 {
		return "", errors.New("invalid ticket code length")
	}

	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	result := make([]byte, length)
	for i := range buf {
		result[i] = ticketCodeCharset[int(buf[i])%len(ticketCodeCharset)]
	}

	return string(result), nil
}

func (c *Controller) verifyCaptcha(token, ipAddress string) (bool, error) {
	secret := strings.TrimSpace(config.GetEnvOrDefault(config.EnvTurnstileSecretKey, ""))
	env := strings.ToLower(strings.TrimSpace(config.GetEnvOrDefault(config.Env, "development")))

	if token == "" {
		return false, errors.New("captcha token is required")
	}

	if secret == "" {
		if env == "production" {
			return false, errors.New("captcha secret key is not configured")
		}

		logger.Logger.Warn("Turnstile secret missing, bypassing captcha verification in non-production", "env", env)
		return true, nil
	}

	payload := map[string]string{
		"secret":   secret,
		"response": token,
	}
	if strings.TrimSpace(ipAddress) != "" {
		payload["remoteip"] = strings.TrimSpace(ipAddress)
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return false, err
	}

	req, err := http.NewRequest(http.MethodPost, "https://challenges.cloudflare.com/turnstile/v0/siteverify", bytes.NewReader(body))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	var verifyResp turnstileVerifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&verifyResp); err != nil {
		return false, err
	}

	if !verifyResp.Success {
		return false, fmt.Errorf("captcha verification failed: %s", strings.Join(verifyResp.ErrorCodes, ","))
	}

	return true, nil
}

func normalizeTicketCategory(raw string) string {
	category := strings.ToLower(strings.TrimSpace(raw))
	if category == "" {
		return string(supportmodel.TicketCategoryOther)
	}
	return category
}

func buildCategoryLabel(category string) string {
	switch strings.ToLower(strings.TrimSpace(category)) {
	case string(supportmodel.TicketCategoryAccountLocked):
		return "Account Locked"
	case string(supportmodel.TicketCategoryAccountDeactivated):
		return "Account Deactivated"
	case string(supportmodel.TicketCategoryEmailVerification):
		return "Email Verification"
	case string(supportmodel.TicketCategoryLost2FA):
		return "Lost 2FA"
	case string(supportmodel.TicketCategoryBilling):
		return "Billing"
	case string(supportmodel.TicketCategoryBugReport):
		return "Bug Report"
	case string(supportmodel.TicketCategoryFeatureRequest):
		return "Feature Request"
	default:
		return "Other"
	}
}

func hashSupportAccessCode(raw string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(raw)))
	return fmt.Sprintf("%x", sum[:])
}

func (c *Controller) frontendURL() string {
	return strings.TrimRight(
		strings.TrimSpace(config.GetEnvOrDefault(config.EnvFrontendURL, "http://localhost:3000")),
		"/",
	)
}

func supportMessagePreview(raw string) string {
	preview := strings.TrimSpace(raw)
	if preview == "" {
		return "New support message."
	}
	preview = strings.ReplaceAll(preview, "\n", " ")
	preview = strings.Join(strings.Fields(preview), " ")
	if len(preview) > 220 {
		return preview[:220] + "..."
	}
	return preview
}

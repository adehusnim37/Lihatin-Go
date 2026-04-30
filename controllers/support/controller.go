package support

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/adehusnim37/lihatin-go/controllers"
	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
	apperrors "github.com/adehusnim37/lihatin-go/internal/pkg/errors"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/internal/pkg/mail"
	"github.com/adehusnim37/lihatin-go/internal/pkg/storage"
	"github.com/adehusnim37/lihatin-go/middleware"
	supportmodel "github.com/adehusnim37/lihatin-go/models/support"
	"github.com/adehusnim37/lihatin-go/repositories/authrepo"
	"github.com/adehusnim37/lihatin-go/repositories/supportrepo"
	"github.com/gin-gonic/gin"
)

const (
	ticketPrefix                  = "LHTK-"
	ticketCodeLength              = 6
	ticketCodeCharset             = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	supportAccessOTPRequestLimit  = 5
	supportAccessOTPRequestWindow = 30 * time.Minute
	supportAccessOTPResendLimit   = 3
	supportAccessOTPResendWindow  = time.Hour
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
	Hostname   string   `json:"hostname"`
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
	token = strings.TrimSpace(token)

	if token == "" {
		return false, errors.New("captcha token is required")
	}

	if env != "production" && token == "XXXX.DUMMY.TOKEN.XXXX" {
		logger.Logger.Warn("Turnstile dev bypass accepted in non-production", "env", env)
		return true, nil
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

	if !isAllowedTurnstileHostname(verifyResp.Hostname) {
		return false, fmt.Errorf("captcha hostname not allowed: %s", verifyResp.Hostname)
	}

	return true, nil
}

func isAllowedTurnstileHostname(hostname string) bool {
	hostname = strings.ToLower(strings.TrimSpace(hostname))
	if hostname == "" {
		return false
	}

	for _, candidate := range allowedTurnstileHostnames() {
		if hostname == candidate {
			return true
		}
	}

	return false
}

func allowedTurnstileHostnames() []string {
	allowed := make([]string, 0, 4)

	appendHostname := func(raw string) {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return
		}

		if strings.Contains(raw, "://") {
			if parsed, err := url.Parse(raw); err == nil && parsed.Hostname() != "" {
				raw = parsed.Hostname()
			}
		}

		raw = strings.ToLower(strings.TrimSpace(raw))
		if raw == "" {
			return
		}

		for _, existing := range allowed {
			if existing == raw {
				return
			}
		}

		allowed = append(allowed, raw)
	}

	appendHostname(config.GetEnvOrDefault(config.EnvFrontendURL, ""))
	for _, origin := range strings.Split(config.GetEnvOrDefault(config.EnvAllowedOrigins, ""), ",") {
		appendHostname(origin)
	}

	appendHostname("localhost")
	appendHostname("127.0.0.1")
	appendHostname("::1")

	return allowed
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

func hashSupportRateLimitSubject(action, ticket, email string) string {
	normalized := strings.Join([]string{
		strings.ToLower(strings.TrimSpace(action)),
		strings.ToUpper(strings.TrimSpace(ticket)),
		strings.ToLower(strings.TrimSpace(email)),
	}, ":")
	sum := sha256.Sum256([]byte(normalized))
	return fmt.Sprintf("%x", sum[:])
}

func (c *Controller) enforceSupportAccessRateLimit(
	ctx context.Context,
	action string,
	ticket string,
	email string,
	limit int,
	window time.Duration,
) (blocked bool, err error) {
	manager := middleware.GetSessionManager()
	if manager == nil || manager.GetRedisClient() == nil {
		return false, nil
	}

	key := fmt.Sprintf(
		"support_access_rate:%s:%s",
		strings.ToLower(strings.TrimSpace(action)),
		hashSupportRateLimitSubject(action, ticket, email),
	)

	redisClient := manager.GetRedisClient()
	count, err := redisClient.Incr(ctx, key).Result()
	if err != nil {
		logger.Logger.Warn("Support access rate limit Redis error", "error", err.Error(), "action", action)
		return false, nil
	}

	if count == 1 {
		redisClient.Expire(ctx, key, window)
	}

	return count > int64(limit), nil
}

func (c *Controller) frontendURL() string {
	return strings.TrimRight(
		strings.TrimSpace(config.GetEnvOrDefault(config.EnvFrontendURL, "http://localhost:3000")),
		"/",
	)
}

func (c *Controller) handleAppError(ctx *gin.Context, err error) bool {
	if err == nil {
		return false
	}

	httputil.HandleError(ctx, err, strings.TrimSpace(ctx.GetString("user_id")))
	return true
}

func (c *Controller) handleAppErrorAs(ctx *gin.Context, err error, field string) bool {
	if err == nil {
		return false
	}

	var appErr *apperrors.AppError
	if errors.As(err, &appErr) && strings.TrimSpace(field) != "" {
		cloned := *appErr
		cloned.Field = field
		httputil.HandleError(ctx, &cloned, strings.TrimSpace(ctx.GetString("user_id")))
		return true
	}

	return c.handleAppError(ctx, err)
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

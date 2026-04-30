package disposable

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/repositories/userrepo"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const (
	defaultSourceURL                 = "https://raw.githubusercontent.com/disposable/disposable-email-domains/master/domains.txt"
	cacheKeyLastKnownGoodDomainList  = "disposable_email_domains:last_known_good"
	defaultHTTPTimeout               = 12 * time.Second
	SettingKeyDisposableEmailEnabled = userrepo.SettingDisposableEmailCheckEnabled
)

//go:embed data/disposable_email_blocklist.conf
var seedFS embed.FS

// Policy manages disposable email domain blocklist and enforcement state.
type Policy struct {
	mu         sync.RWMutex
	domains    map[string]struct{}
	lastSource string

	settingRepo userrepo.SystemSettingRepository
	redisClient *redis.Client
	httpClient  *http.Client
	sourceURL   string
	env         string
}

var (
	globalPolicy   *Policy
	globalPolicyMu sync.RWMutex

	// ErrDisposableEmailBlocked indicates email belongs to disposable domain list.
	ErrDisposableEmailBlocked = errors.New("disposable email blocked")
)

// NewPolicy creates a disposable email policy manager.
func NewPolicy(settingRepo userrepo.SystemSettingRepository, redisClient *redis.Client, env, sourceURL string, httpClient *http.Client) *Policy {
	if sourceURL == "" {
		sourceURL = defaultSourceURL
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultHTTPTimeout}
	}

	return &Policy{
		domains:     loadSeedDomains(),
		lastSource:  "seed",
		settingRepo: settingRepo,
		redisClient: redisClient,
		httpClient:  httpClient,
		sourceURL:   sourceURL,
		env:         strings.ToLower(strings.TrimSpace(env)),
	}
}

// InitGlobal initializes the global policy instance.
func InitGlobal(gormDB *gorm.DB, redisClient *redis.Client) error {
	if gormDB == nil {
		return errors.New("gorm db is required")
	}

	env := config.GetEnvOrDefault(config.Env, "development")
	sourceURL := config.GetEnvOrDefault(config.EnvDisposableEmailBlockListURL, defaultSourceURL)
	repo := userrepo.NewSystemSettingRepository(gormDB)

	policy := NewPolicy(repo, redisClient, env, sourceURL, nil)
	if err := policy.Bootstrap(context.Background()); err != nil {
		return err
	}

	globalPolicyMu.Lock()
	globalPolicy = policy
	globalPolicyMu.Unlock()

	return nil
}

// Global returns initialized global policy manager.
func Global() *Policy {
	globalPolicyMu.RLock()
	defer globalPolicyMu.RUnlock()
	return globalPolicy
}

// Bootstrap initializes defaults and warms up domain list from cache/source.
func (p *Policy) Bootstrap(ctx context.Context) error {
	if p.settingRepo != nil {
		if err := p.settingRepo.EnsureBool(SettingKeyDisposableEmailEnabled, true, "system"); err != nil {
			return fmt.Errorf("failed to ensure disposable email setting default: %w", err)
		}
	}

	p.loadFromCache(ctx)

	if err := p.RefreshFromSource(ctx); err != nil {
		logger.Logger.Warn("Disposable email list refresh failed, using existing domain snapshot",
			"error", err.Error(),
			"source", p.GetLastSource(),
		)
	}

	return nil
}

// ShouldEnforce returns true only when environment is production and toggle enabled.
func ShouldEnforce(env string, enabled bool) bool {
	return strings.EqualFold(strings.TrimSpace(env), "production") && enabled
}

// IsEnforcementActive returns effective enforcement state.
func (p *Policy) IsEnforcementActive(ctx context.Context) (bool, error) {
	if p.settingRepo == nil {
		return ShouldEnforce(p.env, true), nil
	}

	enabled, err := p.settingRepo.GetBool(SettingKeyDisposableEmailEnabled, true)
	if err != nil {
		return false, err
	}
	return ShouldEnforce(p.env, enabled), nil
}

// ShouldBlockEmail evaluates whether given email should be blocked.
func (p *Policy) ShouldBlockEmail(ctx context.Context, email string) (bool, error) {
	enforced, err := p.IsEnforcementActive(ctx)
	if err != nil {
		return false, err
	}
	if !enforced {
		return false, nil
	}

	isDisposable, parseErr := p.IsDisposableEmail(email)
	if parseErr != nil {
		// Validation layer already handles malformed emails.
		return false, nil
	}
	return isDisposable, nil
}

// IsDisposableEmail checks whether email belongs to disposable domain list.
func (p *Policy) IsDisposableEmail(email string) (bool, error) {
	domain, err := ExtractDomain(email)
	if err != nil {
		return false, err
	}
	return p.IsDisposableDomain(domain), nil
}

// IsDisposableDomain checks exact and parent domains.
func (p *Policy) IsDisposableDomain(domain string) bool {
	normalized := NormalizeDomain(domain)
	if normalized == "" {
		return false
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	check := normalized
	for {
		if _, exists := p.domains[check]; exists {
			return true
		}
		dot := strings.IndexByte(check, '.')
		if dot < 0 {
			return false
		}
		check = check[dot+1:]
	}
}

// RefreshFromSource refreshes domain list from remote source.
func (p *Policy) RefreshFromSource(ctx context.Context) error {
	raw, err := p.fetchDomainList(ctx)
	if err != nil {
		return err
	}

	parsed := ParseDomains(raw)
	if len(parsed) == 0 {
		return errors.New("domain list is empty after parsing")
	}

	p.setDomains(parsed, "remote")

	if p.redisClient != nil {
		if cacheErr := p.redisClient.Set(ctx, cacheKeyLastKnownGoodDomainList, raw, 0).Err(); cacheErr != nil {
			logger.Logger.Warn("Failed to persist disposable email domain cache", "error", cacheErr.Error())
		}
	}

	return nil
}

// GetLastSource returns current domain source marker (seed/cache/remote).
func (p *Policy) GetLastSource() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.lastSource
}

func (p *Policy) loadFromCache(ctx context.Context) {
	if p.redisClient == nil {
		return
	}

	raw, err := p.redisClient.Get(ctx, cacheKeyLastKnownGoodDomainList).Result()
	if err != nil {
		return
	}

	parsed := ParseDomains(raw)
	if len(parsed) == 0 {
		return
	}

	p.setDomains(parsed, "cache")
}

func (p *Policy) fetchDomainList(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.sourceURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return "", fmt.Errorf("source responded with status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func (p *Policy) setDomains(domains map[string]struct{}, source string) {
	p.mu.Lock()
	p.domains = domains
	p.lastSource = source
	p.mu.Unlock()
}

func loadSeedDomains() map[string]struct{} {
	raw, err := seedFS.ReadFile("data/disposable_email_blocklist.conf")
	if err != nil {
		logger.Logger.Warn("Failed to load seed disposable domain snapshot", "error", err.Error())
		return map[string]struct{}{}
	}
	return ParseDomains(string(raw))
}

// ParseDomains parses newline-separated domain list.
func ParseDomains(raw string) map[string]struct{} {
	result := make(map[string]struct{})
	for _, line := range strings.Split(raw, "\n") {
		candidate := strings.TrimSpace(line)
		if candidate == "" || strings.HasPrefix(candidate, "#") {
			continue
		}

		normalized := NormalizeDomain(candidate)
		if normalized == "" {
			continue
		}
		result[normalized] = struct{}{}
	}
	return result
}

// NormalizeDomain normalizes and validates domain representation.
func NormalizeDomain(domain string) string {
	d := strings.ToLower(strings.TrimSpace(domain))
	d = strings.TrimSuffix(d, ".")
	if d == "" || strings.Contains(d, " ") {
		return ""
	}
	if strings.Count(d, ".") < 1 {
		return ""
	}
	return d
}

// ExtractDomain extracts and normalizes domain from email.
func ExtractDomain(email string) (string, error) {
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	at := strings.LastIndex(normalizedEmail, "@")
	if at <= 0 || at == len(normalizedEmail)-1 {
		return "", errors.New("invalid email format")
	}

	domain := NormalizeDomain(normalizedEmail[at+1:])
	if domain == "" {
		return "", errors.New("invalid email domain")
	}

	return domain, nil
}

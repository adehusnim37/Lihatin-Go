package auth

import (
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
	"github.com/gin-gonic/gin"
)

// CookieSettings defines common auth cookie attributes.
type CookieSettings struct {
	Domain        string
	Secure        bool
	SameSite      http.SameSite
	SameSiteLabel string
}

// ResolveAuthCookieSettings returns safe cookie settings for both development and production.
func ResolveAuthCookieSettings(ctx *gin.Context) CookieSettings {
	secure := ctx.Request.TLS != nil ||
		strings.EqualFold(ctx.GetHeader("X-Forwarded-Proto"), "https") ||
		strings.EqualFold(ctx.GetHeader("X-Forwarded-Ssl"), "on")

	domain := normalizeCookieDomain(config.GetEnvOrDefault(config.EnvDomain, ""))
	if isLocalCookieHost(domain) {
		domain = ""
	}
	if domain != "" && !strings.HasPrefix(domain, ".") {
		domain = "." + domain
	}

	// SameSite=None requires Secure=true. For HTTP development we must use Lax.
	sameSite := http.SameSiteLaxMode
	sameSiteLabel := "Lax"
	if secure {
		sameSite = http.SameSiteNoneMode
		sameSiteLabel = "None"
	}

	return CookieSettings{
		Domain:        domain,
		Secure:        secure,
		SameSite:      sameSite,
		SameSiteLabel: sameSiteLabel,
	}
}

func normalizeCookieDomain(domain string) string {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return ""
	}

	if strings.Contains(domain, "://") {
		parsed, err := url.Parse(domain)
		if err == nil && parsed.Host != "" {
			domain = parsed.Host
		}
	}

	if host, _, err := net.SplitHostPort(domain); err == nil {
		domain = host
	}

	domain = strings.Trim(domain, "[]")
	domain = strings.TrimPrefix(domain, ".")
	return domain
}

func isLocalCookieHost(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

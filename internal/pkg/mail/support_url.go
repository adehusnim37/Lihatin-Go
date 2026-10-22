package mail

import (
	"fmt"
	"net/url"
	"strings"
)

func normalizeSupportFrontendURL(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.IsAbs() == false || parsed.Hostname() == "" || parsed.User != nil {
		return "", fmt.Errorf("invalid support frontend URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("unsupported support frontend URL scheme")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Opaque != "" {
		return "", fmt.Errorf("support frontend URL must not contain query or fragment data")
	}

	parsed.Path = strings.TrimRight(parsed.Path, "/")
	parsed.RawPath = ""
	return strings.TrimRight(parsed.String(), "/"), nil
}

func buildPublicSupportAccessURL(baseURL, ticketCode string) (string, error) {
	normalizedBase, err := normalizeSupportFrontendURL(baseURL)
	if err != nil {
		return "", err
	}
	parsed, err := url.Parse(normalizedBase)
	if err != nil {
		return "", err
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/support/access"
	query := url.Values{}
	query.Set("ticket", strings.TrimSpace(ticketCode))
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

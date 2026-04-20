package disposable

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/adehusnim37/lihatin-go/models/user"
)

type mockSettingRepo struct {
	enabled bool
}

func (m *mockSettingRepo) GetByKey(_ string) (*user.SystemSetting, error) {
	value := "false"
	if m.enabled {
		value = "true"
	}
	return &user.SystemSetting{Value: value}, nil
}

func (m *mockSettingRepo) GetBool(_ string, defaultValue bool) (bool, error) {
	_ = defaultValue
	return m.enabled, nil
}

func (m *mockSettingRepo) SetBool(_ string, value bool, _ string) error {
	m.enabled = value
	return nil
}

func (m *mockSettingRepo) EnsureBool(_ string, value bool, _ string) error {
	m.enabled = value
	return nil
}

func TestParseDomains(t *testing.T) {
	raw := `
	# comment
	TempMail.COM
	
	guerrillamail.com
	`

	parsed := ParseDomains(raw)
	if _, ok := parsed["tempmail.com"]; !ok {
		t.Fatalf("expected normalized tempmail.com in parsed domains")
	}
	if _, ok := parsed["guerrillamail.com"]; !ok {
		t.Fatalf("expected guerrillamail.com in parsed domains")
	}
}

func TestIsDisposableDomainSupportsSubdomain(t *testing.T) {
	p := &Policy{
		domains: map[string]struct{}{
			"tempmail.com": {},
		},
	}

	if !p.IsDisposableDomain("tempmail.com") {
		t.Fatalf("expected exact domain match to be disposable")
	}
	if !p.IsDisposableDomain("mx1.tempmail.com") {
		t.Fatalf("expected subdomain to be disposable")
	}
	if p.IsDisposableDomain("example.com") {
		t.Fatalf("expected normal domain not disposable")
	}
}

func TestExtractDomainValidation(t *testing.T) {
	domain, err := ExtractDomain("User@TempMail.COM")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if domain != "tempmail.com" {
		t.Fatalf("expected normalized domain tempmail.com, got %s", domain)
	}

	if _, err := ExtractDomain("invalid-email"); err == nil {
		t.Fatalf("expected invalid email parsing error")
	}
}

func TestShouldEnforce(t *testing.T) {
	if !ShouldEnforce("production", true) {
		t.Fatalf("expected enforcement active in production when enabled")
	}
	if ShouldEnforce("production", false) {
		t.Fatalf("expected enforcement disabled when toggle false")
	}
	if ShouldEnforce("development", true) {
		t.Fatalf("expected enforcement disabled outside production")
	}
}

func TestRefreshFallbackKeepsLastKnownGood(t *testing.T) {
	repo := &mockSettingRepo{enabled: true}
	transport := &stubRoundTripper{
		statusCode: http.StatusOK,
		body:       "tempmail.com\nmailinator.com\n",
	}
	client := &http.Client{Transport: transport}

	p := NewPolicy(repo, nil, "production", "http://disposable.test/list", client)
	if err := p.RefreshFromSource(context.Background()); err != nil {
		t.Fatalf("expected refresh success, got error: %v", err)
	}
	if !p.IsDisposableDomain("foo.tempmail.com") {
		t.Fatalf("expected refreshed domain available")
	}

	transport.statusCode = http.StatusInternalServerError
	transport.body = "upstream failed"

	if err := p.RefreshFromSource(context.Background()); err == nil {
		t.Fatalf("expected refresh error when source fails")
	}
	if !p.IsDisposableDomain("mailinator.com") {
		t.Fatalf("expected last-known-good domains retained after failed refresh")
	}
}

type stubRoundTripper struct {
	statusCode int
	body       string
}

func (s *stubRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: s.statusCode,
		Body:       io.NopCloser(strings.NewReader(s.body)),
		Header:     make(http.Header),
	}, nil
}

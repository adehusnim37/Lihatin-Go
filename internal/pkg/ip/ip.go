package ip

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
	"github.com/gin-gonic/gin"
)

type LocationResponse struct {
	IP       string `json:"ip"`
	Location struct {
		ContinentName       string `json:"continent_name"`
		CountryCode2        string `json:"country_code2"`
		CountryCode3        string `json:"country_code3"`
		CountryName         string `json:"country_name"`
		CountryNameOfficial string `json:"country_name_official"`
		CountryCapital      string `json:"country_capital"`
		StateProv           string `json:"state_prov"`
		StateCode           string `json:"state_code"`
		District            string `json:"district"`
		City                string `json:"city"`
		Zipcode             string `json:"zipcode"`
		Latitude            string `json:"latitude"`
		Longitude           string `json:"longitude"`
		IsEU                bool   `json:"is_eu"`
		CountryFlag         string `json:"country_flag"`
		GeonameID           string `json:"geoname_id"`
		CountryEmoji        string `json:"country_emoji"`
	} `json:"location"`
	CountryMetadata struct {
		CallingCode string   `json:"calling_code"`
		TLD         string   `json:"tld"`
		Languages   []string `json:"languages"`
	} `json:"country_metadata"`
}

func IPGeolocation(ip string) (*LocationResponse, error) {
	APIKey := config.GetEnvOrDefault(config.EnvIPGeoAPIKey, "")
	if APIKey == "" {
		// Return specific error so caller knows to skip
		return nil, fmt.Errorf("IP_GEOLOCATION_API_KEY not set")
	}
	url := `https://api.ipgeolocation.io/v2/ipgeo?apiKey=` + APIKey + `&ip=` + ip
	method := "GET"

	client := &http.Client{
		Timeout: 5 * time.Second, // Timeout is critical
	}
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status: %d", res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var locationResponse LocationResponse
	if err := json.Unmarshal(body, &locationResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &locationResponse, nil
}

func GetLocationString(ip string) string {
	country, city := GetLocation(ip)
	if country == "Unknown Country" && city == "Unknown City" {
		return "Unknown Location"
	}

	if country == "Local Machine" {
		return "Local Machine"
	}

	var parts []string
	if city != "Unknown City" {
		parts = append(parts, city)
	}
	if country != "Unknown Country" {
		parts = append(parts, country)
	}

	return joinWithComma(parts)
}

func GetLocation(ip string) (country, city string) {
	if ip == "" || ip == "127.0.0.1" || ip == "::1" || ip == "localhost" {
		return "Local Machine", "Local Machine"
	}

	locationResponse, err := IPGeolocation(ip)
	if err != nil {
		return "Unknown Country", "Unknown City"
	}

	c := locationResponse.Location.CountryName
	if c == "" {
		c = "Unknown Country"
	}

	cy := locationResponse.Location.City
	if cy == "" {
		cy = "Unknown City"
	}

	return c, cy
}

func GetCountryName(ip string) string {
	c, _ := GetLocation(ip)
	return c
}

func GetCityName(ip string) string {
	_, cy := GetLocation(ip)
	return cy
}

// Helper function to join strings with comma
func joinWithComma(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	if len(parts) == 1 {
		return parts[0]
	}

	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += ", " + parts[i]
	}
	return result
}

// deviceIDMaxLen caps the length of a client-supplied device fingerprint.
// The DB column is size 255 (see models/user/auth.go last_device_id), but we
// keep a smaller headroom budget and always prefix client values to avoid
// collisions with our own generated fallback fingerprint.
const deviceIDMaxLen = 120

// normalizeDeviceID validates and sanitizes a client-supplied device ID.
//
// The fingerprint is metadata only — it is NOT a security token, so we do NOT
// use it to gate authorization. We only bound its length (to keep DB writes
// safe) and reject obviously oversized payloads coming off any client.
func normalizeDeviceID(raw string) string {
	// Trim surrounding whitespace/control chars.
	device := strings.TrimSpace(raw)

	// Reject values that are too long to ever be legitimate.
	if len(device) > 2048 {
		return ""
	}

	// Strip any non-printable / control characters that could leak into logs
	// or DB (malicious clients may embed newlines, etc.).
	var b strings.Builder
	for _, r := range device {
		if r >= 32 && r <= 126 || r == 9 { // 9 = tab
			b.WriteRune(r)
		}
	}
	device = strings.TrimSpace(b.String())

	if device == "" {
		return ""
	}

	// Prefix client values so they cannot be confused with (or spoof a)
	// server-generated fallback fingerprint.
	device = "d_" + device

	if len(device) > deviceIDMaxLen {
		device = device[:deviceIDMaxLen]
	}

	return device
}

// GetDeviceAndIPInfo extracts device ID and IP from request context
func GetDeviceAndIPInfo(ctx *gin.Context) (deviceID *string, lastIP *string) {
	// Prefer a client-provided fingerprint (e.g. generated by ThumbmarkJS),
	// but treat it as untrusted metadata: it is validated but never used as an
	// authorization gate.
	device := normalizeDeviceID(ctx.GetHeader("X-Device-ID"))

	// If no device ID (missing/empty/oversized), create a server-side fallback
	// fingerprint derived from request headers.
	if device == "" {
		userAgent := ctx.GetHeader("User-Agent")
		acceptLang := ctx.GetHeader("Accept-Language")
		acceptEnc := ctx.GetHeader("Accept-Encoding")

		// Simple fingerprint (you can make this more sophisticated)
		fingerprint := fmt.Sprintf("%s_%s_%s",
			hashString(userAgent),
			hashString(acceptLang),
			hashString(acceptEnc),
		)
		device = "fp_" + fingerprint
	}

	// Get real IP (considering proxies)
	ip := getRealClientIP(ctx)

	// Return pointers (nil if empty)
	if device != "" {
		deviceID = &device
	}
	if ip != "" {
		lastIP = &ip
	}

	return
}

// getRealClientIP gets real client IP considering proxies
func getRealClientIP(ctx *gin.Context) string {
	// Check X-Forwarded-For header first
	if xForwardedFor := ctx.GetHeader("X-Forwarded-For"); xForwardedFor != "" {
		// Get first IP from comma-separated list
		ips := strings.Split(xForwardedFor, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	if xRealIP := ctx.GetHeader("X-Real-IP"); xRealIP != "" {
		return xRealIP
	}

	// Fallback to RemoteAddr
	return ctx.ClientIP()
}

func hashString(s string) string {
	// Simple hash function (you can replace with a proper hash like SHA256)
	var hash int
	for _, char := range s {
		hash += int(char)
	}
	return fmt.Sprintf("%x", hash)
}

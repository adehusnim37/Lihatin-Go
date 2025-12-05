package ip

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

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
	APIKey := config.GetRequiredEnv(config.EnvIPGeoAPIKey)
	url := `https://api.ipgeolocation.io/v2/ipgeo?apiKey=` + APIKey + `&ip=` + ip
	method := "GET"

	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer res.Body.Close()

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
	if ip == "" || ip == "127.0.0.1" || ip == "::1" {
		return "Local Machine"
	}

	locationResponse, err := IPGeolocation(ip)
	if err != nil {
		return "Unknown Location"
	}

	var parts []string

	// Add city if available
	if locationResponse.Location.City != "" {
		parts = append(parts, locationResponse.Location.City)
	}

	// Add country name
	if locationResponse.Location.CountryName != "" {
		countryInfo := locationResponse.Location.CountryName

		// Add capital in parentheses if available
		if locationResponse.Location.CountryCapital != "" {
			countryInfo += " (" + locationResponse.Location.CountryCapital + ")"
		}

		parts = append(parts, countryInfo)
	}

	if len(parts) > 0 {
		return joinWithComma(parts)
	}

	return "Unknown Location"
}

func GetCountryName(ip string) string {
	if ip == "" || ip == "127.0.0.1" || ip == "::1" {
		return "Local Machine"
	}

	resultCh := make(chan string, 1)
	go func() {
		locationResponse, err := IPGeolocation(ip)
		if err != nil || locationResponse.Location.CountryName == "" {
			resultCh <- "Unknown Country"
			return
		}
		resultCh <- locationResponse.Location.CountryName
	}()
	return <-resultCh
}

func GetCityName(ip string) string {
	if ip == "" || ip == "127.0.0.1" || ip == "::1" {
		return "Local Machine"
	}

	resultCh := make(chan string, 1)
	go func() {
		locationResponse, err := IPGeolocation(ip)
		if err != nil || locationResponse.Location.City == "" {
			resultCh <- "Unknown City"
			return
		}
		resultCh <- locationResponse.Location.City
	}()
	return <-resultCh
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

// GetDeviceAndIPInfo extracts device ID and IP from request context
func GetDeviceAndIPInfo(ctx *gin.Context) (deviceID *string, lastIP *string) {
	// Try to get device ID from header
	device := ctx.GetHeader("X-Device-ID")

	// If no device ID, create fingerprint
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

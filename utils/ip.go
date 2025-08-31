package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	APIKey := GetRequiredEnv("IP_GEOLOCATION_API_KEY")
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

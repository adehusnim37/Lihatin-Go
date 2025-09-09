package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func testValidation() {
	baseURL := "http://localhost:8080"

	fmt.Println("=== TEST VALIDATION SYSTEM ===\n")

	testCases := []struct {
		name        string
		method      string
		endpoint    string
		body        interface{}
		description string
	}{
		{
			name:        "1. Create Short Link - Body Kosong",
			method:      "POST",
			endpoint:    "/v1/short/create",
			body:        map[string]string{},
			description: "Request dengan body kosong",
		},
		{
			name:        "2. Create Short Link - URL Invalid",
			method:      "POST",
			endpoint:    "/v1/short/create",
			body:        map[string]string{"original_url": "not-a-url"},
			description: "URL yang tidak valid",
		},
		{
			name:     "3. Create Short Link - Custom Code dengan Spasi",
			method:   "POST",
			endpoint: "/v1/short/create",
			body: map[string]interface{}{
				"original_url": "https://example.com",
				"custom_code":  "test code",
			},
			description: "Custom code mengandung spasi",
		},
		{
			name:        "4. Bulk Delete - Array Kosong",
			method:      "DELETE",
			endpoint:    "/v1/admin/shorts/bulk-delete",
			body:        map[string][]string{"codes": {}},
			description: "Array codes kosong",
		},
		{
			name:        "5. Bulk Delete - Item Duplikat",
			method:      "DELETE",
			endpoint:    "/v1/admin/shorts/bulk-delete",
			body:        map[string][]string{"codes": {"test1", "test2", "test1"}},
			description: "Array dengan item duplikat",
		},
		{
			name:        "6. Login - Password Kosong",
			method:      "POST",
			endpoint:    "/v1/auth/login",
			body:        map[string]string{"email_or_username": "test@example.com"},
			description: "Password tidak diisi",
		},
		{
			name:        "7. JSON Format Error",
			method:      "POST",
			endpoint:    "/v1/short/create",
			body:        `{"original_url": "https://example.com",}`, // Invalid JSON trailing comma
			description: "JSON format tidak valid",
		},
	}

	for _, tc := range testCases {
		fmt.Printf("### %s\n", tc.name)
		fmt.Printf("**Deskripsi:** %s\n", tc.description)

		var bodyReader io.Reader
		if strBody, ok := tc.body.(string); ok {
			bodyReader = strings.NewReader(strBody)
			fmt.Printf("**Request Body:** `%s`\n", strBody)
		} else {
			jsonBody, _ := json.Marshal(tc.body)
			bodyReader = bytes.NewBuffer(jsonBody)
			fmt.Printf("**Request Body:** `%s`\n", string(jsonBody))
		}

		req, err := http.NewRequest(tc.method, baseURL+tc.endpoint, bodyReader)
		if err != nil {
			fmt.Printf("**Error:** Failed to create request: %v\n\n", err)
			continue
		}

		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("**Error:** Request failed: %v\n\n", err)
			continue
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("**Error:** Failed to read response: %v\n\n", err)
			continue
		}

		// Pretty print JSON response
		var jsonResp map[string]interface{}
		if err := json.Unmarshal(respBody, &jsonResp); err == nil {
			prettyJSON, _ := json.MarshalIndent(jsonResp, "", "  ")
			fmt.Printf("**Response (%d):**\n```json\n%s\n```\n\n", resp.StatusCode, string(prettyJSON))
		} else {
			fmt.Printf("**Response (%d):** %s\n\n", resp.StatusCode, string(respBody))
		}
	}

	fmt.Println("=== TEST SELESAI ===")
}

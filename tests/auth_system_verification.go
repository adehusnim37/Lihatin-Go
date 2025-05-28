package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/adehusnim37/lihatin-go/models"
)

// SystemVerification provides methods to verify the authentication system
type SystemVerification struct {
	BaseURL string
}

// NewSystemVerification creates a new system verification instance
func NewSystemVerification(baseURL string) *SystemVerification {
	return &SystemVerification{
		BaseURL: baseURL,
	}
}

// VerifyRegistration tests user registration endpoint
func (sv *SystemVerification) VerifyRegistration() error {
	url := sv.BaseURL + "/api/auth/register"

	payload := models.RegisterRequest{
		Username: "testuser_" + fmt.Sprintf("%d", time.Now().Unix()),
		Email:    "test_" + fmt.Sprintf("%d", time.Now().Unix()) + "@example.com",
		Password: "TestPassword123!",
	}

	jsonData, _ := json.Marshal(payload)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("registration request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("registration failed with status: %d", resp.StatusCode)
	}

	fmt.Println("✓ User registration endpoint working")
	return nil
}

// VerifyLogin tests user login endpoint
func (sv *SystemVerification) VerifyLogin() error {
	// First register a user
	username := "logintest_" + fmt.Sprintf("%d", time.Now().Unix())
	email := "logintest_" + fmt.Sprintf("%d", time.Now().Unix()) + "@example.com"
	password := "TestPassword123!"

	regPayload := models.RegisterRequest{
		Username: username,
		Email:    email,
		Password: password,
	}

	jsonData, _ := json.Marshal(regPayload)
	regResp, err := http.Post(sv.BaseURL+"/api/auth/register", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("registration for login test failed: %w", err)
	}
	regResp.Body.Close()

	// Now try to login
	loginPayload := models.LoginRequest{
		Email:    email,
		Password: password,
	}

	jsonData, _ = json.Marshal(loginPayload)
	loginResp, err := http.Post(sv.BaseURL+"/api/auth/login", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	defer loginResp.Body.Close()

	if loginResp.StatusCode != http.StatusOK {
		return fmt.Errorf("login failed with status: %d", loginResp.StatusCode)
	}

	// Parse response to get token
	var loginResponse models.LoginResponse
	if err := json.NewDecoder(loginResp.Body).Decode(&loginResponse); err != nil {
		return fmt.Errorf("failed to parse login response: %w", err)
	}

	if loginResponse.AccessToken == "" {
		return fmt.Errorf("login response missing access token")
	}

	fmt.Println("✓ User login endpoint working")
	return nil
}

// VerifyProtectedEndpoint tests a protected endpoint with JWT token
func (sv *SystemVerification) VerifyProtectedEndpoint() error {
	// First register and login to get a token
	username := "protectedtest_" + fmt.Sprintf("%d", time.Now().Unix())
	email := "protectedtest_" + fmt.Sprintf("%d", time.Now().Unix()) + "@example.com"
	password := "TestPassword123!"

	// Register
	regPayload := models.RegisterRequest{
		Username: username,
		Email:    email,
		Password: password,
	}

	jsonData, _ := json.Marshal(regPayload)
	regResp, err := http.Post(sv.BaseURL+"/api/auth/register", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("registration for protected test failed: %w", err)
	}
	regResp.Body.Close()

	// Login
	loginPayload := models.LoginRequest{
		Email:    email,
		Password: password,
	}

	jsonData, _ = json.Marshal(loginPayload)
	loginResp, err := http.Post(sv.BaseURL+"/api/auth/login", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("login for protected test failed: %w", err)
	}
	defer loginResp.Body.Close()

	var loginResponse models.LoginResponse
	if err := json.NewDecoder(loginResp.Body).Decode(&loginResponse); err != nil {
		return fmt.Errorf("failed to parse login response: %w", err)
	}

	// Test protected endpoint
	req, err := http.NewRequest("GET", sv.BaseURL+"/api/auth/profile", nil)
	if err != nil {
		return fmt.Errorf("failed to create profile request: %w", err)
	}

	req.Header.Add("Authorization", "Bearer "+loginResponse.AccessToken)

	client := &http.Client{}
	profileResp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("profile request failed: %w", err)
	}
	defer profileResp.Body.Close()

	if profileResp.StatusCode != http.StatusOK {
		return fmt.Errorf("protected endpoint failed with status: %d", profileResp.StatusCode)
	}

	fmt.Println("✓ Protected endpoint authentication working")
	return nil
}

// VerifyAPIKeyEndpoint tests API key management endpoints
func (sv *SystemVerification) VerifyAPIKeyEndpoint() error {
	// First register and login to get a token
	username := "apikeytest_" + fmt.Sprintf("%d", time.Now().Unix())
	email := "apikeytest_" + fmt.Sprintf("%d", time.Now().Unix()) + "@example.com"
	password := "TestPassword123!"

	// Register
	regPayload := models.RegisterRequest{
		Username: username,
		Email:    email,
		Password: password,
	}

	jsonData, _ := json.Marshal(regPayload)
	regResp, err := http.Post(sv.BaseURL+"/api/auth/register", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("registration for API key test failed: %w", err)
	}
	regResp.Body.Close()

	// Login
	loginPayload := models.LoginRequest{
		Email:    email,
		Password: password,
	}

	jsonData, _ = json.Marshal(loginPayload)
	loginResp, err := http.Post(sv.BaseURL+"/api/auth/login", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("login for API key test failed: %w", err)
	}
	defer loginResp.Body.Close()

	var loginResponse models.LoginResponse
	if err := json.NewDecoder(loginResp.Body).Decode(&loginResponse); err != nil {
		return fmt.Errorf("failed to parse login response: %w", err)
	}

	// Test API key creation
	apiKeyPayload := models.APIKeyRequest{
		Name:        "Test API Key",
		Permissions: []string{"read", "write"},
	}

	jsonData, _ = json.Marshal(apiKeyPayload)
	req, err := http.NewRequest("POST", sv.BaseURL+"/api/api-keys/", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create API key request: %w", err)
	}

	req.Header.Add("Authorization", "Bearer "+loginResponse.AccessToken)
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	apiKeyResp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("API key creation request failed: %w", err)
	}
	defer apiKeyResp.Body.Close()

	if apiKeyResp.StatusCode != http.StatusCreated {
		return fmt.Errorf("API key creation failed with status: %d", apiKeyResp.StatusCode)
	}

	fmt.Println("✓ API key management endpoints working")
	return nil
}

// RunAllVerifications runs all system verification tests
func (sv *SystemVerification) RunAllVerifications() error {
	fmt.Println("Starting Authentication System Verification...")
	fmt.Println("===========================================")

	tests := []struct {
		name string
		fn   func() error
	}{
		{"Registration", sv.VerifyRegistration},
		{"Login", sv.VerifyLogin},
		{"Protected Endpoint", sv.VerifyProtectedEndpoint},
		{"API Key Management", sv.VerifyAPIKeyEndpoint},
	}

	for _, test := range tests {
		fmt.Printf("Testing %s...\n", test.name)
		if err := test.fn(); err != nil {
			return fmt.Errorf("%s test failed: %w", test.name, err)
		}
		time.Sleep(100 * time.Millisecond) // Small delay between tests
	}

	fmt.Println("===========================================")
	fmt.Println("✓ All authentication system tests passed!")
	return nil
}

// Example usage function
func ExampleUsage() {
	// Initialize the verifier
	verifier := NewSystemVerification("http://localhost:8080")

	// Run all verification tests
	if err := verifier.RunAllVerifications(); err != nil {
		fmt.Printf("❌ System verification failed: %v\n", err)
		return
	}

	fmt.Println("🎉 Authentication system is fully functional!")
}

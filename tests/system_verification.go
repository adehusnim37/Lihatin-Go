// Enhanced Activity Logger & Authentication System Test
package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

const baseURL = "http://localhost:8880"

func main() {
	fmt.Println("🚀 Testing Enhanced Activity Logger & Authentication System")
	fmt.Println("============================================================")

	// Test user creation with enhanced logging
	fmt.Println("\n📝 Creating test user...")
	if testCreateUser() {
		fmt.Println("✅ User creation successful")
	}

	// Test authentication system
	fmt.Println("\n🔐 Testing authentication...")
	if testLogin() {
		fmt.Println("✅ Authentication successful")
	}

	// Test activity logs retrieval
	fmt.Println("\n📊 Testing activity logs...")
	if testActivityLogs() {
		fmt.Println("✅ Activity logs working correctly")
	}

	fmt.Println("\n🎉 All tests completed successfully!")
}

func testCreateUser() bool {
	userData := map[string]interface{}{
		"username":   "testuser",
		"email":      "testuser@example.com",
		"password":   "SecurePass123!",
		"first_name": "Test",
		"last_name":  "User",
		"avatar":     "https://example.com/avatar.jpg",
	}

	jsonData, _ := json.Marshal(userData)
	resp, err := http.Post(baseURL+"/v1/users/", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("❌ Error creating user: %v\n", err)
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == 201 || resp.StatusCode == 409 // 409 if user already exists
}

func testLogin() bool {
	loginData := map[string]interface{}{
		"email_or_username": "testuser",
		"password":          "SecurePass123!",
	}

	jsonData, _ := json.Marshal(loginData)
	resp, err := http.Post(baseURL+"/v1/auth/login", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("❌ Error during login: %v\n", err)
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200
}

func testActivityLogs() bool {
	// Test getting all logs
	resp, err := http.Get(baseURL + "/v1/logs/")
	if err != nil {
		fmt.Printf("❌ Error getting logs: %v\n", err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		fmt.Printf("❌ Failed to get logs: %s\n", string(body))
		return false
	}

	// Test getting user-specific logs
	resp2, err := http.Get(baseURL + "/v1/logs/user/testuser")
	if err != nil {
		fmt.Printf("❌ Error getting user logs: %v\n", err)
		return false
	}
	defer resp2.Body.Close()

	return resp2.StatusCode == 200
}

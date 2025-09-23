package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// Example request struct with comprehensive validation
type BulkDeleteRequest struct {
	Codes []string `json:"codes" label:"Kode Short Link" binding:"required,min=1,max=100,unique,dive,required,no_space,alphanum"`
}

type UserRegistrationRequest struct {
	Username string `json:"username" label:"Nama Pengguna" binding:"required,min=3,max=20,username"`
	Email    string `json:"email" label:"Email" binding:"required,email"`
	Password string `json:"password" label:"Kata Sandi" binding:"required,pwdcomplex"`
}

func main() {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Setup custom validators
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		utils.SetupCustomValidators(v)
	}

	// Create router
	r := gin.New()

	// Add test endpoints
	r.POST("/bulk-delete", handleBulkDelete)
	r.POST("/register", handleUserRegistration)

	fmt.Println("=== CONTOH OUTPUT VALIDASI ===")

	// Test scenarios
	testScenarios := []struct {
		name        string
		endpoint    string
		requestBody string
		description string
	}{
		{
			name:        "1. Body Kosong",
			endpoint:    "/bulk-delete",
			requestBody: "",
			description: "Request tanpa body JSON",
		},
		{
			name:        "2. Slice Kosong",
			endpoint:    "/bulk-delete",
			requestBody: `{"codes": []}`,
			description: "Array codes kosong",
		},
		{
			name:        "3. Item Duplikat",
			endpoint:    "/bulk-delete",
			requestBody: `{"codes": ["abc123", "def456", "abc123", "ghi789"]}`,
			description: "Array dengan item duplikat",
		},
		{
			name:        "4. Tipe Data Salah",
			endpoint:    "/bulk-delete",
			requestBody: `{"codes": "not-an-array"}`,
			description: "Field codes bukan array",
		},
		{
			name:        "5. Item dengan Spasi",
			endpoint:    "/bulk-delete",
			requestBody: `{"codes": ["abc 123", "def456"]}`,
			description: "Item mengandung spasi",
		},
		{
			name:        "6. Format Item Tidak Valid",
			endpoint:    "/bulk-delete",
			requestBody: `{"codes": ["abc@123", "def456"]}`,
			description: "Item dengan karakter tidak diizinkan (harus alphanum)",
		},
		{
			name:        "7. Password Tidak Kompleks",
			endpoint:    "/register",
			requestBody: `{"username": "testuser", "email": "test@example.com", "password": "simple"}`,
			description: "Password tidak memenuhi kompleksitas",
		},
		{
			name:        "8. Username Invalid",
			endpoint:    "/register",
			requestBody: `{"username": "test user", "email": "test@example.com", "password": "Complex123!"}`,
			description: "Username mengandung spasi",
		},
	}

	for _, scenario := range testScenarios {
		fmt.Printf("### %s\n", scenario.name)
		fmt.Printf("**Deskripsi:** %s\n", scenario.description)
		fmt.Printf("**Request:** `%s`\n", scenario.requestBody)

		// Create request
		var req *http.Request
		if scenario.requestBody == "" {
			req = httptest.NewRequest("POST", scenario.endpoint, nil)
		} else {
			req = httptest.NewRequest("POST", scenario.endpoint, strings.NewReader(scenario.requestBody))
		}
		req.Header.Set("Content-Type", "application/json")

		// Create response recorder
		w := httptest.NewRecorder()

		// Serve request
		r.ServeHTTP(w, req)

		// Pretty print response
		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err == nil {
			prettyJSON, _ := json.MarshalIndent(response, "", "  ")
			fmt.Printf("**Response:**\n```json\n%s\n```\n\n", string(prettyJSON))
		} else {
			fmt.Printf("**Response:** %s\n\n", w.Body.String())
		}
	}
}

func handleBulkDelete(c *gin.Context) {
	var req BulkDeleteRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendValidationError(c, err, &req)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Bulk delete berhasil",
		"data": gin.H{
			"deleted_codes": req.Codes,
			"count":         len(req.Codes),
		},
	})
}

func handleUserRegistration(c *gin.Context) {
	var req UserRegistrationRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendValidationError(c, err, &req)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Registrasi berhasil",
		"data": gin.H{
			"username": req.Username,
			"email":    req.Email,
		},
	})
}

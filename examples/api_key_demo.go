package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/adehusnim37/lihatin-go/internal/pkg/auth"
)

func RunDemoAPIKEY() {
	fmt.Println("🔑 API Key Generation Demo")
	fmt.Println(strings.Repeat("=", 50))

	// Old method (deprecated)
	fmt.Println("\n📊 OLD METHOD (Deprecated):")
	oldKey, err := auth.GenerateAPIKey()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Generated Key: %s\n", oldKey)
	fmt.Printf("Length: %d characters\n", len(oldKey))
	fmt.Printf("Format Valid: %v\n", auth.ValidateAPIKeyFormat(oldKey))

	// New method (recommended)
	fmt.Println("\n🚀 NEW METHOD (Recommended):")
	keyID, secretKey, secretHash, keyPreview, err := auth.GenerateAPIKeyPair("")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Key ID: %s\n", keyID)
	fmt.Printf("Secret Key: %s\n", secretKey)
	fmt.Printf("Secret Hash: %s\n", secretHash[:20]+"...") // Show only first 20 chars of hash
	fmt.Printf("Key Preview: %s\n", keyPreview)
	fmt.Printf("Key ID Format Valid: %v\n", auth.ValidateAPIKeyIDFormat(keyID))

	// Test validation
	fmt.Println("\n🔐 VALIDATION TEST:")
	isValid := auth.ValidateAPISecretKey(secretKey, secretHash)
	fmt.Printf("Secret validation: %v\n", isValid)

	// Test with wrong secret
	isInvalid := auth.ValidateAPISecretKey("wrong_secret", secretHash)
	fmt.Printf("Wrong secret validation: %v\n", isInvalid)

	// Test preview generation
	fmt.Println("\n👁️ PREVIEW GENERATION:")
	preview1 := auth.GetKeyPreview("sk_lh_very_long_key_id_12345")
	preview2 := auth.GetKeyPreview("short")
	fmt.Printf("Long key preview: %s\n", preview1)
	fmt.Printf("Short key preview: %s\n", preview2)

	fmt.Println("\n✅ Demo completed successfully!")
}
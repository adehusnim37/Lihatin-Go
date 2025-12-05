package main

import (
	"fmt"
	"log"

	"github.com/adehusnim37/lihatin-go/internal/pkg/mail"
)

func main() {
	fmt.Println("Testing email service configuration...")

	emailService := mail.NewEmailService()

	// Test sending a simple email
	err := emailService.SendVerificationEmail("test@example.com", "Test User", "test-token-123")
	if err != nil {
		log.Printf("Error sending email: %v", err)
	} else {
		fmt.Println("Email sent successfully!")
	}
}

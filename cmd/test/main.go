package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/adehusnim37/lihatin-go/tests"
)

func main() {
	// Parse command line flags
	baseURL := flag.String("url", "http://localhost:8080", "Base URL for the authentication system")
	help := flag.Bool("help", false, "Show help message")
	flag.Parse()

	if *help {
		fmt.Println("Authentication System Test Runner")
		fmt.Println("=================================")
		fmt.Println("Usage: go run cmd/test/main.go [options]")
		fmt.Println()
		fmt.Println("Options:")
		flag.PrintDefaults()
		fmt.Println()
		fmt.Println("Example:")
		fmt.Println("  go run cmd/test/main.go -url http://localhost:8080")
		os.Exit(0)
	}

	fmt.Printf("Testing authentication system at: %s\n", *baseURL)
	fmt.Println()

	// Initialize the system verifier
	verifier := tests.NewSystemVerification(*baseURL)

	// Run all verification tests
	if err := verifier.RunAllVerifications(); err != nil {
		log.Fatalf("❌ System verification failed: %v", err)
	}

	fmt.Println()
	fmt.Println("🎉 Authentication system is fully functional!")
	fmt.Println("All endpoints are working correctly!")
}

package main

import (
	"fmt"
	"log"
	"testing"

	"github.com/adehusnim37/lihatin-go/models/user"
	"github.com/adehusnim37/lihatin-go/repositories"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/google/uuid"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestUserRepository(t *testing.T) {
	fmt.Println("🧪 Testing GORM User Repository")
	fmt.Println("===============================")

	// Setup database
	utils.GetEnvOrDefault("", "")
	dsn := utils.GetEnvOrDefault("DATABASE_URL", "")

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatal("Database connection failed:", err)
	}

	// Create repository
	repo := repositories.NewUserRepository(db)

	// Test: Get all users
	fmt.Println("\n📋 Testing GetAllUsers...")
	users, err := repo.GetAllUsers()
	if err != nil {
		fmt.Printf("❌ Failed: %s\n", err.Error())
	} else {
		fmt.Printf("✅ Success! Found %d users\n", len(users))
	}

	// Test: Create a test user
	fmt.Println("\n👤 Testing CreateUser...")
	testUser := &user.User{
		ID:        uuid.New().String(),
		Username:  "testuser_" + uuid.New().String()[:8],
		FirstName: "Test",
		LastName:  "User",
		Email:     "test_" + uuid.New().String()[:8] + "@example.com",
		Password:  "testpassword123",
		Role:      "user",
	}

	err = repo.CreateUser(testUser)
	if err != nil {
		fmt.Printf("❌ Failed: %s\n", err.Error())
	} else {
		fmt.Printf("✅ Success! Created user: %s\n", testUser.Email)

		// Test: Get user by ID
		fmt.Println("\n🔍 Testing GetUserByID...")
		foundUser, err := repo.GetUserByID(testUser.ID)
		if err != nil {
			fmt.Printf("❌ Failed: %s\n", err.Error())
		} else {
			fmt.Printf("✅ Success! Found user: %s (%s)\n", foundUser.Email, foundUser.Username)
		}

		// Test: Get user by email
		fmt.Println("\n📧 Testing GetUserByEmailOrUsername...")
		foundUser2, err := repo.GetUserByEmailOrUsername(testUser.Email)
		if err != nil {
			fmt.Printf("❌ Failed: %s\n", err.Error())
		} else {
			fmt.Printf("✅ Success! Found user by email: %s\n", foundUser2.Email)
		}

		// Test: Check if user is locked
		fmt.Println("\n🔒 Testing IsUserLocked...")
		isLocked, err := repo.IsUserLocked(testUser.ID)
		if err != nil {
			fmt.Printf("❌ Failed: %s\n", err.Error())
		} else {
			fmt.Printf("✅ Success! User locked status: %v\n", isLocked)
		}

		// Cleanup: Delete test user
		fmt.Println("\n🧹 Cleaning up...")
		err = repo.DeleteUserPermanent(testUser.ID)
		if err != nil {
			fmt.Printf("⚠️  Warning: Failed to cleanup: %s\n", err.Error())
		} else {
			fmt.Printf("✅ Test user cleaned up\n")
		}
	}

	fmt.Println("\n🎉 GORM User Repository test completed!")
}

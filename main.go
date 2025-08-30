package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/adehusnim37/lihatin-go/models/logging"
	"github.com/adehusnim37/lihatin-go/models/shortlink"
	"github.com/adehusnim37/lihatin-go/models/user"
	"github.com/adehusnim37/lihatin-go/routes"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/go-playground/validator/v10"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// runMigrations handles all model migrations
func runMigrations(db *gorm.DB) error {
	// Migrate User models
	if err := db.AutoMigrate(&user.User{}); err != nil {
		return fmt.Errorf("failed to migrate User model: %w", err)
	}

	if err := db.AutoMigrate(&user.UserAuth{}); err != nil {
		return fmt.Errorf("failed to migrate UserAuth model: %w", err)
	}

	if err := db.AutoMigrate(&user.AuthMethod{}); err != nil {
		return fmt.Errorf("failed to migrate AuthMethod model: %w", err)
	}

	if err := db.AutoMigrate(&user.APIKey{}); err != nil {
		return fmt.Errorf("failed to migrate APIKey model: %w", err)
	}

	// Migrate ShortLink models
	if err := db.AutoMigrate(&shortlink.ShortLink{}); err != nil {
		return fmt.Errorf("failed to migrate ShortLink model: %w", err)
	}

	if err := db.AutoMigrate(&shortlink.ShortLinkDetail{}); err != nil {
		return fmt.Errorf("failed to migrate ShortLinkDetail model: %w", err)
	}

	if err := db.AutoMigrate(&shortlink.ViewLinkDetail{}); err != nil {
		return fmt.Errorf("failed to migrate ViewLinkDetail model: %w", err)
	}

	// Migrate Logging models
	if err := db.AutoMigrate(&logging.ActivityLog{}); err != nil {
		return fmt.Errorf("failed to migrate ActivityLog model: %w", err)
	}

	log.Println("✅ All models migrated successfully!")
	return nil
}

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	// Initialize GORM for new auth repositories
	dsn := os.Getenv("DATABASE_URL") // Ambil dari environment variable
	if dsn == "" {
		log.Fatal("DATABASE_URL config is required")
	}
	// Setup database connection for sql.DB (existing code)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Printf("Please check your database connection during hitting the server")
		log.Printf("Error connecting to database: %v", err)
		panic(err)
	}
	defer db.Close()

	// Setup GORM connection for migrations
	gormDB, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Printf("Failed to connect database with GORM: %v", err)
		panic(err)
	}

	// Run Auto Migration
	log.Println("Running database migrations...")
	err = runMigrations(gormDB)
	if err != nil {
		log.Printf("Migration failed: %v", err)
		panic(err)
	}
	log.Println("Database migrations completed successfully!")

	// Initialize validator with custom rules
	validate := validator.New()
	utils.SetupCustomValidators(validate)

	r := routes.SetupRouter(db, validate)
	fmt.Println("Server running on localhost:8880 Please check your browser")
	r.Run(":8880")
}

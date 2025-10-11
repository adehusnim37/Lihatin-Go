package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/adehusnim37/lihatin-go/models/logging"
	"github.com/adehusnim37/lihatin-go/models/shortlink"
	"github.com/adehusnim37/lihatin-go/models/user"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	// Database connection
	dsn := "adehusnim:ryugamine123A@tcp(localhost:3306)/LihatinGo?charset=utf8mb4&parseTime=True&loc=Local"
	ensureDatabase()
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect database: %v", err)
	}

	// Check command line arguments
	args := os.Args[1:]
	if len(args) == 0 {
		log.Fatalf("Usage: go run cmd/migrate/main.go [migrate|rollback|fresh|status]")
	}

	command := args[0]

	switch command {
	case "migrate":
		runMigrations(db)
	case "fresh":
		dropAllTables(db)
		runMigrations(db)
	case "rollback":
		rollbackMigrations(db)
	case "status":
		showMigrationStatus(db)
	default:
		log.Fatalf("Unknown command: %s. Available: migrate, fresh, rollback, status", command)
	}
}

func ensureDatabase() {
	dsnRoot := "adehusnim:ryugamine123A@tcp(localhost:3306)/"
	db, err := sql.Open("mysql", dsnRoot)
	log.Printf("Connecting to database with DSN: %s", dsnRoot)
	if err != nil {
		log.Fatalf("Failed to connect root: %v", err)
	}
	log.Printf("Checking/creating database LihatinGo...")
	_, err = db.Exec("CREATE DATABASE IF NOT EXISTS LihatinGo DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;")
	if err != nil {
		log.Fatalf("Failed to create database: %v", err)
	}
	log.Printf("Database created/exists. Testing connection...")
	err = db.Close()
	if err != nil {
		return
	}
}

// runMigrations creates/updates all tables
func runMigrations(db *gorm.DB) {
	fmt.Println("🚀 Running database migrations...")

	models := []interface{}{
		// User models
		&user.User{},
		&user.UserAuth{},
		&user.AuthMethod{},
		&user.LoginAttempt{},
		&user.APIKey{},
		&user.HistoryUser{},

		// ShortLink models
		&shortlink.ShortLink{},
		&shortlink.ShortLinkDetail{},
		&shortlink.ViewLinkDetail{},

		// Logging models
		&logging.ActivityLog{},
	}

	for _, model := range models {
		if err := db.AutoMigrate(model); err != nil {
			log.Fatalf("❌ Failed to migrate %T: %v", model, err)
		}
		fmt.Printf("✅ Migrated: %T\n", model)
	}

	fmt.Println("🎉 All migrations completed successfully!")
}

// dropAllTables drops all tables (fresh migration)
func dropAllTables(db *gorm.DB) {
	fmt.Println("🗑️  Dropping all tables...")

	tables := []interface{}{
		&logging.ActivityLog{},
		&shortlink.ViewLinkDetail{},
		&shortlink.ShortLinkDetail{},
		&shortlink.ShortLink{},
		&user.APIKey{},
		&user.LoginAttempt{},
		&user.AuthMethod{},
		&user.UserAuth{},
		&user.User{},
	}

	for _, table := range tables {
		if err := db.Migrator().DropTable(table); err != nil {
			log.Printf("⚠️  Warning: Failed to drop %T: %v", table, err)
		} else {
			fmt.Printf("🗑️  Dropped: %T\n", table)
		}
	}

	fmt.Println("✅ All tables dropped!")
}

// rollbackMigrations drops all tables (simple rollback)
func rollbackMigrations(db *gorm.DB) {
	fmt.Println("⏪ Rolling back migrations...")
	dropAllTables(db)
	fmt.Println("🎉 Rollback completed!")
}

// showMigrationStatus shows current database status
func showMigrationStatus(db *gorm.DB) {
	fmt.Println("📊 Migration Status:")

	models := []interface{}{
		&user.User{},
		&user.UserAuth{},
		&user.AuthMethod{},
		&user.APIKey{},
		&shortlink.ShortLink{},
		&shortlink.ShortLinkDetail{},
		&shortlink.ViewLinkDetail{},
		&logging.ActivityLog{},
	}

	for _, model := range models {
		exists := db.Migrator().HasTable(model)
		status := "❌ Missing"
		if exists {
			status = "✅ Exists"
		}
		fmt.Printf("%s %T\n", status, model)
	}
}

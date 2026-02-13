package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
	"github.com/adehusnim37/lihatin-go/models/logging"
	"github.com/adehusnim37/lihatin-go/models/shortlink"
	"github.com/adehusnim37/lihatin-go/models/user"
	"github.com/go-sql-driver/mysql"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	// Load environment variables
	dsn := config.GetRequiredEnv(config.EnvDatabaseURL)

	ensureDatabase(dsn)
	db, err := gorm.Open(gormmysql.Open(dsn), &gorm.Config{})
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

func ensureDatabase(dsn string) {
	// Parse DSN to get config
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		log.Fatalf("Failed to parse DSN: %v", err)
	}

	// Store original DB name and clear it to connect to root
	dbName := cfg.DBName
	cfg.DBName = ""
	dsnRoot := cfg.FormatDSN()

	db, err := sql.Open("mysql", dsnRoot)
	log.Printf("Connecting to database server...")
	if err != nil {
		log.Fatalf("Failed to connect root: %v", err)
	}
	defer db.Close()

	log.Printf("Checking/creating database %s...", dbName)
	query := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;", dbName)
	_, err = db.Exec(query)
	if err != nil {
		log.Fatalf("Failed to create database: %v", err)
	}
	log.Printf("Database %s created/exists.", dbName)
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
		&user.PremiumKey{},
		&user.PremiumKeyUsage{},

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
		&user.PremiumKey{},
		&user.PremiumKeyUsage{},
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
		&user.PremiumKey{},
		&user.PremiumKeyUsage{},
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

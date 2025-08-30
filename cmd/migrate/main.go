package main

import (
	"fmt"
	"log"
	"os"

	"github.com/adehusnim37/lihatin-go/models/logging"
	"github.com/adehusnim37/lihatin-go/models/migrations"
	"github.com/adehusnim37/lihatin-go/models/shortlink"
	"github.com/adehusnim37/lihatin-go/models/user"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	// Database connection
	dsn := "adehusnim:ryugamine123A@tcp(localhost:3306)/LihatinGo?charset=utf8mb4&parseTime=True&loc=Local"
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

		// ShortLink models
		&shortlink.ShortLink{},
		&shortlink.ShortLinkDetail{},
		&shortlink.ViewLinkDetail{},

		// Logging models
		&logging.ActivityLog{},

		// Migration tracking
		&migrations.Migration{},
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

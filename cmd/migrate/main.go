package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	// Database connection - using same credentials as main app
	db, err := sql.Open("mysql", "adehusnim:ryugamine123A@tcp(localhost:3306)/LihatinGo?parseTime=true")
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	fmt.Println("Connected to database successfully!")

	// Get migrations directory
	migrationsDir := "./migrations"
	if len(os.Args) > 1 {
		migrationsDir = os.Args[1]
	}

	// Run migrations
	err = runMigrations(db, migrationsDir)
	if err != nil {
		log.Fatal("Migration failed:", err)
	}

	fmt.Println("All migrations completed successfully!")
}

func runMigrations(db *sql.DB, migrationsDir string) error {
	// Create migrations table if it doesn't exist
	createMigrationsTable := `
	CREATE TABLE IF NOT EXISTS migrations (
		id INT AUTO_INCREMENT PRIMARY KEY,
		migration_name VARCHAR(255) NOT NULL UNIQUE,
		applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	_, err := db.Exec(createMigrationsTable)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %v", err)
	}

	// Read migration files
	files, err := ioutil.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %v", err)
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) != ".sql" {
			continue
		}

		migrationName := file.Name()

		// Check if migration already applied
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM migrations WHERE migration_name = ?", migrationName).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to check migration status: %v", err)
		}

		if count > 0 {
			fmt.Printf("Migration %s already applied, skipping...\n", migrationName)
			continue
		}

		// Read and execute migration file
		migrationPath := filepath.Join(migrationsDir, migrationName)
		content, err := ioutil.ReadFile(migrationPath)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %v", migrationName, err)
		}

		fmt.Printf("Applying migration: %s\n", migrationName)

		// Execute migration (split by semicolons for multiple statements)
		statements := splitSQL(string(content))
		for _, stmt := range statements {
			if stmt == "" {
				continue
			}
			_, err = db.Exec(stmt)
			if err != nil {
				return fmt.Errorf("failed to execute migration %s: %v\nStatement: %s", migrationName, err, stmt)
			}
		}

		// Record migration as applied
		_, err = db.Exec("INSERT INTO migrations (migration_name) VALUES (?)", migrationName)
		if err != nil {
			return fmt.Errorf("failed to record migration %s: %v", migrationName, err)
		}

		fmt.Printf("Migration %s applied successfully!\n", migrationName)
	}

	return nil
}

// Simple SQL statement splitter (handles basic cases)
func splitSQL(content string) []string {
	var statements []string
	var current string

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip comments and empty lines
		if strings.HasPrefix(line, "--") || line == "" {
			continue
		}

		current += line + " "

		// Check if statement ends with semicolon
		if strings.HasSuffix(strings.TrimSpace(line), ";") {
			statements = append(statements, strings.TrimSpace(current))
			current = ""
		}
	}

	// Add final statement if it doesn't end with semicolon
	if strings.TrimSpace(current) != "" {
		statements = append(statements, strings.TrimSpace(current))
	}

	return statements
}

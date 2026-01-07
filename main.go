package main

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/adehusnim37/lihatin-go/internal/jobs"
	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
	"github.com/adehusnim37/lihatin-go/internal/pkg/migrations"
	appvalidator "github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	"github.com/adehusnim37/lihatin-go/middleware"
	"github.com/adehusnim37/lihatin-go/routes"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {

	// Initialize GORM for new auth repositories
	_ = godotenv.Load() // ignore error kalau .env ga ada
	dsn := config.GetRequiredEnv(config.EnvDatabaseURL)
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
	err = migrations.RunMigrations(gormDB)
	if err != nil {
		log.Printf("Migration failed: %v", err)
		panic(err)
	}
	log.Println("Database migrations completed successfully!")

	// Initialize Redis session manager
	log.Println("Initializing session manager...")
	if err := middleware.InitSessionManager(); err != nil {
		log.Printf("Failed to initialize session manager: %v", err)
		log.Println("❌ Redis belum aktif! Pastikan Docker Redis sudah running sebelum start server. Bila tidak, fitur session seperti login tidak akan berfungsi.")
		panic("Application stopped: Redis session manager is required for authentication/session features.")
	} else {
		log.Println("✅ Session manager initialized successfully!")
	}

	log.Println("🔁 Initializing scheduler...")
	scheduler := jobs.NewScheduler(gormDB)
	log.Println("⬇️ Registering jobs...")
	scheduler.Register(jobs.NewDeactivateExpiredLinksJob(gormDB))
	scheduler.Register(jobs.NewCleanupLoginAttemptsJob(gormDB))
	log.Println("⬇ Starting scheduler...")
	scheduler.Start()
	log.Println("✅ Scheduler started successfully!")
	defer scheduler.Stop()

	// Minimal validator instance untuk backward compatibility dengan controller lama
	validate := validator.New()
	appvalidator.SetupCustomValidators(validate)

	// ✅ TAMBAHKAN INI: Override Gin's default validator dengan custom validator
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		appvalidator.SetupCustomValidators(v)
	}

	r := routes.SetupRouter(validate)
	fmt.Println("Server running on port:", config.GetRequiredEnv(config.EnvAppPort))
	r.Run(config.GetRequiredEnv(config.EnvAppPort))
}

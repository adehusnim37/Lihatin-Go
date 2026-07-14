package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/adehusnim37/lihatin-go/internal/jobs"
	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
	"github.com/adehusnim37/lihatin-go/internal/pkg/disposable"
	"github.com/adehusnim37/lihatin-go/internal/pkg/migrations"
	appvalidator "github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	"github.com/adehusnim37/lihatin-go/middleware"
	"github.com/adehusnim37/lihatin-go/routes"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func configureGinMode() {
	// Allow explicit GIN_MODE override from environment.
	if mode := strings.TrimSpace(os.Getenv("GIN_MODE")); mode != "" {
		gin.SetMode(mode)
		return
	}

	// Fallback policy: production ENV defaults to release mode.
	env := strings.ToLower(strings.TrimSpace(config.GetEnvOrDefault(config.Env, "development")))
	if env == "production" {
		gin.SetMode(gin.ReleaseMode)
		return
	}

	gin.SetMode(gin.DebugMode)
}

func main() {
	configureGinMode()

	// Initialize GORM for new auth repositories
	envErr := godotenv.Load() // ignore error kalau .env ga ada
	if envErr != nil {
		log.Printf("Error loading .env file, Please check your environment variables: %v", envErr)
	}
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

	log.Println("📧 Initializing disposable email policy...")
	if err := disposable.InitGlobal(gormDB, middleware.GetSessionManager().GetRedisClient()); err != nil {
		log.Printf("Failed to initialize disposable email policy: %v", err)
		panic(err)
	}
	log.Println("✅ Disposable email policy initialized")

	log.Println("🔁 Initializing scheduler...")
	scheduler, err := jobs.NewScheduler(gormDB)
	if err != nil {
		log.Printf("Failed to initialize scheduler: %v", err)
		panic(err)
	}
	log.Println("⬇️ Registering jobs...")
	if err := scheduler.RegisterMany(
		jobs.NewDeactivateExpiredLinksJob(gormDB),
		jobs.NewCleanupLoginAttemptsJob(gormDB),
		jobs.NewRefreshDisposableDomainsJob(),
	); err != nil {
		log.Printf("Failed to register scheduler jobs: %v", err)
		panic(err)
	}
	log.Println("⬇ Starting scheduler...")
	scheduler.Start()
	log.Println("✅ Scheduler started successfully!")
	defer scheduler.Stop()

	// Minimal validator instance untuk backward compatibility dengan controller lama
	validate := validator.New()
	if err := appvalidator.SetupCustomValidators(validate); err != nil {
		log.Printf("Failed to register custom validators: %v", err)
		panic(err)
	}

	// ✅ TAMBAHKAN INI: Override Gin's default validator dengan custom validator
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		if err := appvalidator.SetupCustomValidators(v); err != nil {
			log.Printf("Failed to register custom validators for Gin binding validator: %v", err)
			panic(err)
		}
	}

	r := routes.SetupRouter(validate)
	fmt.Println("Server running on port:", config.GetRequiredEnv(config.EnvAppPort))
	r.Run(config.GetRequiredEnv(config.EnvAppPort))
}

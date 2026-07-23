package routes

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/controllers"
	"github.com/adehusnim37/lihatin-go/controllers/auth"
	"github.com/adehusnim37/lihatin-go/controllers/auth/email"
	"github.com/adehusnim37/lihatin-go/controllers/auth/totp"
	"github.com/adehusnim37/lihatin-go/controllers/logger"
	"github.com/adehusnim37/lihatin-go/controllers/shortlink"
	pkgauth "github.com/adehusnim37/lihatin-go/internal/pkg/auth"
	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
	"github.com/adehusnim37/lihatin-go/internal/pkg/csrf"
	"github.com/adehusnim37/lihatin-go/middleware"
	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/adehusnim37/lihatin-go/repositories/authrepo"
	"github.com/adehusnim37/lihatin-go/repositories/loggerrepo"
	"github.com/adehusnim37/lihatin-go/repositories/userrepo"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func csrfTokenSkipRules() []csrf.SkipRule {
	return []csrf.SkipRule{
		// Public auth bootstrap. Origin/Referer validation still applies.
		{Method: http.MethodPost, Path: "/v1/auth/login"},
		{Method: http.MethodPost, Path: "/v1/auth/oauth/google/start"},
		{Method: http.MethodPost, Path: "/v1/auth/oauth/google/callback"},
		{Method: http.MethodPost, Path: "/v1/auth/signup/start"},
		{Method: http.MethodPost, Path: "/v1/auth/signup/resend-otp"},
		{Method: http.MethodPost, Path: "/v1/auth/signup/verify-otp"},
		{Method: http.MethodPost, Path: "/v1/auth/signup/complete"},
		{Method: http.MethodPost, Path: "/v1/auth/login/email-otp/verify"},
		{Method: http.MethodPost, Path: "/v1/auth/login/email-otp/resend"},
		{Method: http.MethodPost, Path: "/v1/auth/forgot-password"},
		{Method: http.MethodPost, Path: "/v1/auth/reset-password"},
		{Method: http.MethodPost, Path: "/v1/auth/verify-totp-login"},

		// Public support access & conversation.
		{Method: http.MethodPost, Path: "/v1/support/tickets"},
		{Method: http.MethodGet, Path: "/v1/support/track"},
		{Method: http.MethodPost, Path: "/v1/support/access/request-otp"},
		{Method: http.MethodPost, Path: "/v1/support/access/resend-otp"},
		{Method: http.MethodPost, Path: "/v1/support/access/verify-otp"},
		{Method: http.MethodPost, Path: "/v1/support/access/verify-code"},
		{Method: http.MethodGet, Path: "/v1/support/tickets/:ticketCode/messages"},
		{Method: http.MethodPost, Path: "/v1/support/tickets/:ticketCode/messages"},
		{Method: http.MethodGet, Path: "/v1/support/tickets/:ticketCode/attachments/:attachmentID"},

		// API-key clients do not use browser cookies and are not CSRF targets.
		// Their route middleware still validates the API key after this bypass.
		{Method: http.MethodPost, Path: "/v1/api/short", SkipOriginCheck: true},
		{Method: http.MethodPut, Path: "/v1/api/short/:code", SkipOriginCheck: true},
		{Method: http.MethodDelete, Path: "/v1/api/short/:code", SkipOriginCheck: true},
	}
}

func SetupRouter(validate *validator.Validate) *gin.Engine {
	// Use gin.New and attach middleware once to avoid duplicate default middleware warnings.
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.BlockSensitivePaths())

	// Apply CORS middleware first
	r.Use(pkgauth.CORSMiddleware())

	// Keep CSRF enabled in every environment so development and CI exercise
	// the same browser-security contract as production.
	csrfOpts := csrf.DefaultOptions()
	csrfOpts.SkipRules = csrfTokenSkipRules()
	r.Use(csrf.Middleware(csrfOpts))

	// Initialize GORM for new auth repositories
	dsn := config.GetRequiredEnv(config.EnvDatabaseURL)
	gormDB, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("Failed to connect to database with GORM: " + err.Error())
	}

	// Create base controller with GORM for auth
	baseController := controllers.NewBaseControllerWithGorm(gormDB, validate)

	// Inisialisasi controller spesifik
	authController := auth.NewAuthController(baseController)
	loggerController := logger.NewController(baseController)
	shortController := shortlink.NewController(baseController)
	emailController := email.NewController(baseController)
	totpController := totp.NewController(baseController)

	// Setup repositories for middleware
	loggerRepo := loggerrepo.NewLoggerRepository(gormDB)
	userRepo := userrepo.NewUserRepository(gormDB) // Changed to use GORM
	userAuthRepo := authrepo.NewUserAuthRepository(gormDB)
	loginAttemptRepo := authrepo.NewLoginAttemptRepository(gormDB)
	authRepo := authrepo.NewAuthRepository(gormDB) // Add auth repository for API key middleware

	// Apply global middleware for activity logging
	r.Use(middleware.ActivityLogger(loggerRepo))

	// Definisikan route untuk user, auth, dan logger
	v1 := r.Group("/v1")

	// CSRF Token endpoint (untuk SPA fetch token)
	v1.GET("/csrf-token", csrfTokenHandler)

	RegisterAuthRoutes(v1, authController, userRepo, userAuthRepo, *loginAttemptRepo, emailController, totpController, baseController)
	RegisterSupportRoutes(v1, userRepo, userAuthRepo, baseController)
	RegisterLoggerRoutes(v1, userRepo, userAuthRepo, loggerController)
	RegisterShortRoutes(v1, shortController, userRepo, userAuthRepo, authRepo)

	// Route health check
	v1.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, common.APIResponse{
			Success: true,
			Data:    "OK",
			Message: "Health check passed",
			Error:   nil,
		})
	})

	return r
}

func csrfTokenHandler(c *gin.Context) {
	c.Header("Cache-Control", "no-store, max-age=0")
	c.Header("Pragma", "no-cache")
	token := csrf.GetMaskedToken(c)
	c.JSON(http.StatusOK, common.APIResponse{
		Success: true,
		Data: gin.H{
			"csrfToken": token,
		},
		Message: "CSRF token generated",
		Error:   nil,
	})
}

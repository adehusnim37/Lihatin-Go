package routes

import (
	"net/http"
	"strings"

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
	"github.com/adehusnim37/lihatin-go/repositories/announcementrepo"
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
		// Authentication bootstrap routes still enforce origin validation.
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
		{Method: http.MethodPost, Path: "/v1/auth/passkeys/login/begin"},
		{Method: http.MethodPost, Path: "/v1/auth/passkeys/login/finish"},
		// Signed one-click email endpoint; it does not use browser cookies.
		{Method: http.MethodPost, Path: "/v1/notifications/unsubscribe", SkipOriginCheck: true},

		// Public support access and conversation routes.
		{Method: http.MethodPost, Path: "/v1/support/tickets"},
		{Method: http.MethodPost, Path: "/v1/support/access/request-otp"},
		{Method: http.MethodPost, Path: "/v1/support/access/resend-otp"},
		{Method: http.MethodPost, Path: "/v1/support/access/verify-otp"},
		{Method: http.MethodPost, Path: "/v1/support/access/verify-code"},
		{Method: http.MethodGet, Path: "/v1/support/tickets/:ticketCode/messages"},
		{Method: http.MethodPost, Path: "/v1/support/tickets/:ticketCode/messages"},
		{Method: http.MethodGet, Path: "/v1/support/tickets/:ticketCode/attachments/:attachmentID"},
		{Method: http.MethodPost, Path: "/v1/support/tickets/:ticketCode/access/logout"},
		// Recovery requires the ticket-scoped HttpOnly support session; origin checks remain enabled.
		{Method: http.MethodPost, Path: "/v1/support/tickets/:ticketCode/recovery/otp"},
		{Method: http.MethodPost, Path: "/v1/support/tickets/:ticketCode/recovery/verify-otp"},

		// API-key clients do not use browser cookies or require CSRF protection.
		{Method: http.MethodPost, Path: "/v1/api/short", SkipOriginCheck: true},
		{Method: http.MethodPut, Path: "/v1/api/short/:code", SkipOriginCheck: true},
		{Method: http.MethodDelete, Path: "/v1/api/short/:code", SkipOriginCheck: true},
	}
}

func SetupRouter(validate *validator.Validate) *gin.Engine {
	r := gin.New()

	// Trust forwarded client IPs only from explicitly configured proxies.
	proxyCIDRs := strings.TrimSpace(config.GetEnvOrDefault("TRUSTED_PROXY_CIDRS", ""))
	var trustedProxies []string
	if proxyCIDRs != "" {
		for _, cidr := range strings.Split(proxyCIDRs, ",") {
			trustedProxies = append(trustedProxies, strings.TrimSpace(cidr))
		}
	}
	if err := r.SetTrustedProxies(trustedProxies); err != nil {
		panic("Invalid TRUSTED_PROXY_CIDRS: " + err.Error())
	}
	// Global middleware.
	r.Use(gin.Logger(), gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.BlockSensitivePaths())
	r.Use(pkgauth.CORSMiddleware())

	// Keep CSRF enabled in every environment.
	csrfOpts := csrf.DefaultOptions()
	csrfOpts.SkipRules = csrfTokenSkipRules()
	r.Use(csrf.Middleware(csrfOpts))

	// Database and base controller.
	dsn := config.GetRequiredEnv(config.EnvDatabaseURL)
	gormDB, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("Failed to connect to database with GORM: " + err.Error())
	}

	baseController := controllers.NewBaseControllerWithGorm(gormDB, validate)

	// Controllers.
	authController := auth.NewAuthController(baseController)
	loggerController := logger.NewController(baseController)
	shortController := shortlink.NewController(baseController)
	emailController := email.NewController(baseController)
	totpController := totp.NewController(baseController, authController)

	// Repositories used by middleware and route registration.
	loggerRepo := loggerrepo.NewLoggerRepository(gormDB)
	userRepo := userrepo.NewUserRepository(gormDB)
	userAuthRepo := authrepo.NewUserAuthRepository(gormDB)
	loginAttemptRepo := authrepo.NewLoginAttemptRepository(gormDB)
	authRepo := authrepo.NewAuthRepository(gormDB)
	announcementRepo := announcementrepo.NewInAppRepository(gormDB)

	// Activity logging must run after its repository is initialized.
	r.Use(middleware.ActivityLogger(loggerRepo))

	v1 := r.Group("/v1")
	v1.GET("/csrf-token", csrfTokenHandler)

	RegisterAuthRoutes(
		v1,
		authController,
		userRepo,
		userAuthRepo,
		*loginAttemptRepo,
		emailController,
		totpController,
		baseController,
	)
	RegisterSupportRoutes(v1, userRepo, userAuthRepo, baseController)
	RegisterLoggerRoutes(v1, userRepo, userAuthRepo, loggerController)
	RegisterShortRoutes(v1, shortController, userRepo, userAuthRepo, authRepo)
	RegisterNotificationRoutes(v1, baseController, userRepo, userAuthRepo, announcementRepo)

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

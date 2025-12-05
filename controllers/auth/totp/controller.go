package totp

import (
	"github.com/adehusnim37/lihatin-go/controllers"
	"github.com/adehusnim37/lihatin-go/repositories"
	"github.com/adehusnim37/lihatin-go/internal/pkg/mail"
)

// Controller handles TOTP/2FA-related authentication operations
type Controller struct {
	*controllers.BaseController
	repo         *repositories.AuthRepository
	emailService *mail.EmailService
}

// NewController creates a new TOTP authentication controller
func NewController(base *controllers.BaseController) *Controller {
	if base.GormDB == nil {
		panic("GormDB is required for TOTPController")
	}

	authRepo := repositories.NewAuthRepository(base.GormDB)
	emailService := mail.NewEmailService()

	return &Controller{
		BaseController: base,
		repo:           authRepo,
		emailService:   emailService,
	}
}

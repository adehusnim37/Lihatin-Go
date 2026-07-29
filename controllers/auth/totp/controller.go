package totp

import (
	"github.com/adehusnim37/lihatin-go/controllers"
	authcontroller "github.com/adehusnim37/lihatin-go/controllers/auth"
	"github.com/adehusnim37/lihatin-go/internal/pkg/mail"
	"github.com/adehusnim37/lihatin-go/repositories/authrepo"
)

// Controller handles TOTP/2FA-related authentication operations
type Controller struct {
	*controllers.BaseController
	repo            *authrepo.AuthRepository
	loginController *authcontroller.Controller
	emailService    *mail.EmailService
}

// NewController creates a new TOTP authentication controller
func NewController(base *controllers.BaseController, loginController *authcontroller.Controller) *Controller {
	if base.GormDB == nil {
		panic("GormDB is required for TOTPController")
	}
	if loginController == nil {
		panic("AuthController is required for TOTPController")
	}

	authRepo := authrepo.NewAuthRepository(base.GormDB)
	emailService := mail.NewEmailService()

	return &Controller{
		BaseController:  base,
		repo:            authRepo,
		loginController: loginController,
		emailService:    emailService,
	}
}

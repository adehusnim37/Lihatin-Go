package email

import (
	"github.com/adehusnim37/lihatin-go/controllers"
	"github.com/adehusnim37/lihatin-go/repositories"
	"github.com/adehusnim37/lihatin-go/utils/mail"
)

// Controller handles email-related authentication operations
type Controller struct {
	*controllers.BaseController
	repo         *repositories.AuthRepository
	emailService *mail.EmailService
}

// NewController creates a new email authentication controller
func NewController(base *controllers.BaseController) *Controller {
	if base.GormDB == nil {
		panic("GormDB is required for EmailController")
	}

	authRepo := repositories.NewAuthRepository(base.GormDB)
	emailService := mail.NewEmailService()

	return &Controller{
		BaseController: base,
		repo:           authRepo,
		emailService:   emailService,
	}
}

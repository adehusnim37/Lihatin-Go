package premium

import (
	"github.com/adehusnim37/lihatin-go/controllers"
	"github.com/adehusnim37/lihatin-go/internal/pkg/mail"
	"github.com/adehusnim37/lihatin-go/repositories/userrepo"
)

type Controller struct {
	*controllers.BaseController
	premiumRepo *userrepo.UserPremiumKeyRepository
	userRepo    userrepo.UserRepository
	emailSvc    *mail.EmailService
}

func NewPremiumController(base *controllers.BaseController) *Controller {
	if base.GormDB == nil {
		panic("GormDB is required for PremiumController")
	}

	premiumRepo := userrepo.NewUserPremiumKeyRepository(base.GormDB)
	userRepo := userrepo.NewUserRepository(base.GormDB)
	emailSvc := mail.NewEmailService()

	return &Controller{
		BaseController: base,
		premiumRepo:    premiumRepo,
		userRepo:       userRepo,
		emailSvc:       emailSvc,
	}
}

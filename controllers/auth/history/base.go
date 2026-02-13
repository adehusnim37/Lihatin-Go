package history

import (
	"github.com/adehusnim37/lihatin-go/controllers"
	"github.com/adehusnim37/lihatin-go/repositories/userrepo"
)

type Controller struct {
	*controllers.BaseController
	userRepo userrepo.UserRepository
}

func NewHistoryUserController(base *controllers.BaseController) *Controller {
	if base.GormDB == nil {
		panic("GormDB is required for PremiumController")
	}

	userRepo := userrepo.NewUserRepository(base.GormDB)

	return &Controller{
		BaseController: base,
		userRepo:       userRepo,
	}
}

package loginattempts

import (
	"github.com/adehusnim37/lihatin-go/controllers"
	"github.com/adehusnim37/lihatin-go/repositories/authrepo"
)

// Controller handles login attempts operations
type Controller struct {
	*controllers.BaseController
	repo *authrepo.AuthRepository
}

// NewLoginAttemptsController creates a new login attempts controller
func NewLoginAttemptsController(base *controllers.BaseController) *Controller {
	if base.GormDB == nil {
		panic("GormDB is required for LoginAttemptsController")
	}
	authRepo := authrepo.NewAuthRepository(base.GormDB)
	return &Controller{
		BaseController: base,
		repo:           authRepo,
	}
}

package history

import (
	"github.com/adehusnim37/lihatin-go/controllers"
	"github.com/adehusnim37/lihatin-go/repositories/authrepo"
)

// Controller handles email-related authentication operations
type Controller struct {
	*controllers.BaseController
	repo         *authrepo.AuthRepository
	historyRepo  *authrepo.HistoryUserRepository
}

// NewHistoryUserController creates a new email authentication controller
func NewHistoryUserController(base *controllers.BaseController) *Controller {
	if base.GormDB == nil {
		panic("GormDB is required for EmailController")
	}

	authRepo := authrepo.NewAuthRepository(base.GormDB)
	historyRepo := authrepo.NewHistoryUserRepository(base.GormDB)

	return &Controller{
		BaseController: base,
		repo:           authRepo,
		historyRepo:    historyRepo,
	}
}

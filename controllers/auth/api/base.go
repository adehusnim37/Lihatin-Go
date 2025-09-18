package api

import (
	"github.com/adehusnim37/lihatin-go/controllers"
	"github.com/adehusnim37/lihatin-go/repositories"
)

// Controller provides all handlers for API key operations
type Controller struct {
	*controllers.BaseController
	repo *repositories.AuthRepository
}

// NewAPIKeyController creates a new API key controller instance
func NewAPIKeyController(base *controllers.BaseController) *Controller {
	if base.GormDB == nil {
		panic("GormDB is required for APIKeyController")
	}

	authRepo := repositories.NewAuthRepository(base.GormDB)

	return &Controller{
		BaseController: base,
		repo:           authRepo,
	}
}

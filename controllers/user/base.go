package user

import (
	"github.com/adehusnim37/lihatin-go/controllers"
	"github.com/adehusnim37/lihatin-go/repositories"
)

// Controller menyediakan semua handler untuk operasi user
type Controller struct {
	*controllers.BaseController
	repo repositories.UserRepository
}

// NewController membuat instance baru controller user
func NewController(base *controllers.BaseController) *Controller {
	userRepo := repositories.NewUserRepository(base.DB)
	return &Controller{
		BaseController: base,
		repo:           userRepo,
	}
}

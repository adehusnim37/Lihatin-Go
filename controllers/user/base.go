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
	// Use GORM database for user repository
	var userRepo repositories.UserRepository
	if base.GormDB != nil {
		userRepo = repositories.NewUserRepository(base.GormDB)
	} else {
		// Fallback to SQL DB if GORM is not available (shouldn't happen in normal flow)
		panic("GORM database connection is required for user repository")
	}
	return &Controller{
		BaseController: base,
		repo:           userRepo,
	}
}

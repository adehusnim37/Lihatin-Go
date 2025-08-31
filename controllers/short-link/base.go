package shortlink

import (
	"github.com/adehusnim37/lihatin-go/controllers"
	shortlinkrepo "github.com/adehusnim37/lihatin-go/repositories/short-link"
)

// Controller menyediakan semua handler untuk operasi short link
type Controller struct {
	*controllers.BaseController
	repo *shortlinkrepo.ShortLinkRepository
}

// NewController membuat instance baru controller short link
func NewController(base *controllers.BaseController) *Controller {
	shortLinkRepo := shortlinkrepo.NewShortLinkRepository(base.GormDB)
	return &Controller{
		BaseController: base,
		repo:           shortLinkRepo,
	}
}

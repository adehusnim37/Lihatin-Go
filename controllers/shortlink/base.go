package shortlink

import (
	"github.com/adehusnim37/lihatin-go/controllers"
	"github.com/adehusnim37/lihatin-go/internal/pkg/mail"
	shortlinkrepo "github.com/adehusnim37/lihatin-go/repositories/shortlink"
)

// Controller menyediakan semua handler untuk operasi short link
type Controller struct {
	*controllers.BaseController
	repo         *shortlinkrepo.ShortLinkRepository
	emailService *mail.EmailService
}

// NewController membuat instance baru controller short link
func NewController(base *controllers.BaseController) *Controller {
	shortLinkRepo := shortlinkrepo.NewShortLinkRepository(base.GormDB)
	emailService := mail.NewEmailService()
	return &Controller{
		BaseController: base,
		repo:           shortLinkRepo,
		emailService:   emailService,
	}
}

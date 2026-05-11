package logger

import (
	"github.com/adehusnim37/lihatin-go/controllers"
	"github.com/adehusnim37/lihatin-go/repositories/loggerrepo"
)

// Controller handles logging operations
type Controller struct {
	*controllers.BaseController
	repo *loggerrepo.LoggerRepository
}

// NewController creates a new logger controller
func NewController(base *controllers.BaseController) *Controller {
	loggerRepo := loggerrepo.NewLoggerRepository(base.GormDB)
	return &Controller{
		BaseController: base,
		repo:           loggerRepo,
	}
}

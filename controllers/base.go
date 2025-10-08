package controllers

import (
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)

// BaseController menyediakan data umum yang diperlukan oleh semua controller
type BaseController struct {
	GormDB   *gorm.DB
	Validate *validator.Validate
}

// NewBaseControllerWithGorm creates a base controller with both SQL and GORM DB connections
func NewBaseControllerWithGorm(gormDB *gorm.DB, validate *validator.Validate) *BaseController {
	return &BaseController{
		GormDB:   gormDB,
		Validate: validate,
	}
}

package controllers

import (
	"database/sql"

	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)

// BaseController menyediakan data umum yang diperlukan oleh semua controller
type BaseController struct {
	DB       *sql.DB
	GormDB   *gorm.DB
	Validate *validator.Validate
}


// NewBaseControllerWithGorm creates a base controller with both SQL and GORM DB connections
func NewBaseControllerWithGorm(db *sql.DB, gormDB *gorm.DB, validate *validator.Validate) *BaseController {
	return &BaseController{
		DB:       db,
		GormDB:   gormDB,
		Validate: validate,
	}
}

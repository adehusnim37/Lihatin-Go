package controllers

import (
	"database/sql"

	"github.com/go-playground/validator/v10"
)

// BaseController menyediakan data umum yang diperlukan oleh semua controller
type BaseController struct {
	DB       *sql.DB
	Validate *validator.Validate
}

// NewBaseController membuat instance baru base controller
func NewBaseController(db *sql.DB, validate *validator.Validate) *BaseController {
	return &BaseController{
		DB:       db,
		Validate: validate,
	}
}

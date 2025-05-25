package models

import "time"

type User struct {
	ID        string     `json:"id"`
	Username  string     `json:"username" validate:"required,min=3,max=50"`
	FirstName string     `json:"first_name" validate:"required,min=3,max=50"`
	LastName  string     `json:"last_name" validate:"required,min=3,max=50"`
	Email     string     `json:"email" validate:"required,email"`
	Password  string     `json:"password" validate:"required,min=8,max=50,pwdcomplex"`
	CreatedAt string     `json:"created_at" default:"null"`
	UpdatedAt string     `json:"updated_at" default:"null"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" default:"null"`
	IsPremium bool       `json:"is_premium" default:"false"`
	Avatar    string     `json:"avatar" validate:"required,url"`
}

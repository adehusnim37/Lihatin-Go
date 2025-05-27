package models

type UpdateUser struct {
	FirstName string     `json:"first_name" validate:"required,min=3,max=50"`
	LastName  string     `json:"last_name" validate:"required,min=3,max=50"`
	Email     string     `json:"email" validate:"required,email"`
	Password  string     `json:"password" validate:"required,min=8,max=50,pwdcomplex"`
	Avatar    string     `json:"avatar" validate:"omitempty,url"`
}

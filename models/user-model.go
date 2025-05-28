package models

import "time"

type User struct {
	ID           string     `json:"id"`
	Username     string     `json:"username" validate:"required,min=3,max=50"`
	FirstName    string     `json:"first_name" validate:"required,min=3,max=50"`
	LastName     string     `json:"last_name" validate:"required,min=3,max=50"`
	Email        string     `json:"email" validate:"required,email"`
	Password     string     `json:"password" validate:"required,min=8,max=50,pwdcomplex"`
	CreatedAt    string     `json:"created_at" default:"null"`
	UpdatedAt    string     `json:"updated_at" default:"null"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty" default:"null"`
	IsPremium    bool       `json:"is_premium" default:"false"`
	Avatar       string     `json:"avatar" validate:"required,url"`
	IsLocked     bool       `json:"is_locked" default:"false"`
	LockedAt     *time.Time `json:"locked_at,omitempty" default:"null"`
	LockedReason string     `json:"locked_reason,omitempty"`
	Role         string     `json:"role" default:"user"` // user, admin, super_admin
}

// TableName specifies the table name for GORM
func (User) TableName() string {
	return "users"
}

// AdminLockUserRequest represents the request to lock a user account
type AdminLockUserRequest struct {
	Reason string `json:"reason" validate:"required,min=10,max=500"`
}

// AdminUnlockUserRequest represents the request to unlock a user account
type AdminUnlockUserRequest struct {
	Reason string `json:"reason,omitempty" validate:"max=500"`
}

// AdminUserResponse represents user data for admin views
type AdminUserResponse struct {
	ID           string     `json:"id"`
	Username     string     `json:"username"`
	FirstName    string     `json:"first_name"`
	LastName     string     `json:"last_name"`
	Email        string     `json:"email"`
	CreatedAt    string     `json:"created_at"`
	UpdatedAt    string     `json:"updated_at"`
	IsPremium    bool       `json:"is_premium"`
	IsLocked     bool       `json:"is_locked"`
	LockedAt     *time.Time `json:"locked_at,omitempty"`
	LockedReason string     `json:"locked_reason,omitempty"`
	Role         string     `json:"role"`
}

// PaginatedUsersResponse represents paginated user results
type PaginatedUsersResponse struct {
	Users      []AdminUserResponse `json:"users"`
	TotalCount int64               `json:"total_count"`
	Page       int                 `json:"page"`
	Limit      int                 `json:"limit"`
	TotalPages int                 `json:"total_pages"`
}

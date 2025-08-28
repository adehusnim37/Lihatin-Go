package user

import (
	"time"
)

type User struct {
	ID           string     `json:"id" gorm:"primaryKey"`
	Username     string     `json:"username" gorm:"uniqueIndex;size:50;not null" validate:"required,min=3,max=50"`
	FirstName    string     `json:"first_name" gorm:"size:50" validate:"required,min=3,max=50"`
	LastName     string     `json:"last_name" gorm:"size:50" validate:"required,min=3,max=50"`
	Email        string     `json:"email" gorm:"uniqueIndex;size:100;not null" validate:"required,email"`
	Password     string     `json:"password" gorm:"size:255" validate:"required,min=8,max=50,pwdcomplex"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty" gorm:"index"`
	IsPremium    bool       `json:"is_premium" gorm:"default:false"`
	Avatar       string     `json:"avatar" gorm:"size:255"`
	IsLocked     bool       `json:"is_locked" gorm:"default:false"`
	LockedAt     *time.Time `json:"locked_at,omitempty"`
	LockedReason string     `json:"locked_reason,omitempty" gorm:"size:500"`
	Role         string     `json:"role" gorm:"size:20;default:user"` // user, admin, super_admin

	// Relationships
	UserAuth []UserAuth `json:"user_auth,omitempty" gorm:"foreignKey:UserID"`
	APIKeys  []APIKey   `json:"api_keys,omitempty" gorm:"foreignKey:UserID"`
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

// AdminUserResponse represents the response format for admin user data
type AdminUserResponse struct {
    ID           string    `json:"id"`
    Username     string    `json:"username"`
    FirstName    string    `json:"first_name"`
    LastName     string    `json:"last_name"`
    Email        string    `json:"email"`
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
    IsPremium    bool      `json:"is_premium"`
    IsLocked     bool      `json:"is_locked"`
    LockedAt     *time.Time `json:"locked_at,omitempty"`
    LockedReason string    `json:"locked_reason,omitempty"`
    Role         string    `json:"role"`
}

// PaginatedUsersResponse represents paginated user results
type PaginatedUsersResponse struct {
	Users      []AdminUserResponse `json:"users"`
	TotalCount int64               `json:"total_count"`
	Page       int                 `json:"page"`
	Limit      int                 `json:"limit"`
	TotalPages int                 `json:"total_pages"`
}

// UserProfileResponse represents user profile for regular users
type UserProfileResponse struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Email     string    `json:"email"`
	Avatar    string    `json:"avatar"`
	IsPremium bool      `json:"is_premium"`
	CreatedAt time.Time `json:"created_at"`
}

type UpdateUser struct {
	FirstName string     `json:"first_name" validate:"required,min=3,max=50"`
	LastName  string     `json:"last_name" validate:"required,min=3,max=50"`
	Email     string     `json:"email" validate:"required,email"`
	Password  string     `json:"password" validate:"required,min=8,max=50,pwdcomplex"`
	Avatar    string     `json:"avatar" validate:"omitempty,url"`
}

package user

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type User struct {
	ID              string     `json:"id" gorm:"primaryKey"`
	Username        string     `json:"username" gorm:"uniqueIndex;size:50;not null" validate:"required,min=3,max=50"`
	FirstName       string     `json:"first_name" gorm:"size:50" validate:"required,min=3,max=50"`
	LastName        string     `json:"last_name" gorm:"size:50" validate:"required,min=3,max=50"`
	Email           string     `json:"email" gorm:"uniqueIndex;size:100;not null" validate:"required,email"`
	Password        string     `json:"password" gorm:"size:255" validate:"required,min=8,max=50,pwdcomplex"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	DeletedAt       *time.Time `json:"deleted_at,omitempty" gorm:"index"`
	UsernameChanged bool       `json:"username_changed,omitempty" gorm:"default:false"`
	IsPremium       bool       `json:"is_premium" gorm:"default:false"`
	Avatar          string     `json:"avatar" gorm:"size:255"`
	IsLocked        bool       `json:"is_locked" gorm:"default:false"`
	LockedAt        *time.Time `json:"locked_at,omitempty"`
	LockedReason    string     `json:"locked_reason,omitempty" gorm:"size:500"`
	Role            string     `json:"role" gorm:"size:20;default:user"` // user, admin, super_admin

	// Relationships
	UserAuth        []UserAuth        `json:"user_auth,omitempty" gorm:"foreignKey:UserID"`
	APIKeys         []APIKey          `json:"api_keys,omitempty" gorm:"foreignKey:UserID"`
	HistoryUsers    []HistoryUser     `json:"history_users,omitempty" gorm:"foreignKey:UserID"`
	PremiumKeyUsage []PremiumKeyUsage `json:"premium_key_usage,omitempty" gorm:"foreignKey:UserID"`
}

// TableName specifies the table name for GORM
func (User) TableName() string {
	return "users"
}

type HistoryUser struct {
	ID            uint              `gorm:"primaryKey"`
	UserID        string            `gorm:"index:idx_user_action,priority:1;not null"`
	ActionType    ActionHistoryUser `gorm:"index:idx_user_action,priority:2;type:varchar(50);not null"` // e.g. email_change, password_change
	OldValue      datatypes.JSON    `gorm:"type:jsonb"`                                                 // JSON snapshot before change
	NewValue      datatypes.JSON    `gorm:"type:jsonb"`                                                 // JSON snapshot after change
	Reason        string            `gorm:"type:varchar(255)"`                                          // Reason for change (user/admin/system)
	ChangedBy     *string           `gorm:"type:varchar(50)"`                                           // Who made the change (userID/adminID/system)
	RevokeToken   string            `gorm:"type:varchar(100);index"`                                    // Token to revoke sessions if needed
	RevokeExpires *time.Time        `gorm:""`                                                           // Expiry for the revoke token
	IPAddress     *string           `gorm:"type:varchar(45)"`                                           // IPv4/IPv6
	UserAgent     *string           `gorm:"type:text"`                                                  // User agent string
	ChangedAt     time.Time         `gorm:"autoCreateTime;index"`                                       // Timestamp of change
	DeletedAt     gorm.DeletedAt    `gorm:"index"`
}

type ActionHistoryUser string

// ActionType constants for audit trail
const (
	ActionEmailChange    ActionHistoryUser = "email_change"
	ActionPasswordChange ActionHistoryUser = "password_change"
	ActionPhoneChange    ActionHistoryUser = "phone_change"
	ActionUsernameChange ActionHistoryUser = "username_change"
	ActionProfileUpdate  ActionHistoryUser = "profile_update"
	ActionAccountLock    ActionHistoryUser = "account_lock"
	ActionAccountUnlock  ActionHistoryUser = "account_unlock"
)

package user

import (
	"time"

	"gorm.io/gorm"
)

type PremiumKey struct {
	ID         int               `json:"id" gorm:"primaryKey"`
	Code       string            `json:"code" gorm:"uniqueIndex;size:255;not null"`
	ValidUntil *time.Time        `json:"valid_until,omitempty" gorm:"index"`
	LimitUsage *int64            `json:"limit_usage,omitempty" gorm:"default:null"`
	UsageCount int64             `json:"usage_count" gorm:"default:0"`
	CreatedAt  time.Time         `json:"created_at" gorm:"autoCreateTime;index"`
	UpdatedAt  time.Time         `json:"updated_at" gorm:"autoUpdateTime;index"`
	DeletedAt  gorm.DeletedAt    `json:"deleted_at,omitempty" gorm:"index"`
	Usage      []PremiumKeyUsage `json:"usage,omitempty" gorm:"foreignKey:PremiumKeyID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type PremiumKeyUsage struct {
	ID           int       `json:"id" gorm:"primaryKey"`
	PremiumKeyID int       `json:"premium_key_id" gorm:"not null;index;uniqueIndex:ux_key_user"`
	UserID       string    `json:"user_id" gorm:"not null;index;uniqueIndex:ux_key_user"`
	CreatedAt    time.Time `json:"created_at" gorm:"autoCreateTime;index"`
	UpdatedAt    time.Time `json:"updated_at" gorm:"autoUpdateTime;index"`
}

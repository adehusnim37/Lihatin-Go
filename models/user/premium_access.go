package user

import "time"

type PremiumAccessStatus string

const (
	PremiumAccessStatusActive  PremiumAccessStatus = "active"
	PremiumAccessStatusRevoked PremiumAccessStatus = "revoked"
)

type PremiumAccessRevokeType string

const (
	PremiumAccessRevokeTypeTemporary PremiumAccessRevokeType = "temporary"
	PremiumAccessRevokeTypePermanent PremiumAccessRevokeType = "permanent"
)

// PremiumAccess is the current premium entitlement for one user. Absence of a
// row means the user has never received premium access and is on the free tier.
// PremiumAccessEvent remains the append-only audit trail.
type PremiumAccess struct {
	UserID          string            `json:"user_id" gorm:"primaryKey;size:191"`
	User            *User             `json:"-" gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
	Status          PremiumAccessStatus     `json:"status" gorm:"size:20;not null;index"`
	Tier            string            `json:"tier" gorm:"size:30;not null;default:premium"`
	Source          string            `json:"source" gorm:"size:50;not null"`
	GrantedAt       time.Time         `json:"granted_at" gorm:"not null"`
	ExpiresAt       *time.Time        `json:"expires_at,omitempty" gorm:"index"`
	RevokeType      PremiumAccessRevokeType `json:"revoke_type,omitempty" gorm:"size:20"`
	StatusChangedAt *time.Time        `json:"status_changed_at,omitempty" gorm:"index"`
	StatusChangedBy *string           `json:"status_changed_by,omitempty" gorm:"size:191;index"`
	StatusReason    string            `json:"status_reason,omitempty" gorm:"size:500"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

func (PremiumAccess) TableName() string {
	return "user_premium_access"
}

func (p *PremiumAccess) IsActiveAt(now time.Time) bool {
	if p == nil || p.Status != PremiumAccessStatusActive {
		return false
	}
	return p.ExpiresAt == nil || now.Before(*p.ExpiresAt)
}

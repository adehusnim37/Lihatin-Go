package user

import (
	"time"

	"gorm.io/datatypes"
)

type PremiumStatus string

const (
	PremiumStatusActive  PremiumStatus = "active"
	PremiumStatusRevoked PremiumStatus = "revoked"
)

type PremiumRevokeType string

const (
	PremiumRevokeTypeTemporary PremiumRevokeType = "temporary"
	PremiumRevokeTypePermanent PremiumRevokeType = "permanent"
)

type PremiumStatusEventAction string

const (
	PremiumStatusEventActionRevoke     PremiumStatusEventAction = "revoke"
	PremiumStatusEventActionReactivate PremiumStatusEventAction = "reactivate"
)

type PremiumStatusEvent struct {
	ID          uint                     `json:"id" gorm:"primaryKey"`
	UserID      string                   `json:"user_id" gorm:"index:idx_premium_status_events_user_created,priority:1;size:50;not null"`
	Action      PremiumStatusEventAction `json:"action" gorm:"type:varchar(20);not null"`
	OldStatus   string                   `json:"old_status" gorm:"type:varchar(20);not null"`
	NewStatus   string                   `json:"new_status" gorm:"type:varchar(20);not null"`
	OldRole     string                   `json:"old_role" gorm:"type:varchar(20);not null"`
	NewRole     string                   `json:"new_role" gorm:"type:varchar(20);not null"`
	RevokeType  string                   `json:"revoke_type,omitempty" gorm:"type:varchar(20)"`
	Reason      string                   `json:"reason" gorm:"type:varchar(500);not null"`
	ChangedBy   *string                  `json:"changed_by,omitempty" gorm:"type:varchar(50);index"`
	ChangedRole string                   `json:"changed_role" gorm:"type:varchar(20);not null"`
	Metadata    datatypes.JSON           `json:"metadata,omitempty" gorm:"type:json"`
	CreatedAt   time.Time                `json:"created_at" gorm:"index:idx_premium_status_events_user_created,priority:2"`
	UpdatedAt   time.Time                `json:"updated_at"`
}

func (PremiumStatusEvent) TableName() string {
	return "premium_status_events"
}

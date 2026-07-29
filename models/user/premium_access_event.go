package user

import (
	"time"

	"gorm.io/datatypes"
)

type PremiumAccessEventAction string

const (
	PremiumAccessEventActionRevoke     PremiumAccessEventAction = "revoke"
	PremiumAccessEventActionReactivate PremiumAccessEventAction = "reactivate"
)

type PremiumAccessEvent struct {
	ID         uint                     `json:"id" gorm:"primaryKey"`
	UserID     string                   `json:"user_id" gorm:"index:idx_premium_access_events_user_created,priority:1;size:50;not null"`
	Action     PremiumAccessEventAction `json:"action" gorm:"type:varchar(20);not null"`
	OldStatus  string                   `json:"old_status" gorm:"type:varchar(20);not null"`
	NewStatus  string                   `json:"new_status" gorm:"type:varchar(20);not null"`
	RevokeType string                   `json:"revoke_type,omitempty" gorm:"type:varchar(20)"`
	Reason     string                   `json:"reason" gorm:"type:varchar(500);not null"`
	ActorID    *string                  `json:"actor_id,omitempty" gorm:"size:191;index"`
	ActorRole  string                   `json:"actor_role" gorm:"type:varchar(20);not null;default:system"`
	Metadata   datatypes.JSON           `json:"metadata,omitempty" gorm:"type:json"`
	CreatedAt  time.Time                `json:"created_at" gorm:"index:idx_premium_access_events_user_created,priority:2"`
	UpdatedAt  time.Time                `json:"updated_at"`
}

func (PremiumAccessEvent) TableName() string {
	return "premium_access_events"
}

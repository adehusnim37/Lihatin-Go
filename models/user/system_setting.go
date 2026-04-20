package user

import "time"

// SystemSetting stores global key/value settings managed by admins.
type SystemSetting struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Key       string    `json:"key" gorm:"uniqueIndex;size:120;not null"`
	Value     string    `json:"value" gorm:"size:64;not null"`
	UpdatedBy *string   `json:"updated_by,omitempty" gorm:"size:50"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName specifies the table name for GORM.
func (SystemSetting) TableName() string {
	return "system_settings"
}

package shortlink

import "time"

const (
	ShortCodeAlphabet  = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	MinShortCodeLength = 2
	MaxShortCodeLength = 8
)

// ShortCodeAllocator stores the next ordinal to reserve for each code length.
// Rows are locked with SELECT ... FOR UPDATE while codes are allocated, so two
// application instances cannot reserve the same ordinal.
type ShortCodeAllocator struct {
	CodeLength uint8     `json:"code_length" gorm:"primaryKey;autoIncrement:false"`
	NextValue  uint64    `json:"next_value" gorm:"type:bigint unsigned;not null;default:0"`
	Capacity   uint64    `json:"capacity" gorm:"type:bigint unsigned;not null"`
	CreatedAt  time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt  time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (ShortCodeAllocator) TableName() string {
	return "short_code_allocators"
}

func ShortCodeCapacity(length int) uint64 {
	capacity := uint64(1)
	for range length {
		capacity *= uint64(len(ShortCodeAlphabet))
	}
	return capacity
}

package history

import "gorm.io/gorm"

type HistoryUserRepository struct {
	db *gorm.DB
}

// NewHistoryUserRepository creates a new HistoryUser repository
func NewHistoryUserRepository(db *gorm.DB) *HistoryUserRepository {
	return &HistoryUserRepository{db: db}
}


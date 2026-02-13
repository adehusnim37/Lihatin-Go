package migrations

import (
	"fmt"
	"log"

	"github.com/adehusnim37/lihatin-go/models/logging"
	"github.com/adehusnim37/lihatin-go/models/shortlink"
	"github.com/adehusnim37/lihatin-go/models/user"
	"gorm.io/gorm"
)

// runMigrations handles all model migrations

func RunMigrations(db *gorm.DB) error {
	// Migrate User models
	if err := db.AutoMigrate(&user.User{}); err != nil {
		return fmt.Errorf("failed to migrate User model: %w", err)
	}

	if err := db.AutoMigrate(&user.UserAuth{}); err != nil {
		return fmt.Errorf("failed to migrate UserAuth model: %w", err)
	}

	if err := db.AutoMigrate(&user.HistoryUser{}); err != nil {
		return fmt.Errorf("failed to migrate HistoryUser model: %w", err)
	}

	if err := db.AutoMigrate(&user.LoginAttempt{}); err != nil {
		return fmt.Errorf("failed to migrate LoginAttempt model: %w", err)
	}

	if err := db.AutoMigrate(&user.AuthMethod{}); err != nil {
		return fmt.Errorf("failed to migrate AuthMethod model: %w", err)
	}

	if err := db.AutoMigrate(&user.APIKey{}); err != nil {
		return fmt.Errorf("failed to migrate APIKey model: %w", err)
	}

	// Migrate ShortLink models
	if err := db.AutoMigrate(&shortlink.ShortLink{}); err != nil {
		return fmt.Errorf("failed to migrate ShortLink model: %w", err)
	}

	if err := db.AutoMigrate(&shortlink.ShortLinkDetail{}); err != nil {
		return fmt.Errorf("failed to migrate ShortLinkDetail model: %w", err)
	}

	if err := db.AutoMigrate(&shortlink.ViewLinkDetail{}); err != nil {
		return fmt.Errorf("failed to migrate ViewLinkDetail model: %w", err)
	}

	// Migrate Logging models
	if err := db.AutoMigrate(&logging.ActivityLog{}); err != nil {
		return fmt.Errorf("failed to migrate ActivityLog model: %w", err)
	}

	// Migrate Premium models
	if err := db.AutoMigrate(&user.PremiumKey{}); err != nil {
		return fmt.Errorf("failed to migrate PremiumKey model: %w", err)
	}

	if err := db.AutoMigrate(&user.PremiumKeyUsage{}); err != nil {
		return fmt.Errorf("failed to migrate PremiumKeyUsage model: %w", err)
	}

	log.Println("✅ All models migrated successfully!")
	return nil
}

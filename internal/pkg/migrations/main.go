package migrations

import (
	"fmt"
	"log"

	"github.com/adehusnim37/lihatin-go/models/logging"
	"github.com/adehusnim37/lihatin-go/models/shortlink"
	supportmodel "github.com/adehusnim37/lihatin-go/models/support"
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

	if err := db.AutoMigrate(&user.SystemSetting{}); err != nil {
		return fmt.Errorf("failed to migrate SystemSetting model: %w", err)
	}

	if err := db.AutoMigrate(&user.NotificationPreference{}); err != nil {
		return fmt.Errorf("failed to migrate NotificationPreference model: %w", err)
	}

	if err := db.AutoMigrate(&user.WeeklySummaryDelivery{}); err != nil {
		return fmt.Errorf("failed to migrate WeeklySummaryDelivery model: %w", err)
	}

	if err := db.AutoMigrate(&user.PromotionalCampaign{}); err != nil {
		return fmt.Errorf("failed to migrate PromotionalCampaign model: %w", err)
	}

	if err := db.AutoMigrate(&user.PromotionalEmailDelivery{}); err != nil {
		return fmt.Errorf("failed to migrate PromotionalEmailDelivery model: %w", err)
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

	if err := renamePremiumAccessEventTable(db); err != nil {
		return err
	}
	if err := db.AutoMigrate(&user.PremiumAccessEvent{}); err != nil {
		return fmt.Errorf("failed to migrate PremiumAccessEvent model: %w", err)
	}
	if err := db.AutoMigrate(&user.PremiumAccess{}); err != nil {
		return fmt.Errorf("failed to migrate PremiumAccess model: %w", err)
	}
	if err := normalizeUserAccountSchema(db); err != nil {
		return err
	}

	// Migrate Support models
	if err := db.AutoMigrate(&supportmodel.SupportTicket{}); err != nil {
		return fmt.Errorf("failed to migrate SupportTicket model: %w", err)
	}
	if err := db.AutoMigrate(&supportmodel.SupportMessage{}); err != nil {
		return fmt.Errorf("failed to migrate SupportMessage model: %w", err)
	}
	if err := db.AutoMigrate(&supportmodel.SupportAttachment{}); err != nil {
		return fmt.Errorf("failed to migrate SupportAttachment model: %w", err)
	}
	if err := normalizeSupportAttachmentSchema(db); err != nil {
		return err
	}

	log.Println("✅ All models migrated successfully!")
	return nil
}

func normalizeSupportAttachmentSchema(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("gorm DB is required")
	}

	if !db.Migrator().HasTable(&supportmodel.SupportAttachment{}) {
		return nil
	}

	if db.Migrator().HasColumn(&supportmodel.SupportAttachment{}, "blob_data") {
		log.Println("ℹ️ Dropping legacy support_attachments.blob_data column")
		if err := db.Migrator().DropColumn(&supportmodel.SupportAttachment{}, "blob_data"); err != nil {
			return fmt.Errorf("failed to drop legacy support_attachments.blob_data column: %w", err)
		}
	}

	return nil
}

func renamePremiumAccessEventTable(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("gorm DB is required")
	}
	migrator := db.Migrator()
	hasLegacyTable := migrator.HasTable("premium_status_events")
	hasCurrentTable := migrator.HasTable("premium_access_events")
	if hasLegacyTable && hasCurrentTable {
		return fmt.Errorf("both premium_status_events and premium_access_events exist; reconcile them before migration")
	}
	if !hasLegacyTable {
		return nil
	}
	if err := migrator.RenameTable("premium_status_events", "premium_access_events"); err != nil {
		return fmt.Errorf("rename premium status event table: %w", err)
	}
	return nil
}

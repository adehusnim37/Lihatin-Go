package migrations

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/adehusnim37/lihatin-go/models/user"
	"gorm.io/gorm"
)

type legacyUserAccountRow struct {
	UserID                       string
	UserCreatedAt                time.Time
	LegacyPassword               string
	LegacyIsLocked               bool
	LegacyLockedAt               *time.Time
	LegacyLockedReason           string
	LegacyIsPremium              bool
	LegacyPremiumStatus          string
	LegacyPremiumRevokeType      string
	LegacyPremiumRevokedAt       *time.Time
	LegacyPremiumRevokedBy       *string
	LegacyPremiumRevokedReason   string
	LegacyPremiumReactivatedAt   *time.Time
	LegacyPremiumReactivatedBy   *string
	LegacyPremiumReactivatedNote string
	AuthID                       string
	AuthPasswordHash             string
	LegacyAuthIsActive           bool
	LegacyDeviceID               *string
	LegacyLastIP                 *string
	LegacyLockoutUntil           *time.Time
}

// normalizeUserAccountSchema performs the one-time data move from the legacy
// users/user_auth layout into normalized auth and premium tables. It is
// intentionally rerunnable.
func normalizeUserAccountSchema(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("gorm DB is required")
	}

	migrator := db.Migrator()
	hasLegacyUsers := migrator.HasColumn("users", "password")
	hasLegacyAuth := migrator.HasColumn("user_auth", "is_active")
	hasLegacyPremiumAccess := migrator.HasColumn("user_premium_access", "last_action")
	hasLegacyPremiumEvents := migrator.HasColumn("premium_access_events", "old_role") ||
		migrator.HasColumn("premium_access_events", "new_role") ||
		migrator.HasColumn("premium_access_events", "changed_by") ||
		migrator.HasColumn("premium_access_events", "changed_role")
	if !hasLegacyUsers && !hasLegacyAuth && !hasLegacyPremiumAccess && !hasLegacyPremiumEvents {
		return nil
	}

	if hasLegacyPremiumAccess {
		if err := migrator.DropColumn("user_premium_access", "last_action"); err != nil {
			return fmt.Errorf("drop redundant user_premium_access.last_action: %w", err)
		}
	}

	if hasLegacyUsers && hasLegacyAuth {
		var missingAuthRecords int64
		if err := db.Table("users").
			Joins("LEFT JOIN user_auth ON user_auth.user_id = users.id").
			Where("users.deleted_at IS NULL AND user_auth.id IS NULL").
			Count(&missingAuthRecords).Error; err != nil {
			return fmt.Errorf("check users without auth records: %w", err)
		}
		if missingAuthRecords > 0 {
			return fmt.Errorf("refusing to drop legacy password data: %d active user(s) have no user_auth record", missingAuthRecords)
		}

		var rows []legacyUserAccountRow
		if err := db.Table("users").
			Select(`
			users.id AS user_id,
			users.created_at AS user_created_at,
			users.password AS legacy_password,
			users.is_locked AS legacy_is_locked,
			users.locked_at AS legacy_locked_at,
			users.locked_reason AS legacy_locked_reason,
			users.is_premium AS legacy_is_premium,
			users.premium_status AS legacy_premium_status,
			users.premium_revoke_type AS legacy_premium_revoke_type,
			users.premium_revoked_at AS legacy_premium_revoked_at,
			users.premium_revoked_by AS legacy_premium_revoked_by,
			users.premium_revoked_reason AS legacy_premium_revoked_reason,
			users.premium_reactivated_at AS legacy_premium_reactivated_at,
			users.premium_reactivated_by AS legacy_premium_reactivated_by,
			users.premium_reactivated_reason AS legacy_premium_reactivated_note,
			user_auth.id AS auth_id,
			user_auth.password_hash AS auth_password_hash,
			user_auth.is_active AS legacy_auth_is_active,
			user_auth.device_id AS legacy_device_id,
			user_auth.last_ip AS legacy_last_ip,
			user_auth.lockout_until AS legacy_lockout_until
		`).
			Joins("JOIN user_auth ON user_auth.user_id = users.id").
			Scan(&rows).Error; err != nil {
			return fmt.Errorf("read legacy user account data: %w", err)
		}

		if err := db.Transaction(func(tx *gorm.DB) error {
			for i := range rows {
				row := rows[i]
				passwordHash := row.AuthPasswordHash
				if passwordHash == "" {
					passwordHash = row.LegacyPassword
				}

				status := user.AccountStatusActive
				statusChangedAt := (*time.Time)(nil)
				statusReason := ""
				if !row.LegacyAuthIsActive {
					status = user.AccountStatusDisabled
				}
				if row.LegacyIsLocked {
					status = user.AccountStatusLocked
					statusChangedAt = row.LegacyLockedAt
					statusReason = row.LegacyLockedReason
				}

				if err := tx.Model(&user.UserAuth{}).
					Where("id = ?", row.AuthID).
					Updates(map[string]any{
						"password_hash":       passwordHash,
						"account_status":      status,
						"status_changed_at":   statusChangedAt,
						"status_reason":       statusReason,
						"last_device_id":      row.LegacyDeviceID,
						"last_login_ip":       row.LegacyLastIP,
						"login_blocked_until": row.LegacyLockoutUntil,
					}).Error; err != nil {
					return err
				}

				shouldCreatePremium := row.LegacyIsPremium ||
					row.LegacyPremiumStatus == string(user.PremiumAccessStatusRevoked) ||
					row.LegacyPremiumRevokedAt != nil
				if !shouldCreatePremium {
					continue
				}

				accessStatus := user.PremiumAccessStatusActive
				changedAt := row.UserCreatedAt
				changedBy := (*string)(nil)
				reason := ""
				revokeType := user.PremiumAccessRevokeType("")
				if row.LegacyPremiumStatus == string(user.PremiumAccessStatusRevoked) {
					accessStatus = user.PremiumAccessStatusRevoked
					revokeType = user.PremiumAccessRevokeType(row.LegacyPremiumRevokeType)
					if row.LegacyPremiumRevokedAt != nil {
						changedAt = *row.LegacyPremiumRevokedAt
					}
					changedBy = row.LegacyPremiumRevokedBy
					reason = row.LegacyPremiumRevokedReason
				} else if row.LegacyPremiumReactivatedAt != nil {
					changedAt = *row.LegacyPremiumReactivatedAt
					changedBy = row.LegacyPremiumReactivatedBy
					reason = row.LegacyPremiumReactivatedNote
				}

				access := user.PremiumAccess{
					UserID:          row.UserID,
					Status:          accessStatus,
					Tier:            "premium",
					Source:          "legacy_migration",
					GrantedAt:       row.UserCreatedAt,
					RevokeType:      revokeType,
					StatusChangedAt: &changedAt,
					StatusChangedBy: changedBy,
					StatusReason:    reason,
				}
				if err := tx.Where("user_id = ?", row.UserID).FirstOrCreate(&access).Error; err != nil &&
					!errors.Is(err, gorm.ErrDuplicatedKey) {
					return err
				}
			}
			return nil
		}); err != nil {
			return fmt.Errorf("backfill normalized user account data: %w", err)
		}

		var missingPasswordHashes int64
		if err := db.Model(&user.UserAuth{}).
			Where("password_hash IS NULL OR password_hash = ''").
			Count(&missingPasswordHashes).Error; err != nil {
			return fmt.Errorf("check missing password hashes: %w", err)
		}
		if missingPasswordHashes > 0 {
			return fmt.Errorf("refusing to drop legacy password data: %d user_auth record(s) have no password hash", missingPasswordHashes)
		}
	}

	legacyUserColumns := []string{
		"password",
		"is_premium",
		"is_locked",
		"locked_at",
		"locked_reason",
		"premium_status",
		"premium_revoke_type",
		"premium_revoked_at",
		"premium_revoked_by",
		"premium_revoked_reason",
		"premium_reactivated_at",
		"premium_reactivated_by",
		"premium_reactivated_reason",
	}
	for _, column := range legacyUserColumns {
		if migrator.HasColumn("users", column) {
			if err := migrator.DropColumn("users", column); err != nil {
				return fmt.Errorf("drop legacy users.%s: %w", column, err)
			}
		}
	}

	legacyAuthColumns := []string{"is_active", "is_totp_enabled", "lockout_until", "device_id", "last_ip"}
	for _, column := range legacyAuthColumns {
		if migrator.HasColumn("user_auth", column) {
			if err := migrator.DropColumn("user_auth", column); err != nil {
				return fmt.Errorf("drop legacy user_auth.%s: %w", column, err)
			}
		}
	}

	if hasLegacyPremiumEvents {
		if migrator.HasColumn("premium_access_events", "changed_by") {
			if err := db.Exec(`
				UPDATE premium_access_events
				SET actor_id = changed_by
				WHERE actor_id IS NULL AND changed_by IS NOT NULL
			`).Error; err != nil {
				return fmt.Errorf("backfill premium event actor_id: %w", err)
			}
		}
		if migrator.HasColumn("premium_access_events", "changed_role") {
			if err := db.Exec(`
				UPDATE premium_access_events
				SET actor_role = changed_role
				WHERE (actor_role IS NULL OR actor_role = '' OR actor_role = 'system')
					AND changed_role IS NOT NULL
			`).Error; err != nil {
				return fmt.Errorf("backfill premium event actor_role: %w", err)
			}
		}

		for _, column := range []string{"old_role", "new_role", "changed_by", "changed_role"} {
			if migrator.HasColumn("premium_access_events", column) {
				if err := migrator.DropColumn("premium_access_events", column); err != nil {
					return fmt.Errorf("drop legacy premium_access_events.%s: %w", column, err)
				}
			}
		}
	}

	log.Printf("✅ Normalized users, user_auth, and user_premium_access schema")
	return nil
}

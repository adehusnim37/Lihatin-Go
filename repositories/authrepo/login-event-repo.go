package authrepo

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/adehusnim37/lihatin-go/models/user"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SuccessfulLogin contains the data captured only after authentication completes.
type SuccessfulLogin struct {
	UserID    string
	SessionID string
	Method    user.LoginMethod
	DeviceID  string
	IPAddress string
	UserAgent string
}

// LoginEventRepository stores immutable completed-login events.
type LoginEventRepository struct {
	db *gorm.DB
}

func NewLoginEventRepository(db *gorm.DB) *LoginEventRepository {
	return &LoginEventRepository{db: db}
}

// RecordSuccessfulLogin atomically updates the account-level latest-login
// snapshot and inserts the immutable event. It returns the preceding event.
func (r *LoginEventRepository) RecordSuccessfulLogin(input SuccessfulLogin) (*user.LoginEvent, *user.LoginEvent, error) {
	if input.UserID == "" || input.SessionID == "" {
		return nil, nil, errors.New("user ID and session ID are required")
	}

	now := time.Now()
	event := &user.LoginEvent{
		ID:              uuid.New().String(),
		UserID:          input.UserID,
		SessionIDHash:   hashSessionID(input.SessionID),
		Method:          input.Method,
		DeviceID:        input.DeviceID,
		IPAddress:       input.IPAddress,
		UserAgent:       input.UserAgent,
		AuthenticatedAt: now,
	}

	var previous *user.LoginEvent
	err := r.db.Transaction(func(tx *gorm.DB) error {
		// Serialize completed logins per user so concurrent devices get a
		// deterministic previous-login chain and cannot regress the snapshot.
		var lockedAuth user.UserAuth
		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Select("id").
			Where("user_id = ?", input.UserID).
			First(&lockedAuth).Error; err != nil {
			return fmt.Errorf("lock user authentication record: %w", err)
		}

		var found user.LoginEvent
		findErr := tx.
			Where("user_id = ?", input.UserID).
			Order("authenticated_at DESC").
			Order("created_at DESC").
			First(&found).Error
		switch {
		case findErr == nil:
			previous = &found
		case errors.Is(findErr, gorm.ErrRecordNotFound):
			previous = nil
		default:
			return fmt.Errorf("find previous login event: %w", findErr)
		}

		result := tx.Model(&user.UserAuth{}).
			Where("user_id = ?", input.UserID).
			Updates(map[string]any{
				"last_login_at":         now,
				"failed_login_attempts": 0,
				"login_blocked_until":   nil,
				"last_device_id":        input.DeviceID,
				"last_login_ip":         input.IPAddress,
				"last_email_send_at":    nil,
			})
		if result.Error != nil {
			return fmt.Errorf("update latest login snapshot: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return errors.New("user authentication record not found")
		}

		if err := tx.Create(event).Error; err != nil {
			return fmt.Errorf("create login event: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	return event, previous, nil
}

// GetPreviousLoginForSession returns the login immediately before the supplied
// session. This keeps "previous login" stable even if another device logs in later.
func (r *LoginEventRepository) GetPreviousLoginForSession(userID, sessionID string, sessionCreatedAt time.Time) (*user.LoginEvent, error) {
	if userID == "" || sessionID == "" {
		return nil, nil
	}

	cutoff := sessionCreatedAt
	var current user.LoginEvent
	err := r.db.
		Where("user_id = ? AND session_id_hash = ?", userID, hashSessionID(sessionID)).
		First(&current).Error
	switch {
	case err == nil:
		cutoff = current.AuthenticatedAt
	case errors.Is(err, gorm.ErrRecordNotFound):
		// Sessions created before login_events was introduced have no matching row.
	default:
		return nil, fmt.Errorf("find current login event: %w", err)
	}

	var previous user.LoginEvent
	err = r.db.
		Where("user_id = ? AND authenticated_at < ?", userID, cutoff).
		Order("authenticated_at DESC").
		Order("created_at DESC").
		First(&previous).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find previous login event: %w", err)
	}
	return &previous, nil
}

func hashSessionID(sessionID string) string {
	sum := sha256.Sum256([]byte(sessionID))
	return hex.EncodeToString(sum[:])
}

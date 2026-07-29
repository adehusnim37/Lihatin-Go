package user

import "time"

// LoginMethod identifies the final authentication factor that completed a login.
type LoginMethod string

const (
	LoginMethodEmailOTP    LoginMethod = "email_otp"
	LoginMethodTOTP        LoginMethod = "totp"
	LoginMethodGoogleOAuth LoginMethod = "google_oauth"
)

// LoginEvent is an immutable record of a completed authentication.
// Failed and incomplete authentication stages remain in login_attempts.
type LoginEvent struct {
	ID              string      `json:"id" gorm:"primaryKey;type:char(36)"`
	UserID          string      `json:"user_id" gorm:"size:191;not null;index:idx_login_events_user_time,priority:1"`
	User            *User       `json:"-" gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
	SessionIDHash   string      `json:"-" gorm:"type:char(64);not null;uniqueIndex"`
	Method          LoginMethod `json:"method" gorm:"size:32;not null"`
	DeviceID        string      `json:"device_id,omitempty" gorm:"size:255;index"`
	IPAddress       string      `json:"ip_address,omitempty" gorm:"size:45"`
	UserAgent       string      `json:"user_agent,omitempty" gorm:"size:512"`
	AuthenticatedAt time.Time   `json:"authenticated_at" gorm:"not null;index:idx_login_events_user_time,priority:2,sort:desc"`
	CreatedAt       time.Time   `json:"created_at"`
}

func (LoginEvent) TableName() string {
	return "login_events"
}

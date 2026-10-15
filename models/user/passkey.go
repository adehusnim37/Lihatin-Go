package user

import "time"

// Passkey stores the public WebAuthn credential record; private keys stay with the authenticator.
type Passkey struct {
	ID             string     `json:"id" gorm:"type:char(36);primaryKey"`
	User           *User      `json:"-" gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
	UserID         string     `json:"-" gorm:"size:191;not null;index"`
	CredentialID   string     `json:"-" gorm:"size:1024;not null"`
	CredentialHash string     `json:"-" gorm:"type:char(64);not null;uniqueIndex"`
	CredentialData string     `json:"-" gorm:"type:text;not null"`
	Name           string     `json:"name" gorm:"size:100;not null"`
	CreatedAt      time.Time  `json:"created_at"`
	LastUsedAt     *time.Time `json:"last_used_at,omitempty"`
}

func (Passkey) TableName() string { return "user_passkeys" }

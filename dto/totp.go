package dto

import (
	"time"
)

// TOTPRequest represents a request to generate a TOTP
type TOTPRequest struct {
	UserID string `json:"user_id" binding:"required"`
}

// TOTPResponse represents a response containing the TOTP
type TOTPResponse struct {
	TOTP     string    `json:"totp"`
	Expires  time.Time `json:"expires"`
}

type DisableTOTPRequest struct {
	Password string `json:"password,omitempty" label:"Kata Sandi" binding:"omitempty,min=8,max=100"`
	TOTPCode string `json:"totp_code,omitempty" label:"Kode TOTP" binding:"omitempty,len=6,numeric"`
}

type VerifyTOTPRequest struct {
	TOTPCode string `json:"totp_code" label:"Kode TOTP" binding:"required,len=6,numeric"`
}

type DisableTOTPResponse struct {
	Message string `json:"message"`
}
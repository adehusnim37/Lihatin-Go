package dto

import "time"

type CreateShortLinkRequest struct {
    UserID      string     `json:"user_id,omitempty"`
    Passcode    string     `json:"passcode,omitempty"    binding:"omitempty,len=6,numeric"`
    OriginalURL string     `json:"original_url"          binding:"required,url"`
    Title       string     `json:"title,omitempty"       binding:"omitempty,max=255"`
    Description string     `json:"description,omitempty"`
    CustomCode  string     `json:"custom_code,omitempty" binding:"omitempty,max=10,alphanum"`
    ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

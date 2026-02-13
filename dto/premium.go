package dto

import "time"

type GeneratePremiumCodeRequest struct {
	ValidUntil time.Time `json:"valid_until" binding:"required" label:"Valid Until"`
	LimitUsage int       `json:"limit_usage" binding:"required,gte=1" label:"Limit Usage"`
	IsBulk     bool      `json:"is_bulk" label:"Is Bulk"`
	Amount     int       `json:"amount,omitempty" binding:"omitempty,min=1,max=100" label:"Amount"`
}

type RedeemPremiumCodeRequest struct {
	SecretCode string `json:"secret_code" binding:"required" label:"Secret Code"`
}

type GeneratePremiumCodeResponse struct {
	ID         int                       `json:"id"`
	SecretCode string                    `json:"secret_code"`
	ValidUntil *time.Time                `json:"valid_until,omitempty"`
	LimitUsage *int64                    `json:"limit_usage,omitempty"`
	UsageCount int64                     `json:"usage_count"`
	CreatedAt  time.Time                 `json:"created_at"`
	UpdatedAt  time.Time                 `json:"updated_at"`
	KeyUsage   []PremiumKeyUsageResponse `json:"key_usage,omitempty"`
}

type PremiumKeyUsageResponse struct {
	ID           int       `json:"id"`
	PremiumKeyID int       `json:"premium_key_id"`
	UserID       string    `json:"user_id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type PaginatedPremiumKeyResponse struct {
	Keys       []GeneratePremiumCodeResponse `json:"keys"`
	Page       int                           `json:"page"`
	Limit      int                           `json:"limit"`
	Total      int                           `json:"total"`
	TotalPages int                           `json:"total_pages"`
	Sort       string                        `json:"sort"`
	OrderBy    string                        `json:"order_by"`
}

type GeneratePremiumCodeListResponse struct {
	IsBulk bool `json:"is_bulk"`
	Total  int  `json:"total"`
	Items  []GeneratePremiumCodeResponse `json:"items"`
}

type RedeemPremiumCodeResponse struct {
	SecretCode string    `json:"secret_code"`
	UpdatedAt  time.Time `json:"updated_at"`
}

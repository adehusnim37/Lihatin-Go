package dto

import "time"

// CreateShortLinkRequest represents request to create short link
type CreateShortLinkRequest struct {
	UserID      string     `json:"user_id,omitempty"`
	Passcode    string     `json:"passcode,omitempty"    binding:"omitempty,len=6,numeric"`
	OriginalURL string     `json:"original_url"          binding:"required,url"`
	Title       string     `json:"title,omitempty"       binding:"omitempty,max=255"`
	Description string     `json:"description,omitempty"`
	CustomCode  string     `json:"custom_code,omitempty" binding:"omitempty,max=10,alphanum"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// ShortLinkResponse represents short link data for API responses
type ShortsLinkResponse struct {
	ID          string     `json:"id"`
	UserID      *string    `json:"user_id,omitempty"`
	ShortCode   string     `json:"short_code"`
	OriginalURL string     `json:"original_url"`
	Title       string     `json:"title,omitempty"`
	Description string     `json:"description,omitempty"`
	IsActive    bool       `json:"is_active"`
	ExpiresAt   *time.Time `json:"expires_at"`
	CreatedAt   time.Time  `json:"created_at,omitempty"`
	UpdatedAt   time.Time  `json:"updated_at,omitempty"`
	ClickCount  int        `json:"click_count,omitempty"`
}

type ShortLinkResponse struct {
	ID              string                    `json:"id"`
	UserID          *string                   `json:"user_id,omitempty"`
	ShortCode       string                    `json:"short_code"`
	OriginalURL     string                    `json:"original_url"`
	Title           string                    `json:"title,omitempty"`
	Description     string                    `json:"description,omitempty"`
	IsActive        bool                      `json:"is_active"`
	ExpiresAt       *time.Time                `json:"expires_at"`
	CreatedAt       time.Time                 `json:"created_at"`
	UpdatedAt       time.Time                 `json:"updated_at,omitempty"`
	ShortLinkDetail *ShortLinkDetailsResponse `json:"detail,omitempty"`
	RecentViews     []ViewLinkDetailResponse  `json:"recent_views,omitempty"`
}
type ShortLinkDetailsResponse struct {
	ID            string `json:"id"`
	Passcode      int    `json:"passcode,omitempty"`
	ClickLimit    int    `json:"click_limit,omitempty"`
	CurrentClicks int    `json:"current_clicks,omitempty"`
	EnableStats   bool   `json:"enable_stats,omitempty"`
	CustomDomain  string `json:"custom_domain,omitempty"`
	UTMSource     string `json:"utm_source,omitempty"`
	UTMMedium     string `json:"utm_medium,omitempty"`
	UTMCampaign   string `json:"utm_campaign,omitempty"`
	UTMTerm       string `json:"utm_term,omitempty"`
	UTMContent    string `json:"utm_content,omitempty"`
}

type ViewLinkDetailResponse struct {
	ID        string    `json:"id"`
	IPAddress string    `json:"ip_address,omitempty"`
	UserAgent string    `json:"user_agent,omitempty"`
	Referer   string    `json:"referer,omitempty"`
	Country   string    `json:"country,omitempty"`
	City      string    `json:"city,omitempty"`
	Device    string    `json:"device,omitempty"`
	Browser   string    `json:"browser,omitempty"`
	OS        string    `json:"os,omitempty"`
	ClickedAt time.Time `json:"clicked_at,omitempty"`
}

// UpdateShortLinkRequest represents request to update short link
type UpdateShortLinkRequest struct {
	Title       *string    `json:"title" binding:"omitempty,max=255,min=3"`
	Description *string    `json:"description" binding:"omitempty,max=500,min=3"`
	IsActive    *bool      `json:"is_active" binding:"omitempty"`
	ExpiresAt   *time.Time `json:"expires_at" binding:"omitempty"`
}

// PaginatedShortLinksResponse represents paginated short links
type PaginatedShortLinksResponse struct {
	ShortLinks []ShortsLinkResponse `json:"short_links"`
	TotalCount int64                `json:"total_count"`
	Page       int                  `json:"page"`
	Limit      int                  `json:"limit"`
	TotalPages int                  `json:"total_pages"`
	Sort       string               `json:"sort" binding:"omitempty,oneof=created_at updated_at"`
	OrderBy    string               `json:"order_by" binding:"omitempty,oneof=asc desc"`
}

type CodeRequest struct {
	Code string `json:"code" binding:"required,min=1,max=100,alphanum"`
}

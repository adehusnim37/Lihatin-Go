package dto

import "time"

// In-app announcement lifecycle states persisted in the database.
const (
	AnnouncementStatusDraft     = "draft"
	AnnouncementStatusPublished = "published"
	AnnouncementStatusArchived  = "archived"
)

type CreateInAppAnnouncementRequest struct {
	Title       string     `json:"title" binding:"required,max=100"`
	Body        string     `json:"body" binding:"required,max=300"`
	Href        string     `json:"href" binding:"omitempty,url,max=150"`
	Status      string     `json:"status" binding:"required,oneof=draft published archived"`
	EndsAt      *time.Time `json:"ends_at" binding:"omitempty,gtfield=StartAt"`
	StartAt     *time.Time `json:"start_at" binding:"omitempty"`
	PublishedAt *time.Time `json:"published_at" binding:"omitempty"`
}

// UpdateInAppAnnouncementRequest replaces the editable fields of an announcement.
// The lifecycle state is changed through the dedicated publish and archive endpoints.
type UpdateInAppAnnouncementRequest struct {
	Title   string     `json:"title" binding:"required,max=100"`
	Body    string     `json:"body" binding:"required,max=300"`
	Href    string     `json:"href" binding:"omitempty,url,max=150"`
	StartAt *time.Time `json:"start_at" binding:"omitempty"`
	EndsAt  *time.Time `json:"ends_at" binding:"omitempty,gtfield=StartAt"`
}

// InAppAnnouncementInboxItem is one announcement in a user's notification inbox
// carrying that account's own read state.
type InAppAnnouncementInboxItem struct {
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	Href      string     `json:"href"`
	IsRead    bool       `json:"is_read"`
	ReadAt    *time.Time `json:"read_at,omitempty"`
	StartAt   *time.Time `json:"start_at,omitempty"`
	EndsAt    *time.Time `json:"ends_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// InAppAnnouncementInboxResponse is a paginated inbox page. Read announcements
// stay in the list so users can revisit them.
type InAppAnnouncementInboxResponse struct {
	Announcements []InAppAnnouncementInboxItem `json:"announcements"`
	UnreadCount   int64                        `json:"unread_count"`
	Page          int                          `json:"page"`
	Limit         int                          `json:"limit"`
	Total         int64                        `json:"total"`
	TotalPages    int                          `json:"total_pages"`
}

// InAppAnnouncementUnreadCountResponse backs the header inbox badge.
type InAppAnnouncementUnreadCountResponse struct {
	UnreadCount int64 `json:"unread_count"`
}

// MarkInAppAnnouncementsReadResponse reports how many announcements a bulk
// read-all call transitioned.
type MarkInAppAnnouncementsReadResponse struct {
	MarkedCount int64 `json:"marked_count"`
}

// InAppAnnouncementReaderDTO is one account's interaction with an announcement.
type InAppAnnouncementReaderDTO struct {
	UserID      string     `json:"user_id"`
	Username    string     `json:"username"`
	FirstName   string     `json:"first_name"`
	LastName    string     `json:"last_name"`
	Email       string     `json:"email"`
	Avatar      string     `json:"avatar"`
	IsRead      bool       `json:"is_read"`
	ReadAt      *time.Time `json:"read_at,omitempty"`
	FirstSeenAt time.Time  `json:"first_seen_at"`
}

// InAppAnnouncementAudienceResponse describes who has seen an announcement.
type InAppAnnouncementAudienceResponse struct {
	ReadCount   int64                        `json:"read_count"`
	UnreadCount int64                        `json:"unread_count"`
	TotalUsers  int64                        `json:"total_users"`
	Readers     []InAppAnnouncementReaderDTO `json:"readers"`
	Page        int                          `json:"page"`
	Limit       int                          `json:"limit"`
	Total       int64                        `json:"total"`
	TotalPages  int                          `json:"total_pages"`
	Status      string                       `json:"status"`
}

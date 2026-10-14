package dto

import "time"

type CreateInAppAnnouncementRequest struct {
	Title       string     `json:"title" binding:"required,max=100"`
	Body        string     `json:"body" binding:"required,max=150"`
	Href        string     `json:"href" binding:"omitempty,url,max=150"`
	Status      string     `json:"status" binding:"required,oneof=draft published archived"`
	EndsAt      *time.Time `json:"ends_at" binding:"omitempty,gtfield=StartAt"`
	StartAt     *time.Time `json:"start_at" binding:"omitempty"`
	PublishedAt *time.Time `json:"published_at" binding:"omitempty"`
}

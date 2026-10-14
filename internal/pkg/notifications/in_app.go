package notifications

import (
	"context"
	"time"

	"github.com/adehusnim37/lihatin-go/internal/pkg/identifier"
	"github.com/adehusnim37/lihatin-go/models/announcements"
	"github.com/adehusnim37/lihatin-go/repositories/announcementrepo"
)

const APIRateLimitAnnouncementID = "api-rate-limits-2026-09"

type InAppAnnouncementService struct {
	repository *announcementrepo.InAppRepository
	now        func() time.Time
}

func NewInAppAnnouncementService(repository *announcementrepo.InAppRepository) *InAppAnnouncementService {
	return &InAppAnnouncementService{repository: repository, now: time.Now}
}

type InAppAnnouncement struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	Href      string `json:"href"`
	CreatedBy string `json:"created_by"`
}

func (s *InAppAnnouncementService) Pending(ctx context.Context, userID string) ([]InAppAnnouncement, error) {
	rows, err := s.repository.ListPendingForUser(ctx, userID, s.now().UTC())
	if err != nil {
		return nil, err
	}
	result := make([]InAppAnnouncement, 0, len(rows))
	for _, row := range rows {
		result = append(result, InAppAnnouncement{ID: row.ID, Title: row.Title, Body: row.Body, Href: row.Href, CreatedBy: row.CreatedBy})
	}
	return result, nil
}

func (s *InAppAnnouncementService) MarkRead(ctx context.Context, userID, announcementID string) error {
	if _, err := s.repository.Get(ctx, announcementID); err != nil {
		return err
	}
	return s.repository.MarkRead(ctx, userID, announcementID)
}

func (s *InAppAnnouncementService) CreateAnnouncement(ctx context.Context, title, body, href, createdBy, status string, startAt, endsAt, publishedAt *time.Time) (*announcements.InAppAnnouncement, error) {
	announcement := &announcements.InAppAnnouncement{
		ID:          identifier.NewUUIDV7(),
		Title:       title,
		Body:        body,
		Href:        href,
		Status:      status,
		StartAt:     startAt,
		EndsAt:      endsAt,
		PublishedAt: publishedAt,
		CreatedBy:   createdBy,
	}
	if err := s.repository.Create(ctx, announcement); err != nil {
		return nil, err
	}
	return announcement, nil
}

func (s *InAppAnnouncementService) ArchiveAnnouncement(ctx context.Context, announcementID string) error {
	return s.repository.Archive(ctx, announcementID)
}

func (s *InAppAnnouncementService) UpdateAnnouncement(ctx context.Context, announcementID, title, body, href string, startAt, endsAt *time.Time) (*announcements.InAppAnnouncement, error) {
	announcement, err := s.repository.Get(ctx, announcementID)
	if err != nil {
		return nil, err
	}

	announcement.Title = title
	announcement.Body = body
	announcement.Href = href
	announcement.StartAt = startAt
	announcement.EndsAt = endsAt

	if err := s.repository.Update(ctx, announcement); err != nil {
		return nil, err
	}
	return announcement, nil
}

func (s *InAppAnnouncementService) DeleteAnnouncement(ctx context.Context, announcementID string) (*announcements.InAppAnnouncement, error) {
	announcement, err := s.repository.Get(ctx, announcementID)
	if err != nil {
		return nil, err
	}

	if err := s.repository.Delete(ctx, announcement); err != nil {
		return nil, err
	}
	return announcement, nil
}

func (s *InAppAnnouncementService) GetAnnouncement(ctx context.Context, announcementID string) (*announcements.InAppAnnouncement, error) {
	return s.repository.Get(ctx, announcementID)
}

func (s *InAppAnnouncementService) ListAnnouncements(ctx context.Context, status string) ([]announcements.InAppAnnouncement, error) {
	return s.repository.List(ctx, status)
}

func (s *InAppAnnouncementService) CountPendingAnnouncements(ctx context.Context, userID string) (int64, error) {
	return s.repository.CountPendingForUser(ctx, userID, s.now().UTC())
}

func (s *InAppAnnouncementService) MarkAllRead(ctx context.Context, userID string) error {
	active, err := s.repository.ListActive(ctx, s.now().UTC())
	if err != nil {
		return err
	}

	for _, announcement := range active {
		if err := s.MarkRead(ctx, userID, announcement.ID); err != nil {
			return err
		}
	}
	return nil
}

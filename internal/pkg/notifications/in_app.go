package notifications

import (
	"context"
	"time"

	"github.com/adehusnim37/lihatin-go/dto"
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

// MarkUnread moves an announcement back into the account's unread state so the
// header badge and auto-popup can surface it again.
func (s *InAppAnnouncementService) MarkUnread(ctx context.Context, userID, announcementID string) error {
	if _, err := s.repository.Get(ctx, announcementID); err != nil {
		return err
	}
	return s.repository.MarkUnread(ctx, userID, announcementID)
}

// Inbox returns the account's notification inbox, including announcements the
// account already dismissed so they can be read again.
func (s *InAppAnnouncementService) Inbox(ctx context.Context, userID string, page, limit int) (*dto.InAppAnnouncementInboxResponse, error) {
	now := s.now().UTC()
	rows, total, err := s.repository.ListInboxForUser(ctx, userID, now, limit, (page-1)*limit)
	if err != nil {
		return nil, err
	}

	unread, err := s.repository.CountUnreadForUser(ctx, userID, now)
	if err != nil {
		return nil, err
	}

	items := make([]dto.InAppAnnouncementInboxItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, dto.InAppAnnouncementInboxItem{
			ID:        row.ID,
			Title:     row.Title,
			Body:      row.Body,
			Href:      row.Href,
			IsRead:    row.IsRead,
			ReadAt:    row.ReadAt,
			StartAt:   row.StartAt,
			EndsAt:    row.EndsAt,
			CreatedAt: row.CreatedAt,
		})
	}

	return &dto.InAppAnnouncementInboxResponse{
		Announcements: items,
		UnreadCount:   unread,
		Page:          page,
		Limit:         limit,
		Total:         total,
		TotalPages:    pageCount(total, limit),
	}, nil
}

// UnreadCount powers the header inbox badge.
func (s *InAppAnnouncementService) UnreadCount(ctx context.Context, userID string) (int64, error) {
	return s.repository.CountUnreadForUser(ctx, userID, s.now().UTC())
}

// MarkAllRead dismisses every announcement currently visible to the account.
func (s *InAppAnnouncementService) MarkAllRead(ctx context.Context, userID string) (int64, error) {
	return s.repository.MarkAllReadForUser(ctx, userID, s.now().UTC())
}

// Audience reports who dismissed an announcement out of its eligible reach.
// read=true lists accounts that dismissed it, read=false lists accounts with a
// pending record.
func (s *InAppAnnouncementService) Audience(ctx context.Context, announcementID string, read bool, page, limit int) (*dto.InAppAnnouncementAudienceResponse, error) {
	if _, err := s.repository.Get(ctx, announcementID); err != nil {
		return nil, err
	}

	readers, total, err := s.repository.ListReaders(ctx, announcementID, read, limit, (page-1)*limit)
	if err != nil {
		return nil, err
	}

	aggregate, err := s.repository.AggregateForAnnouncement(ctx, announcementID, s.now().UTC())
	if err != nil {
		return nil, err
	}

	items := make([]dto.InAppAnnouncementReaderDTO, 0, len(readers))
	for _, reader := range readers {
		items = append(items, dto.InAppAnnouncementReaderDTO{
			UserID:      reader.UserID,
			Username:    reader.Username,
			FirstName:   reader.FirstName,
			LastName:    reader.LastName,
			Email:       reader.Email,
			Avatar:      reader.Avatar,
			IsRead:      reader.IsRead,
			ReadAt:      reader.ReadAt,
			FirstSeenAt: reader.FirstSeenAt,
		})
	}

	status := "unread"
	if read {
		status = "read"
	}

	return &dto.InAppAnnouncementAudienceResponse{
		ReadCount:   aggregate.ReadCount,
		UnreadCount: aggregate.UnreadCount,
		TotalUsers:  aggregate.TotalUsers,
		Readers:     items,
		Page:        page,
		Limit:       limit,
		Total:       total,
		TotalPages:  pageCount(total, limit),
		Status:      status,
	}, nil
}

// ReadAgain clears the account's read state so the announcement reappears in
// the auto-popup queue.
func (s *InAppAnnouncementService) ReadAgain(ctx context.Context, userID, announcementID string) error {
	return s.MarkUnread(ctx, userID, announcementID)
}

func pageCount(total int64, limit int) int {
	if limit < 1 {
		return 1
	}
	return int((total + int64(limit) - 1) / int64(limit))
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

// PublishAnnouncement makes a draft or archived announcement visible to users.
// An explicit publication timestamp is set once so announcements without a
// start window still surface immediately.
func (s *InAppAnnouncementService) PublishAnnouncement(ctx context.Context, announcementID string) (*AdminAnnouncement, error) {
	announcement, err := s.repository.Get(ctx, announcementID)
	if err != nil {
		return nil, err
	}

	if announcement.PublishedAt == nil {
		publishedAt := s.now().UTC()
		announcement.PublishedAt = &publishedAt
	}
	announcement.Status = "published"

	if err := s.repository.Update(ctx, announcement); err != nil {
		return nil, err
	}
	return s.decorateForAdmin(ctx, announcement)
}

func (s *InAppAnnouncementService) UpdateAnnouncement(ctx context.Context, announcementID, title, body, href string, startAt, endsAt *time.Time) (*AdminAnnouncement, error) {
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
	return s.decorateForAdmin(ctx, announcement)
}

func (s *InAppAnnouncementService) DeleteAnnouncement(ctx context.Context, announcementID string) (*AdminAnnouncement, error) {
	announcement, err := s.repository.Get(ctx, announcementID)
	if err != nil {
		return nil, err
	}

	// Read counts are captured before deletion because the dismissal records are
	// removed together with the announcement.
	decorated, err := s.decorateForAdmin(ctx, announcement)
	if err != nil {
		return nil, err
	}

	if err := s.repository.Delete(ctx, announcement); err != nil {
		return nil, err
	}
	return decorated, nil
}

func (s *InAppAnnouncementService) GetAnnouncement(ctx context.Context, announcementID string) (*AdminAnnouncement, error) {
	announcement, err := s.repository.Get(ctx, announcementID)
	if err != nil {
		return nil, err
	}
	return s.decorateForAdmin(ctx, announcement)
}

func (s *InAppAnnouncementService) ListAnnouncements(ctx context.Context, status string) ([]announcements.InAppAnnouncement, error) {
	return s.repository.List(ctx, status)
}

// AdminAnnouncement augments a stored announcement with the number of accounts
// that have already dismissed it.
type AdminAnnouncement struct {
	announcements.InAppAnnouncement
	ReadCount int64 `json:"read_count"`
}

// AdminAnnouncementPage is a single page of announcements prepared for the
// administration dashboard.
type AdminAnnouncementPage struct {
	Announcements []AdminAnnouncement `json:"announcements"`
	Page          int                 `json:"page"`
	Limit         int                 `json:"limit"`
	Total         int64               `json:"total"`
	TotalPages    int                 `json:"total_pages"`
	StatusCounts  map[string]int64    `json:"status_counts"`
}

func (s *InAppAnnouncementService) ListForAdmin(ctx context.Context, status string, page, limit int) (*AdminAnnouncementPage, error) {
	rows, total, err := s.repository.ListPaginated(ctx, status, limit, (page-1)*limit)
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	readCounts, err := s.repository.ReadCounts(ctx, ids)
	if err != nil {
		return nil, err
	}

	statusCounts, err := s.repository.StatusCounts(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]AdminAnnouncement, 0, len(rows))
	for _, row := range rows {
		result = append(result, AdminAnnouncement{
			InAppAnnouncement: row,
			ReadCount:         readCounts[row.ID],
		})
	}

	totalPages := int((total + int64(limit) - 1) / int64(limit))
	return &AdminAnnouncementPage{
		Announcements: result,
		Page:          page,
		Limit:         limit,
		Total:         total,
		TotalPages:    totalPages,
		StatusCounts:  statusCounts,
	}, nil
}

// decorateForAdmin attaches the dismissal count so single-record admin
// responses carry the same `read_count` field as the paginated list.
func (s *InAppAnnouncementService) decorateForAdmin(ctx context.Context, announcement *announcements.InAppAnnouncement) (*AdminAnnouncement, error) {
	counts, err := s.repository.ReadCounts(ctx, []string{announcement.ID})
	if err != nil {
		return nil, err
	}
	return &AdminAnnouncement{
		InAppAnnouncement: *announcement,
		ReadCount:         counts[announcement.ID],
	}, nil
}

func (s *InAppAnnouncementService) CountPendingAnnouncements(ctx context.Context, userID string) (int64, error) {
	return s.repository.CountPendingForUser(ctx, userID, s.now().UTC())
}

package announcementrepo

import (
	"context"
	"errors"
	"time"

	apperrors "github.com/adehusnim37/lihatin-go/internal/pkg/errors"
	"github.com/adehusnim37/lihatin-go/models/announcements"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type InAppRepository struct {
	db *gorm.DB
}

func NewInAppRepository(db *gorm.DB) *InAppRepository {
	return &InAppRepository{db: db}
}

func (r *InAppRepository) ListPendingForUser(ctx context.Context, userID string, now time.Time) ([]announcements.InAppAnnouncement, error) {
	var result []announcements.InAppAnnouncement
	err := r.db.WithContext(ctx).
		Table("announcements_in_app AS a").
		Select("a.*").
		Where("a.status = ?", "published").
		Where("(a.start_at IS NULL OR a.start_at <= ?)", now).
		Where("(a.ends_at IS NULL OR a.ends_at > ?)", now).
		Where("NOT EXISTS (?)", r.db.Table("announcements_in_app_reads AS r").
			Select("1").
			Where("r.announcement_id = a.id AND r.user_id = ? AND r.is_read = ?", userID, true)).
		Order("a.start_at DESC, a.created_at DESC").
		Scan(&result).Error
	if err != nil {
		return nil, apperrors.ErrAnnouncementListFailed.WithError(err)
	}
	return result, nil
}

// InboxItem is one announcement in a user's notification inbox together with
// that account's own read state.
type InboxItem struct {
	announcements.InAppAnnouncement
	IsRead bool       `json:"is_read"`
	ReadAt *time.Time `json:"read_at,omitempty"`
}

// ListInboxForUser returns every announcement the account can see — with the
// newest first — annotating whether that account already dismissed each one.
// Unlike ListPendingForUser it keeps read announcements so users can revisit
// and re-read announcements from the header inbox.
func (r *InAppRepository) ListInboxForUser(ctx context.Context, userID string, now time.Time, limit, offset int) ([]InboxItem, int64, error) {
	visible := func() *gorm.DB {
		return r.db.WithContext(ctx).
			Table("announcements_in_app AS a").
			Where("a.status = ?", "published").
			Where("(a.start_at IS NULL OR a.start_at <= ?)", now).
			Where("(a.ends_at IS NULL OR a.ends_at > ?)", now)
	}

	var total int64
	if err := visible().Count(&total).Error; err != nil {
		return nil, 0, apperrors.ErrAnnouncementListFailed.WithError(err)
	}

	var result []InboxItem
	if err := visible().
		Select("a.*, COALESCE(r.is_read, ?) AS is_read, r.read_at AS read_at", false).
		Joins("LEFT JOIN announcements_in_app_reads AS r ON r.announcement_id = a.id AND r.user_id = ?", userID).
		Order("a.start_at DESC, a.created_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(&result).Error; err != nil {
		return nil, 0, apperrors.ErrAnnouncementListFailed.WithError(err)
	}

	return result, total, nil
}

// CountUnreadForUser counts announcements the account can still see but has not
// dismissed yet. Used for the header inbox badge.
func (r *InAppRepository) CountUnreadForUser(ctx context.Context, userID string, now time.Time) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("announcements_in_app AS a").
		Where("a.status = ?", "published").
		Where("(a.start_at IS NULL OR a.start_at <= ?)", now).
		Where("(a.ends_at IS NULL OR a.ends_at > ?)", now).
		Where("NOT EXISTS (?)", r.db.Table("announcements_in_app_reads AS r").
			Select("1").
			Where("r.announcement_id = a.id AND r.user_id = ? AND r.is_read = ?", userID, true)).
		Count(&count).Error
	if err != nil {
		return 0, apperrors.ErrAnnouncementListFailed.WithError(err)
	}
	return count, nil
}

// MarkAllReadForUser dismisses every currently visible announcement for one
// account in a single upsert.
func (r *InAppRepository) MarkAllReadForUser(ctx context.Context, userID string, now time.Time) (int64, error) {
	var ids []string
	if err := r.db.WithContext(ctx).
		Model(&announcements.InAppAnnouncement{}).
		Where("status = ?", "published").
		Where("(start_at IS NULL OR start_at <= ?)", now).
		Where("(ends_at IS NULL OR ends_at > ?)", now).
		Pluck("id", &ids).Error; err != nil {
		return 0, apperrors.ErrAnnouncementListFailed.WithError(err)
	}
	if len(ids) == 0 {
		return 0, nil
	}

	readAt := now
	rows := make([]announcements.InAppAnnouncementRead, 0, len(ids))
	for _, id := range ids {
		rows = append(rows, announcements.InAppAnnouncementRead{
			ID:             uuid.NewString(),
			UserID:         userID,
			AnnouncementID: id,
			IsRead:         true,
			ReadAt:         &readAt,
		})
	}

	result := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "announcement_id"}},
		DoUpdates: clause.Assignments(map[string]any{"is_read": true, "read_at": readAt}),
	}).Create(&rows)
	if result.Error != nil {
		return 0, apperrors.ErrAnnouncementReadFailed.WithError(result.Error)
	}
	return result.RowsAffected, nil
}

func (r *InAppRepository) Get(ctx context.Context, announcementID string) (*announcements.InAppAnnouncement, error) {
	var announcement announcements.InAppAnnouncement
	if err := r.db.WithContext(ctx).Where("id = ?", announcementID).First(&announcement).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.ErrAnnouncementNotFound.WithError(err)
		}
		return nil, apperrors.ErrAnnouncementFindFailed.WithError(err)
	}
	return &announcement, nil
}

func (r *InAppRepository) Create(ctx context.Context, announcement *announcements.InAppAnnouncement) error {
	if err := r.db.WithContext(ctx).Create(announcement).Error; err != nil {
		return apperrors.ErrAnnouncementCreateFailed.WithError(err)
	}
	return nil
}

func (r *InAppRepository) Update(ctx context.Context, announcement *announcements.InAppAnnouncement) error {
	if err := r.db.WithContext(ctx).Save(announcement).Error; err != nil {
		return apperrors.ErrAnnouncementUpdateFailed.WithError(err)
	}
	return nil
}

func (r *InAppRepository) Archive(ctx context.Context, announcementID string) error {
	result := r.db.WithContext(ctx).Model(&announcements.InAppAnnouncement{}).
		Where("id = ?", announcementID).
		Update("status", "archived")
	if result.Error != nil {
		return apperrors.ErrAnnouncementArchiveFailed.WithError(result.Error)
	}
	if result.RowsAffected == 0 {
		return apperrors.ErrAnnouncementNotFound
	}
	return nil
}

// Delete removes an announcement together with every dismissal record that
// points at it, so no orphaned read rows survive the deletion.
func (r *InAppRepository) Delete(ctx context.Context, announcement *announcements.InAppAnnouncement) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("announcement_id = ?", announcement.ID).
			Delete(&announcements.InAppAnnouncementRead{}).Error; err != nil {
			return err
		}
		return tx.Delete(announcement).Error
	})
	if err != nil {
		return apperrors.ErrAnnouncementDeleteFailed.WithError(err)
	}
	return nil
}

func (r *InAppRepository) List(ctx context.Context, status string) ([]announcements.InAppAnnouncement, error) {
	var result []announcements.InAppAnnouncement
	query := r.db.WithContext(ctx).Model(&announcements.InAppAnnouncement{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.Order("start_at DESC, created_at DESC").Find(&result).Error; err != nil {
		return nil, apperrors.ErrAnnouncementListFailed.WithError(err)
	}
	return result, nil
}

// ListPaginated returns one page of announcements together with the total row
// count that matches the optional status filter.
func (r *InAppRepository) ListPaginated(ctx context.Context, status string, limit, offset int) ([]announcements.InAppAnnouncement, int64, error) {
	base := func() *gorm.DB {
		query := r.db.WithContext(ctx).Model(&announcements.InAppAnnouncement{})
		if status != "" {
			query = query.Where("status = ?", status)
		}
		return query
	}

	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, apperrors.ErrAnnouncementListFailed.WithError(err)
	}

	var result []announcements.InAppAnnouncement
	if err := base().
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&result).Error; err != nil {
		return nil, 0, apperrors.ErrAnnouncementListFailed.WithError(err)
	}

	return result, total, nil
}

// StatusCounts returns how many announcements exist for each lifecycle state.
// States without rows are omitted from the map.
func (r *InAppRepository) StatusCounts(ctx context.Context) (map[string]int64, error) {
	type row struct {
		Status string
		Total  int64
	}

	var rows []row
	if err := r.db.WithContext(ctx).
		Model(&announcements.InAppAnnouncement{}).
		Select("status, COUNT(*) AS total").
		Group("status").
		Scan(&rows).Error; err != nil {
		return nil, apperrors.ErrAnnouncementListFailed.WithError(err)
	}

	counts := make(map[string]int64, len(rows))
	for _, item := range rows {
		counts[item.Status] = item.Total
	}
	return counts, nil
}

// ReadCounts maps each announcement ID to the number of accounts that dismissed it.
func (r *InAppRepository) ReadCounts(ctx context.Context, announcementIDs []string) (map[string]int64, error) {
	counts := make(map[string]int64, len(announcementIDs))
	if len(announcementIDs) == 0 {
		return counts, nil
	}

	type row struct {
		AnnouncementID string
		Total          int64
	}

	var rows []row
	if err := r.db.WithContext(ctx).
		Model(&announcements.InAppAnnouncementRead{}).
		Select("announcement_id, COUNT(*) AS total").
		Where("announcement_id IN ?", announcementIDs).
		Where("is_read = ?", true).
		Group("announcement_id").
		Scan(&rows).Error; err != nil {
		return nil, apperrors.ErrAnnouncementListFailed.WithError(err)
	}

	for _, item := range rows {
		counts[item.AnnouncementID] = item.Total
	}
	return counts, nil
}

// UpdateStatus moves an announcement into another lifecycle state.
func (r *InAppRepository) UpdateStatus(ctx context.Context, announcementID, status string) error {
	result := r.db.WithContext(ctx).Model(&announcements.InAppAnnouncement{}).
		Where("id = ?", announcementID).
		Update("status", status)
	if result.Error != nil {
		return apperrors.ErrAnnouncementUpdateFailed.WithError(result.Error)
	}
	if result.RowsAffected == 0 {
		return apperrors.ErrAnnouncementNotFound
	}
	return nil
}

func (r *InAppRepository) ListActive(ctx context.Context, now time.Time) ([]announcements.InAppAnnouncement, error) {
	var result []announcements.InAppAnnouncement
	err := r.db.WithContext(ctx).
		Where("status = ?", "published").
		Where("(start_at IS NULL OR start_at <= ?)", now).
		Where("(ends_at IS NULL OR ends_at > ?)", now).
		Find(&result).Error
	if err != nil {
		return nil, apperrors.ErrAnnouncementListFailed.WithError(err)
	}
	return result, nil
}

func (r *InAppRepository) CountPendingForUser(ctx context.Context, userID string, now time.Time) (int64, error) {
	return r.CountUnreadForUser(ctx, userID, now)
}

func (r *InAppRepository) MarkRead(ctx context.Context, userID, announcementID string) error {
	readAt := time.Now().UTC()
	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "announcement_id"}},
		DoUpdates: clause.Assignments(map[string]any{"is_read": true, "read_at": readAt}),
	}).Create(&announcements.InAppAnnouncementRead{
		ID: uuid.NewString(), UserID: userID, AnnouncementID: announcementID, IsRead: true, ReadAt: &readAt,
	}).Error; err != nil {
		return apperrors.ErrAnnouncementReadFailed.WithError(err)
	}
	return nil
}

// MarkUnread lets an account put an announcement back into its unread state so
// the header badge and auto-popup surface it again.
func (r *InAppRepository) MarkUnread(ctx context.Context, userID, announcementID string) error {
	result := r.db.WithContext(ctx).Model(&announcements.InAppAnnouncementRead{}).
		Where("user_id = ? AND announcement_id = ?", userID, announcementID).
		Updates(map[string]any{"is_read": false, "read_at": nil})
	if result.Error != nil {
		return apperrors.ErrAnnouncementReadFailed.WithError(result.Error)
	}
	return nil
}

// AnnouncementReader is one account's dismissal record joined with its profile.
type AnnouncementReader struct {
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

// ListReaders returns the accounts that interacted with an announcement.
// Pass read=true for accounts that dismissed it, read=false for accounts that
// have a record but have not dismissed it yet.
func (r *InAppRepository) ListReaders(ctx context.Context, announcementID string, read bool, limit, offset int) ([]AnnouncementReader, int64, error) {
	base := func() *gorm.DB {
		return r.db.WithContext(ctx).
			Table("announcements_in_app_reads AS rr").
			Joins("JOIN users AS u ON u.id = rr.user_id AND u.deleted_at IS NULL").
			Where("rr.announcement_id = ?", announcementID).
			Where("rr.is_read = ?", read)
	}

	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, apperrors.ErrAnnouncementListFailed.WithError(err)
	}

	var result []AnnouncementReader
	if err := base().
		Select(`u.id AS user_id, u.username, u.first_name, u.last_name, u.email, u.avatar,
			rr.is_read, rr.read_at, rr.created_at AS first_seen_at`).
		Order("rr.read_at DESC, rr.created_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(&result).Error; err != nil {
		return nil, 0, apperrors.ErrAnnouncementListFailed.WithError(err)
	}

	return result, total, nil
}

// ReadAggregate summarizes an announcement's audience: how many accounts have
// dismissed it and how many are still able to see it.
type ReadAggregate struct {
	ReadCount   int64
	UnreadCount int64
	TotalUsers  int64
}

func (r *InAppRepository) AggregateForAnnouncement(ctx context.Context, announcementID string, now time.Time) (ReadAggregate, error) {
	var aggregate ReadAggregate

	if err := r.db.WithContext(ctx).
		Model(&announcements.InAppAnnouncementRead{}).
		Where("announcement_id = ? AND is_read = ?", announcementID, true).
		Count(&aggregate.ReadCount).Error; err != nil {
		return aggregate, apperrors.ErrAnnouncementListFailed.WithError(err)
	}

	if err := r.db.WithContext(ctx).
		Model(&announcements.InAppAnnouncementRead{}).
		Where("announcement_id = ? AND is_read = ?", announcementID, false).
		Count(&aggregate.UnreadCount).Error; err != nil {
		return aggregate, apperrors.ErrAnnouncementListFailed.WithError(err)
	}

	// Eligible audience: verified, active accounts — the same population the
	// announcement endpoints expose the notice to.
	if err := r.db.WithContext(ctx).
		Table("users AS u").
		Joins("JOIN user_auth AS ua ON ua.user_id = u.id AND ua.deleted_at IS NULL").
		Where("u.deleted_at IS NULL").
		Where("ua.account_status = ? AND ua.is_email_verified = ?", "active", true).
		Count(&aggregate.TotalUsers).Error; err != nil {
		return aggregate, apperrors.ErrAnnouncementListFailed.WithError(err)
	}

	return aggregate, nil
}

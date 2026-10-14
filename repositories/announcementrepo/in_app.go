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

func (r *InAppRepository) Delete(ctx context.Context, announcement *announcements.InAppAnnouncement) error {
	if err := r.db.WithContext(ctx).Delete(announcement).Error; err != nil {
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
	var count int64
	err := r.db.WithContext(ctx).
		Model(&announcements.InAppAnnouncement{}).
		Where("status = ?", "published").
		Where("(start_at IS NULL OR start_at <= ?)", now).
		Where("(ends_at IS NULL OR ends_at > ?)", now).
		Where("NOT EXISTS (?)", r.db.Table("announcements_in_app_reads AS r").
			Select("1").
			Where("r.announcement_id = announcements_in_app.id AND r.user_id = ? AND r.is_read = ?", userID, true)).
		Count(&count).Error
	if err != nil {
		return 0, apperrors.ErrAnnouncementListFailed.WithError(err)
	}
	return count, nil
}

func (r *InAppRepository) MarkRead(ctx context.Context, userID, announcementID string) error {
	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "announcement_id"}},
		DoUpdates: clause.Assignments(map[string]any{"is_read": true}),
	}).Create(&announcements.InAppAnnouncementRead{
		ID: uuid.NewString(), UserID: userID, AnnouncementID: announcementID, IsRead: true,
	}).Error; err != nil {
		return apperrors.ErrAnnouncementReadFailed.WithError(err)
	}
	return nil
}

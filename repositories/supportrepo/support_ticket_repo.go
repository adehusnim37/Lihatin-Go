package supportrepo

import (
	"errors"
	"strings"
	"time"

	apperrors "github.com/adehusnim37/lihatin-go/internal/pkg/errors"
	supportmodel "github.com/adehusnim37/lihatin-go/models/support"
	"gorm.io/gorm"
)

type TicketListFilters struct {
	Status   string
	Category string
	Priority string
	Search   string
	Email    string
}

type Pagination struct {
	Page  int
	Limit int
}

type MessageListFilters struct {
	IncludeInternal bool
}

type SupportTicketRepository struct {
	db *gorm.DB
}

func NewSupportTicketRepository(db *gorm.DB) *SupportTicketRepository {
	return &SupportTicketRepository{db: db}
}

func supportTicketNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return apperrors.ErrSupportTicketNotFound.WithError(err)
	}
	return apperrors.ErrSupportTicketFindFailed.WithError(err)
}

func supportAttachmentNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return apperrors.ErrSupportAttachmentNotFound.WithError(err)
	}
	return apperrors.ErrSupportAttachmentFindFailed.WithError(err)
}

func (r *SupportTicketRepository) CreateTicket(ticket *supportmodel.SupportTicket) error {
	if err := r.db.Create(ticket).Error; err != nil {
		return apperrors.ErrSupportTicketCreateFailed.WithError(err)
	}
	return nil
}

func (r *SupportTicketRepository) CreateMessage(message *supportmodel.SupportMessage) error {
	if err := r.db.Create(message).Error; err != nil {
		return apperrors.ErrSupportMessageCreateFailed.WithError(err)
	}
	return nil
}

func (r *SupportTicketRepository) CreateMessageWithAttachments(message *supportmodel.SupportMessage, attachments []supportmodel.SupportAttachment) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(message).Error; err != nil {
			return apperrors.ErrSupportMessageCreateFailed.WithError(err)
		}

		if len(attachments) > 0 {
			if err := tx.Create(&attachments).Error; err != nil {
				return apperrors.ErrSupportAttachmentCreateFailed.WithError(err)
			}
		}

		return nil
	})
}

func (r *SupportTicketRepository) GetStatusByCode(code string) (*supportmodel.SupportTicket, error) {
	var ticket supportmodel.SupportTicket
	if err := r.db.Where("ticket_code = ?", strings.TrimSpace(code)).First(&ticket).Error; err != nil {
		return nil, supportTicketNotFound(err)
	}
	return &ticket, nil
}

func (r *SupportTicketRepository) GetTicketByCode(code string) (*supportmodel.SupportTicket, error) {
	var ticket supportmodel.SupportTicket
	if err := r.db.Where("ticket_code = ?", strings.TrimSpace(code)).First(&ticket).Error; err != nil {
		return nil, supportTicketNotFound(err)
	}
	return &ticket, nil
}

func (r *SupportTicketRepository) TicketCodeExists(code string) (bool, error) {
	var count int64
	if err := r.db.Model(&supportmodel.SupportTicket{}).Where("ticket_code = ?", strings.TrimSpace(code)).Count(&count).Error; err != nil {
		return false, apperrors.ErrSupportTicketCheckFailed.WithError(err)
	}
	return count > 0, nil
}

func (r *SupportTicketRepository) GetTicketByID(id string) (*supportmodel.SupportTicket, error) {
	var ticket supportmodel.SupportTicket
	if err := r.db.Where("id = ?", strings.TrimSpace(id)).First(&ticket).Error; err != nil {
		return nil, supportTicketNotFound(err)
	}
	return &ticket, nil
}

func (r *SupportTicketRepository) GetTicketByCodeAndEmail(code, email string) (*supportmodel.SupportTicket, error) {
	var ticket supportmodel.SupportTicket
	err := r.db.
		Where("ticket_code = ? AND LOWER(email) = LOWER(?)", strings.TrimSpace(code), strings.TrimSpace(email)).
		First(&ticket).Error
	if err != nil {
		return nil, supportTicketNotFound(err)
	}
	return &ticket, nil
}

func (r *SupportTicketRepository) ListTickets(filters TicketListFilters, pagination Pagination) ([]supportmodel.SupportTicket, int64, error) {
	query := r.db.Model(&supportmodel.SupportTicket{})

	if status := strings.TrimSpace(filters.Status); status != "" {
		query = query.Where("status = ?", status)
	}
	if category := strings.TrimSpace(filters.Category); category != "" {
		query = query.Where("category = ?", category)
	}
	if priority := strings.TrimSpace(filters.Priority); priority != "" {
		query = query.Where("priority = ?", priority)
	}
	if email := strings.TrimSpace(filters.Email); email != "" {
		query = query.Where("LOWER(email) = LOWER(?)", email)
	}
	if search := strings.TrimSpace(filters.Search); search != "" {
		like := "%" + search + "%"
		query = query.Where("ticket_code LIKE ? OR subject LIKE ? OR email LIKE ?", like, like, like)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, apperrors.ErrSupportTicketCountFailed.WithError(err)
	}

	if pagination.Page < 1 {
		pagination.Page = 1
	}
	if pagination.Limit < 1 {
		pagination.Limit = 20
	}
	if pagination.Limit > 100 {
		pagination.Limit = 100
	}

	offset := (pagination.Page - 1) * pagination.Limit
	items := make([]supportmodel.SupportTicket, 0)
	if err := query.
		Order("created_at DESC").
		Offset(offset).
		Limit(pagination.Limit).
		Find(&items).Error; err != nil {
		return nil, 0, apperrors.ErrSupportTicketListFailed.WithError(err)
	}

	return items, total, nil
}

func (r *SupportTicketRepository) ListTicketsForUser(userID, email string, pagination Pagination) ([]supportmodel.SupportTicket, int64, error) {
	query := r.db.Model(&supportmodel.SupportTicket{})

	normalizedUserID := strings.TrimSpace(userID)
	normalizedEmail := strings.TrimSpace(email)
	if normalizedUserID != "" {
		query = query.Where("user_id = ? OR LOWER(email) = LOWER(?)", normalizedUserID, normalizedEmail)
	} else {
		query = query.Where("LOWER(email) = LOWER(?)", normalizedEmail)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, apperrors.ErrSupportTicketCountFailed.WithError(err)
	}

	if pagination.Page < 1 {
		pagination.Page = 1
	}
	if pagination.Limit < 1 {
		pagination.Limit = 20
	}
	if pagination.Limit > 100 {
		pagination.Limit = 100
	}

	offset := (pagination.Page - 1) * pagination.Limit
	items := make([]supportmodel.SupportTicket, 0)
	if err := query.
		Order("created_at DESC").
		Offset(offset).
		Limit(pagination.Limit).
		Find(&items).Error; err != nil {
		return nil, 0, apperrors.ErrSupportTicketListFailed.WithError(err)
	}

	return items, total, nil
}

func (r *SupportTicketRepository) ListMessagesByTicketID(ticketID string, filters MessageListFilters) ([]supportmodel.SupportMessage, error) {
	query := r.db.Model(&supportmodel.SupportMessage{}).
		Where("ticket_id = ?", strings.TrimSpace(ticketID))

	if !filters.IncludeInternal {
		query = query.Where("is_internal = ?", false)
	}

	items := make([]supportmodel.SupportMessage, 0)
	if err := query.
		Preload("Attachments", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at ASC")
		}).
		Order("created_at ASC").
		Find(&items).Error; err != nil {
		return nil, apperrors.ErrSupportMessageListFailed.WithError(err)
	}

	return items, nil
}

func (r *SupportTicketRepository) GetAttachmentByID(id string) (*supportmodel.SupportAttachment, error) {
	var item supportmodel.SupportAttachment
	if err := r.db.Where("id = ?", strings.TrimSpace(id)).First(&item).Error; err != nil {
		return nil, supportAttachmentNotFound(err)
	}
	return &item, nil
}

func (r *SupportTicketRepository) UpdateTicketStatus(id, status, priority string, adminNotes *string, resolvedBy *string) error {
	updates := map[string]any{
		"status":      strings.TrimSpace(status),
		"admin_notes": adminNotes,
	}

	if p := strings.TrimSpace(priority); p != "" {
		updates["priority"] = p
	}

	normalized := strings.TrimSpace(status)
	if normalized == string(supportmodel.TicketStatusResolved) || normalized == string(supportmodel.TicketStatusClosed) {
		now := time.Now()
		updates["resolved_at"] = &now
		updates["resolved_by"] = resolvedBy
	} else {
		updates["resolved_at"] = nil
		updates["resolved_by"] = nil
	}

	if err := r.db.Model(&supportmodel.SupportTicket{}).Where("id = ?", strings.TrimSpace(id)).Updates(updates).Error; err != nil {
		return apperrors.ErrSupportTicketUpdateFailed.WithError(err)
	}

	return nil
}

func (r *SupportTicketRepository) CountTicketsByEmailToday(email string) (int64, error) {
	location := time.Now().Location()
	now := time.Now().In(location)
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location)
	end := start.Add(24 * time.Hour)

	var count int64
	err := r.db.Model(&supportmodel.SupportTicket{}).
		Where("LOWER(email) = LOWER(?)", strings.TrimSpace(email)).
		Where("created_at >= ? AND created_at < ?", start, end).
		Count(&count).Error
	if err != nil {
		return 0, apperrors.ErrSupportTicketCountFailed.WithError(err)
	}

	return count, nil
}

func (r *SupportTicketRepository) MarkTicketAsActiveByReply(ticketID string) error {
	normalizedID := strings.TrimSpace(ticketID)
	now := time.Now()

	updates := map[string]any{
		"updated_at": now,
	}

	var ticket supportmodel.SupportTicket
	if err := r.db.Where("id = ?", normalizedID).First(&ticket).Error; err != nil {
		return supportTicketNotFound(err)
	}

	if ticket.Status == string(supportmodel.TicketStatusResolved) || ticket.Status == string(supportmodel.TicketStatusClosed) {
		updates["status"] = string(supportmodel.TicketStatusInProgress)
		updates["resolved_at"] = nil
		updates["resolved_by"] = nil
	}

	if err := r.db.Model(&supportmodel.SupportTicket{}).Where("id = ?", normalizedID).Updates(updates).Error; err != nil {
		return apperrors.ErrSupportTicketUpdateFailed.WithError(err)
	}

	return nil
}

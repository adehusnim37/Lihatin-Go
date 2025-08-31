package shortlink

import (
	"math/rand"
	"time"

	"github.com/adehusnim37/lihatin-go/models/shortlink"
	"github.com/google/uuid"

	"gorm.io/gorm"
)

var rng *rand.Rand

func init() {
	rng = rand.New(rand.NewSource(time.Now().UnixNano()))
}

type ShortLinkRepository struct {
	db *gorm.DB
}

func NewShortLinkRepository(db *gorm.DB) *ShortLinkRepository {
	return &ShortLinkRepository{db: db}
}

func (r *ShortLinkRepository) CreateShortLink(link *shortlink.CreateShortLinkRequest) error {
	if err := r.db.Where("short_code = ?", link.CustomCode).First(&shortlink.ShortLink{}).Error; err == nil {
		return gorm.ErrDuplicatedKey
	}

	if link.CustomCode == "" {
		link.CustomCode = r.generateCustomCode(link.OriginalURL)
	}

	return r.db.Create(&shortlink.ShortLink{
		ID:          uuid.New().String(),
		UserID:      link.UserID,
		ShortCode:   link.CustomCode,
		OriginalURL: link.OriginalURL,
		Title:       link.Title,
		Description: link.Description,
		ExpiresAt:   link.ExpiresAt,
	}).Error
}

func (r *ShortLinkRepository) generateCustomCode(url string) string {
	if len(url) == 0 {
		// Generate a random code if URL is empty
		const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
		code := make([]byte, 6)
		for i := range code {
			code[i] = charset[rng.Intn(len(charset))]
		}
		return string(code)
	}

	// Use the first 4 characters of URL (or whatever is available)
	endIndex := len(url)
	if endIndex > 4 {
		endIndex = 4
	}

	return url[:endIndex]
}

// GetShortLinksByUserIDWithPagination gets short links with pagination
func (r *ShortLinkRepository) GetShortLinksByUserIDWithPagination(userID string, page, limit int) (*shortlink.PaginatedShortLinksResponse, error) {
	var links []shortlink.ShortLink
	var totalCount int64

	// Get total count
	if err := r.db.Model(&shortlink.ShortLink{}).Where("user_id = ?", userID).Count(&totalCount).Error; err != nil {
		return nil, err
	}

	// Calculate offset
	offset := (page - 1) * limit

	// Get paginated results
	if err := r.db.Where("user_id = ?", userID).Offset(offset).Limit(limit).Find(&links).Error; err != nil {
		return nil, err
	}

	// Convert ShortLink to ShortLinkResponse with pre-allocated capacity
	shortLinkResponses := make([]shortlink.ShortLinkResponse, 0, len(links))
	for _, link := range links {
		// Calculate click count from views using a separate query (more efficient for large datasets)
		var clickCount int64
		r.db.Model(&shortlink.ViewLinkDetail{}).Where("short_link_id = ?", link.ID).Count(&clickCount)

		shortLinkResponses = append(shortLinkResponses, shortlink.ShortLinkResponse{
			ID:          link.ID, // Now both are strings - consistent!
			UserID:      link.UserID,
			ShortCode:   link.ShortCode,
			OriginalURL: link.OriginalURL,
			Title:       link.Title,
			Description: link.Description,
			IsActive:    link.IsActive,
			ExpiresAt:   link.ExpiresAt,
			CreatedAt:   link.CreatedAt,
			UpdatedAt:   link.UpdatedAt,
			ClickCount:  int(clickCount), // Real click count from Views
		})
	}

	// Calculate total pages
	totalPages := int((totalCount + int64(limit) - 1) / int64(limit))

	response := &shortlink.PaginatedShortLinksResponse{
		ShortLinks: shortLinkResponses,
		TotalCount: totalCount,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}

	return response, nil
}

func (r *ShortLinkRepository) GetShortLinkByShortCode(code string) (*shortlink.ShortLink, error) {
	var link shortlink.ShortLink
	if err := r.db.Where("short_code = ?", code).First(&link).Error; err != nil {
		return nil, err
	}
	return &link, nil
}

func (r *ShortLinkRepository) GetAllShortLinks() ([]shortlink.ShortLink, error) {
	var links []shortlink.ShortLink
	if err := r.db.Find(&links).Error; err != nil {
		return nil, err
	}
	return links, nil
}

func (r *ShortLinkRepository) UpdateShortLinkByShortCode(code string, updates shortlink.UpdateShortLinkRequest) error {
	return r.db.Model(&shortlink.ShortLink{}).Where("short_code = ?", code).Updates(updates).Error
}

func (r *ShortLinkRepository) DeleteShortLinkByShortCode(code string) error {
	return r.db.Where("short_code = ?", code).Delete(&shortlink.ShortLink{}).Error
}

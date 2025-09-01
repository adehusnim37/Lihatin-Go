package shortlink

import (
	"errors"
	"math/rand"
	"time"

	"github.com/adehusnim37/lihatin-go/models/shortlink"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/adehusnim37/lihatin-go/dto"
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

func (r *ShortLinkRepository) CreateShortLink(link *dto.CreateShortLinkRequest) error {
	if err := r.db.Where("short_code = ?", link.CustomCode).First(&shortlink.ShortLink{}).Error; err == nil {
		return gorm.ErrDuplicatedKey
	}

	if link.CustomCode == "" {
		link.CustomCode = r.generateCustomCode(link.OriginalURL)
	}

	shortLink := shortlink.ShortLink{
		ID:          uuid.New().String(),
		UserID:      link.UserID,
		ShortCode:   link.CustomCode,
		OriginalURL: link.OriginalURL,
		Title:       link.Title,
		Description: link.Description,
		ExpiresAt:   link.ExpiresAt,
	}

	if err := r.db.Create(&shortLink).Error; err != nil {
		return err
	}

	shortLinkDetail := shortlink.ShortLinkDetail{
		ID:          uuid.New().String(), // Add missing ID generation
		ShortLinkID: shortLink.ID,
		Passcode:    utils.StringToInt(link.Passcode),
	}

	if err := r.db.Create(&shortLinkDetail).Error; err != nil {
		return err
	}

	return nil
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

// RedirectsByUserIDWithPagination gets short links with pagination
func (r *ShortLinkRepository) GetShortByUserIDWithPagination(userID string, page, limit int) (*shortlink.PaginatedShortLinksResponse, error) {
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

func (r *ShortLinkRepository) RedirectByShortCode(code string, ipAddress, userAgent, referer string, passcode int) (*shortlink.ShortLink, error) {
	var link shortlink.ShortLink
	// Find the short link by code with proper validation
	err := r.db.Where("short_code = ?", code).First(&link).Error
	if err != nil {
		return nil, err // Link not found
	}

	// Check if link is expired
	if link.ExpiresAt != nil && link.ExpiresAt.Before(time.Now()) {
		return nil, gorm.ErrRecordNotFound // Link expired
	}

	// Check if link is active
	if !link.IsActive {
		return nil, gorm.ErrRecordNotFound // Link inactive
	}

	// Load existing detail or create if doesn't exist
	var detail shortlink.ShortLinkDetail
	err = r.db.Where("short_link_id = ?", link.ID).First(&detail).Error
	if err == gorm.ErrRecordNotFound {
		// Create detail if doesn't exist
		detail = shortlink.ShortLinkDetail{
			ID:            uuid.New().String(),
			ShortLinkID:   link.ID,
			CurrentClicks: 0,
			EnableStats:   true,
			Passcode:      0, // No passcode by default
		}
		if err := r.db.Create(&detail).Error; err != nil {
			// Log error but continue
		}
	} else if err != nil {
		return nil, err
	}

	// Check if passcode is required and validate (only if passcode is set in database)
	if detail.Passcode != 0 {

		if detail.Passcode != passcode {
			// Passcode is incorrect or not provided - return error immediately
			// fmt.Printf("Passcode mismatch! Returning error.\n")
			return nil, errors.New("incorrect or missing passcode")
		}
		// fmt.Printf("Passcode matches! Continuing...\n")
		// Passcode is correct, continue with the function
	}
	// If passcode is 0 in database, no passcode validation needed

	// Check click limit if enabled
	if detail.ClickLimit > 0 && detail.CurrentClicks >= detail.ClickLimit {
		return nil, gorm.ErrRecordNotFound // Click limit reached
	}

	// Get location information from IP
	var country, city string
	if ipAddress != "" && ipAddress != "127.0.0.1" && ipAddress != "::1" {
		if locationResponse, err := utils.IPGeolocation(ipAddress); err == nil {
			country = locationResponse.Location.CountryName
			city = locationResponse.Location.City
		}
	}

	// Parse user agent for device, browser, OS info
	deviceInfo := utils.ParseUserAgent(userAgent)

	// Create view record to track this click
	viewRecord := shortlink.ViewLinkDetail{
		ID:          uuid.New().String(),
		ShortLinkID: link.ID,
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
		Referer:     referer,
		Country:     country,
		City:        city,
		Device:      deviceInfo.Device,
		Browser:     deviceInfo.Browser,
		OS:          deviceInfo.OS,
		ClickedAt:   time.Now(),
	}

	// Save view record
	if err := r.db.Create(&viewRecord).Error; err != nil {
		// Log error but don't fail the redirect
		// The user should still be able to access the link
	}

	// Update click count in detail
	r.db.Model(&detail).UpdateColumn("current_clicks", gorm.Expr("current_clicks + ?", 1))

	return &link, nil
}

func (r *ShortLinkRepository) GetAllShortLinks() ([]shortlink.ShortLink, error) {
	var links []shortlink.ShortLink
	if err := r.db.Find(&links).Error; err != nil {
		return nil, err
	}
	return links, nil
}

func (r *ShortLinkRepository) GetShortLinkByShortCode(code string) (*shortlink.ShortLink, error) {
	var link shortlink.ShortLink
	if err := r.db.Where("short_code = ?", code).First(&link).Error; err != nil {
		return nil, err
	}
	return &link, nil
}

func (r *ShortLinkRepository) UpdateShortLinkByShortCode(code string, updates shortlink.UpdateShortLinkRequest) error {
	return r.db.Model(&shortlink.ShortLink{}).Where("short_code = ?", code).Updates(updates).Error
}

func (r *ShortLinkRepository) DeleteShortLinkByShortCode(code string) error {
	return r.db.Where("short_code = ?", code).Delete(&shortlink.ShortLink{}).Error
}

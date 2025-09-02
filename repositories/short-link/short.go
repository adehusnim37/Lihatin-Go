package shortlink

import (
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/adehusnim37/lihatin-go/dto"
	shortlink "github.com/adehusnim37/lihatin-go/models/shortlink"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var localRng *rand.Rand

func init() {
	localRng = rand.New(rand.NewSource(time.Now().UnixNano()))
}

type ShortLinkRepository struct {
	db *gorm.DB
}

func NewShortLinkRepository(db *gorm.DB) *ShortLinkRepository {
	return &ShortLinkRepository{db: db}
}

func (r *ShortLinkRepository) CreateShortLink(link *dto.CreateShortLinkRequest) error {
	// Check for duplicate short code first
	if link.CustomCode != "" {
		if err := r.db.Where("short_code = ?", link.CustomCode).First(&shortlink.ShortLink{}).Error; err == nil {
			return gorm.ErrDuplicatedKey
		}
	}

	if link.CustomCode == "" {
		link.CustomCode = r.generateCustomCode(link.OriginalURL)
	}

	// Handle nullable UserID - convert string to *string for database
	var userIDPtr *string
	if link.UserID != "" {
		userIDPtr = &link.UserID
		utils.Logger.Info("Creating short link for authenticated user",
			"user_id", link.UserID,
			"short_code", link.CustomCode,
		)
	} else {
		utils.Logger.Info("Creating short link for anonymous user",
			"short_code", link.CustomCode,
		)
	}

	shortLink := shortlink.ShortLink{
		ID:          uuid.New().String(),
		UserID:      userIDPtr, // ✅ Use pointer for nullable field
		ShortCode:   link.CustomCode,
		OriginalURL: link.OriginalURL,
		Title:       link.Title,
		Description: link.Description,
		ExpiresAt:   link.ExpiresAt,
	}

	if err := r.db.Create(&shortLink).Error; err != nil {
		utils.Logger.Error("Failed to create short link", "error", err.Error())
		return err
	}

	shortLinkDetail := shortlink.ShortLinkDetail{
		ID:          uuid.New().String(),
		ShortLinkID: shortLink.ID,
		Passcode:    utils.StringToInt(link.Passcode),
	}

	if err := r.db.Create(&shortLinkDetail).Error; err != nil {
		utils.Logger.Error("Failed to create short link detail", "error", err.Error())
		return err
	}

	utils.Logger.Info("Short link created successfully",
		"id", shortLink.ID,
		"short_code", shortLink.ShortCode,
		"user_id", link.UserID,
	)

	return nil
}

func (r *ShortLinkRepository) generateCustomCode(url string) string {
	if len(url) == 0 {
		// Generate a random code if URL is empty
		const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
		code := make([]byte, 6)
		for i := range code {
			code[i] = charset[localRng.Intn(len(charset))]
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

// GetShortsByUserIDWithPagination gets short links with pagination and sorting
func (r *ShortLinkRepository) GetShortsByUserIDWithPagination(userID string, page, limit int, sort, orderBy string) (*dto.PaginatedShortLinksResponse, error) {
	var links []shortlink.ShortLink
	var totalCount int64

	// Get total count
	if err := r.db.Model(&shortlink.ShortLink{}).Where("user_id = ?", userID).Count(&totalCount).Error; err != nil {
		return nil, err
	}

	// Calculate offset
	offset := (page - 1) * limit

	// Build order clause
	orderClause := fmt.Sprintf("%s %s", sort, orderBy)

	// Get paginated results with sorting
	if err := r.db.Where("user_id = ?", userID).
		Order(orderClause).
		Offset(offset).
		Limit(limit).
		Find(&links).Error; err != nil {
		return nil, err
	}

	for _, link := range links {
		if link.UserID == nil || *link.UserID != userID {
			utils.Logger.Warn("Unauthorized access attempt to short link",
				"short_code", link.ShortCode,
				"requesting_user", userID,
				"owner_user", link.UserID,
			)
			return nil, utils.ErrShortLinkUnauthorized
		}
	}

	// Convert ShortLink to ShortsLinkResponse with pre-allocated capacity
	shortLinkResponses := make([]dto.ShortsLinkResponse, 0, len(links))
	for _, link := range links {
		// Calculate click count from views using a separate query (more efficient for large datasets)
		var clickCount int64
		r.db.Model(&shortlink.ViewLinkDetail{}).Where("short_link_id = ?", link.ID).Count(&clickCount)

		shortLinkResponses = append(shortLinkResponses, dto.ShortsLinkResponse{
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

	response := &dto.PaginatedShortLinksResponse{
		ShortLinks: shortLinkResponses,
		TotalCount: totalCount,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
		Sort:       sort,
		OrderBy:    orderBy,
	}

	return response, nil
}

func (r *ShortLinkRepository) RedirectByShortCode(code string, ipAddress, userAgent, referer string, passcode int) (*shortlink.ShortLink, error) {
	var link shortlink.ShortLink
	// Find the short link by code with proper validation
	err := r.db.Where("short_code = ?", code).First(&link).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.Logger.Warn("Short link not found",
				"short_code", code,
				"ip_address", ipAddress,
			)
			return nil, utils.ErrShortLinkNotFound
		}

		utils.Logger.Error("Database error while fetching short link",
			"short_code", code,
			"error", err.Error(),
		)
		return nil, err
	}

	// Check if short link is active
	if !link.IsActive {
		utils.Logger.Warn("Inactive short link accessed",
			"short_code", code,
			"ip_address", ipAddress,
		)
		return nil, utils.ErrShortLinkInactive
	}

	// Check if short link is expired
	if link.ExpiresAt != nil && link.ExpiresAt.Before(time.Now()) {
		utils.Logger.Warn("Expired short link accessed",
			"short_code", code,
			"expires_at", *link.ExpiresAt,
			"ip_address", ipAddress,
		)
		return nil, utils.ErrShortLinkExpired
	}

	// Check passcode if provided
	if passcode > 0 {
		var detail shortlink.ShortLinkDetail
		if err := r.db.Where("short_link_id = ?", link.ID).First(&detail).Error; err != nil {
			utils.Logger.Error("Failed to fetch short link detail",
				"short_code", code,
				"error", err.Error(),
			)
			return nil, err
		}

		if detail.Passcode != passcode {
			utils.Logger.Warn("Invalid passcode attempt",
				"short_code", code,
				"ip_address", ipAddress,
			)
			return nil, utils.ErrPasscodeIncorrect
		}

		if detail.ClickLimit > 0 && detail.CurrentClicks >= detail.ClickLimit {
			utils.Logger.Warn("Click limit reached",
				"short_code", code,
				"ip_address", ipAddress,
			)
			return nil, utils.ErrClickLimitReached
		}

		detail.CurrentClicks++
		detail.UpdatedAt = time.Now()
		if err := r.db.Save(&detail).Error; err != nil {
			utils.Logger.Error("Failed to update short link detail",
				"short_code", code,
				"ip_address", ipAddress,
				"error", err.Error(),
			)
			return nil, err
		}
	}

	// Track the click with basic info
	go func() {
		viewDetail := shortlink.ViewLinkDetail{
			ID:          uuid.New().String(),
			ShortLinkID: link.ID,
			IPAddress:   ipAddress,
			UserAgent:   userAgent,
			Referer:     referer,
			Country:     "Unknown", // TODO: Implement IP geolocation
			City:        "Unknown", // TODO: Implement IP geolocation
			ClickedAt:   time.Now(),
		}

		if err := r.db.Create(&viewDetail).Error; err != nil {
			utils.Logger.Error("Failed to track click",
				"short_code", code,
				"ip_address", ipAddress,
				"error", err.Error(),
			)
		}
	}()

	return &link, nil
}

func (r *ShortLinkRepository) GetShortLinkByShortCode(code string, userID string) (*dto.ShortLinkResponse, error) {
	var link shortlink.ShortLink
	var detail shortlink.ShortLinkDetail

	// Get the short link first
	err := r.db.Where("short_code = ?", code).First(&link).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, utils.ErrShortLinkNotFound
		}

		utils.Logger.Error("Database error while fetching short link",
			"short_code", code,
			"user_id", userID,
			"error", err.Error(),
		)
		return nil, err
	}

	// Check if the link belongs to the user (handle nullable UserID)
	if link.UserID == nil || *link.UserID != userID {
		utils.Logger.Warn("Unauthorized access attempt to short link",
			"short_code", code,
			"requesting_user", userID,
			"owner_user", link.UserID,
		)
		return nil, utils.ErrShortLinkUnauthorized
	}

	// Get the short link detail
	if err := r.db.Where("short_link_id = ?", link.ID).First(&detail).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			utils.Logger.Error("Failed to fetch short link detail", "error", err.Error())
		}
		// Continue even if detail not found - use empty detail
	}

	// Get recent views (last 10)
	var recentViews []shortlink.ViewLinkDetail
	r.db.Where("short_link_id = ?", link.ID).
		Order("clicked_at DESC").
		Limit(10).
		Find(&recentViews)


	// Convert recent views to response format
	recentViewsResponse := make([]dto.ViewLinkDetailResponse, 0, len(recentViews))
	for _, view := range recentViews {
		recentViewsResponse = append(recentViewsResponse, dto.ViewLinkDetailResponse{
			ID:          view.ID,
			IPAddress:   view.IPAddress,
			UserAgent:   view.UserAgent,
			Referer:     view.Referer,
			Country:     view.Country,
			City:        view.City,
			Device:      view.Device,
			Browser:     view.Browser,
			OS:          view.OS,
			ClickedAt:   view.ClickedAt,
		})
	}

	utils.Logger.Info("Short link retrieved successfully",
		"short_code", code,
		"user_id", userID,
	)

	return &dto.ShortLinkResponse{
		ID:           link.ID,
		UserID:       link.UserID,
		ShortCode:    link.ShortCode,
		OriginalURL:  link.OriginalURL,
		Title:        link.Title,
		Description:  link.Description,
		IsActive:     link.IsActive,
		ExpiresAt:    link.ExpiresAt,
		CreatedAt:    link.CreatedAt,
		UpdatedAt:    link.UpdatedAt,
		ShortLinkDetail: &dto.ShortLinkDetailsResponse{
			ID:            detail.ID,
			Passcode:      detail.Passcode,
			ClickLimit:    detail.ClickLimit,
			CurrentClicks: detail.CurrentClicks, // Real-time current clicks
			EnableStats:   detail.EnableStats,
			CustomDomain:  detail.CustomDomain,
			UTMSource:     detail.UTMSource,
			UTMMedium:     detail.UTMMedium,
			UTMCampaign:   detail.UTMCampaign,
			UTMTerm:       detail.UTMTerm,
			UTMContent:    detail.UTMContent,
		},
		RecentViews: recentViewsResponse,
	}, nil
}

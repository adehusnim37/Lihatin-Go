package shortlink

import (
	"encoding/base64"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/adehusnim37/lihatin-go/dto"
	shortlink "github.com/adehusnim37/lihatin-go/models/shortlink"
	apperrors "github.com/adehusnim37/lihatin-go/internal/pkg/errors"
	"github.com/adehusnim37/lihatin-go/internal/pkg/helpers"
	"github.com/adehusnim37/lihatin-go/internal/pkg/ip"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
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

func (r *ShortLinkRepository) CreateShortLink(link *dto.CreateShortLinkRequest) (*shortlink.ShortLink, *shortlink.ShortLinkDetail, error) {
	// Check for duplicate short code first
	if link.CustomCode != "" {
		if err := r.db.Where("short_code = ?", link.CustomCode).First(&shortlink.ShortLink{}).Error; err == nil {
			return nil, nil, gorm.ErrDuplicatedKey
		}
	}

	if link.CustomCode == "" {
		link.CustomCode = r.generateCustomCode(link.OriginalURL)
	}

	// Handle nullable UserID - convert string to *string for database
	var userIDPtr *string
	if link.UserID != "" {
		userIDPtr = &link.UserID
		logger.Logger.Info("Creating short link for authenticated user",
			"user_id", link.UserID,
			"short_code", link.CustomCode,
		)
	} else {
		logger.Logger.Info("Creating short link for anonymous user",
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
		logger.Logger.Error("Failed to create short link", "error", err.Error())
		return nil, nil, err
	}

	shortLinkDetail := shortlink.ShortLinkDetail{
		ID:          uuid.New().String(),
		ShortLinkID: shortLink.ID,
		Passcode:    helpers.StringToInt(link.Passcode),
	}

	if err := r.db.Create(&shortLinkDetail).Error; err != nil {
		logger.Logger.Error("Failed to create short link detail", "error", err.Error())
		return nil, nil, err
	}

	logger.Logger.Info("Short link created successfully",
		"id", shortLink.ID,
		"short_code", shortLink.ShortCode,
		"user_id", link.UserID,
	)

	return &shortLink, &shortLinkDetail, nil
}


// CreateBulkShortLinks creates multiple short links in a single transaction
func (r *ShortLinkRepository) CreateBulkShortLinks(links []dto.CreateShortLinkRequest) ([]shortlink.ShortLink, []shortlink.ShortLinkDetail, error) {
    if len(links) == 0 {
        return nil, nil, apperrors.ErrEmptyBulkLinksList
	}
    
    if len(links) > 15 { // Set reasonable limit
        return nil, nil, apperrors.ErrBulkCreateLimitExceeded
    }

    var createdLinks []shortlink.ShortLink
    var createdDetails []shortlink.ShortLinkDetail

    // Single transaction for all operations
    err := r.db.Transaction(func(tx *gorm.DB) error {
        for i, linkReq := range links {
            // Generate short code if not provided
            if linkReq.CustomCode == "" {
                linkReq.CustomCode = r.generateCustomCode(linkReq.OriginalURL)
            }

            // Check for duplicate codes within batch
            for j := range i {
                if links[j].CustomCode == linkReq.CustomCode {
                    return apperrors.ErrDuplicateShortCodeInBatch
                }
            }

            // Check existing codes in database
            var existingLink shortlink.ShortLink
            if err := tx.Where("short_code = ?", linkReq.CustomCode).First(&existingLink).Error; err == nil {
                return apperrors.ErrDuplicateShortCode
            }

            // Handle nullable UserID
            var userIDPtr *string
            if linkReq.UserID != "" {
                userIDPtr = &linkReq.UserID
            }

            // Create ShortLink
            shortLink := shortlink.ShortLink{
                ID:          uuid.New().String(),
                UserID:      userIDPtr,
                ShortCode:   linkReq.CustomCode,
                OriginalURL: linkReq.OriginalURL,
                Title:       linkReq.Title,
                Description: linkReq.Description,
                ExpiresAt:   linkReq.ExpiresAt,
            }

            if err := tx.Create(&shortLink).Error; err != nil {
                return apperrors.ErrShortCreatedFailed
            }
            createdLinks = append(createdLinks, shortLink)

            // Create ShortLinkDetail
            shortLinkDetail := shortlink.ShortLinkDetail{
                ID:          uuid.New().String(),
                ShortLinkID: shortLink.ID,
                Passcode:    helpers.StringToInt(linkReq.Passcode),
            }

            if err := tx.Create(&shortLinkDetail).Error; err != nil {
                return apperrors.ErrShortDetailCreatedFailed
            }
            createdDetails = append(createdDetails, shortLinkDetail)
        }

        return nil
    })

    if err != nil {
        logger.Logger.Error("Failed to create bulk short links", "error", err.Error())
        return nil, nil, err
    }

    logger.Logger.Info("Bulk short links created successfully",
        "count", len(createdLinks),
        "user_id", links[0].UserID,
    )

    return createdLinks, createdDetails, nil
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

	// Encode the URL to base64 and take 4 random characters from the encoded string
	encodedString := base64.RawURLEncoding.EncodeToString([]byte(url))
	code := make([]byte, 4)
	for i := range code {
		code[i] = encodedString[localRng.Intn(len(encodedString))]
		logger.Logger.Info("Generated character", "char", string(code[i]))
	}
	return string(code)
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
			logger.Logger.Warn("Unauthorized access attempt to short link",
				"short_code", link.ShortCode,
				"requesting_user", userID,
				"owner_user", link.UserID,
			)
			return nil, apperrors.ErrShortLinkUnauthorized
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
			logger.Logger.Warn("Short link not found",
				"short_code", code,
				"ip_address", ipAddress,
			)
			return nil, apperrors.ErrShortLinkNotFound
		}

		logger.Logger.Error("Database error while fetching short link",
			"short_code", code,
			"error", err.Error(),
		)
		return nil, err
	}

	if link.DeletedAt.Valid {
		logger.Logger.Warn("Deleted short link accessed",
			"short_code", code,
			"ip_address", ipAddress,
		)
		return nil, apperrors.ErrShortLinkAlreadyDeleted
	}

	// Check if short link is active
	if !link.IsActive {
		logger.Logger.Warn("Inactive short link accessed",
			"short_code", code,
			"ip_address", ipAddress,
		)
		return nil, apperrors.ErrShortLinkInactive
	}

	// Check if short link is expired
	if link.ExpiresAt != nil && link.ExpiresAt.Before(time.Now()) {
		logger.Logger.Warn("Expired short link accessed",
			"short_code", code,
			"expires_at", *link.ExpiresAt,
			"ip_address", ipAddress,
		)
		return nil, apperrors.ErrShortLinkExpired
	}

	// Check passcode if provided
	var detail shortlink.ShortLinkDetail
	if err := r.db.Where("short_link_id = ?", link.ID).First(&detail).Error; err != nil {
		logger.Logger.Error("Failed to fetch short link detail",
			"short_code", code,
			"error", err.Error(),
		)
		return nil, err
	}

	// Passcode checks
	if detail.Passcode != 0 && passcode == 0 {
		logger.Logger.Warn("Passcode required but not provided",
			"short_code", code,
			"ip_address", ipAddress,
		)
		return nil, apperrors.ErrPasscodeRequired
	}

	if detail.Passcode != 0 && passcode != detail.Passcode {
		logger.Logger.Warn("Invalid passcode attempt",
			"short_code", code,
			"ip_address", ipAddress,
		)
		return nil, apperrors.ErrPasscodeIncorrect
	}

	if detail.IsBanned {
		logger.Logger.Warn("Banned short link accessed",
			"short_code", code,
			"ip_address", ipAddress,
			"banned_reason", detail.BannedReason,
		)

		return nil, fmt.Errorf("%w: %s", apperrors.ErrLinkIsBanned, detail.BannedReason)
	}

	if detail.ClickLimit > 0 && detail.CurrentClicks >= detail.ClickLimit {
		logger.Logger.Warn("Click limit reached",
			"short_code", code,
			"ip_address", ipAddress,
		)
		return nil, apperrors.ErrClickLimitReached
	}

	detail.CurrentClicks++
	detail.UpdatedAt = time.Now()
	if err := r.db.Save(&detail).Error; err != nil {
		logger.Logger.Error("Failed to update short link detail",
			"short_code", code,
			"ip_address", ipAddress,
			"error", err.Error(),
		)
		return nil, err
	}

	// Track the click with basic info
	go func() {
		viewDetail := shortlink.ViewLinkDetail{
			ID:          uuid.New().String(),
			ShortLinkID: link.ID,
			IPAddress:   ipAddress,
			UserAgent:   userAgent,
			Referer:     referer,
			Country:     ip.GetCountryName(ipAddress), // Placeholder function
			City:        ip.GetCityName(ipAddress),    // Placeholder function
			ClickedAt:   time.Now(),
		}

		if err := r.db.Create(&viewDetail).Error; err != nil {
			logger.Logger.Error("Failed to track click",
				"short_code", code,
				"ip_address", ipAddress,
				"error", err.Error(),
			)
		}
	}()

	return &link, nil
}

func (r *ShortLinkRepository) GetShortLink(code string, userID string, userRole string) (*dto.ShortLinkResponse, error) {
	var link shortlink.ShortLink
	var detail shortlink.ShortLinkDetail
	var recentViews []shortlink.ViewLinkDetail

	// Fetch short link based on role
	var err error
	if userRole != "admin" {
		err = r.db.Where("short_code = ? AND user_id = ?", code, userID).First(&link).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, apperrors.ErrShortLinkNotFound
			}
			logger.Logger.Error("Database error while fetching short link",
				"short_code", code,
				"error", err.Error(),
			)
			return nil, err
		}
		// Check ownership
		if link.UserID == nil || *link.UserID != userID {
			logger.Logger.Warn("Unauthorized access attempt to short link",
				"short_code", code,
				"requesting_user", userID,
				"owner_user", link.UserID,
			)
			return nil, apperrors.ErrShortLinkUnauthorized
		}
	} else {
		err = r.db.Where("short_code = ?", code).First(&link).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, apperrors.ErrShortLinkNotFound
			}
			logger.Logger.Error("Database error while fetching short link",
				"short_code", code,
				"error", err.Error(),
			)
			return nil, err
		}
	}

	// Fetch detail
	err = r.db.Where("short_link_id = ?", link.ID).First(&detail).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Logger.Error("Short link detail not found",
				"short_code", code,
				"short_link_id", link.ID,
			)
			return nil, apperrors.ErrShortLinkNotFound
		}
		logger.Logger.Error("Database error while fetching short link detail",
			"short_code", code,
			"error", err.Error(),
		)
		return nil, err
	}

	// Fetch recent views if stats enabled
	if detail.EnableStats {
		err = r.db.Where("short_link_id = ?", link.ID).
			Order("clicked_at DESC").
			Limit(10).
			Find(&recentViews).Error
		if err != nil {
			logger.Logger.Error("Database error while fetching recent views",
				"short_code", code,
				"error", err.Error(),
			)
			return nil, err
		}
	}

	// Convert views to response format
	viewsResponse := make([]dto.ViewLinkDetailResponse, 0, len(recentViews))
	for _, view := range recentViews {
		viewsResponse = append(viewsResponse, dto.ViewLinkDetailResponse{
			ID:        view.ID,
			IPAddress: view.IPAddress,
			UserAgent: view.UserAgent,
			Referer:   view.Referer,
			Country:   view.Country,
			City:      view.City,
			Device:    view.Device,
			Browser:   view.Browser,
			OS:        view.OS,
			ClickedAt: view.ClickedAt,
		})
	}

	// Build detail response
	detailResponse := &dto.ShortLinkDetailsResponse{
		ID:            detail.ID,
		Passcode:      detail.Passcode,
		ClickLimit:    detail.ClickLimit,
		CurrentClicks: detail.CurrentClicks,
		EnableStats:   detail.EnableStats,
		CustomDomain:  detail.CustomDomain,
		UTMSource:     detail.UTMSource,
		UTMMedium:     detail.UTMMedium,
		UTMCampaign:   detail.UTMCampaign,
		UTMTerm:       detail.UTMTerm,
		UTMContent:    detail.UTMContent,
	}

	// Build main response
	shortLinkResponse := &dto.ShortLinkResponse{
		ID:              link.ID,
		UserID:          link.UserID,
		ShortCode:       link.ShortCode,
		OriginalURL:     link.OriginalURL,
		Title:           link.Title,
		Description:     link.Description,
		IsActive:        link.IsActive,
		ExpiresAt:       link.ExpiresAt,
		CreatedAt:       link.CreatedAt,
		UpdatedAt:       link.UpdatedAt,
		ShortLinkDetail: detailResponse,
		RecentViews:     viewsResponse,
	}

	return shortLinkResponse, nil
}

func (r *ShortLinkRepository) GetStatsShortLink(code string, userId string, userRole string) (*dto.ShortLinkWithStatsResponse, error) {
	var link shortlink.ShortLink
	var totalCount int64
	var uniqueVisitors int64
	var countries []dto.Country
	var devices []dto.TopDevice
	var referrers []dto.TopReferrer
	var last24hCount int64
	var last7dCount int64
	var last30dCount int64

	if userRole != "admin" {
		err := r.db.Where("short_code = ? AND user_id = ?", code, userId).First(&link).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, apperrors.ErrShortLinkNotFound
			}
			logger.Logger.Error("Database error while fetching short link",
				"short_code", code,
				"error", err.Error(),
			)
			return nil, err
		}
	} else {
		err := r.db.Where("short_code = ?", code).First(&link).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, apperrors.ErrShortLinkNotFound
			}
			logger.Logger.Error("Database error while fetching short link",
				"short_code", code,
				"error", err.Error(),
			)
			return nil, err
		}
	}

	// Get total views
	if err := r.db.Model(&shortlink.ViewLinkDetail{}).Where("short_link_id = ?", link.ID).Count(&totalCount).Error; err != nil {
		return nil, err
	}

	// Get unique visitors based on distinct IP addresses
	if err := r.db.Model(&shortlink.ViewLinkDetail{}).Where("short_link_id = ?", link.ID).Distinct("ip_address").Count(&uniqueVisitors).Error; err != nil {
		return nil, err
	}

	// Get top Countries by views
	// This is a simplified example; in a real scenario, you might want to limit the number of results
	if err := r.db.Model(&shortlink.ViewLinkDetail{}).
		Select("country, COUNT(*) as count").
		Where("short_link_id = ?", link.ID).
		Group("country").
		Order("count DESC").
		Limit(5).
		Scan(&countries).Error; err != nil {
		return nil, err
	}

	if err := r.db.Model(&shortlink.ViewLinkDetail{}).
		Select("device, COUNT(*) as count").
		Where("short_link_id = ?", link.ID).
		Group("device").
		Order("count DESC").
		Limit(5).
		Scan(&devices).Error; err != nil {
		return nil, err
	}

	if err := r.db.Model(&shortlink.ViewLinkDetail{}).
		Select("referer, COUNT(*) as count").
		Where("short_link_id = ?", link.ID).
		Group("referer").
		Order("count DESC").
		Limit(5).
		Scan(&referrers).Error; err != nil {
		return nil, err
	}

	if err := r.db.Model(&shortlink.ViewLinkDetail{}).
		Where("short_link_id = ? AND clicked_at >= ?", link.ID, time.Now().Add(-24*time.Hour)).
		Count(&last24hCount).Error; err != nil {
		return nil, err
	}

	if err := r.db.Model(&shortlink.ViewLinkDetail{}).
		Where("short_link_id = ? AND clicked_at >= ?", link.ID, time.Now().Add(-7*24*time.Hour)).
		Count(&last7dCount).Error; err != nil {
		return nil, err
	}

	if err := r.db.Model(&shortlink.ViewLinkDetail{}).
		Where("short_link_id = ? AND clicked_at >= ?", link.ID, time.Now().Add(-30*24*time.Hour)).
		Count(&last30dCount).Error; err != nil {
		return nil, err
	}

	return &dto.ShortLinkWithStatsResponse{
		ShortCode:      link.ShortCode,
		TotalClicks:    int(totalCount),
		UniqueVisitors: int(uniqueVisitors),
		Last24h:        int(last24hCount),
		Last7d:         int(last7dCount),
		Last30d:        int(last30dCount),
		TopReferrers:   referrers,
		TopDevices:     devices,
		TopCountries:   countries,
	}, nil
}

func (r *ShortLinkRepository) GetStatsAllShortLinks(userId string, userRole string, page, limit int, sort, orderBy string) (*dto.PaginatedShortLinkWithStatsResponse, error) {
	var links []shortlink.ShortLink

	offset := (page - 1) * limit
	orderClause := fmt.Sprintf("%s %s", sort, orderBy)

	if userRole != "admin" {
		err := r.db.Where("user_id = ?", userId).Order(orderClause).Offset(offset).Limit(limit).Find(&links).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, apperrors.ErrShortLinkNotFound
			}
			logger.Logger.Error("Database error while fetching short links",
				"user_id", userId,
				"error", err.Error(),
			)
			return nil, err
		}
	} else {
		err := r.db.Order(orderClause).Offset(offset).Limit(limit).Find(&links).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, apperrors.ErrShortLinkNotFound
			}
			logger.Logger.Error("Database error while fetching short links",
				"error", err.Error(),
			)
			return nil, err
		}
	}

	shortLinksStats := make([]dto.ShortLinkWithStatsResponse, 0, len(links))
	for _, link := range links {
		var totalCount int64
		var uniqueVisitors int64
		var countries []dto.Country
		var devices []dto.TopDevice
		var referrers []dto.TopReferrer
		var last24hCount int64
		var last7dCount int64
		var last30dCount int64

		r.db.Model(&shortlink.ViewLinkDetail{}).Where("short_link_id = ?", link.ID).Count(&totalCount)
		r.db.Model(&shortlink.ViewLinkDetail{}).Where("short_link_id = ?", link.ID).Distinct("ip_address").Count(&uniqueVisitors)
		r.db.Model(&shortlink.ViewLinkDetail{}).Select("country, COUNT(*) as count").Where("short_link_id = ?", link.ID).Group("country").Order("count DESC").Limit(5).Scan(&countries)
		r.db.Model(&shortlink.ViewLinkDetail{}).Select("device, COUNT(*) as count").Where("short_link_id = ?", link.ID).Group("device").Order("count DESC").Limit(5).Scan(&devices)
		r.db.Model(&shortlink.ViewLinkDetail{}).Select("referer, COUNT(*) as count").Where("short_link_id = ?", link.ID).Group("referer").Order("count DESC").Limit(5).Scan(&referrers)
		r.db.Model(&shortlink.ViewLinkDetail{}).Where("short_link_id = ? AND clicked_at >= ?", link.ID, time.Now().Add(-24*time.Hour)).Count(&last24hCount)
		r.db.Model(&shortlink.ViewLinkDetail{}).Where("short_link_id = ? AND clicked_at >= ?", link.ID, time.Now().Add(-7*24*time.Hour)).Count(&last7dCount)
		r.db.Model(&shortlink.ViewLinkDetail{}).Where("short_link_id = ? AND clicked_at >= ?", link.ID, time.Now().Add(-30*24*time.Hour)).Count(&last30dCount)

		shortLinksStats = append(shortLinksStats, dto.ShortLinkWithStatsResponse{
			ShortCode:      link.ShortCode,
			TotalClicks:    int(totalCount),
			UniqueVisitors: int(uniqueVisitors),
			Last24h:        int(last24hCount),
			Last7d:         int(last7dCount),
			Last30d:        int(last30dCount),
			TopReferrers:   referrers,
			TopDevices:     devices,
			TopCountries:   countries,
		})
	}

	// Get total count of all short links for pagination
	var totalLinks int64
	if userRole != "admin" {
		r.db.Model(&shortlink.ShortLink{}).Where("user_id = ?", userId).Count(&totalLinks)
	} else {
		r.db.Model(&shortlink.ShortLink{}).Count(&totalLinks)
	}

	totalPages := int((totalLinks + int64(limit) - 1) / int64(limit))

	return &dto.PaginatedShortLinkWithStatsResponse{
		ShortLinks: shortLinksStats,
		TotalCount: totalLinks,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
		Sort:       sort,
		OrderBy:    orderBy,
	}, nil
}

// GetShortLinkViewsPaginated gets paginated views for a specific short link
func (r *ShortLinkRepository) GetShortLinkViewsPaginated(code string, userID string, page, limit int, sort, orderBy string, userRole string) (*dto.PaginatedShortLinkDetailWithStatsResponse, error) {
	var link shortlink.ShortLink
	var detail shortlink.ShortLinkDetail
	var viewDetails []shortlink.ViewLinkDetail
	var totalCount int64

	// Validate the short link exists and user has access
	if userRole != "admin" {
		err := r.db.Where("short_code = ? AND user_id = ?", code, userID).First(&link).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, apperrors.ErrShortLinkNotFound
			}
			logger.Logger.Error("Database error while fetching short link",
				"short_code", code,
				"error", err.Error(),
			)
			return nil, err
		}
	} else {
		err := r.db.Where("short_code = ?", code).First(&link).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, apperrors.ErrShortLinkNotFound
			}
			logger.Logger.Error("Database error while fetching short link",
				"short_code", code,
				"error", err.Error(),
			)
			return nil, err
		}
	}

	// Get the short link detail
	err := r.db.Where("short_link_id = ?", link.ID).First(&detail).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Logger.Error("Short link detail not found",
				"short_code", code,
				"short_link_id", link.ID,
			)
			return nil, apperrors.ErrShortLinkNotFound
		}
		logger.Logger.Error("Database error while fetching short link detail",
			"short_code", code,
			"error", err.Error(),
		)
		return nil, err
	}

	// Get total count of views
	if err := r.db.Model(&shortlink.ViewLinkDetail{}).Where("short_link_id = ?", link.ID).Count(&totalCount).Error; err != nil {
		return nil, err
	}

	// Calculate offset
	offset := (page - 1) * limit

	// Build order clause - default to clicked_at DESC if not specified
	if sort == "" {
		sort = "clicked_at"
	}
	if orderBy == "" {
		orderBy = "desc"
	}
	orderClause := fmt.Sprintf("%s %s", sort, orderBy)

	// Get paginated views
	err = r.db.Where("short_link_id = ?", link.ID).
		Order(orderClause).
		Offset(offset).
		Limit(limit).
		Find(&viewDetails).Error
	if err != nil {
		logger.Logger.Error("Database error while fetching paginated views",
			"short_code", code,
			"error", err.Error(),
		)
		return nil, err
	}

	// Convert views to response format
	viewsResponse := make([]dto.ViewLinkDetailResponse, 0, len(viewDetails))
	for _, view := range viewDetails {
		viewsResponse = append(viewsResponse, dto.ViewLinkDetailResponse{
			ID:        view.ID,
			IPAddress: view.IPAddress,
			UserAgent: view.UserAgent,
			Referer:   view.Referer,
			Country:   view.Country,
			City:      view.City,
			Device:    view.Device,
			Browser:   view.Browser,
			OS:        view.OS,
			ClickedAt: view.ClickedAt,
		})
	}

	// Calculate total pages
	totalPages := int((totalCount + int64(limit) - 1) / int64(limit))

	// Build the short link response
	shortLinkResponse := dto.ShortLinkResponse{
		ID:          link.ID,
		UserID:      link.UserID,
		ShortCode:   link.ShortCode,
		OriginalURL: link.OriginalURL,
		Title:       link.Title,
		Description: link.Description,
		IsActive:    link.IsActive,
		ExpiresAt:   link.ExpiresAt,
		CreatedAt:   link.CreatedAt,
		UpdatedAt:   link.UpdatedAt,
	}

	// Build the detail response
	detailResponse := dto.ShortLinkDetailsResponse{
		ID:            detail.ID,
		Passcode:      detail.Passcode,
		ClickLimit:    detail.ClickLimit,
		CurrentClicks: detail.CurrentClicks,
		EnableStats:   detail.EnableStats,
		CustomDomain:  detail.CustomDomain,
		UTMSource:     detail.UTMSource,
		UTMMedium:     detail.UTMMedium,
		UTMCampaign:   detail.UTMCampaign,
		UTMTerm:       detail.UTMTerm,
		UTMContent:    detail.UTMContent,
	}

	// Build the paginated views response
	paginatedViews := dto.PaginatedViewLinkDetailResponse{
		Views:      viewsResponse,
		TotalCount: totalCount,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
		Sort:       sort,
		OrderBy:    orderBy,
	}

	// Build the complete response
	return &dto.PaginatedShortLinkDetailWithStatsResponse{
		ShortLinks:      []dto.ShortLinkResponse{shortLinkResponse},
		ShortLinkDetail: detailResponse,
		Views:           paginatedViews,
	}, nil
}

func (r *ShortLinkRepository) CheckShortCode(code *dto.CodeRequest) (bool, error) {
	var link shortlink.ShortLink

	err := r.db.Where("short_code = ?", code.Code).First(&link).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil // Code does not exist
		}
		logger.Logger.Error("Database error while checking short code",
			"short_code", code.Code,
			"error", err.Error(),
		)
		return false, err // Some other database error
	}

	return true, nil // Code exists
}

func (r *ShortLinkRepository) UpdateShortLink(code, userID, userRole string, in *dto.UpdateShortLinkRequest) error {
	tx := r.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 1) Ambil link sekali
	var link shortlink.ShortLink
	q := tx.Where("short_code = ?", code)
	if userRole != "admin" {
		q = q.Where("user_id = ?", userID)
	}
	if err := q.First(&link).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperrors.ErrShortLinkNotFound
		}
		return err
	}

	// 2) Bangun map updates agar hanya kolom yang berubah yang di-update
	linkUpd := map[string]any{}
	if in.Title != nil {
		linkUpd["title"] = *in.Title
	}
	if in.Description != nil {
		linkUpd["description"] = *in.Description
	}
	if in.IsActive != nil {
		linkUpd["is_active"] = *in.IsActive
	}
	if in.ExpiresAt != nil {
		linkUpd["expires_at"] = in.ExpiresAt
	}

	if len(linkUpd) > 0 {
		if err := tx.Model(&shortlink.ShortLink{}).
			Where("id = ?", link.ID).
			Updates(linkUpd).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	detailUpd := map[string]any{}
	if in.Passcode != nil {
		detailUpd["passcode"] = helpers.StringToInt(*in.Passcode)
	}
	if in.ClickLimit != nil {
		detailUpd["click_limit"] = *in.ClickLimit
	}
	if in.EnableStats != nil {
		detailUpd["enable_stats"] = *in.EnableStats
	}
	if in.CustomDomain != nil {
		detailUpd["custom_domain"] = *in.CustomDomain
	}

	if len(detailUpd) > 0 {
		if err := tx.Model(&shortlink.ShortLinkDetail{}).
			Where("short_link_id = ?", link.ID).
			Updates(detailUpd).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

func (r *ShortLinkRepository) DeleteShortLink(code string, userID string, passcode int, roleUser string) error {
	var link shortlink.ShortLink

	if roleUser != "admin" {
		err := r.db.Where("short_links.short_code = ? AND short_links.user_id = ?", code, userID).
			Joins("LEFT JOIN short_link_details ON short_links.id = short_link_details.short_link_id").
			Select("short_links.*, short_link_details.passcode").
			First(&link).Error

		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apperrors.ErrShortLinkNotFound
			}

			logger.Logger.Error("Database error while fetching short link",
				"short_code", code,
				"error", err.Error(),
			)
			return err
		}

		// Validate passcode if set
		if link.Detail.Passcode != 0 && link.Detail.Passcode != passcode {
			return apperrors.ErrPasscodeIncorrect
		}
	} else {
		err := r.db.Where("short_code = ?", code).
			First(&link).Error

		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apperrors.ErrShortLinkNotFound
			}

			logger.Logger.Error("Database error while fetching short link",
				"short_code", code,
				"error", err.Error(),
			)
			return err
		}
	}

	if link.DeletedAt.Valid {
		return apperrors.ErrShortLinkAlreadyDeleted
	}

	if err := r.db.Where("short_code = ?", link.ShortCode).Delete(&link).Error; err != nil {
		logger.Logger.Error("Failed to delete short link",
			"short_code", link.ShortCode,
			"error", err.Error(),
		)
		return err
	}

	return nil
}

func (r *ShortLinkRepository) DeleteShortsLink(req *dto.BulkDeleteRequest) error {
	var links []shortlink.ShortLink

	if len(req.Codes) == 0 {
		return apperrors.ErrEmptyCodesList
	}

	// perform a check to see if all codes exist
	if err := r.db.Where("short_code IN ?", req.Codes).Find(&links).Error; err != nil {
		logger.Logger.Error("Failed to fetch short links for bulk delete",
			"short_codes", req.Codes,
			"error", err.Error(),
		)
		return err
	}

	if len(req.Codes) != len(links) {
		return apperrors.ErrSomeShortLinksNotFound
	}
	// Perform bulk delete
	if err := r.db.Where("short_code IN ?", req.Codes).Delete(&shortlink.ShortLink{}).Error; err != nil {
		logger.Logger.Error("Failed to delete short links",
			"short_codes", req.Codes,
			"error", err.Error(),
		)
		return err
	}

	return nil
}

func (r *ShortLinkRepository) ListAllShortLinks(page, limit int, sort, orderBy string) (*dto.PaginatedShortLinksAdminResponse, error) {
	var shortLinks []shortlink.ShortLink
	var totalCount int64

	logger.Logger.Info("Fetching all short links for admin",
		"page", page,
		"limit", limit,
		"sort", sort,
		"order_by", orderBy,
	)

	// Get total count (without joins for better performance)
	if err := r.db.Model(&shortlink.ShortLink{}).Count(&totalCount).Error; err != nil {
		logger.Logger.Error("Failed to count short links", "error", err.Error())
		return nil, err
	}

	// Calculate offset and build order clause
	offset := (page - 1) * limit
	orderClause := sort + " " + orderBy

	// Query with LEFT JOINs to get all data including details and view counts
	if err := r.db.
		Preload("Detail"). // Load detail relationship
		Preload("Views").  // Load views relationship for click counts
		Order(orderClause).
		Limit(limit).
		Offset(offset).
		Find(&shortLinks).Error; err != nil {
		logger.Logger.Error("Failed to fetch short links", "error", err.Error())
		return nil, err
	}

	// Convert to response format with detailed information
	shortLinkResponses := make([]dto.ShortLinkResponse, 0, len(shortLinks))
	for _, link := range shortLinks {
		// Count total clicks for this short link
		var clickCount int64
		r.db.Model(&shortlink.ViewLinkDetail{}).
			Where("short_link_id = ?", link.ID).
			Count(&clickCount)

		// Build detail response
		var detailResponse *dto.ShortLinkDetailsResponse
		if link.Detail != nil {
			detailResponse = &dto.ShortLinkDetailsResponse{
				ID:            link.Detail.ID,
				Passcode:      link.Detail.Passcode,
				ClickLimit:    link.Detail.ClickLimit,
				CurrentClicks: link.Detail.CurrentClicks,
				EnableStats:   link.Detail.EnableStats,
				CustomDomain:  link.Detail.CustomDomain,
				UTMSource:     link.Detail.UTMSource,
				UTMMedium:     link.Detail.UTMMedium,
				UTMCampaign:   link.Detail.UTMCampaign,
				UTMTerm:       link.Detail.UTMTerm,
				UTMContent:    link.Detail.UTMContent,
			}
		}

		// Build main response
		shortLinkResponse := dto.ShortLinkResponse{
			ID:              link.ID,
			UserID:          link.UserID,
			ShortCode:       link.ShortCode,
			OriginalURL:     link.OriginalURL,
			Title:           link.Title,
			Description:     link.Description,
			IsActive:        link.IsActive,
			ExpiresAt:       link.ExpiresAt,
			CreatedAt:       link.CreatedAt,
			UpdatedAt:       link.UpdatedAt,
			ShortLinkDetail: detailResponse,
		}

		shortLinkResponses = append(shortLinkResponses, shortLinkResponse)
	}

	// Calculate total pages
	totalPages := int((totalCount + int64(limit) - 1) / int64(limit))

	logger.Logger.Info("Successfully fetched short links for admin",
		"total_count", totalCount,
		"returned_count", len(shortLinkResponses),
		"total_pages", totalPages,
	)

	return &dto.PaginatedShortLinksAdminResponse{
		ShortLinks: shortLinkResponses,
		TotalCount: totalCount,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
		Sort:       sort,
		OrderBy:    orderBy,
	}, nil
}

func (r *ShortLinkRepository) BannedShortByAdmin(request *dto.BannedRequest, userID string, code *dto.CodeRequest) error {
	var link shortlink.ShortLink
	var detail shortlink.ShortLinkDetail

	err := r.db.Where("short_code = ?", code.Code).First(&link).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperrors.ErrShortLinkNotFound
		}

		logger.Logger.Error("Database error while fetching short link",
			"short_code", code.Code,
			"error", err.Error(),
		)
		return err
	}

	link.IsActive = false

	err = r.db.Where("short_link_id = ?", link.ID).First(&detail).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperrors.ErrShortLinkNotFound
		}

		logger.Logger.Error("Database error while fetching short link detail",
			"short_link_id", link.ID,
			"error", err.Error(),
		)
		return err
	}

	detail.EnableStats = false
	detail.IsBanned = true
	detail.BannedBy = &userID
	detail.BannedReason = request.Reason

	if err := r.db.Save(&link).Error; err != nil {
		logger.Logger.Error("Failed to update short link",
			"short_code", code,
			"error", err.Error(),
		)
		return err
	}

	if err := r.db.Save(&detail).Error; err != nil {
		logger.Logger.Error("Failed to update short link detail",
			"short_link_id", link.ID,
			"error", err.Error(),
		)
		return err
	}

	return nil
}

func (r *ShortLinkRepository) RestoreShortByAdmin(code string) error {
	var link shortlink.ShortLink
	var detail shortlink.ShortLinkDetail

	err := r.db.Where("short_code = ?", code).First(&link).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperrors.ErrShortLinkNotFound
		}

		logger.Logger.Error("Database error while fetching short link",
			"short_code", code,
			"error", err.Error(),
		)
		return err
	}

	link.IsActive = true

	err = r.db.Where("short_link_id = ?", link.ID).First(&detail).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperrors.ErrShortLinkNotFound
		}

		logger.Logger.Error("Database error while fetching short link detail",
			"short_link_id", link.ID,
			"error", err.Error(),
		)
		return err
	}

	detail.IsBanned = false
	detail.BannedBy = nil
	detail.BannedReason = ""

	if err := r.db.Save(&link).Error; err != nil {
		logger.Logger.Error("Failed to update short link",
			"short_code", code,
			"error", err.Error(),
		)
		return err
	}

	if err := r.db.Save(&detail).Error; err != nil {
		logger.Logger.Error("Failed to update short link detail",
			"short_link_id", link.ID,
			"error", err.Error(),
		)
		return err
	}

	return nil
}

func (r *ShortLinkRepository) RestoreDeletedShortByAdmin(code string) error {
	var link shortlink.ShortLink

	err := r.db.Unscoped().Where("short_code = ?", code).First(&link).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperrors.ErrShortLinkNotFound
		}

		logger.Logger.Error("Database error while fetching short link",
			"short_code", code,
			"error", err.Error(),
		)
		return err
	}

	if !link.DeletedAt.Valid {
		return apperrors.ErrShortIsNotDeleted
	}

	link.DeletedAt = gorm.DeletedAt{Valid: false}

	if err := r.db.Unscoped().Save(&link).Error; err != nil {
		logger.Logger.Error("Failed to restore deleted short link",
			"short_code", code,
			"error", err.Error(),
		)
		return err
	}

	return nil
}

func (r *ShortLinkRepository) ResetPasscodeShortLink(code string, oldPasscode, newPasscode int, userID, roleUser string) error {
	var link shortlink.ShortLink

	if roleUser != "admin" {
		err := r.db.Where("short_links.short_code = ? AND short_links.user_id = ?", code, userID).
			Joins("LEFT JOIN short_link_details ON short_links.id = short_link_details.short_link_id").
			Select("short_links.*, short_link_details.passcode, short_link_details.id as detail_id").
			First(&link).Error

		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apperrors.ErrShortLinkNotFound
			}
			logger.Logger.Error("Database error while fetching short link",
				"short_code", code,
				"error", err.Error(),
			)
			return err
		}

		// Validate old passcode
		if link.Detail.Passcode != oldPasscode {
			return apperrors.ErrPasscodeIncorrect
		}

		link.Detail.Passcode = newPasscode

		// Update the detail record
		err = r.db.Model(&shortlink.ShortLinkDetail{}).Where("id = ?", link.Detail.ID).Update("passcode", newPasscode).Error
		if err != nil {
			logger.Logger.Error("Failed to update passcode",
				"short_code", code,
				"error", err.Error(),
			)
			return err
		}

	} else {
		err := r.db.Where("short_code = ?", code).
			Joins("LEFT JOIN short_link_details ON short_links.id = short_link_details.short_link_id").
			Select("short_links.*, short_link_details.passcode, short_link_details.id as detail_id").
			First(&link).Error

		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apperrors.ErrShortLinkNotFound
			}
			logger.Logger.Error("Database error while fetching short link",
				"short_code", code,
				"error", err.Error(),
			)
			return err
		}

		link.Detail.Passcode = newPasscode

		// Update the detail record
		err = r.db.Model(&shortlink.ShortLinkDetail{}).Where("id = ?", link.Detail.ID).Update("passcode", newPasscode).Error
		if err != nil {
			logger.Logger.Error("Failed to update passcode",
				"short_code", code,
				"error", err.Error(),
			)
			return err
		}
	}
	return nil
}

func (r *ShortLinkRepository) ForgotPasscodeShortLink(code string, newPasscode int, userID string, roleUser string) error {
	var link shortlink.ShortLink

	if roleUser != "admin" {
		err := r.db.Where("short_links.short_code = ? AND short_links.user_id = ?", code, userID).
			Joins("LEFT JOIN short_link_details ON short_links.id = short_link_details.short_link_id").
			Select("short_links.*, short_link_details.passcode, short_link_details.id as detail_id").
			First(&link).Error

		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apperrors.ErrShortLinkNotFound
			}
			logger.Logger.Error("Database error while fetching short link",
				"short_code", code,
				"error", err.Error(),
			)
			return err
		}

	} else {
		err := r.db.Where("short_code = ?", code).
			Joins("LEFT JOIN short_link_details ON short_links.id = short_link_details.short_link_id").
			Select("short_links.*, short_link_details.passcode, short_link_details.id as detail_id").
			First(&link).Error

		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apperrors.ErrShortLinkNotFound
			}
			logger.Logger.Error("Database error while fetching short link",
				"short_code", code,
				"error", err.Error(),
			)
			return err
		}
	}

	return nil
}

func (r *ShortLinkRepository) ValidatePasscodeToken(token string) (*shortlink.ShortLinkDetail, error) {
	var detail shortlink.ShortLinkDetail
	err := r.db.Where("passcode_token = ?", token).First(&detail).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Token not found
		}
		logger.Logger.Error("Database error while validating passcode token",
			"token", token,
			"error", err.Error(),
		)
		return nil, err
	}
	// Optionally, check expiry
	if !detail.PasscodeTokenExpiresAt.IsZero() && detail.PasscodeTokenExpiresAt.Before(time.Now()) {
		return nil, nil // Token expired
	}
	return &detail, nil // Token is valid
}

func (r *ShortLinkRepository) setPasscodeToken(userID string, linkID uint, token string, expiresAt time.Time) error {
	var detail shortlink.ShortLinkDetail

	err := r.db.Where("short_link_id = ?", linkID).First(&detail).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperrors.ErrShortLinkNotFound
		}

		logger.Logger.Error("Database error while fetching short link detail",
			"short_link_id", linkID,
			"error", err.Error(),
		)
		return err
	}
	detail.PasscodeTokenExpiresAt = expiresAt

	if err := r.db.Save(&detail).Error; err != nil {
		logger.Logger.Error("Failed to set passcode token",
			"short_link_id", linkID,
			"error", err.Error(),
		)
		return err
	}

	return nil
}

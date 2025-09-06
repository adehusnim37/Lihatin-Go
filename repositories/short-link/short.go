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

	if link.DeletedAt.Valid {
		utils.Logger.Warn("Deleted short link accessed",
			"short_code", code,
			"ip_address", ipAddress,
		)
		return nil, utils.ErrShortLinkAlreadyDeleted
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

	utils.Logger.Info("Short link retrieved successfully",
		"short_code", code,
		"user_id", userID,
	)

	return &dto.ShortLinkResponse{
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

func (r *ShortLinkRepository) ShortLinkStats(code string, userID string) (*dto.PaginatedViewLinkDetailResponse, error) {
	var link shortlink.ShortLink
	var viewDetail []shortlink.ViewLinkDetail

	// Get the short link first
	err := r.db.Where("short_code = ? AND user_id = ?", code, userID).First(&link).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, utils.ErrShortLinkNotFound
		}

		utils.Logger.Error("Database error while fetching short link",
			"short_code", code,
			"error", err.Error(),
		)
		return nil, err
	}

	err = r.db.Where("short_link_id = ?", link.ID).Order("clicked_at DESC").Find(&viewDetail).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, utils.ErrShortLinkNotFound
		}

		utils.Logger.Error("Database error while fetching short link stats",
			"short_code", code,
			"error", err.Error(),
		)
		return nil, err
	}

	// Convert recent views to response format
	recentViewsResponse := make([]dto.ViewLinkDetailResponse, 0, len(viewDetail))
	for _, view := range viewDetail {
		recentViewsResponse = append(recentViewsResponse, dto.ViewLinkDetailResponse{
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

	return &dto.PaginatedViewLinkDetailResponse{
		Views: recentViewsResponse,
	}, nil
}

func (r *ShortLinkRepository) UpdateShortLink(code string, userID string, updateData *dto.UpdateShortLinkRequest) error {
	var link shortlink.ShortLink

	err := r.db.Where("short_code = ? AND user_id = ?", code, userID).First(&link).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return utils.ErrShortLinkNotFound
		}

		utils.Logger.Error("Database error while fetching short link",
			"short_code", code,
			"error", err.Error(),
		)
		return err
	}

	// Update the link fields
	if updateData.Title != nil {
		link.Title = *updateData.Title
	}
	if updateData.Description != nil {
		link.Description = *updateData.Description
	}
	if updateData.IsActive != nil {
		link.IsActive = *updateData.IsActive
	}
	if updateData.ExpiresAt != nil {
		link.ExpiresAt = updateData.ExpiresAt
	}

	// Save the updated link
	if err := r.db.Save(&link).Error; err != nil {
		utils.Logger.Error("Failed to update short link",
			"short_code", code,
			"error", err.Error(),
		)
		return err
	}

	return nil
}

func (r *ShortLinkRepository) DeleteShortLink(code string, userID string, passcode int) error {
	var link shortlink.ShortLink
	
	err := r.db.Where("short_links.short_code = ? AND short_links.user_id = ?", code, userID).
		Joins("LEFT JOIN short_link_details ON short_links.id = short_link_details.short_link_id").
		Select("short_links.*, short_link_details.passcode").
		First(&link).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return utils.ErrShortLinkNotFound
		}

		utils.Logger.Error("Database error while fetching short link",
			"short_code", code,
			"error", err.Error(),
		)
		return err
	}

	if link.Detail.Passcode != passcode {
		return utils.ErrShortLinkUnauthorized
	}

	if link.DeletedAt.Valid {
		return utils.ErrShortLinkAlreadyDeleted
	}

	link.DeletedAt.Time = time.Now()
	link.DeletedAt.Valid = true

	if err := r.db.Save(&link).Error; err != nil {
		utils.Logger.Error("Failed to delete short link",
			"short_code", code,
			"error", err.Error(),
		)
		return err
	}

	return nil
}

// Admin

func (r *ShortLinkRepository) ListAllShortLinks(page, limit int, sort, orderBy string) (*dto.PaginatedShortLinksAdminResponse, error) {
	var shortLinks []shortlink.ShortLink
	var totalCount int64

	utils.Logger.Info("Fetching all short links for admin",
		"page", page,
		"limit", limit,
		"sort", sort,
		"order_by", orderBy,
	)

	// Get total count (without joins for better performance)
	if err := r.db.Model(&shortlink.ShortLink{}).Count(&totalCount).Error; err != nil {
		utils.Logger.Error("Failed to count short links", "error", err.Error())
		return nil, err
	}

	// Calculate offset and build order clause
	offset := (page - 1) * limit
	orderClause := sort + " " + orderBy

	// Query with LEFT JOINs to get all data including details and view counts
	if err := r.db.
		Preload("Detail"). // Load detail relationship
		Order(orderClause).
		Limit(limit).
		Offset(offset).
		Find(&shortLinks).Error; err != nil {
		utils.Logger.Error("Failed to fetch short links", "error", err.Error())
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

	utils.Logger.Info("Successfully fetched short links for admin",
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

// GetShortLinkStatsAdmin gets detailed statistics for a specific short link (admin only)
func (r *ShortLinkRepository) GetShortLinkStatsAdmin(shortCode string) (*dto.ShortLinkResponse, error) {
	var link shortlink.ShortLink

	utils.Logger.Info("Fetching short link stats for admin", "short_code", shortCode)

	// Get short link with detail
	if err := r.db.Preload("Detail").
		Where("short_code = ?", shortCode).
		First(&link).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, utils.ErrShortLinkNotFound
		}
		utils.Logger.Error("Failed to fetch short link", "error", err.Error())
		return nil, err
	}

	// Get total clicks
	var totalClicks int64
	r.db.Model(&shortlink.ViewLinkDetail{}).
		Where("short_link_id = ?", link.ID).
		Count(&totalClicks)

	// Get unique visitors
	var uniqueVisitors int64
	r.db.Model(&shortlink.ViewLinkDetail{}).
		Select("DISTINCT ip_address").
		Where("short_link_id = ?", link.ID).
		Count(&uniqueVisitors)

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
	response := &dto.ShortLinkResponse{
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
		RecentViews:     recentViewsResponse,
	}

	// Add latest view if exists
	if len(recentViews) > 0 {
		latestView := recentViews[0]
		response.RecentViews = append(response.RecentViews, dto.ViewLinkDetailResponse{
			ID:        latestView.ID,
			IPAddress: latestView.IPAddress,
			UserAgent: latestView.UserAgent,
			Referer:   latestView.Referer,
			Country:   latestView.Country,
			City:      latestView.City,
			Device:    latestView.Device,
			Browser:   latestView.Browser,
			OS:        latestView.OS,
			ClickedAt: latestView.ClickedAt,
		})
	}

	utils.Logger.Info("Successfully fetched short link stats for admin",
		"short_code", shortCode,
		"total_clicks", totalClicks,
		"unique_visitors", uniqueVisitors,
	)

	return response, nil
}

// DeleteShortLinkAdmin allows admin to delete any short link
func (r *ShortLinkRepository) DeleteShortLinkAdmin(shortCode string) error {
	utils.Logger.Info("Admin deleting short link", "short_code", shortCode)

	// Soft delete the short link
	result := r.db.Where("short_code = ?", shortCode).Delete(&shortlink.ShortLink{})
	if result.Error != nil {
		utils.Logger.Error("Failed to delete short link", "error", result.Error)
		return result.Error
	}

	if result.RowsAffected == 0 {
		return utils.ErrShortLinkNotFound
	}

	utils.Logger.Info("Successfully deleted short link", "short_code", shortCode)
	return nil
}

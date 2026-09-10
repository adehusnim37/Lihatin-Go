package shortlink

import (
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/adehusnim37/lihatin-go/dto"
	apperrors "github.com/adehusnim37/lihatin-go/internal/pkg/errors"
	"github.com/adehusnim37/lihatin-go/internal/pkg/helpers"
	"github.com/adehusnim37/lihatin-go/internal/pkg/identifier"
	"github.com/adehusnim37/lihatin-go/internal/pkg/ip"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	shortlink "github.com/adehusnim37/lihatin-go/models/shortlink"
	mysqldriver "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

type ShortLinkRepository struct {
	db *gorm.DB
}

func NewShortLinkRepository(db *gorm.DB) *ShortLinkRepository {
	return &ShortLinkRepository{db: db}
}

func (r *ShortLinkRepository) CreateShortLink(link *dto.CreateShortLinkRequest) (*shortlink.ShortLink, *shortlink.ShortLinkDetail, error) {
	// Fast-path UX check: if the user supplied a custom code, look it up first so we can
	// return a clear error quickly instead of waiting for the DB unique constraint to fire.
	// NOTE: this is purely a UX optimization - the unique index on short_code is still the
	// real safety net against race conditions (two requests could pass this check at the
	// same time), so we still handle gorm.ErrDuplicatedKey below.
	if link.CustomCode != "" {
		if err := r.db.Unscoped().Select("id").Where("short_code = ?", link.CustomCode).First(&shortlink.ShortLink{}).Error; err == nil {
			return nil, nil, apperrors.ErrDuplicateShortCode
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Logger.Error("Database error while checking duplicate short code", "error", err.Error())
			return nil, nil, apperrors.ErrShortGetFailed.WithError(err)
		}
	}

	isGeneratedCode := link.CustomCode == ""
	var permutationKey []byte
	if isGeneratedCode {
		var err error
		permutationKey, err = loadShortCodePermutationKey()
		if err != nil {
			return nil, nil, apperrors.ErrShortCreatedFailed.WithError(err)
		}
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
		ID:          identifier.NewUUIDV7(),
		UserID:      userIDPtr, // ✅ Use pointer for nullable field
		ShortCode:   link.CustomCode,
		OriginalURL: link.OriginalURL,
		Title:       link.Title,
		Description: link.Description,
		ExpiresAt:   link.ExpiresAt,
	}

	var utmSource, utmMedium, utmCampaign, utmTerm, utmContent string
	if link.Tags != nil {
		utmSource = helpers.PtrToString(link.Tags.UTMSource)
		utmMedium = helpers.PtrToString(link.Tags.UTMMedium)
		utmCampaign = helpers.PtrToString(link.Tags.UTMCampaign)
		utmTerm = helpers.PtrToString(link.Tags.UTMTerm)
		utmContent = helpers.PtrToString(link.Tags.UTMContent)
	}

	shortLinkDetail := shortlink.ShortLinkDetail{
		ID:          identifier.NewUUIDV7(),
		ShortLinkID: shortLink.ID,
		Passcode:    helpers.StringToInt(link.Passcode),
		ClickLimit:  helpers.PtrToValue(link.Limit, 0),
		EnableStats: helpers.PtrToValue(link.EnableStats, true),
		UTMSource:   utmSource,
		UTMMedium:   utmMedium,
		UTMCampaign: utmCampaign,
		UTMTerm:     utmTerm,
		UTMContent:  utmContent,
	}

	// The allocator reservation, short link, and detail are committed atomically. If
	// any write fails, the counter is rolled back together with the link.
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if isGeneratedCode {
			allocator := newShortCodeAllocatorSession(tx, permutationKey)
			for {
				code, err := allocator.nextCode()
				if err != nil {
					return apperrors.ErrShortCreatedFailed.WithError(err)
				}
				shortLink.ShortCode = code

				if err := tx.Create(&shortLink).Error; err != nil {
					if isDuplicateKeyError(err) {
						// Existing custom/legacy codes can occupy an ordinal. Consume
						// it and deterministically try the next one.
						continue
					}
					logger.Logger.Error("Failed to create short link", "error", err.Error())
					return apperrors.ErrShortCreatedFailed.WithError(err)
				}
				if err := allocator.flush(); err != nil {
					return apperrors.ErrShortCreatedFailed.WithError(err)
				}
				break
			}
		} else if err := tx.Create(&shortLink).Error; err != nil {
			logger.Logger.Error("Failed to create short link", "error", err.Error())
			if isDuplicateKeyError(err) {
				return apperrors.ErrDuplicateShortCode
			}
			return apperrors.ErrShortCreatedFailed.WithError(err)
		}

		if err := tx.Create(&shortLinkDetail).Error; err != nil {
			logger.Logger.Error("Failed to create short link detail", "error", err.Error())
			return apperrors.ErrShortDetailCreatedFailed.WithError(err)
		}

		return nil
	})

	if err != nil {
		return nil, nil, err
	}
	link.CustomCode = shortLink.ShortCode

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

	hasGeneratedCode := false
	for i := range links {
		if links[i].CustomCode == "" {
			hasGeneratedCode = true
			break
		}
	}
	var permutationKey []byte
	if hasGeneratedCode {
		var err error
		permutationKey, err = loadShortCodePermutationKey()
		if err != nil {
			return nil, nil, apperrors.ErrShortBulkCreateFailed.WithError(err)
		}
	}

	var createdLinks []shortlink.ShortLink
	var createdDetails []shortlink.ShortLinkDetail

	// Allocation and both batch inserts share one transaction. Bulk allocation
	// locks one allocator row and advances it once, rather than once per link.
	err := r.db.Transaction(func(tx *gorm.DB) error {
		preparedLinks, err := r.prepareBulkShortLinkRequests(tx, links, permutationKey)
		if err != nil {
			return err
		}

		createdLinks = make([]shortlink.ShortLink, 0, len(preparedLinks))
		createdDetails = make([]shortlink.ShortLinkDetail, 0, len(preparedLinks))
		for _, linkReq := range preparedLinks {
			var userIDPtr *string
			if linkReq.UserID != "" {
				userID := linkReq.UserID
				userIDPtr = &userID
			}

			createdLinks = append(createdLinks, shortlink.ShortLink{
				ID:          identifier.NewUUIDV7(),
				UserID:      userIDPtr,
				ShortCode:   linkReq.CustomCode,
				OriginalURL: linkReq.OriginalURL,
				Title:       linkReq.Title,
				Description: linkReq.Description,
				ExpiresAt:   linkReq.ExpiresAt,
			})
		}

		for i, linkReq := range preparedLinks {
			createdDetails = append(createdDetails, shortlink.ShortLinkDetail{
				ID:          identifier.NewUUIDV7(),
				ShortLinkID: createdLinks[i].ID,
				Passcode:    helpers.StringToInt(linkReq.Passcode),
			})
		}

		if err := tx.Create(&createdLinks).Error; err != nil {
			if isDuplicateKeyError(err) {
				return apperrors.ErrDuplicateShortCode
			}
			return apperrors.ErrShortCreatedFailed.WithError(err)
		}

		if err := tx.Create(&createdDetails).Error; err != nil {
			return apperrors.ErrShortDetailCreatedFailed.WithError(err)
		}

		return nil
	})

	if err != nil {
		logger.Logger.Error("Failed to create bulk short links", "error", err.Error())
		return nil, nil, apperrors.ErrShortBulkCreateFailed.WithError(err)
	}

	logger.Logger.Info("Bulk short links created successfully",
		"count", len(createdLinks),
		"user_id", links[0].UserID,
	)

	return createdLinks, createdDetails, nil
}

func (r *ShortLinkRepository) prepareBulkShortLinkRequests(tx *gorm.DB, links []dto.CreateShortLinkRequest, permutationKey []byte) ([]dto.CreateShortLinkRequest, error) {
	prepared := append([]dto.CreateShortLinkRequest(nil), links...)
	generated := make([]bool, len(prepared))
	reserved := make(map[string]struct{}, len(prepared))

	customCodes := make([]string, 0, len(prepared))
	for i := range prepared {
		if prepared[i].CustomCode == "" {
			generated[i] = true
			continue
		}
		if _, exists := reserved[prepared[i].CustomCode]; exists {
			return nil, apperrors.ErrDuplicateShortCodeInBatch
		}
		reserved[prepared[i].CustomCode] = struct{}{}
		customCodes = append(customCodes, prepared[i].CustomCode)
	}

	if len(customCodes) > 0 {
		var existingCustomCodes []string
		if err := tx.Unscoped().Model(&shortlink.ShortLink{}).
			Where("short_code IN ?", customCodes).
			Pluck("short_code", &existingCustomCodes).Error; err != nil {
			return nil, apperrors.ErrShortGetFailed.WithError(err)
		}
		if len(existingCustomCodes) > 0 {
			return nil, apperrors.ErrDuplicateShortCode
		}
	}

	allocator := newShortCodeAllocatorSession(tx, permutationKey)
	generateAt := func(i int) error {
		for {
			code, err := allocator.nextCode()
			if err != nil {
				return err
			}
			if _, exists := reserved[code]; exists {
				continue
			}
			prepared[i].CustomCode = code
			reserved[code] = struct{}{}
			return nil
		}
	}

	for i := range prepared {
		if generated[i] {
			if err := generateAt(i); err != nil {
				return nil, apperrors.ErrShortCreatedFailed.WithError(err)
			}
		}
	}

	for {
		codes := make([]string, len(prepared))
		for i := range prepared {
			codes[i] = prepared[i].CustomCode
		}

		var existingCodes []string
		if err := tx.Unscoped().Model(&shortlink.ShortLink{}).
			Where("short_code IN ?", codes).
			Pluck("short_code", &existingCodes).Error; err != nil {
			return nil, apperrors.ErrShortGetFailed.WithError(err)
		}
		if len(existingCodes) == 0 {
			if err := allocator.flush(); err != nil {
				return nil, apperrors.ErrShortCreatedFailed.WithError(err)
			}
			return prepared, nil
		}

		existing := make(map[string]struct{}, len(existingCodes))
		for _, code := range existingCodes {
			existing[code] = struct{}{}
			reserved[code] = struct{}{}
		}

		regenerated := false
		for i := range prepared {
			if _, collides := existing[prepared[i].CustomCode]; !collides {
				continue
			}
			if !generated[i] {
				return nil, apperrors.ErrDuplicateShortCode
			}
			if err := generateAt(i); err != nil {
				return nil, apperrors.ErrShortCreatedFailed.WithError(err)
			}
			regenerated = true
		}
		if !regenerated {
			return nil, apperrors.ErrDuplicateShortCode
		}
	}
}

func isDuplicateKeyError(err error) bool {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	var mysqlErr *mysqldriver.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}

// GetShortsByUserIDWithPagination gets short links with pagination and sorting
func (r *ShortLinkRepository) GetShortsByUserIDWithPagination(userID string, page, limit int, sort, orderBy, search string) (*dto.PaginatedShortLinksResponse, error) {
	var links []shortlink.ShortLink
	var totalCount int64

	baseQuery := r.db.Model(&shortlink.ShortLink{}).Where("user_id = ?", userID)
	if strings.TrimSpace(search) != "" {
		keyword := "%" + strings.ToLower(strings.TrimSpace(search)) + "%"
		baseQuery = baseQuery.Where(
			"LOWER(short_code) LIKE ? OR LOWER(original_url) LIKE ? OR LOWER(title) LIKE ? OR LOWER(description) LIKE ?",
			keyword, keyword, keyword, keyword,
		)
	}

	// Get total count
	if err := baseQuery.Count(&totalCount).Error; err != nil {
		return nil, apperrors.ErrShortGetFailed.WithError(err)
	}

	// Calculate offset
	offset := (page - 1) * limit

	// Build order clause
	orderClause := fmt.Sprintf("%s %s", sort, orderBy)

	// Get paginated results with sorting
	findQuery := r.db.Where("user_id = ?", userID)
	if strings.TrimSpace(search) != "" {
		keyword := "%" + strings.ToLower(strings.TrimSpace(search)) + "%"
		findQuery = findQuery.Where(
			"LOWER(short_code) LIKE ? OR LOWER(original_url) LIKE ? OR LOWER(title) LIKE ? OR LOWER(description) LIKE ?",
			keyword, keyword, keyword, keyword,
		)
	}

	if err := findQuery.
		Order(orderClause).
		Offset(offset).
		Limit(limit).
		Find(&links).Error; err != nil {
		return nil, apperrors.ErrShortGetFailed.WithError(err)
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

	clickCounts := make(map[string]int64, len(links))
	if len(links) > 0 {
		linkIDs := make([]string, len(links))
		for i := range links {
			linkIDs[i] = links[i].ID
		}

		var counts []struct {
			ShortLinkID string
			Count       int64
		}
		if err := r.db.Model(&shortlink.ViewLinkDetail{}).
			Select("short_link_id, COUNT(*) AS count").
			Where("short_link_id IN ?", linkIDs).
			Group("short_link_id").
			Scan(&counts).Error; err != nil {
			return nil, apperrors.ErrShortGetFailed.WithError(err)
		}
		for _, count := range counts {
			clickCounts[count.ShortLinkID] = count.Count
		}
	}

	// Convert ShortLink to ShortsLinkResponse with pre-allocated capacity
	shortLinkResponses := make([]dto.ShortsLinkResponse, 0, len(links))
	for _, link := range links {
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
			ClickCount:  int(clickCounts[link.ID]),
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

func (r *ShortLinkRepository) RedirectByShortCode(code string, ipAddress, userAgent, referer, device, browser, os string, passcode int) (*shortlink.ShortLink, error) {
	var link shortlink.ShortLink
	// Detail is required for every redirect, so load the one-to-one association in
	// the same round trip as the short link.
	err := r.db.Joins("Detail").Where("short_links.short_code = ?", code).First(&link).Error
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
		return nil, apperrors.ErrShortGetFailed.WithError(err)
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

	if link.Detail == nil {
		logger.Logger.Error("Failed to fetch short link detail",
			"short_code", code,
		)
		return nil, apperrors.ErrShortDetailNotFound
	}
	detail := *link.Detail

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

		return nil, apperrors.ErrLinkIsBanned
	}

	if detail.ClickLimit > 0 && detail.CurrentClicks >= detail.ClickLimit {
		logger.Logger.Warn("Click limit reached",
			"short_code", code,
			"ip_address", ipAddress,
		)
		return nil, apperrors.ErrClickLimitReached
	}

	// The limit predicate and increment happen atomically. The earlier check is only
	// a fast path; RowsAffected protects the limit under concurrent redirects.
	updateQuery := r.db.Model(&shortlink.ShortLinkDetail{}).Where("id = ?", detail.ID)
	if detail.ClickLimit > 0 {
		updateQuery = updateQuery.Where("current_clicks < ?", detail.ClickLimit)
	}
	result := updateQuery.
		Updates(map[string]interface{}{
			"current_clicks": gorm.Expr("current_clicks + ?", 1),
			"updated_at":     time.Now(),
		})
	if result.Error != nil {
		logger.Logger.Error("Failed to update short link detail",
			"short_code", code,
			"ip_address", ipAddress,
			"error", result.Error.Error(),
		)
		return nil, apperrors.ErrShortDetailUpdateFailed.WithError(result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, apperrors.ErrClickLimitReached
	}

	detail.CurrentClicks++
	link.Detail = &detail

	// Track the click with basic info in background
	go func() {
		// Use GetLocation once to avoid double API calls and potential blocking
		country, city := ip.GetLocation(ipAddress)

		viewDetail := shortlink.ViewLinkDetail{
			ID:          identifier.NewUUIDV7(),
			ShortLinkID: link.ID,
			IPAddress:   ipAddress,
			UserAgent:   userAgent,
			Referer:     referer,
			Country:     country,
			City:        city,
			Device:      device,
			Browser:     browser,
			OS:          os,
			ClickedAt:   time.Now(),
		}

		if err := r.db.Create(&viewDetail).Error; err != nil {
			logger.Logger.Error("Failed to track click",
				"short_code", code,
				"ip_address", ipAddress,
				"error", err.Error(),
			)
		}

		logger.Logger.Info("Click tracked successfully",
			"short_code", code,
			"ip_address", ipAddress,
		)
	}()

	return &link, nil
}

func (r *ShortLinkRepository) GetShortLink(code string, userID string, userRole string) (*dto.ShortLinkResponse, error) {
	var link shortlink.ShortLink
	var detail shortlink.ShortLinkDetail

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
			return nil, apperrors.ErrShortGetFailed.WithError(err)
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
			return nil, apperrors.ErrShortGetFailed.WithError(err)
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
			return nil, apperrors.ErrShortDetailNotFound
		}
		logger.Logger.Error("Database error while fetching short link detail",
			"short_code", code,
			"error", err.Error(),
		)
		return nil, apperrors.ErrShortDetailFindFailed.WithError(err)
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
		IsBanned:      detail.IsBanned,
		BannedReason:  detail.BannedReason,
		BannedBy:      detail.BannedBy,
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
	}

	return shortLinkResponse, nil
}

func (r *ShortLinkRepository) GetStatsShortLink(code string, userId string, userRole string) (*dto.ShortLinkWithStatsResponse, error) {
	var link shortlink.ShortLink
	var countries []dto.Country
	var devices []dto.TopDevice
	var referrers []dto.TopReferrer

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
			return nil, apperrors.ErrShortGetFailed.WithError(err)
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
			return nil, apperrors.ErrShortGetFailed.WithError(err)
		}
	}

	now := time.Now()
	var counters struct {
		TotalClicks    int64
		UniqueVisitors int64
		Last24h        int64
		Last7d         int64
		Last30d        int64
		Last60d        int64
		Last90d        int64
	}
	if err := r.db.Model(&shortlink.ViewLinkDetail{}).
		Select(`COUNT(*) AS total_clicks,
			COUNT(DISTINCT ip_address) AS unique_visitors,
			COALESCE(SUM(CASE WHEN clicked_at >= ? THEN 1 ELSE 0 END), 0) AS last24h,
			COALESCE(SUM(CASE WHEN clicked_at >= ? THEN 1 ELSE 0 END), 0) AS last7d,
			COALESCE(SUM(CASE WHEN clicked_at >= ? THEN 1 ELSE 0 END), 0) AS last30d,
			COALESCE(SUM(CASE WHEN clicked_at >= ? THEN 1 ELSE 0 END), 0) AS last60d,
			COALESCE(SUM(CASE WHEN clicked_at >= ? THEN 1 ELSE 0 END), 0) AS last90d`,
			now.Add(-24*time.Hour),
			now.Add(-7*24*time.Hour),
			now.Add(-30*24*time.Hour),
			now.Add(-60*24*time.Hour),
			now.Add(-90*24*time.Hour)).
		Where("short_link_id = ?", link.ID).
		Scan(&counters).Error; err != nil {
		return nil, apperrors.ErrShortViewTrackFailed.WithError(err)
	}

	// Get top countries by views.
	if err := r.db.Model(&shortlink.ViewLinkDetail{}).
		Select("country, COUNT(*) as count").
		Where("short_link_id = ?", link.ID).
		Group("country").
		Order("count DESC").
		Scan(&countries).Error; err != nil {
		return nil, apperrors.ErrShortStatsFailed.WithError(err)
	}

	// Add "Other" if > 5
	if len(countries) > 5 {
		var otherCount int
		for i := 5; i < len(countries); i++ {
			otherCount += countries[i].Count
		}
		countries = countries[:5]
		if otherCount > 0 {
			countries = append(countries, dto.Country{Country: "Other", Count: otherCount})
		}
	}

	// Parse and Aggregate Referrers (Host-based)
	var rawReferrers []struct {
		Referer string
		Count   int
	}
	if err := r.db.Model(&shortlink.ViewLinkDetail{}).
		Select("referer, COUNT(*) as count").
		Where("short_link_id = ?", link.ID).
		Group("referer").
		Scan(&rawReferrers).Error; err != nil {
		return nil, apperrors.ErrShortStatsFailed.WithError(err)
	}

	referrerMap := make(map[string]int)
	for _, ref := range rawReferrers {
		host := "Direct / None"
		if ref.Referer != "" {
			if u, err := url.Parse(ref.Referer); err == nil && u.Host != "" {
				host = u.Host
			} else {
				host = ref.Referer // Fallback
			}
		}
		referrerMap[host] += ref.Count
	}

	for host, count := range referrerMap {
		referrers = append(referrers, dto.TopReferrer{Host: host, Count: count})
	}
	// Sort referrers DESC
	sort.Slice(referrers, func(i, j int) bool {
		return referrers[i].Count > referrers[j].Count
	})
	if len(referrers) > 5 {
		var otherCount int
		for i := 5; i < len(referrers); i++ {
			otherCount += referrers[i].Count
		}
		referrers = referrers[:5]
		if otherCount > 0 {
			referrers = append(referrers, dto.TopReferrer{Host: "Other", Count: otherCount})
		}
	}

	// Device is normalized when the view is written, so aggregate that low-cardinality
	// column directly instead of grouping and transferring every distinct user-agent.
	if err := r.db.Model(&shortlink.ViewLinkDetail{}).
		Select("COALESCE(NULLIF(device, ''), 'Unknown') AS device, COUNT(*) AS count").
		Where("short_link_id = ?", link.ID).
		Group("device").
		Order("count DESC").
		Scan(&devices).Error; err != nil {
		return nil, apperrors.ErrShortStatsFailed.WithError(err)
	}
	if len(devices) > 5 {
		var otherCount int
		for i := 5; i < len(devices); i++ {
			otherCount += devices[i].Count
		}
		devices = devices[:5]
		if otherCount > 0 {
			devices = append(devices, dto.TopDevice{Device: "Other", Count: otherCount})
		}
	}

	// Get Click History (Daily for last 90 days)
	var history []dto.ClickHistoryItem
	if err := r.db.Model(&shortlink.ViewLinkDetail{}).
		Select("DATE_FORMAT(clicked_at, '%Y-%m-%d') as date, COUNT(*) as count").
		Where("short_link_id = ? AND clicked_at >= ?", link.ID, now.Add(-90*24*time.Hour)).
		Group("DATE_FORMAT(clicked_at, '%Y-%m-%d')").
		Order("date ASC").
		Scan(&history).Error; err != nil {
		return nil, apperrors.ErrShortStatsFailed.WithError(err)
	}

	// Get Click History Hourly (Last 24 Hours)
	var historyHourly []dto.ClickHistoryItem
	if err := r.db.Model(&shortlink.ViewLinkDetail{}).
		Select("DATE_FORMAT(clicked_at, '%Y-%m-%d %H:00') as date, COUNT(*) as count").
		Where("short_link_id = ? AND clicked_at >= ?", link.ID, now.Add(-24*time.Hour)).
		Group("DATE_FORMAT(clicked_at, '%Y-%m-%d %H:00')").
		Order("date ASC").
		Scan(&historyHourly).Error; err != nil {
		return nil, apperrors.ErrShortStatsFailed.WithError(err)
	}

	return &dto.ShortLinkWithStatsResponse{
		ShortCode:          link.ShortCode,
		TotalClicks:        int(counters.TotalClicks),
		UniqueVisitors:     int(counters.UniqueVisitors),
		Last24h:            int(counters.Last24h),
		Last7d:             int(counters.Last7d),
		Last30d:            int(counters.Last30d),
		Last60d:            int(counters.Last60d),
		Last90d:            int(counters.Last90d),
		TopReferrers:       referrers,
		TopDevices:         devices,
		TopCountries:       countries,
		ClickHistory:       history,
		ClickHistoryHourly: historyHourly,
	}, nil
}

func (r *ShortLinkRepository) GetDashboardStats(userId string, userRole string, startDate, endDate string) (*dto.DashboardStatsResponse, error) {
	var summary dto.DashboardSummary
	useDateFilter := startDate != "" && endDate != ""
	periodStart := startDate + " 00:00:00"
	periodEnd := endDate + " 23:59:59"
	now := time.Now()

	linkQuery := r.db.Model(&shortlink.ShortLink{})
	if userRole != "admin" {
		linkQuery = linkQuery.Where("user_id = ?", userId)
	}
	var linkCounts struct {
		TotalLinks  int64
		ActiveLinks int64
	}
	if err := linkQuery.
		Select("COUNT(*) AS total_links, COALESCE(SUM(CASE WHEN is_active = ? THEN 1 ELSE 0 END), 0) AS active_links", true).
		Scan(&linkCounts).Error; err != nil {
		return nil, apperrors.ErrShortStatsFailed.WithError(err)
	}
	summary.TotalLinks = linkCounts.TotalLinks
	summary.ActiveLinks = linkCounts.ActiveLinks
	summary.InactiveLinks = linkCounts.TotalLinks - linkCounts.ActiveLinks

	newViewQuery := func() *gorm.DB {
		query := r.db.Model(&shortlink.ViewLinkDetail{}).
			Joins("JOIN short_links ON short_links.id = view_link_details.short_link_id").
			Where("short_links.deleted_at IS NULL")
		if userRole != "admin" {
			query = query.Where("short_links.user_id = ?", userId)
		}
		return query
	}
	applyDateFilter := func(query *gorm.DB) *gorm.DB {
		if useDateFilter {
			return query.Where("view_link_details.clicked_at >= ? AND view_link_details.clicked_at <= ?", periodStart, periodEnd)
		}
		return query
	}

	totalExpression := "COUNT(*)"
	uniqueExpression := "COUNT(DISTINCT view_link_details.ip_address)"
	selectArgs := make([]any, 0, 9)
	if useDateFilter {
		totalExpression = "COUNT(CASE WHEN view_link_details.clicked_at >= ? AND view_link_details.clicked_at <= ? THEN 1 END)"
		uniqueExpression = "COUNT(DISTINCT CASE WHEN view_link_details.clicked_at >= ? AND view_link_details.clicked_at <= ? THEN view_link_details.ip_address END)"
		selectArgs = append(selectArgs, periodStart, periodEnd, periodStart, periodEnd)
	}
	selectArgs = append(selectArgs,
		now.Add(-24*time.Hour),
		now.Add(-7*24*time.Hour),
		now.Add(-30*24*time.Hour),
		now.Add(-60*24*time.Hour),
		now.Add(-90*24*time.Hour),
	)
	statsSelect := fmt.Sprintf(`%s AS total_clicks,
		%s AS total_unique_visitors,
		COALESCE(SUM(CASE WHEN view_link_details.clicked_at >= ? THEN 1 ELSE 0 END), 0) AS clicks_last24h,
		COALESCE(SUM(CASE WHEN view_link_details.clicked_at >= ? THEN 1 ELSE 0 END), 0) AS clicks_last7d,
		COALESCE(SUM(CASE WHEN view_link_details.clicked_at >= ? THEN 1 ELSE 0 END), 0) AS clicks_last30d,
		COALESCE(SUM(CASE WHEN view_link_details.clicked_at >= ? THEN 1 ELSE 0 END), 0) AS clicks_last60d,
		COALESCE(SUM(CASE WHEN view_link_details.clicked_at >= ? THEN 1 ELSE 0 END), 0) AS clicks_last90d`, totalExpression, uniqueExpression)
	var viewCounters struct {
		TotalClicks         int64
		TotalUniqueVisitors int64
		ClicksLast24h       int64
		ClicksLast7d        int64
		ClicksLast30d       int64
		ClicksLast60d       int64
		ClicksLast90d       int64
	}
	if err := newViewQuery().Select(statsSelect, selectArgs...).Scan(&viewCounters).Error; err != nil {
		return nil, apperrors.ErrShortStatsFailed.WithError(err)
	}
	summary.TotalClicks = viewCounters.TotalClicks
	summary.TotalUniqueVisitors = viewCounters.TotalUniqueVisitors
	summary.ClicksLast24h = viewCounters.ClicksLast24h
	summary.ClicksLast7d = viewCounters.ClicksLast7d
	summary.ClicksLast30d = viewCounters.ClicksLast30d
	summary.ClicksLast60d = viewCounters.ClicksLast60d
	summary.ClicksLast90d = viewCounters.ClicksLast90d

	var aggCountries []dto.Country
	if err := applyDateFilter(newViewQuery()).
		Select("COALESCE(NULLIF(view_link_details.country, ''), 'Unknown') AS country, COUNT(*) AS count").
		Group("view_link_details.country").
		Order("count DESC").
		Limit(5).
		Scan(&aggCountries).Error; err != nil {
		return nil, apperrors.ErrShortStatsFailed.WithError(err)
	}
	summary.TopCountries = aggCountries

	var aggDevices []dto.TopDevice
	if err := applyDateFilter(newViewQuery()).
		Select("COALESCE(NULLIF(view_link_details.device, ''), 'Unknown') AS device, COUNT(*) AS count").
		Group("view_link_details.device").
		Order("count DESC").
		Limit(5).
		Scan(&aggDevices).Error; err != nil {
		return nil, apperrors.ErrShortStatsFailed.WithError(err)
	}
	summary.TopDevices = aggDevices

	var rawRefs []struct {
		Referer string
		Count   int
	}
	if err := applyDateFilter(newViewQuery()).
		Select("view_link_details.referer, COUNT(*) AS count").
		Group("view_link_details.referer").
		Scan(&rawRefs).Error; err != nil {
		return nil, apperrors.ErrShortStatsFailed.WithError(err)
	}
	refMap := make(map[string]int)
	for _, ref := range rawRefs {
		host := "Direct / None"
		if ref.Referer != "" {
			if parsed, err := url.Parse(ref.Referer); err == nil && parsed.Host != "" {
				host = parsed.Host
			} else {
				host = ref.Referer
			}
		}
		refMap[host] += ref.Count
	}
	aggReferrers := make([]dto.TopReferrer, 0, len(refMap))
	for host, count := range refMap {
		aggReferrers = append(aggReferrers, dto.TopReferrer{Host: host, Count: count})
	}
	sort.Slice(aggReferrers, func(i, j int) bool {
		return aggReferrers[i].Count > aggReferrers[j].Count
	})
	if len(aggReferrers) > 5 {
		aggReferrers = aggReferrers[:5]
	}
	summary.TopReferrers = aggReferrers

	var clickHistory []dto.ClickHistoryItem
	historyQuery := newViewQuery().
		Select("DATE_FORMAT(view_link_details.clicked_at, '%Y-%m-%d') AS date, COUNT(*) AS count")
	if useDateFilter {
		historyQuery = applyDateFilter(historyQuery)
	} else {
		historyQuery = historyQuery.Where("view_link_details.clicked_at >= ?", now.Add(-90*24*time.Hour))
	}
	if err := historyQuery.
		Group("DATE_FORMAT(view_link_details.clicked_at, '%Y-%m-%d')").
		Order("date ASC").
		Scan(&clickHistory).Error; err != nil {
		return nil, apperrors.ErrShortStatsFailed.WithError(err)
	}
	summary.ClickHistory = clickHistory

	return &dto.DashboardStatsResponse{
		Summary: &summary,
	}, nil
}

// GetShortLinkViewsPaginated gets paginated views for a specific short link
func (r *ShortLinkRepository) GetShortLinkViewsPaginated(code string, userID string, page, limit int, sort, orderBy string, userRole string) (*dto.PaginatedShortLinkDetailWithStatsResponse, error) {
	var link shortlink.ShortLink
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
			return nil, apperrors.ErrShortGetFailed.WithError(err)
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
			return nil, apperrors.ErrShortGetFailed.WithError(err)
		}
	}

	// Get total count of views
	if err := r.db.Model(&shortlink.ViewLinkDetail{}).Where("short_link_id = ?", link.ID).Count(&totalCount).Error; err != nil {
		return nil, apperrors.ErrShortGetFailed.WithError(err)
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
	err := r.db.Where("short_link_id = ?", link.ID).
		Order(orderClause).
		Offset(offset).
		Limit(limit).
		Find(&viewDetails).Error
	if err != nil {
		logger.Logger.Error("Database error while fetching paginated views",
			"short_code", code,
			"error", err.Error(),
		)
		return nil, apperrors.ErrShortGetFailed.WithError(err)
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
		Views: paginatedViews,
	}, nil
}

func (r *ShortLinkRepository) CheckShortCode(code *dto.CodeRequest) (*dto.ShortLinkPreviewResponse, error) {
	type previewRecord struct {
		ShortCode   string
		OriginalURL string
		Title       string
		Description string
		Passcode    int
	}

	var preview previewRecord
	q := r.db.Table("short_links").
		Select("short_links.short_code, short_links.original_url, short_links.title, short_links.description, short_link_details.passcode").
		Joins("JOIN short_link_details ON short_link_details.short_link_id = short_links.id AND short_link_details.deleted_at IS NULL").
		Where("short_links.short_code = ? AND short_links.deleted_at IS NULL", code.Code)
	if code.Passcode != "" {
		passcodeInt := helpers.StringToInt(code.Passcode)
		q = q.Where("short_link_details.passcode = ?", passcodeInt)
	}
	err := q.Take(&preview).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Code or supplied passcode does not exist
		}
		logger.Logger.Error("Database error while checking short code",
			"short_code", code.Code,
			"error", err.Error(),
		)
		return nil, apperrors.ErrShortCheckFailed.WithError(err)
	}

	destination, err := url.Parse(preview.OriginalURL)
	if err != nil {
		logger.Logger.Warn("Unable to parse destination URL for public preview",
			"short_code", code.Code,
			"error", err.Error(),
		)
	}

	var destinationHost, destinationScheme string
	if destination != nil {
		destinationHost = destination.Hostname()
		destinationScheme = strings.ToLower(destination.Scheme)
	}

	return &dto.ShortLinkPreviewResponse{
		ShortCode:         preview.ShortCode,
		DestinationHost:   destinationHost,
		DestinationScheme: destinationScheme,
		Title:             preview.Title,
		Description:       preview.Description,
		RequiresPasscode:  preview.Passcode != 0,
	}, nil
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
		return apperrors.ErrShortUpdateFailed.WithError(err)
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
	if in.ExpiresAtSet {
		linkUpd["expires_at"] = in.ExpiresAt
	}
	if in.ShortCode != nil {
		linkUpd["short_code"] = *in.ShortCode
	}

	if len(linkUpd) > 0 {
		if err := tx.Model(&shortlink.ShortLink{}).
			Where("id = ?", link.ID).
			Updates(linkUpd).Error; err != nil {
			tx.Rollback()
			return apperrors.ErrShortUpdateFailed.WithError(err)
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
	if in.UTMSource != nil {
		detailUpd["utm_source"] = *in.UTMSource
	}
	if in.UTMMedium != nil {
		detailUpd["utm_medium"] = *in.UTMMedium
	}
	if in.UTMCampaign != nil {
		detailUpd["utm_campaign"] = *in.UTMCampaign
	}
	if in.UTMTerm != nil {
		detailUpd["utm_term"] = *in.UTMTerm
	}
	if in.UTMContent != nil {
		detailUpd["utm_content"] = *in.UTMContent
	}

	if len(detailUpd) > 0 {
		if err := tx.Model(&shortlink.ShortLinkDetail{}).
			Where("short_link_id = ?", link.ID).
			Updates(detailUpd).Error; err != nil {
			tx.Rollback()
			return apperrors.ErrShortDetailUpdateFailed.WithError(err)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return apperrors.ErrShortUpdateFailed.WithError(err)
	}
	return nil
}

func (r *ShortLinkRepository) ToggleActiveInActiveShort(code string, userID string, roleUser string) error {
	var link shortlink.ShortLink

	q := r.db.Where("short_code = ?", code)
	if roleUser != "admin" {
		q = q.Where("user_id = ?", userID)
	}

	if err := q.First(&link).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperrors.ErrShortLinkNotFound
		}
		return apperrors.ErrShortGetFailed.WithError(err)
	}

	if link.IsActive {
		link.IsActive = false
	} else {
		link.IsActive = true
	}

	if link.ExpiresAt != nil {
		link.ExpiresAt = nil
	}

	if err := r.db.Save(&link).Error; err != nil {
		return apperrors.ErrShortUpdateFailed.WithError(err)
	}

	return nil
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
			return apperrors.ErrShortGetFailed.WithError(err)
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
			return apperrors.ErrShortGetFailed.WithError(err)
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
		return apperrors.ErrShortDeleteFailed.WithError(err)
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
		return apperrors.ErrShortGetFailed.WithError(err)
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
		return apperrors.ErrShortDeleteFailed.WithError(err)
	}

	return nil
}

func (r *ShortLinkRepository) ListAllShortLinks(userID string, page, limit int, sort, orderBy, search string) (*dto.PaginatedShortLinksAdminResponse, error) {
	var shortLinks []shortlink.ShortLink
	var totalCount int64

	logger.Logger.Info("Fetching all short links for admin",
		"page", page,
		"limit", limit,
		"sort", sort,
		"order_by", orderBy,
	)

	queryCount := r.db.Model(&shortlink.ShortLink{})
	if userID != "" {
		queryCount = queryCount.Where("user_id = ?", userID)
	}
	if strings.TrimSpace(search) != "" {
		keyword := "%" + strings.ToLower(strings.TrimSpace(search)) + "%"
		queryCount = queryCount.Where(
			"LOWER(short_code) LIKE ? OR LOWER(original_url) LIKE ? OR LOWER(title) LIKE ? OR LOWER(description) LIKE ?",
			keyword, keyword, keyword, keyword,
		)
	}

	// Get total count (without joins for better performance)
	if err := queryCount.Count(&totalCount).Error; err != nil {
		logger.Logger.Error("Failed to count short links", "error", err.Error())
		return nil, apperrors.ErrShortListFailed.WithError(err)
	}

	// Calculate offset and build order clause
	offset := (page - 1) * limit
	orderClause := sort + " " + orderBy

	// Detail is the only association used by the admin response. Avoid loading all
	// views: it can grow without bound and this response does not expose them.
	queryFind := r.db.
		Preload("Detail").
		Order(orderClause).
		Limit(limit).
		Offset(offset)

	if userID != "" {
		queryFind = queryFind.Where("user_id = ?", userID)
	}
	if strings.TrimSpace(search) != "" {
		keyword := "%" + strings.ToLower(strings.TrimSpace(search)) + "%"
		queryFind = queryFind.Where(
			"LOWER(short_code) LIKE ? OR LOWER(original_url) LIKE ? OR LOWER(title) LIKE ? OR LOWER(description) LIKE ?",
			keyword, keyword, keyword, keyword,
		)
	}

	if err := queryFind.Find(&shortLinks).Error; err != nil {
		logger.Logger.Error("Failed to fetch short links", "error", err.Error())
		return nil, apperrors.ErrShortListFailed.WithError(err)
	}

	// Convert to response format with detailed information
	shortLinkResponses := make([]dto.ShortLinkResponse, 0, len(shortLinks))
	for _, link := range shortLinks {
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
		return apperrors.ErrShortGetFailed.WithError(err)
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
		return apperrors.ErrShortDetailFindFailed.WithError(err)
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
		return apperrors.ErrShortBanFailed.WithError(err)
	}

	if err := r.db.Save(&detail).Error; err != nil {
		logger.Logger.Error("Failed to update short link detail",
			"short_link_id", link.ID,
			"error", err.Error(),
		)
		return apperrors.ErrShortBanFailed.WithError(err)
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
		return apperrors.ErrShortGetFailed.WithError(err)
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
		return apperrors.ErrShortDetailFindFailed.WithError(err)
	}

	detail.IsBanned = false
	detail.BannedBy = nil
	detail.BannedReason = ""

	if err := r.db.Save(&link).Error; err != nil {
		logger.Logger.Error("Failed to update short link",
			"short_code", code,
			"error", err.Error(),
		)
		return apperrors.ErrShortRestoreFailed.WithError(err)
	}

	if err := r.db.Save(&detail).Error; err != nil {
		logger.Logger.Error("Failed to update short link detail",
			"short_link_id", link.ID,
			"error", err.Error(),
		)
		return apperrors.ErrShortRestoreFailed.WithError(err)
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
		return apperrors.ErrShortGetFailed.WithError(err)
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
		return apperrors.ErrShortRestoreFailed.WithError(err)
	}

	return nil
}

// func (r *ShortLinkRepository) ResetPasscodeShortLink(code string, oldPasscode, newPasscode int, userID, roleUser string) error {
// 	var link shortlink.ShortLink

// 	if roleUser != "admin" {
// 		err := r.db.Where("short_links.short_code = ? AND short_links.user_id = ?", code, userID).
// 			Joins("LEFT JOIN short_link_details ON short_links.id = short_link_details.short_link_id").
// 			Select("short_links.*, short_link_details.passcode, short_link_details.id as detail_id").
// 			First(&link).Error

// 		if err != nil {
// 			if errors.Is(err, gorm.ErrRecordNotFound) {
// 				return apperrors.ErrShortLinkNotFound
// 			}
// 			logger.Logger.Error("Database error while fetching short link",
// 				"short_code", code,
// 				"error", err.Error(),
// 			)
// 			return apperrors.ErrShortGetFailed.WithError(err)
// 		}

// 		// Validate old passcode
// 		if link.Detail.Passcode != oldPasscode {
// 			return apperrors.ErrPasscodeIncorrect
// 		}

// 		link.Detail.Passcode = newPasscode

// 		// Update the detail record
// 		err = r.db.Model(&shortlink.ShortLinkDetail{}).Where("id = ?", link.Detail.ID).Update("passcode", newPasscode).Error
// 		if err != nil {
// 			logger.Logger.Error("Failed to update passcode",
// 				"short_code", code,
// 				"error", err.Error(),
// 			)
// 			return apperrors.ErrShortResetPasscodeFailed.WithError(err)
// 		}

// 	} else {
// 		err := r.db.Where("short_code = ?", code).
// 			Joins("LEFT JOIN short_link_details ON short_links.id = short_link_details.short_link_id").
// 			Select("short_links.*, short_link_details.passcode, short_link_details.id as detail_id").
// 			First(&link).Error

// 		if err != nil {
// 			if errors.Is(err, gorm.ErrRecordNotFound) {
// 				return apperrors.ErrShortLinkNotFound
// 			}
// 			logger.Logger.Error("Database error while fetching short link",
// 				"short_code", code,
// 				"error", err.Error(),
// 			)
// 			return apperrors.ErrShortGetFailed.WithError(err)
// 		}

// 		link.Detail.Passcode = newPasscode

// 		// Update the detail record
// 		err = r.db.Model(&shortlink.ShortLinkDetail{}).Where("id = ?", link.Detail.ID).Update("passcode", newPasscode).Error
// 		if err != nil {
// 			logger.Logger.Error("Failed to update passcode",
// 				"short_code", code,
// 				"error", err.Error(),
// 			)
// 			return apperrors.ErrShortResetPasscodeFailed.WithError(err)
// 		}
// 	}
// 	return nil
// }

// func (r *ShortLinkRepository) ForgotPasscodeShortLink(code string, newPasscode int, userID string, roleUser string) error {
// 	var link shortlink.ShortLink

// 	if roleUser != "admin" {
// 		err := r.db.Where("short_links.short_code = ? AND short_links.user_id = ?", code, userID).
// 			Joins("LEFT JOIN short_link_details ON short_links.id = short_link_details.short_link_id").
// 			Select("short_links.*, short_link_details.passcode, short_link_details.id as detail_id").
// 			First(&link).Error

// 		if err != nil {
// 			if errors.Is(err, gorm.ErrRecordNotFound) {
// 				return apperrors.ErrShortLinkNotFound
// 			}
// 			logger.Logger.Error("Database error while fetching short link",
// 				"short_code", code,
// 				"error", err.Error(),
// 			)
// 			return apperrors.ErrShortGetFailed.WithError(err)
// 		}

// 	} else {
// 		err := r.db.Where("short_code = ?", code).
// 			Joins("LEFT JOIN short_link_details ON short_links.id = short_link_details.short_link_id").
// 			Select("short_links.*, short_link_details.passcode, short_link_details.id as detail_id").
// 			First(&link).Error

// 		if err != nil {
// 			if errors.Is(err, gorm.ErrRecordNotFound) {
// 				return apperrors.ErrShortLinkNotFound
// 			}
// 			logger.Logger.Error("Database error while fetching short link",
// 				"short_code", code,
// 				"error", err.Error(),
// 			)
// 			return apperrors.ErrShortGetFailed.WithError(err)
// 		}
// 	}

// 	return nil
// }

// func (r *ShortLinkRepository) ValidatePasscodeToken(token string) (*shortlink.ShortLinkDetail, error) {
// 	var detail shortlink.ShortLinkDetail
// 	err := r.db.Where("passcode_token = ?", token).First(&detail).Error
// 	if err != nil {
// 		if errors.Is(err, gorm.ErrRecordNotFound) {
// 			return nil, nil // Token not found
// 		}
// 		logger.Logger.Error("Database error while validating passcode token",
// 			"token", token,
// 			"error", err.Error(),
// 		)
// 		return nil, apperrors.ErrShortValidateTokenFailed.WithError(err)
// 	}
// 	// Optionally, check expiry
// 	if !detail.PasscodeTokenExpiresAt.IsZero() && detail.PasscodeTokenExpiresAt.Before(time.Now()) {
// 		return nil, nil // Token expired
// 	}
// 	return &detail, nil // Token is valid
// }

// func (r *ShortLinkRepository) setPasscodeToken(userID string, linkID uint, token string, expiresAt time.Time) error {
// 	var detail shortlink.ShortLinkDetail

// 	err := r.db.Where("short_link_id = ?", linkID).First(&detail).Error
// 	if err != nil {
// 		if errors.Is(err, gorm.ErrRecordNotFound) {
// 			return apperrors.ErrShortLinkNotFound
// 		}

// 		logger.Logger.Error("Database error while fetching short link detail",
// 			"short_link_id", linkID,
// 			"error", err.Error(),
// 		)
// 		return apperrors.ErrShortDetailFindFailed.WithError(err)
// 	}
// 	detail.PasscodeTokenExpiresAt = expiresAt

// 	if err := r.db.Save(&detail).Error; err != nil {
// 		logger.Logger.Error("Failed to set passcode token",
// 			"short_link_id", linkID,
// 			"error", err.Error(),
// 		)
// 		return apperrors.ErrShortUpdateFailed.WithError(err)
// 	}

// 	return nil
// }

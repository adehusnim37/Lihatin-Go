package shortlink

import (
	"fmt"
	"strconv"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/gin-gonic/gin"
)

func (c *Controller) Create(ctx *gin.Context) {
	var req dto.ShortLinkRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		// Use new validation error handler
		utils.SendValidationError(ctx, err, &req)
		return
	}

	userID := ctx.GetString("user_id")
	userEmail := ctx.GetString("email")
	userName := ctx.GetString("username")

	// Debug logging
	utils.Logger.Info("Request received",
		"is_bulky", req.IsBulky,
		"has_link", req.Link != nil,
		"links_count", len(req.Links),
		"user_id", userID,
	)

	// Bulk mode: is_bulky true & links >= 1
	if req.IsBulky {
		if len(req.Links) == 0 {
			utils.SendValidationError(ctx, fmt.Errorf("links wajib diisi untuk mode bulk"), &req)
			return
		}
		c.handleBulkCreation(ctx, req.Links, userID, userEmail, userName)
		return
	}

	// Single mode: is_bulky false
	if !req.IsBulky {
		if len(req.Links) == 1 {
			c.handleSingleCreation(ctx, req.Links[0], userID, userEmail, userName)
			return
		}
		if len(req.Links) > 1 {
			utils.SendValidationError(ctx, fmt.Errorf("gunakan 1 link saja untuk mode single"), &req)
			return
		}
		if req.Link != nil {
			c.handleSingleCreation(ctx, *req.Link, userID, userEmail, userName)
			return
		}
	}

	// Final fallback: neither bulk nor single data provided
	utils.Logger.Error("No valid request data provided", "user_id", userID)
	utils.SendValidationError(ctx, fmt.Errorf("request harus berisi 'link' untuk single atau 'links' untuk bulk"), &req)
}

// Handle single short link creation
func (c *Controller) handleSingleCreation(ctx *gin.Context, req dto.CreateShortLinkRequest, userID, userEmail, userName string) {
	link := dto.CreateShortLinkRequest{
		UserID:      userID,
		OriginalURL: req.OriginalURL,
		Title:       req.Title,
		Description: req.Description,
		CustomCode:  req.CustomCode,
		Passcode:    req.Passcode,
		ExpiresAt:   req.ExpiresAt,
	}

	// Call repository to create short link
	createdLink, createdDetail, err := c.repo.CreateShortLink(&link)
	if err != nil {
		utils.Logger.Error("Failed to create short link",
			"error", err.Error(),
		)
		// Use universal error handler
		utils.HandleError(ctx, err, userID)
		return
	}

	// Build response using created data
	response := dto.ShortLinkResponse{
		ID:          createdLink.ID,
		UserID:      createdLink.UserID,
		ShortCode:   createdLink.ShortCode,
		OriginalURL: createdLink.OriginalURL,
		Title:       createdLink.Title,
		Description: createdLink.Description,
		IsActive:    createdLink.IsActive,
		ExpiresAt:   createdLink.ExpiresAt,
		CreatedAt:   createdLink.CreatedAt,
		UpdatedAt:   createdLink.UpdatedAt,
	}

	utils.Logger.Info("Short link created successfully",
		"short_code", response.ShortCode,
		"user_id", userID,
	)

	// Send email notification in goroutine if user is authenticated
	if userID != "" && userEmail != "" && userName != "" {
		go func() {
			// Build complete short URL
			var fullShortURL string
			backendURL := utils.GetRequiredEnv(utils.EnvBackendURL)

			// Format dates
			createdAt := createdLink.CreatedAt.Format("January 2, 2006 at 3:04 PM MST")
			var expiresAt string
			if createdLink.ExpiresAt != nil {
				expiresAt = createdLink.ExpiresAt.Format("January 2, 2006 at 3:04 PM MST")
			} else {
				expiresAt = "Never"
			}

			// Format passcode
			var passcode string = "-"
			if createdDetail.Passcode != 0 {
				passcode = fmt.Sprintf("%d", createdDetail.Passcode)
				fullShortURL = fmt.Sprintf("%s/%s?passcode=%s", backendURL, createdLink.ShortCode, strconv.Itoa(createdDetail.Passcode))
			} else {
				fullShortURL = fmt.Sprintf("%s/%s", backendURL, createdLink.ShortCode)
			}

			if createdLink.Title == "" {
				createdLink.Title = "No Title Provided For This Link."
			}

			// Send email notification
			err := c.emailService.SendInformationShortCreate(
				userEmail,               // toEmail
				userName,                // toName
				fullShortURL,            // url
				createdLink.Title,       // title
				expiresAt,               // expires_at
				createdAt,               // created_at
				passcode,                // passcode
				createdLink.OriginalURL, // urlOrigin
				createdLink.ShortCode,   // shortCode
			)

			if err != nil {
				utils.Logger.Error("Failed to send email notification", "error", err.Error())
			} else {
				utils.Logger.Info("Email notification sent successfully",
					"user_id", userID,
					"short_code", createdLink.ShortCode,
					"email", userEmail,
				)
			}
		}()
	}

	utils.SendCreatedResponse(ctx, response, "Short link created successfully")
}

// Handle bulk short links creation
func (c *Controller) handleBulkCreation(ctx *gin.Context, links []dto.CreateShortLinkRequest, userID, userEmail, userName string) {
	// For bulk, we need multiple link data

	// Set user ID for all links
	for i := range links {
		links[i].UserID = userID
	}

	// Create bulk short links
	createdLinks, createdDetails, err := c.repo.CreateBulkShortLinks(links)
	if err != nil {
		utils.Logger.Error("Failed to create bulk short links", "error", err.Error())
		utils.HandleError(ctx, err, userID)
		return
	}

	// Build bulk response
	responses := make([]dto.ShortLinkResponse, len(createdLinks))
	for i, link := range createdLinks {
		responses[i] = dto.ShortLinkResponse{
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
	}

	// Send single summary email for bulk creation (async)
	if userID != "" && userEmail != "" && userName != "" {
		go func() {
			err := c.emailService.SendBulkCreationSummary(
				userEmail,
				userName,
				createdLinks,
				createdDetails,
			)
			if err != nil {
				utils.Logger.Error("Failed to send bulk creation email", "error", err.Error())
			}
		}()
	}

	utils.Logger.Info("Bulk short links created successfully",
		"count", len(createdLinks),
		"user_id", userID,
	)

	utils.SendCreatedResponse(ctx, map[string]interface{}{
		"links":       responses,
		"total_count": len(responses),
		"message":     fmt.Sprintf("Successfully created %d short links", len(responses)),
	}, "Bulk short links created successfully")
}

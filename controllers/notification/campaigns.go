package notification

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/adehusnim37/lihatin-go/dto"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/identifier"
	"github.com/adehusnim37/lihatin-go/models/user"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const maxCampaignBodyLength = 20000

func (c *Controller) CreateCampaign(ctx *gin.Context) {
	var request dto.CreatePromotionalCampaignRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		httputil.SendErrorResponse(ctx, http.StatusBadRequest, "INVALID_CAMPAIGN_PAYLOAD", "Invalid promotional campaign payload", "campaign")
		return
	}

	request.Name = strings.TrimSpace(request.Name)
	request.Subject = strings.TrimSpace(request.Subject)
	request.Preheader = strings.TrimSpace(request.Preheader)
	request.Body = strings.TrimSpace(request.Body)
	request.ImageURL = strings.TrimSpace(request.ImageURL)
	request.ImageAlt = strings.TrimSpace(request.ImageAlt)
	request.CTALabel = strings.TrimSpace(request.CTALabel)
	request.CTAURL = strings.TrimSpace(request.CTAURL)
	if errMessage := validateCampaignContent(request.Name, request.Subject, request.Body, request.CTALabel, request.CTAURL); errMessage != "" {
		httputil.SendErrorResponse(ctx, http.StatusBadRequest, "INVALID_CAMPAIGN_CONTENT", errMessage, "campaign")
		return
	}
	if errMessage := validateCampaignImageURL(request.ImageURL); errMessage != "" {
		httputil.SendErrorResponse(ctx, http.StatusBadRequest, "INVALID_CAMPAIGN_IMAGE", errMessage, "image_url")
		return
	}

	campaign := user.PromotionalCampaign{
		ID:        identifier.NewUUIDV7(),
		Name:      request.Name,
		Subject:   request.Subject,
		Preheader: request.Preheader,
		Body:      request.Body,
		ImageURL:  request.ImageURL,
		ImageAlt:  request.ImageAlt,
		CTALabel:  request.CTALabel,
		CTAURL:    request.CTAURL,
		Status:    user.PromotionalCampaignDraft,
		CreatedBy: strings.TrimSpace(ctx.GetString("user_id")),
	}
	if err := c.GormDB.WithContext(ctx).Create(&campaign).Error; err != nil {
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "CAMPAIGN_CREATE_FAILED", "Failed to create promotional campaign", "campaign")
		return
	}

	httputil.SendCreatedResponse(ctx, campaign, "Promotional campaign created successfully")
}

func (c *Controller) ListCampaigns(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	var total int64
	if err := c.GormDB.WithContext(ctx).Model(&user.PromotionalCampaign{}).Count(&total).Error; err != nil {
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "CAMPAIGNS_READ_FAILED", "Failed to retrieve promotional campaigns", "campaigns")
		return
	}

	var campaigns []user.PromotionalCampaign
	if err := c.GormDB.WithContext(ctx).
		Order("created_at DESC").
		Limit(limit).
		Offset((page - 1) * limit).
		Find(&campaigns).Error; err != nil {
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "CAMPAIGNS_READ_FAILED", "Failed to retrieve promotional campaigns", "campaigns")
		return
	}

	totalPages := int((total + int64(limit) - 1) / int64(limit))
	httputil.SendOKResponse(ctx, gin.H{
		"campaigns":   campaigns,
		"page":        page,
		"limit":       limit,
		"total":       total,
		"total_pages": totalPages,
	}, "Promotional campaigns retrieved successfully")
}

func (c *Controller) GetCampaign(ctx *gin.Context) {
	campaign, ok := c.loadCampaign(ctx)
	if !ok {
		return
	}
	httputil.SendOKResponse(ctx, campaign, "Promotional campaign retrieved successfully")
}

func (c *Controller) ListCampaignDeliveries(ctx *gin.Context) {
	if _, ok := c.loadCampaign(ctx); !ok {
		return
	}

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	status := strings.ToLower(strings.TrimSpace(ctx.Query("status")))
	if status != "" && status != "pending" && status != "sending" &&
		status != "sent" && status != "failed" && status != "skipped" {
		httputil.SendErrorResponse(ctx, http.StatusBadRequest, "INVALID_DELIVERY_STATUS", "Invalid delivery status filter", "status")
		return
	}

	query := c.GormDB.WithContext(ctx).
		Model(&user.PromotionalEmailDelivery{}).
		Where("campaign_id = ?", ctx.Param("id"))
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "CAMPAIGN_DELIVERIES_READ_FAILED", "Failed to retrieve campaign deliveries", "deliveries")
		return
	}

	var deliveries []user.PromotionalEmailDelivery
	if err := query.
		Order("created_at DESC").
		Limit(limit).
		Offset((page - 1) * limit).
		Find(&deliveries).Error; err != nil {
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "CAMPAIGN_DELIVERIES_READ_FAILED", "Failed to retrieve campaign deliveries", "deliveries")
		return
	}

	items := make([]dto.PromotionalDeliveryResponse, len(deliveries))
	for index, delivery := range deliveries {
		items[index] = dto.PromotionalDeliveryResponse{
			ID:           delivery.ID,
			UserID:       delivery.UserID,
			Email:        delivery.Email,
			Status:       string(delivery.Status),
			ErrorMessage: delivery.ErrorMessage,
			SentAt:       delivery.SentAt,
			CreatedAt:    delivery.CreatedAt,
			UpdatedAt:    delivery.UpdatedAt,
		}
	}

	totalPages := int((total + int64(limit) - 1) / int64(limit))
	httputil.SendOKResponse(ctx, dto.PromotionalDeliveryListResponse{
		Deliveries: items,
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
		Status:     status,
	}, "Campaign deliveries retrieved successfully")
}

func (c *Controller) UpdateCampaign(ctx *gin.Context) {
	campaign, ok := c.loadCampaign(ctx)
	if !ok {
		return
	}
	if campaign.Status != user.PromotionalCampaignDraft {
		httputil.SendErrorResponse(ctx, http.StatusConflict, "CAMPAIGN_NOT_EDITABLE", "Only draft campaigns can be edited", "status")
		return
	}

	var request dto.UpdatePromotionalCampaignRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		httputil.SendErrorResponse(ctx, http.StatusBadRequest, "INVALID_CAMPAIGN_PAYLOAD", "Invalid promotional campaign payload", "campaign")
		return
	}

	updates := make(map[string]any)
	name, subject, body := campaign.Name, campaign.Subject, campaign.Body
	ctaLabel, ctaURL := campaign.CTALabel, campaign.CTAURL
	imageURL := campaign.ImageURL
	if request.Name != nil {
		name = strings.TrimSpace(*request.Name)
		updates["name"] = name
	}
	if request.Subject != nil {
		subject = strings.TrimSpace(*request.Subject)
		updates["subject"] = subject
	}
	if request.Preheader != nil {
		updates["preheader"] = strings.TrimSpace(*request.Preheader)
	}
	if request.Body != nil {
		body = strings.TrimSpace(*request.Body)
		updates["body"] = body
	}
	if request.ImageURL != nil {
		imageURL = strings.TrimSpace(*request.ImageURL)
		updates["image_url"] = imageURL
	}
	if request.ImageAlt != nil {
		updates["image_alt"] = strings.TrimSpace(*request.ImageAlt)
	}
	if request.CTALabel != nil {
		ctaLabel = strings.TrimSpace(*request.CTALabel)
		updates["cta_label"] = ctaLabel
	}
	if request.CTAURL != nil {
		ctaURL = strings.TrimSpace(*request.CTAURL)
		updates["cta_url"] = ctaURL
	}
	if len(updates) == 0 {
		httputil.SendOKResponse(ctx, campaign, "No campaign changes were provided")
		return
	}
	if errMessage := validateCampaignContent(name, subject, body, ctaLabel, ctaURL); errMessage != "" {
		httputil.SendErrorResponse(ctx, http.StatusBadRequest, "INVALID_CAMPAIGN_CONTENT", errMessage, "campaign")
		return
	}
	if errMessage := validateCampaignImageURL(imageURL); errMessage != "" {
		httputil.SendErrorResponse(ctx, http.StatusBadRequest, "INVALID_CAMPAIGN_IMAGE", errMessage, "image_url")
		return
	}

	updates["updated_at"] = time.Now().UTC()
	if err := c.GormDB.WithContext(ctx).
		Model(&user.PromotionalCampaign{}).
		Where("id = ? AND status = ?", campaign.ID, user.PromotionalCampaignDraft).
		Updates(updates).Error; err != nil {
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "CAMPAIGN_UPDATE_FAILED", "Failed to update promotional campaign", "campaign")
		return
	}
	updatedCampaign, ok := c.loadCampaign(ctx)
	if !ok {
		return
	}
	httputil.SendOKResponse(ctx, updatedCampaign, "Promotional campaign updated successfully")
}

func (c *Controller) ScheduleCampaign(ctx *gin.Context) {
	var request dto.SchedulePromotionalCampaignRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		httputil.SendErrorResponse(ctx, http.StatusBadRequest, "INVALID_CAMPAIGN_SCHEDULE", "Invalid campaign schedule", "scheduled_at")
		return
	}

	scheduledAt := time.Now().UTC()
	if request.ScheduledAt != nil && request.ScheduledAt.After(scheduledAt) {
		scheduledAt = request.ScheduledAt.UTC()
	}
	now := time.Now().UTC()
	result := c.GormDB.WithContext(ctx).
		Model(&user.PromotionalCampaign{}).
		Where("id = ? AND status IN ?", ctx.Param("id"), []user.PromotionalCampaignStatus{
			user.PromotionalCampaignDraft,
			user.PromotionalCampaignFailed,
		}).
		Updates(map[string]any{
			"status":       user.PromotionalCampaignScheduled,
			"scheduled_at": scheduledAt,
			"started_at":   nil,
			"completed_at": nil,
			"updated_at":   now,
		})
	if result.Error != nil {
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "CAMPAIGN_SCHEDULE_FAILED", "Failed to schedule promotional campaign", "campaign")
		return
	}
	if result.RowsAffected == 0 {
		httputil.SendErrorResponse(ctx, http.StatusConflict, "CAMPAIGN_NOT_SCHEDULABLE", "Campaign must be a draft or failed campaign", "status")
		return
	}

	campaign, ok := c.loadCampaign(ctx)
	if !ok {
		return
	}
	httputil.SendOKResponse(ctx, campaign, "Promotional campaign scheduled successfully")
}

func (c *Controller) CancelCampaign(ctx *gin.Context) {
	now := time.Now().UTC()
	result := c.GormDB.WithContext(ctx).
		Model(&user.PromotionalCampaign{}).
		Where("id = ? AND status = ?", ctx.Param("id"), user.PromotionalCampaignScheduled).
		Updates(map[string]any{
			"status":     user.PromotionalCampaignCancelled,
			"updated_at": now,
		})
	if result.Error != nil {
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "CAMPAIGN_CANCEL_FAILED", "Failed to cancel promotional campaign", "campaign")
		return
	}
	if result.RowsAffected == 0 {
		httputil.SendErrorResponse(ctx, http.StatusConflict, "CAMPAIGN_NOT_CANCELLABLE", "Only scheduled campaigns can be cancelled", "status")
		return
	}

	campaign, ok := c.loadCampaign(ctx)
	if !ok {
		return
	}
	httputil.SendOKResponse(ctx, campaign, "Promotional campaign cancelled successfully")
}

func (c *Controller) DeleteCampaign(ctx *gin.Context) {
	result := c.GormDB.WithContext(ctx).
		Where("id = ? AND status IN ?", ctx.Param("id"), []user.PromotionalCampaignStatus{
			user.PromotionalCampaignDraft,
			user.PromotionalCampaignCancelled,
		}).
		Delete(&user.PromotionalCampaign{})
	if result.Error != nil {
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "CAMPAIGN_DELETE_FAILED", "Failed to delete promotional campaign", "campaign")
		return
	}
	if result.RowsAffected == 0 {
		httputil.SendErrorResponse(ctx, http.StatusConflict, "CAMPAIGN_NOT_DELETABLE", "Only draft or cancelled campaigns can be deleted", "status")
		return
	}

	httputil.SendOKResponse(ctx, nil, "Promotional campaign deleted successfully")
}

func (c *Controller) loadCampaign(ctx *gin.Context) (*user.PromotionalCampaign, bool) {
	var campaign user.PromotionalCampaign
	err := c.GormDB.WithContext(ctx).Where("id = ?", ctx.Param("id")).First(&campaign).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			httputil.SendErrorResponse(ctx, http.StatusNotFound, "CAMPAIGN_NOT_FOUND", "Promotional campaign was not found", "id")
		} else {
			httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "CAMPAIGN_READ_FAILED", "Failed to retrieve promotional campaign", "campaign")
		}
		return nil, false
	}
	return &campaign, true
}

func validateCampaignContent(name, subject, body, ctaLabel, ctaURL string) string {
	if name == "" {
		return "Campaign name is required"
	}
	if subject == "" {
		return "Campaign subject is required"
	}
	if strings.ContainsAny(subject, "\r\n") {
		return "Campaign subject must be a single line"
	}
	if body == "" {
		return "Campaign body is required"
	}
	if utf8.RuneCountInString(body) > maxCampaignBodyLength {
		return "Campaign body must not exceed 20,000 characters"
	}
	if (ctaLabel == "") != (ctaURL == "") {
		return "CTA label and CTA URL must be provided together"
	}
	return ""
}

func validateCampaignImageURL(imageURL string) string {
	if imageURL == "" {
		return ""
	}
	parsed, err := url.ParseRequestURI(imageURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "Campaign image URL must be a valid HTTP or HTTPS URL"
	}
	return ""
}

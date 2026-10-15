package notification

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/adehusnim37/lihatin-go/dto"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	"github.com/gin-gonic/gin"
)

const (
	defaultAnnouncementPageLimit = 20
	maxAnnouncementPageLimit     = 100
)

var allowedAnnouncementStatuses = map[string]struct{}{
	"draft":     {},
	"published": {},
	"archived":  {},
}

func (c *Controller) PendingInAppAnnouncements(ctx *gin.Context) {
	announcements, err := c.inAppService.Pending(ctx.Request.Context(), ctx.GetString("user_id"))
	if err != nil {
		httputil.HandleError(ctx, err, "announcements")
		return
	}
	httputil.SendOKResponse(ctx, announcements, "Pending announcements retrieved")
}

// ListInAppAnnouncementInbox returns the account's full notification inbox,
// including announcements it already read.
func (c *Controller) ListInAppAnnouncementInbox(ctx *gin.Context) {
	page := parsePositiveInt(ctx.Query("page"), 1)
	limit := parsePositiveInt(ctx.Query("limit"), defaultAnnouncementPageLimit)
	if limit > maxAnnouncementPageLimit {
		limit = maxAnnouncementPageLimit
	}

	result, err := c.inAppService.Inbox(ctx.Request.Context(), ctx.GetString("user_id"), page, limit)
	if err != nil {
		httputil.HandleError(ctx, err, "announcements")
		return
	}
	httputil.SendOKResponse(ctx, result, "Announcement inbox retrieved")
}

func (c *Controller) InAppAnnouncementUnreadCount(ctx *gin.Context) {
	count, err := c.inAppService.UnreadCount(ctx.Request.Context(), ctx.GetString("user_id"))
	if err != nil {
		httputil.HandleError(ctx, err, "announcements")
		return
	}
	httputil.SendOKResponse(
		ctx,
		dto.InAppAnnouncementUnreadCountResponse{UnreadCount: count},
		"Unread announcement count retrieved",
	)
}

func (c *Controller) MarkInAppAnnouncementRead(ctx *gin.Context) {
	err := c.inAppService.MarkRead(ctx.Request.Context(), ctx.GetString("user_id"), ctx.Param("id"))
	if err != nil {
		httputil.HandleError(ctx, err, "announcement")
		return
	}
	httputil.SendOKResponse(ctx, gin.H{"id": ctx.Param("id")}, "Announcement marked as read")
}

// MarkInAppAnnouncementUnread lets an account read an announcement again.
func (c *Controller) MarkInAppAnnouncementUnread(ctx *gin.Context) {
	err := c.inAppService.ReadAgain(ctx.Request.Context(), ctx.GetString("user_id"), ctx.Param("id"))
	if err != nil {
		httputil.HandleError(ctx, err, "announcement")
		return
	}
	httputil.SendOKResponse(ctx, gin.H{"id": ctx.Param("id")}, "Announcement marked as unread")
}

func (c *Controller) MarkAllInAppAnnouncementsRead(ctx *gin.Context) {
	marked, err := c.inAppService.MarkAllRead(ctx.Request.Context(), ctx.GetString("user_id"))
	if err != nil {
		httputil.HandleError(ctx, err, "announcements")
		return
	}
	httputil.SendOKResponse(
		ctx,
		dto.MarkInAppAnnouncementsReadResponse{MarkedCount: marked},
		"All announcements marked as read",
	)
}

func (c *Controller) CreateInAppAnnouncement(ctx *gin.Context) {
	body := dto.CreateInAppAnnouncementRequest{}
	if err := ctx.ShouldBindJSON(&body); err != nil {
		validator.SendValidationError(ctx, err, &body)
		return
	}

	body.Title = strings.TrimSpace(body.Title)
	body.Body = strings.TrimSpace(body.Body)
	body.Href = strings.TrimSpace(body.Href)

	announcement, err := c.inAppService.CreateAnnouncement(
		ctx.Request.Context(),
		body.Title,
		body.Body,
		body.Href,
		ctx.GetString("user_id"),
		body.Status,
		body.StartAt,
		body.EndsAt,
		body.PublishedAt,
	)
	if err != nil {
		httputil.HandleError(ctx, err, "announcement")
		return
	}
	httputil.SendCreatedResponse(ctx, announcement, "Announcement created successfully")
}

func (c *Controller) ListInAppAnnouncements(ctx *gin.Context) {
	status := strings.ToLower(strings.TrimSpace(ctx.Query("status")))
	if status != "" {
		if _, ok := allowedAnnouncementStatuses[status]; !ok {
			httputil.SendErrorResponse(
				ctx,
				http.StatusBadRequest,
				"INVALID_ANNOUNCEMENT_STATUS",
				"Status filter must be draft, published, or archived",
				"status",
			)
			return
		}
	}

	page := parsePositiveInt(ctx.Query("page"), 1)
	limit := parsePositiveInt(ctx.Query("limit"), defaultAnnouncementPageLimit)
	if limit > maxAnnouncementPageLimit {
		limit = maxAnnouncementPageLimit
	}

	result, err := c.inAppService.ListForAdmin(ctx.Request.Context(), status, page, limit)
	if err != nil {
		httputil.HandleError(ctx, err, "announcements")
		return
	}
	httputil.SendOKResponse(ctx, result, "Announcements retrieved successfully")
}

func (c *Controller) GetInAppAnnouncement(ctx *gin.Context) {
	announcement, err := c.inAppService.GetAnnouncement(ctx.Request.Context(), ctx.Param("id"))
	if err != nil {
		httputil.HandleError(ctx, err, "announcement")
		return
	}
	httputil.SendOKResponse(ctx, announcement, "Announcement retrieved successfully")
}

// ListInAppAnnouncementAudience reports which accounts dismissed an
// announcement. Use `status=read` (default) or `status=unread`.
func (c *Controller) ListInAppAnnouncementAudience(ctx *gin.Context) {
	status := strings.ToLower(strings.TrimSpace(ctx.Query("status")))
	if status == "" {
		status = "read"
	}
	if status != "read" && status != "unread" {
		httputil.SendErrorResponse(
			ctx,
			http.StatusBadRequest,
			"INVALID_AUDIENCE_STATUS",
			"Audience status must be read or unread",
			"status",
		)
		return
	}

	page := parsePositiveInt(ctx.Query("page"), 1)
	limit := parsePositiveInt(ctx.Query("limit"), defaultAnnouncementPageLimit)
	if limit > maxAnnouncementPageLimit {
		limit = maxAnnouncementPageLimit
	}

	result, err := c.inAppService.Audience(
		ctx.Request.Context(),
		ctx.Param("id"),
		status == "read",
		page,
		limit,
	)
	if err != nil {
		httputil.HandleError(ctx, err, "announcement")
		return
	}
	httputil.SendOKResponse(ctx, result, "Announcement audience retrieved successfully")
}

func (c *Controller) UpdateInAppAnnouncement(ctx *gin.Context) {
	body := dto.UpdateInAppAnnouncementRequest{}
	if err := ctx.ShouldBindJSON(&body); err != nil {
		validator.SendValidationError(ctx, err, &body)
		return
	}

	body.Title = strings.TrimSpace(body.Title)
	body.Body = strings.TrimSpace(body.Body)
	body.Href = strings.TrimSpace(body.Href)

	announcement, err := c.inAppService.UpdateAnnouncement(
		ctx.Request.Context(),
		ctx.Param("id"),
		body.Title,
		body.Body,
		body.Href,
		body.StartAt,
		body.EndsAt,
	)
	if err != nil {
		httputil.HandleError(ctx, err, "announcement")
		return
	}
	httputil.SendOKResponse(ctx, announcement, "Announcement updated successfully")
}

func (c *Controller) PublishInAppAnnouncement(ctx *gin.Context) {
	announcement, err := c.inAppService.PublishAnnouncement(ctx.Request.Context(), ctx.Param("id"))
	if err != nil {
		httputil.HandleError(ctx, err, "announcement")
		return
	}
	httputil.SendOKResponse(ctx, announcement, "Announcement published successfully")
}

func (c *Controller) ArchiveInAppAnnouncement(ctx *gin.Context) {
	if err := c.inAppService.ArchiveAnnouncement(ctx.Request.Context(), ctx.Param("id")); err != nil {
		httputil.HandleError(ctx, err, "announcement")
		return
	}
	httputil.SendOKResponse(ctx, gin.H{"id": ctx.Param("id")}, "Announcement archived successfully")
}

func (c *Controller) DeleteInAppAnnouncement(ctx *gin.Context) {
	announcement, err := c.inAppService.DeleteAnnouncement(ctx.Request.Context(), ctx.Param("id"))
	if err != nil {
		httputil.HandleError(ctx, err, "announcement")
		return
	}
	httputil.SendOKResponse(ctx, announcement, "Announcement deleted successfully")
}

func parsePositiveInt(raw string, fallback int) int {
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return fallback
	}
	return value
}

package notification

import (
	"github.com/adehusnim37/lihatin-go/dto"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	"github.com/gin-gonic/gin"
)

func (c *Controller) PendingInAppAnnouncements(ctx *gin.Context) {
	announcements, err := c.inAppService.Pending(ctx.Request.Context(), ctx.GetString("user_id"))
	if err != nil {
		httputil.HandleError(ctx, err, "announcements")
		return
	}
	httputil.SendOKResponse(ctx, announcements, "Pending announcements retrieved")
}

func (c *Controller) MarkInAppAnnouncementRead(ctx *gin.Context) {
	err := c.inAppService.MarkRead(ctx.Request.Context(), ctx.GetString("user_id"), ctx.Param("id"))
	if err != nil {
		httputil.HandleError(ctx, err, "announcement")
		return
	}
	httputil.SendOKResponse(ctx, gin.H{"id": ctx.Param("id")}, "Announcement marked as read")
}

func (c *Controller) CreateInAppAnnouncement(ctx *gin.Context) {
	body := dto.CreateInAppAnnouncementRequest{}
	if err := ctx.ShouldBindJSON(&body); err != nil {
		validator.SendValidationError(ctx, err, &body)
		return
	}
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

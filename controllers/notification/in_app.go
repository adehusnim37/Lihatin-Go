package notification

import (
	"errors"
	"net/http"

	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/notifications"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (c *Controller) PendingInAppAnnouncements(ctx *gin.Context) {
	announcements, err := notifications.PendingInAppAnnouncements(ctx.Request.Context(), c.GormDB, ctx.GetString("user_id"))
	if err != nil {
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "ANNOUNCEMENTS_READ_FAILED", "Failed to load announcements", "announcements")
		return
	}
	httputil.SendOKResponse(ctx, announcements, "Pending announcements retrieved")
}

func (c *Controller) MarkInAppAnnouncementRead(ctx *gin.Context) {
	err := notifications.MarkInAppAnnouncementRead(ctx.Request.Context(), c.GormDB, ctx.GetString("user_id"), ctx.Param("id"))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		httputil.SendErrorResponse(ctx, http.StatusNotFound, "ANNOUNCEMENT_NOT_FOUND", "Announcement not found", "id")
		return
	}
	if err != nil {
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "ANNOUNCEMENT_READ_FAILED", "Failed to mark announcement as read", "announcement")
		return
	}
	httputil.SendOKResponse(ctx, gin.H{"id": ctx.Param("id")}, "Announcement marked as read")
}

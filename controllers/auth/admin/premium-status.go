package admin

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/adehusnim37/lihatin-go/dto"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	"github.com/adehusnim37/lihatin-go/models/user"
	"github.com/gin-gonic/gin"
)

func (c *Controller) RevokePremiumAccess(ctx *gin.Context) {
	userID := strings.TrimSpace(ctx.Param("id"))
	if userID == "" {
		httputil.SendErrorResponse(ctx, http.StatusBadRequest, "USER_ID_REQUIRED", "User ID is required", "user_id")
		return
	}

	var req dto.AdminRevokePremiumRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		validator.SendValidationError(ctx, err, &req)
		return
	}

	adminID := strings.TrimSpace(ctx.GetString("user_id"))
	adminRole := strings.TrimSpace(ctx.GetString("role"))

	updatedUser, err := c.repo.GetUserAdminRepository().RevokePremiumAccess(
		userID,
		req.Reason,
		req.RevokeType,
		adminID,
		adminRole,
	)
	if err != nil {
		httputil.HandleError(ctx, err, userID)
		return
	}

	go c.sendPremiumRevokedEmail(updatedUser, req.Reason, req.RevokeType)

	response := dto.AdminPremiumStatusMutationResponse{
		UserID:                   updatedUser.ID,
		IsPremium:                updatedUser.IsPremium,
		Role:                     updatedUser.Role,
		PremiumRevokeType:        normalizePremiumRevokeType(updatedUser.PremiumRevokeType),
		PremiumRevokedAt:         updatedUser.PremiumRevokedAt,
		PremiumReactivatedAt:     updatedUser.PremiumReactivatedAt,
		PremiumRevokedReason:     updatedUser.PremiumRevokedReason,
		PremiumReactivatedReason: updatedUser.PremiumReactivatedReason,
	}

	httputil.SendOKResponse(ctx, response, "Premium access revoked successfully")
}

func (c *Controller) ReactivatePremiumAccess(ctx *gin.Context) {
	userID := strings.TrimSpace(ctx.Param("id"))
	if userID == "" {
		httputil.SendErrorResponse(ctx, http.StatusBadRequest, "USER_ID_REQUIRED", "User ID is required", "user_id")
		return
	}

	var req dto.AdminReactivatePremiumRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		validator.SendValidationError(ctx, err, &req)
		return
	}

	adminID := strings.TrimSpace(ctx.GetString("user_id"))
	adminRole := strings.TrimSpace(ctx.GetString("role"))

	updatedUser, err := c.repo.GetUserAdminRepository().ReactivatePremiumAccess(
		userID,
		req.Reason,
		adminID,
		adminRole,
		req.OverridePermanent,
	)
	if err != nil {
		httputil.HandleError(ctx, err, userID)
		return
	}

	go c.sendPremiumReactivatedEmail(updatedUser, req.Reason)

	response := dto.AdminPremiumStatusMutationResponse{
		UserID:                   updatedUser.ID,
		IsPremium:                updatedUser.IsPremium,
		Role:                     updatedUser.Role,
		PremiumRevokeType:        normalizePremiumRevokeType(updatedUser.PremiumRevokeType),
		PremiumRevokedAt:         updatedUser.PremiumRevokedAt,
		PremiumReactivatedAt:     updatedUser.PremiumReactivatedAt,
		PremiumRevokedReason:     updatedUser.PremiumRevokedReason,
		PremiumReactivatedReason: updatedUser.PremiumReactivatedReason,
	}

	httputil.SendOKResponse(ctx, response, "Premium access reactivated successfully")
}

func (c *Controller) GetPremiumStatusEvents(ctx *gin.Context) {
	userID := strings.TrimSpace(ctx.Param("id"))
	if userID == "" {
		httputil.SendErrorResponse(ctx, http.StatusBadRequest, "USER_ID_REQUIRED", "User ID is required", "user_id")
		return
	}

	limit := 20
	if rawLimit := strings.TrimSpace(ctx.DefaultQuery("limit", "20")); rawLimit != "" {
		parsedLimit, err := strconv.Atoi(rawLimit)
		if err != nil || parsedLimit < 1 || parsedLimit > 100 {
			httputil.SendErrorResponse(
				ctx,
				http.StatusBadRequest,
				"INVALID_LIMIT",
				"Limit must be an integer between 1 and 100",
				"limit",
			)
			return
		}
		limit = parsedLimit
	}

	if _, err := c.repo.GetUserRepository().GetUserByID(userID); err != nil {
		httputil.HandleError(ctx, err, userID)
		return
	}

	events, err := c.repo.GetUserAdminRepository().GetPremiumStatusEvents(userID, limit)
	if err != nil {
		httputil.HandleError(ctx, err, userID)
		return
	}

	items := make([]dto.AdminPremiumStatusEventResponse, len(events))
	for i, item := range events {
		items[i] = dto.AdminPremiumStatusEventResponse{
			ID:          item.ID,
			UserID:      item.UserID,
			Action:      string(item.Action),
			OldStatus:   item.OldStatus,
			NewStatus:   item.NewStatus,
			OldRole:     item.OldRole,
			NewRole:     item.NewRole,
			RevokeType:  item.RevokeType,
			Reason:      item.Reason,
			ChangedBy:   item.ChangedBy,
			ChangedRole: item.ChangedRole,
			CreatedAt:   item.CreatedAt,
		}
	}

	httputil.SendOKResponse(ctx, dto.AdminPremiumStatusEventsListResponse{
		UserID: userID,
		Total:  len(items),
		Items:  items,
	}, "Premium status events retrieved successfully")
}

func (c *Controller) sendPremiumRevokedEmail(targetUser *user.User, reason, revokeType string) {
	if targetUser == nil || strings.TrimSpace(targetUser.Email) == "" {
		return
	}

	targetUserName := strings.TrimSpace(targetUser.FirstName + " " + targetUser.LastName)
	if targetUserName == "" {
		targetUserName = strings.TrimSpace(targetUser.Username)
	}

	revokedAt := time.Now()
	if targetUser.PremiumRevokedAt != nil {
		revokedAt = *targetUser.PremiumRevokedAt
	}

	if err := c.emailService.SendPremiumAccessRevokedEmail(
		targetUser.Email,
		targetUserName,
		reason,
		revokeType,
		revokedAt,
	); err != nil {
		logger.Logger.Error("Failed to send premium revoked email",
			"email", targetUser.Email,
			"error", err.Error(),
		)
	}
}

func (c *Controller) sendPremiumReactivatedEmail(targetUser *user.User, reason string) {
	if targetUser == nil || strings.TrimSpace(targetUser.Email) == "" {
		return
	}

	targetUserName := strings.TrimSpace(targetUser.FirstName + " " + targetUser.LastName)
	if targetUserName == "" {
		targetUserName = strings.TrimSpace(targetUser.Username)
	}

	reactivatedAt := time.Now()
	if targetUser.PremiumReactivatedAt != nil {
		reactivatedAt = *targetUser.PremiumReactivatedAt
	}

	if err := c.emailService.SendPremiumAccessReactivatedEmail(
		targetUser.Email,
		targetUserName,
		reason,
		reactivatedAt,
	); err != nil {
		logger.Logger.Error("Failed to send premium reactivated email",
			"email", targetUser.Email,
			"error", err.Error(),
		)
	}
}

package support

import (
	apperrors "github.com/adehusnim37/lihatin-go/internal/pkg/errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/adehusnim37/lihatin-go/dto"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/repositories/supportrepo"
	"github.com/gin-gonic/gin"
)

func (c *Controller) ListTickets(ctx *gin.Context) {
	page := parsePositiveInt(ctx.DefaultQuery("page", "1"), 1)
	limit := parsePositiveInt(ctx.DefaultQuery("limit", "20"), 20)
	if limit > 100 {
		limit = 100
	}

	filters := supportrepo.TicketListFilters{
		Status:   strings.TrimSpace(ctx.Query("status")),
		Category: strings.TrimSpace(ctx.Query("category")),
		Priority: strings.TrimSpace(ctx.Query("priority")),
		Search:   strings.TrimSpace(ctx.Query("search")),
		Email:    strings.TrimSpace(ctx.Query("email")),
	}

	items, total, err := c.repo.ListTickets(filters, supportrepo.Pagination{
		Page:  page,
		Limit: limit,
	})
	if err != nil {
		c.handleAppError(ctx, err)
		return
	}

	result := make([]dto.AdminSupportTicketItem, 0, len(items))
	for _, item := range items {
		result = append(result, dto.AdminSupportTicketItem{
			ID:         item.ID,
			TicketCode: item.TicketCode,
			Email:      item.Email,
			Category:   item.Category,
			Subject:    item.Subject,
			Status:     item.Status,
			Priority:   item.Priority,
			UserID:     item.UserID,
			CreatedAt:  item.CreatedAt,
			UpdatedAt:  item.UpdatedAt,
			ResolvedAt: item.ResolvedAt,
		})
	}

	totalPages := int((total + int64(limit) - 1) / int64(limit))
	if totalPages == 0 {
		totalPages = 1
	}

	httputil.SendOKResponse(ctx, dto.AdminListSupportTicketsResponse{
		Items:      result,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, "Support tickets retrieved successfully")
}

func (c *Controller) GetTicket(ctx *gin.Context) {
	id := strings.TrimSpace(ctx.Param("id"))
	if id == "" {
		httputil.HandleError(ctx, apperrors.NewAppError("TICKET_ID_REQUIRED", "Ticket ID is required", http.StatusBadRequest, "id"), nil)
		return
	}

	ticket, err := c.repo.GetTicketByID(id)
	if err != nil {
		c.handleAppErrorAs(ctx, err, "id")
		return
	}
	if ticket == nil {
		httputil.HandleError(ctx, apperrors.NewAppError("TICKET_NOT_FOUND", "Support ticket not found", http.StatusNotFound, "id"), nil)
		return
	}

	response := dto.AdminSupportTicketDetailResponse{
		ID:          ticket.ID,
		TicketCode:  ticket.TicketCode,
		Email:       ticket.Email,
		Category:    ticket.Category,
		Subject:     ticket.Subject,
		Description: ticket.Description,
		Status:      ticket.Status,
		Priority:    ticket.Priority,
		UserID:      ticket.UserID,
		AdminNotes:  ticket.AdminNotes,
		ResolvedBy:  ticket.ResolvedBy,
		ResolvedAt:  ticket.ResolvedAt,
		CreatedAt:   ticket.CreatedAt,
		UpdatedAt:   ticket.UpdatedAt,
	}

	httputil.SendOKResponse(ctx, response, "Support ticket retrieved successfully")
}

func parsePositiveInt(raw string, fallback int) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || parsed < 1 {
		return fallback
	}
	return parsed
}

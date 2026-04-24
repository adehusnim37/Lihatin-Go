package support

import (
	"net/http"
	"strings"

	"github.com/adehusnim37/lihatin-go/dto"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	"github.com/gin-gonic/gin"
)

func (c *Controller) TrackTicket(ctx *gin.Context) {
	var req dto.TrackSupportTicketQuery
	if err := ctx.ShouldBindQuery(&req); err != nil {
		validator.SendValidationError(ctx, err, &req)
		return
	}

	req.Ticket = strings.TrimSpace(strings.ToUpper(req.Ticket))
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	ticket, err := c.repo.GetTicketByCodeAndEmail(req.Ticket, req.Email)
	if err != nil || ticket == nil {
		httputil.SendErrorResponse(ctx, http.StatusNotFound, "TICKET_NOT_FOUND", "Ticket not found for provided email", "ticket")
		return
	}

	response := dto.TrackSupportTicketResponse{
		TicketCode: ticket.TicketCode,
		Category:   ticket.Category,
		Subject:    ticket.Subject,
		Status:     ticket.Status,
		CreatedAt:  ticket.CreatedAt,
		ResolvedAt: ticket.ResolvedAt,
	}

	httputil.SendOKResponse(ctx, response, "Ticket found")
}

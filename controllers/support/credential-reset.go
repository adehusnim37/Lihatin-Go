package support

import (
	"context"
	"net/http"
	"strings"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/internal/pkg/auth"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	"github.com/adehusnim37/lihatin-go/middleware"
	"github.com/adehusnim37/lihatin-go/repositories/supportrepo"
	"github.com/gin-gonic/gin"
)

// ResetCredentials requires an explicit human approval and an enrolled MFA proof.
// Normal ticket updates and public support requests cannot call this operation.
func (c *Controller) ResetCredentials(ctx *gin.Context) {
	ctx.Header("Cache-Control", "no-store")
	var req dto.AdminResetSupportCredentialsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		validator.SendValidationError(ctx, err, &req)
		return
	}
	actorID := ctx.GetString("user_id")
	if ctx.GetString("role") != "admin" && ctx.GetString("role") != "super_admin" {
		httputil.SendErrorResponse(ctx, http.StatusForbidden, "ADMIN_REQUIRED", "Administrator permission is required", "auth")
		return
	}
	manager := middleware.GetSessionManager()
	if manager == nil || !auth.ConsumeMFASecurityGrant(ctx.Request.Context(), manager.GetRedisClient(), actorID, ctx.GetString("session_id"), req.VerificationToken) {
		httputil.SendErrorResponse(ctx, http.StatusUnauthorized, "MFA_VERIFICATION_REQUIRED", "Verify your own authenticator or passkey again before approving recovery", "verification_token")
		return
	}
	result, err := c.repo.ApproveCredentialReset(ctx.Request.Context(), supportrepo.CredentialResetApproval{
		ActorID: actorID, TicketID: strings.TrimSpace(ctx.Param("id")), TicketCode: strings.TrimSpace(req.TicketCode), Method: req.Method, Approved: req.Approved, Reason: req.Reason, IPAddress: ctx.ClientIP(),
	}, func(revokeCtx context.Context, userID string) error {
		if err := middleware.DeleteAllUserSessions(revokeCtx, userID); err != nil {
			return err
		}
		return auth.NewRefreshTokenManager(manager.GetRedisClient()).DeleteAllUserRefreshTokens(revokeCtx, userID)
	})
	if err != nil {
		c.handleAppError(ctx, err)
		return
	}
	httputil.SendOKResponse(ctx, result, "Recovery approved. Credentials removed, sessions revoked, and a security email queued.")
}

package auth

import (
	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	"github.com/gin-gonic/gin"
)

func (c *Controller) DeleteAccount(ctx *gin.Context) {
	userID := ctx.GetString("user_id")

	if err := ctx.ShouldBindUri(&dto.DeleteAccountRequest{ID: userID}); err != nil {
		validator.SendValidationError(ctx, err, &dto.DeleteAccountRequest{ID: userID})
		return
	}

	// Call the repository to delete the account
	if err := c.repo.GetUserAdminRepository().DeleteUserPermanent(userID); err != nil {
		http.HandleError(ctx, err, userID)
		return
	}

	http.SendSuccessResponse(ctx, 204, nil, "Akun berhasil dihapus secara permanen")
}

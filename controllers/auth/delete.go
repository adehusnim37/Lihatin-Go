package auth

import (
	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/utils"
	"github.com/gin-gonic/gin"
)

func (c *Controller) DeleteAccount(ctx *gin.Context) {
	userID := ctx.GetString("user_id")

	if err := ctx.ShouldBindUri(&dto.DeleteAccountRequest{ID: userID}); err != nil {
		utils.SendValidationError(ctx, err, &dto.DeleteAccountRequest{ID: userID})
		return
	}

	// Call the repository to delete the account
	if err := c.repo.GetUserRepository().DeleteUserPermanent(userID); err != nil {
		utils.HandleError(ctx, err, userID)
		return
	}

	utils.SendSuccessResponse(ctx, 204, nil, "Akun berhasil dihapus secara permanen")
}

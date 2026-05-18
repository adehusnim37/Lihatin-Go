package auth

import (
	"strings"

	"github.com/adehusnim37/lihatin-go/repositories/userrepo"
)

func (c *Controller) redeemPremiumCodeInDB(secretCode, userID string) error {
	if strings.TrimSpace(secretCode) == "" || strings.TrimSpace(userID) == "" {
		return nil
	}

	premiumRepo := userrepo.NewUserPremiumKeyRepository(c.GormDB)
	return premiumRepo.RedeemPremiumCode(secretCode, userID)
}

package admin

import (
	"time"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/models/user"
)

func toAdminUserResponse(u user.User) dto.AdminUserResponse {
	return dto.AdminUserResponse{
		ID:                     u.ID,
		Username:               u.Username,
		FirstName:              u.FirstName,
		LastName:               u.LastName,
		Email:                  u.Email,
		CreatedAt:              u.CreatedAt,
		UpdatedAt:              u.UpdatedAt,
		AccountStatus:          accountStatus(u.UserAuth),
		AccountStatusChangedAt: accountStatusChangedAt(u.UserAuth),
		AccountStatusReason:    accountStatusReason(u.UserAuth),
		Role:                   u.Role,
		PremiumAccess:          dto.NewPremiumAccessResponse(u.PremiumAccess),
	}
}

func accountStatus(auth *user.UserAuth) string {
	if auth == nil {
		return ""
	}
	return string(auth.AccountStatus)
}

func accountStatusChangedAt(auth *user.UserAuth) *time.Time {
	if auth == nil {
		return nil
	}
	return auth.StatusChangedAt
}

func accountStatusReason(auth *user.UserAuth) string {
	if auth == nil {
		return ""
	}
	return auth.StatusReason
}

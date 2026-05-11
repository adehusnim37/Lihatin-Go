package admin

import (
	"strings"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/models/user"
)

func toAdminUserResponse(u user.User) dto.AdminUserResponse {
	return dto.AdminUserResponse{
		ID:                       u.ID,
		Username:                 u.Username,
		FirstName:                u.FirstName,
		LastName:                 u.LastName,
		Email:                    u.Email,
		CreatedAt:                u.CreatedAt,
		UpdatedAt:                u.UpdatedAt,
		IsPremium:                u.IsPremium,
		IsLocked:                 u.IsLocked,
		LockedAt:                 u.LockedAt,
		LockedReason:             u.LockedReason,
		Role:                     u.Role,
		PremiumRevokeType:        normalizePremiumRevokeType(u.PremiumRevokeType),
		PremiumRevokedAt:         u.PremiumRevokedAt,
		PremiumRevokedBy:         u.PremiumRevokedBy,
		PremiumRevokedReason:     u.PremiumRevokedReason,
		PremiumReactivatedAt:     u.PremiumReactivatedAt,
		PremiumReactivatedBy:     u.PremiumReactivatedBy,
		PremiumReactivatedReason: u.PremiumReactivatedReason,
	}
}

func normalizePremiumRevokeType(raw string) string {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	return normalized
}

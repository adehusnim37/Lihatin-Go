package user

import (
	"testing"
	"time"
)

func TestUserAuthStatusAndLoginBlockHaveDifferentSemantics(t *testing.T) {
	now := time.Now()
	blockedUntil := now.Add(10 * time.Minute)
	auth := &UserAuth{
		AccountStatus:     AccountStatusActive,
		LoginBlockedUntil: &blockedUntil,
	}
	auth.HydrateDerivedState()

	if auth.AccountStatus != AccountStatusActive {
		t.Fatal("temporary failed-login block must not change account status")
	}
	if !auth.IsLoginBlockedAt(now) {
		t.Fatal("expected login to be temporarily blocked")
	}

	auth.AccountStatus = AccountStatusLocked
	auth.LoginBlockedUntil = nil
	auth.HydrateDerivedState()
	if auth.AccountStatus != AccountStatusLocked {
		t.Fatal("persistent lock must remain an explicit account status")
	}
	if auth.IsLoginBlockedAt(now) {
		t.Fatal("persistent account lock must not masquerade as a temporary login block")
	}

	auth.AccountStatus = AccountStatusDisabled
	auth.HydrateDerivedState()
	if auth.AccountStatus != AccountStatusDisabled {
		t.Fatal("disabled account status must remain explicit")
	}
}

func TestUserHydratesNormalizedAuthAndPremiumState(t *testing.T) {
	now := time.Now()
	reason := "security review"
	account := &User{
		UserAuth: &UserAuth{
			AccountStatus:   AccountStatusLocked,
			StatusChangedAt: &now,
			StatusReason:    reason,
		},
		PremiumAccess: &PremiumAccess{Status: PremiumAccessStatusActive},
	}

	account.HydrateDerivedState()

	if !account.IsAccountLocked() || account.UserAuth.StatusReason != reason {
		t.Fatal("expected lock state to remain in user_auth")
	}
	if !account.HasPremiumAccessAt(now) {
		t.Fatal("expected premium state to remain in premium access")
	}
}

func TestExpiredPremiumAccessIsInactive(t *testing.T) {
	expiresAt := time.Now().Add(-time.Minute)
	access := &PremiumAccess{
		Status:    PremiumAccessStatusActive,
		ExpiresAt: &expiresAt,
	}
	if access.IsActiveAt(time.Now()) {
		t.Fatal("expired premium access must not be active")
	}
}

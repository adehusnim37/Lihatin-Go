package dto

import (
	"testing"

	"github.com/adehusnim37/lihatin-go/internal/pkg/identifier"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

func TestAPIKeyIDRequestAcceptsIssuedUUIDV7AndLegacyUUIDV4(t *testing.T) {
	validate := validator.New()
	for _, id := range []string{identifier.NewUUIDV7(), uuid.NewString()} {
		if err := validate.Var(id, "required,uuid"); err != nil {
			t.Fatalf("issued key ID %s rejected: %v", id, err)
		}
	}
	if err := validate.Var("not-a-uuid", "required,uuid"); err == nil {
		t.Fatal("invalid key ID accepted")
	}
}

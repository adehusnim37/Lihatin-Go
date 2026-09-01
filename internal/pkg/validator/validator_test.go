package validator

import (
	"testing"

	"github.com/go-playground/validator/v10"
)

type meaningfulTextFixture struct {
	Text string `validate:"meaningful_text"`
}

func TestValidateMeaningfulText(t *testing.T) {
	tests := []struct {
		name  string
		value string
		valid bool
	}{
		{
			name:  "Indonesian sentence",
			value: "Saya tidak bisa masuk ke akun saya sejak pagi.",
			valid: true,
		},
		{
			name:  "English sentence",
			value: "I cannot log in because the reset link returns an error.",
			valid: true,
		},
		{
			name:  "Mixed support language",
			value: "Login gagal dengan error 403 setelah verifikasi email.",
			valid: true,
		},
		{
			name:  "Technical error value",
			value: "The API returns ERR_CONNECTION_REFUSED on the dashboard.",
			valid: true,
		},
		{
			name:  "Keyboard mash without spaces",
			value: "asfkajnfkashfjkaskhsafkjbjkdsf",
			valid: false,
		},
		{
			name:  "Keyboard rows with spaces",
			value: "qwe rty asd zxc vbn",
			valid: false,
		},
		{
			name:  "Repeated syllable spam",
			value: "papapapaparaaam papapapa paparraarm",
			valid: false,
		},
		{
			name:  "Consonant spam",
			value: "zzzzzzzzzzzzzzzz",
			valid: false,
		},
	}

	validate := validator.New()
	if err := SetupCustomValidators(validate); err != nil {
		t.Fatalf("setup custom validators: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.Struct(meaningfulTextFixture{Text: tt.value})
			if (err == nil) != tt.valid {
				t.Fatalf("validateMeaningfulText(%q) error = %v, valid = %v", tt.value, err, tt.valid)
			}
		})
	}
}

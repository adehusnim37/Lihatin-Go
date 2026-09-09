package validator

import (
	"strings"
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
		{name: "Long Indonesian word", value: "pertanggungjawaban", valid: true},
		{name: "Long availability word", value: "ketidaktersediaan", valid: true},
		{name: "Technical JWT", value: "JWT RS256 verification failed", valid: true},
		{name: "Technical DNS", value: "DNS TXT record belum aktif", valid: true},
		{name: "HTTP code", value: "404", valid: true},
		{name: "Short informal Indonesian", value: "tolong dong min", valid: true},
		{name: "Japanese support", value: "日本語のサポート", valid: true},
		{name: "Cyrillic support", value: "Не могу войти", valid: true},
		{name: "Long numeric only", value: "123456789012345", valid: false},
		{name: "Punctuation only", value: "!!!!!!!!!!!!!!!", valid: false},
		{name: "Emoji only", value: "😀😀😀😀😀😀😀😀", valid: false},
		{name: "Repeated alphabet chunk", value: "abcdefabcdefabcdef", valid: false},
		{name: "Keyboard sequence", value: "qwertyuiopasdfgh", valid: false},
		{name: "Repeated technical-looking spam", value: "error asdfasdfasdfasdf", valid: false},
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

func FuzzIsMeaningfulText(f *testing.F) {
	seeds := []string{
		"Saya tidak bisa login",
		"papapapaparaaam",
		"JWT RS256 failed",
		"日本語",
		strings.Repeat("a", 5000),
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		if len(input) > 10_000 {
			t.Skip()
		}
		first := IsMeaningfulText(input)
		second := IsMeaningfulText(input)
		if first != second {
			t.Fatalf("non-deterministic result for %q", input)
		}
	})
}

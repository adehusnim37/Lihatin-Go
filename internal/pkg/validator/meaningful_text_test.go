package validator

import (
	"reflect"
	"strings"
	"sync"
	"testing"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

func TestAnalyzeMeaningfulText(t *testing.T) {
	tests := []struct {
		name, input    string
		classification TextClassification
		reason         TextReason
	}{
		{"normal report", "Saya tidak bisa login sejak pagi, mohon bantuannya.", TextValid, ""},
		{"keyboard mentioned in prose", "Saya memakai keyboard qwerty", TextSuspicious, ReasonKeyboardPattern},
		{"short keyboard token in prose", "Tombol asdf di keyboard saya rusak", TextSuspicious, ReasonKeyboardPattern},
		{"reverse keyboard", "poiuytrewq", TextGibberish, ReasonKeyboardPattern},
		{"short ambiguous token", "asd", TextSuspicious, ReasonKeyboardPattern},
		{"keyboard columns", "qaz wsx edc", TextGibberish, ReasonKeyboardPattern},
		{"repeated words", "abc abc abc abc", TextGibberish, ReasonRepetition},
		{"repeated phrases", "tolong bantu tolong bantu tolong bantu", TextGibberish, ReasonRepetition},
		{"repeated non Latin", "哈哈哈哈哈哈哈哈", TextGibberish, ReasonRepetition},
		{"repeated Cyrillic", "абвабвабвабв", TextGibberish, ReasonRepetition},
		{"laughter in report", "Hahahahaha, setelah dicoba lagi ternyata akun saya masih gagal login sejak pagi ini.", TextSuspicious, ReasonRepetition},
		{"Hindi combining marks", "मुझे खाते में लॉगिन करने में मदद चाहिए", TextValid, ""},
		{"technical error alone", "ERR_CONNECTION_REFUSED", TextValid, ""},
		{"technical path alone", `C:\logs\asdfasdf.txt`, TextValid, ""},
		{"technical URL alone", "https://example.com/asdfasdf", TextValid, ""},
		{"technical UUID alone", "550e8400-e29b-41d4-a716-446655440000", TextValid, ""},
		{"technical code cannot exempt spam", "ERR_CONNECTION_REFUSED asdfasdfasdfasdf", TextGibberish, ReasonRepetition},
		{"URL cannot exempt punctuation spam", "https://example.com !!!!!!!!!!", TextGibberish, ReasonNoLetters},
		{"technical keyword cannot exempt spam", "error asdfasdfasdfasdf", TextGibberish, ReasonRepetition},
		{"technical consonant code in prose", "Dashboard menampilkan ERR_SSL_VERSION_OR_CIPHER_MISMATCH saat saya membuka halaman login", TextValid, ""},
		{"slash punctuation is not a path", "/!!!!!!!!!!", TextGibberish, ReasonNoLetters},
		{"invalid encoding", "\xff\xfe", TextGibberish, ReasonInvalidUTF8},
		{"empty optional input", "", TextValid, ""},
		{"whitespace-only supplied input", " \n\t", TextGibberish, ReasonNoLetters},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AnalyzeMeaningfulText(tt.input)
			if result.Classification != tt.classification {
				t.Fatalf("AnalyzeMeaningfulText(%q) = %+v, want %s", tt.input, result, tt.classification)
			}
			if tt.reason != "" {
				found := false
				for _, reason := range result.Reasons {
					found = found || reason == tt.reason
				}
				if !found {
					t.Fatalf("missing reason %s: %+v", tt.reason, result)
				}
			}
			if IsMeaningfulText(tt.input) != (tt.classification != TextGibberish) {
				t.Fatal("boolean wrapper disagrees with classification")
			}
		})
	}
}

func TestMeaningfulTextUnicodeNormalization(t *testing.T) {
	for _, input := range []string{
		"Échec de connexion au compte, réessayez après quelques minutes.",
		"अभी खाते में लॉगिन नहीं हो रहा है",
		"éééééééééééé",
	} {
		composed := AnalyzeMeaningfulText(norm.NFC.String(input))
		decomposed := AnalyzeMeaningfulText(norm.NFD.String(input))
		if !reflect.DeepEqual(composed, decomposed) {
			t.Fatalf("normalization changes classification for %q: %+v / %+v", input, composed, decomposed)
		}
	}
}

func TestMeaningfulTextLongLegitimateProse(t *testing.T) {
	// Natural long text uses a finite alphabet; global unique-letter ratios must
	// not combine with a single mention of qwerty to reject the whole report.
	input := strings.Repeat("Saya sudah mencoba login dari perangkat lain tetapi akun masih terkunci. ", 50) + "Keyboard saya qwerty."
	if !IsMeaningfulText(input) {
		t.Fatal("rejected long legitimate support report")
	}
}

func TestMeaningfulTextConcurrent(t *testing.T) {
	var wg sync.WaitGroup
	for range 16 {
		wg.Go(func() {
			for _, input := range []string{"HTTPS SMTP DNS", "JWT RS256 failed", "DNS TXT record belum aktif", "abc abc abc abc", "日本語のサポート"} {
				if got, want := AnalyzeMeaningfulText(input), AnalyzeMeaningfulText(input); !reflect.DeepEqual(got, want) {
					t.Errorf("inconsistent concurrent result for %q", input)
				}
			}
		})
	}
	wg.Wait()
}

func FuzzAnalyzeMeaningfulTextNormalization(f *testing.F) {
	for _, seed := range []string{"Échec de connexion", "abc abc abc abc", "https://example.com", "日本語", "qwerty"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		if !utf8.ValidString(input) || len(input) > 10_000 {
			t.Skip()
		}
		got := AnalyzeMeaningfulText(input)
		normalized := AnalyzeMeaningfulText(norm.NFC.String(input))
		if !reflect.DeepEqual(got, normalized) {
			t.Fatalf("NFC changed result for %q", input)
		}
	})
}

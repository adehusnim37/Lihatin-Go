package validator

import (
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/go-playground/validator/v10"
	"github.com/pemistahl/lingua-go"
	"golang.org/x/text/unicode/norm"
)

// TextClassification describes heuristic evidence, not semantic correctness.
type TextClassification string

const (
	// TextValid means the heuristics found no strong evidence of gibberish.
	TextValid TextClassification = "valid"
	// TextSuspicious retains weak signals for inspection without rejecting text.
	TextSuspicious TextClassification = "suspicious"
	// TextGibberish indicates a dominant pattern or unusable text encoding.
	TextGibberish TextClassification = "gibberish"
)

// TextReason identifies a signal contributing to a text classification.
type TextReason string

const (
	ReasonNoLetters       TextReason = "no_letters"
	ReasonRepetition      TextReason = "repetition"
	ReasonKeyboardPattern TextReason = "keyboard_pattern"
	ReasonConsonantRun    TextReason = "consonant_run"
	ReasonLowVowelRatio   TextReason = "low_vowel_ratio"
	ReasonLongToken       TextReason = "long_token"
	ReasonInvalidUTF8     TextReason = "invalid_utf8"
)

// MeaningfulTextResult explains a decision. Score is an additive heuristic,
// not a probability. Suspicious text is accepted by IsMeaningfulText.
type MeaningfulTextResult struct {
	Classification TextClassification `json:"classification"`
	Score          int                `json:"score"`
	Reasons        []TextReason       `json:"reasons"`
}

// dominantTextRatio requires a signal to cover most prose letters before it
// rejects a whole message. This initial policy needs calibration on real input.
const dominantTextRatio = 0.60

var meaningfulTextDetector = sync.OnceValue(func() lingua.LanguageDetector {
	return lingua.NewLanguageDetectorBuilder().
		FromLanguages(lingua.Indonesian, lingua.English).
		Build()
})

func validateMeaningfulText(fl validator.FieldLevel) bool {
	return IsMeaningfulText(fl.Field().String())
}

// IsMeaningfulText accepts valid and uncertain text, rejecting clear gibberish.
// An empty string passes; whitespace-only input is rejected. Enforce length separately.
func IsMeaningfulText(raw string) bool {
	return AnalyzeMeaningfulText(raw).Classification != TextGibberish
}

// AnalyzeMeaningfulText analyzes an NFC-normalized copy without changing raw.
// It exempts structured technical tokens and checks repetition across scripts.
// Use required/min/max rules separately; callers should bound request size.
func AnalyzeMeaningfulText(raw string) MeaningfulTextResult {
	result := MeaningfulTextResult{Classification: TextValid, Reasons: []TextReason{}}
	if !utf8.ValidString(raw) {
		result.add(ReasonInvalidUTF8, 2)
		result.Classification = TextGibberish
		return result
	}
	value := strings.TrimSpace(norm.NFC.String(raw))
	if value == "" {
		if raw != "" {
			result.add(ReasonNoLetters, 2)
			result.Classification = TextGibberish
		}
		return result
	}
	metrics := analyzeMeaningfulText(value)
	if metrics.letterCount == 0 {
		// Structured URLs, paths and error codes may be the whole support message.
		if (metrics.technicalCount > 0 && metrics.digitCount == 0 && metrics.symbolCount == 0) ||
			(metrics.digitCount == 3 && metrics.symbolCount == 0) {
			return result
		}
		result.add(ReasonNoLetters, 2)
		result.Classification = TextGibberish
		return result
	}
	repetition := float64(metrics.repeatedLetters) / float64(metrics.letterCount)
	keyboard := float64(metrics.keyboardLetters) / float64(metrics.letterCount)
	consonants := float64(metrics.consonantLetters) / float64(metrics.letterCount)
	if repetition > 0 {
		result.add(ReasonRepetition, signalWeight(repetition))
	}
	if keyboard > 0 {
		result.add(ReasonKeyboardPattern, signalWeight(keyboard))
	}
	if consonants > 0 {
		result.add(ReasonConsonantRun, signalWeight(consonants))
	}
	if metrics.lowVowelRatio {
		result.add(ReasonLowVowelRatio, 1)
	}
	if metrics.wordCount == 1 && metrics.latinLetterCount >= 15 {
		result.add(ReasonLongToken, 1)
	}
	switch {
	case repetition >= dominantTextRatio,
		keyboard >= dominantTextRatio && metrics.letterCount >= 8,
		consonants >= dominantTextRatio && metrics.lowVowelRatio:
		result.Classification = TextGibberish
	case result.Score > 0:
		result.Classification = TextSuspicious
		// Language confidence only helps with abbreviation-heavy short prose.
		// It must never override dominant keyboard patterns or repetition.
		if metrics.lowVowelRatio && consonants == 0 && repetition == 0 && keyboard == 0 &&
			metrics.wordCount <= 3 && metrics.latinLetterCount <= 256 && isConfidentEnglishOrIndonesian(value) {
			result.Classification = TextValid
		}
	}
	return result
}

func (r *MeaningfulTextResult) add(reason TextReason, score int) {
	r.Reasons = append(r.Reasons, reason)
	r.Score += score
}

func signalWeight(coverage float64) int {
	if coverage >= dominantTextRatio {
		return 2
	}
	return 1
}

type meaningfulTextMetrics struct {
	wordCount, letterCount, latinLetterCount, vowelCount int
	digitCount, symbolCount, technicalCount              int
	repeatedLetters, keyboardLetters, consonantLetters   int
	lowVowelRatio                                        bool
}

func analyzeMeaningfulText(value string) meaningfulTextMetrics {
	var metrics meaningfulTextMetrics
	// Replace each structured token with a boundary so adjacent prose cannot join.
	prose := technicalTextRegex.ReplaceAllStringFunc(value, func(string) string {
		metrics.technicalCount++
		return " "
	})
	words := meaningfulTextTokenRegex.FindAllString(strings.ToLower(prose), -1)
	lengths := make([]int, len(words))
	for i, word := range words {
		metrics.wordCount++
		var latin, vowels, run, maxRun int
		for _, r := range word {
			if !unicode.IsLetter(r) {
				continue
			} // Keep combining marks inside tokens.
			lengths[i]++
			if !unicode.In(r, unicode.Latin) {
				run = 0
				continue
			}
			latin++
			if isLatinVowel(r) {
				vowels++
				run = 0
			} else {
				run++
				maxRun = max(maxRun, run)
			}
		}
		metrics.letterCount += lengths[i]
		metrics.latinLetterCount += latin
		metrics.vowelCount += vowels
		// A token must itself be mostly a keyboard walk to count against the prose.
		if keyboardCoverage(word) >= dominantTextRatio {
			metrics.keyboardLetters += lengths[i]
		}
		if maxRun >= 6 {
			metrics.consonantLetters += lengths[i]
		}
		span := repeatedSpan([]rune(word), 8)
		if span >= 8 && float64(span)/float64(utf8.RuneCountInString(word)) >= 0.50 {
			metrics.repeatedLetters += lengths[i]
		}
	}
	// Use the strongest repetition signal without counting the same text twice.
	metrics.repeatedLetters = max(metrics.repeatedLetters, repeatedWordLetters(words, lengths))
	for _, r := range prose {
		if unicode.IsDigit(r) {
			metrics.digitCount++
		} else if !unicode.IsLetter(r) && !unicode.IsMark(r) && !unicode.IsSpace(r) {
			metrics.symbolCount++
		}
	}
	metrics.lowVowelRatio = metrics.latinLetterCount >= 12 && float64(metrics.vowelCount)/float64(metrics.latinLetterCount) < 0.20
	return metrics
}

func isLatinVowel(r rune) bool {
	if r < utf8.RuneSelf {
		return strings.ContainsRune("aeiou", r)
	}
	// NFD also handles Latin vowels with accents not enumerated in a fixed list.
	for _, base := range norm.NFD.String(string(r)) {
		return strings.ContainsRune("aeiou", base)
	}
	return false
}

func isConfidentEnglishOrIndonesian(value string) bool {
	// The confidence values compare languages; they do not measure meaningfulness.
	values := meaningfulTextDetector().ComputeLanguageConfidenceValues(value)
	return len(values) > 0 && values[0].Value() >= 0.65
}

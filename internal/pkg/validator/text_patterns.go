package validator

import (
	"regexp"
	"unicode/utf8"
)

var (
	meaningfulTextTokenRegex = regexp.MustCompile(`[\p{L}][\p{L}\p{M}]*`)
	// Exempt structured tokens, never a whole message just because it contains
	// a keyword such as "error". Match before lowercasing to retain code casing.
	technicalTextRegex = regexp.MustCompile(`(?i:https?://[^\s]+|www\.[^\s]+|\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b|\b[0-9a-f]{32,}\b)|\b[A-Z][A-Z0-9]*(?:_[A-Z0-9]+)+\b|(?:[A-Za-z]:)?[/\\](?:[\p{L}\p{N}._-]+[/\\])*[\p{L}\p{N}._-]+`)
)

var keyboardRows = [...]string{"qwertyuiop", "asdfghjkl", "zxcvbnm", "qaz", "wsx", "edc", "rfv", "tgb", "yhn", "ujm"}

// keyboardCoverage counts disjoint forward/reverse runs of at least three
// characters on keyboard rows/columns. It is applied to tokens, not substrings
// anywhere in the message. No language decision depends on one short token.
func keyboardCoverage(word string) float64 {
	covered := 0
	for i := 0; i < len(word); {
		longest := 0
		for _, row := range keyboardRows {
			for start := 0; start < len(row); start++ {
				for _, direction := range []int{-1, 1} {
					length := 0
					for j := start; j >= 0 && j < len(row) && i+length < len(word) && word[i+length] == row[j]; j += direction {
						length++
					}
					if length >= 3 {
						longest = max(longest, length)
					}
				}
			}
		}
		if longest == 0 {
			i++
		} else {
			covered += longest
			i += longest
		}
	}
	return float64(covered) / float64(utf8.RuneCountInString(word))
}

// repeatedSpan finds the longest contiguous run of >=3 identical chunks.
// Checking at most eight chunk sizes bounds work to O(n), including long spam.
func repeatedSpan(items []rune, minSpan int) int {
	longest := 0
	for size := 1; size <= 8 && size*3 <= len(items); size++ {
		matched := 0
		for i := size; i < len(items); i++ {
			if items[i] == items[i-size] {
				matched++
			} else {
				matched = 0
			}
			span := matched + size
			if matched >= 2*size && span >= minSpan {
				longest = max(longest, span)
			}
		}
	}
	return longest
}

func repeatedWordLetters(words []string, lengths []int) int {
	// Prefix sums let each repeated phrase contribute letters rather than token
	// counts, so short repeated filler does not outweigh a longer explanation.
	prefix := make([]int, len(words)+1)
	for i, length := range lengths {
		prefix[i+1] = prefix[i] + length
	}
	covered := 0
	for size := 1; size <= 4 && size*3 <= len(words); size++ {
		matched := 0
		for i := size; i < len(words); i++ {
			if words[i] == words[i-size] {
				matched++
			} else {
				matched = 0
			}
			if matched >= 2*size {
				letters := prefix[i+1] - prefix[i+1-matched-size]
				if letters >= 8 {
					covered = max(covered, letters)
				}
			}
		}
	}
	return covered
}

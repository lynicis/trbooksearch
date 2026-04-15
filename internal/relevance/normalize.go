package relevance

import (
	"strings"
	"unicode"
)

// turkishReplacements maps Turkish-specific characters to ASCII equivalents.
var turkishReplacements = map[rune]rune{
	'ş': 's', 'Ş': 's',
	'ç': 'c', 'Ç': 'c',
	'ö': 'o', 'Ö': 'o',
	'ü': 'u', 'Ü': 'u',
	'ğ': 'g', 'Ğ': 'g',
	'ı': 'i', 'İ': 'i',
	'â': 'a', 'Â': 'a',
	'î': 'i', 'Î': 'i',
	'û': 'u', 'Û': 'u',
}

// NormalizeTurkish lowercases, replaces Turkish-specific characters with ASCII
// equivalents, strips non-alphanumeric/non-space characters, and collapses whitespace.
func NormalizeTurkish(s string) string {
	// Turkish-aware lowercasing: İ→i, I→ı (but we normalize ı→i too)
	s = strings.ToLowerSpecial(unicode.TurkishCase, s)

	var b strings.Builder
	b.Grow(len(s))
	prevSpace := false

	for _, r := range s {
		if rep, ok := turkishReplacements[r]; ok {
			r = rep
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			prevSpace = false
		} else if unicode.IsSpace(r) {
			if !prevSpace && b.Len() > 0 {
				b.WriteRune(' ')
				prevSpace = true
			}
		}
		// skip other characters (punctuation, etc.)
	}

	return strings.TrimSpace(b.String())
}

// Tokenize normalizes a string and splits it into lowercase, de-accented word tokens.
func Tokenize(s string) []string {
	normalized := NormalizeTurkish(s)
	if normalized == "" {
		return nil
	}
	return strings.Fields(normalized)
}

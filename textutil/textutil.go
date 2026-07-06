// Package textutil provides small, generic (non-domain) string helpers that apps
// otherwise reinvent (U7).
package textutil

import (
	"strings"
	"unicode"
)

// Humanize turns an enum/identifier like "credit_card", "creditCard", or
// "credit-card" into a human Title-case-first phrase: "Credit card".
func Humanize(parseValue string) string {
	parseWords := splitWords(parseValue)
	if len(parseWords) == 0 {
		return ""
	}
	for parseI := range parseWords {
		parseWords[parseI] = strings.ToLower(parseWords[parseI])
	}
	parseJoined := strings.Join(parseWords, " ")
	return upperFirstRune(parseJoined)
}

// TitleCase capitalizes the first letter of every word in an identifier:
// "credit_card" -> "Credit Card".
func TitleCase(parseValue string) string {
	parseWords := splitWords(parseValue)
	for parseI, parseWord := range parseWords {
		if parseWord == "" {
			continue
		}
		parseWords[parseI] = upperFirstRune(strings.ToLower(parseWord))
	}
	return strings.Join(parseWords, " ")
}

// upperFirstRune uppercases the first RUNE of a string. Byte-index slicing
// (s[:1]) split multi-byte leading runes ("über" -> corrupted UTF-8).
func upperFirstRune(parseValue string) string {
	for parseIndex, parseR := range parseValue {
		return string(unicode.ToUpper(parseR)) + parseValue[parseIndex+len(string(parseR)):]
	}
	return parseValue
}

// splitWords breaks snake_case, kebab-case, spaces, and camelCase boundaries.
func splitWords(parseValue string) []string {
	var parseWords []string
	var parseCur strings.Builder
	parseFlush := func() {
		if parseCur.Len() > 0 {
			parseWords = append(parseWords, parseCur.String())
			parseCur.Reset()
		}
	}
	parseRunes := []rune(parseValue)
	for parseI, parseR := range parseRunes {
		switch {
		case parseR == '_' || parseR == '-' || unicode.IsSpace(parseR):
			parseFlush()
		case unicode.IsUpper(parseR) && parseI > 0 && (unicode.IsLower(parseRunes[parseI-1]) || unicode.IsDigit(parseRunes[parseI-1])):
			parseFlush()
			parseCur.WriteRune(parseR)
		default:
			parseCur.WriteRune(parseR)
		}
	}
	parseFlush()
	return parseWords
}

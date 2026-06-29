package runtime2

import (
	"strings"
	"unicode/utf8"
)

// parseRuntimeHasTrimmedNonWhitespaceText reports whether one text value contains any non-whitespace content.
func parseRuntimeHasTrimmedNonWhitespaceText(parseText string) bool {
	for parseIndex := 0; parseIndex < len(parseText); parseIndex++ {
		parseByte := parseText[parseIndex]
		if parseByte >= utf8.RuneSelf {
			return strings.TrimSpace(parseText) != ""
		}
		switch parseByte {
		case ' ', '\t', '\n', '\r', '\f', '\v':
			continue
		default:
			return true
		}
	}
	return false
}

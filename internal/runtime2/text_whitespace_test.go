package runtime2

import "strings"

import "testing"

// TestParseRuntimeHasTrimmedNonWhitespaceText verifies runtime trim checks match strings.TrimSpace semantics for ASCII and representative Unicode text.
func TestParseRuntimeHasTrimmedNonWhitespaceText(parseT *testing.T) {
	for parseByte := byte(0); parseByte < 0x80; parseByte++ {
		parseText := string([]byte{parseByte})
		getExpected := strings.TrimSpace(parseText) != ""
		getActual := parseRuntimeHasTrimmedNonWhitespaceText(parseText)
		if getActual != getExpected {
			parseT.Fatalf("expected ASCII byte %d result %t, got %t", parseByte, getExpected, getActual)
		}
	}
	getUnicodeCases := []string{
		"",
		" ",
		"\t",
		"\r\n",
		"\u2003",
		"\u00A0",
		"中",
		" 中 ",
		"\u2003text",
		"text\u2003",
		"🔥",
	}
	for _, getText := range getUnicodeCases {
		getExpected := strings.TrimSpace(getText) != ""
		getActual := parseRuntimeHasTrimmedNonWhitespaceText(getText)
		if getActual != getExpected {
			parseT.Fatalf("expected text %q result %t, got %t", getText, getExpected, getActual)
		}
	}
}

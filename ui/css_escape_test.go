package ui

import "testing"

// TestCSSEscapeCorrectness covers the WHATWG CSS.escape behaviors that matter for
// turning an arbitrary id into a safe selector (G29).
func TestCSSEscapeCorrectness(parseT *testing.T) {
	parseCases := []struct {
		parseName string
		parseIn   string
		parseWant string
	}{
		{"empty", "", ""},
		{"plain", "hello", "hello"},
		{"already-safe-generated-id", "gwc-3-1", "gwc-3-1"},
		{"underscore-and-digits", "a_1-2", "a_1-2"},
		{"colon-pseudo-class-trap", "gwc:3:1", `gwc\:3\:1`},
		{"space", "a b", `a\ b`},
		{"leading-digit", "1abc", `\31 abc`},
		{"leading-hyphen-digit", "-1abc", `-\31 abc`},
		{"lone-hyphen", "-", `\-`},
		{"double-hyphen-ok", "--custom", "--custom"},
		{"dot-and-hash", "a.b#c", `a\.b\#c`},
		{"null-becomes-replacement", "a\x00b", "a�b"},
		{"control-char", "a\x01b", "a\\1 b"},
		{"unicode-passthrough", "café", "café"},
		// Authoritative cross-checks against the browser CSS.escape() spec.
		{"single-digit", "9", `\39 `},
		{"tab-control", "\t", `\9 `},
		{"double-leading-hyphen", "--custom-prop", "--custom-prop"},
		{"hyphen-then-letter", "-a", "-a"},
		{"trailing-and-mid-special", "a#b.c d", `a\#b\.c\ d`},
		{"percent-and-paren-leading-digit", "50%(x)", `\35 0\%\(x\)`},
	}

	for _, parseCase := range parseCases {
		parseT.Run(parseCase.parseName, func(parseT *testing.T) {
			if parseGot := CSSEscape(parseCase.parseIn); parseGot != parseCase.parseWant {
				parseT.Fatalf("CSSEscape(%q) = %q, want %q", parseCase.parseIn, parseGot, parseCase.parseWant)
			}
		})
	}
}

// TestSelectorIDPrefixesHash verifies SelectorID prepends "#" to the escaped id.
func TestSelectorIDPrefixesHash(parseT *testing.T) {
	if parseGot := SelectorID("gwc-3-1"); parseGot != "#gwc-3-1" {
		parseT.Fatalf("SelectorID(safe) = %q, want #gwc-3-1", parseGot)
	}
	if parseGot := SelectorID("gwc:3:1"); parseGot != `#gwc\:3\:1` {
		parseT.Fatalf("SelectorID(colon) = %q, want escaped", parseGot)
	}
}

// TestCSSEscapeIdempotentOnSafeInput proves escaping an already-safe identifier
// is a no-op — generated ids never get double-escaped.
func TestCSSEscapeIdempotentOnSafeInput(parseT *testing.T) {
	parseSafe := []string{"gwc-0-0", "gwc-999-12", "field_name", "-x", "_x", "AbZ09"}
	for _, parseID := range parseSafe {
		if parseGot := CSSEscape(parseID); parseGot != parseID {
			parseT.Fatalf("CSSEscape(%q) should be a no-op, got %q", parseID, parseGot)
		}
	}
}

//go:build !js || !wasm

package runtime

import (
	"strings"
	"testing"
)

// FuzzSanitizeURLAttributeValue proves the core security invariant: for ANY
// input on a URL-bearing attribute, the sanitized value can never carry a
// javascript: or vbscript: scheme (after the same whitespace/control-char/case
// normalization an attacker would rely on the browser to perform). It also
// asserts the sanitizer never panics and never rewrites a non-URL attribute.
func FuzzSanitizeURLAttributeValue(parseF *testing.F) {
	for _, parseSeed := range []string{
		"javascript:alert(1)", "JavaScript:x", " javascript:x",
		"java\tscript:x", "\t\r\n Jav\rasCr\r\niP\t\n\rt\n:x",
		"\x00\x1fjavascript:x", "vbscript:x", "VBScript:x",
		"http://example.com", "https://a/b?c#d", "//cdn/x", "/rel", "./rel",
		"#frag", "?q=1", "mailto:a@b.c", "data:image/png;base64,AA",
		"http://javascript:0/ok", "", "notascheme", "tel:+1",
	} {
		parseF.Add(parseSeed)
	}

	parseF.Fuzz(func(parseT *testing.T, parseIn string) {
		// URL-bearing attribute: an executable scheme must never survive.
		parseOut := SanitizeURLAttributeValue("href", parseIn)
		if parseScheme, parseHas := ssrURLScheme(strings.TrimSpace(parseOut)); parseHas {
			switch normalizeSSRScheme(parseScheme) {
			case "javascript", "vbscript":
				parseT.Fatalf("executable scheme survived sanitization: in=%q out=%q", parseIn, parseOut)
			}
		}

		// Non-URL attribute: value must pass through byte-for-byte.
		if parsePass := SanitizeURLAttributeValue("title", parseIn); parsePass != parseIn {
			parseT.Fatalf("non-URL attribute was rewritten: in=%q out=%q", parseIn, parsePass)
		}
	})
}

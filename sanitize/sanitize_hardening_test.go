package sanitize_test

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/sanitize"
)

// TestSanitizeDropsTemplateSubtree pins that <template> content is dropped
// whole rather than unwrapped into live, out-of-context output (x/net/html does
// not model template's inert content fragment).
func TestSanitizeDropsTemplateSubtree(parseT *testing.T) {
	parseOut := sanitize.Sanitize(`<template><tr><td>leaked-row</td></tr></template>`)
	if strings.Contains(parseOut, "leaked-row") || strings.Contains(parseOut, "<tr") {
		parseT.Fatalf("template subtree was not dropped: %q", parseOut)
	}
}

// TestSanitizeValidatesSpanAttributes pins that oversized/negative/non-numeric
// colspan/rowspan values are dropped while legitimate ones survive.
func TestSanitizeValidatesSpanAttributes(parseT *testing.T) {
	parseBad := sanitize.Sanitize(`<table><tr><td colspan="99999999999999999999" rowspan="-5">c</td></tr></table>`)
	if strings.Contains(parseBad, "colspan") || strings.Contains(parseBad, "rowspan") {
		parseT.Fatalf("invalid span attributes were not dropped: %q", parseBad)
	}
	parseGood := sanitize.Sanitize(`<table><tr><td colspan="2" rowspan="3">c</td></tr></table>`)
	if !strings.Contains(parseGood, `colspan="2"`) || !strings.Contains(parseGood, `rowspan="3"`) {
		parseT.Fatalf("valid span attributes were dropped: %q", parseGood)
	}
}

// TestSanitizeEmitsCleanedURL pins that the URL emitted is the same
// control-char-stripped string that passed scheme validation, not the raw value.
func TestSanitizeEmitsCleanedURL(parseT *testing.T) {
	// Embedded zero-width space inside the scheme.
	parseOut := sanitize.Sanitize("<a href=\"ht​tps://example.test/x\">link</a>")
	if strings.Contains(parseOut, "​") {
		parseT.Fatalf("raw (uncleaned) URL was emitted: %q", parseOut)
	}
	// A javascript: scheme disguised with a control char must still be dropped.
	parseJS := sanitize.Sanitize("<a href=\"java\tscript:alert(1)\">x</a>")
	if strings.Contains(parseJS, "href") {
		parseT.Fatalf("control-char-disguised javascript URL was not dropped: %q", parseJS)
	}
}

// TestSanitizePreservesInteriorSpaceInURL pins that a legitimate relative URL
// containing an ASCII space is emitted intact rather than collapsed.
func TestSanitizePreservesInteriorSpaceInURL(parseT *testing.T) {
	parseOut := sanitize.Sanitize(`<a href="/search?q=hello world">x</a>`)
	if !strings.Contains(parseOut, `href="/search?q=hello world"`) {
		parseT.Fatalf("interior space was stripped from a safe URL: %q", parseOut)
	}
	parseJS := sanitize.Sanitize(`<a href="java script:alert(1)">x</a>`)
	if strings.Contains(parseJS, "href") {
		parseT.Fatalf("space-split scheme was not dropped: %q", parseJS)
	}
}

// TestSanitizeURLVectors is a table of known URL-scheme bypass vectors that a
// permissive custom policy must still neutralise.
func TestSanitizeURLVectors(parseT *testing.T) {
	parsePolicy := sanitize.DefaultPolicy()
	for _, parseAttr := range []string{"srcset", "poster", "cite", "background", "ping", "data", "longdesc"} {
		parsePolicy.AllowedAttributes[parseAttr] = true
	}
	for _, parseTag := range []string{"video", "object", "area"} {
		parsePolicy.AllowedTags[parseTag] = true
	}
	parseCases := []struct{ parseName, parseInput string }{
		{"mixed case", `<a href="JaVaScRiPt:alert(1)">x</a>`},
		{"entity colon", `<a href="javascript&colon;alert(1)">x</a>`},
		{"numeric entity", `<a href="&#106;avascript:alert(1)">x</a>`},
		{"newline in scheme", "<a href=\"java\nscript:alert(1)\">x</a>"},
		{"leading space", `<a href="   javascript:alert(1)">x</a>`},
		{"leading control", "<a href=\"\x01javascript:alert(1)\">x</a>"},
		{"data html img", `<img src="data:text/html;base64,PHNjcmlwdD4=">`},
		{"srcset js", `<img src="/a.png" srcset="javascript:alert(1) 1x">`},
		{"srcset second js", `<img src="/a.png" srcset="/ok.png 1x, javascript:alert(1) 2x">`},
		{"video poster", `<video poster="javascript:alert(1)"></video>`},
		{"blockquote cite", `<blockquote cite="javascript:alert(1)">q</blockquote>`},
		{"table background", `<table background="javascript:alert(1)"></table>`},
		{"area ping", `<area ping="javascript:alert(1)">`},
		{"xlink href", `<a xlink:href="javascript:alert(1)">x</a>`},
		{"on prefix upper", `<img src=x ONERROR=alert(1)>`},
		{"svg in a", `<a href="/x"><svg><script>alert(1)</script></svg></a>`},
		{"math", `<math><mtext><script>alert(1)</script></mtext></math>`},
		{"noscript breakout", `<noscript><p title="</noscript><img src=x onerror=alert(1)>"></noscript>`},
		{"style breakout", `<style><a id="</style><img src=x onerror=alert(1)>"></style>`},
	}
	for _, parseCase := range parseCases {
		parseCase := parseCase
		parseT.Run(parseCase.parseName, func(parseT *testing.T) {
			parseOut := strings.ToLower(parsePolicy.Sanitize(parseCase.parseInput))
			for _, parseBad := range []string{"javascript:", "<script", "onerror", "data:text", "<img src=x"} {
				if strings.Contains(parseOut, parseBad) {
					parseT.Fatalf("output contains %q: %q", parseBad, parseOut)
				}
			}
		})
	}
}

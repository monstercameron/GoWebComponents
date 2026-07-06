package sanitize_test

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/sanitize"
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

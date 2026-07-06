//go:build !(js && wasm)

package ssr

import (
	"strings"
	"testing"
)

// TestStructuredExtractsHeadElementsFromFragment pins the #64 ssr-parse behavior:
// Structured() wraps the fragment in a <div> and parses it as a full document,
// which relocates head-only elements (title/meta) into a synthesized <head>. The
// whole-tree walk must still extract them, so a test asserting SEO/meta output
// does not silently false-pass/fail.
func TestStructuredExtractsHeadElementsFromFragment(parseT *testing.T) {
	parseSnap := Snapshot{HTML: `<title>Home</title>` +
		`<meta name="description" content="A page">` +
		`<meta property="og:title" content="OG Home">` +
		`<link rel="canonical" href="https://example.com/home">` +
		`<script type="application/ld+json" id="jsonld-home">{"@type":"WebPage"}</script>` +
		`<p>body</p>`}

	parseStructured := parseSnap.Structured(parseT)

	if parseGot := parseStructured.MetaName("description"); parseGot != "A page" {
		parseT.Fatalf("MetaName(description) = %q, want %q", parseGot, "A page")
	}
	if parseGot := parseStructured.MetaProperty("og:title"); parseGot != "OG Home" {
		parseT.Fatalf("MetaProperty(og:title) = %q, want %q", parseGot, "OG Home")
	}
	if parseGot := parseStructured.CanonicalURL(); parseGot != "https://example.com/home" {
		parseT.Fatalf("CanonicalURL() = %q, want the canonical href", parseGot)
	}
	if parseGot := parseStructured.JSONLD("jsonld-home"); !strings.Contains(parseGot, "WebPage") {
		parseT.Fatalf("JSONLD(jsonld-home) = %q, want it to contain WebPage", parseGot)
	}
	if parseStructured.Title != "Home" {
		parseT.Fatalf("Title = %q, want Home (title must survive div-wrapped fragment parsing)", parseStructured.Title)
	}
}

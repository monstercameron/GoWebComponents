package main

import (
	"strings"
	"testing"
)

// TestRenderExamplesListingHTMLEscapesReflectedInput pins that the reflected
// search query and the filesystem-derived link name/href are HTML-escaped, so a
// crafted ?q= value or a maliciously named example folder cannot inject markup.
func TestRenderExamplesListingHTMLEscapesReflectedInput(parseT *testing.T) {
	parseLinks := []exampleLink{{
		Name: `<script>alert('name')</script>`,
		Href: `"><script>alert('href')</script>`,
	}}
	parseQuery := `<script>alert('q')</script>`

	parseHTML := renderExamplesListingHTML(parseLinks, parseQuery)

	if strings.Contains(parseHTML, "<script>alert(") {
		parseT.Fatalf("reflected/derived input was not escaped:\n%s", parseHTML)
	}
	// The escaped forms must be present (proves the values were rendered, escaped).
	if !strings.Contains(parseHTML, "&lt;script&gt;alert(&#39;q&#39;)") &&
		!strings.Contains(parseHTML, "&lt;script&gt;alert('q')") {
		parseT.Fatalf("expected escaped query in output:\n%s", parseHTML)
	}
}

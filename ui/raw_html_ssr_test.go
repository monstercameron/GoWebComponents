//go:build !js || !wasm

package ui_test

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// Ported from React's ReactDOMServerIntegrationElements dangerouslySetInnerHTML
// tests, adapted to GWC's html.RawHTML — which renders raw markup but sanitizes
// it by default (a safety improvement over React's raw injection). html.RawHTMLUnsafe
// renders the markup verbatim.
func TestRawHTMLSSR(t *testing.T) {
	render := func(parseNodes []ui.Node) string {
		parseOut, parseErr := ui.RenderToString(html.Div(html.Props{}, parseNodes...))
		if parseErr != nil {
			t.Fatalf("render error: %v", parseErr)
		}
		return parseOut
	}

	// Safe markup is preserved.
	if parseGot := render(html.RawHTML("<b>bold</b>")); parseGot != "<div><b>bold</b></div>" {
		t.Errorf("safe RawHTML = %q, want <div><b>bold</b></div>", parseGot)
	}

	// A <script> is sanitized out; surrounding text is kept.
	if parseGot := render(html.RawHTML("<script>alert(1)</script>keep")); strings.Contains(parseGot, "<script") || !strings.Contains(parseGot, "keep") {
		t.Errorf("RawHTML did not sanitize <script>: %q", parseGot)
	}

	// A javascript: href is stripped by the sanitizer.
	if parseGot := render(html.RawHTML(`<a href="javascript:alert(1)">x</a>`)); strings.Contains(strings.ToLower(parseGot), "javascript:") {
		t.Errorf("RawHTML did not strip javascript: href: %q", parseGot)
	}

	// An on*= handler is stripped.
	if parseGot := render(html.RawHTML(`<img src=x onerror="alert(1)">`)); strings.Contains(strings.ToLower(parseGot), "onerror") {
		t.Errorf("RawHTML did not strip on*= handler: %q", parseGot)
	}

	// RawHTMLUnsafe renders verbatim (no sanitization).
	if parseGot := render(html.RawHTMLUnsafe("<b>raw</b>")); parseGot != "<div><b>raw</b></div>" {
		t.Errorf("RawHTMLUnsafe = %q, want <div><b>raw</b></div>", parseGot)
	}
}

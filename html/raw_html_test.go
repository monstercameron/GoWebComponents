//go:build !(js && wasm)

package html_test

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/html"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

// renderNodes wraps RawHTML output in a div and serializes it via the real SSR
// path, so tests assert on the actual rendered tree (built through the safe
// constructors), not on intermediate structures.
func renderNodes(t *testing.T, parseNodes []ui.Node) string {
	t.Helper()
	parseMarkup, parseErr := ui.RenderToString(html.Div(html.Props{ID: "host"}, parseNodes...))
	if parseErr != nil {
		t.Fatalf("render: %v", parseErr)
	}
	return parseMarkup
}

func TestRawHTMLBuildsRealNodes(t *testing.T) {
	parseOut := renderNodes(t, html.RawHTML(`<p class="lead">hello <b>world</b></p>`))
	for _, parseWant := range []string{`<p class="lead">`, `hello `, `<b>world</b>`, `</p>`} {
		if !strings.Contains(parseOut, parseWant) {
			t.Fatalf("missing %q in rendered output:\n%s", parseWant, parseOut)
		}
	}
}

func TestRawHTMLStripsScriptAndHandlers(t *testing.T) {
	parseOut := renderNodes(t, html.RawHTML(
		`<div onclick="steal()">ok<script>alert(1)</script><img src="javascript:evil()"></div>`,
	))
	for _, parseBad := range []string{"<script", "alert(1)", "onclick", "javascript:"} {
		if strings.Contains(parseOut, parseBad) {
			t.Fatalf("sanitizer let through %q:\n%s", parseBad, parseOut)
		}
	}
	if !strings.Contains(parseOut, "ok") {
		t.Fatalf("legitimate text was dropped:\n%s", parseOut)
	}
}

func TestRawHTMLEmptyAndWhitespace(t *testing.T) {
	if parseNodes := html.RawHTML(""); parseNodes != nil {
		t.Fatalf("empty input should yield nil, got %v", parseNodes)
	}
	if parseNodes := html.RawHTML("   \n\t "); parseNodes != nil {
		t.Fatalf("whitespace input should yield nil, got %v", parseNodes)
	}
}

func TestRawHTMLMalformedDoesNotPanicAndRecovers(t *testing.T) {
	defer func() {
		if parseR := recover(); parseR != nil {
			t.Fatalf("panicked on malformed markup: %v", parseR)
		}
	}()
	// Unclosed tags / stray markup: the parser recovers; we just must not panic.
	parseOut := renderNodes(t, html.RawHTML(`<p>unclosed <b>bold <i>italic`))
	if !strings.Contains(parseOut, "unclosed") {
		t.Fatalf("expected recovered text content:\n%s", parseOut)
	}
}

func TestRawHTMLStyleAttributeBecomesStyleMap(t *testing.T) {
	// A style attribute must be parsed into the Style map (not emitted as a bare
	// string that would break the style differ). DefaultPolicy strips style, so
	// use the unsafe path to exercise the attribute mapping.
	parseOut := renderNodes(t, html.RawHTMLUnsafe(`<span style="color: red; font-weight: bold">x</span>`))
	if !strings.Contains(parseOut, "color:red") || !strings.Contains(parseOut, "font-weight:bold") {
		t.Fatalf("inline style not mapped into style attribute:\n%s", parseOut)
	}
}

func TestRawHTMLUnsafeKeepsSVG(t *testing.T) {
	// The sanitized path drops <svg>; the trusted path keeps it (still as real
	// nodes). This is the icon/chart use case.
	parseSafe := renderNodes(t, html.RawHTML(`<svg><circle r="5"></circle></svg>`))
	if strings.Contains(parseSafe, "<svg") || strings.Contains(parseSafe, "<circle") {
		t.Fatalf("sanitized RawHTML should drop svg, got:\n%s", parseSafe)
	}
	parseTrusted := renderNodes(t, html.RawHTMLUnsafe(`<svg viewBox="0 0 10 10"><circle r="5"></circle></svg>`))
	if !strings.Contains(parseTrusted, "<svg") || !strings.Contains(parseTrusted, "<circle") {
		t.Fatalf("trusted RawHTMLUnsafe should keep svg:\n%s", parseTrusted)
	}
	if !strings.Contains(parseTrusted, `viewBox="0 0 10 10"`) {
		t.Fatalf("svg attributes should survive:\n%s", parseTrusted)
	}
}

func TestRawHTMLNestedAndAttributes(t *testing.T) {
	parseOut := renderNodes(t, html.RawHTML(
		`<ul><li id="a">one</li><li class="hi">two</li></ul>`,
	))
	for _, parseWant := range []string{`<ul>`, `<li id="a">one</li>`, `<li class="hi">two</li>`, `</ul>`} {
		if !strings.Contains(parseOut, parseWant) {
			t.Fatalf("missing %q in:\n%s", parseWant, parseOut)
		}
	}
}

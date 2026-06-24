//go:build !js || !wasm

package ui_test

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

// safeURLSentinel is the inert value the SSR serializer substitutes for a
// blocked script-executing URL. Kept in sync with internal/runtime/ssr.go.
const safeURLSentinel = "about:blank"

// urlAttrNode builds a host element carrying parseValue on the URL-bearing
// attribute parseAttr, using Props.Raw so attributes without a typed Props field
// (formaction, poster, object data, xlink:href) are covered too.
func urlAttrNode(parseTag, parseAttr, parseValue string) ui.Node {
	return html.Tag(parseTag, html.Props{Raw: map[string]any{parseAttr: parseValue}}, ui.Text("x"))
}

// TestURLSanitizationBlocksScriptSchemes mirrors React's
// ReactDOMServerIntegrationUntrustedURL suite: every script-executing URL,
// including the classic obfuscations (leading whitespace, embedded NUL and
// control characters, intermediate CR/LF, mixed casing), must be neutralized to
// the inert sentinel across every URL-bearing attribute, while legitimate URLs
// pass through untouched.
func TestURLSanitizationBlocksScriptSchemes(t *testing.T) {
	blocked := []struct {
		name  string
		value string
	}{
		{"plain", "javascript:notfine"},
		{"upper", "JAVASCRIPT:alert(1)"},
		{"mixed-case", "JavaScript:alert(1)"},
		{"leading-space", " javascript:alert(1)"},
		{"leading-ws-control", "  \t \x00\x1f\x03javascript\n: notfine"},
		{"interspersed-crlf", "\t\r\n Jav\rasCr\r\niP\t\n\rt\n:notfine"},
		{"tab-in-scheme", "java\tscript:alert(1)"},
		{"vbscript", "vbscript:msgbox(1)"},
		{"vbscript-case", "VBScript:msgbox(1)"},
	}

	// Attribute names paired with an element they legitimately appear on.
	attrs := []struct{ tag, attr string }{
		{"a", "href"},
		{"area", "href"},
		{"img", "src"},
		{"iframe", "src"},
		{"form", "action"},
		{"button", "formaction"},
		{"input", "formaction"},
		{"object", "data"},
		{"video", "poster"},
		{"link", "href"},
		{"use", "xlink:href"},
	}

	for _, parseAttr := range attrs {
		for _, parseCase := range blocked {
			out, err := ui.RenderToString(urlAttrNode(parseAttr.tag, parseAttr.attr, parseCase.value))
			if err != nil {
				t.Fatalf("%s/%s %s: unexpected render error: %v", parseAttr.tag, parseAttr.attr, parseCase.name, err)
			}
			if !strings.Contains(out, parseAttr.attr+`="`+safeURLSentinel+`"`) {
				t.Errorf("%s/%s %s: expected sentinel, got %q", parseAttr.tag, parseAttr.attr, parseCase.name, out)
			}
			if strings.Contains(strings.ToLower(out), "script:") {
				t.Errorf("%s/%s %s: script scheme survived: %q", parseAttr.tag, parseAttr.attr, parseCase.name, out)
			}
		}
	}
}

// TestURLSanitizationAllowsSafeURLs guards against over-blocking: legitimate URLs
// (and the deliberately-tricky http://javascript:0/ which only contains the word
// "javascript") must render unchanged.
func TestURLSanitizationAllowsSafeURLs(t *testing.T) {
	allowed := []struct {
		name  string
		value string
	}{
		{"http", "http://example.com/p"},
		{"https", "https://example.com/p?q=1#h"},
		{"protocol-relative", "//example.com/p"},
		{"root-relative", "/dashboard"},
		{"dot-relative", "./sibling"},
		{"fragment", "#section"},
		{"query", "?tab=2"},
		{"mailto", "mailto:user@example.com"},
		{"tel", "tel:+15551234567"},
		{"data-image", "data:image/png;base64,iVBORw0KGgo="},
		{"word-javascript-in-http", "http://javascript:0/thisisfine"},
		{"empty", ""},
	}
	for _, parseCase := range allowed {
		out, err := ui.RenderToString(urlAttrNode("a", "href", parseCase.value))
		if err != nil {
			t.Fatalf("%s: unexpected render error: %v", parseCase.name, err)
		}
		if !strings.Contains(out, `href="`+parseCase.value+`"`) {
			t.Errorf("%s: safe URL was altered: input %q -> %q", parseCase.name, parseCase.value, out)
		}
	}
}

// TestURLSanitizationLeavesNonURLAttributesAlone confirms the denylist is scoped
// to URL-bearing attributes: a "javascript:" string in a non-URL attribute (e.g.
// title, or a data-* attribute) is inert and must not be rewritten.
func TestURLSanitizationLeavesNonURLAttributesAlone(t *testing.T) {
	for _, parseAttr := range []string{"title", "data-url", "id", "alt"} {
		out, err := ui.RenderToString(urlAttrNode("div", parseAttr, "javascript:alert(1)"))
		if err != nil {
			t.Fatalf("%s: render error: %v", parseAttr, err)
		}
		if strings.Contains(out, safeURLSentinel) {
			t.Errorf("%s: non-URL attribute was wrongly sanitized: %q", parseAttr, out)
		}
	}
}

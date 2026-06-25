//go:build !js || !wasm

package ui_test

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

// TestTextContentEscaping ports React's text-escaping guarantee
// (escapeTextForBrowser): &, <, >, ", ' in text content are entity-encoded so a
// payload in any text position — a bare Text node, an element child, a nested
// child, or one of several sibling children — can never open a markup tag.
func TestTextContentEscaping(t *testing.T) {
	const payload = `<script>alert("x&y's")</script>`
	const wantEscaped = `&lt;script&gt;alert(&#34;x&amp;y&#39;s&#34;)&lt;/script&gt;`

	cases := []struct {
		name string
		node ui.Node
	}{
		{"bare-text", ui.Text(payload)},
		{"element-child", html.Div(html.Props{}, ui.Text(payload))},
		{"nested-child", html.Div(html.Props{}, html.Span(html.Props{}, ui.Text(payload)))},
	}
	for _, parseCase := range cases {
		out, err := ui.RenderToString(parseCase.node)
		if err != nil {
			t.Fatalf("%s: render error: %v", parseCase.name, err)
		}
		if !strings.Contains(out, wantEscaped) {
			t.Errorf("%s: expected escaped payload in %q", parseCase.name, out)
		}
		if strings.Contains(out, "<script") {
			t.Errorf("%s: raw <script survived: %q", parseCase.name, out)
		}
	}
}

// TestTextContentMultipleSiblings confirms each sibling text child is escaped
// independently (no concatenation gap re-opens an injection point).
func TestTextContentMultipleSiblings(t *testing.T) {
	out, err := ui.RenderToString(html.Div(html.Props{},
		ui.Text("a<b"), ui.Text("c&d"), ui.Text("e>f")))
	if err != nil {
		t.Fatalf("render error: %v", err)
	}
	const want = "<div>a&lt;bc&amp;de&gt;f</div>"
	if out != want {
		t.Errorf("expected %q, got %q", want, out)
	}
}

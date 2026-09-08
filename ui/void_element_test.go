//go:build !js || !wasm

package ui_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// TestVoidElementSerialization locks in correct HTML void-element handling: the
// 14 spec void elements render with no closing tag, any children handed to a
// void element are dropped (an <img>text</img> would be invalid markup), and
// non-void elements always get an explicit closing tag.
func TestVoidElementSerialization(t *testing.T) {
	cases := []struct {
		name string
		node ui.Node
		want string
	}{
		{"img", html.Img(html.Props{Src: "/a.png"}), `<img src="/a.png">`},
		{"br", html.Br(html.Props{}), "<br>"},
		{"hr", html.Hr(html.Props{}), "<hr>"},
		{"input", html.Input(html.Props{Type: "text"}), `<input type="text">`},
		{"void-drops-children", html.Tag("img", html.Props{Src: "/a.png"}, ui.Text("nope")), `<img src="/a.png">`},
		{"br-drops-children", html.Tag("br", html.Props{}, ui.Text("nope")), "<br>"},
		{"div-closes", html.Div(html.Props{}), "<div></div>"},
		{"custom-closes", html.Tag("custom-el", html.Props{}), "<custom-el></custom-el>"},
	}
	for _, parseCase := range cases {
		out, err := ui.RenderToString(parseCase.node)
		if err != nil {
			t.Fatalf("%s: render error: %v", parseCase.name, err)
		}
		if out != parseCase.want {
			t.Errorf("%s: want %q, got %q", parseCase.name, parseCase.want, out)
		}
	}
}

// TestAllVoidElementsHaveNoClosingTag exhaustively checks every HTML void
// element so a future edit to the void-element set can't silently regress one
// into emitting a spurious </tag>.
func TestAllVoidElementsHaveNoClosingTag(t *testing.T) {
	voids := []string{"area", "base", "br", "col", "embed", "hr", "img",
		"input", "link", "meta", "param", "source", "track", "wbr"}
	for _, parseTag := range voids {
		out, err := ui.RenderToString(html.Tag(parseTag, html.Props{}))
		if err != nil {
			t.Fatalf("%s: render error: %v", parseTag, err)
		}
		if want := "<" + parseTag + ">"; out != want {
			t.Errorf("%s: want %q, got %q", parseTag, want, out)
		}
	}
}

package shorthand

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

func TestMixedArgumentTagsNormalizeOptionsAndChildren(t *testing.T) {
	node := Div(
		Class("panel"),
		"hello ",
		Span(Class("value"), "world"),
		[]interface{}{" ", Textf("%d", 2)},
	)

	markup, err := ui.RenderToString(node)
	if err != nil {
		t.Fatalf("RenderToString returned error: %v", err)
	}

	const want = `<div class="panel">hello <span class="value">world</span> 2</div>`
	if markup != want {
		t.Fatalf("unexpected markup\nwant: %s\n got: %s", want, markup)
	}
}

func TestFromPropsPreservesExplicitZeroOverrides(t *testing.T) {
	node := Button(
		FromProps(Props{Class: "base", Disabled: true}),
		Disabled(false),
		Class("override"),
		"Save",
	)

	markup, err := ui.RenderToString(node)
	if err != nil {
		t.Fatalf("RenderToString returned error: %v", err)
	}

	const want = `<button class="override">Save</button>`
	if markup != want {
		t.Fatalf("unexpected markup\nwant: %s\n got: %s", want, markup)
	}
}

func TestShorthandParityWithTypedHTML(t *testing.T) {
	explicit := html.Div(html.Props{Class: "panel"},
		html.H2(html.Props{}, html.Text("Title")),
		html.Button(html.Props{Type: "button"}, html.Text("Save")),
	)
	shorthand := Div(
		Class("panel"),
		H2("Title"),
		Button(Type("button"), "Save"),
	)

	explicitMarkup, err := ui.RenderToString(explicit)
	if err != nil {
		t.Fatalf("RenderToString explicit returned error: %v", err)
	}
	shorthandMarkup, err := ui.RenderToString(shorthand)
	if err != nil {
		t.Fatalf("RenderToString shorthand returned error: %v", err)
	}

	if shorthandMarkup != explicitMarkup {
		t.Fatalf("unexpected parity mismatch\nwant: %s\n got: %s", explicitMarkup, shorthandMarkup)
	}
}

func TestVoidTagsStaySimple(t *testing.T) {
	markup, err := ui.RenderToString(Input(Type("text"), Value("hello"), Attr("data-mode", "demo")))
	if err != nil {
		t.Fatalf("RenderToString returned error: %v", err)
	}

	if !strings.HasPrefix(markup, `<input`) ||
		!strings.Contains(markup, `type="text"`) ||
		!strings.Contains(markup, `value="hello"`) ||
		!strings.Contains(markup, `data-mode="demo"`) {
		t.Fatalf("unexpected markup: %s", markup)
	}
}

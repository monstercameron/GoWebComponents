package shorthand

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

func TestMixedArgumentTagsNormalizeOptionsAndChildren(parseT *testing.T) {
	parseNode := Div(
		ClassStr("panel"),
		"hello ",
		Span(ClassStr("value"), "world"),
		[]any{" ", Textf("%d", 2)},
	)

	parseMarkup, parseErr := ui.RenderToString(parseNode)
	if parseErr != nil {
		parseT.Fatalf("RenderToString returned error: %v", parseErr)
	}

	const want = `<div class="panel">hello <span class="value">world</span> 2</div>`
	if parseMarkup != want {
		parseT.Fatalf("unexpected markup\nwant: %s\n got: %s", want, parseMarkup)
	}
}

func TestFromPropsPreservesExplicitZeroOverrides(parseT *testing.T) {
	parseNode := Button(
		FromProps(Props{Class: "base", Disabled: true}),
		Disabled(false),
		ClassStr("override"),
		"Save",
	)

	parseMarkup, parseErr := ui.RenderToString(parseNode)
	if parseErr != nil {
		parseT.Fatalf("RenderToString returned error: %v", parseErr)
	}

	const want = `<button class="override">Save</button>`
	if parseMarkup != want {
		parseT.Fatalf("unexpected markup\nwant: %s\n got: %s", want, parseMarkup)
	}
}

func TestShorthandParityWithTypedHTML(parseT *testing.T) {
	parseExplicit := html.Div(html.Props{Class: "panel"},
		html.H2(html.Props{}, html.Text("Title")),
		html.Button(html.Props{Type: "button"}, html.Text("Save")),
	)
	parseShorthand := Div(
		ClassStr("panel"),
		H2("Title"),
		Button(Type("button"), "Save"),
	)

	parseExplicitMarkup, parseErr := ui.RenderToString(parseExplicit)
	if parseErr != nil {
		parseT.Fatalf("RenderToString explicit returned error: %v", parseErr)
	}
	parseShorthandMarkup, parseErr := ui.RenderToString(parseShorthand)
	if parseErr != nil {
		parseT.Fatalf("RenderToString shorthand returned error: %v", parseErr)
	}

	if parseShorthandMarkup != parseExplicitMarkup {
		parseT.Fatalf("unexpected parity mismatch\nwant: %s\n got: %s", parseExplicitMarkup, parseShorthandMarkup)
	}
}

func TestVoidTagsStaySimple(parseT *testing.T) {
	parseMarkup, parseErr := ui.RenderToString(Input(Type("text"), Value("hello"), Attr("data-mode", "demo")))
	if parseErr != nil {
		parseT.Fatalf("RenderToString returned error: %v", parseErr)
	}

	if !strings.HasPrefix(parseMarkup, `<input`) ||
		!strings.Contains(parseMarkup, `type="text"`) ||
		!strings.Contains(parseMarkup, `value="hello"`) ||
		!strings.Contains(parseMarkup, `data-mode="demo"`) {
		parseT.Fatalf("unexpected markup: %s", parseMarkup)
	}
}

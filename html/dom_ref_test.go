//go:build !(js && wasm)

package html_test

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

// On the native/SSR build a DOM ref never resolves to a node (there is no DOM),
// and its sink must never leak into the serialized HTML.
func TestDOMRefSSRDoesNotLeakAndStaysNull(t *testing.T) {
	var parseCaptured ui.DOMRef
	parseComponent := func(struct{}) ui.Node {
		parseRef := ui.UseDOMRef()
		parseCaptured = parseRef
		return shorthand.Input(
			shorthand.Ref(parseRef),
			shorthand.FromProps(shorthand.Props{ID: "field"}),
		)
	}

	parseMarkup, parseErr := ui.RenderToString(ui.CreateElement(parseComponent, struct{}{}))
	if parseErr != nil {
		t.Fatalf("render: %v", parseErr)
	}
	if strings.Contains(parseMarkup, "__gwc_dom_ref__") {
		t.Fatalf("ref sink leaked into SSR markup:\n%s", parseMarkup)
	}
	if !strings.Contains(parseMarkup, `id="field"`) {
		t.Fatalf("element did not render normally with a ref attached:\n%s", parseMarkup)
	}
	if parseCaptured.Mounted() {
		t.Fatal("ref should not be mounted on the native/SSR build")
	}
	if parseCaptured.Node() != nil {
		t.Fatalf("ref node should be nil on native, got %#v", parseCaptured.Node())
	}
}

// A zero-value DOMRef (not produced by UseDOMRef) must be a safe no-op.
func TestRefWithZeroDOMRefIsNoOp(t *testing.T) {
	var parseZero ui.DOMRef
	parseProps := html.PropsOf(html.Ref(parseZero), html.ID("x"))
	if parseProps.Raw != nil {
		if _, parseExists := parseProps.Raw["__gwc_dom_ref__"]; parseExists {
			t.Fatal("zero DOMRef should not register a ref sink")
		}
	}
	if parseProps.ID != "x" {
		t.Fatalf("other options should still apply; got ID=%q", parseProps.ID)
	}
	if parseZero.Mounted() {
		t.Fatal("zero DOMRef must not be mounted")
	}
}

//go:build js && wasm

package render_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/html"
	"github.com/monstercameron/GoWebComponents/v5/testkit/render"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

// TestMapKeyedComponentNilRender probes a row render that returns nil for some
// items (inline filtering): the other rows must render and there must be no panic.
func TestMapKeyedComponentNilRender(parseT *testing.T) {
	parseItems := []string{"a", "b", "c"}
	parseApp := func() ui.Node {
		parseKids := html.MapKeyedComponent(parseItems,
			func(parseID string) any { return parseID },
			func(parseID string) ui.Node {
				if parseID == "b" {
					return nil // filtered out
				}
				return html.P(html.Props{ID: "row-" + parseID}, html.Text(parseID))
			})
		return html.Div(html.Props{ID: "nil-list"}, parseKids...)
	}

	parseFixture := render.New(parseT)
	parseFixture.Render(ui.CreateElement(parseApp))
	parseFixture.Flush()

	if parseFixture.ByID("row-a") == nil {
		parseT.Fatal("row-a should render")
	}
	if parseFixture.ByID("row-c") == nil {
		parseT.Fatal("row-c should render")
	}
	if parseFixture.ByID("row-b") != nil {
		parseT.Fatal("row-b returned nil and must be absent")
	}
}

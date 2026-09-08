package ui_test

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// exampleCounter reads and updates its own state with the State handle
// returned by UseState. Hooks belong at the top level of a component function.
func exampleCounter(_ struct{}) ui.Node {
	parseCount := ui.UseState(0)
	return html.Div(html.Props{},
		html.Button(html.Props{}, html.Text(fmt.Sprintf("count: %d", parseCount.Get()))),
	)
}

// ExampleUseState shows local component state: a counter component reads and
// updates its own state with the State handle returned by UseState.
func ExampleUseState() {
	// Mount with ui.Render(ui.CreateElement(exampleCounter, struct{}{}), "#app")
	// in a browser; here we just build the element tree.
	_ = ui.CreateElement(exampleCounter, struct{}{})
}

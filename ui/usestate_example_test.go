package ui_test

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

// ExampleUseState shows local component state: a counter component reads and
// updates its own state with the State handle returned by UseState.
func ExampleUseState() {
	counter := func(_ struct{}) ui.Node {
		parseCount := ui.UseState(0)
		return html.Div(html.Props{},
			html.Button(html.Props{}, html.Text(fmt.Sprintf("count: %d", parseCount.Get()))),
		)
	}
	// Mount with ui.Render(ui.CreateElement(counter, struct{}{}), "#app") in a
	// browser; here we just build the element tree.
	_ = ui.CreateElement(counter, struct{}{})
}

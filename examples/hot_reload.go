// ./examples/hot_reload_test.go

//go:build js && wasm
// +build js,wasm

package example

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/dom"
	"github.com/monstercameron/GoWebComponents/hooks"
)

func StateTest() func(Attrs) *Element {
	return func(props Attrs) *Element {
		// Counter state
		count, setCount := hooks.UseState(0)

		// Input field state
		inputValue, setInputValue := hooks.UseState("")

		// Click handler for counter using GoUseFunc
		handleClick := hooks.GoUseFunc(func(event dom.GoEvent) {
			event.PreventDefault()
			currentCount := count()
			setCount(currentCount + 1)
		})

		// Change handler for input using GoUseFunc
		handleChange := hooks.GoUseFunc(func(event dom.GoEvent) {
			newValue := event.GetValue()
			setInputValue(newValue)
		})

		return dom.Div(nil,
			dom.H3(nil, dom.Text("State Test Component")),
			dom.Div(nil,
				dom.P(nil, dom.Text("Counter: "), dom.Text(fmt.Sprintf("%d", count()))),
				dom.Button(Attrs{
					"onclick": handleClick,
				}, dom.Text("Click me!")),
			),
			dom.Div(nil,
				dom.P(nil, dom.Text("Input: "), dom.Text(inputValue())),
				dom.Input(Attrs{
					"type":        "text",
					"placeholder": "Enter some text...1",
					"value":       inputValue(),
					"onchange":    handleChange,
				}),
			),
		)
	}
}

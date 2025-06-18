// ./examples/hot_reload_test.go

//go:build js && wasm
// +build js,wasm

package examples

import (
	"fmt"

	. "github.com/monstercameron/GoWebComponents/fiber"
)

func StateTest() func(Attrs) *Element {
	return func(props Attrs) *Element {
		// Counter state
		count, setCount := GoUseState(0)

		// Input field state
		inputValue, setInputValue := GoUseState("")

		// Click handler for counter using GoUseFunc
		handleClick := GoUseFunc(func(event GoEvent) {
			event.PreventDefault()
			currentCount := count()
			setCount(currentCount + 1)
		})

		// Change handler for input using GoUseFunc
		handleChange := GoUseFunc(func(event GoEvent) {
			newValue := event.GetValue()
			setInputValue(newValue)
		})

		return Div(nil,
			H3(nil, Text("State Test Component")),
			Div(nil,
				P(nil, Text("Counter: "), Text(fmt.Sprintf("%d", count()))),
				Button(Attrs{
					"onclick": handleClick,
				}, Text("Click me!")),
			),
			Div(nil,
				P(nil, Text("Input: "), Text(inputValue())),
				Input(Attrs{
					"type":        "text",
					"placeholder": "Enter some text...1",
					"value":       inputValue(),
					"onchange":    handleChange,
				}),
			),
		)
	}
}

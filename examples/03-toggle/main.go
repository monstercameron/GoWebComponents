//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/dom"
	"github.com/monstercameron/GoWebComponents/hooks"
	"github.com/monstercameron/GoWebComponents/render"
)

type Attrs = dom.Attrs
type Element = render.Element

func ToggleExample(_ Attrs) *Element {
	isOn, setIsOn := hooks.UseState(false)
	currentState := isOn()

	toggle := func(this js.Value, args []js.Value) interface{} {
		setIsOn(!currentState)
		return nil
	}

	return dom.Div(Attrs{
		"class": "max-w-md mx-auto mt-10 p-6 bg-white rounded-lg shadow-lg",
	},
		dom.H2(Attrs{
			"class": "text-2xl font-bold text-gray-800 mb-4",
		}, dom.Text("Toggle Example")),

		dom.Div(Attrs{
			"class": "text-center mb-6",
		},
			dom.Div(Attrs{
				"class": func() string {
					if currentState {
						return "text-6xl mb-4 text-green-500"
					}
					return "text-6xl mb-4 text-red-500"
				}(),
			}, dom.Text(func() string {
				if currentState {
					return "ON"
				}
				return "OFF"
			}())),
			dom.P(Attrs{
				"class": "text-gray-600",
			}, dom.Text(fmt.Sprintf("Switch is: %s", func() string {
				if currentState {
					return "ON"
				}
				return "OFF"
			}()))),
		),

		dom.Button(Attrs{
			"onclick": js.FuncOf(toggle),
			"class": func() string {
				if currentState {
					return "w-full px-6 py-3 bg-red-500 text-white font-semibold rounded-lg hover:bg-red-600 transition-colors"
				}
				return "w-full px-6 py-3 bg-green-500 text-white font-semibold rounded-lg hover:bg-green-600 transition-colors"
			}(),
		}, dom.Text(func() string {
			if currentState {
				return "Turn OFF"
			}
			return "Turn ON"
		}())),
	)
}

func main() {
	fmt.Println("Toggle Example Started")
	container := js.Global().Get("document").Call("getElementById", "app")
	if container.IsUndefined() || container.IsNull() {
		fmt.Println("No element with id 'app' found!")
		return
	}
	element := dom.CreateElement(ToggleExample, nil)
	render.ToElement(element, container)
	fmt.Println("Toggle Example Rendered")
	select {}
}

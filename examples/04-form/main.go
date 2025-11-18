//go:build js && wasm

package main

import (
	"fmt"
	"strconv"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/dom"
	"github.com/monstercameron/GoWebComponents/hooks"
	"github.com/monstercameron/GoWebComponents/render"
)

type Attrs = dom.Attrs
type Element = render.Element

type Person struct {
	Name string
	Age  int
}

func PersonForm(_ Attrs) *Element {
	person, setPerson := hooks.UseState(Person{Name: "", Age: 0})
	currentPerson := person()

	updateName := func(this js.Value, args []js.Value) interface{} {
		if len(args) > 0 {
			newName := args[0].Get("target").Get("value").String()
			setPerson(Person{Name: newName, Age: currentPerson.Age})
		}
		return nil
	}

	updateAge := func(this js.Value, args []js.Value) interface{} {
		if len(args) > 0 {
			ageStr := args[0].Get("target").Get("value").String()
			if age, err := strconv.Atoi(ageStr); err == nil {
				setPerson(Person{Name: currentPerson.Name, Age: age})
			}
		}
		return nil
	}

	reset := func(this js.Value, args []js.Value) interface{} {
		setPerson(Person{Name: "", Age: 0})
		return nil
	}

	return dom.Div(Attrs{
		"class": "max-w-md mx-auto mt-8 p-6 bg-white rounded-lg shadow-lg",
	},
		dom.H2(Attrs{
			"class": "text-2xl font-bold mb-6 text-gray-800",
		}, dom.Text("Person Form")),
		
		dom.Div(Attrs{"class": "mb-4"},
			dom.Label(Attrs{
				"class": "block text-gray-700 font-medium mb-2",
			}, dom.Text("Name:")),
			dom.Input(Attrs{
				"type":    "text",
				"value":   currentPerson.Name,
				"oninput": js.FuncOf(updateName),
				"class":   "w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500",
			}),
		),

		dom.Div(Attrs{"class": "mb-4"},
			dom.Label(Attrs{
				"class": "block text-gray-700 font-medium mb-2",
			}, dom.Text("Age:")),
			dom.Input(Attrs{
				"type":    "number",
				"value":   strconv.Itoa(currentPerson.Age),
				"oninput": js.FuncOf(updateAge),
				"class":   "w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500",
			}),
		),

		dom.Div(Attrs{"class": "flex gap-4 mb-4"},
			dom.Button(Attrs{
				"onclick": js.FuncOf(reset),
				"class":   "px-4 py-2 bg-red-500 text-white rounded-lg hover:bg-red-600 transition-colors",
			}, dom.Text("Reset")),
		),

		dom.Div(Attrs{
			"class": "p-4 bg-gray-50 rounded-lg border border-gray-200",
		},
			dom.P(Attrs{
				"class": "text-gray-700",
			}, dom.Text(fmt.Sprintf("Name: %s", currentPerson.Name))),
			dom.P(Attrs{
				"class": "text-gray-700",
			}, dom.Text(fmt.Sprintf("Age: %d", currentPerson.Age))),
		),
	)
}

func main() {
	container := js.Global().Get("document").Call("getElementById", "app")
	element := dom.CreateElement(PersonForm, nil)
	render.ToElement(element, container)
	select {}
}

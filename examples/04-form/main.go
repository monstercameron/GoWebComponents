//go:build js && wasm

package main

import (
	"fmt"
	"strconv"

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

	updateName := hooks.GoUseFunc(func(event dom.GoEvent) {
		newName := event.GetValue()
		setPerson(Person{Name: newName, Age: currentPerson.Age})
	})

	updateAge := hooks.GoUseFunc(func(event dom.GoEvent) {
		ageStr := event.GetValue()
		if age, err := strconv.Atoi(ageStr); err == nil {
			setPerson(Person{Name: currentPerson.Name, Age: age})
		}
	})

	reset := hooks.GoUseFunc(func(event dom.GoEvent) {
		setPerson(Person{Name: "", Age: 0})
	})

	return dom.Div(Attrs{
		"class": "min-h-screen flex items-center justify-center bg-[#0a0a0a] text-white p-4",
	},
		dom.Div(Attrs{
			"class": "max-w-md w-full bg-white/5 border border-white/10 rounded-xl backdrop-blur-sm p-8 shadow-2xl",
		},
			dom.H2(Attrs{
				"class": "text-3xl font-bold text-center mb-8 bg-clip-text text-transparent bg-gradient-to-r from-blue-400 to-purple-500",
			}, dom.Text("Person Form")),

			dom.Div(Attrs{"class": "space-y-6 mb-8"},
				dom.Div(nil,
					dom.Label(Attrs{
						"class": "block text-gray-400 text-sm font-bold mb-2 uppercase tracking-wider",
					}, dom.Text("Name")),
					dom.Input(Attrs{
						"type":        "text",
						"value":       currentPerson.Name,
						"oninput":     updateName,
						"class":       "w-full px-4 py-3 bg-black/20 border border-white/10 rounded-lg focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 text-white placeholder-gray-600 transition-all",
						"placeholder": "Enter name",
					}),
				),

				dom.Div(nil,
					dom.Label(Attrs{
						"class": "block text-gray-400 text-sm font-bold mb-2 uppercase tracking-wider",
					}, dom.Text("Age")),
					dom.Input(Attrs{
						"type":    "number",
						"value":   strconv.Itoa(currentPerson.Age),
						"oninput": updateAge,
						"class":   "w-full px-4 py-3 bg-black/20 border border-white/10 rounded-lg focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 text-white placeholder-gray-600 transition-all",
					}),
				),
			),

			dom.Div(Attrs{"class": "mb-8"},
				dom.Button(Attrs{
					"onclick": reset,
					"class":   "w-full px-4 py-3 bg-white/5 hover:bg-white/10 border border-white/10 text-gray-300 font-semibold rounded-lg transition-colors",
				}, dom.Text("Reset Form")),
			),

			dom.Div(Attrs{
				"class": "p-6 bg-black/20 rounded-lg border border-white/5",
			},
				dom.H3(Attrs{
					"class": "text-gray-400 text-xs uppercase tracking-widest mb-4 border-b border-white/5 pb-2",
				}, dom.Text("Current State")),

				dom.Div(Attrs{"class": "space-y-2"},
					dom.Div(Attrs{"class": "flex justify-between"},
						dom.Span(Attrs{"class": "text-gray-500"}, dom.Text("Name")),
						dom.Span(Attrs{"class": "text-white font-medium"}, dom.Text(func() string {
							if currentPerson.Name == "" {
								return "-"
							}
							return currentPerson.Name
						}())),
					),
					dom.Div(Attrs{"class": "flex justify-between"},
						dom.Span(Attrs{"class": "text-gray-500"}, dom.Text("Age")),
						dom.Span(Attrs{"class": "text-blue-400 font-mono"}, dom.Text(fmt.Sprintf("%d", currentPerson.Age))),
					),
				),
			),
		),
	)
}

func main() {
	render.To(dom.CreateElement(PersonForm, nil), "#app")
	select {}
}

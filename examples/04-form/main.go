//go:build js && wasm

package main

import (
	"fmt"
	"strconv"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

type Person struct {
	Name string
	Age  int
}

func PersonForm() ui.Node {
	person := ui.UseState(Person{Name: "", Age: 0})
	currentPerson := person.Get()

	updateName := ui.UseEvent(func(event ui.InputEvent) {
		newName := event.GetValue()
		person.Set(Person{Name: newName, Age: currentPerson.Age})
	})

	updateAge := ui.UseEvent(func(event ui.InputEvent) {
		ageStr := event.GetValue()
		if age, err := strconv.Atoi(ageStr); err == nil {
			person.Set(Person{Name: currentPerson.Name, Age: age})
		}
	})

	reset := ui.UseEvent(func() {
		person.Set(Person{Name: "", Age: 0})
	})

	return html.Div(html.Props{
		Class: "min-h-screen flex items-center justify-center bg-[#0a0a0a] text-white p-4",
	},
		html.Div(html.Props{
			Class: "max-w-md w-full bg-white/5 border border-white/10 rounded-xl backdrop-blur-sm p-8 shadow-2xl",
		},
			html.H2(html.Props{
				Class: "text-3xl font-bold text-center mb-8 bg-clip-text text-transparent bg-gradient-to-r from-blue-400 to-purple-500",
			}, html.Text("Person Form")),

			html.Div(html.Props{Class: "space-y-6 mb-8"},
				html.Div(html.Props{},
					html.Label(html.Props{
						Class: "block text-gray-400 text-sm font-bold mb-2 uppercase tracking-wider",
					}, html.Text("Name")),
					html.Input(html.Props{
						Type:        "text",
						Value:       currentPerson.Name,
						OnInput:     updateName,
						Class:       "w-full px-4 py-3 bg-black/20 border border-white/10 rounded-lg focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 text-white placeholder-gray-600 transition-all",
						Placeholder: "Enter name",
					}),
				),

				html.Div(html.Props{},
					html.Label(html.Props{
						Class: "block text-gray-400 text-sm font-bold mb-2 uppercase tracking-wider",
					}, html.Text("Age")),
					html.Input(html.Props{
						Type:    "number",
						Value:   strconv.Itoa(currentPerson.Age),
						OnInput: updateAge,
						Class:   "w-full px-4 py-3 bg-black/20 border border-white/10 rounded-lg focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 text-white placeholder-gray-600 transition-all",
					}),
				),
			),

			html.Div(html.Props{Class: "mb-8"},
				html.Button(html.Props{
					OnClick: reset,
					Class:   "w-full px-4 py-3 bg-white/5 hover:bg-white/10 border border-white/10 text-gray-300 font-semibold rounded-lg transition-colors",
				}, html.Text("Reset Form")),
			),

			html.Div(html.Props{
				Class: "p-6 bg-black/20 rounded-lg border border-white/5",
			},
				html.H3(html.Props{
					Class: "text-gray-400 text-xs uppercase tracking-widest mb-4 border-b border-white/5 pb-2",
				}, html.Text("Current State")),

				html.Div(html.Props{Class: "space-y-2"},
					html.Div(html.Props{Class: "flex justify-between"},
						html.Span(html.Props{Class: "text-gray-500"}, html.Text("Name")),
						html.Span(html.Props{Class: "text-white font-medium"}, html.Text(func() string {
							if currentPerson.Name == "" {
								return "-"
							}
							return currentPerson.Name
						}())),
					),
					html.Div(html.Props{Class: "flex justify-between"},
						html.Span(html.Props{Class: "text-gray-500"}, html.Text("Age")),
						html.Span(html.Props{Class: "text-blue-400 font-mono"}, html.Text(fmt.Sprintf("%d", currentPerson.Age))),
					),
				),
			),
		),
	)
}

func main() {
	ui.Render(ui.CreateElement(PersonForm), "#app")
	select {}
}

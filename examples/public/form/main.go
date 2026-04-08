//go:build js && wasm

package main

import (
	"fmt"
	"strconv"

	"github.com/monstercameron/GoWebComponents/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/examples/shared"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

type Person struct {
	Name string
	Age  int
}

func PersonForm() ui.Node {
	parsePerson := ui.UseState(Person{Name: "", Age: 0})
	parseCurrentPerson := parsePerson.Get()

	parseUpdateName := ui.UseEvent(func(parseEvent ui.InputEvent) {
		parseNewName := parseEvent.GetValue()
		parsePerson.Set(Person{Name: parseNewName, Age: parseCurrentPerson.Age})
	})

	parseUpdateAge := ui.UseEvent(func(parseEvent2 ui.InputEvent) {
		parseAgeStr := parseEvent2.GetValue()
		if parseAge, parseErr := strconv.Atoi(parseAgeStr); parseErr == nil {
			parsePerson.Set(Person{Name: parseCurrentPerson.Name, Age: parseAge})
		}
	})

	reset := ui.UseEvent(func() {
		parsePerson.Set(Person{Name: "", Age: 0})
	})

	return shared.ExamplePage(
		"Form",
		"ui.UseState",
		"Update structured form state from independent inputs and reflect it in one live snapshot.",
		shared.ExamplePanel("Fields",
			html.Div(html.PropsOf(html.Class("space-y-6")),
				html.Div(html.Props{},
					html.Label(html.PropsOf(
						html.Class("block text-xs font-semibold uppercase tracking-[0.18em] text-slate-400"),
					), html.Text("Name")),
					html.Input(html.PropsOf(
						html.Type("text"),
						html.Value(parseCurrentPerson.Name),
						html.OnInput(parseUpdateName),
						html.Class("mt-2 w-full rounded-2xl border border-white/10 bg-slate-950/70 px-4 py-3 text-slate-100 placeholder:text-slate-500 focus:outline-none"),
						html.Placeholder("Enter name"),
					)),
				),

				html.Div(html.Props{},
					html.Label(html.PropsOf(
						html.Class("block text-xs font-semibold uppercase tracking-[0.18em] text-slate-400"),
					), html.Text("Age")),
					html.Input(html.PropsOf(
						html.Type("number"),
						html.Value(strconv.Itoa(parseCurrentPerson.Age)),
						html.OnInput(parseUpdateAge),
						html.Class("mt-2 w-full rounded-2xl border border-white/10 bg-slate-950/70 px-4 py-3 text-slate-100 placeholder:text-slate-500 focus:outline-none"),
					)),
				),
			),
			html.Div(html.PropsOf(html.Class("flex flex-wrap gap-2")),
				html.Button(html.PropsOf(
					html.OnClick(reset),
					html.Class("rounded-2xl border border-white/10 bg-white/5 px-4 py-2 text-sm font-medium text-slate-200 transition hover:border-cyan-300/30 hover:text-cyan-100"),
				), html.Text("Reset Form")),
			),
		),
		shared.ExamplePanel("Current State",
			html.Div(html.Props{Class: "grid gap-3 sm:grid-cols-2"},
				shared.ExampleStat("Name", func() string {
					if parseCurrentPerson.Name == "" {
						return "-"
					}
					return parseCurrentPerson.Name
				}()),
				shared.ExampleStat("Age", fmt.Sprintf("%d", parseCurrentPerson.Age)),
			),
		),
	)
}

func main() {
	exampleboot.RenderExampleRoot(ui.CreateElement(PersonForm))
	exampleboot.WaitExampleRuntime()
}

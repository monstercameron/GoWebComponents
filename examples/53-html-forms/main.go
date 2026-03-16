//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

func htmlFormsExample() ui.Node {
	name := ui.UseState("")
	team := ui.UseState("platform")
	notes := ui.UseState("")
	updates := ui.UseState(false)

	setName := ui.UseEvent(func(event ui.InputEvent) { name.Set(event.GetValue()) })
	setTeam := ui.UseEvent(func(event ui.ChangeEvent) { team.Set(event.GetValue()) })
	setNotes := ui.UseEvent(func(event ui.InputEvent) { notes.Set(event.GetValue()) })
	setUpdates := ui.UseEvent(func(event ui.ChangeEvent) { updates.Set(event.IsChecked()) })

	return shared.ExamplePage(
		"html form controls",
		"Use typed html.Props for labels, inputs, selects, checkboxes, and textareas",
		"The html package gives you typed props instead of stringly-typed attr maps, which keeps common form composition readable while still mapping closely to the resulting markup.",
		shared.ExamplePanel("Typed controls",
			html.Div(html.Props{Class: "mt-3 grid gap-4"},
				html.Div(html.Props{},
					html.Label(html.Props{For: "html-form-name", Class: "block text-sm uppercase tracking-[0.25em] text-slate-400"}, html.Text("Name")),
					html.Input(html.Props{ID: "html-form-name", Value: name.Get(), OnInput: setName, Placeholder: "Cam", Class: "mt-2 w-full rounded-2xl border border-white/10 bg-slate-950/70 px-4 py-3 text-slate-100"}),
				),
				html.Div(html.Props{},
					html.Label(html.Props{For: "html-form-team", Class: "block text-sm uppercase tracking-[0.25em] text-slate-400"}, html.Text("Team")),
					html.Select(html.Props{ID: "html-form-team", Value: team.Get(), OnChange: setTeam, Class: "mt-2 w-full rounded-2xl border border-white/10 bg-slate-950/70 px-4 py-3 text-slate-100"},
						html.Option(html.Props{Value: "platform"}, html.Text("Platform")),
						html.Option(html.Props{Value: "design"}, html.Text("Design")),
						html.Option(html.Props{Value: "ops"}, html.Text("Operations")),
					),
				),
				html.Label(html.Props{Class: "flex items-center gap-3 rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3"},
					html.Input(html.Props{Type: "checkbox", Checked: updates.Get(), OnChange: setUpdates}),
					html.Text("Receive weekly release notes"),
				),
				html.Div(html.Props{},
					html.Label(html.Props{For: "html-form-notes", Class: "block text-sm uppercase tracking-[0.25em] text-slate-400"}, html.Text("Notes")),
					html.Textarea(html.Props{ID: "html-form-notes", Rows: 5, Value: notes.Get(), OnInput: setNotes, Placeholder: "What should the team know?", Class: "mt-2 w-full rounded-2xl border border-white/10 bg-slate-950/70 px-4 py-3 text-slate-100"}),
				),
			),
		),
		shared.ExamplePanel("Current values",
			html.Div(html.Props{Class: "mt-3 grid gap-4 md:grid-cols-4"},
				shared.ExampleStat("Name", name.Get()),
				shared.ExampleStat("Team", team.Get()),
				shared.ExampleStat("Updates", fmt.Sprintf("%t", updates.Get())),
				shared.ExampleStat("Notes chars", fmt.Sprintf("%d", len(notes.Get()))),
			),
			shared.ExampleCode(
				`html.Input(html.Props{Value: name, OnInput: setName})`,
				`html.Select(html.Props{Value: team, OnChange: setTeam}, ...)`,
				`html.Textarea(html.Props{Rows: 5, Value: notes, OnInput: setNotes})`,
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(htmlFormsExample), "#app")
	select {}
}

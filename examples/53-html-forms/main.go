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
			html.Div(html.PropsOf(html.Class("mt-3 grid gap-4")),
				html.Div(html.Props{},
					html.Label(html.PropsOf(
						html.For("html-form-name"),
						html.Class("block text-sm uppercase tracking-[0.25em] text-slate-400"),
					), html.Text("Name")),
					html.Input(html.PropsOf(
						html.ID("html-form-name"),
						html.Value(name.Get()),
						html.OnInput(setName),
						html.Placeholder("Cam"),
						html.Class("mt-2 w-full rounded-2xl border border-white/10 bg-slate-950/70 px-4 py-3 text-slate-100"),
					)),
				),
				html.Div(html.Props{},
					html.Label(html.PropsOf(
						html.For("html-form-team"),
						html.Class("block text-sm uppercase tracking-[0.25em] text-slate-400"),
					), html.Text("Team")),
					html.Select(html.PropsOf(
						html.ID("html-form-team"),
						html.Value(team.Get()),
						html.OnChange(setTeam),
						html.Class("mt-2 w-full rounded-2xl border border-white/10 bg-slate-950/70 px-4 py-3 text-slate-100"),
					),
						html.Option(html.PropsOf(html.Value("platform")), html.Text("Platform")),
						html.Option(html.PropsOf(html.Value("design")), html.Text("Design")),
						html.Option(html.PropsOf(html.Value("ops")), html.Text("Operations")),
					),
				),
				html.Label(html.PropsOf(html.Class("flex items-center gap-3 rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3")),
					html.Input(html.PropsOf(
						html.Type("checkbox"),
						html.Checked(updates.Get()),
						html.OnChange(setUpdates),
					)),
					html.Text("Receive weekly release notes"),
				),
				html.Div(html.Props{},
					html.Label(html.PropsOf(
						html.For("html-form-notes"),
						html.Class("block text-sm uppercase tracking-[0.25em] text-slate-400"),
					), html.Text("Notes")),
					html.Textarea(html.PropsOf(
						html.ID("html-form-notes"),
						html.Rows(5),
						html.Value(notes.Get()),
						html.OnInput(setNotes),
						html.Placeholder("What should the team know?"),
						html.Class("mt-2 w-full rounded-2xl border border-white/10 bg-slate-950/70 px-4 py-3 text-slate-100"),
					)),
				),
			),
		),
		shared.ExamplePanel("Current values",
			html.Div(html.PropsOf(html.Class("mt-3 grid gap-4 md:grid-cols-4")),
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

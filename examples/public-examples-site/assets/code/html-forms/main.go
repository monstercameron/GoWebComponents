//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/v6/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v6/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/v6/examples/shared"
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/ui"
	"github.com/monstercameron/GoWebComponents/v6/utils"
)

func htmlFormsExample() ui.Node {
	parseName := ui.UseState("")
	parseTeam := ui.UseState("platform")
	parseNotes := ui.UseState("")
	parseUpdates := ui.UseState(false)

	setName := ui.UseEvent(func(parseEvent ui.InputEvent) { parseName.Set(parseEvent.GetValue()) })
	setTeam := ui.UseEvent(func(parseEvent2 ui.ChangeEvent) { parseTeam.Set(parseEvent2.GetValue()) })
	setNotes := ui.UseEvent(func(parseEvent3 ui.InputEvent) { parseNotes.Set(parseEvent3.GetValue()) })
	setUpdates := ui.UseEvent(func(parseEvent4 ui.ChangeEvent) { parseUpdates.Set(parseEvent4.IsChecked()) })

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
						html.Value(parseName.Get()),
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
						html.Value(parseTeam.Get()),
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
						html.Checked(parseUpdates.Get()),
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
						html.Value(parseNotes.Get()),
						html.OnInput(setNotes),
						html.Placeholder("What should the team know?"),
						html.Class("mt-2 w-full rounded-2xl border border-white/10 bg-slate-950/70 px-4 py-3 text-slate-100"),
					)),
				),
			),
		),
		shared.ExamplePanel("Current values",
			html.Div(html.PropsOf(html.Class("mt-3 grid gap-4 md:grid-cols-4")),
				shared.ExampleStat("Name", parseName.Get()),
				shared.ExampleStat("Team", parseTeam.Get()),
				shared.ExampleStat("Updates", fmt.Sprintf("%t", parseUpdates.Get())),
				shared.ExampleStat("Notes chars", fmt.Sprintf("%d", len(parseNotes.Get()))),
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
	exampleboot.RenderExampleRoot(ui.CreateElement(htmlFormsExample))
	exampleboot.WaitExampleRuntime()
}

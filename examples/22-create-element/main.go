//go:build js && wasm
// +build js,wasm

package main

import (
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"strings"

	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

type badgeProps struct {
	Label string
	Tone  string
}

func badge(props badgeProps) ui.Node {
	className := "rounded-full px-4 py-2 text-sm font-semibold "
	switch props.Tone {
	case "warn":
		className += "bg-amber-400/15 text-amber-100 border border-amber-400/30"
	case "good":
		className += "bg-emerald-400/15 text-emerald-100 border border-emerald-400/30"
	default:
		className += "bg-cyan-400/15 text-cyan-100 border border-cyan-400/30"
	}
	return html.Span(html.Props{Class: className}, html.Text(props.Label))
}

func createElementExample() ui.Node {
	selected := ui.UseState("base")
	setTone := func(tone string) ui.Handler {
		return ui.UseEvent(func() { selected.Set(tone) })
	}

	labels := []badgeProps{
		{Label: "Dynamic component", Tone: selected.Get()},
		{Label: strings.ToUpper(selected.Get()) + " props", Tone: selected.Get()},
	}

	items := make([]ui.Node, 0, len(labels))
	for _, item := range labels {
		items = append(items, ui.CreateElement(badge, item))
	}

	return shared.ExamplePage(
		"ui.CreateElement",
		"Create elements from component functions and typed props",
		"This example builds the same badge component multiple times by calling ui.CreateElement directly with a props struct.",
		shared.ExamplePanel("Created components",
			html.Div(html.Props{Class: "mt-4 flex flex-wrap gap-3"}, items...),
			html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
				shared.ExampleButton("Base", setTone("base")),
				shared.ExampleButton("Warn", setTone("warn")),
				shared.ExampleButton("Good", setTone("good")),
			),
		),
		shared.ExamplePanel("Pattern",
			shared.ExampleCode(
				"ui.CreateElement(badge, badgeProps{",
				"    Label: \"Dynamic component\",",
				"    Tone: selectedTone,",
				"})",
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(createElementExample), "#app")
	select {}
}

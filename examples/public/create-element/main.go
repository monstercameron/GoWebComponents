//go:build js && wasm
// +build js,wasm

package main

import (
	"github.com/monstercameron/GoWebComponents/v5/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v5/examples/internal/examplelog"
	"strings"

	"github.com/monstercameron/GoWebComponents/v5/examples/shared"
	"github.com/monstercameron/GoWebComponents/v5/html"
	"github.com/monstercameron/GoWebComponents/v5/ui"
	"github.com/monstercameron/GoWebComponents/v5/utils"
)

type badgeProps struct {
	Label string
	Tone  string
}

func badge(parseProps badgeProps) ui.Node {
	parseClassName := "rounded-full px-4 py-2 text-sm font-semibold "
	switch parseProps.Tone {
	case "warn":
		parseClassName += "bg-amber-400/15 text-amber-100 border border-amber-400/30"
	case "good":
		parseClassName += "bg-emerald-400/15 text-emerald-100 border border-emerald-400/30"
	default:
		parseClassName += "bg-cyan-400/15 text-cyan-100 border border-cyan-400/30"
	}
	return html.Span(html.Props{Class: parseClassName}, html.Text(parseProps.Label))
}

func createElementExample() ui.Node {
	parseSelected := ui.UseState("base")
	setTone := func(parseTone string) ui.Handler {
		return ui.UseEvent(func() { parseSelected.Set(parseTone) })
	}

	parseLabels := []badgeProps{
		{Label: "Dynamic component", Tone: parseSelected.Get()},
		{Label: strings.ToUpper(parseSelected.Get()) + " props", Tone: parseSelected.Get()},
	}

	parseItems := make([]ui.Node, 0, len(parseLabels))
	for _, parseItem := range parseLabels {
		parseItems = append(parseItems, ui.CreateElement(badge, parseItem))
	}

	return shared.ExamplePage(
		"ui.CreateElement",
		"Create elements from component functions and typed props",
		"This example builds the same badge component multiple times by calling ui.CreateElement directly with a props struct.",
		shared.ExamplePanel("Created components",
			html.Div(html.Props{Class: "mt-4 flex flex-wrap gap-3"}, parseItems...),
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
	exampleboot.RenderExampleRoot(ui.CreateElement(createElementExample))
	exampleboot.WaitExampleRuntime()
}

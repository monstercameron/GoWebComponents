//go:build js && wasm

package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/monstercameron/GoWebComponents/v5/devtools"
	"github.com/monstercameron/GoWebComponents/v5/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v5/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/v5/examples/shared"
	"github.com/monstercameron/GoWebComponents/v5/html"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

type streamItem struct {
	ID    string
	Label string
}

func App() ui.Node {
	parseCount := ui.UseState(0)
	filter := ui.UseState("")
	parsePulse := ui.UseState(false)
	parseShowInspector := ui.UseState(true)
	parseItems := ui.UseState([]streamItem{
		{ID: "alpha", Label: "Profile runtime tree"},
		{ID: "beta", Label: "Inspect hook values"},
		{ID: "", Label: "Trigger missing-key diagnostic"},
	})

	parseFilteredCount := ui.UseMemo(func() int {
		parseQuery := strings.TrimSpace(strings.ToLower(filter.Get()))
		if parseQuery == "" {
			return len(parseItems.Get())
		}

		parseCount2 := 0
		for _, parseItem := range parseItems.Get() {
			if strings.Contains(strings.ToLower(parseItem.Label), parseQuery) {
				parseCount2++
			}
		}
		return parseCount2
	}, parseItems.Get(), filter.Get())

	parseIncrement := ui.UseEvent(func() {
		parseCount.Update(func(parseCurrent int) int { return parseCurrent + 1 })
	})

	parseTogglePulse := ui.UseEvent(func() {
		parsePulse.Update(func(isCurrent bool) bool { return !isCurrent })
	})

	parseToggleInspector := ui.UseEvent(func() {
		parseShowInspector.Update(func(isCurrent2 bool) bool { return !isCurrent2 })
	})

	parseUpdateFilter := ui.UseEvent(func(parseEvent ui.InputEvent) {
		filter.Set(parseEvent.GetValue())
	})

	parseAppendItem := ui.UseEvent(func() {
		parseNextID := fmt.Sprintf("node-%d", len(parseItems.Get())+1)
		parseItems.Update(func(parseCurrent2 []streamItem) []streamItem {
			parseNext := append([]streamItem(nil), parseCurrent2...)
			parseNext = append(parseNext, streamItem{ID: parseNextID, Label: fmt.Sprintf("Observe commit %d", len(parseCurrent2)+1)})
			return parseNext
		})
	})

	parseCards := make([]ui.Node, 0, len(parseItems.Get()))
	for parseIndex, parseItem2 := range parseItems.Get() {
		parseItem3 := parseItem2
		if strings.TrimSpace(filter.Get()) != "" && !strings.Contains(strings.ToLower(parseItem3.Label), strings.ToLower(filter.Get())) {
			continue
		}

		parseProps := html.Props{Class: "rounded-[20px] border border-white/10 bg-slate-950/60 p-4 text-left", Key: parseItem3.ID}
		parseCards = append(parseCards, html.Article(parseProps,
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-cyan-300"}, html.Text(fmt.Sprintf("Card %d", parseIndex+1))),
			html.H3(html.Props{Class: "mt-2 text-lg font-bold text-white"}, html.Text(parseItem3.Label)),
			html.P(html.Props{Class: "mt-2 text-sm text-slate-400"}, html.Text("One item intentionally omits a key so the diagnostics view has something concrete to surface.")),
		))
	}

	var parseInspectorNode ui.Node = html.P(html.Props{Class: "text-sm text-slate-400"}, html.Text("Devtools panel hidden. Reopen it to inspect the runtime tree."))
	if parseShowInspector.Get() {
		parseInspectorNode = ui.CreateElement(devtools.Panel, devtools.PanelProps{
			Title:           "Standalone Devtools",
			InitiallyOpen:   true,
			RefreshInterval: 500 * time.Millisecond,
			MaxDepth:        4,
		})
	}

	return shared.ExamplePage(
		"Devtools",
		"devtools.Panel",
		"Trigger local state changes, then inspect the component tree, hook values, and diagnostics in the standalone panel.",
		shared.ExamplePanel("Controls",
			html.Div(html.Props{Class: "flex flex-wrap gap-2"},
				shared.ExampleButton("Increment Counter", parseIncrement),
				shared.ExampleButton("Toggle Pulse", parseTogglePulse),
				shared.ExampleButton("Append Card", parseAppendItem),
				shared.ExampleButton(func() string {
					if parseShowInspector.Get() {
						return "Hide Devtools"
					}
					return "Show Devtools"
				}(), parseToggleInspector),
			),
			html.Div(html.Props{Class: "space-y-2"},
				html.Label(html.Props{Class: "block text-xs font-semibold uppercase tracking-[0.18em] text-slate-400"}, html.Text("Filter cards")),
				html.Input(html.Props{Value: filter.Get(), OnInput: parseUpdateFilter, Placeholder: "Type to filter the card list", Class: "w-full rounded-2xl border border-white/10 bg-slate-950/70 px-4 py-3 text-slate-100 placeholder:text-slate-500 focus:outline-none"}),
			),
		),
		shared.ExamplePanel("Observed State",
			html.Div(html.Props{Class: "grid gap-3 sm:grid-cols-2 lg:grid-cols-3"},
				shared.ExampleStat("Counter", fmt.Sprintf("%d", parseCount.Get())),
				shared.ExampleStat("Pulse", fmt.Sprintf("%t", parsePulse.Get())),
				shared.ExampleStat("Visible Cards", fmt.Sprintf("%d", parseFilteredCount)),
			),
		),
		shared.ExamplePanel("Cards",
			html.Div(html.Props{Class: "grid gap-4 md:grid-cols-2"}, parseCards...),
		),
		shared.ExamplePanel("Inspector",
			parseInspectorNode,
		),
	)
}

func main() {
	exampleboot.RenderExampleRoot(ui.CreateElement(App))
	exampleboot.WaitExampleRuntime()
}

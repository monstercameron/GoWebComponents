//go:build js && wasm

package main

import (
	"fmt"
	"github.com/monstercameron/GoWebComponents/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"strings"
	"time"

	"github.com/monstercameron/GoWebComponents/devtools"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
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

		parseProps := html.Props{Class: "rounded-2xl border border-white/10 bg-white/5 p-4 text-left", Key: parseItem3.ID}
		parseCards = append(parseCards, html.Article(parseProps,
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-cyan-300"}, html.Text(fmt.Sprintf("Card %d", parseIndex+1))),
			html.H3(html.Props{Class: "mt-2 text-lg font-bold text-white"}, html.Text(parseItem3.Label)),
			html.P(html.Props{Class: "mt-2 text-sm text-slate-400"}, html.Text("One card intentionally omits a key so the inspector diagnostics panel has something concrete to show.")),
		))
	}

	return html.Div(html.Props{Class: "min-h-screen bg-[#08111d] text-slate-100"},
		html.Div(html.Props{Class: "mx-auto max-w-5xl px-6 py-12"},
			html.Div(html.Props{Class: "rounded-[2rem] border border-white/10 bg-[radial-gradient(circle_at_top,_rgba(34,211,238,0.18),_transparent_45%),rgba(15,23,42,0.92)] p-8 shadow-2xl"},
				html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("Devtools Example")),
				html.H1(html.Props{Class: "mt-4 text-5xl font-black tracking-tight text-white"}, html.Text("Inspect runtime state without the OMI shell")),
				html.P(html.Props{Class: "mt-4 max-w-3xl text-lg leading-8 text-slate-300"}, html.Text("This smaller example keeps the UI simple so the devtools panel can highlight component tree shape, hook values, profiling counters, and diagnostics with minimal noise.")),
				html.Div(html.Props{Class: "mt-8 flex flex-wrap gap-3"},
					html.Button(html.Props{OnClick: parseIncrement, Class: "rounded-full border border-cyan-900/80 bg-cyan-950/70 px-5 py-3 font-semibold text-cyan-100 hover:bg-cyan-900/80"}, html.Text("Increment counter")),
					html.Button(html.Props{OnClick: parseTogglePulse, Class: "rounded-full border border-white/10 bg-white/5 px-5 py-3 font-semibold text-white hover:bg-white/10"}, html.Text("Toggle pulse")),
					html.Button(html.Props{OnClick: parseAppendItem, Class: "rounded-full border border-white/10 bg-white/5 px-5 py-3 font-semibold text-white hover:bg-white/10"}, html.Text("Append card")),
					html.Button(html.Props{OnClick: parseToggleInspector, Class: "rounded-full border border-amber-400/30 bg-amber-400/10 px-5 py-3 font-semibold text-amber-100 hover:bg-amber-400/20"}, html.Text(func() string {
						if parseShowInspector.Get() {
							return "Hide devtools"
						}
						return "Show devtools"
					}())),
				),
			),
			html.Div(html.Props{Class: "mt-8 grid gap-6 md:grid-cols-3"},
				html.Div(html.Props{Class: "rounded-[1.5rem] border border-white/10 bg-white/5 p-6"},
					html.Small(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text("Counter")),
					html.P(html.Props{Class: "mt-3 text-4xl font-black text-white"}, html.Text(fmt.Sprintf("%d", parseCount.Get()))),
				),
				html.Div(html.Props{Class: "rounded-[1.5rem] border border-white/10 bg-white/5 p-6"},
					html.Small(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text("Pulse state")),
					html.P(html.Props{Class: "mt-3 text-2xl font-bold text-white"}, html.Text(fmt.Sprintf("%t", parsePulse.Get()))),
					html.P(html.Props{Class: "mt-2 text-sm text-slate-400"}, html.Text(func() string {
						if parsePulse.Get() {
							return "This toggles a local boolean so the inspector can show state changes."
						}
						return "Flip the toggle to generate a small render update."
					}())),
				),
				html.Div(html.Props{Class: "rounded-[1.5rem] border border-white/10 bg-white/5 p-6"},
					html.Small(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text("Visible cards")),
					html.P(html.Props{Class: "mt-3 text-4xl font-black text-white"}, html.Text(fmt.Sprintf("%d", parseFilteredCount))),
				),
			),
			html.Div(html.Props{Class: "mt-8 rounded-[1.5rem] border border-white/10 bg-white/5 p-6"},
				html.Label(html.Props{Class: "mb-2 block text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text("Filter cards")),
				html.Input(html.Props{Value: filter.Get(), OnInput: parseUpdateFilter, Placeholder: "Type to filter the card list", Class: "w-full rounded-2xl border border-white/10 bg-slate-950/70 px-4 py-3 text-slate-100 placeholder:text-slate-500 focus:outline-none"}),
			),
			html.Section(html.Props{Class: "mt-8 grid gap-4 md:grid-cols-2"}, parseCards...),
			func() ui.Node {
				if !parseShowInspector.Get() {
					return nil
				}
				return ui.CreateElement(devtools.Panel, devtools.PanelProps{
					Title:           "Standalone Devtools",
					InitiallyOpen:   true,
					RefreshInterval: 500 * time.Millisecond,
					MaxDepth:        4,
				})
			}(),
		),
	)
}

func main() {
	exampleboot.RenderExampleRoot(ui.CreateElement(App))
	exampleboot.WaitExampleRuntime()
}

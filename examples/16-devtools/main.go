//go:build js && wasm

package main

import (
	"fmt"
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
	count := ui.UseState(0)
	filter := ui.UseState("")
	pulse := ui.UseState(false)
	showInspector := ui.UseState(true)
	items := ui.UseState([]streamItem{
		{ID: "alpha", Label: "Profile runtime tree"},
		{ID: "beta", Label: "Inspect hook values"},
		{ID: "", Label: "Trigger missing-key diagnostic"},
	})

	filteredCount := ui.UseMemo(func() int {
		query := strings.TrimSpace(strings.ToLower(filter.Get()))
		if query == "" {
			return len(items.Get())
		}

		count := 0
		for _, item := range items.Get() {
			if strings.Contains(strings.ToLower(item.Label), query) {
				count++
			}
		}
		return count
	}, items.Get(), filter.Get())

	increment := ui.UseEvent(func() {
		count.Update(func(current int) int { return current + 1 })
	})

	togglePulse := ui.UseEvent(func() {
		pulse.Update(func(current bool) bool { return !current })
	})

	toggleInspector := ui.UseEvent(func() {
		showInspector.Update(func(current bool) bool { return !current })
	})

	updateFilter := ui.UseEvent(func(event ui.InputEvent) {
		filter.Set(event.GetValue())
	})

	appendItem := ui.UseEvent(func() {
		nextID := fmt.Sprintf("node-%d", len(items.Get())+1)
		items.Update(func(current []streamItem) []streamItem {
			next := append([]streamItem(nil), current...)
			next = append(next, streamItem{ID: nextID, Label: fmt.Sprintf("Observe commit %d", len(current)+1)})
			return next
		})
	})

	cards := make([]ui.Node, 0, len(items.Get()))
	for index, item := range items.Get() {
		item := item
		if strings.TrimSpace(filter.Get()) != "" && !strings.Contains(strings.ToLower(item.Label), strings.ToLower(filter.Get())) {
			continue
		}

		props := html.Props{Class: "rounded-2xl border border-white/10 bg-white/5 p-4 text-left", Key: item.ID}
		cards = append(cards, html.Article(props,
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-cyan-300"}, html.Text(fmt.Sprintf("Card %d", index+1))),
			html.H3(html.Props{Class: "mt-2 text-lg font-bold text-white"}, html.Text(item.Label)),
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
					html.Button(html.Props{OnClick: increment, Class: "rounded-full border border-cyan-900/80 bg-cyan-950/70 px-5 py-3 font-semibold text-cyan-100 hover:bg-cyan-900/80"}, html.Text("Increment counter")),
					html.Button(html.Props{OnClick: togglePulse, Class: "rounded-full border border-white/10 bg-white/5 px-5 py-3 font-semibold text-white hover:bg-white/10"}, html.Text("Toggle pulse")),
					html.Button(html.Props{OnClick: appendItem, Class: "rounded-full border border-white/10 bg-white/5 px-5 py-3 font-semibold text-white hover:bg-white/10"}, html.Text("Append card")),
					html.Button(html.Props{OnClick: toggleInspector, Class: "rounded-full border border-amber-400/30 bg-amber-400/10 px-5 py-3 font-semibold text-amber-100 hover:bg-amber-400/20"}, html.Text(func() string {
						if showInspector.Get() {
							return "Hide devtools"
						}
						return "Show devtools"
					}())),
				),
			),
			html.Div(html.Props{Class: "mt-8 grid gap-6 md:grid-cols-3"},
				html.Div(html.Props{Class: "rounded-[1.5rem] border border-white/10 bg-white/5 p-6"},
					html.Small(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text("Counter")),
					html.P(html.Props{Class: "mt-3 text-4xl font-black text-white"}, html.Text(fmt.Sprintf("%d", count.Get()))),
				),
				html.Div(html.Props{Class: "rounded-[1.5rem] border border-white/10 bg-white/5 p-6"},
					html.Small(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text("Pulse state")),
					html.P(html.Props{Class: "mt-3 text-2xl font-bold text-white"}, html.Text(fmt.Sprintf("%t", pulse.Get()))),
					html.P(html.Props{Class: "mt-2 text-sm text-slate-400"}, html.Text(func() string {
						if pulse.Get() {
							return "This toggles a local boolean so the inspector can show state changes."
						}
						return "Flip the toggle to generate a small render update."
					}())),
				),
				html.Div(html.Props{Class: "rounded-[1.5rem] border border-white/10 bg-white/5 p-6"},
					html.Small(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text("Visible cards")),
					html.P(html.Props{Class: "mt-3 text-4xl font-black text-white"}, html.Text(fmt.Sprintf("%d", filteredCount))),
				),
			),
			html.Div(html.Props{Class: "mt-8 rounded-[1.5rem] border border-white/10 bg-white/5 p-6"},
				html.Label(html.Props{Class: "mb-2 block text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text("Filter cards")),
				html.Input(html.Props{Value: filter.Get(), OnInput: updateFilter, Placeholder: "Type to filter the card list", Class: "w-full rounded-2xl border border-white/10 bg-slate-950/70 px-4 py-3 text-slate-100 placeholder:text-slate-500 focus:outline-none"}),
			),
			html.Section(html.Props{Class: "mt-8 grid gap-4 md:grid-cols-2"}, cards...),
			func() ui.Node {
				if !showInspector.Get() {
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
	ui.Render(ui.CreateElement(App), "#app")
	select {}
}

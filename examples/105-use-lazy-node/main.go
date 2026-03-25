//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

func useLazyNodeExample() ui.Node {
	parsePanel := ui.UseState("inventory summary")
	parseFailNext := ui.UseState(false)

	handle := ui.UseLazyNode(func(parseCtx context.Context) (ui.Node, error) {
		select {
		case <-parseCtx.Done():
			return nil, parseCtx.Err()
		case <-time.After(420 * time.Millisecond):
		}

		if parseFailNext.Get() {
			return nil, errors.New("deferred panel failed to load")
		}

		return html.Div(html.Props{Class: "rounded-[1.6rem] border border-emerald-400/25 bg-emerald-400/10 p-6 text-emerald-50"},
			html.P(html.Props{Class: "text-xs font-semibold uppercase tracking-[0.28em] text-emerald-200"}, html.Text("Loaded with ui.UseLazyNode")),
			html.H3(html.Props{Class: "mt-3 text-2xl font-semibold"}, html.Text(parsePanel.Get())),
			html.P(html.Props{Class: "mt-3 text-sm leading-7 text-emerald-100"}, html.Text("The hook resolved this subtree asynchronously, and the page decides how loading, error, reload, and cancel states should render.")),
		), nil
	}, parsePanel.Get(), parseFailNext.Get())

	parseState := handle.Get()

	parseSwitchPanel := ui.UseEvent(func() {
		if parsePanel.Get() == "inventory summary" {
			parsePanel.Set("delivery exceptions")
			return
		}
		parsePanel.Set("inventory summary")
	})
	parseReload := ui.UseEvent(func() {
		handle.Reload()
	})
	parseCancel := ui.UseEvent(func() {
		handle.Cancel()
	})
	parseToggleFailure := ui.UseEvent(func() {
		parseFailNext.Update(func(isPrevious bool) bool {
			return !isPrevious
		})
	})

	parseStatus := "Idle with the latest committed node."
	if parseState.Loading {
		parseStatus = "Loader is running; AsyncBoundary will show the delayed fallback."
	} else if parseState.Error != nil {
		parseStatus = parseState.Error.Error()
	} else if parseState.Ready {
		parseStatus = "Lazy node resolved successfully."
	}

	return shared.ExamplePage(
		"ui.UseLazyNode",
		"Resolve a subtree lazily while keeping manual control over loading state",
		"UseLazyNode is the hook-level lazy primitive. It gives you the current lazy-node state plus Reload and Cancel so you can decide how the fallback, timing, and error UI should behave before reaching for the simpler ui.Lazy wrapper.",
		shared.ExamplePanel("Hook-managed lazy node",
			html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-3"},
				shared.ExampleButton("Switch panel", parseSwitchPanel),
				shared.ExampleButton("Reload node", parseReload),
				shared.ExampleButton("Cancel load", parseCancel),
				shared.ExampleButton("Toggle failure", parseToggleFailure),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-5"},
				shared.ExampleStat("Panel", parsePanel.Get()),
				shared.ExampleStat("Loading", fmt.Sprintf("%t", parseState.Loading)),
				shared.ExampleStat("Ready", fmt.Sprintf("%t", parseState.Ready)),
				shared.ExampleStat("Error", fmt.Sprintf("%t", parseState.Error != nil)),
				shared.ExampleStat("Fail next", fmt.Sprintf("%t", parseFailNext.Get())),
			),
			html.P(html.Props{Class: "mt-5 text-sm leading-7 text-slate-300"}, html.Text(parseStatus)),
			html.Div(html.Props{Class: "mt-6"},
				ui.AsyncBoundary(ui.AsyncBoundaryProps{
					Pending: parseState.Loading,
					Error:   parseState.Error,
					Delay:   140 * time.Millisecond,
					Fallback: html.Div(html.Props{Class: "rounded-[1.6rem] border border-cyan-400/25 bg-cyan-400/10 p-6 text-cyan-50"},
						html.P(html.Props{Class: "text-xs font-semibold uppercase tracking-[0.28em] text-cyan-200"}, html.Text("Fallback after 140ms")),
						html.P(html.Props{Class: "mt-3 text-sm leading-7 text-cyan-100"}, html.Text("UseLazyNode only gives you state. AsyncBoundary is what turns that loading state into a delayed fallback UI.")),
					),
					ErrorFallback: func(parseErr error) ui.Node {
						return html.Div(html.Props{Class: "rounded-[1.6rem] border border-red-400/25 bg-red-400/10 p-6 text-red-50"},
							html.P(html.Props{Class: "text-xs font-semibold uppercase tracking-[0.28em] text-red-200"}, html.Text("Hook-level error surface")),
							html.P(html.Props{Class: "mt-3 text-sm leading-7 text-red-100"}, html.Text(parseErr.Error())),
						)
					},
					Content: parseState.Node,
				}),
			),
		),
		shared.ExamplePanel("Choose the right lazy tool",
			html.Ul(html.Props{Class: "mt-3 grid gap-3 text-sm leading-7 text-slate-300"},
				html.Li(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/45 px-4 py-3"}, html.Text("Use ui.UseLazyNode(...) when the component needs to inspect lazy state directly or expose imperative Reload and Cancel actions.")),
				html.Li(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/45 px-4 py-3"}, html.Text("Use ui.Lazy(...) when you only want a lazy subtree plus fallback props and do not need hook-level control.")),
				html.Li(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/45 px-4 py-3"}, html.Text("Use route loaders or ui.UseWorkerTask(...) when the work is route-owned data loading or CPU-heavy background work instead of local subtree mounting.")),
			),
			shared.ExampleCode(
				`handle := ui.UseLazyNode(loader, panel.Get(), failNext.Get())`,
				`state := handle.Get()`,
				`ui.AsyncBoundary(ui.AsyncBoundaryProps{Pending: state.Loading, Error: state.Error, Content: state.Node})`,
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(useLazyNodeExample), "#app")
	select {}
}

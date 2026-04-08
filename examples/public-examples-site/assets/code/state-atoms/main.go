//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/state"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

const (
	CounterAtom = "counter"
	ThemeAtom   = "theme"
)

func CounterDisplay() ui.Node {
	parseCount := state.UseAtom(CounterAtom, 0)
	parseTheme := state.UseAtom(ThemeAtom, "light")
	parseThemeLabel := state.UseComputed(func() string {
		if parseTheme.Get() == "dark" {
			return "Dark"
		}
		return "Light"
	}, parseTheme.Get())
	parseCountSummary := state.UseComputed(func() string {
		if parseCount.Get()%2 != 0 {
			return "Odd"
		}
		return "Even"
	}, parseCount.Get())

	parseSurfaceClass := "rounded-[20px] border border-amber-200/10 bg-amber-50/[0.03] p-5"
	if parseTheme.Get() == "dark" {
		parseSurfaceClass = "rounded-[20px] border border-slate-400/15 bg-slate-950/80 p-5"
	}

	return html.Div(
		html.Props{Class: parseSurfaceClass},
		html.Div(
			html.Props{Class: "flex flex-wrap items-start justify-between gap-4"},
			html.Div(
				html.Props{Class: "space-y-2"},
				html.Div(html.Props{Class: "text-xs font-semibold uppercase tracking-[0.18em] text-slate-400"}, html.Text("Shared counter")),
				html.Div(html.Props{Class: "font-mono text-6xl font-semibold tracking-tight text-white"}, html.Text(fmt.Sprintf("%d", parseCount.Get()))),
			),
			html.Div(
				html.Props{Class: "rounded-full border border-cyan-300/20 bg-cyan-400/10 px-3 py-1 text-[11px] font-medium uppercase tracking-[0.18em] text-cyan-100"},
				html.Text("Two components read the same atoms"),
			),
		),
		html.Div(
			html.Props{Class: "mt-5 grid gap-3 sm:grid-cols-2 lg:grid-cols-3"},
			shared.ExampleStat("Theme", parseThemeLabel.Get()),
			shared.ExampleStat("Parity", parseCountSummary.Get()),
			shared.ExampleStat("Atom ID", CounterAtom),
		),
	)
}

func Controls() ui.Node {
	parseCount := state.UseAtom(CounterAtom, 0)
	parseTheme := state.UseAtom(ThemeAtom, "light")

	parseIncrement := ui.UseEvent(func() {
		parseCount.Set(parseCount.Get() + 1)
	})

	parseDecrement := ui.UseEvent(func() {
		parseCount.Set(parseCount.Get() - 1)
	})

	parseToggleTheme := ui.UseEvent(func() {
		if parseTheme.Get() == "light" {
			parseTheme.Set("dark")
			return
		}
		parseTheme.Set("light")
	})

	return html.Div(
		html.Props{Class: "space-y-4"},
		html.Div(
			html.Props{Class: "flex flex-wrap gap-2"},
			shared.ExampleButton("Decrement", parseDecrement),
			shared.ExampleButton("Increment", parseIncrement),
			shared.ExampleButton(func() string {
				if parseTheme.Get() == "light" {
					return "Switch Theme: Dark"
				}
				return "Switch Theme: Light"
			}(), parseToggleTheme),
		),
		html.P(
			html.Props{Class: "rounded-[20px] border border-white/10 bg-slate-950/60 px-4 py-4 text-sm leading-6 text-slate-300"},
			html.Text("Use one set of controls here, then watch the separate display panel update from the same shared counter and theme atoms."),
		),
	)
}

func App() ui.Node {
	return shared.ExamplePage(
		"State Atoms",
		"state.UseAtom",
		"Share one counter and one theme value across separate components without prop wiring.",
		shared.ExamplePanel("Shared Values", ui.CreateElement(CounterDisplay)),
		shared.ExamplePanel("Controls", ui.CreateElement(Controls)),
	)
}

func main() {
	utils.DisableAllDebug()
	exampleboot.RenderExampleRoot(ui.CreateElement(App))
	exampleboot.WaitExampleRuntime()
}

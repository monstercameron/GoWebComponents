//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"

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
			return "Dark theme active"
		}
		return "Light theme active"
	}, parseTheme.Get())
	parseCountSummary := state.UseComputed(func() string {
		parseValue := parseCount.Get()
		parseSign := "even"
		if parseValue%2 != 0 {
			parseSign = "odd"
		}
		return fmt.Sprintf("%d is %s", parseValue, parseSign)
	}, parseCount.Get())

	parseTextColor := "text-gray-900"
	if parseTheme.Get() == "dark" {
		parseTextColor = "text-white"
	}

	return html.Div(
		html.Props{Class: "text-center p-8 bg-white/5 border border-white/10 rounded-xl backdrop-blur-sm shadow-2xl"},
		html.H2(
			html.Props{Class: "text-2xl font-bold " + parseTextColor + " mb-4"},
			html.Text("Current Count"),
		),
		html.Div(
			html.Props{Class: "text-7xl font-black bg-clip-text text-transparent bg-gradient-to-r from-blue-400 to-purple-500 font-mono tracking-tighter"},
			html.Text(fmt.Sprintf("%d", parseCount.Get())),
		),
		html.P(
			html.Props{Class: "mt-4 text-sm text-gray-400"},
			html.Text(parseThemeLabel.Get()),
		),
		html.P(
			html.Props{Class: "mt-1 text-xs uppercase tracking-[0.2em] text-gray-500"},
			html.Text(parseCountSummary.Get()),
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
		} else {
			parseTheme.Set("light")
		}
	})

	return html.Div(
		html.Props{Class: "flex flex-col space-y-6"},
		html.Div(
			html.Props{Class: "flex justify-center space-x-6"},
			html.Button(
				html.Props{
					Class:   "w-16 h-16 flex items-center justify-center bg-white/5 hover:bg-white/10 border border-white/10 rounded-full text-2xl text-white transition-all duration-200 hover:scale-110 active:scale-95",
					OnClick: parseDecrement,
				},
				html.Text("-"),
			),
			html.Button(
				html.Props{
					Class:   "w-16 h-16 flex items-center justify-center bg-gradient-to-r from-blue-500 to-purple-600 rounded-full text-2xl text-white shadow-lg shadow-purple-500/20 transition-all duration-200 hover:scale-110 active:scale-95",
					OnClick: parseIncrement,
				},
				html.Text("+"),
			),
		),
		html.Button(
			html.Props{
				Class:   "px-6 py-3 bg-white/5 hover:bg-white/10 border border-white/10 text-white rounded-lg font-medium transition-colors duration-200",
				OnClick: parseToggleTheme,
			},
			html.Text(func() string {
				if parseTheme.Get() == "light" {
					return "ðŸŒ™ Switch to Dark Mode"
				}
				return "â˜€ï¸ Switch to Light Mode"
			}()),
		),
	)
}

func App() ui.Node {
	parseTheme := state.UseAtom(ThemeAtom, "light")

	parseContainerClass := "min-h-screen transition-colors duration-500 flex items-center justify-center p-4"
	if parseTheme.Get() == "dark" {
		parseContainerClass += " bg-[#0a0a0a]"
	} else {
		parseContainerClass += " bg-gray-100"
	}

	parseCardClass := "max-w-md w-full rounded-2xl shadow-2xl p-8 transition-colors duration-500"
	if parseTheme.Get() == "dark" {
		parseCardClass += " bg-black/40 border border-white/10"
	} else {
		parseCardClass += " bg-white"
	}

	return html.Div(
		html.Props{Class: parseContainerClass},
		html.Div(
			html.Props{Class: parseCardClass},
			html.Div(
				html.Props{Class: "text-center mb-10"},
				html.H1(
					html.Props{Class: "text-3xl font-extrabold bg-clip-text text-transparent bg-gradient-to-r from-blue-400 to-purple-500"},
					html.Text("Global State (Atoms)"),
				),
				html.P(
					html.Props{Class: "mt-2 text-gray-500"},
					html.Text("State shared across independent components"),
				),
			),
			html.Div(
				html.Props{Class: "space-y-10"},
				ui.CreateElement(CounterDisplay),
				html.Div(html.Props{Class: "border-t border-white/10"}),
				ui.CreateElement(Controls),
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(App), "#app")
	select {}
}

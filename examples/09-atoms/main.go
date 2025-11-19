//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/dom"
	"github.com/monstercameron/GoWebComponents/hooks"
	"github.com/monstercameron/GoWebComponents/render"
	"github.com/monstercameron/GoWebComponents/state"
)

// Define global atom keys for shared state
const (
	CounterAtom = "counter"
	ThemeAtom   = "theme"
)

func CounterDisplay(_ dom.Attrs) *dom.Element {
	// Subscribe to the counter atom
	count, _ := state.UseAtom(CounterAtom, 0)
	theme, _ := state.UseAtom(ThemeAtom, "light")

	textColor := "text-gray-900"
	if theme() == "dark" {
		textColor = "text-white"
	}

	return dom.Div(
		dom.Attrs{"class": "text-center p-8 bg-white/5 border border-white/10 rounded-xl backdrop-blur-sm shadow-2xl"},
		dom.H2(
			dom.Attrs{"class": "text-2xl font-bold " + textColor + " mb-4"},
			"Current Count",
		),
		dom.Div(
			dom.Attrs{"class": "text-7xl font-black bg-clip-text text-transparent bg-gradient-to-r from-blue-400 to-purple-500 font-mono tracking-tighter"},
			fmt.Sprintf("%d", count()),
		),
	)
}

func Controls(_ dom.Attrs) *dom.Element {
	// Subscribe to atoms
	count, setCount := state.UseAtom(CounterAtom, 0)
	theme, setTheme := state.UseAtom(ThemeAtom, "light")

	increment := hooks.GoUseFunc(func(e dom.GoEvent) {
		setCount(count() + 1)
	})

	decrement := hooks.GoUseFunc(func(e dom.GoEvent) {
		setCount(count() - 1)
	})

	toggleTheme := hooks.GoUseFunc(func(e dom.GoEvent) {
		if theme() == "light" {
			setTheme("dark")
		} else {
			setTheme("light")
		}
	})

	return dom.Div(
		dom.Attrs{"class": "flex flex-col space-y-6"},
		dom.Div(
			dom.Attrs{"class": "flex justify-center space-x-6"},
			dom.Button(
				dom.Attrs{
					"class":   "w-16 h-16 flex items-center justify-center bg-white/5 hover:bg-white/10 border border-white/10 rounded-full text-2xl text-white transition-all duration-200 hover:scale-110 active:scale-95",
					"onclick": decrement,
				},
				"-",
			),
			dom.Button(
				dom.Attrs{
					"class":   "w-16 h-16 flex items-center justify-center bg-gradient-to-r from-blue-500 to-purple-600 rounded-full text-2xl text-white shadow-lg shadow-purple-500/20 transition-all duration-200 hover:scale-110 active:scale-95",
					"onclick": increment,
				},
				"+",
			),
		),
		dom.Button(
			dom.Attrs{
				"class":   "px-6 py-3 bg-white/5 hover:bg-white/10 border border-white/10 text-white rounded-lg font-medium transition-colors duration-200",
				"onclick": toggleTheme,
			},
			func() string {
				if theme() == "light" {
					return "🌙 Switch to Dark Mode"
				}
				return "☀️ Switch to Light Mode"
			}(),
		),
	)
}

func App(_ dom.Attrs) *dom.Element {
	theme, _ := state.UseAtom(ThemeAtom, "light")

	containerClass := "min-h-screen transition-colors duration-500 flex items-center justify-center p-4"
	if theme() == "dark" {
		containerClass += " bg-[#0a0a0a]"
	} else {
		containerClass += " bg-gray-100"
	}

	cardClass := "max-w-md w-full rounded-2xl shadow-2xl p-8 transition-colors duration-500"
	if theme() == "dark" {
		cardClass += " bg-black/40 border border-white/10"
	} else {
		cardClass += " bg-white"
	}

	return dom.Div(
		dom.Attrs{"class": containerClass},
		dom.Div(
			dom.Attrs{"class": cardClass},
			dom.Div(
				dom.Attrs{"class": "text-center mb-10"},
				dom.H1(
					dom.Attrs{"class": "text-3xl font-extrabold bg-clip-text text-transparent bg-gradient-to-r from-blue-400 to-purple-500"},
					"Global State (Atoms)",
				),
				dom.P(
					dom.Attrs{"class": "mt-2 text-gray-500"},
					"State shared across independent components",
				),
			),
			dom.Div(
				dom.Attrs{"class": "space-y-10"},
				dom.CreateElement(CounterDisplay, nil),
				dom.Div(dom.Attrs{"class": "border-t border-white/10"}),
				dom.CreateElement(Controls, nil),
			),
		),
	)
}

func main() {
	render.To(dom.CreateElement(App, nil), "body")
}

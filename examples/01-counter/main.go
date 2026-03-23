//go:build js && wasm
// +build js,wasm

package main

import (
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

// Counter component - demonstrates basic number state management
func CounterExample() ui.Node {
	count := ui.UseState(0)
	currentCount := count.Get()

	increment := ui.UseEvent(func() {
		count.Update(func(prev int) int { return prev + 1 })
	})

	decrement := ui.UseEvent(func() {
		count.Update(func(prev int) int { return prev - 1 })
	})

	reset := ui.UseEvent(func() {
		count.Set(0)
	})

	buttonBaseClass := "transition-all duration-200 active:scale-95"
	iconButtonClass := html.ClassNames(
		"w-16 h-16 flex items-center justify-center rounded-full text-2xl",
		buttonBaseClass,
		"hover:scale-110",
	)
	neutralButtonClass := html.ClassNames(
		iconButtonClass,
		"bg-white/5 hover:bg-white/10 border border-white/10",
	)
	primaryButtonClass := html.ClassNames(
		iconButtonClass,
		"bg-gradient-to-r from-blue-500 to-purple-600 shadow-lg shadow-purple-500/20",
	)
	resetButtonClass := html.ClassNames(
		"px-6 h-16 flex items-center justify-center rounded-full text-sm font-bold uppercase tracking-wider",
		buttonBaseClass,
		"bg-white/5 hover:bg-white/10 border border-white/10 hover:scale-105",
	)

	return html.Div(html.Props{
		Class: "min-h-screen flex items-center justify-center bg-[#0a0a0a] text-white p-4",
	},
		html.Div(html.Props{
			Class: "max-w-md w-full bg-white/5 border border-white/10 rounded-xl backdrop-blur-sm p-8 shadow-2xl",
		},
			html.H2(html.Props{
				Class: "text-3xl font-bold text-center mb-8 bg-clip-text text-transparent bg-gradient-to-r from-blue-400 to-purple-500",
			}, html.Text("Counter Example")),

			html.Div(html.Props{
				Class: "text-center mb-10",
			},
				html.Div(html.Props{
					Class: "text-7xl font-black text-white mb-2 font-mono tracking-tighter",
				}, html.Textf("%d", currentCount)),
				html.P(html.Props{
					Class: "text-gray-400 uppercase tracking-widest text-xs font-semibold",
				}, html.Text("Current Count")),
				html.IfElse(currentCount == 0,
					html.P(html.Props{Class: "mt-3 text-xs text-emerald-300"}, html.Text("Counter is centered")),
					html.P(html.Props{Class: "mt-3 text-xs text-slate-400"}, html.Textf("Offset from zero: %d", currentCount)),
				),
			),

			html.Div(html.Props{
				Class: "flex gap-4 justify-center",
			},
				html.Button(html.Props{
					OnClick: decrement,
					Class:   neutralButtonClass,
				}, html.Text("-")),

				html.Button(html.Props{
					OnClick: reset,
					Class:   resetButtonClass,
				}, html.Text("Reset")),

				html.Button(html.Props{
					OnClick: increment,
					Class:   primaryButtonClass,
				}, html.Text("+")),
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(CounterExample), "#app")
	select {}
}

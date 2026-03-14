//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"

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
				}, html.Text(fmt.Sprintf("%d", currentCount))),
				html.P(html.Props{
					Class: "text-gray-400 uppercase tracking-widest text-xs font-semibold",
				}, html.Text("Current Count")),
			),

			html.Div(html.Props{
				Class: "flex gap-4 justify-center",
			},
				html.Button(html.Props{
					OnClick: decrement,
					Class:   "w-16 h-16 flex items-center justify-center bg-white/5 hover:bg-white/10 border border-white/10 rounded-full text-2xl transition-all duration-200 hover:scale-110 active:scale-95",
				}, html.Text("âˆ’")),

				html.Button(html.Props{
					OnClick: reset,
					Class:   "px-6 h-16 flex items-center justify-center bg-white/5 hover:bg-white/10 border border-white/10 rounded-full text-sm font-bold uppercase tracking-wider transition-all duration-200 hover:scale-105 active:scale-95",
				}, html.Text("Reset")),

				html.Button(html.Props{
					OnClick: increment,
					Class:   "w-16 h-16 flex items-center justify-center bg-gradient-to-r from-blue-500 to-purple-600 rounded-full text-2xl shadow-lg shadow-purple-500/20 transition-all duration-200 hover:scale-110 active:scale-95",
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

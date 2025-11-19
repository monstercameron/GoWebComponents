//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/dom"
	"github.com/monstercameron/GoWebComponents/hooks"
	"github.com/monstercameron/GoWebComponents/render"
	"github.com/monstercameron/GoWebComponents/utils"
)

type Attrs = dom.Attrs
type Element = render.Element

// Counter component - demonstrates basic number state management
func CounterExample(_ Attrs) *Element {
	count, setCount := hooks.UseState(0)
	currentCount := count()

	increment := hooks.GoUseFunc(func() {
		setCount(func(prev int) int { return prev + 1 })
	})

	decrement := hooks.GoUseFunc(func() {
		setCount(func(prev int) int { return prev - 1 })
	})

	reset := hooks.GoUseFunc(func() {
		setCount(0)
	})

	return dom.Div(Attrs{
		"class": "min-h-screen flex items-center justify-center bg-[#0a0a0a] text-white p-4",
	},
		dom.Div(Attrs{
			"class": "max-w-md w-full bg-white/5 border border-white/10 rounded-xl backdrop-blur-sm p-8 shadow-2xl",
		},
			dom.H2(Attrs{
				"class": "text-3xl font-bold text-center mb-8 bg-clip-text text-transparent bg-gradient-to-r from-blue-400 to-purple-500",
			}, dom.Text("Counter Example")),

			dom.Div(Attrs{
				"class": "text-center mb-10",
			},
				dom.Div(Attrs{
					"class": "text-7xl font-black text-white mb-2 font-mono tracking-tighter",
				}, dom.Text(fmt.Sprintf("%d", currentCount))),
				dom.P(Attrs{
					"class": "text-gray-400 uppercase tracking-widest text-xs font-semibold",
				}, dom.Text("Current Count")),
			),

			dom.Div(Attrs{
				"class": "flex gap-4 justify-center",
			},
				dom.Button(Attrs{
					"onclick": decrement,
					"class":   "w-16 h-16 flex items-center justify-center bg-white/5 hover:bg-white/10 border border-white/10 rounded-full text-2xl transition-all duration-200 hover:scale-110 active:scale-95",
				}, dom.Text("−")),

				dom.Button(Attrs{
					"onclick": reset,
					"class":   "px-6 h-16 flex items-center justify-center bg-white/5 hover:bg-white/10 border border-white/10 rounded-full text-sm font-bold uppercase tracking-wider transition-all duration-200 hover:scale-105 active:scale-95",
				}, dom.Text("Reset")),

				dom.Button(Attrs{
					"onclick": increment,
					"class":   "w-16 h-16 flex items-center justify-center bg-gradient-to-r from-blue-500 to-purple-600 rounded-full text-2xl shadow-lg shadow-purple-500/20 transition-all duration-200 hover:scale-110 active:scale-95",
				}, dom.Text("+")),
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	render.To(dom.CreateElement(CounterExample, nil), "#app")
	select {}
}

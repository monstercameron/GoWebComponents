//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"

	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/examples/internal/exampleboot"
	h "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/ui"
)

func ToggleExample() ui.Node {
	isOn := ui.UseState(false)
	parseCurrentState := isOn.Get()

	parseToggle := ui.UseEvent(func() {
		isOn.Set(!parseCurrentState)
	})

	return h.Div(
		h.FromProps(h.Props{
			Class: "min-h-screen flex items-center justify-center bg-[#0a0a0a] text-white p-4",
		}),
		h.Div(
			h.Class("max-w-md w-full bg-white/5 border border-white/10 rounded-xl backdrop-blur-sm p-8 shadow-2xl"),
			h.H2(
				h.Class("text-3xl font-bold text-center mb-8 bg-clip-text text-transparent bg-gradient-to-r from-blue-400 to-purple-500"),
				"Toggle Example",
			),
			h.Div(
				h.Class("flex flex-col items-center justify-center mb-10"),
				h.Div(
					h.Class(func() string {
						parseBase := "w-24 h-24 rounded-full flex items-center justify-center text-4xl mb-6 transition-all duration-500 shadow-lg "
						if parseCurrentState {
							return parseBase + "bg-green-500/20 text-green-400 shadow-green-500/20 border border-green-500/50"
						}
						return parseBase + "bg-red-500/20 text-red-400 shadow-red-500/20 border border-red-500/50"
					}()),
					h.Text(func() string {
						if parseCurrentState {
							return "ON"
						}
						return "OFF"
					}),
				),
				h.P(
					h.Class("text-gray-400 uppercase tracking-widest text-xs font-semibold"),
					h.Textf("System Status: %s", func() string {
						if parseCurrentState {
							return "ACTIVE"
						}
						return "INACTIVE"
					}()),
				),
			),
			h.Button(
				h.OnClick(parseToggle),
				h.Class(func() string {
					parseBase2 := "w-full px-6 py-4 font-bold rounded-lg transition-all duration-300 transform hover:scale-[1.02] active:scale-[0.98] "
					if parseCurrentState {
						return parseBase2 + "bg-white/5 hover:bg-white/10 border border-white/10 text-gray-300"
					}
					return parseBase2 + "bg-gradient-to-r from-blue-500 to-purple-600 text-white shadow-lg shadow-purple-500/20"
				}()),
				h.Text(func() string {
					if parseCurrentState {
						return "Power Down System"
					}
					return "Initialize System"
				}),
			),
		),
	)
}

func main() {
	fmt.Println("Toggle Example Started")
	exampleboot.RenderExampleRoot(ui.CreateElement(ToggleExample))
	fmt.Println("Toggle Example Rendered")
	exampleboot.WaitExampleRuntime()
}

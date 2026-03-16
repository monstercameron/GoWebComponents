//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

func ToggleExample() ui.Node {
	isOn := ui.UseState(false)
	currentState := isOn.Get()

	toggle := ui.UseEvent(func() {
		isOn.Set(!currentState)
	})

	return html.Div(html.Props{
		Class: "min-h-screen flex items-center justify-center bg-[#0a0a0a] text-white p-4",
	},
		html.Div(html.Props{
			Class: "max-w-md w-full bg-white/5 border border-white/10 rounded-xl backdrop-blur-sm p-8 shadow-2xl",
		},
			html.H2(html.Props{
				Class: "text-3xl font-bold text-center mb-8 bg-clip-text text-transparent bg-gradient-to-r from-blue-400 to-purple-500",
			}, html.Text("Toggle Example")),

			html.Div(html.Props{
				Class: "flex flex-col items-center justify-center mb-10",
			},
				html.Div(html.Props{
					Class: func() string {
						base := "w-24 h-24 rounded-full flex items-center justify-center text-4xl mb-6 transition-all duration-500 shadow-lg "
						if currentState {
							return base + "bg-green-500/20 text-green-400 shadow-green-500/20 border border-green-500/50"
						}
						return base + "bg-red-500/20 text-red-400 shadow-red-500/20 border border-red-500/50"
					}(),
				}, html.Text(func() string {
					if currentState {
						return "ON"
					}
					return "OFF"
				}())),

				html.P(html.Props{
					Class: "text-gray-400 uppercase tracking-widest text-xs font-semibold",
				}, html.Text(fmt.Sprintf("System Status: %s", func() string {
					if currentState {
						return "ACTIVE"
					}
					return "INACTIVE"
				}()))),
			),

			html.Button(html.Props{
				OnClick: toggle,
				Class: func() string {
					base := "w-full px-6 py-4 font-bold rounded-lg transition-all duration-300 transform hover:scale-[1.02] active:scale-[0.98] "
					if currentState {
						return base + "bg-white/5 hover:bg-white/10 border border-white/10 text-gray-300"
					}
					return base + "bg-gradient-to-r from-blue-500 to-purple-600 text-white shadow-lg shadow-purple-500/20"
				}(),
			}, html.Text(func() string {
				if currentState {
					return "Power Down System"
				}
				return "Initialize System"
			}())),
		),
	)
}

func main() {
	fmt.Println("Toggle Example Started")
	ui.Render(ui.CreateElement(ToggleExample), "#app")
	fmt.Println("Toggle Example Rendered")
	select {}
}

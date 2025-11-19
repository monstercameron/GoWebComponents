//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/dom"
	"github.com/monstercameron/GoWebComponents/hooks"
	"github.com/monstercameron/GoWebComponents/render"
)

type Attrs = dom.Attrs
type Element = render.Element

func ToggleExample(_ Attrs) *Element {
	isOn, setIsOn := hooks.UseState(false)
	currentState := isOn()

	toggle := hooks.GoUseFunc(func(event dom.GoEvent) {
		setIsOn(!currentState)
	})

	return dom.Div(Attrs{
		"class": "min-h-screen flex items-center justify-center bg-[#0a0a0a] text-white p-4",
	},
		dom.Div(Attrs{
			"class": "max-w-md w-full bg-white/5 border border-white/10 rounded-xl backdrop-blur-sm p-8 shadow-2xl",
		},
			dom.H2(Attrs{
				"class": "text-3xl font-bold text-center mb-8 bg-clip-text text-transparent bg-gradient-to-r from-blue-400 to-purple-500",
			}, dom.Text("Toggle Example")),

			dom.Div(Attrs{
				"class": "flex flex-col items-center justify-center mb-10",
			},
				dom.Div(Attrs{
					"class": func() string {
						base := "w-24 h-24 rounded-full flex items-center justify-center text-4xl mb-6 transition-all duration-500 shadow-lg "
						if currentState {
							return base + "bg-green-500/20 text-green-400 shadow-green-500/20 border border-green-500/50"
						}
						return base + "bg-red-500/20 text-red-400 shadow-red-500/20 border border-red-500/50"
					}(),
				}, dom.Text(func() string {
					if currentState {
						return "ON"
					}
					return "OFF"
				}())),

				dom.P(Attrs{
					"class": "text-gray-400 uppercase tracking-widest text-xs font-semibold",
				}, dom.Text(fmt.Sprintf("System Status: %s", func() string {
					if currentState {
						return "ACTIVE"
					}
					return "INACTIVE"
				}()))),
			),

			dom.Button(Attrs{
				"onclick": toggle,
				"class": func() string {
					base := "w-full px-6 py-4 font-bold rounded-lg transition-all duration-300 transform hover:scale-[1.02] active:scale-[0.98] "
					if currentState {
						return base + "bg-white/5 hover:bg-white/10 border border-white/10 text-gray-300"
					}
					return base + "bg-gradient-to-r from-blue-500 to-purple-600 text-white shadow-lg shadow-purple-500/20"
				}(),
			}, dom.Text(func() string {
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
	render.To(dom.CreateElement(ToggleExample, nil), "#app")
	fmt.Println("Toggle Example Rendered")
	select {}
}

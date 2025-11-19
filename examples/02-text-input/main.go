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

// Text input component - demonstrates string state management
func TextInputExample(_ Attrs) *Element {
	text, setText := hooks.UseState("")
	currentText := text()

	handleInput := hooks.GoUseFunc(func(event dom.GoEvent) {
		setText(event.GetValue())
	})

	clear := hooks.GoUseFunc(func(event dom.GoEvent) {
		setText("")
	})

	return dom.Div(Attrs{
		"class": "min-h-screen flex items-center justify-center bg-[#0a0a0a] text-white p-4",
	},
		dom.Div(Attrs{
			"class": "max-w-md w-full bg-white/5 border border-white/10 rounded-xl backdrop-blur-sm p-8 shadow-2xl",
		},
			dom.H2(Attrs{
				"class": "text-3xl font-bold text-center mb-8 bg-clip-text text-transparent bg-gradient-to-r from-blue-400 to-purple-500",
			}, dom.Text("Text Input Example")),

			dom.Div(Attrs{
				"class": "mb-6",
			},
				dom.Label(Attrs{
					"class": "block text-gray-400 text-sm font-bold mb-2 uppercase tracking-wider",
				}, dom.Text("Type something")),

				dom.Input(Attrs{
					"type":        "text",
					"value":       currentText,
					"oninput":     handleInput,
					"class":       "w-full px-4 py-3 bg-black/20 border border-white/10 rounded-lg focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 text-white placeholder-gray-600 transition-all",
					"placeholder": "Enter text here...",
				}),
			),

			dom.Div(Attrs{
				"class": "mb-8",
			},
				dom.Button(Attrs{
					"onclick": clear,
					"class":   "w-full px-4 py-2 bg-white/5 hover:bg-white/10 border border-white/10 text-gray-300 font-semibold rounded-lg transition-colors",
				}, dom.Text("Clear Text")),
			),

			dom.Div(Attrs{
				"class": "p-6 bg-black/20 rounded-lg border border-white/5",
			},
				dom.P(Attrs{
					"class": "text-gray-400 text-xs uppercase tracking-widest mb-2",
				}, dom.Text("Live Preview")),
				dom.P(Attrs{
					"class": "text-xl text-white mb-4 font-medium break-all",
				}, dom.Text(func() string {
					if currentText == "" {
						return "..."
					}
					return currentText
				}())),

				dom.Div(Attrs{
					"class": "flex justify-between items-center pt-4 border-t border-white/5",
				},
					dom.Span(Attrs{
						"class": "text-gray-500 text-xs",
					}, dom.Text("Character count")),
					dom.Span(Attrs{
						"class": "text-blue-400 font-mono font-bold",
					}, dom.Text(fmt.Sprintf("%d", len(currentText)))),
				),
			),
		),
	)
}

func main() {
	fmt.Println("🚀 Text Input Example Started")
	render.To(dom.CreateElement(TextInputExample, nil), "#app")
	fmt.Println("✅ Text Input Example Rendered")
	select {}
}

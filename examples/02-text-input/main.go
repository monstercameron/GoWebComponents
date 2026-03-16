//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"time"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

// Text input component - demonstrates string state management
func TextInputExample() ui.Node {
	text := ui.UseState("")
	currentText := text.Get()
	debouncedText := ui.UseDebounced(currentText, 450*time.Millisecond)
	throttledCount := ui.UseThrottled(len(currentText), 250*time.Millisecond)

	handleInput := ui.UseEvent(func(event ui.InputEvent) {
		text.Set(event.GetValue())
	})

	clear := ui.UseEvent(func() {
		text.Set("")
	})

	return html.Div(html.Props{
		Class: "min-h-screen flex items-center justify-center bg-[#0a0a0a] text-white p-4",
	},
		html.Div(html.Props{
			Class: "max-w-md w-full bg-white/5 border border-white/10 rounded-xl backdrop-blur-sm p-8 shadow-2xl",
		},
			html.H2(html.Props{
				Class: "text-3xl font-bold text-center mb-8 bg-clip-text text-transparent bg-gradient-to-r from-blue-400 to-purple-500",
			}, html.Text("Text Input Example")),

			html.Div(html.Props{
				Class: "mb-6",
			},
				html.Label(html.Props{
					Class: "block text-gray-400 text-sm font-bold mb-2 uppercase tracking-wider",
				}, html.Text("Type something")),

				html.Input(html.Props{
					Type:        "text",
					Value:       currentText,
					OnInput:     handleInput,
					Class:       "w-full px-4 py-3 bg-black/20 border border-white/10 rounded-lg focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 text-white placeholder-gray-600 transition-all",
					Placeholder: "Enter text here...",
				}),
			),

			html.Div(html.Props{
				Class: "mb-8",
			},
				html.Button(html.Props{
					OnClick: clear,
					Class:   "w-full px-4 py-2 bg-white/5 hover:bg-white/10 border border-white/10 text-gray-300 font-semibold rounded-lg transition-colors",
				}, html.Text("Clear Text")),
			),

			html.Div(html.Props{Class: "grid gap-4 mb-8 md:grid-cols-2"},
				html.Div(html.Props{Class: "p-4 bg-black/20 rounded-lg border border-white/5"},
					html.P(html.Props{Class: "text-gray-400 text-xs uppercase tracking-widest mb-2"}, html.Text("Immediate input")),
					html.P(html.Props{Class: "text-lg text-white font-medium break-all min-h-[1.75rem]"}, html.Text(func() string {
						if currentText == "" {
							return "..."
						}
						return currentText
					}())),
				),
				html.Div(html.Props{Class: "p-4 bg-black/20 rounded-lg border border-white/5"},
					html.P(html.Props{Class: "text-gray-400 text-xs uppercase tracking-widest mb-2"}, html.Text("Debounced preview")),
					html.P(html.Props{Class: "text-lg text-white font-medium break-all min-h-[1.75rem]"}, html.Text(func() string {
						if debouncedText.Get() == "" {
							return "..."
						}
						return debouncedText.Get()
					}())),
					html.P(html.Props{Class: "mt-2 text-xs text-cyan-300"}, html.Text(func() string {
						if debouncedText.Pending() {
							return "Waiting for debounce window"
						}
						return "Debounced value settled"
					}())),
				),
			),

			html.Div(html.Props{
				Class: "p-6 bg-black/20 rounded-lg border border-white/5",
			},
				html.P(html.Props{
					Class: "text-gray-400 text-xs uppercase tracking-widest mb-2",
				}, html.Text("Live Preview")),
				html.P(html.Props{
					Class: "text-xl text-white mb-4 font-medium break-all",
				}, html.Text(func() string {
					if currentText == "" {
						return "..."
					}
					return currentText
				}())),

				html.Div(html.Props{
					Class: "flex justify-between items-center pt-4 border-t border-white/5",
				},
					html.Span(html.Props{
						Class: "text-gray-500 text-xs",
					}, html.Text("Character count")),
					html.Span(html.Props{
						Class: "text-blue-400 font-mono font-bold",
					}, html.Text(fmt.Sprintf("%d", len(currentText)))),
				),
				html.Div(html.Props{Class: "flex justify-between items-center pt-4 border-t border-white/5 mt-4"},
					html.Span(html.Props{Class: "text-gray-500 text-xs"}, html.Text("Throttled count")),
					html.Span(html.Props{Class: "text-cyan-300 font-mono font-bold"}, html.Text(fmt.Sprintf("%d", throttledCount.Get()))),
				),
				html.P(html.Props{Class: "mt-3 text-xs text-cyan-300"}, html.Text(func() string {
					if throttledCount.Pending() {
						return "Count is throttled while typing"
					}
					return "Count is synced"
				}())),
			),
		),
	)
}

func main() {
	fmt.Println("ðŸš€ Text Input Example Started")
	ui.Render(ui.CreateElement(TextInputExample), "#app")
	fmt.Println("âœ… Text Input Example Rendered")
	select {}
}

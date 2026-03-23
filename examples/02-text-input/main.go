//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"time"

	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"

	h "github.com/monstercameron/GoWebComponents/html/shorthand"
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

	return h.Div(
		h.FromProps(h.Props{
			Class: "min-h-screen flex items-center justify-center bg-[#0a0a0a] text-white p-4",
		}),
		h.Div(
			h.Class("max-w-md w-full bg-white/5 border border-white/10 rounded-xl backdrop-blur-sm p-8 shadow-2xl"),
			h.H2(
				h.Class("text-3xl font-bold text-center mb-8 bg-clip-text text-transparent bg-gradient-to-r from-blue-400 to-purple-500"),
				"Text Input Example",
			),
			h.Div(
				h.Class("mb-6"),
				h.Label(
					h.Class("block text-gray-400 text-sm font-bold mb-2 uppercase tracking-wider"),
					"Type something",
				),
				h.Input(
					h.Type("text"),
					h.Value(currentText),
					h.OnInput(handleInput),
					h.Class("w-full px-4 py-3 bg-black/20 border border-white/10 rounded-lg focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 text-white placeholder-gray-600 transition-all"),
					h.Placeholder("Enter text here..."),
				),
			),
			h.Div(
				h.Class("mb-8"),
				h.Button(
					h.OnClick(clear),
					h.Class("w-full px-4 py-2 bg-white/5 hover:bg-white/10 border border-white/10 text-gray-300 font-semibold rounded-lg transition-colors"),
					"Clear Text",
				),
			),
			h.Div(
				h.Class("grid gap-4 mb-8 md:grid-cols-2"),
				h.Div(
					h.Class("md:col-span-2 p-4 bg-blue-500/10 rounded-lg border border-blue-400/20"),
					h.P(
						h.Class("text-blue-200 text-sm font-semibold uppercase tracking-[0.2em]"),
						"Choosing the right pacing helper",
					),
					h.P(
						h.Class("mt-2 text-sm leading-6 text-slate-300"),
						"Use debouncing when work should wait until typing pauses, like search or validation. Use throttling when updates should continue during typing, but at a fixed rate, like counters or live telemetry.",
					),
				),
				h.Div(
					h.Class("p-4 bg-black/20 rounded-lg border border-white/5"),
					h.P(
						h.Class("text-gray-400 text-xs uppercase tracking-widest mb-2"),
						"Immediate input",
					),
					h.P(
						h.Class("text-lg text-white font-medium break-all min-h-[1.75rem]"),
						h.Text(func() string {
							if currentText == "" {
								return "..."
							}
							return currentText
						}),
					),
				),
				h.Div(
					h.Class("p-4 bg-black/20 rounded-lg border border-white/5"),
					h.P(
						h.Class("text-gray-400 text-xs uppercase tracking-widest mb-2"),
						"Debounced preview",
					),
					h.P(
						h.Class("text-lg text-white font-medium break-all min-h-[1.75rem]"),
						h.Text(func() string {
							if debouncedText.Get() == "" {
								return "..."
							}
							return debouncedText.Get()
						}),
					),
					h.P(
						h.Class("mt-2 text-xs text-cyan-300"),
						h.Text(func() string {
							if debouncedText.Pending() {
								return "Waiting for debounce window"
							}
							return "Debounced value settled"
						}),
					),
				),
			),
			h.Div(
				h.Class("p-6 bg-black/20 rounded-lg border border-white/5"),
				h.P(
					h.Class("text-gray-400 text-xs uppercase tracking-widest mb-2"),
					"Live Preview",
				),
				h.P(
					h.Class("text-xl text-white mb-4 font-medium break-all"),
					h.Text(func() string {
						if currentText == "" {
							return "..."
						}
						return currentText
					}),
				),
				h.Div(
					h.Class("flex justify-between items-center pt-4 border-t border-white/5"),
					h.Span(
						h.Class("text-gray-500 text-xs"),
						"Character count",
					),
					h.Span(
						h.Class("text-blue-400 font-mono font-bold"),
						h.Textf("%d", len(currentText)),
					),
				),
				h.Div(
					h.Class("flex justify-between items-center pt-4 border-t border-white/5 mt-4"),
					h.Span(
						h.Class("text-gray-500 text-xs"),
						"Throttled count",
					),
					h.Span(
						h.Class("text-cyan-300 font-mono font-bold"),
						h.Textf("%d", throttledCount.Get()),
					),
				),
				h.P(
					h.Class("mt-3 text-xs text-cyan-300"),
					h.Text(func() string {
						if throttledCount.Pending() {
							return "Count is throttled while typing"
						}
						return "Count is synced"
					}),
				),
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

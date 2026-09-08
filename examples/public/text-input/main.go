//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"time"

	_ "github.com/monstercameron/GoWebComponents/v6/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/v6/examples/internal/exampleboot"
	"github.com/monstercameron/GoWebComponents/v6/examples/shared"
	h "github.com/monstercameron/GoWebComponents/v6/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// Text input component - demonstrates string state management
func TextInputExample() ui.Node {
	parseText := ui.UseState("")
	parseCurrentText := parseText.Get()
	parseDebouncedText := ui.UseDebounced(parseCurrentText, 450*time.Millisecond)
	parseThrottledCount := ui.UseThrottled(len(parseCurrentText), 250*time.Millisecond)

	handleInput := ui.UseEvent(func(parseEvent ui.InputEvent) {
		parseText.Set(parseEvent.GetValue())
	})

	clear := ui.UseEvent(func() {
		parseText.Set("")
	})

	return shared.ExamplePage(
		"Text Input",
		"ui.UseDebounced + ui.UseThrottled",
		"Compare immediate, debounced, and throttled values from one text field.",
		shared.ExamplePanel("Input",
			h.Div(
				h.ClassStr("space-y-3"),
				h.Label(h.ClassStr("block text-xs font-semibold uppercase tracking-[0.18em] text-slate-400"), "Type something"),
				h.Input(
					h.Type("text"),
					h.Value(parseCurrentText),
					h.OnInput(handleInput),
					h.ClassStr("w-full rounded-2xl border border-white/10 bg-slate-950/70 px-4 py-3 text-slate-100 placeholder:text-slate-500 focus:outline-none"),
					h.Placeholder("Enter text here..."),
				),
				h.Div(
					h.ClassStr("flex flex-wrap gap-2"),
					shared.ExampleButton("Clear", clear),
				),
			),
		),
		shared.ExamplePanel("Output",
			h.Div(
				h.ClassStr("grid gap-3 lg:grid-cols-2"),
				shared.ExampleStat("Immediate", func() string {
					if parseCurrentText == "" {
						return "..."
					}
					return parseCurrentText
				}()),
				shared.ExampleStat("Debounced", func() string {
					if parseDebouncedText.Get() == "" {
						return "..."
					}
					return parseDebouncedText.Get()
				}()),
			),
			h.Div(
				h.ClassStr("grid gap-3 lg:grid-cols-3"),
				shared.ExampleStat("Chars", fmt.Sprintf("%d", len(parseCurrentText))),
				shared.ExampleStat("Throttled", fmt.Sprintf("%d", parseThrottledCount.Get())),
				shared.ExampleStat("Debounce", func() string {
					if parseDebouncedText.Pending() {
						return "Waiting"
					}
					return "Ready"
				}()),
			),
		),
	)
}

func main() {
	fmt.Println("ðŸš€ Text Input Example Started")
	exampleboot.RenderExampleRoot(ui.CreateElement(TextInputExample))
	fmt.Println("âœ… Text Input Example Rendered")
	exampleboot.WaitExampleRuntime()
}

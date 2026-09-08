//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"

	_ "github.com/monstercameron/GoWebComponents/v6/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/v6/examples/internal/exampleboot"
	"github.com/monstercameron/GoWebComponents/v6/examples/shared"
	h "github.com/monstercameron/GoWebComponents/v6/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

func ToggleExample() ui.Node {
	isOn := ui.UseState(false)
	parseCurrentState := isOn.Get()

	parseToggle := ui.UseEvent(func() {
		isOn.Set(!parseCurrentState)
	})

	return shared.ExamplePage(
		"Toggle",
		"ui.UseState",
		"Flip one local boolean and watch the rendered state respond immediately.",
		shared.ExamplePanel("State",
			h.Div(
				h.ClassStr("flex flex-col items-center justify-center gap-4"),
				h.Div(
					h.ClassStr(func() string {
						parseBase := "flex h-24 w-24 items-center justify-center rounded-full border text-3xl font-semibold tracking-[0.18em] transition-all duration-300 "
						if parseCurrentState {
							return parseBase + "border-emerald-400/30 bg-emerald-400/10 text-emerald-100"
						}
						return parseBase + "border-rose-400/30 bg-rose-400/10 text-rose-100"
					}()),
					h.Text(func() string {
						if parseCurrentState {
							return "ON"
						}
						return "OFF"
					}),
				),
				h.Div(
					h.ClassStr("grid w-full gap-3 sm:grid-cols-2"),
					shared.ExampleStat("Signal", func() string {
						if parseCurrentState {
							return "Active"
						}
						return "Idle"
					}()),
					shared.ExampleStat("Boolean", fmt.Sprintf("%t", parseCurrentState)),
				),
			),
		),
		shared.ExamplePanel("Controls",
			h.Div(
				h.ClassStr("flex flex-wrap gap-2"),
				shared.ExampleButton(func() string {
					if parseCurrentState {
						return "Turn Off"
					}
					return "Turn On"
				}(), parseToggle),
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

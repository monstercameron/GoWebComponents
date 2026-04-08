//go:build js && wasm
// +build js,wasm

package main

import (
	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/ui"
)

// renderNotFoundPageCompact renders the portfolio 404 route in the shared example shell.
func renderNotFoundPageCompact(_ Attrs) *Element {
	return Div(
		Attrs{"class": "min-h-screen bg-[#0a0a0a] text-white"},
		shared.ExamplePage(
			"Portfolio Site",
			"Not found",
			"Unknown routes should send people back to the landing page or the docs route without extra chrome.",
			shared.ExamplePanel("Current route",
				shared.ExampleStat("State", "404"),
			),
			shared.ExamplePanel("Navigate",
				Div(
					Attrs{"class": "flex flex-wrap gap-3"},
					shared.ExampleButton("Go home", ui.UseEvent(func() { router.Navigate("/") })),
					shared.ExampleButton("View docs", ui.UseEvent(func() { router.Navigate("/docs") })),
				),
			),
		),
	)
}

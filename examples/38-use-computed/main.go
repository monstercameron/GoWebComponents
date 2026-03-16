//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/state"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

type pricingSummary struct {
	Subtotal int
	Support  int
	Total    int
	Tier     string
}

func useComputedExample() ui.Node {
	seats := state.UseAtom("catalog-state-computed-seats", 6)
	price := state.UseAtom("catalog-state-computed-price", 24)

	summary := state.UseComputed(func() pricingSummary {
		subtotal := seats.Get() * price.Get()
		support := 12
		tier := "Growth"
		if seats.Get() >= 10 {
			support = 24
			tier = "Scale"
		}
		return pricingSummary{
			Subtotal: subtotal,
			Support:  support,
			Total:    subtotal + support,
			Tier:     tier,
		}
	}, seats.Get(), price.Get())

	decreaseSeats := ui.UseEvent(func() {
		if seats.Get() > 1 {
			seats.Update(func(previous int) int { return previous - 1 })
		}
	})
	increaseSeats := ui.UseEvent(func() { seats.Update(func(previous int) int { return previous + 1 }) })
	standard := ui.UseEvent(func() { price.Set(24) })
	premium := ui.UseEvent(func() { price.Set(32) })

	computed := summary.Get()

	return shared.ExamplePage(
		"state.UseComputed",
		"Memoize typed derived values inside a component",
		"UseComputed keeps render-time derivations explicit: you provide a typed compute function and the concrete dependency values that should trigger recomputation.",
		shared.ExamplePanel("Inputs",
			html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-3"},
				shared.ExampleButton("-1 seat", decreaseSeats),
				shared.ExampleButton("+1 seat", increaseSeats),
				shared.ExampleButton("Standard $24", standard),
				shared.ExampleButton("Premium $32", premium),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-2"},
				shared.ExampleStat("Seats", fmt.Sprintf("%d", seats.Get())),
				shared.ExampleStat("Price per seat", fmt.Sprintf("$%d", price.Get())),
			),
		),
		shared.ExamplePanel("Computed output",
			html.Div(html.Props{Class: "mt-3 grid gap-4 md:grid-cols-4"},
				shared.ExampleStat("Subtotal", fmt.Sprintf("$%d", computed.Subtotal)),
				shared.ExampleStat("Support", fmt.Sprintf("$%d", computed.Support)),
				shared.ExampleStat("Total", fmt.Sprintf("$%d", computed.Total)),
				shared.ExampleStat("Tier", computed.Tier),
			),
			shared.ExampleCode(
				`summary := state.UseComputed(func() pricingSummary { ... }, seats.Get(), price.Get())`,
				`computed := summary.Get()`,
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(useComputedExample), "#app")
	select {}
}

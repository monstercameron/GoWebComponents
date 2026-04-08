//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"github.com/monstercameron/GoWebComponents/examples/internal/exampleboot"
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
	parseSeats := state.UseAtom("catalog-state-computed-seats", 6)
	parsePrice := state.UseAtom("catalog-state-computed-price", 24)

	parseSummary := state.UseComputed(func() pricingSummary {
		parseSubtotal := parseSeats.Get() * parsePrice.Get()
		parseSupport := 12
		parseTier := "Growth"
		if parseSeats.Get() >= 10 {
			parseSupport = 24
			parseTier = "Scale"
		}
		return pricingSummary{
			Subtotal: parseSubtotal,
			Support:  parseSupport,
			Total:    parseSubtotal + parseSupport,
			Tier:     parseTier,
		}
	}, parseSeats.Get(), parsePrice.Get())

	parseDecreaseSeats := ui.UseEvent(func() {
		if parseSeats.Get() > 1 {
			parseSeats.Update(func(parsePrevious int) int { return parsePrevious - 1 })
		}
	})
	parseIncreaseSeats := ui.UseEvent(func() { parseSeats.Update(func(parsePrevious2 int) int { return parsePrevious2 + 1 }) })
	parseStandard := ui.UseEvent(func() { parsePrice.Set(24) })
	parsePremium := ui.UseEvent(func() { parsePrice.Set(32) })

	parseComputed := parseSummary.Get()

	return shared.ExamplePage(
		"state.UseComputed",
		"Memoize typed derived values inside a component",
		"UseComputed keeps render-time derivations explicit: you provide a typed compute function and the concrete dependency values that should trigger recomputation.",
		shared.ExamplePanel("Inputs",
			html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-3"},
				shared.ExampleButton("-1 seat", parseDecreaseSeats),
				shared.ExampleButton("+1 seat", parseIncreaseSeats),
				shared.ExampleButton("Standard $24", parseStandard),
				shared.ExampleButton("Premium $32", parsePremium),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-2"},
				shared.ExampleStat("Seats", fmt.Sprintf("%d", parseSeats.Get())),
				shared.ExampleStat("Price per seat", fmt.Sprintf("$%d", parsePrice.Get())),
			),
		),
		shared.ExamplePanel("Computed output",
			html.Div(html.Props{Class: "mt-3 grid gap-4 md:grid-cols-4"},
				shared.ExampleStat("Subtotal", fmt.Sprintf("$%d", parseComputed.Subtotal)),
				shared.ExampleStat("Support", fmt.Sprintf("$%d", parseComputed.Support)),
				shared.ExampleStat("Total", fmt.Sprintf("$%d", parseComputed.Total)),
				shared.ExampleStat("Tier", parseComputed.Tier),
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
	exampleboot.RenderExampleRoot(ui.CreateElement(useComputedExample))
	exampleboot.WaitExampleRuntime()
}

//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"strings"

	"github.com/monstercameron/GoWebComponents/v5/anim"
	"github.com/monstercameron/GoWebComponents/v5/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v5/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/v5/examples/shared"
	"github.com/monstercameron/GoWebComponents/v5/html"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

// flipExample demonstrates anim.DiffKeyedRects: given the before/after layout rectangles of a keyed
// list, it classifies each item as entering, exiting, or moving and computes the FLIP invert
// transform for survivors — the math a UI layer applies as transforms. It owns no DOM and no clock.
func flipExample() ui.Node {
	parsePrev := []anim.KeyedRect{
		{Key: "a", Rect: anim.Rect{X: 0, Y: 0, Width: 200, Height: 40}},
		{Key: "b", Rect: anim.Rect{X: 0, Y: 48, Width: 200, Height: 40}},
		{Key: "c", Rect: anim.Rect{X: 0, Y: 96, Width: 200, Height: 40}},
	}
	parseNext := []anim.KeyedRect{
		{Key: "c", Rect: anim.Rect{X: 0, Y: 0, Width: 200, Height: 40}},
		{Key: "b", Rect: anim.Rect{X: 0, Y: 48, Width: 200, Height: 40}},
		{Key: "d", Rect: anim.Rect{X: 0, Y: 96, Width: 200, Height: 40}},
	}
	parseTransition := anim.DiffKeyedRects(parsePrev, parseNext)

	parseMoves := make([]ui.Node, 0)
	for _, parseKey := range parseTransition.MovedKeys() {
		parseTransform := parseTransition.Moving[parseKey]
		parseMoves = append(parseMoves, html.Div(html.Props{Class: "font-mono text-sm text-sky-300"},
			html.Text(fmt.Sprintf("%s: translateY(%.0fpx)", parseKey, parseTransform.TranslateY))))
	}

	return shared.ExamplePage(
		"anim.DiffKeyedRects (FLIP)",
		"Classify a keyed-list layout change into enter / exit / move",
		"Given the old and new positions of a keyed list, DiffKeyedRects reports which items entered, exited, and moved, and computes each survivor's FLIP invert transform. A UI layer then animates from the inverted transform back to identity. The Transition state machine and MotionPreference handle enter/exit timing and reduced motion.",
		shared.ExamplePanel("List reordered (c→top, a removed, d added)",
			html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-4"},
				shared.ExampleStat("Entering", strings.Join(parseTransition.Entering, ", ")),
				shared.ExampleStat("Exiting", strings.Join(parseTransition.Exiting, ", ")),
				shared.ExampleStat("Moved", strings.Join(parseTransition.MovedKeys(), ", ")),
			),
			html.Div(html.Props{Class: "mt-6 space-y-1 rounded-lg bg-slate-950 p-4"}, parseMoves...),
		),
	)
}

func main() {
	exampleboot.RenderExampleRoot(ui.CreateElement(flipExample))
	exampleboot.WaitExampleRuntime()
}

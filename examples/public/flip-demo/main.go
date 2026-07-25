//go:build js && wasm

// Command flip-demo renders a keyed list that reorders on click and computes the FLIP invert
// transform for each item from its measured before/after rects (via anim.ComputeFLIP), so the
// browser-lane test can verify keyed-list FLIP in a real browser. Each item records its last
// computed invert translateY in data-flip-dy (a persisted attribute, so the assertion is not
// timing-sensitive).
package main

import (
	"strconv"

	"github.com/monstercameron/GoWebComponents/v5/anim"
	. "github.com/monstercameron/GoWebComponents/v5/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v5/interop"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

func toAnimRect(parseRect interop.Rect) anim.Rect {
	return anim.Rect{X: parseRect.X, Y: parseRect.Top, Width: parseRect.Width, Height: parseRect.Height}
}

type itemProps struct {
	id    string
	label string
}

// flipItem measures its current rect, computes the FLIP invert from the previous render's
// rect, and records the invert translateY in data-flip-dy.
func flipItem(parseProps itemProps) ui.Node {
	parseRef := ui.UseDOMRef()
	parseGeo := ui.UseElementGeometry(parseRef)
	parsePrev := ui.UseRef(parseGeo)
	parseFlip := anim.ComputeFLIP(toAnimRect(parsePrev.Get()), toAnimRect(parseGeo))
	parsePrev.Set(parseGeo)

	return Div(
		Ref(parseRef),
		FromProps(Props{
			ID:    parseProps.id,
			Class: "flip-item",
			Key:   parseProps.id,
			Style: map[string]string{"height": "40px"},
			Raw:   map[string]any{"data-flip-dy": strconv.FormatFloat(parseFlip.TranslateY, 'f', 1, 64)},
		}),
		Text(parseProps.label),
	)
}

// App is the FLIP demo: a shuffle button + a keyed list.
func App() ui.Node {
	parseOrder := ui.UseState([]int{0, 1, 2})
	parseItems := []itemProps{
		{id: "item-a", label: "Alpha"},
		{id: "item-b", label: "Bravo"},
		{id: "item-c", label: "Charlie"},
	}
	parseShuffle := ui.UseEvent(func() {
		parseOrder.Update(func(parseOld []int) []int {
			return []int{parseOld[2], parseOld[0], parseOld[1]} // rotate
		})
	})

	parseChildren := []any{
		Button(FromProps(Props{ID: "shuffle", Type: "button", OnClick: parseShuffle}), Text("Shuffle")),
	}
	for _, parseIdx := range parseOrder.Get() {
		parseChildren = append(parseChildren, ui.CreateElement(flipItem, parseItems[parseIdx]))
	}
	parseArgs := append([]any{FromProps(Props{ID: "app-root"})}, parseChildren...)
	return Div(parseArgs...)
}

func main() {
	ui.Render(ui.CreateElement(App), "#app")
	select {}
}

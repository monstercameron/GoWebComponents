//go:build js && wasm

// Command radiogroup is the e2e fixture for U5: an interactive WAI-ARIA radio
// group built by composing ui.UseCompositeNavigation (the existing roving-
// tabindex engine) with role=radio/aria-checked and the G2 DOM ref for
// focus-follows-selection — no per-control reimplementation.
package main

import (
	"strconv"

	. "github.com/monstercameron/GoWebComponents/v4/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

func boolStr(parseValue bool) string {
	if parseValue {
		return "true"
	}
	return "false"
}

type radioProps struct {
	id       string
	label    string
	active   bool
	tabIndex int
	onSelect func()
}

func radio(parseProps radioProps) ui.Node {
	parseRef := ui.UseDOMRef()
	// Focus follows selection: when this radio becomes active, focus it.
	ui.UseEffect(func() func() {
		if parseProps.active {
			parseRef.Focus()
		}
		return nil
	}, parseProps.active)

	return Div(
		Ref(parseRef),
		FromProps(Props{
			ID:   parseProps.id,
			Role: "radio",
			// tabindex via Raw so 0 is not omitted (roving tab stop).
			Raw:     map[string]any{"tabindex": strconv.Itoa(parseProps.tabIndex)},
			Aria:    map[string]string{"checked": boolStr(parseProps.active)},
			OnClick: ui.UseEvent(parseProps.onSelect),
		}),
		Text(parseProps.label),
	)
}

func App() ui.Node {
	parseItems := []ui.CompositeItem{{ID: "opt-0"}, {ID: "opt-1"}, {ID: "opt-2"}}
	parseLabels := []string{"Small", "Medium", "Large"}
	parseNav := ui.UseCompositeNavigation(parseItems, ui.CompositeNavigationOptions{
		Orientation: "horizontal",
		Loop:        true,
	})

	parseArgs := []any{
		FromProps(Props{
			ID:        "group",
			Role:      "radiogroup",
			OnKeyDown: ui.UseEvent(func(parseEvent ui.KeyboardEvent) { parseNav.OnKeyDown(parseEvent) }),
		}),
	}
	for parseIndex := range parseItems {
		parseI := parseIndex
		parseArgs = append(parseArgs, ui.CreateElement(radio, radioProps{
			id:       parseItems[parseI].ID,
			label:    parseLabels[parseI],
			active:   parseNav.IsActive(parseI),
			tabIndex: parseNav.TabIndex(parseI),
			onSelect: func() { parseNav.SetActive(parseI) },
		}))
	}
	return Div(parseArgs...)
}

func main() {
	ui.Render(ui.CreateElement(App), "#app")
	select {}
}

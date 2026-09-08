// Package components holds the headless component templates that `gwc add` copies
// into your project. They are compiled here so the catalog never ships a broken
// component; on copy, only the package clause changes — you own the result.
package components

import (
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// DisclosureProps configures a Disclosure.
type DisclosureProps struct {
	// Summary is the always-visible trigger label.
	Summary string
	// DefaultOpen sets the initial expanded state.
	DefaultOpen bool
	// Children render inside the collapsible region.
	Children []ui.Node
}

// Disclosure is an accessible show/hide region implementing the WAI-ARIA disclosure
// pattern: a <button> carries aria-expanded and aria-controls pointing at a region
// whose visibility is driven by component state, so assistive technology always
// announces the correct expanded/collapsed state. Restyle freely via the gwc-disclosure
// classes — you own this file.
func Disclosure(parseProps DisclosureProps) ui.Node {
	parseOpen := ui.UseState(parseProps.DefaultOpen)
	parseRegionID := ui.UseId() + "-region"

	parseToggle := ui.UseEvent(func() {
		parseOpen.Update(func(parsePrev bool) bool { return !parsePrev })
	})

	return html.Div(html.Props{Class: "gwc-disclosure"},
		html.Button(html.Props{
			Type:    "button",
			Class:   "gwc-disclosure-trigger",
			OnClick: parseToggle,
			Aria: map[string]string{
				"expanded": disclosureBoolAttr(parseOpen.Get()),
				"controls": parseRegionID,
			},
		}, html.Text(parseProps.Summary)),
		html.Tag("div", html.Props{
			ID:     parseRegionID,
			Class:  "gwc-disclosure-region",
			Role:   "region",
			Hidden: !parseOpen.Get(),
		}, parseProps.Children...),
	)
}

// disclosureBoolAttr renders a Go bool as an ARIA "true"/"false" string.
func disclosureBoolAttr(parseValue bool) string {
	if parseValue {
		return "true"
	}
	return "false"
}

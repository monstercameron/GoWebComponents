package components

import (
	"github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

// TabItem is one tab and its panel content.
type TabItem struct {
	// ID is a stable, unique identifier for the tab (used to link tab and panel).
	ID string
	// Label is the tab's button text.
	Label string
	// Content renders inside the tab's panel.
	Content []ui.Node
}

// TabsProps configures a Tabs widget.
type TabsProps struct {
	// ID is the base id for the tablist.
	ID string
	// Tabs are the tabs, in order.
	Tabs []TabItem
	// DefaultIndex is the initially selected tab.
	DefaultIndex int
}

// Tabs implements the WAI-ARIA tabs pattern: a role="tablist" of role="tab" buttons —
// each with aria-selected, aria-controls, and a single roving tab stop (tabindex 0 on the
// active tab, -1 on the rest) — controlling role="tabpanel" regions linked back with
// aria-labelledby. Selection is click-driven here; to add arrow-key roving, wire the tab
// buttons through ui.UseCompositeNavigation. You own this file — restyle via the gwc-tabs
// classes.
func Tabs(parseProps TabsProps) ui.Node {
	parseActive := ui.UseState(parseProps.DefaultIndex)
	parseSelected := parseActive.Get()

	parseTabButtons := make([]ui.Node, 0, len(parseProps.Tabs))
	parsePanels := make([]ui.Node, 0, len(parseProps.Tabs))
	for parseIndex, parseTab := range parseProps.Tabs {
		parseTabID := parseTab.ID + "-tab"
		parsePanelID := parseTab.ID + "-panel"
		parseIsActive := parseIndex == parseSelected

		parseIndexCopy := parseIndex
		parseSelect := ui.UseEvent(func() {
			parseActive.Set(parseIndexCopy)
		})
		// Arrow-key roving (WAI-ARIA tabs): Left/Right move selection with wrap-around, Home/End
		// jump to the first/last tab. Selection follows focus; the newly-active tab becomes the
		// single roving tab stop (tabindex 0) on re-render.
		parseKeyNav := ui.UseEvent(func(parseEvent ui.KeyboardEvent) {
			parseNext := tabsNextIndex(parseEvent.GetKey(), parseIndexCopy, len(parseProps.Tabs))
			if parseNext != parseIndexCopy {
				parseEvent.PreventDefault()
				parseActive.Set(parseNext)
			}
		})

		parseTabButtons = append(parseTabButtons, html.Button(html.Props{
			ID:        parseTabID,
			Class:     "gwc-tabs-tab",
			Type:      "button",
			Role:      "tab",
			OnClick:   parseSelect,
			OnKeyDown: parseKeyNav,
			// tabindex must always render (0 and -1), so it goes through Raw — Props.TabIndex==0
			// is intentionally omitted by the serializer, which would erase the roving tab stop.
			Raw: map[string]any{"tabindex": tabsTabIndex(parseIsActive)},
			Aria: map[string]string{
				"selected": tabsBoolAttr(parseIsActive),
				"controls": parsePanelID,
			},
		}, html.Text(parseTab.Label)))

		parsePanels = append(parsePanels, html.Tag("div", html.Props{
			ID:     parsePanelID,
			Class:  "gwc-tabs-panel",
			Role:   "tabpanel",
			Hidden: !parseIsActive,
			Aria:   map[string]string{"labelledby": parseTabID},
		}, parseTab.Content...))
	}

	parseChildren := make([]ui.Node, 0, len(parsePanels)+1)
	parseChildren = append(parseChildren, html.Tag("div", html.Props{
		ID:    parseProps.ID,
		Class: "gwc-tabs-list",
		Role:  "tablist",
	}, parseTabButtons...))
	parseChildren = append(parseChildren, parsePanels...)

	return html.Div(html.Props{Class: "gwc-tabs"}, parseChildren...)
}

// tabsBoolAttr renders a Go bool as an ARIA "true"/"false" string.
func tabsBoolAttr(parseValue bool) string {
	if parseValue {
		return "true"
	}
	return "false"
}

// tabsTabIndex returns the roving tab-stop index: 0 for the active tab, -1 otherwise.
func tabsTabIndex(parseActive bool) string {
	if parseActive {
		return "0"
	}
	return "-1"
}

// tabsNextIndex computes the next selected tab for a keyboard event under the WAI-ARIA tabs
// pattern: ArrowRight advances (wrapping past the end), ArrowLeft retreats (wrapping past the
// start), Home selects the first tab and End the last; any other key leaves selection unchanged.
// It is pure so the keyboard logic is unit-testable without a browser.
func tabsNextIndex(parseKey string, parseCurrent int, parseCount int) int {
	if parseCount <= 0 {
		return parseCurrent
	}
	switch parseKey {
	case "ArrowRight", "ArrowDown":
		return (parseCurrent + 1) % parseCount
	case "ArrowLeft", "ArrowUp":
		return (parseCurrent - 1 + parseCount) % parseCount
	case "Home":
		return 0
	case "End":
		return parseCount - 1
	default:
		return parseCurrent
	}
}

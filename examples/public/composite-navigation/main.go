//go:build js && wasm
// +build js,wasm

package main

import (
	"github.com/monstercameron/GoWebComponents/v5/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v5/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/v5/examples/shared"
	"github.com/monstercameron/GoWebComponents/v5/html"
	"github.com/monstercameron/GoWebComponents/v5/ui"
	"github.com/monstercameron/GoWebComponents/v5/utils"
)

type compositePanel struct {
	Title   string
	Summary string
}

var tabItems = []ui.CompositeItem{
	{ID: "tab-overview", Text: "Overview"},
	{ID: "tab-api", Text: "API checklist"},
	{ID: "tab-notes", Text: "Release notes"},
}

var tabPanels = []compositePanel{
	{Title: "Overview", Summary: "The first tab explains the roving tabindex path. Focus stays on one tab stop while arrow keys move between siblings."},
	{Title: "API checklist", Summary: "Home and End jump to the first and last enabled tab, while the active tab exposes aria-selected and controls one panel."},
	{Title: "Release notes", Summary: "The hook stays reusable because it does not own rendering. The page decides whether the active state becomes tabs, menus, or something else."},
}

var listboxItems = []ui.CompositeItem{
	{ID: "owner-design", Text: "Design system"},
	{ID: "owner-platform", Text: "Platform"},
	{ID: "owner-release", Text: "Release engineering"},
	{ID: "owner-support", Text: "Support desk"},
}

func compositeNavigationExample() ui.Node {
	parseTabNav := ui.UseCompositeNavigation(tabItems, ui.CompositeNavigationOptions{Orientation: "horizontal", Loop: true})
	parseListboxNav := ui.UseCompositeNavigation(listboxItems, ui.CompositeNavigationOptions{Orientation: "vertical", Loop: true})

	parseSelectedTab := tabPanels[parseTabNav.ActiveIndex()]
	parseSelectedOwner := listboxItems[parseListboxNav.ActiveIndex()]
	parseTabKeyDown := ui.UseEvent(func(parseEvent ui.KeyboardEvent) { parseTabNav.OnKeyDown(parseEvent) })
	parseListboxKeyDown := ui.UseEvent(func(parseEvent2 ui.KeyboardEvent) { parseListboxNav.OnKeyDown(parseEvent2) })

	parseTabButtons := make([]ui.Node, 0, len(tabItems))
	for parseIndex, parseItem := range tabItems {
		parseCurrentIndex := parseIndex
		parseCurrentItem := parseItem
		parseClassName := "rounded-full border px-4 py-3 text-sm font-semibold transition"
		if parseTabNav.IsActive(parseIndex) {
			parseClassName += " border-cyan-400/40 bg-cyan-400/10 text-cyan-100"
		} else {
			parseClassName += " border-white/10 text-slate-200"
		}
		parseTabButtons = append(parseTabButtons,
			html.Button(html.Props{
				ID:        parseCurrentItem.ID,
				Role:      "tab",
				Class:     parseClassName,
				OnKeyDown: parseTabKeyDown,
				OnClick:   ui.UseEvent(func() { parseTabNav.SetActive(parseCurrentIndex) }),
				Aria: map[string]string{
					"selected": map[bool]string{true: "true", false: "false"}[parseTabNav.IsActive(parseIndex)],
					"controls": "tab-panel",
				},
				Raw: map[string]interface{}{"tabIndex": parseTabNav.TabIndex(parseIndex)},
			}, html.Text(parseCurrentItem.Text)),
		)
	}

	parseOptions := make([]ui.Node, 0, len(listboxItems))
	for parseIndex2, parseItem2 := range listboxItems {
		parseCurrentIndex2 := parseIndex2
		parseCurrentItem2 := parseItem2
		parseClassName2 := "rounded-[1.25rem] border px-4 py-3 text-left text-sm transition"
		if parseListboxNav.IsActive(parseIndex2) {
			parseClassName2 += " border-emerald-400/40 bg-emerald-400/10 text-emerald-50"
		} else {
			parseClassName2 += " border-white/10 bg-slate-950/50 text-slate-200"
		}
		parseOptions = append(parseOptions,
			html.Div(html.Props{
				ID:    parseCurrentItem2.ID,
				Role:  "option",
				Class: parseClassName2,
				OnClick: ui.UseEvent(func() {
					parseListboxNav.SetActive(parseCurrentIndex2)
				}),
				Aria: map[string]string{
					"selected": map[bool]string{true: "true", false: "false"}[parseListboxNav.IsActive(parseIndex2)],
				},
			}, html.Text(parseCurrentItem2.Text)),
		)
	}

	return shared.ExamplePage(
		"ui.UseCompositeNavigation",
		"Roving tabindex, arrow-key movement, Home/End, and typeahead for composite widgets",
		"UseCompositeNavigation keeps the keyboard model reusable for tabs, listboxes, menus, and similar widgets. The page below applies the same hook to a roving-tabindex tablist and an aria-activedescendant listbox.",
		shared.ExamplePanel("Tabs",
			html.P(html.Props{Class: "mt-3 text-slate-300"}, html.Text("Focus any tab and use ArrowLeft, ArrowRight, Home, or End. The hook controls tabindex, selection state, and active-descendant data without owning the visual styling.")),
			html.Div(html.Props{Role: "tablist", Class: "mt-6 flex flex-wrap gap-3"}, parseTabButtons...),
			html.Div(html.Props{ID: "tab-panel", Role: "tabpanel", Class: "mt-6 rounded-[1.75rem] border border-white/10 bg-slate-950/45 p-6"},
				html.H2(html.Props{Class: "text-2xl font-bold text-white"}, html.Text(parseSelectedTab.Title)),
				html.P(html.Props{Class: "mt-3 text-sm leading-7 text-slate-300"}, html.Text(parseSelectedTab.Summary)),
			),
		),
		shared.ExamplePanel("Listbox and typeahead",
			html.P(html.Props{Class: "mt-3 text-slate-300"}, html.Text("Focus the listbox container and use ArrowUp, ArrowDown, Home, End, or type the first letter of an owner name. The container exposes aria-activedescendant while the options stay simple semantic nodes.")),
			html.Div(html.Props{
				ID:        "owner-listbox",
				Role:      "listbox",
				Class:     "mt-6 grid gap-3 rounded-[1.75rem] border border-white/10 bg-slate-950/45 p-5",
				OnKeyDown: parseListboxKeyDown,
				Aria: map[string]string{
					"label":            "Owner",
					"activedescendant": parseListboxNav.ActiveDescendant(),
				},
				Raw: map[string]interface{}{"tabIndex": 0},
			}, parseOptions...),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-2"},
				shared.ExampleStat("Active tab", parseSelectedTab.Title),
				shared.ExampleStat("Active owner", parseSelectedOwner.Text),
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	exampleboot.RenderExampleRoot(ui.CreateElement(compositeNavigationExample))
	exampleboot.WaitExampleRuntime()
}

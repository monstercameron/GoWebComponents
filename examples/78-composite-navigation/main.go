//go:build js && wasm
// +build js,wasm

package main

import (
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
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
	tabNav := ui.UseCompositeNavigation(tabItems, ui.CompositeNavigationOptions{Orientation: "horizontal", Loop: true})
	listboxNav := ui.UseCompositeNavigation(listboxItems, ui.CompositeNavigationOptions{Orientation: "vertical", Loop: true})

	selectedTab := tabPanels[tabNav.ActiveIndex()]
	selectedOwner := listboxItems[listboxNav.ActiveIndex()]
	tabKeyDown := ui.UseEvent(func(event ui.KeyboardEvent) { tabNav.OnKeyDown(event) })
	listboxKeyDown := ui.UseEvent(func(event ui.KeyboardEvent) { listboxNav.OnKeyDown(event) })

	tabButtons := make([]ui.Node, 0, len(tabItems))
	for index, item := range tabItems {
		currentIndex := index
		currentItem := item
		className := "rounded-full border px-4 py-3 text-sm font-semibold transition"
		if tabNav.IsActive(index) {
			className += " border-cyan-400/40 bg-cyan-400/10 text-cyan-100"
		} else {
			className += " border-white/10 text-slate-200"
		}
		tabButtons = append(tabButtons,
			html.Button(html.Props{
				ID:        currentItem.ID,
				Role:      "tab",
				Class:     className,
				OnKeyDown: tabKeyDown,
				OnClick:   ui.UseEvent(func() { tabNav.SetActive(currentIndex) }),
				Aria: map[string]string{
					"selected": map[bool]string{true: "true", false: "false"}[tabNav.IsActive(index)],
					"controls": "tab-panel",
				},
				Raw: map[string]interface{}{"tabIndex": tabNav.TabIndex(index)},
			}, html.Text(currentItem.Text)),
		)
	}

	options := make([]ui.Node, 0, len(listboxItems))
	for index, item := range listboxItems {
		currentIndex := index
		currentItem := item
		className := "rounded-[1.25rem] border px-4 py-3 text-left text-sm transition"
		if listboxNav.IsActive(index) {
			className += " border-emerald-400/40 bg-emerald-400/10 text-emerald-50"
		} else {
			className += " border-white/10 bg-slate-950/50 text-slate-200"
		}
		options = append(options,
			html.Div(html.Props{
				ID:    currentItem.ID,
				Role:  "option",
				Class: className,
				OnClick: ui.UseEvent(func() {
					listboxNav.SetActive(currentIndex)
				}),
				Aria: map[string]string{
					"selected": map[bool]string{true: "true", false: "false"}[listboxNav.IsActive(index)],
				},
			}, html.Text(currentItem.Text)),
		)
	}

	return shared.ExamplePage(
		"ui.UseCompositeNavigation",
		"Roving tabindex, arrow-key movement, Home/End, and typeahead for composite widgets",
		"UseCompositeNavigation keeps the keyboard model reusable for tabs, listboxes, menus, and similar widgets. The page below applies the same hook to a roving-tabindex tablist and an aria-activedescendant listbox.",
		shared.ExamplePanel("Tabs",
			html.P(html.Props{Class: "mt-3 text-slate-300"}, html.Text("Focus any tab and use ArrowLeft, ArrowRight, Home, or End. The hook controls tabindex, selection state, and active-descendant data without owning the visual styling.")),
			html.Div(html.Props{Role: "tablist", Class: "mt-6 flex flex-wrap gap-3"}, tabButtons...),
			html.Div(html.Props{ID: "tab-panel", Role: "tabpanel", Class: "mt-6 rounded-[1.75rem] border border-white/10 bg-slate-950/45 p-6"},
				html.H2(html.Props{Class: "text-2xl font-bold text-white"}, html.Text(selectedTab.Title)),
				html.P(html.Props{Class: "mt-3 text-sm leading-7 text-slate-300"}, html.Text(selectedTab.Summary)),
			),
		),
		shared.ExamplePanel("Listbox and typeahead",
			html.P(html.Props{Class: "mt-3 text-slate-300"}, html.Text("Focus the listbox container and use ArrowUp, ArrowDown, Home, End, or type the first letter of an owner name. The container exposes aria-activedescendant while the options stay simple semantic nodes.")),
			html.Div(html.Props{
				ID:        "owner-listbox",
				Role:      "listbox",
				Class:     "mt-6 grid gap-3 rounded-[1.75rem] border border-white/10 bg-slate-950/45 p-5",
				OnKeyDown: listboxKeyDown,
				Aria: map[string]string{
					"activedescendant": listboxNav.ActiveDescendant(),
				},
				Raw: map[string]interface{}{"tabIndex": 0},
			}, options...),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-2"},
				shared.ExampleStat("Active tab", selectedTab.Title),
				shared.ExampleStat("Active owner", selectedOwner.Text),
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(compositeNavigationExample), "#app")
	select {}
}

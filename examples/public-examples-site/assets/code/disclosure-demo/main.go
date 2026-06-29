//go:build js && wasm

// Command disclosure-demo renders the WAI-ARIA disclosure pattern (the `gwc add disclosure`
// component shape) as a real wasm app, so the browser-lane test can drive the interaction:
// a button toggles aria-expanded and shows/hides the controlled region.
package main

import (
	"github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

// App is the disclosure demo.
func App() ui.Node {
	parseOpen := ui.UseState(false)
	parseToggle := ui.UseEvent(func() {
		parseOpen.Update(func(parsePrev bool) bool { return !parsePrev })
	})

	parseExpanded := "false"
	if parseOpen.Get() {
		parseExpanded = "true"
	}

	parseChildren := []ui.Node{
		html.Button(html.Props{
			ID:      "disclosure-trigger",
			Type:    "button",
			OnClick: parseToggle,
			Aria:    map[string]string{"expanded": parseExpanded, "controls": "disclosure-region"},
		}, html.Text("Details")),
	}
	if parseOpen.Get() {
		parseChildren = append(parseChildren, html.Tag("div", html.Props{
			ID:   "disclosure-region",
			Role: "region",
		}, html.Text("Hidden content revealed")))
	}
	return html.Div(html.Props{ID: "app-root"}, parseChildren...)
}

func main() {
	ui.Render(ui.CreateElement(App), "#app")
	select {}
}

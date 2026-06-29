// Package gallery renders a component workbench as a GoWebComponents component — the visual
// half of gwc workbench (FB5). It is the gallery view over the same workbench.Story list
// that workbench.RunStories smoke-tests, so the stories you see in the gallery are exactly
// the ones verified under `go test`. The framework renders its own component explorer.
package gallery

import (
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/workbench"
)

// Gallery renders each story as a labeled section containing its live preview. Restyle via
// the gwc-workbench classes.
func Gallery(parseStories ...workbench.Story) ui.Node {
	parseSections := make([]ui.Node, 0, len(parseStories))
	for _, parseStory := range parseStories {
		parseChildren := []ui.Node{
			html.Tag("h3", html.Props{Class: "gwc-workbench-story-name"}, html.Text(parseStory.Name)),
		}
		if parseStory.Render != nil {
			if parsePreview := parseStory.Render(); parsePreview != nil {
				parseChildren = append(parseChildren, html.Div(html.Props{Class: "gwc-workbench-preview"}, parsePreview))
			}
		}
		parseSections = append(parseSections, html.Tag("section", html.Props{
			Class: "gwc-workbench-story",
			Role:  "listitem",
		}, parseChildren...))
	}
	return html.Div(html.Props{Class: "gwc-workbench", Role: "list"}, parseSections...)
}

//go:build js && wasm

// Command use-head is the runnable e2e fixture for head.UseHead: it applies a Document's title and
// head tags (description, canonical, OpenGraph) to the live document on mount, and changes them
// when a button switches the logical page — exercised by the playwright e2e to prove client-side
// head updates (not just title) without a full reload.
package main

import (
	"github.com/monstercameron/GoWebComponents/v4/head"
	"github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/router"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

func App() ui.Node {
	parsePage := ui.UseState("home")

	parseDoc := head.Document{
		Metadata: router.Metadata{
			Title:        "Home Title",
			Description:  "home description",
			CanonicalURL: "https://example.com/home",
		},
		Social: head.SocialMetadata{Type: "website", Title: "Home OG Title"},
	}
	if parsePage.Get() == "about" {
		parseDoc.Metadata.Title = "About Title"
		parseDoc.Metadata.Description = "about description"
	}
	head.UseHead(parseDoc)

	parseGoAbout := ui.UseEvent(func(ui.MouseEvent) { parsePage.Set("about") })
	return html.Div(html.Props{ID: "app-root"},
		html.Button(html.Props{ID: "to-about", OnClick: parseGoAbout}, html.Text("about")),
		html.P(html.Props{ID: "page"}, html.Text(parsePage.Get())),
	)
}

func main() { ui.Run("#app", App) }

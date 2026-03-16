//go:build js && wasm
// +build js,wasm

package main

import (
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

func currentMeta(name string) string {
	document := js.Global().Get("document")
	if !document.Truthy() {
		return ""
	}
	meta := document.Call("querySelector", `meta[name="`+name+`"]`)
	if !meta.Truthy() {
		return ""
	}
	return meta.Call("getAttribute", "content").String()
}

func currentCanonical() string {
	document := js.Global().Get("document")
	if !document.Truthy() {
		return ""
	}
	link := document.Call("querySelector", `link[rel="canonical"]`)
	if !link.Truthy() {
		return ""
	}
	return link.Call("getAttribute", "href").String()
}

func metadataPage(title, summary string) *router.Element {
	nav := router.UseNavigate()
	documentTitle := js.Global().Get("document").Get("title").String()
	return shared.ExamplePage(
		title,
		"Route metadata options",
		summary,
		shared.ExamplePanel("Managed head tags",
			html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-3"},
				shared.ExampleButton("Docs route", ui.UseEvent(func() { nav.Navigate("/docs") })),
				shared.ExampleButton("Pricing route", ui.UseEvent(func() { nav.Navigate("/pricing") })),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-3"},
				shared.ExampleStat("document.title", documentTitle),
				shared.ExampleStat("meta description", currentMeta("description")),
				shared.ExampleStat("canonical", currentCanonical()),
			),
			shared.ExampleCode(
				`router.Options{Title: "Docs", Description: "Guides", CanonicalURL: "https://example.com/docs"}`,
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	r := router.NewHashRouter(router.RouterOptions{DefaultRoute: "/docs"})
	r.Register("/docs", func(router.Attrs) *router.Element {
		return metadataPage("Router metadata", "This route manages title, description, and canonical URL from route options.")
	}, router.Options{Title: "Docs", Description: "Framework guides and API documentation", CanonicalURL: "https://example.com/docs"})
	r.Register("/pricing", func(router.Attrs) *router.Element {
		return metadataPage("Pricing metadata", "Navigating between routes updates or replaces the managed head tags." )
	}, router.Options{Title: "Pricing", Description: "Plan comparison and pricing", CanonicalURL: "https://example.com/pricing"})
	r.Mount("#app")
	select {}
}
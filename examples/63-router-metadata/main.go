//go:build js && wasm
// +build js,wasm

package main

import (
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

func currentMeta(parseName string) string {
	parseDocument := js.Global().Get("document")
	if !parseDocument.Truthy() {
		return ""
	}
	parseMeta := parseDocument.Call("querySelector", `meta[name="`+parseName+`"]`)
	if !parseMeta.Truthy() {
		return ""
	}
	return parseMeta.Call("getAttribute", "content").String()
}

func currentCanonical() string {
	parseDocument := js.Global().Get("document")
	if !parseDocument.Truthy() {
		return ""
	}
	parseLink := parseDocument.Call("querySelector", `link[rel="canonical"]`)
	if !parseLink.Truthy() {
		return ""
	}
	return parseLink.Call("getAttribute", "href").String()
}

type metadataPageProps struct {
	Title   string
	Summary string
}

func metadataPageView(parseProps metadataPageProps) ui.Node {
	parseNav := router.UseNavigate()
	parseDocumentTitle := js.Global().Get("document").Get("title").String()
	return shared.ExamplePage(
		parseProps.Title,
		"Route metadata options",
		parseProps.Summary,
		shared.ExamplePanel("Managed head tags",
			html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-3"},
				shared.ExampleButton("Docs route", ui.UseEvent(func() { parseNav.Navigate("/docs") })),
				shared.ExampleButton("Pricing route", ui.UseEvent(func() { parseNav.Navigate("/pricing") })),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-3"},
				shared.ExampleStat("document.title", parseDocumentTitle),
				shared.ExampleStat("meta description", currentMeta("description")),
				shared.ExampleStat("canonical", currentCanonical()),
			),
			shared.ExampleCode(
				`router.Options{Title: "Docs", Description: "Guides", CanonicalURL: "https://example.com/docs"}`,
			),
		),
	)
}

func metadataPage(parseTitle, parseSummary string) *router.Element {
	return ui.CreateElement(metadataPageView, metadataPageProps{Title: parseTitle, Summary: parseSummary})
}

func main() {
	utils.DisableAllDebug()
	parseR := router.NewHashRouter(router.RouterOptions{DefaultRoute: "/docs"})
	parseR.Register("/docs", func(router.Attrs) *router.Element {
		return metadataPage("Router metadata", "This route manages title, description, and canonical URL from route options.")
	}, router.Options{Title: "Docs", Description: "Framework guides and API documentation", CanonicalURL: "https://example.com/docs"})
	parseR.Register("/pricing", func(router.Attrs) *router.Element {
		return metadataPage("Pricing metadata", "Navigating between routes updates or replaces the managed head tags.")
	}, router.Options{Title: "Pricing", Description: "Plan comparison and pricing", CanonicalURL: "https://example.com/pricing"})
	parseR.Mount("#app")
	select {}
}

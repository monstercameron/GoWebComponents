//go:build js && wasm
// +build js,wasm

// The GoWebComponents docs site, written entirely in Go as a GWC application.
// There are no authored .html or .js files: routing, search, syntax
// highlighting, and styling all live in this wasm module. The boot shell is
// generated at build time by tools/sitegen.
package main

import (
	"strings"

	. "github.com/monstercameron/GoWebComponents/v6/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
	"github.com/monstercameron/GoWebComponents/v6/utils"
)

// siteChapters is loaded once from the embedded manual data.
var siteChapters = loadChapters()

// activeNavKey derives the highlighted nav section from the current route.
func activeNavKey() string {
	parsePath := router.InspectCurrentRoute().Path
	switch {
	case strings.HasPrefix(parsePath, "/learn"), strings.HasPrefix(parsePath, "/concepts"):
		return "learn"
	case strings.HasPrefix(parsePath, "/examples"), strings.HasPrefix(parsePath, "/source"):
		return "examples"
	case strings.HasPrefix(parsePath, "/api"):
		return "api"
	default:
		return "home"
	}
}

// renderShell wraps routed content in the site chrome. The stylesheet rides
// along as a <style> node so no .css file needs to be served either.
func renderShell(parseContent ui.Node) ui.Node {
	parseSearchOpen := ui.UseState(false)
	parseActive := activeNavKey()

	parseNavLink := func(parseLabel string, parseHref string, parseKey string) ui.Node {
		parseAttrs := []interface{}{Href(parseHref), Text(parseLabel)}
		if parseActive == parseKey {
			parseAttrs = append(parseAttrs, Attr("data-active", "true"))
		}
		return A(parseAttrs...)
	}

	return Div(
		Tag("style", Text(siteCSS)),
		Header(ClassStr("site-header"),
			Div(ClassStr("site-header-inner"),
				A(ClassStr("site-logo"), Href("#/"),
					Span(ClassStr("logo-accent"), Text("Go")), Text("WebComponents")),
				Nav(ClassStr("site-nav"),
					parseNavLink("Learn", "#/learn", "learn"),
					parseNavLink("Examples", "#/examples", "examples"),
					parseNavLink("API", "#/api", "api"),
					A(Href("https://github.com/monstercameron/GoWebComponents"), Text("GitHub")),
					Button(ClassStr("search-button"), Attr("type", "button"),
						OnClick(func() { parseSearchOpen.Set(true) }),
						Text("Search"), Tag("kbd", Text("/"))),
				),
			),
		),
		parseContent,
		Footer(ClassStr("site-footer"),
			Div(ClassStr("site-footer-inner"),
				Span(Text("GoWebComponents — Go-native UIs for the browser. MIT licensed. This site is a GWC app: every page is Go.")),
				Span(
					A(Href("https://github.com/monstercameron/GoWebComponents"), Text("GitHub")),
					Text("  ·  "),
					A(Href("public-examples-site/index.html"), Text("Interactive catalog")),
				),
			),
		),
		renderSearchModal(parseSearchOpen),
	)
}

// routePage adapts a page component into a routed, shell-wrapped element.
func routePage(parseRender func(router.Attrs) ui.Node) func(router.Attrs) *router.Element {
	return func(parseAttrs router.Attrs) *router.Element {
		return ui.CreateElement(func(parseProps map[string]interface{}) ui.Node {
			return renderShell(ui.CreateElement(func(map[string]interface{}) ui.Node {
				return parseRender(parseAttrs)
			}, nil))
		}, nil)
	}
}

func main() {
	utils.DisableAllDebug()

	parseR := router.NewHashRouter(router.RouterOptions{DefaultRoute: "/"})
	parseR.Register("/", routePage(func(router.Attrs) ui.Node { return renderLandingPage() }),
		router.Options{Title: "GoWebComponents — Go-native UIs for the browser"})
	parseR.Register("/learn", routePage(func(router.Attrs) ui.Node { return renderLearnIndex() }),
		router.Options{Title: "Learn — GoWebComponents"})
	parseR.Register("/learn/:slug", routePage(func(router.Attrs) ui.Node {
		return renderChapterPage(router.UseParams().Get("slug"))
	}), router.Options{Title: "Manual — GoWebComponents"})
	parseR.Register("/concepts/:slug", routePage(func(router.Attrs) ui.Node {
		return renderConceptPage(router.UseParams().Get("slug"))
	}), router.Options{Title: "Concepts — GoWebComponents"})
	parseR.Register("/examples", routePage(func(router.Attrs) ui.Node { return renderExamplesGallery() }),
		router.Options{Title: "Examples — GoWebComponents"})
	parseR.Register("/source/:slug", routePage(func(router.Attrs) ui.Node {
		return renderSourceViewer(router.UseParams().Get("slug"))
	}), router.Options{Title: "Source — GoWebComponents"})
	parseR.Register("/api", routePage(func(router.Attrs) ui.Node { return renderAPIIndex() }),
		router.Options{Title: "API — GoWebComponents"})
	parseR.Register("*", routePage(func(router.Attrs) ui.Node { return renderNotFound() }),
		router.Options{Title: "Not found — GoWebComponents"})

	parseR.Mount("#app")
	utils.WaitForever()
}

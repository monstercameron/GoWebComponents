package main

import (
	"strings"

	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/ui"
)

// pageChrome describes one rendered page's placement inside the site shell.
type pageChrome struct {
	Title       string
	Description string
	// RootPrefix is the relative path from this page back to the site root
	// (e.g. "." for /, ".." for /learn/, "../.." for /learn/x/).
	RootPrefix string
	ActiveNav  string
}

// asset returns a root-relative asset URL for the current page depth.
func (parseChrome pageChrome) asset(parsePath string) string {
	return parseChrome.RootPrefix + "/" + strings.TrimPrefix(parsePath, "/")
}

// renderPage wraps page content in the shared document chrome.
func renderPage(parseChrome pageChrome, parseContent ...ui.Node) ui.Node {
	parseNavLink := func(parseLabel string, parseHref string, parseKey string) ui.Node {
		parseAttrs := []interface{}{Href(parseChrome.asset(parseHref)), Text(parseLabel)}
		if parseChrome.ActiveNav == parseKey {
			parseAttrs = append(parseAttrs, Attr("data-active", "true"))
		}
		return A(parseAttrs...)
	}

	parseBody := []interface{}{
		Attr("data-root", parseChrome.RootPrefix),
		Header(Class("site-header"),
			Div(Class("site-header-inner"),
				A(Class("site-logo"), Href(parseChrome.asset("index.html")),
					Span(Class("logo-accent"), Text("Go")), Text("WebComponents")),
				Nav(Class("site-nav"),
					parseNavLink("Learn", "learn/index.html", "learn"),
					parseNavLink("Examples", "examples/index.html", "examples"),
					parseNavLink("API", "api/index.html", "api"),
					A(Href("https://github.com/monstercameron/GoWebComponents"), Text("GitHub")),
					Button(Class("search-button"), Attr("type", "button"), Attr("data-search-open", "true"),
						Text("Search"), Tag("kbd", Text("Ctrl K"))),
				),
			),
		),
	}
	for _, parseNode := range parseContent {
		parseBody = append(parseBody, parseNode)
	}
	parseBody = append(parseBody,
		Footer(Class("site-footer"),
			Div(Class("site-footer-inner"),
				Span(Text("GoWebComponents — Go-native UIs for the browser. MIT licensed.")),
				Span(
					A(Href("https://github.com/monstercameron/GoWebComponents"), Text("GitHub")),
					Text("  ·  "),
					A(Href(parseChrome.asset("public-examples-site/index.html")), Text("Interactive catalog")),
				),
			),
		),
		renderSearchModal(),
		Script(Src(parseChrome.asset("site/site.js")), Attr("defer", "defer")),
	)

	return Html(Attr("lang", "en"),
		Head(
			Meta(Attr("charset", "utf-8")),
			Meta(Attr("name", "viewport"), Attr("content", "width=device-width, initial-scale=1")),
			Meta(Attr("name", "description"), Attr("content", parseChrome.Description)),
			Tag("title", Text(parseChrome.Title)),
			Tag("link", Attr("rel", "stylesheet"), Href(parseChrome.asset("site/site.css"))),
			Tag("link", Attr("rel", "icon"), Attr("type", "image/x-icon"), Href(parseChrome.asset("static/images/favicon/favicon.ico"))),
		),
		Body(parseBody...),
	)
}

// renderSearchModal emits the hidden command-K search dialog markup.
func renderSearchModal() ui.Node {
	return Div(ID("search-overlay"), Class("search-overlay"),
		Div(Class("search-panel"),
			Input(ID("search-input"), Class("search-input"),
				Attr("type", "search"), Attr("placeholder", "Search docs, examples, APIs..."),
				Attr("autocomplete", "off"), Attr("spellcheck", "false")),
			Div(ID("search-results"), Class("search-results")),
		),
	)
}

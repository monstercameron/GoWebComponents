package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

type guideArticle struct {
	ID         string
	Title      string
	Summary    string
	Highlights []string
}

type demoShellView struct {
	Mode          string
	ActivePath    string
	BootstrapPath string
	Transport     string
	SectionID     string
	SectionTitle  string
	SectionBody   string
	CurrentTab    string
	LoadRevision  int
	SearchQuery   string
	SearchResults []guideArticle
	SecureRole    string
	SecureUser    string
	Notice        string
}

var guideCatalog = map[string]guideArticle{
	guideSectionSSR: {
		ID:      guideSectionSSR,
		Title:   "SSR transport and hydration",
		Summary: "Server-rendered HTML shell with a bootstrap sidecar that the wasm client restores before reusing matching DOM during hydration.",
		Highlights: []string{
			"Static HTML ships with real route content before wasm starts.",
			"A JSON sidecar bootstrap payload is fetched before hydration.",
			"Hydration restores the bootstrap payload, reuses matching DOM, and falls back per subtree only when structure no longer matches.",
		},
	},
	guideSectionRouting: {
		ID:      guideSectionRouting,
		Title:   "Advanced route loaders and redirects",
		Summary: "Route patterns, query-aware loaders, redirect routes, guard checks, and manual revalidation all run through the same client router.",
		Highlights: []string{
			"/docs/:section uses params plus query-aware loader revalidation.",
			"/legacy redirects into the docs route without leaving a stale entry.",
			"/secure uses a before-enter redirect and a protected loader result.",
		},
	},
	guideSectionBenchmarks: {
		ID:      guideSectionBenchmarks,
		Title:   "Microbenchmarks for SSR delivery",
		Summary: "The demo keeps native microbenchmarks around render-to-string, bootstrap marshaling, and bootstrap reference script generation.",
		Highlights: []string{
			"RenderToString benchmarks measure the demo shell as a public SSR surface.",
			"Bootstrap helpers are measured separately for transport overhead.",
			"The same shared view model is used by the tests and benchmarks.",
		},
	},
}

func defaultBootstrapPayload() ui.SSRBootstrap {
	return ui.SSRBootstrap{
		Route: ui.SSRRouteBootstrap{
			Path:   "/docs/" + guideSectionSSR,
			Params: map[string]string{"section": guideSectionSSR},
		},
		Data: map[string]interface{}{
			"transport": transportJSONSidecar,
			"demo":      ssrRoutingDemoName,
		},
		IDSeed: 17,
	}
}

func defaultServerView() demoShellView {
	parseArticle := articleForSection(guideSectionSSR)
	parsePayload := defaultBootstrapPayload()
	return demoShellView{
		Mode:          modeDocs,
		ActivePath:    parsePayload.Route.Path,
		BootstrapPath: parsePayload.Route.Path,
		Transport:     bootstrapTransport(parsePayload),
		SectionID:     parseArticle.ID,
		SectionTitle:  parseArticle.Title,
		SectionBody:   parseArticle.Summary,
		CurrentTab:    tabOverview,
		LoadRevision:  1,
		Notice:        "Restores a sidecar bootstrap payload, reuses matching server DOM, and falls back per subtree only when hydration cannot continue safely.",
		SearchResults: catalogList(),
	}
}

func articleForSection(parseSection string) guideArticle {
	if parseArticle, parseOk := guideCatalog[parseSection]; parseOk {
		return parseArticle
	}
	return guideCatalog[guideSectionSSR]
}

func catalogList() []guideArticle {
	parseKeys := make([]string, 0, len(guideCatalog))
	for parseKey := range guideCatalog {
		parseKeys = append(parseKeys, parseKey)
	}
	sort.Strings(parseKeys)
	parseArticles := make([]guideArticle, 0, len(parseKeys))
	for _, parseKey2 := range parseKeys {
		parseArticles = append(parseArticles, guideCatalog[parseKey2])
	}
	return parseArticles
}

func bootstrapTransport(parsePayload ui.SSRBootstrap) string {
	if parsePayload.Data == nil {
		return transportJSONSidecar
	}
	if parseValue, parseOk := parsePayload.Data["transport"].(string); parseOk && parseValue != "" {
		return parseValue
	}
	return transportJSONSidecar
}

func renderDemoShell(parseView demoShellView, parseActions ...ui.Node) ui.Node {
	parsePage := renderPage(parseView, parseActions...)

	return html.Div(html.Props{Class: "min-h-screen bg-[#06131f] text-slate-100"},
		html.Div(html.Props{Class: "mx-auto max-w-6xl px-6 py-10"},
			html.Div(html.Props{Class: "rounded-[2rem] border border-white/10 bg-[radial-gradient(circle_at_top,_rgba(56,189,248,0.14),_transparent_42%),rgba(15,23,42,0.92)] p-8 shadow-2xl"},
				html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("SSR Routing Demo")),
				html.H1(html.Props{Class: "mt-4 text-5xl font-black tracking-tight text-white"}, html.Text("Server render first, hydrate into advanced routes")),
				html.P(html.Props{Class: "mt-4 max-w-3xl text-lg leading-8 text-slate-300"}, html.Text("This demo serves a real SSR shell, restores a bootstrap sidecar, reuses matching DOM during hydration, and then continues as a hash-router app with params, redirects, guards, loaders, query state, and manual revalidation.")),
				html.Div(html.Props{Class: "mt-8 flex flex-wrap gap-3"},
					navLink("Overview", routeHomeHash, parseView.ActivePath == "/"),
					navLink("Docs", routeDocsSSRHash, strings.HasPrefix(parseView.ActivePath, "/docs")),
					navLink("Search", routeSearchHash, strings.HasPrefix(parseView.ActivePath, "/search")),
					navLink("Protected", routeSecureHash, strings.HasPrefix(parseView.ActivePath, "/secure")),
					navLink("Legacy Redirect", routeLegacyHash, strings.HasPrefix(parseView.ActivePath, "/legacy")),
					navLink("Sign In", routeSignInHash, strings.HasPrefix(parseView.ActivePath, "/signin")),
				),
				html.Div(html.Props{Class: "mt-8 grid gap-4 md:grid-cols-3"},
					statCard("Bootstrap route", emptyFallback(parseView.BootstrapPath, "/docs/"+guideSectionSSR)),
					statCard("Transport", emptyFallback(parseView.Transport, transportJSONSidecar)),
					statCard("Revision", fmt.Sprintf("%d", parseView.LoadRevision)),
				),
			),
			html.Div(html.Props{Class: "mt-8"}, parsePage),
		),
	)
}

func renderPage(parseView demoShellView, parseActions ...ui.Node) ui.Node {
	switch parseView.Mode {
	case "home":
		return renderHomePage(parseView)
	case "search":
		return renderSearchPage(parseView, parseActions...)
	case "signin":
		return renderSignInPage(parseView, parseActions...)
	case "secure":
		return renderSecurePage(parseView, parseActions...)
	default:
		return renderDocsPage(parseView, parseActions...)
	}
}

func renderHomePage(_ demoShellView) ui.Node {
	parseArticles := catalogList()
	parseCards := make([]ui.Node, 0, len(parseArticles))
	for _, parseArticle := range parseArticles {
		parseArticle2 := parseArticle
		parseCards = append(parseCards, html.Article(html.Props{Class: "rounded-[1.5rem] border border-white/10 bg-white/5 p-6"},
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-cyan-300"}, html.Text(strings.ToUpper(parseArticle2.ID))),
			html.H2(html.Props{Class: "mt-3 text-2xl font-bold text-white"}, html.Text(parseArticle2.Title)),
			html.P(html.Props{Class: "mt-3 text-sm leading-7 text-slate-300"}, html.Text(parseArticle2.Summary)),
			html.A(html.Props{Href: "#/docs/" + parseArticle2.ID + "?tab=overview", Class: "mt-5 inline-flex rounded-full border border-cyan-900/80 bg-cyan-950/70 px-4 py-2 text-sm font-semibold text-cyan-100 hover:bg-cyan-900/80"}, html.Text("Open route")),
		))
	}

	return html.Div(html.Props{Class: "grid gap-6 md:grid-cols-3"}, parseCards...)
}

func renderDocsPage(parseView demoShellView, parseActions ...ui.Node) ui.Node {
	parseArticle := articleForSection(parseView.SectionID)
	parseActionNodes := make([]ui.Node, 0, len(parseActions)+3)
	parseActionNodes = append(parseActionNodes,
		navLink("Overview tab", routeDocsHash+parseArticle.ID+"?tab="+tabOverview, parseView.CurrentTab == tabOverview),
		navLink("Loader tab", routeDocsHash+parseArticle.ID+"?tab="+tabLoader, parseView.CurrentTab == tabLoader),
		navLink("Bench tab", routeDocsHash+parseArticle.ID+"?tab="+tabBench, parseView.CurrentTab == tabBench),
	)
	parseActionNodes = append(parseActionNodes, parseActions...)

	parseHighlights := make([]ui.Node, 0, len(parseArticle.Highlights))
	for _, parseItem := range parseArticle.Highlights {
		parseHighlights = append(parseHighlights, html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-white/5 p-4 text-slate-200"}, html.Text(parseItem)))
	}

	return html.Div(html.Props{Class: "space-y-8"},
		html.Section(html.Props{Class: "rounded-[1.75rem] border border-white/10 bg-white/5 p-8"},
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("Docs route")),
			html.H2(html.Props{Class: "mt-3 text-4xl font-black text-white"}, html.Text(parseView.SectionTitle)),
			html.P(html.Props{Class: "mt-4 max-w-3xl text-lg leading-8 text-slate-300"}, html.Text(parseView.SectionBody)),
			html.P(html.Props{Class: "mt-4 text-sm uppercase tracking-[0.28em] text-slate-400"}, html.Text("Current tab: "+emptyFallback(parseView.CurrentTab, tabOverview))),
			html.P(html.Props{Class: "mt-2 text-sm text-slate-500"}, html.Text(emptyFallback(parseView.Notice, "Route loader data is keyed by params and query state."))),
			html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"}, parseActionNodes...),
		),
		html.Section(html.Props{Class: "grid gap-4 md:grid-cols-3"}, parseHighlights...),
		deferredRouteInsights(parseView),
	)
}

func renderSearchPage(parseView demoShellView, parseActions ...ui.Node) ui.Node {
	parseResults := make([]ui.Node, 0, len(parseView.SearchResults))
	for _, parseArticle := range parseView.SearchResults {
		parseArticle2 := parseArticle
		parseResults = append(parseResults, html.Article(html.Props{Class: "rounded-[1.5rem] border border-white/10 bg-white/5 p-6"},
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-cyan-300"}, html.Text(strings.ToUpper(parseArticle2.ID))),
			html.H3(html.Props{Class: "mt-3 text-2xl font-bold text-white"}, html.Text(parseArticle2.Title)),
			html.P(html.Props{Class: "mt-3 text-sm leading-7 text-slate-300"}, html.Text(parseArticle2.Summary)),
			html.A(html.Props{Href: "#/docs/" + parseArticle2.ID + "?tab=overview", Class: "mt-5 inline-flex rounded-full border border-cyan-900/80 bg-cyan-950/70 px-4 py-2 text-sm font-semibold text-cyan-100 hover:bg-cyan-900/80"}, html.Text("Open detail")),
		))
	}
	if len(parseResults) == 0 {
		parseResults = append(parseResults, html.Div(html.Props{Class: "rounded-[1.5rem] border border-dashed border-white/10 bg-white/5 p-6 text-slate-300"}, html.Text("No search results matched the current query.")))
	}

	return html.Div(html.Props{Class: "space-y-8"},
		html.Section(html.Props{Class: "rounded-[1.75rem] border border-white/10 bg-white/5 p-8"},
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("Search route")),
			html.H2(html.Props{Class: "mt-3 text-4xl font-black text-white"}, html.Text("Query-driven loader results")),
			html.P(html.Props{Class: "mt-4 max-w-3xl text-lg leading-8 text-slate-300"}, html.Text("Use the route buttons to replace query state and rerun the search loader without leaving the routed shell.")),
			html.P(html.Props{Class: "mt-4 text-sm uppercase tracking-[0.28em] text-slate-400"}, html.Text("Current query: "+emptyFallback(parseView.SearchQuery, "none"))),
			html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
				navLink("All", routeSearchHash, parseView.SearchQuery == ""),
				navLink("SSR", routeSearchHash+"?q="+searchQuerySSR, parseView.SearchQuery == searchQuerySSR),
				navLink("Routing", routeSearchHash+"?q="+searchQueryRouting, parseView.SearchQuery == searchQueryRouting),
				navLink("Bench", routeSearchHash+"?q="+searchQueryBench, parseView.SearchQuery == searchQueryBench),
			),
			func() ui.Node {
				if len(parseActions) == 0 {
					return nil
				}
				return html.Div(html.Props{Class: "mt-4 flex flex-wrap gap-3"}, parseActions...)
			}(),
			deferredRouteInsights(parseView),
		),
		html.Section(html.Props{Class: "grid gap-4 md:grid-cols-2"}, parseResults...),
	)
}

func renderSignInPage(parseView demoShellView, parseActions ...ui.Node) ui.Node {
	parseActionNodes := make([]ui.Node, 0, len(parseActions)+2)
	parseActionNodes = append(parseActionNodes,
		navLink("Grant Access", routeSecureHash+"?auth=true&role="+secureRoleMaintainer, false),
		navLink("Open overview", routeHomeHash, false),
	)
	parseActionNodes = append(parseActionNodes, parseActions...)

	return html.Div(html.Props{Class: "rounded-[1.75rem] border border-white/10 bg-white/5 p-8"},
		html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("Guard redirect")),
		html.H2(html.Props{Class: "mt-3 text-4xl font-black text-white"}, html.Text("Protected routes can redirect before rendering")),
		html.P(html.Props{Class: "mt-4 max-w-3xl text-lg leading-8 text-slate-300"}, html.Text(emptyFallback(parseView.Notice, "The protected route redirected here because auth=true was not present in the query string."))),
		html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"}, parseActionNodes...),
	)
}

func renderSecurePage(parseView demoShellView, parseActions ...ui.Node) ui.Node {
	parseActionNodes := make([]ui.Node, 0, len(parseActions)+2)
	parseActionNodes = append(parseActionNodes,
		navLink("Switch Role", routeSecureHash+"?auth=true&role=auditor", false),
		navLink("Revoke", routeSignInHash, false),
	)
	parseActionNodes = append(parseActionNodes, parseActions...)

	return html.Div(html.Props{Class: "space-y-8"},
		html.Section(html.Props{Class: "rounded-[1.75rem] border border-white/10 bg-white/5 p-8"},
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("Protected route")),
			html.H2(html.Props{Class: "mt-3 text-4xl font-black text-white"}, html.Text("Guarded content unlocked")),
			html.P(html.Props{Class: "mt-4 max-w-3xl text-lg leading-8 text-slate-300"}, html.Text(emptyFallback(parseView.Notice, "The before-enter guard allowed this route and the loader produced a protected payload."))),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-3"},
				statCard("User", emptyFallback(parseView.SecureUser, "Morgan Reconciler")),
				statCard("Role", emptyFallback(parseView.SecureRole, secureRoleMaintainer)),
				statCard("Loader revision", fmt.Sprintf("%d", parseView.LoadRevision)),
			),
			html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"}, parseActionNodes...),
		),
		deferredRouteInsights(parseView),
	)
}

func deferredRouteInsights(parseView demoShellView) ui.Node {
	parseLoaderKey := deferredRouteKey(parseView)

	return ui.CreateElement(ui.Lazy, ui.LazyProps{
		Loader: func(parseCtx context.Context) (ui.Node, error) {
			select {
			case <-parseCtx.Done():
				return nil, parseCtx.Err()
			case <-time.After(140 * time.Millisecond):
			}

			return html.Section(html.Props{Class: "rounded-[1.75rem] border border-cyan-500/20 bg-cyan-500/5 p-6"},
				html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("Deferred route insights")),
				html.H3(html.Props{Class: "mt-3 text-2xl font-black text-white"}, html.Text("Nested async panel ready")),
				html.P(html.Props{Class: "mt-3 text-sm leading-7 text-slate-300"}, html.Text("This panel resolves through ui.Lazy after the route loader has already produced the shell, showing nested async UI inside a loader-backed route.")),
				html.P(html.Props{Class: "mt-3 text-xs uppercase tracking-[0.28em] text-slate-400"}, html.Text("Loader key: "+parseLoaderKey)),
			), nil
		},
		Dependencies: []interface{}{parseLoaderKey},
		Delay:        50 * time.Millisecond,
		Timeout:      500 * time.Millisecond,
		Fallback: html.Section(html.Props{Class: "rounded-[1.75rem] border border-white/10 bg-white/5 p-6"},
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("Deferred route insights")),
			html.P(html.Props{Class: "mt-3 text-sm text-slate-300"}, html.Text("Preparing nested async panel...")),
		),
		TimeoutFallback: html.Section(html.Props{Class: "rounded-[1.75rem] border border-amber-400/20 bg-amber-400/10 p-6"},
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-amber-300"}, html.Text("Deferred route insights")),
			html.P(html.Props{Class: "mt-3 text-sm text-amber-100"}, html.Text("Still waiting on nested async route content...")),
		),
		ErrorFallback: func(parseErr error) ui.Node {
			return html.Section(html.Props{Class: "rounded-[1.75rem] border border-rose-400/20 bg-rose-400/10 p-6"},
				html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-rose-300"}, html.Text("Deferred route insights")),
				html.P(html.Props{Class: "mt-3 text-sm text-rose-100"}, html.Text("Deferred route panel failed: "+parseErr.Error())),
			)
		},
	})
}

func deferredRouteKey(parseView demoShellView) string {
	switch parseView.Mode {
	case "docs":
		return fmt.Sprintf("docs:%s:%s:%d", emptyFallback(parseView.SectionID, guideSectionSSR), emptyFallback(parseView.CurrentTab, tabOverview), parseView.LoadRevision)
	case "search":
		return fmt.Sprintf("search:%s:%d", emptyFallback(parseView.SearchQuery, searchQueryAll), parseView.LoadRevision)
	case "secure":
		return fmt.Sprintf("secure:%s:%s:%d", emptyFallback(parseView.SecureUser, secureGuestUser), emptyFallback(parseView.SecureRole, secureRoleNone), parseView.LoadRevision)
	default:
		return fmt.Sprintf("%s:%d", emptyFallback(parseView.Mode, defaultRouteBucket), parseView.LoadRevision)
	}
}

func navLink(parseLabel string, parseHref string, isActive bool) ui.Node {
	parseClassName := "inline-flex rounded-full border px-4 py-2 text-sm font-semibold transition-colors "
	if isActive {
		parseClassName += "border-cyan-900/90 bg-cyan-950/70 text-cyan-100"
	} else {
		parseClassName += "border-white/10 bg-white/5 text-slate-200 hover:bg-white/10"
	}
	return html.A(html.Props{Href: parseHref, Class: parseClassName}, html.Text(parseLabel))
}

func statCard(parseLabel string, parseValue string) ui.Node {
	return html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-white/5 p-4"},
		html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text(parseLabel)),
		html.P(html.Props{Class: "mt-3 text-xl font-bold text-white"}, html.Text(parseValue)),
	)
}

func emptyFallback(parseValue string, parseFallback string) string {
	if strings.TrimSpace(parseValue) == "" {
		return parseFallback
	}
	return parseValue
}

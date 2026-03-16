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
	"ssr": {
		ID:      "ssr",
		Title:   "SSR transport and hydration",
		Summary: "Server-rendered HTML shell with a bootstrap sidecar that the wasm client reads before hydration.",
		Highlights: []string{
			"Static HTML ships with real route content before wasm starts.",
			"A JSON sidecar bootstrap payload is fetched before hydration.",
			"Hydration restores the bootstrap payload, reuses matching DOM, and falls back per subtree only when structure no longer matches.",
		},
	},
	"routing": {
		ID:      "routing",
		Title:   "Advanced route loaders and redirects",
		Summary: "Route patterns, query-aware loaders, redirect routes, guard checks, and manual revalidation all run through the same client router.",
		Highlights: []string{
			"/docs/:section uses params plus query-aware loader revalidation.",
			"/legacy redirects into the docs route without leaving a stale entry.",
			"/secure uses a before-enter redirect and a protected loader result.",
		},
	},
	"benchmarks": {
		ID:      "benchmarks",
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
			Path:   "/docs/ssr",
			Params: map[string]string{"section": "ssr"},
		},
		Data: map[string]interface{}{
			"transport": "json-sidecar",
			"demo":      "ssr-routing",
		},
		IDSeed: 17,
	}
}

func defaultServerView() demoShellView {
	article := articleForSection("ssr")
	payload := defaultBootstrapPayload()
	return demoShellView{
		Mode:          "docs",
		ActivePath:    payload.Route.Path,
		BootstrapPath: payload.Route.Path,
		Transport:     bootstrapTransport(payload),
		SectionID:     article.ID,
		SectionTitle:  article.Title,
		SectionBody:   article.Summary,
		CurrentTab:    "overview",
		LoadRevision:  1,
		Notice:        "Restores a sidecar bootstrap payload, reuses matching server DOM, and falls back per subtree only when hydration cannot continue safely.",
		SearchResults: catalogList(),
	}
}

func articleForSection(section string) guideArticle {
	if article, ok := guideCatalog[section]; ok {
		return article
	}
	return guideCatalog["ssr"]
}

func catalogList() []guideArticle {
	keys := make([]string, 0, len(guideCatalog))
	for key := range guideCatalog {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	articles := make([]guideArticle, 0, len(keys))
	for _, key := range keys {
		articles = append(articles, guideCatalog[key])
	}
	return articles
}

func filterCatalog(query string) []guideArticle {
	trimmed := strings.TrimSpace(strings.ToLower(query))
	if trimmed == "" {
		return catalogList()
	}

	matches := make([]guideArticle, 0, len(guideCatalog))
	for _, article := range catalogList() {
		if strings.Contains(strings.ToLower(article.Title), trimmed) || strings.Contains(strings.ToLower(article.Summary), trimmed) {
			matches = append(matches, article)
		}
	}
	return matches
}

func bootstrapTransport(payload ui.SSRBootstrap) string {
	if payload.Data == nil {
		return "json-sidecar"
	}
	if value, ok := payload.Data["transport"].(string); ok && value != "" {
		return value
	}
	return "json-sidecar"
}

func renderDemoShell(view demoShellView, actions ...ui.Node) ui.Node {
	page := renderPage(view, actions...)

	return html.Div(html.Props{Class: "min-h-screen bg-[#06131f] text-slate-100"},
		html.Div(html.Props{Class: "mx-auto max-w-6xl px-6 py-10"},
			html.Div(html.Props{Class: "rounded-[2rem] border border-white/10 bg-[radial-gradient(circle_at_top,_rgba(56,189,248,0.14),_transparent_42%),rgba(15,23,42,0.92)] p-8 shadow-2xl"},
				html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("SSR Routing Demo")),
				html.H1(html.Props{Class: "mt-4 text-5xl font-black tracking-tight text-white"}, html.Text("Server render first, hydrate into advanced routes")),
				html.P(html.Props{Class: "mt-4 max-w-3xl text-lg leading-8 text-slate-300"}, html.Text("This demo serves a real SSR shell, restores a bootstrap sidecar, reuses matching DOM during hydration, and then continues as a hash-router app with params, redirects, guards, loaders, query state, and manual revalidation.")),
				html.Div(html.Props{Class: "mt-8 flex flex-wrap gap-3"},
					navLink("Overview", "#/", view.ActivePath == "/"),
					navLink("Docs", "#/docs/ssr", strings.HasPrefix(view.ActivePath, "/docs")),
					navLink("Search", "#/search", strings.HasPrefix(view.ActivePath, "/search")),
					navLink("Protected", "#/secure", strings.HasPrefix(view.ActivePath, "/secure")),
					navLink("Legacy Redirect", "#/legacy", strings.HasPrefix(view.ActivePath, "/legacy")),
					navLink("Sign In", "#/signin", strings.HasPrefix(view.ActivePath, "/signin")),
				),
				html.Div(html.Props{Class: "mt-8 grid gap-4 md:grid-cols-3"},
					statCard("Bootstrap route", emptyFallback(view.BootstrapPath, "/docs/ssr")),
					statCard("Transport", emptyFallback(view.Transport, "json-sidecar")),
					statCard("Revision", fmt.Sprintf("%d", view.LoadRevision)),
				),
			),
			html.Div(html.Props{Class: "mt-8"}, page),
		),
	)
}

func renderPage(view demoShellView, actions ...ui.Node) ui.Node {
	switch view.Mode {
	case "home":
		return renderHomePage(view)
	case "search":
		return renderSearchPage(view, actions...)
	case "signin":
		return renderSignInPage(view, actions...)
	case "secure":
		return renderSecurePage(view, actions...)
	default:
		return renderDocsPage(view, actions...)
	}
}

func renderHomePage(view demoShellView) ui.Node {
	articles := catalogList()
	cards := make([]ui.Node, 0, len(articles))
	for _, article := range articles {
		article := article
		cards = append(cards, html.Article(html.Props{Class: "rounded-[1.5rem] border border-white/10 bg-white/5 p-6"},
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-cyan-300"}, html.Text(strings.ToUpper(article.ID))),
			html.H2(html.Props{Class: "mt-3 text-2xl font-bold text-white"}, html.Text(article.Title)),
			html.P(html.Props{Class: "mt-3 text-sm leading-7 text-slate-300"}, html.Text(article.Summary)),
			html.A(html.Props{Href: "#/docs/" + article.ID + "?tab=overview", Class: "mt-5 inline-flex rounded-full border border-cyan-900/80 bg-cyan-950/70 px-4 py-2 text-sm font-semibold text-cyan-100 hover:bg-cyan-900/80"}, html.Text("Open route")),
		))
	}

	return html.Div(html.Props{Class: "grid gap-6 md:grid-cols-3"}, cards...)
}

func renderDocsPage(view demoShellView, actions ...ui.Node) ui.Node {
	article := articleForSection(view.SectionID)
	actionNodes := make([]ui.Node, 0, len(actions)+3)
	actionNodes = append(actionNodes,
		navLink("Overview tab", "#/docs/"+article.ID+"?tab=overview", view.CurrentTab == "overview"),
		navLink("Loader tab", "#/docs/"+article.ID+"?tab=loader", view.CurrentTab == "loader"),
		navLink("Bench tab", "#/docs/"+article.ID+"?tab=bench", view.CurrentTab == "bench"),
	)
	actionNodes = append(actionNodes, actions...)

	highlights := make([]ui.Node, 0, len(article.Highlights))
	for _, item := range article.Highlights {
		highlights = append(highlights, html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-white/5 p-4 text-slate-200"}, html.Text(item)))
	}

	return html.Div(html.Props{Class: "space-y-8"},
		html.Section(html.Props{Class: "rounded-[1.75rem] border border-white/10 bg-white/5 p-8"},
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("Docs route")),
			html.H2(html.Props{Class: "mt-3 text-4xl font-black text-white"}, html.Text(view.SectionTitle)),
			html.P(html.Props{Class: "mt-4 max-w-3xl text-lg leading-8 text-slate-300"}, html.Text(view.SectionBody)),
			html.P(html.Props{Class: "mt-4 text-sm uppercase tracking-[0.28em] text-slate-400"}, html.Text("Current tab: "+emptyFallback(view.CurrentTab, "overview"))),
			html.P(html.Props{Class: "mt-2 text-sm text-slate-500"}, html.Text(emptyFallback(view.Notice, "Route loader data is keyed by params and query state."))),
			html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"}, actionNodes...),
		),
		html.Section(html.Props{Class: "grid gap-4 md:grid-cols-3"}, highlights...),
		deferredRouteInsights(view),
	)
}

func renderSearchPage(view demoShellView, actions ...ui.Node) ui.Node {
	results := make([]ui.Node, 0, len(view.SearchResults))
	for _, article := range view.SearchResults {
		article := article
		results = append(results, html.Article(html.Props{Class: "rounded-[1.5rem] border border-white/10 bg-white/5 p-6"},
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-cyan-300"}, html.Text(strings.ToUpper(article.ID))),
			html.H3(html.Props{Class: "mt-3 text-2xl font-bold text-white"}, html.Text(article.Title)),
			html.P(html.Props{Class: "mt-3 text-sm leading-7 text-slate-300"}, html.Text(article.Summary)),
			html.A(html.Props{Href: "#/docs/" + article.ID + "?tab=overview", Class: "mt-5 inline-flex rounded-full border border-cyan-900/80 bg-cyan-950/70 px-4 py-2 text-sm font-semibold text-cyan-100 hover:bg-cyan-900/80"}, html.Text("Open detail")),
		))
	}
	if len(results) == 0 {
		results = append(results, html.Div(html.Props{Class: "rounded-[1.5rem] border border-dashed border-white/10 bg-white/5 p-6 text-slate-300"}, html.Text("No search results matched the current query.")))
	}

	return html.Div(html.Props{Class: "space-y-8"},
		html.Section(html.Props{Class: "rounded-[1.75rem] border border-white/10 bg-white/5 p-8"},
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("Search route")),
			html.H2(html.Props{Class: "mt-3 text-4xl font-black text-white"}, html.Text("Query-driven loader results")),
			html.P(html.Props{Class: "mt-4 max-w-3xl text-lg leading-8 text-slate-300"}, html.Text("Use the route buttons to replace query state and rerun the search loader without leaving the routed shell.")),
			html.P(html.Props{Class: "mt-4 text-sm uppercase tracking-[0.28em] text-slate-400"}, html.Text("Current query: "+emptyFallback(view.SearchQuery, "none"))),
			html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
				navLink("All", "#/search", view.SearchQuery == ""),
				navLink("SSR", "#/search?q=ssr", view.SearchQuery == "ssr"),
				navLink("Routing", "#/search?q=routing", view.SearchQuery == "routing"),
				navLink("Bench", "#/search?q=bench", view.SearchQuery == "bench"),
			),
			func() ui.Node {
				if len(actions) == 0 {
					return nil
				}
				return html.Div(html.Props{Class: "mt-4 flex flex-wrap gap-3"}, actions...)
			}(),
			deferredRouteInsights(view),
		),
		html.Section(html.Props{Class: "grid gap-4 md:grid-cols-2"}, results...),
	)
}

func renderSignInPage(view demoShellView, actions ...ui.Node) ui.Node {
	actionNodes := make([]ui.Node, 0, len(actions)+2)
	actionNodes = append(actionNodes,
		navLink("Grant Access", "#/secure?auth=true&role=maintainer", false),
		navLink("Open overview", "#/", false),
	)
	actionNodes = append(actionNodes, actions...)

	return html.Div(html.Props{Class: "rounded-[1.75rem] border border-white/10 bg-white/5 p-8"},
		html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("Guard redirect")),
		html.H2(html.Props{Class: "mt-3 text-4xl font-black text-white"}, html.Text("Protected routes can redirect before rendering")),
		html.P(html.Props{Class: "mt-4 max-w-3xl text-lg leading-8 text-slate-300"}, html.Text(emptyFallback(view.Notice, "The protected route redirected here because auth=true was not present in the query string."))),
		html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"}, actionNodes...),
	)
}

func renderSecurePage(view demoShellView, actions ...ui.Node) ui.Node {
	actionNodes := make([]ui.Node, 0, len(actions)+2)
	actionNodes = append(actionNodes,
		navLink("Switch Role", "#/secure?auth=true&role=auditor", false),
		navLink("Revoke", "#/signin", false),
	)
	actionNodes = append(actionNodes, actions...)

	return html.Div(html.Props{Class: "space-y-8"},
		html.Section(html.Props{Class: "rounded-[1.75rem] border border-white/10 bg-white/5 p-8"},
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("Protected route")),
			html.H2(html.Props{Class: "mt-3 text-4xl font-black text-white"}, html.Text("Guarded content unlocked")),
			html.P(html.Props{Class: "mt-4 max-w-3xl text-lg leading-8 text-slate-300"}, html.Text(emptyFallback(view.Notice, "The before-enter guard allowed this route and the loader produced a protected payload."))),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-3"},
				statCard("User", emptyFallback(view.SecureUser, "Morgan Reconciler")),
				statCard("Role", emptyFallback(view.SecureRole, "maintainer")),
				statCard("Loader revision", fmt.Sprintf("%d", view.LoadRevision)),
			),
			html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"}, actionNodes...),
		),
		deferredRouteInsights(view),
	)
}

func deferredRouteInsights(view demoShellView) ui.Node {
	loaderKey := deferredRouteKey(view)

	return ui.CreateElement(ui.Lazy, ui.LazyProps{
		Loader: func(ctx context.Context) (ui.Node, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(140 * time.Millisecond):
			}

			return html.Section(html.Props{Class: "rounded-[1.75rem] border border-cyan-500/20 bg-cyan-500/5 p-6"},
				html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("Deferred route insights")),
				html.H3(html.Props{Class: "mt-3 text-2xl font-black text-white"}, html.Text("Nested async panel ready")),
				html.P(html.Props{Class: "mt-3 text-sm leading-7 text-slate-300"}, html.Text("This panel resolves through ui.Lazy after the route loader has already produced the shell, showing nested async UI inside a loader-backed route.")),
				html.P(html.Props{Class: "mt-3 text-xs uppercase tracking-[0.28em] text-slate-400"}, html.Text("Loader key: "+loaderKey)),
			), nil
		},
		Dependencies: []interface{}{loaderKey},
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
		ErrorFallback: func(err error) ui.Node {
			return html.Section(html.Props{Class: "rounded-[1.75rem] border border-rose-400/20 bg-rose-400/10 p-6"},
				html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-rose-300"}, html.Text("Deferred route insights")),
				html.P(html.Props{Class: "mt-3 text-sm text-rose-100"}, html.Text("Deferred route panel failed: "+err.Error())),
			)
		},
	})
}

func deferredRouteKey(view demoShellView) string {
	switch view.Mode {
	case "docs":
		return fmt.Sprintf("docs:%s:%s:%d", emptyFallback(view.SectionID, "ssr"), emptyFallback(view.CurrentTab, "overview"), view.LoadRevision)
	case "search":
		return fmt.Sprintf("search:%s:%d", emptyFallback(view.SearchQuery, "all"), view.LoadRevision)
	case "secure":
		return fmt.Sprintf("secure:%s:%s:%d", emptyFallback(view.SecureUser, "guest"), emptyFallback(view.SecureRole, "none"), view.LoadRevision)
	default:
		return fmt.Sprintf("%s:%d", emptyFallback(view.Mode, "route"), view.LoadRevision)
	}
}

func navLink(label string, href string, active bool) ui.Node {
	className := "inline-flex rounded-full border px-4 py-2 text-sm font-semibold transition-colors "
	if active {
		className += "border-cyan-900/90 bg-cyan-950/70 text-cyan-100"
	} else {
		className += "border-white/10 bg-white/5 text-slate-200 hover:bg-white/10"
	}
	return html.A(html.Props{Href: href, Class: className}, html.Text(label))
}

func statCard(label string, value string) ui.Node {
	return html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-white/5 p-4"},
		html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text(label)),
		html.P(html.Props{Class: "mt-3 text-xl font-bold text-white"}, html.Text(value)),
	)
}

func emptyFallback(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

package main

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

type guideArticle struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Summary    string   `json:"summary"`
	Highlights []string `json:"highlights,omitempty"`
}

type bootstrapRouteData struct {
	Page          string         `json:"page"`
	SectionID     string         `json:"sectionId,omitempty"`
	SectionTitle  string         `json:"sectionTitle,omitempty"`
	SectionBody   string         `json:"sectionBody,omitempty"`
	CurrentTab    string         `json:"currentTab,omitempty"`
	StreamMode    string         `json:"streamMode,omitempty"`
	SearchQuery   string         `json:"searchQuery,omitempty"`
	SearchResults []guideArticle `json:"searchResults,omitempty"`
	SecureRole    string         `json:"secureRole,omitempty"`
	SecureUser    string         `json:"secureUser,omitempty"`
	Notice        string         `json:"notice,omitempty"`
	Revision      int            `json:"revision"`
}

type demoShellView struct {
	Page          string
	ActivePath    string
	Transport     string
	SectionID     string
	SectionTitle  string
	SectionBody   string
	CurrentTab    string
	StreamMode    string
	SearchQuery   string
	SearchResults []guideArticle
	SecureRole    string
	SecureUser    string
	Notice        string
	Revision      int
}

var guideCatalog = map[string]guideArticle{
	serverGuideSectionSSR: {
		ID:      serverGuideSectionSSR,
		Title:   "Server-rendered bootstrap flow",
		Summary: "Each request is rendered on the server, then the browser restores a route-specific bootstrap payload and reuses matching DOM during hydration.",
		Highlights: []string{
			"HTML is generated per request with ui.RenderToString(...).",
			"A route-specific bootstrap endpoint is fetched on startup.",
			"The first hydrated route restores bootstrap data and reuses matching server DOM instead of recomputing blindly.",
			"If hydration hits a structural mismatch, only the affected subtree falls back to client rendering.",
		},
	},
	serverGuideSectionRouting: {
		ID:      serverGuideSectionRouting,
		Title:   "Advanced routing over real URLs",
		Summary: "The demo uses browser/history routing, request-time redirects, protected routes, and query-aware route data across direct URL loads.",
		Highlights: []string{
			"/legacy redirects server-side and client-side into /docs/routing?tab=loader.",
			"/secure redirects to /signin unless auth=true is present.",
			"/search?q=... renders filtered server HTML on every direct navigation.",
		},
	},
	serverGuideSectionTransport: {
		ID:      serverGuideSectionTransport,
		Title:   "SSR transport choices",
		Summary: "The bootstrap payload is explicit and versionable. This demo uses JSON over an HTTP sidecar endpoint so network traffic is easy to inspect.",
		Highlights: []string{
			"You can see the HTML request and bootstrap request separately in DevTools.",
			"The same bootstrap shape is reused by native tests and wasm startup.",
			"The transport can be swapped to binary later without changing the route model.",
		},
	},
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

func articleForSection(parseSection string) guideArticle {
	if parseArticle, parseOk := guideCatalog[parseSection]; parseOk {
		return parseArticle
	}
	return guideCatalog[serverGuideSectionSSR]
}

func filterCatalog(parseQuery string) []guideArticle {
	parseTrimmed := strings.TrimSpace(strings.ToLower(parseQuery))
	if parseTrimmed == "" {
		return catalogList()
	}
	parseResults := make([]guideArticle, 0, len(guideCatalog))
	for _, parseArticle := range catalogList() {
		if strings.Contains(strings.ToLower(parseArticle.Title), parseTrimmed) || strings.Contains(strings.ToLower(parseArticle.Summary), parseTrimmed) {
			parseResults = append(parseResults, parseArticle)
		}
	}
	return parseResults
}

func revisionFromQuery(parseQuery url.Values) int {
	parseValue := strings.TrimSpace(parseQuery.Get("refresh"))
	if parseValue == "" {
		return 1
	}
	parseRevision, parseErr := strconv.Atoi(parseValue)
	if parseErr != nil || parseRevision < 1 {
		return 1
	}
	return parseRevision
}

func transportFromBootstrap(parsePayload ui.SSRBootstrap) string {
	if parsePayload.Data == nil {
		return serverBootstrapTransport
	}
	parseValue, _ := parsePayload.Data["transport"].(string)
	if parseValue == "" {
		return serverBootstrapTransport
	}
	return parseValue
}

func viewFromRouteData(parsePath string, parseTransport string, parseData bootstrapRouteData) demoShellView {
	return demoShellView{
		Page:          parseData.Page,
		ActivePath:    parsePath,
		Transport:     parseTransport,
		SectionID:     parseData.SectionID,
		SectionTitle:  parseData.SectionTitle,
		SectionBody:   parseData.SectionBody,
		CurrentTab:    emptyFallback(parseData.CurrentTab, serverTabOverview),
		StreamMode:    normalizeStreamMode(parseData.StreamMode),
		SearchQuery:   parseData.SearchQuery,
		SearchResults: parseData.SearchResults,
		SecureRole:    parseData.SecureRole,
		SecureUser:    parseData.SecureUser,
		Notice:        parseData.Notice,
		Revision:      maxInt(parseData.Revision, 1),
	}
}

func maxInt(parseValue int, parseFallback int) int {
	if parseValue <= 0 {
		return parseFallback
	}
	return parseValue
}

func normalizePath(parsePath string) string {
	parseTrimmed := strings.TrimSpace(parsePath)
	if parseTrimmed == "" {
		return "/"
	}
	if !strings.HasPrefix(parseTrimmed, "/") {
		parseTrimmed = "/" + parseTrimmed
	}
	if len(parseTrimmed) > 1 {
		parseTrimmed = strings.TrimRight(parseTrimmed, "/")
	}
	return parseTrimmed
}

func renderDemoShell(parseView demoShellView) ui.Node {
	return renderDemoShellWithDeferredMode(parseView, false)
}

func normalizeStreamMode(parseMode string) string {
	switch strings.TrimSpace(strings.ToLower(parseMode)) {
	case serverStreamModeError:
		return serverStreamModeError
	case serverStreamModeNested:
		return serverStreamModeNested
	default:
		return serverStreamModeReady
	}
}

func renderDemoShellWithDeferredMode(parseView demoShellView, isStreamDeferred bool) ui.Node {
	return html.Div(html.Props{Class: "min-h-screen bg-[#07131d] text-slate-100"},
		html.Div(html.Props{Class: "mx-auto max-w-6xl px-6 py-10"},
			html.Div(html.Props{Class: "rounded-[2rem] border border-white/10 bg-[radial-gradient(circle_at_top,_rgba(34,211,238,0.16),_transparent_42%),rgba(15,23,42,0.94)] p-8 shadow-2xl"},
				html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("Server SSR Demo")),
				html.H1(html.Props{Class: "mt-4 text-5xl font-black tracking-tight text-white"}, html.Text("Request-time SSR with real URLs and browser hydration")),
				html.P(html.Props{Class: "mt-4 max-w-3xl text-lg leading-8 text-slate-300"}, html.Text("Every navigation link below can hit the Go server for fresh HTML. The wasm client then restores a route-specific bootstrap payload, reuses matching DOM during hydration, and mirrors the same route model in the browser.")),
				html.Div(html.Props{Class: "mt-8 flex flex-wrap gap-3"},
					navLink("Overview", "/", parseView.ActivePath == "/"),
					navLink("Docs", "/docs/ssr", strings.HasPrefix(parseView.ActivePath, "/docs")),
					navLink("Search", "/search", strings.HasPrefix(parseView.ActivePath, "/search")),
					navLink("Protected", "/secure", strings.HasPrefix(parseView.ActivePath, "/secure")),
					navLink("Legacy Redirect", "/legacy", false),
					navLink("Sign In", "/signin", strings.HasPrefix(parseView.ActivePath, "/signin")),
				),
				html.Div(html.Props{Class: "mt-8 grid gap-4 md:grid-cols-3"},
					statCard("Active path", parseView.ActivePath),
					statCard("Transport", emptyFallback(parseView.Transport, serverBootstrapTransport)),
					statCard("Revision", fmt.Sprintf("%d", maxInt(parseView.Revision, 1))),
				),
			),
			renderPage(parseView, isStreamDeferred),
		),
	)
}

func renderPage(parseView demoShellView, isStreamDeferred bool) ui.Node {
	switch parseView.Page {
	case serverPageHome:
		parseCards := make([]ui.Node, 0, len(catalogList()))
		for _, parseArticle := range catalogList() {
			parseArticle2 := parseArticle
			parseCards = append(parseCards, html.Article(html.Props{Class: "rounded-[1.5rem] border border-white/10 bg-white/5 p-6"},
				html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-cyan-300"}, html.Text(strings.ToUpper(parseArticle2.ID))),
				html.H2(html.Props{Class: "mt-3 text-2xl font-bold text-white"}, html.Text(parseArticle2.Title)),
				html.P(html.Props{Class: "mt-3 text-sm leading-7 text-slate-300"}, html.Text(parseArticle2.Summary)),
				navLink("Open route", "/docs/"+parseArticle2.ID+"?tab=overview", false),
			))
		}
		return html.Section(html.Props{Class: "mt-8 grid gap-4 md:grid-cols-3"}, parseCards...)
	case serverPageDocs:
		parseActionLinks := []ui.Node{
			navLink("Overview tab", "/docs/"+parseView.SectionID+"?tab="+serverTabOverview, parseView.CurrentTab == serverTabOverview),
			navLink("Loader tab", "/docs/"+parseView.SectionID+"?tab="+serverTabLoader, parseView.CurrentTab == serverTabLoader),
			navLink("Transport tab", "/docs/"+parseView.SectionID+"?tab="+serverTabTransport, parseView.CurrentTab == serverTabTransport),
			navLink("Server rerender", "/docs/"+parseView.SectionID+"?tab="+url.QueryEscape(parseView.CurrentTab)+"&refresh="+strconv.Itoa(parseView.Revision+1), false),
		}
		parseArticle3 := articleForSection(parseView.SectionID)
		parseHighlights := make([]ui.Node, 0, len(parseArticle3.Highlights))
		for _, parseItem := range parseArticle3.Highlights {
			parseHighlights = append(parseHighlights, html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-white/5 p-4 text-slate-200"}, html.Text(parseItem)))
		}
		parseContentNodes := []ui.Node{
			html.Section(html.Props{Class: "rounded-[1.75rem] border border-white/10 bg-white/5 p-8"},
				html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("Docs route")),
				html.H2(html.Props{Class: "mt-3 text-4xl font-black text-white"}, html.Text(parseView.SectionTitle)),
				html.P(html.Props{Class: "mt-4 max-w-3xl text-lg leading-8 text-slate-300"}, html.Text(parseView.SectionBody)),
				html.P(html.Props{Class: "mt-4 text-sm uppercase tracking-[0.28em] text-slate-400"}, html.Text("Current tab: "+parseView.CurrentTab)),
				html.P(html.Props{Class: "mt-2 text-sm text-slate-500"}, html.Text(parseView.Notice)),
				html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"}, parseActionLinks...),
			),
			html.Section(html.Props{Class: "grid gap-4 md:grid-cols-3"}, parseHighlights...),
		}
		if shouldRenderDeferredDocsPanel(parseView) {
			if isStreamDeferred {
				parseContentNodes = append(parseContentNodes, renderDeferredDocsPanelPlaceholder())
			} else {
				parseContentNodes = append(parseContentNodes, renderDeferredDocsPanel(parseView))
			}
		}
		return html.Div(html.Props{Class: "mt-8 space-y-8"}, parseContentNodes...)
	case serverPageSearch:
		parseResults := make([]ui.Node, 0, len(parseView.SearchResults))
		for _, parseArticle4 := range parseView.SearchResults {
			parseArticle5 := parseArticle4
			parseResults = append(parseResults, html.Article(html.Props{Class: "rounded-[1.5rem] border border-white/10 bg-white/5 p-6"},
				html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-cyan-300"}, html.Text(strings.ToUpper(parseArticle5.ID))),
				html.H3(html.Props{Class: "mt-3 text-2xl font-bold text-white"}, html.Text(parseArticle5.Title)),
				html.P(html.Props{Class: "mt-3 text-sm leading-7 text-slate-300"}, html.Text(parseArticle5.Summary)),
				navLink("Open detail", "/docs/"+parseArticle5.ID+"?tab=overview", false),
			))
		}
		if len(parseResults) == 0 {
			parseResults = append(parseResults, html.Div(html.Props{Class: "rounded-[1.5rem] border border-dashed border-white/10 bg-white/5 p-6 text-slate-300"}, html.Text("No search results matched the current query.")))
		}
		return html.Div(html.Props{Class: "mt-8 space-y-8"},
			html.Section(html.Props{Class: "rounded-[1.75rem] border border-white/10 bg-white/5 p-8"},
				html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("Search route")),
				html.H2(html.Props{Class: "mt-3 text-4xl font-black text-white"}, html.Text("Query-driven server results")),
				html.P(html.Props{Class: "mt-4 max-w-3xl text-lg leading-8 text-slate-300"}, html.Text("Each search link below can trigger a full server render with URL-specific HTML and a matching bootstrap payload.")),
				html.P(html.Props{Class: "mt-4 text-sm uppercase tracking-[0.28em] text-slate-400"}, html.Text("Current query: "+emptyFallback(parseView.SearchQuery, "none"))),
				html.P(html.Props{Class: "mt-2 text-sm text-slate-500"}, html.Text(parseView.Notice)),
				html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
					navLink("All", "/search", parseView.SearchQuery == ""),
					navLink("SSR", "/search?q=ssr", parseView.SearchQuery == "ssr"),
					navLink("Routing", "/search?q=routing", parseView.SearchQuery == "routing"),
					navLink("Transport", "/search?q="+serverGuideSectionTransport, parseView.SearchQuery == serverGuideSectionTransport),
					navLink("Server rerender", "/search?q="+url.QueryEscape(parseView.SearchQuery)+"&refresh="+strconv.Itoa(parseView.Revision+1), false),
				),
			),
			html.Section(html.Props{Class: "grid gap-4 md:grid-cols-2"}, parseResults...),
		)
	case serverPageSecure:
		return html.Section(html.Props{Class: "mt-8 rounded-[1.75rem] border border-white/10 bg-white/5 p-8"},
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("Protected route")),
			html.H2(html.Props{Class: "mt-3 text-4xl font-black text-white"}, html.Text("Guarded content unlocked")),
			html.P(html.Props{Class: "mt-4 max-w-3xl text-lg leading-8 text-slate-300"}, html.Text(parseView.Notice)),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-3"},
				statCard("User", emptyFallback(parseView.SecureUser, serverSecureUserDefault)),
				statCard("Role", emptyFallback(parseView.SecureRole, serverSecureRoleMaintainer)),
				statCard("Revision", fmt.Sprintf("%d", maxInt(parseView.Revision, 1))),
			),
			html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
				navLink("Switch role", "/secure?auth=true&role=auditor", false),
				navLink("Server rerender", "/secure?auth=true&role="+url.QueryEscape(parseView.SecureRole)+"&refresh="+strconv.Itoa(parseView.Revision+1), false),
				navLink("Revoke access", secureRedirectPath, false),
			),
		)
	case serverPageSignIn:
		return html.Section(html.Props{Class: "mt-8 rounded-[1.75rem] border border-white/10 bg-white/5 p-8"},
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("Guard redirect")),
			html.H2(html.Props{Class: "mt-3 text-4xl font-black text-white"}, html.Text("Protected routes can redirect before rendering")),
			html.P(html.Props{Class: "mt-4 max-w-3xl text-lg leading-8 text-slate-300"}, html.Text(parseView.Notice)),
			html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
				navLink("Grant access", "/secure?auth=true&role="+serverSecureRoleMaintainer, false),
				navLink("Open overview", "/", false),
			),
		)
	default:
		return html.Section(html.Props{Class: "mt-8 rounded-[1.75rem] border border-white/10 bg-white/5 p-8"},
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-rose-300"}, html.Text("Not found")),
			html.H2(html.Props{Class: "mt-3 text-4xl font-black text-white"}, html.Text("Route not found")),
			html.P(html.Props{Class: "mt-4 max-w-3xl text-lg leading-8 text-slate-300"}, html.Text(parseView.Notice)),
		)
	}
}

func shouldRenderDeferredDocsPanel(parseView demoShellView) bool {
	return parseView.Page == serverPageDocs && parseView.CurrentTab == serverTabLoader
}

func renderDeferredDocsPanel(parseView demoShellView) ui.Node {
	switch normalizeStreamMode(parseView.StreamMode) {
	case serverStreamModeError:
		return html.Section(html.Props{ID: serverDeferredPanelID, Class: "rounded-[1.75rem] border border-rose-400/20 bg-rose-400/10 p-6"},
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-rose-300"}, html.Text("Deferred route panel")),
			html.H3(html.Props{Class: "mt-3 text-2xl font-black text-white"}, html.Text("Deferred docs panel failed after shell flush")),
			html.P(html.Props{Class: "mt-3 text-sm leading-7 text-rose-100"}, html.Text("The server replaced the placeholder with an explicit error region instead of silently dropping the streamed segment.")),
			html.P(html.Props{Class: "mt-3 text-xs uppercase tracking-[0.28em] text-rose-200"}, html.Text("Section: "+emptyFallback(parseView.SectionID, serverGuideSectionSSR)+" • Revision: "+strconv.Itoa(maxInt(parseView.Revision, 1)))),
		)
	case serverStreamModeNested:
		return html.Section(html.Props{ID: serverDeferredPanelID, Class: "rounded-[1.75rem] border border-cyan-500/20 bg-cyan-500/5 p-6"},
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("Deferred route panel")),
			html.H3(html.Props{Class: "mt-3 text-2xl font-black text-white"}, html.Text("Nested streamed layout ready")),
			html.P(html.Props{Class: "mt-3 text-sm leading-7 text-slate-300"}, html.Text("This variant keeps the same streamed region identity while proving the server can flush a shell first and later replace it with a nested layout subtree.")),
			html.Div(html.Props{Class: "mt-5 grid gap-4 md:grid-cols-2"},
				html.Article(html.Props{Class: "rounded-2xl border border-white/10 bg-black/20 p-4"},
					html.P(html.Props{Class: "text-xs uppercase tracking-[0.28em] text-slate-400"}, html.Text("Outer layout")),
					html.P(html.Props{Class: "mt-3 text-sm text-slate-200"}, html.Text("Server shell remained stable while the deferred region resolved.")),
				),
				html.Article(html.Props{Class: "rounded-2xl border border-white/10 bg-black/20 p-4"},
					html.P(html.Props{Class: "text-xs uppercase tracking-[0.28em] text-slate-400"}, html.Text("Nested child panel")),
					html.P(html.Props{Class: "mt-3 text-sm text-slate-200"}, html.Text("Hydration should target this final nested subtree, not the original placeholder.")),
				),
			),
		)
	}
	return html.Section(html.Props{ID: serverDeferredPanelID, Class: "rounded-[1.75rem] border border-cyan-500/20 bg-cyan-500/5 p-6"},
		html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("Deferred route panel")),
		html.H3(html.Props{Class: "mt-3 text-2xl font-black text-white"}, html.Text("Streamed docs insights ready")),
		html.P(html.Props{Class: "mt-3 text-sm leading-7 text-slate-300"}, html.Text("This panel is the minimal streamed region for the request-time SSR demo. The server can flush the shell first and send this nested docs panel later while hydration still targets the final assembled DOM.")),
		html.P(html.Props{Class: "mt-3 text-xs uppercase tracking-[0.28em] text-slate-400"}, html.Text("Section: "+emptyFallback(parseView.SectionID, serverGuideSectionSSR)+" • Revision: "+strconv.Itoa(maxInt(parseView.Revision, 1)))),
	)
}

func renderDeferredDocsPanelPlaceholder() ui.Node {
	return html.Section(html.Props{ID: serverDeferredPanelID, Class: "rounded-[1.75rem] border border-white/10 bg-white/5 p-6"},
		html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("Deferred route panel")),
		html.P(html.Props{Class: "mt-3 text-sm text-slate-300"}, html.Text("Streaming nested docs panel...")),
	)
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

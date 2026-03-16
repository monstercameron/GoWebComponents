package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

const bootstrapEndpointPath = "/_gwc/bootstrap"

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
	SearchQuery   string
	SearchResults []guideArticle
	SecureRole    string
	SecureUser    string
	Notice        string
	Revision      int
}

type resolvedRoute struct {
	Status       int
	Redirect     string
	Path         string
	Title        string
	Description  string
	CanonicalURL string
	Bootstrap    ui.SSRBootstrap
	View         demoShellView
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

func articleForSection(section string) guideArticle {
	if article, ok := guideCatalog[section]; ok {
		return article
	}
	return guideCatalog[serverGuideSectionSSR]
}

func filterCatalog(query string) []guideArticle {
	trimmed := strings.TrimSpace(strings.ToLower(query))
	if trimmed == "" {
		return catalogList()
	}
	results := make([]guideArticle, 0, len(guideCatalog))
	for _, article := range catalogList() {
		if strings.Contains(strings.ToLower(article.Title), trimmed) || strings.Contains(strings.ToLower(article.Summary), trimmed) {
			results = append(results, article)
		}
	}
	return results
}

func revisionFromQuery(query url.Values) int {
	value := strings.TrimSpace(query.Get("refresh"))
	if value == "" {
		return 1
	}
	revision, err := strconv.Atoi(value)
	if err != nil || revision < 1 {
		return 1
	}
	return revision
}

func buildBootstrap(path string, query url.Values, params map[string]string, routeData bootstrapRouteData) ui.SSRBootstrap {
	queryMap := make(map[string][]string, len(query))
	for key, values := range query {
		queryMap[key] = append([]string(nil), values...)
	}

	return ui.SSRBootstrap{
		Route: ui.SSRRouteBootstrap{
			Path:   path,
			Query:  queryMap,
			Params: cloneParams(params),
		},
		Data: map[string]interface{}{
			"transport": serverBootstrapTransport,
			"routeData": routeData,
		},
		IDSeed: routeData.Revision,
	}
}

func cloneParams(params map[string]string) map[string]string {
	if len(params) == 0 {
		return map[string]string{}
	}
	clone := make(map[string]string, len(params))
	for key, value := range params {
		clone[key] = value
	}
	return clone
}

func transportFromBootstrap(payload ui.SSRBootstrap) string {
	if payload.Data == nil {
		return serverBootstrapTransport
	}
	value, _ := payload.Data["transport"].(string)
	if value == "" {
		return serverBootstrapTransport
	}
	return value
}

func decodeBootstrapRouteData(payload ui.SSRBootstrap) (bootstrapRouteData, bool) {
	if payload.Data == nil {
		return bootstrapRouteData{}, false
	}
	raw, ok := payload.Data["routeData"]
	if !ok {
		return bootstrapRouteData{}, false
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return bootstrapRouteData{}, false
	}
	var decoded bootstrapRouteData
	if err := json.Unmarshal(data, &decoded); err != nil {
		return bootstrapRouteData{}, false
	}
	return decoded, true
}

func viewFromRouteData(path string, transport string, data bootstrapRouteData) demoShellView {
	return demoShellView{
		Page:          data.Page,
		ActivePath:    path,
		Transport:     transport,
		SectionID:     data.SectionID,
		SectionTitle:  data.SectionTitle,
		SectionBody:   data.SectionBody,
		CurrentTab:    emptyFallback(data.CurrentTab, serverTabOverview),
		SearchQuery:   data.SearchQuery,
		SearchResults: data.SearchResults,
		SecureRole:    data.SecureRole,
		SecureUser:    data.SecureUser,
		Notice:        data.Notice,
		Revision:      maxInt(data.Revision, 1),
	}
}

func maxInt(value int, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}

func resolveRoute(path string, query url.Values) resolvedRoute {
	normalizedPath := normalizePath(path)
	switch {
	case normalizedPath == "/":
		data := bootstrapRouteData{
			Page:     serverPageHome,
			Notice:   "This page is rendered by the Go HTTP server on every request. Use the links below to trigger new server-side renders over real URLs.",
			Revision: revisionFromQuery(query),
		}
		bootstrap := buildBootstrap(normalizedPath, query, nil, data)
		return resolvedRoute{
			Status:       200,
			Path:         normalizedPath,
			Title:        "GWC Server SSR Demo",
			Description:  "Request-time SSR shell rendered by a Go server and hydrated by the wasm client.",
			CanonicalURL: serverCanonicalBaseURL + "/",
			Bootstrap:    bootstrap,
			View:         viewFromRouteData(normalizedPath, transportFromBootstrap(bootstrap), data),
		}
	case normalizedPath == "/legacy":
		return resolvedRoute{Status: 302, Redirect: legacyRedirectPath}
	case strings.HasPrefix(normalizedPath, "/docs/"):
		section := strings.TrimPrefix(normalizedPath, "/docs/")
		article := articleForSection(section)
		currentTab := emptyFallback(strings.TrimSpace(query.Get("tab")), serverTabOverview)
		data := bootstrapRouteData{
			Page:         serverPageDocs,
			SectionID:    article.ID,
			SectionTitle: article.Title,
			SectionBody:  article.Summary,
			CurrentTab:   currentTab,
			Notice:       fmt.Sprintf("Server-rendered docs response for %s with tab=%s.", article.ID, currentTab),
			Revision:     revisionFromQuery(query),
		}
		params := map[string]string{"section": article.ID}
		bootstrap := buildBootstrap(normalizedPath, query, params, data)
		return resolvedRoute{
			Status:       200,
			Path:         normalizedPath,
			Title:        article.Title + " | GWC Server SSR Demo",
			Description:  article.Summary,
			CanonicalURL: serverCanonicalBaseURL + buildPathWithQuery(normalizedPath, query),
			Bootstrap:    bootstrap,
			View:         viewFromRouteData(normalizedPath, transportFromBootstrap(bootstrap), data),
		}
	case normalizedPath == "/search":
		searchQuery := strings.TrimSpace(query.Get("q"))
		data := bootstrapRouteData{
			Page:          serverPageSearch,
			SearchQuery:   searchQuery,
			SearchResults: filterCatalog(searchQuery),
			Notice:        "Every direct search navigation renders fresh HTML and a fresh bootstrap payload on the server.",
			Revision:      revisionFromQuery(query),
		}
		bootstrap := buildBootstrap(normalizedPath, query, nil, data)
		return resolvedRoute{
			Status:       200,
			Path:         normalizedPath,
			Title:        "Search | GWC Server SSR Demo",
			Description:  "Server-rendered search results over real URLs.",
			CanonicalURL: serverCanonicalBaseURL + buildPathWithQuery(normalizedPath, query),
			Bootstrap:    bootstrap,
			View:         viewFromRouteData(normalizedPath, transportFromBootstrap(bootstrap), data),
		}
	case normalizedPath == "/secure":
		if query.Get("auth") != "true" {
			return resolvedRoute{Status: 302, Redirect: secureRedirectPath}
		}
		role := emptyFallback(strings.TrimSpace(query.Get("role")), serverSecureRoleMaintainer)
		data := bootstrapRouteData{
			Page:       serverPageSecure,
			SecureRole: role,
			SecureUser: serverSecureUserDefault,
			Notice:     "The secure route was allowed by the server and mirrored by the client browser router after hydration.",
			Revision:   revisionFromQuery(query),
		}
		bootstrap := buildBootstrap(normalizedPath, query, nil, data)
		return resolvedRoute{
			Status:       200,
			Path:         normalizedPath,
			Title:        "Secure | GWC Server SSR Demo",
			Description:  "Protected route rendered on the server and hydrated on the client.",
			CanonicalURL: serverCanonicalBaseURL + buildPathWithQuery(normalizedPath, query),
			Bootstrap:    bootstrap,
			View:         viewFromRouteData(normalizedPath, transportFromBootstrap(bootstrap), data),
		}
	case normalizedPath == "/signin":
		from := strings.TrimSpace(query.Get("from"))
		notice := "The protected route redirected here because auth=true was not present."
		if from != "" {
			notice = "The protected route redirected here from " + from + "."
		}
		data := bootstrapRouteData{
			Page:     serverPageSignIn,
			Notice:   notice,
			Revision: revisionFromQuery(query),
		}
		bootstrap := buildBootstrap(normalizedPath, query, nil, data)
		return resolvedRoute{
			Status:       200,
			Path:         normalizedPath,
			Title:        "Sign In | GWC Server SSR Demo",
			Description:  "Guard redirect target for the protected server-rendered route.",
			CanonicalURL: serverCanonicalBaseURL + buildPathWithQuery(normalizedPath, query),
			Bootstrap:    bootstrap,
			View:         viewFromRouteData(normalizedPath, transportFromBootstrap(bootstrap), data),
		}
	default:
		data := bootstrapRouteData{
			Page:     serverPageNotFound,
			Notice:   "This path is not registered by the server SSR demo.",
			Revision: 1,
		}
		bootstrap := buildBootstrap(normalizedPath, query, nil, data)
		return resolvedRoute{
			Status:       404,
			Path:         normalizedPath,
			Title:        "Not Found | GWC Server SSR Demo",
			Description:  "Unknown route for the server-rendered SSR demo.",
			CanonicalURL: serverCanonicalBaseURL + buildPathWithQuery(normalizedPath, query),
			Bootstrap:    bootstrap,
			View:         viewFromRouteData(normalizedPath, transportFromBootstrap(bootstrap), data),
		}
	}
}

func buildPathWithQuery(path string, query url.Values) string {
	encoded := query.Encode()
	if encoded == "" {
		return path
	}
	return path + "?" + encoded
}

func normalizePath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "/"
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	if len(trimmed) > 1 {
		trimmed = strings.TrimRight(trimmed, "/")
	}
	return trimmed
}

func bootstrapReferenceURL(path string, query url.Values) string {
	values := url.Values{}
	values.Set("path", path)
	for key, items := range query {
		for _, item := range items {
			values.Add(key, item)
		}
	}
	return bootstrapEndpointPath + "?" + values.Encode()
}

func renderDemoShell(view demoShellView) ui.Node {
	return html.Div(html.Props{Class: "min-h-screen bg-[#07131d] text-slate-100"},
		html.Div(html.Props{Class: "mx-auto max-w-6xl px-6 py-10"},
			html.Div(html.Props{Class: "rounded-[2rem] border border-white/10 bg-[radial-gradient(circle_at_top,_rgba(34,211,238,0.16),_transparent_42%),rgba(15,23,42,0.94)] p-8 shadow-2xl"},
				html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("Server SSR Demo")),
				html.H1(html.Props{Class: "mt-4 text-5xl font-black tracking-tight text-white"}, html.Text("Request-time SSR with real URLs and browser hydration")),
				html.P(html.Props{Class: "mt-4 max-w-3xl text-lg leading-8 text-slate-300"}, html.Text("Every navigation link below can hit the Go server for fresh HTML. The wasm client then restores a route-specific bootstrap payload, reuses matching DOM during hydration, and mirrors the same route model in the browser.")),
				html.Div(html.Props{Class: "mt-8 flex flex-wrap gap-3"},
					navLink("Overview", "/", view.ActivePath == "/"),
					navLink("Docs", "/docs/ssr", strings.HasPrefix(view.ActivePath, "/docs")),
					navLink("Search", "/search", strings.HasPrefix(view.ActivePath, "/search")),
					navLink("Protected", "/secure", strings.HasPrefix(view.ActivePath, "/secure")),
					navLink("Legacy Redirect", "/legacy", false),
					navLink("Sign In", "/signin", strings.HasPrefix(view.ActivePath, "/signin")),
				),
				html.Div(html.Props{Class: "mt-8 grid gap-4 md:grid-cols-3"},
					statCard("Active path", view.ActivePath),
					statCard("Transport", emptyFallback(view.Transport, serverBootstrapTransport)),
					statCard("Revision", fmt.Sprintf("%d", maxInt(view.Revision, 1))),
				),
			),
			renderPage(view),
		),
	)
}

func renderPage(view demoShellView) ui.Node {
	switch view.Page {
	case serverPageHome:
		cards := make([]ui.Node, 0, len(catalogList()))
		for _, article := range catalogList() {
			article := article
			cards = append(cards, html.Article(html.Props{Class: "rounded-[1.5rem] border border-white/10 bg-white/5 p-6"},
				html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-cyan-300"}, html.Text(strings.ToUpper(article.ID))),
				html.H2(html.Props{Class: "mt-3 text-2xl font-bold text-white"}, html.Text(article.Title)),
				html.P(html.Props{Class: "mt-3 text-sm leading-7 text-slate-300"}, html.Text(article.Summary)),
				navLink("Open route", "/docs/"+article.ID+"?tab=overview", false),
			))
		}
		return html.Section(html.Props{Class: "mt-8 grid gap-4 md:grid-cols-3"}, cards...)
	case serverPageDocs:
		actionLinks := []ui.Node{
			navLink("Overview tab", "/docs/"+view.SectionID+"?tab="+serverTabOverview, view.CurrentTab == serverTabOverview),
			navLink("Loader tab", "/docs/"+view.SectionID+"?tab="+serverTabLoader, view.CurrentTab == serverTabLoader),
			navLink("Transport tab", "/docs/"+view.SectionID+"?tab="+serverTabTransport, view.CurrentTab == serverTabTransport),
			navLink("Server rerender", "/docs/"+view.SectionID+"?tab="+url.QueryEscape(view.CurrentTab)+"&refresh="+strconv.Itoa(view.Revision+1), false),
		}
		article := articleForSection(view.SectionID)
		highlights := make([]ui.Node, 0, len(article.Highlights))
		for _, item := range article.Highlights {
			highlights = append(highlights, html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-white/5 p-4 text-slate-200"}, html.Text(item)))
		}
		return html.Div(html.Props{Class: "mt-8 space-y-8"},
			html.Section(html.Props{Class: "rounded-[1.75rem] border border-white/10 bg-white/5 p-8"},
				html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("Docs route")),
				html.H2(html.Props{Class: "mt-3 text-4xl font-black text-white"}, html.Text(view.SectionTitle)),
				html.P(html.Props{Class: "mt-4 max-w-3xl text-lg leading-8 text-slate-300"}, html.Text(view.SectionBody)),
				html.P(html.Props{Class: "mt-4 text-sm uppercase tracking-[0.28em] text-slate-400"}, html.Text("Current tab: "+view.CurrentTab)),
				html.P(html.Props{Class: "mt-2 text-sm text-slate-500"}, html.Text(view.Notice)),
				html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"}, actionLinks...),
			),
			html.Section(html.Props{Class: "grid gap-4 md:grid-cols-3"}, highlights...),
		)
	case serverPageSearch:
		results := make([]ui.Node, 0, len(view.SearchResults))
		for _, article := range view.SearchResults {
			article := article
			results = append(results, html.Article(html.Props{Class: "rounded-[1.5rem] border border-white/10 bg-white/5 p-6"},
				html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-cyan-300"}, html.Text(strings.ToUpper(article.ID))),
				html.H3(html.Props{Class: "mt-3 text-2xl font-bold text-white"}, html.Text(article.Title)),
				html.P(html.Props{Class: "mt-3 text-sm leading-7 text-slate-300"}, html.Text(article.Summary)),
				navLink("Open detail", "/docs/"+article.ID+"?tab=overview", false),
			))
		}
		if len(results) == 0 {
			results = append(results, html.Div(html.Props{Class: "rounded-[1.5rem] border border-dashed border-white/10 bg-white/5 p-6 text-slate-300"}, html.Text("No search results matched the current query.")))
		}
		return html.Div(html.Props{Class: "mt-8 space-y-8"},
			html.Section(html.Props{Class: "rounded-[1.75rem] border border-white/10 bg-white/5 p-8"},
				html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("Search route")),
				html.H2(html.Props{Class: "mt-3 text-4xl font-black text-white"}, html.Text("Query-driven server results")),
				html.P(html.Props{Class: "mt-4 max-w-3xl text-lg leading-8 text-slate-300"}, html.Text("Each search link below can trigger a full server render with URL-specific HTML and a matching bootstrap payload.")),
				html.P(html.Props{Class: "mt-4 text-sm uppercase tracking-[0.28em] text-slate-400"}, html.Text("Current query: "+emptyFallback(view.SearchQuery, "none"))),
				html.P(html.Props{Class: "mt-2 text-sm text-slate-500"}, html.Text(view.Notice)),
				html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
					navLink("All", "/search", view.SearchQuery == ""),
					navLink("SSR", "/search?q=ssr", view.SearchQuery == "ssr"),
					navLink("Routing", "/search?q=routing", view.SearchQuery == "routing"),
					navLink("Transport", "/search?q="+serverGuideSectionTransport, view.SearchQuery == serverGuideSectionTransport),
					navLink("Server rerender", "/search?q="+url.QueryEscape(view.SearchQuery)+"&refresh="+strconv.Itoa(view.Revision+1), false),
				),
			),
			html.Section(html.Props{Class: "grid gap-4 md:grid-cols-2"}, results...),
		)
	case serverPageSecure:
		return html.Section(html.Props{Class: "mt-8 rounded-[1.75rem] border border-white/10 bg-white/5 p-8"},
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("Protected route")),
			html.H2(html.Props{Class: "mt-3 text-4xl font-black text-white"}, html.Text("Guarded content unlocked")),
			html.P(html.Props{Class: "mt-4 max-w-3xl text-lg leading-8 text-slate-300"}, html.Text(view.Notice)),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-3"},
				statCard("User", emptyFallback(view.SecureUser, serverSecureUserDefault)),
				statCard("Role", emptyFallback(view.SecureRole, serverSecureRoleMaintainer)),
				statCard("Revision", fmt.Sprintf("%d", maxInt(view.Revision, 1))),
			),
			html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
				navLink("Switch role", "/secure?auth=true&role=auditor", false),
				navLink("Server rerender", "/secure?auth=true&role="+url.QueryEscape(view.SecureRole)+"&refresh="+strconv.Itoa(view.Revision+1), false),
				navLink("Revoke access", secureRedirectPath, false),
			),
		)
	case serverPageSignIn:
		return html.Section(html.Props{Class: "mt-8 rounded-[1.75rem] border border-white/10 bg-white/5 p-8"},
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("Guard redirect")),
			html.H2(html.Props{Class: "mt-3 text-4xl font-black text-white"}, html.Text("Protected routes can redirect before rendering")),
			html.P(html.Props{Class: "mt-4 max-w-3xl text-lg leading-8 text-slate-300"}, html.Text(view.Notice)),
			html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
				navLink("Grant access", "/secure?auth=true&role="+serverSecureRoleMaintainer, false),
				navLink("Open overview", "/", false),
			),
		)
	default:
		return html.Section(html.Props{Class: "mt-8 rounded-[1.75rem] border border-white/10 bg-white/5 p-8"},
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-rose-300"}, html.Text("Not found")),
			html.H2(html.Props{Class: "mt-3 text-4xl font-black text-white"}, html.Text("Route not found")),
			html.P(html.Props{Class: "mt-4 max-w-3xl text-lg leading-8 text-slate-300"}, html.Text(view.Notice)),
		)
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

package main

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/ui"
)

func TestSharedHelpersAndFallbacks(parseT *testing.T) {
	parsePayload := defaultBootstrapPayload()
	if parsePayload.Route.Path != "/docs/"+guideSectionSSR {
		parseT.Fatalf("defaultBootstrapPayload() path = %q", parsePayload.Route.Path)
	}
	if parsePayload.Route.Params["section"] != guideSectionSSR || parsePayload.IDSeed != 17 {
		parseT.Fatalf("defaultBootstrapPayload() = %+v", parsePayload)
	}
	if parseGot := bootstrapTransport(parsePayload); parseGot != transportJSONSidecar {
		parseT.Fatalf("bootstrapTransport(default) = %q, want %q", parseGot, transportJSONSidecar)
	}
	if parseGot2 := bootstrapTransport(ui.SSRBootstrap{}); parseGot2 != transportJSONSidecar {
		parseT.Fatalf("bootstrapTransport(empty) = %q, want default transport", parseGot2)
	}
	parseCustomPayload := ui.SSRBootstrap{Data: map[string]any{"transport": "binary"}}
	if parseGot3 := bootstrapTransport(parseCustomPayload); parseGot3 != "binary" {
		parseT.Fatalf("bootstrapTransport(custom) = %q, want binary", parseGot3)
	}

	parseView := defaultServerView()
	if parseView.Mode != modeDocs || parseView.ActivePath != parsePayload.Route.Path || parseView.Transport != transportJSONSidecar {
		parseT.Fatalf("defaultServerView() = %+v", parseView)
	}
	if parseArticle := articleForSection("missing"); parseArticle.ID != guideSectionSSR {
		parseT.Fatalf("articleForSection(fallback) = %+v, want SSR article", parseArticle)
	}
	parseArticles := catalogList()
	if len(parseArticles) != 3 || parseArticles[0].ID != guideSectionBenchmarks || parseArticles[1].ID != guideSectionRouting || parseArticles[2].ID != guideSectionSSR {
		parseT.Fatalf("catalogList() order = %+v", parseArticles)
	}
	if parseGot4 := emptyFallback("   ", "fallback"); parseGot4 != "fallback" {
		parseT.Fatalf("emptyFallback(blank) = %q, want fallback", parseGot4)
	}
	if parseGot5 := emptyFallback("value", "fallback"); parseGot5 != "value" {
		parseT.Fatalf("emptyFallback(value) = %q, want value", parseGot5)
	}
}

func TestRenderPageModesAndHelpers(parseT *testing.T) {
	parseTests := []struct {
		name string
		view demoShellView
		want string
	}{
		{
			name: "home",
			view: demoShellView{Mode: "home", ActivePath: "/"},
			want: "Microbenchmarks for SSR delivery",
		},
		{
			name: "search-empty",
			view: demoShellView{Mode: "search", ActivePath: "/search", SearchQuery: "zzz", SearchResults: nil},
			want: "No search results matched the current query.",
		},
		{
			name: "signin",
			view: demoShellView{Mode: "signin", ActivePath: "/signin", Notice: "Guard redirect"},
			want: "Protected routes can redirect before rendering",
		},
		{
			name: "secure",
			view: demoShellView{Mode: "secure", ActivePath: "/secure", SecureUser: "Morgan", SecureRole: "auditor", LoadRevision: 2},
			want: "Guarded content unlocked",
		},
		{
			name: "docs-default",
			view: demoShellView{Mode: "other", ActivePath: "/docs/ssr", SectionID: guideSectionSSR, SectionTitle: "SSR transport and hydration", SectionBody: "Summary", CurrentTab: tabLoader, LoadRevision: 3},
			want: "Docs route",
		},
	}

	for _, parseTest := range parseTests {
		parseT.Run(parseTest.name, func(parseT2 *testing.T) {
			parseMarkup, parseErr := ui.RenderToString(renderPage(parseTest.view))
			if parseErr != nil {
				parseT2.Fatalf("RenderToString(renderPage(%s)) error = %v", parseTest.name, parseErr)
			}
			if !strings.Contains(parseMarkup, parseTest.want) {
				parseT2.Fatalf("renderPage(%s) missing %q\n%s", parseTest.name, parseTest.want, parseMarkup)
			}
		})
	}

	parseDemoMarkup, parseErr2 := ui.RenderToString(renderDemoShell(demoShellView{
		Mode:          modeDocs,
		ActivePath:    "/docs/ssr",
		BootstrapPath: "/docs/ssr",
		Transport:     transportJSONSidecar,
		SectionID:     guideSectionSSR,
		SectionTitle:  "SSR transport and hydration",
		SectionBody:   "Server-rendered HTML shell",
		CurrentTab:    tabOverview,
		LoadRevision:  4,
		Notice:        "Hydration path",
	}))
	if parseErr2 != nil {
		parseT.Fatalf("RenderToString(renderDemoShell) error = %v", parseErr2)
	}
	for _, parseExpected := range []string{"SSR Routing", "Bootstrap Route", "/docs/ssr", "Transport", transportJSONSidecar} {
		if !strings.Contains(parseDemoMarkup, parseExpected) {
			parseT.Fatalf("renderDemoShell() missing %q\n%s", parseExpected, parseDemoMarkup)
		}
	}

	if parseKey := deferredRouteKey(demoShellView{Mode: modeDocs, SectionID: guideSectionSSR, CurrentTab: tabLoader, LoadRevision: 2}); parseKey != "docs:ssr:loader:2" {
		parseT.Fatalf("deferredRouteKey(docs) = %q", parseKey)
	}
	if parseKey2 := deferredRouteKey(demoShellView{Mode: "search", SearchQuery: "", LoadRevision: 3}); parseKey2 != "search:all:3" {
		parseT.Fatalf("deferredRouteKey(search) = %q", parseKey2)
	}
	if parseKey3 := deferredRouteKey(demoShellView{Mode: "secure", SecureUser: "", SecureRole: "", LoadRevision: 4}); parseKey3 != "secure:guest:none:4" {
		parseT.Fatalf("deferredRouteKey(secure) = %q", parseKey3)
	}
	if parseKey4 := deferredRouteKey(demoShellView{Mode: "", LoadRevision: 5}); parseKey4 != "route:5" {
		parseT.Fatalf("deferredRouteKey(default) = %q", parseKey4)
	}

	parseActiveLink, parseErr2 := ui.RenderToString(navLink("Docs", routeDocsSSRHash, true))
	if parseErr2 != nil {
		parseT.Fatalf("RenderToString(navLink active) error = %v", parseErr2)
	}
	if !strings.Contains(parseActiveLink, "border-cyan-900/90") {
		parseT.Fatalf("active navLink markup = %q, want active class", parseActiveLink)
	}
	parseInactiveLink, parseErr2 := ui.RenderToString(navLink("Docs", routeDocsSSRHash, false))
	if parseErr2 != nil {
		parseT.Fatalf("RenderToString(navLink inactive) error = %v", parseErr2)
	}
	if !strings.Contains(parseInactiveLink, "hover:bg-white/10") {
		parseT.Fatalf("inactive navLink markup = %q, want hover class", parseInactiveLink)
	}

	parseCard, parseErr2 := ui.RenderToString(statCard("Revision", "7"))
	if parseErr2 != nil {
		parseT.Fatalf("RenderToString(statCard) error = %v", parseErr2)
	}
	if !strings.Contains(parseCard, "Revision") || !strings.Contains(parseCard, "7") {
		parseT.Fatalf("statCard markup = %q, want label and value", parseCard)
	}
}

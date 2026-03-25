package main

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/ui"
)

func TestSharedHelpersAndFallbacks(t *testing.T) {
	payload := defaultBootstrapPayload()
	if payload.Route.Path != "/docs/"+guideSectionSSR {
		t.Fatalf("defaultBootstrapPayload() path = %q", payload.Route.Path)
	}
	if payload.Route.Params["section"] != guideSectionSSR || payload.IDSeed != 17 {
		t.Fatalf("defaultBootstrapPayload() = %+v", payload)
	}
	if got := bootstrapTransport(payload); got != transportJSONSidecar {
		t.Fatalf("bootstrapTransport(default) = %q, want %q", got, transportJSONSidecar)
	}
	if got := bootstrapTransport(ui.SSRBootstrap{}); got != transportJSONSidecar {
		t.Fatalf("bootstrapTransport(empty) = %q, want default transport", got)
	}
	customPayload := ui.SSRBootstrap{Data: map[string]interface{}{"transport": "binary"}}
	if got := bootstrapTransport(customPayload); got != "binary" {
		t.Fatalf("bootstrapTransport(custom) = %q, want binary", got)
	}

	view := defaultServerView()
	if view.Mode != modeDocs || view.ActivePath != payload.Route.Path || view.Transport != transportJSONSidecar {
		t.Fatalf("defaultServerView() = %+v", view)
	}
	if article := articleForSection("missing"); article.ID != guideSectionSSR {
		t.Fatalf("articleForSection(fallback) = %+v, want SSR article", article)
	}
	articles := catalogList()
	if len(articles) != 3 || articles[0].ID != guideSectionBenchmarks || articles[1].ID != guideSectionRouting || articles[2].ID != guideSectionSSR {
		t.Fatalf("catalogList() order = %+v", articles)
	}
	if got := emptyFallback("   ", "fallback"); got != "fallback" {
		t.Fatalf("emptyFallback(blank) = %q, want fallback", got)
	}
	if got := emptyFallback("value", "fallback"); got != "value" {
		t.Fatalf("emptyFallback(value) = %q, want value", got)
	}
}

func TestRenderPageModesAndHelpers(t *testing.T) {
	tests := []struct {
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

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			markup, err := ui.RenderToString(renderPage(test.view))
			if err != nil {
				t.Fatalf("RenderToString(renderPage(%s)) error = %v", test.name, err)
			}
			if !strings.Contains(markup, test.want) {
				t.Fatalf("renderPage(%s) missing %q\n%s", test.name, test.want, markup)
			}
		})
	}

	demoMarkup, err := ui.RenderToString(renderDemoShell(demoShellView{
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
	if err != nil {
		t.Fatalf("RenderToString(renderDemoShell) error = %v", err)
	}
	for _, expected := range []string{"SSR Routing Demo", "Bootstrap route", "/docs/ssr", "Transport", transportJSONSidecar} {
		if !strings.Contains(demoMarkup, expected) {
			t.Fatalf("renderDemoShell() missing %q\n%s", expected, demoMarkup)
		}
	}

	if key := deferredRouteKey(demoShellView{Mode: modeDocs, SectionID: guideSectionSSR, CurrentTab: tabLoader, LoadRevision: 2}); key != "docs:ssr:loader:2" {
		t.Fatalf("deferredRouteKey(docs) = %q", key)
	}
	if key := deferredRouteKey(demoShellView{Mode: "search", SearchQuery: "", LoadRevision: 3}); key != "search:all:3" {
		t.Fatalf("deferredRouteKey(search) = %q", key)
	}
	if key := deferredRouteKey(demoShellView{Mode: "secure", SecureUser: "", SecureRole: "", LoadRevision: 4}); key != "secure:guest:none:4" {
		t.Fatalf("deferredRouteKey(secure) = %q", key)
	}
	if key := deferredRouteKey(demoShellView{Mode: "", LoadRevision: 5}); key != "route:5" {
		t.Fatalf("deferredRouteKey(default) = %q", key)
	}

	activeLink, err := ui.RenderToString(navLink("Docs", routeDocsSSRHash, true))
	if err != nil {
		t.Fatalf("RenderToString(navLink active) error = %v", err)
	}
	if !strings.Contains(activeLink, "border-cyan-900/90") {
		t.Fatalf("active navLink markup = %q, want active class", activeLink)
	}
	inactiveLink, err := ui.RenderToString(navLink("Docs", routeDocsSSRHash, false))
	if err != nil {
		t.Fatalf("RenderToString(navLink inactive) error = %v", err)
	}
	if !strings.Contains(inactiveLink, "hover:bg-white/10") {
		t.Fatalf("inactive navLink markup = %q, want hover class", inactiveLink)
	}

	card, err := ui.RenderToString(statCard("Revision", "7"))
	if err != nil {
		t.Fatalf("RenderToString(statCard) error = %v", err)
	}
	if !strings.Contains(card, "Revision") || !strings.Contains(card, "7") {
		t.Fatalf("statCard markup = %q, want label and value", card)
	}
}

//go:build !js || !wasm

package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/ui"
)

func TestServerRoutingHelperBranches(parseT *testing.T) {
	parseReport := exampleServerRequestReport("handlePage", "/docs/ssr", os.ErrInvalid, "request failed", "inspect the route")
	if parseReport.Code != "GWC-EXAMPLE-SERVER-REQUEST" || !strings.Contains(parseReport.Headline, "handlePage") || parseReport.Path != "/docs/ssr" {
		parseT.Fatalf("exampleServerRequestReport() = %+v, want populated report", parseReport)
	}

	parseQuery := url.Values{
		"path": {"ignored"},
		"tab":  {"loader"},
		"q":    {"routing"},
	}
	parseClonedQuery := cloneQueryWithoutPath(parseQuery)
	if _, parseOk := parseClonedQuery["path"]; parseOk {
		parseT.Fatalf("cloneQueryWithoutPath() kept path query: %#v", parseClonedQuery)
	}
	parseClonedQuery.Set("tab", "overview")
	if parseGot := parseQuery.Get("tab"); parseGot != "loader" {
		parseT.Fatalf("cloneQueryWithoutPath() mutated original query, got %q", parseGot)
	}

	parseBootstrap := buildBootstrap("/docs/ssr", url.Values{"tab": {"loader"}}, map[string]string{"section": "ssr"}, bootstrapRouteData{
		Page:      serverPageDocs,
		SectionID: serverGuideSectionSSR,
		Revision:  3,
	})
	if parseBootstrap.IDSeed != 3 || parseBootstrap.Route.Path != "/docs/ssr" || parseBootstrap.Route.Params["section"] != "ssr" {
		parseT.Fatalf("buildBootstrap() = %+v, want cloned route bootstrap data", parseBootstrap)
	}
	if transportFromBootstrap(parseBootstrap) != serverBootstrapTransport {
		parseT.Fatalf("transportFromBootstrap() = %q, want %q", transportFromBootstrap(parseBootstrap), serverBootstrapTransport)
	}
	if parseGot2 := buildPathWithQuery("/docs/ssr", url.Values{"tab": {serverTabLoader}}); parseGot2 != "/docs/ssr?tab=loader" {
		parseT.Fatalf("buildPathWithQuery() = %q, want /docs/ssr?tab=loader", parseGot2)
	}
	if parseGot3 := buildPathWithQuery("/docs/ssr", url.Values{}); parseGot3 != "/docs/ssr" {
		parseT.Fatalf("buildPathWithQuery(empty) = %q, want /docs/ssr", parseGot3)
	}
	if parseGot4 := bootstrapReferenceURL("/search", url.Values{"q": {"ssr"}}); !strings.Contains(parseGot4, "path=%2Fsearch") || !strings.Contains(parseGot4, "q=ssr") {
		parseT.Fatalf("bootstrapReferenceURL() = %q, want encoded route path and query", parseGot4)
	}
	if parseGot5 := cloneParams(nil); len(parseGot5) != 0 {
		parseT.Fatalf("cloneParams(nil) = %#v, want empty map", parseGot5)
	}

	parseHome := resolveRoute("/", url.Values{"refresh": {"0"}})
	if parseHome.Status != http.StatusOK || parseHome.View.Page != serverPageHome || parseHome.View.Revision != 1 {
		parseT.Fatalf("resolveRoute(home) = %+v", parseHome)
	}

	parseSearch := resolveRoute("/search", url.Values{"q": {"routing"}, "refresh": {"2"}})
	if parseSearch.Status != http.StatusOK || parseSearch.View.Page != serverPageSearch || len(parseSearch.View.SearchResults) == 0 || parseSearch.View.Revision != 2 {
		parseT.Fatalf("resolveRoute(search) = %+v", parseSearch)
	}

	parseSecure := resolveRoute("/secure", url.Values{"auth": {"true"}, "role": {"auditor"}, "refresh": {"5"}})
	if parseSecure.Status != http.StatusOK || parseSecure.View.Page != serverPageSecure || parseSecure.View.SecureRole != "auditor" || parseSecure.View.Revision != 5 {
		parseT.Fatalf("resolveRoute(secure) = %+v", parseSecure)
	}

	parseSignIn := resolveRoute("/signin", url.Values{"from": {"/secure"}})
	if parseSignIn.Status != http.StatusOK || !strings.Contains(parseSignIn.View.Notice, "/secure") {
		parseT.Fatalf("resolveRoute(signin) = %+v", parseSignIn)
	}

	parseNotFound := resolveRoute("/missing", url.Values{})
	if parseNotFound.Status != http.StatusNotFound || parseNotFound.View.Page != serverPageNotFound {
		parseT.Fatalf("resolveRoute(notFound) = %+v", parseNotFound)
	}

	if !shouldStreamDeferredDocsPanel(resolveRoute("/docs/ssr", url.Values{"tab": {serverTabLoader}})) {
		parseT.Fatal("expected docs loader route to stream deferred panel")
	}
	if shouldStreamDeferredDocsPanel(parseHome) {
		parseT.Fatal("expected home route to avoid deferred streaming")
	}

	parseDocsHead := headDocumentForRoute(resolveRoute("/docs/ssr", url.Values{"tab": {serverTabOverview}}))
	if parseDocsHead.Robots != "index,follow" || len(parseDocsHead.Alternates) != 2 || len(parseDocsHead.JSONLD) != 1 {
		parseT.Fatalf("headDocumentForRoute(docs) = %+v", parseDocsHead)
	}
	parseSearchHead := headDocumentForRoute(parseSearch)
	if parseSearchHead.Robots != "noindex,follow" || len(parseSearchHead.JSONLD) != 1 {
		parseT.Fatalf("headDocumentForRoute(search) = %+v", parseSearchHead)
	}
	parseSecureHead := headDocumentForRoute(parseSecure)
	if parseSecureHead.Robots != "noindex,nofollow" {
		parseT.Fatalf("headDocumentForRoute(secure) = %+v", parseSecureHead)
	}

	parseReplacement := streamedPanelReplacementScript(serverDeferredPanelID, `<section id="done">ready</section>`)
	if !strings.Contains(parseReplacement, "target.outerHTML=") || !strings.Contains(parseReplacement, serverDeferredPanelID) || !strings.Contains(parseReplacement, "ready") {
		parseT.Fatalf("streamedPanelReplacementScript() = %q, want replacement script", parseReplacement)
	}
}

func TestServerRenderBranchesAndRoutes(parseT *testing.T) {
	parseViews := []struct {
		name string
		view demoShellView
		want string
		node func(demoShellView) ui.Node
	}{
		{
			name: "home",
			view: demoShellView{Page: serverPageHome, ActivePath: "/", Revision: 1},
			want: "Server-rendered bootstrap flow",
			node: renderDemoShell,
		},
		{
			name: "search-no-results",
			view: demoShellView{Page: serverPageSearch, ActivePath: "/search", SearchQuery: "zzz", SearchResults: nil, Notice: "Search notice", Revision: 2},
			want: "No search results matched the current query.",
			node: renderDemoShell,
		},
		{
			name: "secure",
			view: demoShellView{Page: serverPageSecure, ActivePath: "/secure", SecureUser: "cam@example.com", SecureRole: "auditor", Notice: "Allowed", Revision: 4},
			want: "Guarded content unlocked",
			node: renderDemoShell,
		},
		{
			name: "signin",
			view: demoShellView{Page: serverPageSignIn, ActivePath: "/signin", Notice: "Redirected", Revision: 1},
			want: "Protected routes can redirect before rendering",
			node: renderDemoShell,
		},
		{
			name: "not-found",
			view: demoShellView{Page: serverPageNotFound, ActivePath: "/missing", Notice: "Missing route"},
			want: "Route not found",
			node: renderDemoShell,
		},
		{
			name: "docs-stream-shell",
			view: demoShellView{Page: serverPageDocs, ActivePath: "/docs/ssr", SectionID: serverGuideSectionSSR, SectionTitle: "Server-rendered bootstrap flow", SectionBody: "Summary", CurrentTab: serverTabLoader, StreamMode: serverStreamModeReady, Revision: 3},
			want: "Streaming nested docs panel...",
			node: renderDemoShellStreamShell,
		},
	}

	for _, parseTest := range parseViews {
		parseT.Run(parseTest.name, func(parseT2 *testing.T) {
			parseMarkup, parseErr := ui.RenderToString(parseTest.node(parseTest.view))
			if parseErr != nil {
				parseT2.Fatalf("RenderToString(%s) error = %v", parseTest.name, parseErr)
			}
			if !strings.Contains(parseMarkup, parseTest.want) {
				parseT2.Fatalf("RenderToString(%s) missing %q\n%s", parseTest.name, parseTest.want, parseMarkup)
			}
		})
	}

	for _, parseMode := range []struct {
		name string
		mode string
		want string
	}{
		{name: "ready", mode: serverStreamModeReady, want: "Streamed docs insights ready"},
		{name: "error", mode: serverStreamModeError, want: "Deferred docs panel failed after shell flush"},
		{name: "nested", mode: serverStreamModeNested, want: "Nested streamed layout ready"},
	} {
		parseT.Run("deferred-"+parseMode.name, func(parseT3 *testing.T) {
			parseMarkup2, parseErr2 := ui.RenderToString(renderDeferredDocsPanel(demoShellView{
				Page:       serverPageDocs,
				SectionID:  serverGuideSectionSSR,
				CurrentTab: serverTabLoader,
				StreamMode: parseMode.mode,
				Revision:   2,
			}))
			if parseErr2 != nil {
				parseT3.Fatalf("RenderToString(renderDeferredDocsPanel(%s)) error = %v", parseMode.name, parseErr2)
			}
			if !strings.Contains(parseMarkup2, parseMode.want) {
				parseT3.Fatalf("renderDeferredDocsPanel(%s) missing %q\n%s", parseMode.name, parseMode.want, parseMarkup2)
			}
		})
	}

	if parseGot := revisionFromQuery(url.Values{"refresh": {"not-a-number"}}); parseGot != 1 {
		parseT.Fatalf("revisionFromQuery(invalid) = %d, want 1", parseGot)
	}
	if parseGot2 := normalizePath(" docs/ssr/ "); parseGot2 != "/docs/ssr" {
		parseT.Fatalf("normalizePath() = %q, want /docs/ssr", parseGot2)
	}
	if parseGot3 := normalizeStreamMode(" nested "); parseGot3 != serverStreamModeNested {
		parseT.Fatalf("normalizeStreamMode() = %q, want %q", parseGot3, serverStreamModeNested)
	}
	if parseGot4 := emptyFallback("   ", "fallback"); parseGot4 != "fallback" {
		parseT.Fatalf("emptyFallback(blank) = %q, want fallback", parseGot4)
	}
	if parseGot5 := maxInt(0, 7); parseGot5 != 7 {
		parseT.Fatalf("maxInt(0, 7) = %d, want 7", parseGot5)
	}
	if parseArticle := articleForSection("missing"); parseArticle.ID != serverGuideSectionSSR {
		parseT.Fatalf("articleForSection(fallback) = %+v, want SSR article", parseArticle)
	}
	if parseResults := filterCatalog(""); len(parseResults) != len(catalogList()) {
		parseT.Fatalf("filterCatalog(empty) len = %d, want %d", len(parseResults), len(catalogList()))
	}
}

func TestAppServerRoutesServeAssetsAndHealth(parseT *testing.T) {
	parseRootDir := parseT.TempDir()
	parseWasmPath := filepath.Join(parseRootDir, "app.wasm")
	parseCssPath := filepath.Join(parseRootDir, "tailwind.css")
	parseLoggerPath := filepath.Join(parseRootDir, "example-logger.js")
	parseWasmExecPath := filepath.Join(parseRootDir, "wasm_exec.js")
	for parsePath, parseContents := range map[string]string{
		parseWasmPath:     "wasm-bytes",
		parseCssPath:      "body{}",
		parseLoggerPath:   "console.log('logger')",
		parseWasmExecPath: "console.log('wasm_exec')",
	} {
		if parseErr := os.WriteFile(parsePath, []byte(parseContents), 0o644); parseErr != nil {
			parseT.Fatalf("WriteFile(%s): %v", parsePath, parseErr)
		}
	}

	parseServer := &appServer{
		wasmPath:     parseWasmPath,
		cssPath:      parseCssPath,
		loggerPath:   parseLoggerPath,
		wasmExecPath: parseWasmExecPath,
	}
	parseHandler := parseServer.routes()

	parseHealth := httptest.NewRecorder()
	parseHandler.ServeHTTP(parseHealth, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if parseHealth.Code != http.StatusOK || !strings.Contains(parseHealth.Body.String(), `{"ok":true}`) {
		parseT.Fatalf("healthz response = code %d body %q", parseHealth.Code, parseHealth.Body.String())
	}

	parseCss := httptest.NewRecorder()
	parseHandler.ServeHTTP(parseCss, httptest.NewRequest(http.MethodGet, "/assets/tailwind.css", nil))
	if !strings.Contains(parseCss.Body.String(), "body{}") || !strings.Contains(parseCss.Result().Header.Get("Content-Type"), "text/css") {
		parseT.Fatalf("css asset response = headers %#v body %q", parseCss.Result().Header, parseCss.Body.String())
	}

	parseFileRes := httptest.NewRecorder()
	fileHandler(parseLoggerPath, "text/javascript; charset=utf-8").ServeHTTP(parseFileRes, httptest.NewRequest(http.MethodGet, "/assets/example-logger.js", nil))
	if !strings.Contains(parseFileRes.Body.String(), "logger") || !strings.Contains(parseFileRes.Result().Header.Get("Content-Type"), "text/javascript") {
		parseT.Fatalf("fileHandler response = headers %#v body %q", parseFileRes.Result().Header, parseFileRes.Body.String())
	}
}

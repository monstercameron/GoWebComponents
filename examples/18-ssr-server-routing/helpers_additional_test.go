//go:build !js || !wasm
// +build !js !wasm

package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/ui"
)

func TestServerRoutingHelperBranches(t *testing.T) {
	report := exampleServerRequestReport("handlePage", "/docs/ssr", os.ErrInvalid, "request failed", "inspect the route")
	if report.Code != "GWC-EXAMPLE-SERVER-REQUEST" || !strings.Contains(report.Headline, "handlePage") || report.Path != "/docs/ssr" {
		t.Fatalf("exampleServerRequestReport() = %+v, want populated report", report)
	}

	query := url.Values{
		"path": {"ignored"},
		"tab":  {"loader"},
		"q":    {"routing"},
	}
	clonedQuery := cloneQueryWithoutPath(query)
	if _, ok := clonedQuery["path"]; ok {
		t.Fatalf("cloneQueryWithoutPath() kept path query: %#v", clonedQuery)
	}
	clonedQuery.Set("tab", "overview")
	if got := query.Get("tab"); got != "loader" {
		t.Fatalf("cloneQueryWithoutPath() mutated original query, got %q", got)
	}

	bootstrap := buildBootstrap("/docs/ssr", url.Values{"tab": {"loader"}}, map[string]string{"section": "ssr"}, bootstrapRouteData{
		Page:      serverPageDocs,
		SectionID: serverGuideSectionSSR,
		Revision:  3,
	})
	if bootstrap.IDSeed != 3 || bootstrap.Route.Path != "/docs/ssr" || bootstrap.Route.Params["section"] != "ssr" {
		t.Fatalf("buildBootstrap() = %+v, want cloned route bootstrap data", bootstrap)
	}
	if transportFromBootstrap(bootstrap) != serverBootstrapTransport {
		t.Fatalf("transportFromBootstrap() = %q, want %q", transportFromBootstrap(bootstrap), serverBootstrapTransport)
	}
	if got := buildPathWithQuery("/docs/ssr", url.Values{"tab": {serverTabLoader}}); got != "/docs/ssr?tab=loader" {
		t.Fatalf("buildPathWithQuery() = %q, want /docs/ssr?tab=loader", got)
	}
	if got := buildPathWithQuery("/docs/ssr", url.Values{}); got != "/docs/ssr" {
		t.Fatalf("buildPathWithQuery(empty) = %q, want /docs/ssr", got)
	}
	if got := bootstrapReferenceURL("/search", url.Values{"q": {"ssr"}}); !strings.Contains(got, "path=%2Fsearch") || !strings.Contains(got, "q=ssr") {
		t.Fatalf("bootstrapReferenceURL() = %q, want encoded route path and query", got)
	}
	if got := cloneParams(nil); len(got) != 0 {
		t.Fatalf("cloneParams(nil) = %#v, want empty map", got)
	}

	home := resolveRoute("/", url.Values{"refresh": {"0"}})
	if home.Status != http.StatusOK || home.View.Page != serverPageHome || home.View.Revision != 1 {
		t.Fatalf("resolveRoute(home) = %+v", home)
	}

	search := resolveRoute("/search", url.Values{"q": {"routing"}, "refresh": {"2"}})
	if search.Status != http.StatusOK || search.View.Page != serverPageSearch || len(search.View.SearchResults) == 0 || search.View.Revision != 2 {
		t.Fatalf("resolveRoute(search) = %+v", search)
	}

	secure := resolveRoute("/secure", url.Values{"auth": {"true"}, "role": {"auditor"}, "refresh": {"5"}})
	if secure.Status != http.StatusOK || secure.View.Page != serverPageSecure || secure.View.SecureRole != "auditor" || secure.View.Revision != 5 {
		t.Fatalf("resolveRoute(secure) = %+v", secure)
	}

	signIn := resolveRoute("/signin", url.Values{"from": {"/secure"}})
	if signIn.Status != http.StatusOK || !strings.Contains(signIn.View.Notice, "/secure") {
		t.Fatalf("resolveRoute(signin) = %+v", signIn)
	}

	notFound := resolveRoute("/missing", url.Values{})
	if notFound.Status != http.StatusNotFound || notFound.View.Page != serverPageNotFound {
		t.Fatalf("resolveRoute(notFound) = %+v", notFound)
	}

	if !shouldStreamDeferredDocsPanel(resolveRoute("/docs/ssr", url.Values{"tab": {serverTabLoader}})) {
		t.Fatal("expected docs loader route to stream deferred panel")
	}
	if shouldStreamDeferredDocsPanel(home) {
		t.Fatal("expected home route to avoid deferred streaming")
	}

	docsHead := headDocumentForRoute(resolveRoute("/docs/ssr", url.Values{"tab": {serverTabOverview}}))
	if docsHead.Robots != "index,follow" || len(docsHead.Alternates) != 2 || len(docsHead.JSONLD) != 1 {
		t.Fatalf("headDocumentForRoute(docs) = %+v", docsHead)
	}
	searchHead := headDocumentForRoute(search)
	if searchHead.Robots != "noindex,follow" || len(searchHead.JSONLD) != 1 {
		t.Fatalf("headDocumentForRoute(search) = %+v", searchHead)
	}
	secureHead := headDocumentForRoute(secure)
	if secureHead.Robots != "noindex,nofollow" {
		t.Fatalf("headDocumentForRoute(secure) = %+v", secureHead)
	}

	replacement := streamedPanelReplacementScript(serverDeferredPanelID, `<section id="done">ready</section>`)
	if !strings.Contains(replacement, "target.outerHTML=") || !strings.Contains(replacement, serverDeferredPanelID) || !strings.Contains(replacement, "ready") {
		t.Fatalf("streamedPanelReplacementScript() = %q, want replacement script", replacement)
	}
}

func TestServerRenderBranchesAndRoutes(t *testing.T) {
	views := []struct {
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

	for _, test := range views {
		t.Run(test.name, func(t *testing.T) {
			markup, err := ui.RenderToString(test.node(test.view))
			if err != nil {
				t.Fatalf("RenderToString(%s) error = %v", test.name, err)
			}
			if !strings.Contains(markup, test.want) {
				t.Fatalf("RenderToString(%s) missing %q\n%s", test.name, test.want, markup)
			}
		})
	}

	for _, mode := range []struct {
		name string
		mode string
		want string
	}{
		{name: "ready", mode: serverStreamModeReady, want: "Streamed docs insights ready"},
		{name: "error", mode: serverStreamModeError, want: "Deferred docs panel failed after shell flush"},
		{name: "nested", mode: serverStreamModeNested, want: "Nested streamed layout ready"},
	} {
		t.Run("deferred-"+mode.name, func(t *testing.T) {
			markup, err := ui.RenderToString(renderDeferredDocsPanel(demoShellView{
				Page:       serverPageDocs,
				SectionID:  serverGuideSectionSSR,
				CurrentTab: serverTabLoader,
				StreamMode: mode.mode,
				Revision:   2,
			}))
			if err != nil {
				t.Fatalf("RenderToString(renderDeferredDocsPanel(%s)) error = %v", mode.name, err)
			}
			if !strings.Contains(markup, mode.want) {
				t.Fatalf("renderDeferredDocsPanel(%s) missing %q\n%s", mode.name, mode.want, markup)
			}
		})
	}

	if got := revisionFromQuery(url.Values{"refresh": {"not-a-number"}}); got != 1 {
		t.Fatalf("revisionFromQuery(invalid) = %d, want 1", got)
	}
	if got := normalizePath(" docs/ssr/ "); got != "/docs/ssr" {
		t.Fatalf("normalizePath() = %q, want /docs/ssr", got)
	}
	if got := normalizeStreamMode(" nested "); got != serverStreamModeNested {
		t.Fatalf("normalizeStreamMode() = %q, want %q", got, serverStreamModeNested)
	}
	if got := emptyFallback("   ", "fallback"); got != "fallback" {
		t.Fatalf("emptyFallback(blank) = %q, want fallback", got)
	}
	if got := maxInt(0, 7); got != 7 {
		t.Fatalf("maxInt(0, 7) = %d, want 7", got)
	}
	if article := articleForSection("missing"); article.ID != serverGuideSectionSSR {
		t.Fatalf("articleForSection(fallback) = %+v, want SSR article", article)
	}
	if results := filterCatalog(""); len(results) != len(catalogList()) {
		t.Fatalf("filterCatalog(empty) len = %d, want %d", len(results), len(catalogList()))
	}
}

func TestAppServerRoutesServeAssetsAndHealth(t *testing.T) {
	rootDir := t.TempDir()
	wasmPath := filepath.Join(rootDir, "app.wasm")
	cssPath := filepath.Join(rootDir, "tailwind.css")
	loggerPath := filepath.Join(rootDir, "example-logger.js")
	wasmExecPath := filepath.Join(rootDir, "wasm_exec.js")
	for path, contents := range map[string]string{
		wasmPath:     "wasm-bytes",
		cssPath:      "body{}",
		loggerPath:   "console.log('logger')",
		wasmExecPath: "console.log('wasm_exec')",
	} {
		if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
			t.Fatalf("WriteFile(%s): %v", path, err)
		}
	}

	server := &appServer{
		wasmPath:     wasmPath,
		cssPath:      cssPath,
		loggerPath:   loggerPath,
		wasmExecPath: wasmExecPath,
	}
	handler := server.routes()

	health := httptest.NewRecorder()
	handler.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if health.Code != http.StatusOK || !strings.Contains(health.Body.String(), `{"ok":true}`) {
		t.Fatalf("healthz response = code %d body %q", health.Code, health.Body.String())
	}

	css := httptest.NewRecorder()
	handler.ServeHTTP(css, httptest.NewRequest(http.MethodGet, "/assets/tailwind.css", nil))
	if !strings.Contains(css.Body.String(), "body{}") || !strings.Contains(css.Result().Header.Get("Content-Type"), "text/css") {
		t.Fatalf("css asset response = headers %#v body %q", css.Result().Header, css.Body.String())
	}

	fileRes := httptest.NewRecorder()
	fileHandler(loggerPath, "text/javascript; charset=utf-8").ServeHTTP(fileRes, httptest.NewRequest(http.MethodGet, "/assets/example-logger.js", nil))
	if !strings.Contains(fileRes.Body.String(), "logger") || !strings.Contains(fileRes.Result().Header.Get("Content-Type"), "text/javascript") {
		t.Fatalf("fileHandler response = headers %#v body %q", fileRes.Result().Header, fileRes.Body.String())
	}
}

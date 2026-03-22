package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/ui"
)

func TestFilterExampleLinksMatchesKeywordsAcrossNameAndHref(t *testing.T) {
	links := []exampleLink{
		{Name: "01-counter", Href: "/examples/01-counter/"},
		{Name: "17-ssr-routing", Href: "/examples/17-ssr-routing/"},
		{Name: "86-atlas-commerce-os", Href: "/examples/86-atlas-commerce-os/"},
	}

	got := filterExampleLinks(links, "atlas commerce")
	if len(got) != 1 {
		t.Fatalf("expected one atlas match, got %d: %#v", len(got), got)
	}
	if got[0].Name != "86-atlas-commerce-os" {
		t.Fatalf("expected atlas example match, got %q", got[0].Name)
	}

	got = filterExampleLinks(links, "17 examples")
	if len(got) != 1 || got[0].Name != "17-ssr-routing" {
		t.Fatalf("expected ssr-routing match from name and href, got %#v", got)
	}

	got = filterExampleLinks(links, "")
	if len(got) != len(links) {
		t.Fatalf("expected empty query to return all links, got %d", len(got))
	}
}

func TestRenderExamplesListingHTMLIncludesSearchState(t *testing.T) {
	html := renderExamplesListingHTML([]exampleLink{{Name: "01-counter", Href: "/examples/01-counter/"}}, `atlas "search"`)
	for _, expected := range []string{"name=\"q\"", "Filtered examples for", "atlas &quot;search&quot;", "01-counter"} {
		if !strings.Contains(html, expected) {
			t.Fatalf("expected examples HTML to contain %q", expected)
		}
	}
}

func TestRenderExamplesListingHTMLShowsNoMatchesState(t *testing.T) {
	html := renderExamplesListingHTML(nil, "nomatch")
	if !strings.Contains(html, "No examples matched this search yet.") {
		t.Fatalf("expected no-match helper text, got %s", html)
	}
}

func TestRenderExamplesAppShellHTMLIncludesWasmCatalogBootstrap(t *testing.T) {
	html := renderExamplesAppShellHTML("/", "/")
	for _, expected := range []string{"gwc-examples-site.wasm", "/examples/list", "GoWebComponents Examples", "gwc-examples-runtime-v1", "loadCachedWasm", "wasm source:", "__GWC_BOOTSTRAP__", "catalogURL"} {
		if !strings.Contains(html, expected) {
			t.Fatalf("expected app shell HTML to contain %q", expected)
		}
	}
	for _, expected := range []string{"\"catalogHref\":\"/\"", "\"path\":\"/\""} {
		if !strings.Contains(html, expected) {
			t.Fatalf("expected root app shell HTML to contain %q", expected)
		}
	}
	for _, unexpected := range []string{"&#39;caches&#39;", "&#39;/static/bin/gwc-examples-site.wasm&#39;", "result =&gt; go.run"} {
		if strings.Contains(html, unexpected) {
			t.Fatalf("expected app shell loader script to remain raw JavaScript, found escaped fragment %q", unexpected)
		}
	}
}

func TestStaticExamplesShellIncludesBootstrapContract(t *testing.T) {
	repoRoot, err := resolveRepoRoot()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	staticShellPath := filepath.Join(repoRoot, "examples", "static", "index.html")
	content, err := os.ReadFile(staticShellPath)
	if err != nil {
		t.Fatalf("read static shell: %v", err)
	}
	html := string(content)
	for _, expected := range []string{"__GWC_BOOTSTRAP__", "\"mode\":\"static\"", "\"catalogURL\":\"catalog.json\"", "\"wasmBase\":\"bin/\"", "\"catalogHref\":\"./index.html#/examples\"", "gwc-examples-site.wasm", "wasm source:"} {
		if !strings.Contains(html, expected) {
			t.Fatalf("expected static shell HTML to contain %q", expected)
		}
	}
}

func TestBuildExamplesListingUsesCatalogEntries(t *testing.T) {
	root := t.TempDir()
	examplesDir := filepath.Join(root, "examples")
	staticDir := filepath.Join(root, "static")
	for _, dirName := range []string{"02-second", "01-first", "notes"} {
		if err := os.MkdirAll(filepath.Join(examplesDir, dirName), 0755); err != nil {
			t.Fatalf("mkdir example dir %q: %v", dirName, err)
		}
	}
	if err := os.MkdirAll(filepath.Join(staticDir, "bin"), 0755); err != nil {
		t.Fatalf("mkdir static bin dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(examplesDir, "01-first", "index.html"), []byte("<title>First</title>"), 0644); err != nil {
		t.Fatalf("write first html: %v", err)
	}
	if err := os.WriteFile(filepath.Join(examplesDir, "02-second", "index.html"), []byte("<title>Second</title>"), 0644); err != nil {
		t.Fatalf("write second html: %v", err)
	}

	launcher := launcher{examplesDir: examplesDir, staticDir: staticDir}
	links, err := launcher.buildExamplesListing()
	if err != nil {
		t.Fatalf("build examples listing: %v", err)
	}
	if len(links) != 2 {
		t.Fatalf("expected two discoverable examples, got %#v", links)
	}
	if links[0] != (exampleLink{Name: "01-first", Href: "/examples/01-first/"}) {
		t.Fatalf("expected first sorted example link, got %#v", links[0])
	}
	if links[1] != (exampleLink{Name: "02-second", Href: "/examples/02-second/"}) {
		t.Fatalf("expected second sorted example link, got %#v", links[1])
	}
}

func TestWriteStaticExamplesCatalogFileWritesStaticHrefCatalog(t *testing.T) {
	root := t.TempDir()
	examplesDir := filepath.Join(root, "examples")
	staticDir := filepath.Join(root, "static")
	if err := os.MkdirAll(filepath.Join(examplesDir, "01-first"), 0755); err != nil {
		t.Fatalf("mkdir example dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(staticDir, "bin"), 0755); err != nil {
		t.Fatalf("mkdir static dir: %v", err)
	}
	html := `<html><head><title>First Example</title></head><body><script src="/static/bin/app.wasm"></script></body></html>`
	if err := os.WriteFile(filepath.Join(examplesDir, "01-first", "index.html"), []byte(html), 0644); err != nil {
		t.Fatalf("write example html: %v", err)
	}
	if err := os.WriteFile(filepath.Join(staticDir, "bin", "app.wasm"), []byte("wasm"), 0644); err != nil {
		t.Fatalf("write wasm binary: %v", err)
	}

	launcher := launcher{examplesDir: examplesDir, staticDir: staticDir}
	blankErr := launcher.writeStaticExamplesCatalogFile("   ")
	if blankErr == nil || !strings.Contains(blankErr.Error(), "static catalog output path is required") {
		t.Fatalf("expected blank output path error, got %v", blankErr)
	}

	targetPath := filepath.Join(root, "out", "catalog.json")
	if err := launcher.writeStaticExamplesCatalogFile(targetPath); err != nil {
		t.Fatalf("write static examples catalog file: %v", err)
	}
	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("read static catalog file: %v", err)
	}
	if !strings.HasSuffix(string(content), "\n") {
		t.Fatalf("expected static catalog file to end with newline, got %q", string(content))
	}
	var payload examplesCatalogPayload
	if err := json.Unmarshal(content, &payload); err != nil {
		t.Fatalf("decode static catalog file: %v", err)
	}
	if payload.TotalExamples != 1 || len(payload.Examples) != 1 {
		t.Fatalf("expected one static catalog entry, got %#v", payload)
	}
	entry := payload.Examples[0]
	if entry.Href != "../01-first/index.html" {
		t.Fatalf("expected static href, got %#v", entry)
	}
	if !entry.UsesWasm || entry.WasmBinary != "app.wasm" {
		t.Fatalf("expected wasm metadata in static catalog entry, got %#v", entry)
	}
	if entry.Title != "First Example" {
		t.Fatalf("expected html title to be preserved, got %#v", entry)
	}
}

func TestWriteStaticExamplesCatalogFileRelativeAndFailurePaths(t *testing.T) {
	root := t.TempDir()
	examplesDir := filepath.Join(root, "examples")
	staticDir := filepath.Join(root, "static")
	if err := os.MkdirAll(filepath.Join(examplesDir, "01-first"), 0755); err != nil {
		t.Fatalf("mkdir example dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(staticDir, "bin"), 0755); err != nil {
		t.Fatalf("mkdir static dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(examplesDir, "01-first", "index.html"), []byte("<title>First</title>"), 0644); err != nil {
		t.Fatalf("write html: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working dir: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir root: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalWD)
	}()

	examplesLauncher := launcher{examplesDir: examplesDir, staticDir: staticDir}
	if err := examplesLauncher.writeStaticExamplesCatalogFile(filepath.Join("out", "catalog.json")); err != nil {
		t.Fatalf("write relative static catalog: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "out", "catalog.json")); err != nil {
		t.Fatalf("expected relative static catalog file: %v", err)
	}

	failingLauncher := launcher{examplesDir: filepath.Join(root, "missing"), staticDir: staticDir}
	if err := failingLauncher.writeStaticExamplesCatalogFile(filepath.Join(root, "broken", "catalog.json")); err == nil {
		t.Fatal("expected static catalog build failure")
	}

	blockedParent := filepath.Join(root, "blocked")
	if err := os.WriteFile(blockedParent, []byte("file"), 0644); err != nil {
		t.Fatalf("write blocked parent: %v", err)
	}
	if err := examplesLauncher.writeStaticExamplesCatalogFile(filepath.Join(blockedParent, "catalog.json")); err == nil || !strings.Contains(err.Error(), "create static catalog directory") {
		t.Fatalf("expected static catalog directory creation failure, got %v", err)
	}

	originalMarshal := examplesCatalogMarshalIndent
	t.Cleanup(func() { examplesCatalogMarshalIndent = originalMarshal })
	examplesCatalogMarshalIndent = func(v interface{}, prefix string, indent string) ([]byte, error) {
		return nil, errors.New("encode failed")
	}
	if err := examplesLauncher.writeStaticExamplesCatalogFile(filepath.Join(root, "encode", "catalog.json")); err == nil || !strings.Contains(err.Error(), "encode static catalog") {
		t.Fatalf("expected static catalog encode failure, got %v", err)
	}
	examplesCatalogMarshalIndent = originalMarshal

	directoryTarget := filepath.Join(root, "directory-target")
	if err := os.MkdirAll(directoryTarget, 0755); err != nil {
		t.Fatalf("mkdir directory target: %v", err)
	}
	if err := examplesLauncher.writeStaticExamplesCatalogFile(directoryTarget); err == nil || !strings.Contains(err.Error(), "write static catalog") {
		t.Fatalf("expected static catalog write failure when target is a directory, got %v", err)
	}
}

func TestBuildBrowserTestEnvPreservesWorkerOverrideAndStripsWasmEnv(t *testing.T) {
	t.Setenv("GOOS", "js")
	t.Setenv("GOARCH", "wasm")
	t.Setenv("PLAYWRIGHT_WORKERS", "9")

	env := buildBrowserTestEnv()
	joined := strings.Join(env, "\n")
	if strings.Contains(joined, "GOOS=js") || strings.Contains(joined, "GOARCH=wasm") {
		t.Fatalf("expected browser env to strip wasm-specific variables, got %#v", env)
	}
	if !strings.Contains(joined, "PLAYWRIGHT_WORKERS=9") {
		t.Fatalf("expected browser env to preserve explicit worker override, got %#v", env)
	}
	if strings.Contains(joined, "PLAYWRIGHT_WORKERS=4") {
		t.Fatalf("expected browser env not to append default workers when already set, got %#v", env)
	}
}

func TestBuildBrowserTestEnvAddsDefaultWorkersWhenUnset(t *testing.T) {
	t.Setenv("GOOS", "js")
	t.Setenv("GOARCH", "wasm")
	originalWorkers, hadWorkers := os.LookupEnv("PLAYWRIGHT_WORKERS")
	if err := os.Unsetenv("PLAYWRIGHT_WORKERS"); err != nil {
		t.Fatalf("unset PLAYWRIGHT_WORKERS: %v", err)
	}
	t.Cleanup(func() {
		if !hadWorkers {
			_ = os.Unsetenv("PLAYWRIGHT_WORKERS")
			return
		}
		_ = os.Setenv("PLAYWRIGHT_WORKERS", originalWorkers)
	})

	env := buildBrowserTestEnv()
	joined := strings.Join(env, "\n")
	if !strings.Contains(joined, "PLAYWRIGHT_WORKERS=4") {
		t.Fatalf("expected browser env to add default workers when unset, got %#v", env)
	}
	if strings.Contains(joined, "GOOS=js") || strings.Contains(joined, "GOARCH=wasm") {
		t.Fatalf("expected browser env to strip wasm variables, got %#v", env)
	}
}

func TestNPMCommandNameUsesWindowsExecutableOnWindows(t *testing.T) {
	if got := npmCommandName(); got != "npm.cmd" {
		t.Fatalf("expected Windows npm command name, got %q", got)
	}
}

func TestJoinHostPortDefaultsMissingValues(t *testing.T) {
	if got := joinHostPort("", ""); got != defaultHost+":"+defaultPort {
		t.Fatalf("expected default host and port, got %q", got)
	}
	if got := joinHostPort("127.0.0.1", ""); got != "127.0.0.1:"+defaultPort {
		t.Fatalf("expected explicit host with default port, got %q", got)
	}
	if got := joinHostPort("", "8123"); got != defaultHost+":8123" {
		t.Fatalf("expected default host with explicit port, got %q", got)
	}
}

func TestApplyDevHeadersSetsWasmAndCacheHeaders(t *testing.T) {
	wasmRecorder := httptest.NewRecorder()
	wasmRequest := httptest.NewRequest(http.MethodGet, "/static/app.wasm", nil)
	applyDevHeaders(wasmRecorder, wasmRequest)
	if got := wasmRecorder.Header().Get("Content-Type"); got != "application/wasm" {
		t.Fatalf("expected wasm content type, got %q", got)
	}
	if got := wasmRecorder.Header().Get("Cache-Control"); got != "no-store, no-cache, must-revalidate" {
		t.Fatalf("expected no-store cache header, got %q", got)
	}

	htmlRecorder := httptest.NewRecorder()
	htmlRequest := httptest.NewRequest(http.MethodGet, "/index.html", nil)
	applyDevHeaders(htmlRecorder, htmlRequest)
	if got := htmlRecorder.Header().Get("Content-Type"); got != "" {
		t.Fatalf("expected non-wasm content type to remain unset, got %q", got)
	}
	if got := htmlRecorder.Header().Get("Cache-Control"); got != "no-store, no-cache, must-revalidate" {
		t.Fatalf("expected cache header for non-wasm response, got %q", got)
	}
}

func TestExamplesCatalogOmitsUnavailableWasmArtifacts(t *testing.T) {
	repoRoot, err := resolveRepoRoot()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	launcher := launcher{
		repoRoot:    repoRoot,
		examplesDir: filepath.Join(repoRoot, "examples"),
		staticDir:   filepath.Join(repoRoot, "examples", "static"),
	}
	catalog, err := launcher.buildExamplesCatalog()
	if err != nil {
		t.Fatalf("build examples catalog: %v", err)
	}
	for _, entry := range catalog.Examples {
		if entry.Name != "88-web-components" {
			continue
		}
		if entry.UsesWasm {
			t.Fatalf("expected 88-web-components to stop advertising an unavailable wasm artifact, got %#v", entry)
		}
		if entry.WasmBinary != "" {
			t.Fatalf("expected 88-web-components wasmBinary to be empty when the artifact is unavailable, got %#v", entry)
		}
		return
	}
	t.Fatalf("expected 88-web-components entry to appear in catalog")
}

func TestExamplesCatalogJSONReportsWasmAndMultiClientEntries(t *testing.T) {
	repoRoot, err := resolveRepoRoot()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	launcher := launcher{
		repoRoot:    repoRoot,
		examplesDir: filepath.Join(repoRoot, "examples"),
		staticDir:   filepath.Join(repoRoot, "examples", "static"),
	}
	handler := launcher.newExamplesHandler("127.0.0.1", "8090")
	request := httptest.NewRequest(http.MethodGet, "/examples/catalog.json", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected catalog endpoint to succeed, got %d with body %s", recorder.Code, recorder.Body.String())
	}

	var payload examplesCatalogPayload
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode catalog payload: %v", err)
	}
	if payload.TotalExamples == 0 || payload.WasmExamples == 0 {
		t.Fatalf("expected non-empty catalog summary, got %#v", payload)
	}

	var foundPresence bool
	var foundUI bool
	for _, entry := range payload.Examples {
		if entry.Name == "97-multi-client-presence" {
			foundPresence = true
			if !entry.MultiClient {
				t.Fatalf("expected multi-client presence example to be marked multi-client")
			}
			if !entry.UsesWasm || entry.WasmBinary == "" {
				t.Fatalf("expected multi-client presence example to advertise wasm binary, got %#v", entry)
			}
		}
		if entry.Name == "21-ui-render" {
			foundUI = true
			if !hasAnyTag(entry.Tags, "ui") {
				t.Fatalf("expected ui-render example to expose ui tag, got %#v", entry)
			}
		}
	}
	if !foundPresence {
		t.Fatalf("expected multi-client presence example to appear in catalog payload")
	}
	if !foundUI {
		t.Fatalf("expected ui-render example to appear in catalog payload")
	}
}

func TestStaticExamplesCatalogUsesHTMLEntrypointsForStaticHosting(t *testing.T) {
	repoRoot, err := resolveRepoRoot()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	launcher := launcher{
		repoRoot:    repoRoot,
		examplesDir: filepath.Join(repoRoot, "examples"),
		staticDir:   filepath.Join(repoRoot, "examples", "static"),
	}
	catalog, err := launcher.buildStaticExamplesCatalog()
	if err != nil {
		t.Fatalf("build static examples catalog: %v", err)
	}

	var foundCounter bool
	var foundHTMLForms bool
	for _, entry := range catalog.Examples {
		if entry.Name != "01-counter" {
			if entry.Name == "53-html-forms" {
				foundHTMLForms = true
				if !hasAnyTag(entry.Tags, "html", "forms") {
					t.Fatalf("expected html-forms example to expose html/forms tags, got %#v", entry)
				}
			}
			continue
		}
		foundCounter = true
		if entry.Href != "../01-counter/counter.html" {
			t.Fatalf("expected static catalog href to target html entrypoint, got %q", entry.Href)
		}
		if !hasAnyTag(entry.Tags, "ui") {
			t.Fatalf("expected counter example to expose ui tag, got %#v", entry)
		}
	}
	if !foundCounter {
		t.Fatalf("expected 01-counter entry in static catalog")
	}
	if !foundHTMLForms {
		t.Fatalf("expected 53-html-forms entry in static catalog")
	}
}

func TestExamplesRouteGeneratesWasmHostPage(t *testing.T) {
	repoRoot, err := resolveRepoRoot()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	launcher := launcher{
		repoRoot:    repoRoot,
		examplesDir: filepath.Join(repoRoot, "examples"),
		staticDir:   filepath.Join(repoRoot, "examples", "static"),
	}
	handler := launcher.newExamplesHandler("127.0.0.1", "8090")
	request := httptest.NewRequest(http.MethodGet, "/examples/01-counter/", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected generated clean route to succeed, got %d with body %s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	for _, expected := range []string{"<div id=\"app\"></div>", "/static/bin/counter.wasm", "Go();", "gwc-examples-runtime-v1", "loadCachedWasm", "__GWC_BOOTSTRAP__", "01-counter"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected generated example page to contain %q, got %s", expected, body)
		}
	}
	for _, unexpected := range []string{"&#39;caches&#39;", "&#39;/static/bin/counter.wasm&#39;", "result =&gt; go.run"} {
		if strings.Contains(body, unexpected) {
			t.Fatalf("expected generated example page loader script to remain raw JavaScript, found escaped fragment %q", unexpected)
		}
	}
}

func TestExamplesRootServesAppShellWithoutRedirect(t *testing.T) {
	repoRoot, err := resolveRepoRoot()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	launcher := launcher{
		repoRoot:    repoRoot,
		examplesDir: filepath.Join(repoRoot, "examples"),
		staticDir:   filepath.Join(repoRoot, "examples", "static"),
	}
	handler := launcher.newExamplesHandler("127.0.0.1", "8090")
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected root route to serve app shell, got %d with body %s", recorder.Code, recorder.Body.String())
	}
	if location := recorder.Header().Get("Location"); location != "" {
		t.Fatalf("expected root route to avoid redirects, got location %q", location)
	}
	body := recorder.Body.String()
	for _, expected := range []string{"<div id=\"app\"></div>", "\"catalogHref\":\"/\"", "\"path\":\"/\""} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected root app shell to contain %q, got %s", expected, body)
		}
	}
}

func TestExamplesCatalogRouteServesAppShellWithoutRedirect(t *testing.T) {
	repoRoot, err := resolveRepoRoot()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	launcher := launcher{
		repoRoot:    repoRoot,
		examplesDir: filepath.Join(repoRoot, "examples"),
		staticDir:   filepath.Join(repoRoot, "examples", "static"),
	}
	handler := launcher.newExamplesHandler("127.0.0.1", "8090")
	request := httptest.NewRequest(http.MethodGet, "/examples/", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected /examples/ route to serve app shell, got %d with body %s", recorder.Code, recorder.Body.String())
	}
	if location := recorder.Header().Get("Location"); location != "" {
		t.Fatalf("expected /examples/ route to avoid redirects, got location %q", location)
	}
	body := recorder.Body.String()
	for _, expected := range []string{"<div id=\"app\"></div>", "\"catalogHref\":\"/examples/\"", "\"path\":\"/examples/\""} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected /examples/ app shell to contain %q, got %s", expected, body)
		}
	}
}

func TestExamplesRouteRedirectsToTrailingSlash(t *testing.T) {
	repoRoot, err := resolveRepoRoot()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	launcher := launcher{
		repoRoot:    repoRoot,
		examplesDir: filepath.Join(repoRoot, "examples"),
		staticDir:   filepath.Join(repoRoot, "examples", "static"),
	}
	handler := launcher.newExamplesHandler("127.0.0.1", "8090")
	request := httptest.NewRequest(http.MethodGet, "/examples/01-counter", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusFound {
		t.Fatalf("expected clean example route redirect, got %d with body %s", recorder.Code, recorder.Body.String())
	}
	if location := recorder.Header().Get("Location"); location != "/examples/01-counter/" {
		t.Fatalf("expected redirect to trailing slash route, got %q", location)
	}
}

func TestExamplesHTMLRouteRedirectsToCleanRoute(t *testing.T) {
	repoRoot, err := resolveRepoRoot()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	launcher := launcher{
		repoRoot:    repoRoot,
		examplesDir: filepath.Join(repoRoot, "examples"),
		staticDir:   filepath.Join(repoRoot, "examples", "static"),
	}
	handler := launcher.newExamplesHandler("127.0.0.1", "8090")
	request := httptest.NewRequest(http.MethodGet, "/examples/01-counter/counter.html", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusFound {
		t.Fatalf("expected html route redirect, got %d with body %s", recorder.Code, recorder.Body.String())
	}
	if location := recorder.Header().Get("Location"); location != "/examples/01-counter/" {
		t.Fatalf("expected redirect to clean example route, got %q", location)
	}
}

func TestExamplesNonHTMLAssetRoutePassesThrough(t *testing.T) {
	repoRoot, err := resolveRepoRoot()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	launcher := launcher{
		repoRoot:    repoRoot,
		examplesDir: filepath.Join(repoRoot, "examples"),
		staticDir:   filepath.Join(repoRoot, "examples", "static"),
	}
	handler := launcher.newExamplesHandler("127.0.0.1", "8090")
	request := httptest.NewRequest(http.MethodGet, "/examples/97-pwa-offline-cache/sw.js", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected sw.js route to pass through, got %d with body %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "OFFLINE_URL") {
		t.Fatalf("expected service worker contents to be served, got %s", recorder.Body.String())
	}
}

func TestDescribeDevPlanIncludesResolvedModesAndURL(t *testing.T) {
	plan := describeDevPlan(devConfig{
		appPath:  `C:\repo\app\main.go`,
		rootPath: `C:\repo\app`,
		host:     "127.0.0.1",
		port:     "8123",
	})
	if plan.ProjectRoot != `C:\repo\app` {
		t.Fatalf("expected project root to be preserved, got %q", plan.ProjectRoot)
	}
	if plan.AppMode != "client-only-wasm" {
		t.Fatalf("expected client-only app mode, got %q", plan.AppMode)
	}
	if plan.ServerMode != "livereload-wasm" {
		t.Fatalf("expected livereload server mode, got %q", plan.ServerMode)
	}
	if plan.ListeningURL != "http://127.0.0.1:8123" {
		t.Fatalf("expected listening URL, got %q", plan.ListeningURL)
	}

	serverPlan := describeDevPlan(devConfig{
		appPath: `C:\repo\app\cmd\web\main.go`,
		host:    "127.0.0.1",
		port:    "8124",
	})
	if serverPlan.AppMode != "server-app" || serverPlan.ServerMode != "server-entrypoint" {
		t.Fatalf("expected server mode classification, got %#v", serverPlan)
	}
}

func TestPrintDevPlanJSONIncludesResolvedSummaryFields(t *testing.T) {
	stdout, restoreStdout, err := captureExamplesStdout()
	if err != nil {
		t.Fatalf("capture stdout: %v", err)
	}
	defer restoreStdout()

	err = printDevPlanJSON(devConfig{
		appPath:  `C:\repo\app\main.go`,
		rootPath: `C:\repo\app`,
		htmlPath: `C:\repo\app\index.html`,
		wasmPath: "main.wasm",
		host:     "127.0.0.1",
		port:     "8125",
		hot:      true,
	})
	if err != nil {
		t.Fatalf("print plan json: %v", err)
	}

	output, err := stdout()
	if err != nil {
		t.Fatalf("read captured stdout: %v", err)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v\n%s", err, output)
	}
	for key, expected := range map[string]string{
		"projectRoot":  `C:\repo\app`,
		"appMode":      "client-only-wasm",
		"serverMode":   "livereload-wasm",
		"listeningURL": "http://127.0.0.1:8125",
	} {
		if payload[key] != expected {
			t.Fatalf("expected %s to be %q, got %#v", key, expected, payload[key])
		}
	}
}

func TestResolveDevConfigPrefersScaffoldMetadata(t *testing.T) {
	tempApp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tempApp, "main.go"), []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempApp, "index.html"), []byte("<html></html>\n"), 0644); err != nil {
		t.Fatalf("write index.html: %v", err)
	}
	metadata := `{
  "projectName": "metadata-dev-app",
  "modulePath": "example.com/metadata-dev-app",
  "tooling": {
    "appPath": "main.go",
    "htmlPath": "index.html",
    "wasmPath": "build/app.wasm",
    "devHost": "0.0.0.0",
    "devPort": "8140"
  }
}
`
	if err := os.WriteFile(filepath.Join(tempApp, "gwc-start.json"), []byte(metadata), 0644); err != nil {
		t.Fatalf("write gwc-start.json: %v", err)
	}

	originalGetwd := devGetwd
	t.Cleanup(func() { devGetwd = originalGetwd })
	devGetwd = func() (string, error) { return tempApp, nil }

	launcher := launcher{}
	config, err := launcher.resolveDevConfig(devConfig{})
	if err != nil {
		t.Fatalf("resolve dev config: %v", err)
	}
	if config.appPath != filepath.Join(tempApp, "main.go") {
		t.Fatalf("expected metadata app path, got %#v", config)
	}
	if config.rootPath != tempApp {
		t.Fatalf("expected metadata root path, got %#v", config)
	}
	if config.htmlPath != filepath.Join(tempApp, "index.html") {
		t.Fatalf("expected metadata html path, got %#v", config)
	}
	if config.wasmPath != "build/app.wasm" {
		t.Fatalf("expected metadata wasm path, got %#v", config)
	}
	if config.host != "0.0.0.0" || config.port != "8140" {
		t.Fatalf("expected metadata host/port, got %#v", config)
	}
}

func TestBuildDoctorReportPassesWithHealthyTooling(t *testing.T) {
	tempRepo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tempRepo, "test", "node_modules", "@playwright", "test"), 0755); err != nil {
		t.Fatalf("create playwright dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempRepo, "test", "package.json"), []byte("{}\n"), 0644); err != nil {
		t.Fatalf("write test package.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempRepo, "test", "node_modules", "@playwright", "test", "package.json"), []byte("{}\n"), 0644); err != nil {
		t.Fatalf("write playwright package.json: %v", err)
	}

	tempApp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tempApp, "main.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempApp, "index.html"), []byte("<html></html>\n"), 0644); err != nil {
		t.Fatalf("write index.html: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempApp, "gwc-start.json"), []byte("{\n  \"projectName\": \"doctor-app\",\n  \"modulePath\": \"example.com/doctor-app\"\n}\n"), 0644); err != nil {
		t.Fatalf("write gwc-start.json: %v", err)
	}

	originalLookPath := doctorLookPath
	originalCommandOutput := doctorCommandOutput
	originalGetwd := doctorGetwd
	originalListen := doctorListen
	originalResolveWasmExec := doctorResolveWasmExec
	t.Cleanup(func() {
		doctorLookPath = originalLookPath
		doctorCommandOutput = originalCommandOutput
		doctorGetwd = originalGetwd
		doctorListen = originalListen
		doctorResolveWasmExec = originalResolveWasmExec
	})

	doctorLookPath = func(name string) (string, error) {
		return filepath.Join("C:\\tools", name), nil
	}
	doctorCommandOutput = func(name string, args ...string) (string, error) {
		switch name {
		case "go":
			return "go version go1.25.0 windows/amd64", nil
		case "node":
			return "v22.0.0", nil
		case "npm":
			return "10.0.0", nil
		default:
			return "ok", nil
		}
	}
	doctorGetwd = func() (string, error) { return tempApp, nil }
	doctorListen = func(network string, address string) (net.Listener, error) {
		return net.Listen("tcp", "127.0.0.1:0")
	}
	doctorResolveWasmExec = func() (string, error) { return filepath.Join("C:\\Go", "lib", "wasm", "wasm_exec.js"), nil }

	launcher := launcher{repoRoot: tempRepo}
	report := launcher.buildDoctorReport(doctorConfig{host: "127.0.0.1", port: "8123"})
	if !report.OK {
		t.Fatalf("expected healthy doctor report, got %#v", report)
	}
	statuses := map[string]string{}
	for _, check := range report.Checks {
		statuses[check.Name] = check.Status
	}
	for _, name := range []string{"Go toolchain", "Node.js", "npm", "wasm_exec.js", "Browser tests", "Scaffold metadata", "Project detection", "Port availability"} {
		if statuses[name] != "pass" {
			t.Fatalf("expected %s to pass, got %#v", name, statuses[name])
		}
	}
	if report.CWD != tempApp {
		t.Fatalf("expected cwd %q, got %q", tempApp, report.CWD)
	}
}

func TestBuildDoctorReportFailsWhenPortIsUnavailable(t *testing.T) {
	originalGetwd := doctorGetwd
	originalListen := doctorListen
	originalLookPath := doctorLookPath
	originalCommandOutput := doctorCommandOutput
	originalResolveWasmExec := doctorResolveWasmExec
	t.Cleanup(func() {
		doctorGetwd = originalGetwd
		doctorListen = originalListen
		doctorLookPath = originalLookPath
		doctorCommandOutput = originalCommandOutput
		doctorResolveWasmExec = originalResolveWasmExec
	})

	doctorGetwd = func() (string, error) { return t.TempDir(), nil }
	doctorLookPath = func(name string) (string, error) { return name, nil }
	doctorCommandOutput = func(name string, args ...string) (string, error) { return name + " ok", nil }
	doctorResolveWasmExec = func() (string, error) { return "wasm_exec.js", nil }
	doctorListen = func(network string, address string) (net.Listener, error) {
		return nil, errors.New("address already in use")
	}

	launcher := launcher{repoRoot: t.TempDir()}
	report := launcher.buildDoctorReport(doctorConfig{host: "127.0.0.1", port: "8090"})
	if report.OK {
		t.Fatalf("expected failing report when port is unavailable, got %#v", report)
	}
	for _, check := range report.Checks {
		if check.Name == "Port availability" {
			if check.Status != "fail" {
				t.Fatalf("expected port availability to fail, got %#v", check)
			}
			return
		}
	}
	t.Fatal("expected port availability check to be present")
}

func TestBuildDoctorStandaloneChecksAdditionalBranches(t *testing.T) {
	originalResolveWasmExec := doctorResolveWasmExec
	t.Cleanup(func() { doctorResolveWasmExec = originalResolveWasmExec })
	doctorResolveWasmExec = func() (string, error) { return "", errors.New("missing wasm exec") }
	wasmCheck := buildDoctorWasmExecCheck()
	if wasmCheck.Status != "fail" || !strings.Contains(wasmCheck.Summary, "could not be resolved") {
		t.Fatalf("expected failing wasm_exec check, got %#v", wasmCheck)
	}

	root := t.TempDir()
	missingPackage := buildDoctorPlaywrightCheck(root)
	if missingPackage.Status != "warn" {
		t.Fatalf("expected missing browser package to warn, got %#v", missingPackage)
	}
	if err := os.MkdirAll(filepath.Join(root, "test"), 0755); err != nil {
		t.Fatalf("mkdir test dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "test", "package.json"), []byte("{}\n"), 0644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	missingDeps := buildDoctorPlaywrightCheck(root)
	if missingDeps.Status != "warn" || !strings.Contains(missingDeps.Summary, "not installed") {
		t.Fatalf("expected missing playwright deps warning, got %#v", missingDeps)
	}

	blankMetadata := buildDoctorMetadataCheck("")
	if blankMetadata.Status != "warn" {
		t.Fatalf("expected blank cwd metadata warning, got %#v", blankMetadata)
	}
	if err := os.WriteFile(filepath.Join(root, "gwc-start.json"), []byte(`{"tooling":`), 0644); err != nil {
		t.Fatalf("write invalid metadata: %v", err)
	}
	invalidMetadata := buildDoctorMetadataCheck(root)
	if invalidMetadata.Status != "fail" || !strings.Contains(invalidMetadata.Summary, "parse scaffold metadata") {
		t.Fatalf("expected invalid metadata failure, got %#v", invalidMetadata)
	}

	blankProjectDetection := buildDoctorProjectDetectionCheck("")
	if blankProjectDetection.Status != "warn" {
		t.Fatalf("expected blank cwd project detection warning, got %#v", blankProjectDetection)
	}
	missingProjectDetection := buildDoctorProjectDetectionCheck(t.TempDir())
	if missingProjectDetection.Status != "warn" {
		t.Fatalf("expected missing app project detection warning, got %#v", missingProjectDetection)
	}
}

func TestProjectHasGoTestsSkipsIgnoredDirectories(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "node_modules", "pkg"), 0755); err != nil {
		t.Fatalf("mkdir ignored dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "node_modules", "pkg", "ignored_test.go"), []byte("package pkg\n"), 0644); err != nil {
		t.Fatalf("write ignored test file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "main_test.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatalf("write root test file: %v", err)
	}

	hasTests, err := projectHasGoTests(root)
	if err != nil {
		t.Fatalf("project has go tests: %v", err)
	}
	if !hasTests {
		t.Fatal("expected root test file to be detected")
	}

	emptyRoot := t.TempDir()
	hasTests, err = projectHasGoTests(emptyRoot)
	if err != nil {
		t.Fatalf("project has go tests for empty root: %v", err)
	}
	if hasTests {
		t.Fatal("expected empty root to report no tests")
	}
}

func TestRunDoctorJSONEmitsMachineReadableReport(t *testing.T) {
	tempRepo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tempRepo, "test", "node_modules", "@playwright", "test"), 0755); err != nil {
		t.Fatalf("create playwright dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempRepo, "test", "package.json"), []byte("{}\n"), 0644); err != nil {
		t.Fatalf("write test package.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempRepo, "test", "node_modules", "@playwright", "test", "package.json"), []byte("{}\n"), 0644); err != nil {
		t.Fatalf("write playwright package.json: %v", err)
	}
	tempApp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tempApp, "main.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	stdout, restoreStdout, err := captureExamplesStdout()
	if err != nil {
		t.Fatalf("capture stdout: %v", err)
	}
	defer restoreStdout()

	originalLookPath := doctorLookPath
	originalCommandOutput := doctorCommandOutput
	originalGetwd := doctorGetwd
	originalListen := doctorListen
	originalResolveWasmExec := doctorResolveWasmExec
	t.Cleanup(func() {
		doctorLookPath = originalLookPath
		doctorCommandOutput = originalCommandOutput
		doctorGetwd = originalGetwd
		doctorListen = originalListen
		doctorResolveWasmExec = originalResolveWasmExec
	})
	doctorLookPath = func(name string) (string, error) { return name, nil }
	doctorCommandOutput = func(name string, args ...string) (string, error) { return name + " version", nil }
	doctorGetwd = func() (string, error) { return tempApp, nil }
	doctorResolveWasmExec = func() (string, error) { return "wasm_exec.js", nil }
	doctorListen = func(network string, address string) (net.Listener, error) {
		return net.Listen("tcp", "127.0.0.1:0")
	}

	launcher := launcher{repoRoot: tempRepo}
	if err := launcher.run([]string{"doctor", "-json", "-port", "8127"}); err != nil {
		t.Fatalf("run doctor json: %v", err)
	}

	output, err := stdout()
	if err != nil {
		t.Fatalf("read captured stdout: %v", err)
	}
	var report doctorReport
	if err := json.Unmarshal([]byte(output), &report); err != nil {
		t.Fatalf("unmarshal doctor report: %v\n%s", err, output)
	}
	if !report.OK {
		t.Fatalf("expected ok doctor report, got %#v", report)
	}
	if len(report.Checks) == 0 {
		t.Fatalf("expected doctor checks in JSON output, got %#v", report)
	}
}

func TestRunDoctorPrintsPassingReportWithoutError(t *testing.T) {
	tempRepo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tempRepo, "test", "node_modules", "@playwright", "test"), 0755); err != nil {
		t.Fatalf("create playwright dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempRepo, "test", "package.json"), []byte("{}\n"), 0644); err != nil {
		t.Fatalf("write test package.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempRepo, "test", "node_modules", "@playwright", "test", "package.json"), []byte("{}\n"), 0644); err != nil {
		t.Fatalf("write playwright package.json: %v", err)
	}
	tempApp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tempApp, "main.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	stdout, restoreStdout, err := captureExamplesStdout()
	if err != nil {
		t.Fatalf("capture stdout: %v", err)
	}
	defer restoreStdout()

	originalLookPath := doctorLookPath
	originalCommandOutput := doctorCommandOutput
	originalGetwd := doctorGetwd
	originalListen := doctorListen
	originalResolveWasmExec := doctorResolveWasmExec
	t.Cleanup(func() {
		doctorLookPath = originalLookPath
		doctorCommandOutput = originalCommandOutput
		doctorGetwd = originalGetwd
		doctorListen = originalListen
		doctorResolveWasmExec = originalResolveWasmExec
	})
	doctorLookPath = func(name string) (string, error) { return name, nil }
	doctorCommandOutput = func(name string, args ...string) (string, error) { return name + " version", nil }
	doctorGetwd = func() (string, error) { return tempApp, nil }
	doctorResolveWasmExec = func() (string, error) { return "wasm_exec.js", nil }
	doctorListen = func(network string, address string) (net.Listener, error) {
		return net.Listen("tcp", "127.0.0.1:0")
	}

	launcher := launcher{repoRoot: tempRepo}
	if err := launcher.runDoctor([]string{"-port", "8129"}); err != nil {
		t.Fatalf("run doctor: %v", err)
	}
	output, err := stdout()
	if err != nil {
		t.Fatalf("read captured stdout: %v", err)
	}
	for _, expected := range []string{"GWC doctor: PASS", "Go toolchain", "Port availability"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected doctor output to contain %q, got:\n%s", expected, output)
		}
	}
}

func TestRunDoctorReturnsErrorWhenChecksFail(t *testing.T) {
	originalLookPath := doctorLookPath
	originalGetwd := doctorGetwd
	t.Cleanup(func() {
		doctorLookPath = originalLookPath
		doctorGetwd = originalGetwd
	})

	doctorLookPath = func(name string) (string, error) {
		return "", errors.New("missing")
	}
	doctorGetwd = func() (string, error) { return t.TempDir(), nil }

	launcher := launcher{repoRoot: t.TempDir()}
	err := launcher.runDoctor([]string{"-port", "8131"})
	if err == nil {
		t.Fatal("expected doctor to fail when required checks fail")
	}
	if !strings.Contains(err.Error(), "doctor found required checks") {
		t.Fatalf("expected doctor failure summary, got %v", err)
	}
}

func TestRunDoctorHandlesHelpAndInvalidFlags(t *testing.T) {
	launcher := launcher{repoRoot: t.TempDir()}
	if err := launcher.runDoctor([]string{"-help"}); err != nil {
		t.Fatalf("expected doctor help to succeed, got %v", err)
	}
	err := launcher.runDoctor([]string{"-definitely-invalid"})
	if err == nil || !strings.Contains(err.Error(), "flag provided but not defined") {
		t.Fatalf("expected invalid doctor flag error, got %v", err)
	}
}

func TestRunExamplesDoesNotPrintListeningURLsWhenBindFails(t *testing.T) {
	tempExamples := t.TempDir()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	stdout, restoreStdout, err := captureExamplesStdout()
	if err != nil {
		t.Fatalf("capture stdout: %v", err)
	}
	defer restoreStdout()

	launcher := launcher{examplesDir: tempExamples, staticDir: tempExamples}
	err = launcher.runExamples([]string{"-host", "127.0.0.1", "-port", strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)})
	if err == nil {
		t.Fatal("expected bind failure")
	}

	output, readErr := stdout()
	if readErr != nil {
		t.Fatalf("read captured stdout: %v", readErr)
	}
	if strings.Contains(output, "GWC examples server listening on") || strings.Contains(output, "Examples: http://") {
		t.Fatalf("expected bind failure to avoid misleading listening output, got %q", output)
	}
	if !strings.Contains(err.Error(), "bind") {
		t.Fatalf("expected bind error, got %v", err)
	}
}

func TestRunExamplesPrintsListeningURLsOnServe(t *testing.T) {
	tempExamples := t.TempDir()
	stdout, restoreStdout, err := captureExamplesStdout()
	if err != nil {
		t.Fatalf("capture stdout: %v", err)
	}
	defer restoreStdout()

	originalExamplesListen := examplesListen
	originalExamplesServe := examplesServe
	t.Cleanup(func() {
		examplesListen = originalExamplesListen
		examplesServe = originalExamplesServe
	})

	realListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	examplesListen = func(network string, address string) (net.Listener, error) {
		return realListener, nil
	}
	examplesServe = func(server *http.Server, listener net.Listener) error {
		if server == nil || listener == nil {
			t.Fatal("expected server and listener to be provided")
		}
		if !strings.Contains(server.Addr, "127.0.0.1:") {
			t.Fatalf("expected server addr to include provided host, got %q", server.Addr)
		}
		if server.Handler == nil {
			t.Fatal("expected examples handler to be configured")
		}
		return http.ErrServerClosed
	}

	launcher := launcher{examplesDir: tempExamples, staticDir: tempExamples}
	if err := launcher.runExamples([]string{"-host", "127.0.0.1", "-port", strconv.Itoa(realListener.Addr().(*net.TCPAddr).Port)}); err != nil {
		t.Fatalf("run examples: %v", err)
	}
	output, err := stdout()
	if err != nil {
		t.Fatalf("read captured stdout: %v", err)
	}
	for _, expected := range []string{"GWC examples server listening on http://127.0.0.1:", "Examples: http://127.0.0.1:", "Counter:  http://127.0.0.1:"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected examples output to contain %q, got %q", expected, output)
		}
	}
}

func TestRunExamplesTreatsServerClosedAsCleanShutdown(t *testing.T) {
	tempExamples := t.TempDir()
	stdout, restoreStdout, err := captureExamplesStdout()
	if err != nil {
		t.Fatalf("capture stdout: %v", err)
	}
	defer restoreStdout()

	originalExamplesListen := examplesListen
	originalExamplesServe := examplesServe
	t.Cleanup(func() {
		examplesListen = originalExamplesListen
		examplesServe = originalExamplesServe
	})

	realListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	examplesListen = func(network string, address string) (net.Listener, error) {
		return realListener, nil
	}
	examplesServe = func(server *http.Server, listener net.Listener) error {
		return http.ErrServerClosed
	}

	launcher := launcher{examplesDir: tempExamples, staticDir: tempExamples}
	if err := launcher.runExamples([]string{"-host", "127.0.0.1", "-port", strconv.Itoa(realListener.Addr().(*net.TCPAddr).Port)}); err != nil {
		t.Fatalf("expected clean shutdown, got %v", err)
	}
	output, err := stdout()
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	if !strings.Contains(output, "GWC examples server listening on") {
		t.Fatalf("expected listening output before clean shutdown, got %q", output)
	}
}

func TestRunExamplesHandlesHelpInvalidFlagAndServeError(t *testing.T) {
	launcher := launcher{examplesDir: t.TempDir(), staticDir: t.TempDir()}
	if err := launcher.runExamples([]string{"-help"}); err != nil {
		t.Fatalf("expected examples help to succeed, got %v", err)
	}
	err := launcher.runExamples([]string{"-definitely-invalid"})
	if err == nil || !strings.Contains(err.Error(), "flag provided but not defined") {
		t.Fatalf("expected invalid examples flag error, got %v", err)
	}

	originalExamplesListen := examplesListen
	originalExamplesServe := examplesServe
	t.Cleanup(func() {
		examplesListen = originalExamplesListen
		examplesServe = originalExamplesServe
	})

	realListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	examplesListen = func(network string, address string) (net.Listener, error) {
		return realListener, nil
	}
	examplesServe = func(server *http.Server, listener net.Listener) error {
		return errors.New("serve failed")
	}

	err = launcher.runExamples([]string{"-host", "127.0.0.1", "-port", strconv.Itoa(realListener.Addr().(*net.TCPAddr).Port)})
	if err == nil || !strings.Contains(err.Error(), "serve failed") {
		t.Fatalf("expected serve failure to bubble, got %v", err)
	}
}

func TestRunExamplesWritesStaticCatalogAndReportsOutputPath(t *testing.T) {
	root := t.TempDir()
	examplesDir := filepath.Join(root, "examples")
	staticDir := filepath.Join(root, "static")
	if err := os.MkdirAll(filepath.Join(examplesDir, "01-first"), 0755); err != nil {
		t.Fatalf("mkdir example dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(staticDir, "bin"), 0755); err != nil {
		t.Fatalf("mkdir static bin dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(examplesDir, "01-first", "index.html"), []byte("<title>First</title>"), 0644); err != nil {
		t.Fatalf("write html: %v", err)
	}

	stdout, restoreStdout, err := captureExamplesStdout()
	if err != nil {
		t.Fatalf("capture stdout: %v", err)
	}
	defer restoreStdout()

	targetPath := filepath.Join(root, "exports", "catalog.json")
	launcher := launcher{examplesDir: examplesDir, staticDir: staticDir}
	if err := launcher.runExamples([]string{"-export-static-catalog", targetPath}); err != nil {
		t.Fatalf("run examples export static catalog: %v", err)
	}
	if _, err := os.Stat(targetPath); err != nil {
		t.Fatalf("expected static catalog file to exist: %v", err)
	}
	output, err := stdout()
	if err != nil {
		t.Fatalf("read captured stdout: %v", err)
	}
	if !strings.Contains(output, "Wrote static examples catalog to "+targetPath) {
		t.Fatalf("expected export confirmation output, got %q", output)
	}
}

func TestRunExamplesErrorsWhenExamplesDirectoryIsMissing(t *testing.T) {
	launcher := launcher{examplesDir: filepath.Join(t.TempDir(), "missing")}
	err := launcher.runExamples(nil)
	if err == nil {
		t.Fatal("expected missing examples directory to fail")
	}
	if !strings.Contains(err.Error(), "examples directory not found") {
		t.Fatalf("expected missing examples directory error, got %v", err)
	}
}

func TestLauncherRunHandlesUsageHelpAndUnknownCommand(t *testing.T) {
	stdout, restoreStdout, err := captureExamplesStdout()
	if err != nil {
		t.Fatalf("capture stdout: %v", err)
	}
	defer restoreStdout()

	appLauncher := launcher{examplesDir: t.TempDir(), staticDir: t.TempDir()}
	for _, args := range [][]string{{}, {"help"}, {"-h"}, {"--help"}} {
		if err := appLauncher.run(args); err != nil {
			t.Fatalf("expected usage/help args %v to succeed, got %v", args, err)
		}
	}
	err = appLauncher.run([]string{"unknown-command"})
	if err == nil || !strings.Contains(err.Error(), `unknown command "unknown-command"`) {
		t.Fatalf("expected unknown command error, got %v", err)
	}
	output, err := stdout()
	if err != nil {
		t.Fatalf("read captured stdout: %v", err)
	}
	if count := strings.Count(output, "GWC launcher"); count < 4 {
		t.Fatalf("expected usage output for each help path, got count=%d output=%q", count, output)
	}
}

func TestLauncherRunDispatchesEachSubcommand(t *testing.T) {
	originalRunTestCommand := runTestCommand
	originalRunExamplesCommand := runExamplesCommand
	originalRunBuildCommand := runBuildCommand
	originalRunReleaseCommand := runReleaseCommand
	originalRunDevCommand := runDevCommand
	originalRunDoctorCommand := runDoctorCommand
	originalRunVerifyCommand := runVerifyCommand
	originalRunStartCommand := runStartCommand
	t.Cleanup(func() {
		runTestCommand = originalRunTestCommand
		runExamplesCommand = originalRunExamplesCommand
		runBuildCommand = originalRunBuildCommand
		runReleaseCommand = originalRunReleaseCommand
		runDevCommand = originalRunDevCommand
		runDoctorCommand = originalRunDoctorCommand
		runVerifyCommand = originalRunVerifyCommand
		runStartCommand = originalRunStartCommand
	})

	tests := []struct {
		name        string
		args        []string
		installStub func(t *testing.T, called *bool)
	}{
		{name: "test", args: []string{"test", "-json"}, installStub: func(t *testing.T, called *bool) {
			runTestCommand = func(l launcher, args []string) error {
				*called = true
				if fmt.Sprint(args) != fmt.Sprint([]string{"-json"}) {
					t.Fatalf("unexpected args: %#v", args)
				}
				return nil
			}
		}},
		{name: "examples", args: []string{"examples", "-port", "9000"}, installStub: func(t *testing.T, called *bool) {
			runExamplesCommand = func(l launcher, args []string) error {
				*called = true
				if fmt.Sprint(args) != fmt.Sprint([]string{"-port", "9000"}) {
					t.Fatalf("unexpected args: %#v", args)
				}
				return nil
			}
		}},
		{name: "build", args: []string{"build", "-json"}, installStub: func(t *testing.T, called *bool) {
			runBuildCommand = func(l launcher, args []string) error { *called = true; return nil }
		}},
		{name: "release", args: []string{"release", "-json"}, installStub: func(t *testing.T, called *bool) {
			runReleaseCommand = func(l launcher, args []string) error { *called = true; return nil }
		}},
		{name: "dev", args: []string{"dev", "-dry-run"}, installStub: func(t *testing.T, called *bool) {
			runDevCommand = func(l launcher, args []string) error { *called = true; return nil }
		}},
		{name: "doctor", args: []string{"doctor", "-json"}, installStub: func(t *testing.T, called *bool) {
			runDoctorCommand = func(l launcher, args []string) error { *called = true; return nil }
		}},
		{name: "verify", args: []string{"verify", "-json"}, installStub: func(t *testing.T, called *bool) {
			runVerifyCommand = func(l launcher, args []string) error { *called = true; return nil }
		}},
		{name: "start", args: []string{"start", "--help"}, installStub: func(t *testing.T, called *bool) {
			runStartCommand = func(l launcher, args []string) error { *called = true; return nil }
		}},
	}

	appLauncher := launcher{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runTestCommand = originalRunTestCommand
			runExamplesCommand = originalRunExamplesCommand
			runBuildCommand = originalRunBuildCommand
			runReleaseCommand = originalRunReleaseCommand
			runDevCommand = originalRunDevCommand
			runDoctorCommand = originalRunDoctorCommand
			runVerifyCommand = originalRunVerifyCommand
			runStartCommand = originalRunStartCommand

			called := false
			test.installStub(t, &called)
			if err := appLauncher.run(test.args); err != nil {
				t.Fatalf("run dispatch failed: %v", err)
			}
			if !called {
				t.Fatal("expected dispatch stub to be called")
			}
		})
	}
}

func TestMainHandlesResolveRepoRootRunAndSuccessPaths(t *testing.T) {
	originalResolveRepoRoot := mainResolveRepoRoot
	originalRunLauncher := mainRunLauncher
	originalExit := mainExit
	originalArgs := mainArgs
	originalPrintError := mainPrintError
	t.Cleanup(func() {
		mainResolveRepoRoot = originalResolveRepoRoot
		mainRunLauncher = originalRunLauncher
		mainExit = originalExit
		mainArgs = originalArgs
		mainPrintError = originalPrintError
	})

	type exitSignal struct{ code int }

	t.Run("resolve repo root failure", func(t *testing.T) {
		printed := ""
		mainResolveRepoRoot = func() (string, error) { return "", errors.New("no repo") }
		mainRunLauncher = func(l launcher, args []string) error {
			t.Fatal("expected run launcher not to be called")
			return nil
		}
		mainArgs = func() []string { return []string{"gwc"} }
		mainPrintError = func(err error) { printed = err.Error() }
		mainExit = func(code int) { panic(exitSignal{code: code}) }

		defer func() {
			recovered := recover()
			signal, ok := recovered.(exitSignal)
			if !ok || signal.code != 1 {
				t.Fatalf("expected exit code 1, got %#v", recovered)
			}
			if printed != "no repo" {
				t.Fatalf("expected printed resolveRepoRoot error, got %q", printed)
			}
		}()
		main()
	})

	t.Run("run failure", func(t *testing.T) {
		printed := ""
		mainResolveRepoRoot = func() (string, error) { return `C:\repo`, nil }
		mainArgs = func() []string { return []string{"gwc", "examples", "-port", "9999"} }
		mainPrintError = func(err error) { printed = err.Error() }
		mainRunLauncher = func(l launcher, args []string) error {
			if l.repoRoot != `C:\repo` || l.examplesDir != filepath.Join(`C:\repo`, "examples") || l.staticDir != filepath.Join(`C:\repo`, "examples", "static") {
				t.Fatalf("expected launcher to be initialized from repo root, got %#v", l)
			}
			if strings.Join(args, " ") != "examples -port 9999" {
				t.Fatalf("expected main args to be forwarded, got %#v", args)
			}
			return errors.New("run failed")
		}
		mainExit = func(code int) { panic(exitSignal{code: code}) }

		defer func() {
			recovered := recover()
			signal, ok := recovered.(exitSignal)
			if !ok || signal.code != 1 {
				t.Fatalf("expected exit code 1, got %#v", recovered)
			}
			if printed != "run failed" {
				t.Fatalf("expected printed run error, got %q", printed)
			}
		}()
		main()
	})

	t.Run("success", func(t *testing.T) {
		printed := false
		exited := false
		called := false
		mainResolveRepoRoot = func() (string, error) { return `C:\repo`, nil }
		mainArgs = func() []string { return []string{"gwc", "doctor"} }
		mainPrintError = func(err error) { printed = true }
		mainExit = func(code int) { exited = true }
		mainRunLauncher = func(l launcher, args []string) error {
			called = true
			if len(args) != 1 || args[0] != "doctor" {
				t.Fatalf("expected success args to be forwarded, got %#v", args)
			}
			return nil
		}

		main()
		if !called {
			t.Fatal("expected main to invoke launcher run on success")
		}
		if printed || exited {
			t.Fatalf("expected success path to avoid error printing and exit, printed=%t exited=%t", printed, exited)
		}
	})
}

func TestResolveRepoRootErrorBranches(t *testing.T) {
	originalCaller := resolveRepoRootCaller
	t.Cleanup(func() { resolveRepoRootCaller = originalCaller })

	resolveRepoRootCaller = func(skip int) (uintptr, string, int, bool) {
		return 0, "", 0, false
	}
	if got, err := resolveRepoRoot(); err == nil || !strings.Contains(err.Error(), "unable to resolve launcher source path") {
		t.Fatalf("expected caller failure, got path=%q err=%v", got, err)
	}

	fakeFile := filepath.Join(t.TempDir(), "tools", "gwc", "main.go")
	if err := os.MkdirAll(filepath.Dir(fakeFile), 0755); err != nil {
		t.Fatalf("mkdir fake source dir: %v", err)
	}
	if err := os.WriteFile(fakeFile, []byte("package main\n"), 0644); err != nil {
		t.Fatalf("write fake source file: %v", err)
	}
	resolveRepoRootCaller = func(skip int) (uintptr, string, int, bool) {
		return 0, fakeFile, 1, true
	}
	if got, err := resolveRepoRoot(); err == nil || !strings.Contains(err.Error(), "unable to resolve repo root") {
		t.Fatalf("expected missing go.mod error, got path=%q err=%v", got, err)
	}
}

func TestExamplesHealthzRouteIncludesLauncherMetadata(t *testing.T) {
	launcher := launcher{repoRoot: `C:\repo`, examplesDir: t.TempDir(), staticDir: t.TempDir()}
	handler := launcher.newExamplesHandler("127.0.0.1", "8090")
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected healthz to succeed, got %d with body %s", recorder.Code, recorder.Body.String())
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode healthz payload: %v", err)
	}
	for key, expected := range map[string]string{
		"service": "gowebcomponents-gwc-examples",
		"root":    `C:\repo`,
		"host":    "127.0.0.1",
		"port":    "8090",
	} {
		if payload[key] != expected {
			t.Fatalf("expected %s=%q, got %#v", key, expected, payload[key])
		}
	}
	if payload["ok"] != true {
		t.Fatalf("expected ok=true, got %#v", payload)
	}
}

func TestExamplesListRouteRendersFilteredCatalogHTML(t *testing.T) {
	root := t.TempDir()
	examplesDir := filepath.Join(root, "examples")
	staticDir := filepath.Join(root, "static")
	if err := os.MkdirAll(filepath.Join(examplesDir, "01-alpha"), 0755); err != nil {
		t.Fatalf("mkdir alpha dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(examplesDir, "02-beta"), 0755); err != nil {
		t.Fatalf("mkdir beta dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(examplesDir, "01-alpha", "index.html"), []byte("<title>Alpha</title>"), 0644); err != nil {
		t.Fatalf("write alpha html: %v", err)
	}
	if err := os.WriteFile(filepath.Join(examplesDir, "02-beta", "index.html"), []byte("<title>Beta</title>"), 0644); err != nil {
		t.Fatalf("write beta html: %v", err)
	}
	if err := os.MkdirAll(staticDir, 0755); err != nil {
		t.Fatalf("mkdir static dir: %v", err)
	}

	launcher := launcher{examplesDir: examplesDir, staticDir: staticDir}
	handler := launcher.newExamplesHandler("127.0.0.1", "8090")
	request := httptest.NewRequest(http.MethodGet, "/examples/list?q=beta", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected examples list to succeed, got %d with body %s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "Filtered examples for") || !strings.Contains(body, "/examples/02-beta/") {
		t.Fatalf("expected filtered examples list html, got %s", body)
	}
	if strings.Contains(body, "/examples/01-alpha/") {
		t.Fatalf("expected filtered examples list to omit alpha entry, got %s", body)
	}
}

func TestPrintHelpersEmitExpectedLauncherOutput(t *testing.T) {
	stdout, restoreStdout, err := captureExamplesStdout()
	if err != nil {
		t.Fatalf("capture stdout: %v", err)
	}
	defer restoreStdout()

	printUsage()
	printBuildSummary(buildSummary{
		Profile:     buildProfile{Name: "ci", Trimpath: true, Ldflags: "-s -w", BuildVCS: "false"},
		AppPath:     `C:\repo\app\main.go`,
		ProjectRoot: `C:\repo\app`,
		PackageDir:  `C:\repo\app`,
		OutputPath:  `C:\repo\app\main.wasm`,
		Bytes:       42,
		SHA256:      "abc123",
	})
	printReleaseSummary(releaseSummary{
		Profile:      buildProfile{Name: "release"},
		AppPath:      `C:\repo\app\main.go`,
		ProjectRoot:  `C:\repo\app`,
		PackageDir:   `C:\repo\app`,
		OutDir:       `C:\repo\dist`,
		ManifestPath: `C:\repo\dist\manifest.json`,
		Artifacts: map[string]releaseArtifactRecord{
			"gzip": {Path: "app.wasm.gz", Bytes: 10},
			"wasm": {Path: "app.wasm", Bytes: 20},
		},
	})
	printVerifySummary(verifySummary{
		AppPath:     `C:\repo\app\main.go`,
		ProjectRoot: `C:\repo\app`,
		Tests:       verifyTestSummary{Ran: true, Command: "go test", PackagePattern: "./..."},
		Build:       buildSummary{Profile: buildProfile{Name: "ci"}, OutputPath: `C:\repo\app\main.wasm`},
	})
	printVerifySummary(verifySummary{
		AppPath:     `C:\repo\app\main.go`,
		ProjectRoot: `C:\repo\app`,
		Tests:       verifyTestSummary{Skipped: true},
		Build:       buildSummary{Profile: buildProfile{Name: "ci"}, OutputPath: `C:\repo\app\main.wasm`},
	})
	printTestSummary(testSummary{
		AppPath:       `C:\repo\app\main.go`,
		ProjectRoot:   `C:\repo\app`,
		SelectedLanes: []string{"unit", "wasm"},
		Lanes: []testLaneSummary{
			{Name: "unit", OK: true, Summary: "Native Go tests passed."},
			{Name: "browser", Skipped: true, Summary: "No Playwright workspace was found."},
		},
	})
	printDoctorReport(doctorReport{
		OK:  false,
		CWD: `C:\repo`,
		Checks: []doctorCheck{
			{Name: "Go toolchain", Status: "pass", Summary: "go version go1.25.0"},
			{Name: "Port availability", Status: "fail", Summary: "busy", Hint: "pick another port"},
		},
	})

	output, err := stdout()
	if err != nil {
		t.Fatalf("read captured stdout: %v", err)
	}
	for _, expected := range []string{
		"GWC launcher",
		"GWC build",
		"ldflags:      -s -w",
		"artifact[gzip]: app.wasm.gz (10 bytes)",
		"artifact[wasm]: app.wasm (20 bytes)",
		"tests:        go test ./...",
		"tests:        skipped",
		"[ok] unit: Native Go tests passed.",
		"[skipped] browser: No Playwright workspace was found.",
		"GWC doctor: FAIL",
		"hint: pick another port",
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected output to contain %q, got:\n%s", expected, output)
		}
	}
	if strings.Index(output, "artifact[gzip]:") > strings.Index(output, "artifact[wasm]:") {
		t.Fatalf("expected release artifacts to be sorted alphabetically, got:\n%s", output)
	}
}

func TestRenderExamplesShellHTMLFallbackIncludesManifestNoscriptAndBodyData(t *testing.T) {
	document := examplesShellDocument{
		Title:             "Examples",
		Description:       "Fallback shell",
		BodyClass:         "example-shell",
		BodyData:          map[string]string{"mode": "fallback", "route": "/examples/"},
		ManifestHref:      "/static/manifest.webmanifest",
		WasmURL:           "/static/bin/app.wasm",
		FailureTitle:      "Examples failed",
		FailureMessage:    "Bootstrap failed",
		FailureHref:       "/examples/catalog.json",
		FailureLinkLabel:  "Open catalog",
		NoScriptMessage:   "Enable JavaScript.",
		NoScriptHref:      "/examples/catalog.json",
		NoScriptLinkLabel: "Static catalog",
	}
	html := renderExamplesShellHTMLFallback(document, `<script id="bootstrap"></script>`)
	for _, expected := range []string{
		`<!doctype html>`,
		`<link rel="manifest" href="/static/manifest.webmanifest">`,
		`<body class="example-shell"`,
		`data-mode="fallback"`,
		`data-route="/examples/"`,
		`<noscript>`,
		`Enable JavaScript.`,
		`<script id="bootstrap"></script>`,
		`/static/bin/app.wasm`,
		`Bootstrap failed`,
	} {
		if !strings.Contains(html, expected) {
			t.Fatalf("expected fallback html to contain %q, got %s", expected, html)
		}
	}
}

func TestRenderExamplesShellHTMLFallsBackWhenRenderToStringFails(t *testing.T) {
	originalRenderToString := renderExamplesToString
	originalRenderBootstrapData := renderExamplesUIBootstrapScript
	t.Cleanup(func() {
		renderExamplesToString = originalRenderToString
		renderExamplesUIBootstrapScript = originalRenderBootstrapData
	})

	renderExamplesToString = func(node ui.Node) (string, error) {
		return "", errors.New("render failed")
	}
	renderExamplesUIBootstrapScript = func(bootstrap ui.SSRBootstrap, nonce string) (string, error) {
		return `<script id="bootstrap"></script>`, nil
	}

	html := renderExamplesShellHTML(examplesShellDocument{
		Title:             "Fallback",
		Description:       "Fallback description",
		BodyClass:         "examples-body",
		BodyData:          map[string]string{"mode": "fallback"},
		RoutePath:         "/examples/fallback/",
		CatalogHref:       "/examples/",
		ExampleSlug:       "fallback",
		ManifestHref:      "./manifest.webmanifest",
		FailureTitle:      "Failed",
		NoScriptMessage:   "Enable JavaScript.",
		NoScriptHref:      "/examples/",
		NoScriptLinkLabel: "Back",
	})
	for _, expected := range []string{`<script id="bootstrap"></script>`, `manifest.webmanifest`, `Enable JavaScript.`, `data-mode="fallback"`} {
		if !strings.Contains(html, expected) {
			t.Fatalf("expected fallback render html to contain %q, got %s", expected, html)
		}
	}
}

func TestStatusWriterDefaultsStatusOnWrite(t *testing.T) {
	recorder := httptest.NewRecorder()
	writer := &statusWriter{ResponseWriter: recorder}
	if _, err := writer.Write([]byte("ok")); err != nil {
		t.Fatalf("write body: %v", err)
	}
	if writer.status != http.StatusOK {
		t.Fatalf("expected implicit status 200, got %d", writer.status)
	}
	writer.WriteHeader(http.StatusAccepted)
	if writer.status != http.StatusAccepted {
		t.Fatalf("expected explicit status to be preserved, got %d", writer.status)
	}
}

func TestPrintDevPlanCoversResolvedAndDefaultFields(t *testing.T) {
	stdout, restoreStdout, err := captureExamplesStdout()
	if err != nil {
		t.Fatalf("capture stdout: %v", err)
	}
	defer restoreStdout()

	printDevPlan(devConfig{
		appPath: `C:\repo\client\main.go`,
		host:    "127.0.0.1",
		port:    "8081",
		hot:     true,
	})
	printDevPlan(devConfig{
		appPath:  `C:\repo\cmd\web\main.go`,
		rootPath: `C:\repo`,
		htmlPath: `public\index.html`,
		wasmPath: `bin\app.wasm`,
		host:     "localhost",
		port:     "9090",
		hot:      false,
	})

	output, err := stdout()
	if err != nil {
		t.Fatalf("read captured stdout: %v", err)
	}
	for _, expected := range []string{
		"GWC dev plan",
		"project root:  C:\\repo\\client",
		"app mode:      client-only-wasm",
		"server mode:   livereload-wasm",
		"html:          <auto-detect skipped>",
		"wasm:          <livereload default>",
		"listening URL: http://127.0.0.1:8081",
		"project root:  C:\\repo",
		"app mode:      server-app",
		"server mode:   server-entrypoint",
		"html:          public\\index.html",
		"wasm:          bin\\app.wasm",
		"hot:           false",
		"listening URL: http://localhost:9090",
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected dev plan output to contain %q, got:\n%s", expected, output)
		}
	}
}

func TestBuildDoctorToolCheckFailureModes(t *testing.T) {
	originalLookPath := doctorLookPath
	originalCommandOutput := doctorCommandOutput
	t.Cleanup(func() {
		doctorLookPath = originalLookPath
		doctorCommandOutput = originalCommandOutput
	})

	doctorLookPath = func(string) (string, error) {
		return "", errors.New("missing")
	}
	missing := buildDoctorToolCheck("go", "Go toolchain", "version", "install go")
	if missing.Status != "fail" {
		t.Fatalf("expected missing tool check to fail, got %#v", missing)
	}
	if !strings.Contains(missing.Summary, "go was not found on PATH") {
		t.Fatalf("expected missing tool summary, got %#v", missing)
	}
	if missing.Hint != "install go" {
		t.Fatalf("expected missing tool hint to be preserved, got %#v", missing)
	}

	doctorLookPath = func(string) (string, error) {
		return `C:\Go\bin\go.exe`, nil
	}
	doctorCommandOutput = func(name string, args ...string) (string, error) {
		return "version command blocked", errors.New("exit status 1")
	}
	commandFailure := buildDoctorToolCheck("go", "Go toolchain", "version", "install go")
	if commandFailure.Status != "fail" {
		t.Fatalf("expected version command failure to report fail, got %#v", commandFailure)
	}
	for _, expected := range []string{"C:\\Go\\bin\\go.exe", "version command blocked"} {
		if !strings.Contains(commandFailure.Summary, expected) {
			t.Fatalf("expected command failure summary to contain %q, got %#v", expected, commandFailure)
		}
	}

	doctorCommandOutput = func(name string, args ...string) (string, error) {
		return "", errors.New("exit status 2")
	}
	emptyOutputFailure := buildDoctorToolCheck("go", "Go toolchain", "version", "install go")
	if emptyOutputFailure.Status != "fail" {
		t.Fatalf("expected empty output failure to report fail, got %#v", emptyOutputFailure)
	}
	if !strings.Contains(emptyOutputFailure.Summary, "exit status 2") {
		t.Fatalf("expected fallback error text in summary, got %#v", emptyOutputFailure)
	}
}

func TestRunExamplesHelpReturnsNil(t *testing.T) {
	launcher := launcher{examplesDir: t.TempDir(), staticDir: t.TempDir()}
	if err := launcher.run([]string{"examples", "-help"}); err != nil {
		t.Fatalf("expected examples help to succeed, got %v", err)
	}
}

func TestRunDoctorHelpReturnsNil(t *testing.T) {
	launcher := launcher{repoRoot: t.TempDir()}
	if err := launcher.run([]string{"doctor", "-help"}); err != nil {
		t.Fatalf("expected doctor help to succeed, got %v", err)
	}
}

func TestRunDevHelpReturnsNil(t *testing.T) {
	launcher := launcher{repoRoot: t.TempDir()}
	if err := launcher.run([]string{"dev", "-help"}); err != nil {
		t.Fatalf("expected dev help to succeed, got %v", err)
	}
}

func TestResolveGeneratedExamplePageReturnsFalseForNonWasmExample(t *testing.T) {
	repoRoot, err := resolveRepoRoot()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	launcher := launcher{
		repoRoot:    repoRoot,
		examplesDir: filepath.Join(repoRoot, "examples"),
		staticDir:   filepath.Join(repoRoot, "examples", "static"),
	}

	page, ok, err := launcher.resolveGeneratedExamplePage("/examples/88-web-components/")
	if err != nil {
		t.Fatalf("resolve generated example page: %v", err)
	}
	if ok {
		t.Fatalf("expected non-wasm example not to generate a wasm host page, got %#v", page)
	}
}

func TestResolveExampleCatalogEntryHandlesMissingAndNonDirectoryTargets(t *testing.T) {
	root := t.TempDir()
	examplesDir := filepath.Join(root, "examples")
	staticDir := filepath.Join(root, "static")
	if err := os.MkdirAll(examplesDir, 0755); err != nil {
		t.Fatalf("mkdir examples dir: %v", err)
	}
	if err := os.MkdirAll(staticDir, 0755); err != nil {
		t.Fatalf("mkdir static dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(examplesDir, "99-file"), []byte("not a directory"), 0644); err != nil {
		t.Fatalf("write marker file: %v", err)
	}

	launcher := launcher{examplesDir: examplesDir, staticDir: staticDir}
	entry, ok, err := launcher.resolveExampleCatalogEntry("missing")
	if err != nil {
		t.Fatalf("resolve missing catalog entry: %v", err)
	}
	if ok {
		t.Fatalf("expected missing example to return ok=false, got %#v", entry)
	}

	entry, ok, err = launcher.resolveExampleCatalogEntry("99-file")
	if err != nil {
		t.Fatalf("resolve non-directory catalog entry: %v", err)
	}
	if ok {
		t.Fatalf("expected non-directory example target to return ok=false, got %#v", entry)
	}
}

func TestExamplesHandlerReturnsJSONErrorsForBrokenCatalogSources(t *testing.T) {
	launcher := launcher{repoRoot: t.TempDir(), examplesDir: filepath.Join(t.TempDir(), "missing"), staticDir: t.TempDir()}
	handler := launcher.newExamplesHandler("127.0.0.1", "8090")

	for path, expectedError := range map[string]string{
		"/examples/list":         "examples_listing_failed",
		"/examples/catalog.json": "examples_catalog_failed",
	} {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("expected %s to fail with 500, got %d", path, recorder.Code)
		}
		if !strings.Contains(recorder.Body.String(), expectedError) {
			t.Fatalf("expected %s response to contain %q, got %s", path, expectedError, recorder.Body.String())
		}
	}
}

func TestExamplesHandlerPassThroughBranches(t *testing.T) {
	root := t.TempDir()
	examplesDir := filepath.Join(root, "examples")
	staticDir := filepath.Join(root, "static")
	if err := os.MkdirAll(filepath.Join(examplesDir, "assets"), 0755); err != nil {
		t.Fatalf("mkdir assets dir: %v", err)
	}
	if err := os.MkdirAll(staticDir, 0755); err != nil {
		t.Fatalf("mkdir static dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(examplesDir, "assets", "readme.txt"), []byte("example asset"), 0644); err != nil {
		t.Fatalf("write example asset: %v", err)
	}
	if err := os.WriteFile(filepath.Join(staticDir, "plain.txt"), []byte("static asset"), 0644); err != nil {
		t.Fatalf("write static asset: %v", err)
	}

	launcher := launcher{repoRoot: root, examplesDir: examplesDir, staticDir: staticDir}
	handler := launcher.newExamplesHandler("127.0.0.1", "8090")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/examples/assets/readme.txt", nil))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "example asset") {
		t.Fatalf("expected examples pass-through asset, got code=%d body=%s", recorder.Code, recorder.Body.String())
	}

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/static/plain.txt", nil))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "static asset") {
		t.Fatalf("expected static pass-through asset, got code=%d body=%s", recorder.Code, recorder.Body.String())
	}

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/not-root.txt", nil))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected root fallback pass-through 404 for unknown asset, got %d", recorder.Code)
	}
}

func TestExamplesHandlerExactAndErrorRoutes(t *testing.T) {
	repoRoot, err := resolveRepoRoot()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	realLauncher := launcher{
		repoRoot:    repoRoot,
		examplesDir: filepath.Join(repoRoot, "examples"),
		staticDir:   filepath.Join(repoRoot, "examples", "static"),
	}
	handler := realLauncher.newExamplesHandler("127.0.0.1", "8090")

	for _, path := range []string{"/examples", "/examples/static/index.html"} {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected %s to serve app shell, got %d with body %s", path, recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), `<div id="app"></div>`) {
			t.Fatalf("expected %s to render app shell, got %s", path, recorder.Body.String())
		}
	}

	brokenRoot := t.TempDir()
	brokenExamplesPath := filepath.Join(brokenRoot, "examples-file")
	if err := os.WriteFile(brokenExamplesPath, []byte("not a directory"), 0644); err != nil {
		t.Fatalf("write examples file: %v", err)
	}
	brokenLauncher := launcher{repoRoot: brokenRoot, examplesDir: brokenExamplesPath, staticDir: filepath.Join(brokenRoot, "static")}
	brokenHandler := brokenLauncher.newExamplesHandler("127.0.0.1", "8090")
	recorder := httptest.NewRecorder()
	brokenHandler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/examples/01-counter/", nil))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected file-backed examples root to fall through with 404, got code=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestExamplesHandlerAppliesHeadersToServedAssets(t *testing.T) {
	root := t.TempDir()
	examplesDir := filepath.Join(root, "examples")
	staticDir := filepath.Join(root, "static")
	if err := os.MkdirAll(filepath.Join(examplesDir, "01-counter"), 0755); err != nil {
		t.Fatalf("mkdir example dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(staticDir, "bin"), 0755); err != nil {
		t.Fatalf("mkdir static bin dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(examplesDir, "01-counter", "notes.txt"), []byte("example-notes"), 0644); err != nil {
		t.Fatalf("write example asset: %v", err)
	}
	if err := os.WriteFile(filepath.Join(staticDir, "bin", "app.wasm"), []byte("wasm-bytes"), 0644); err != nil {
		t.Fatalf("write wasm asset: %v", err)
	}

	handler := (launcher{repoRoot: root, examplesDir: examplesDir, staticDir: staticDir}).newExamplesHandler("127.0.0.1", "8090")

	exampleRecorder := httptest.NewRecorder()
	handler.ServeHTTP(exampleRecorder, httptest.NewRequest(http.MethodGet, "/examples/01-counter/notes.txt", nil))
	if exampleRecorder.Code != http.StatusOK {
		t.Fatalf("expected example asset to be served, got %d body=%s", exampleRecorder.Code, exampleRecorder.Body.String())
	}
	if got := exampleRecorder.Header().Get("Cache-Control"); got != "no-store, no-cache, must-revalidate" {
		t.Fatalf("expected cache header on example asset, got %q", got)
	}

	wasmRecorder := httptest.NewRecorder()
	handler.ServeHTTP(wasmRecorder, httptest.NewRequest(http.MethodGet, "/static/bin/app.wasm", nil))
	if wasmRecorder.Code != http.StatusOK {
		t.Fatalf("expected wasm asset to be served, got %d body=%s", wasmRecorder.Code, wasmRecorder.Body.String())
	}
	if got := wasmRecorder.Header().Get("Content-Type"); got != "application/wasm" {
		t.Fatalf("expected wasm content type, got %q", got)
	}
	if got := wasmRecorder.Header().Get("Cache-Control"); got != "no-store, no-cache, must-revalidate" {
		t.Fatalf("expected cache header on wasm asset, got %q", got)
	}
}

func TestExamplesHandlerFallsBackForEmptyAndMissingRoutes(t *testing.T) {
	repoRoot, err := resolveRepoRoot()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	handler := (launcher{
		repoRoot:    repoRoot,
		examplesDir: filepath.Join(repoRoot, "examples"),
		staticDir:   filepath.Join(repoRoot, "examples", "static"),
	}).newExamplesHandler("127.0.0.1", "8090")

	emptyRecorder := httptest.NewRecorder()
	handler.ServeHTTP(emptyRecorder, httptest.NewRequest(http.MethodGet, "/examples//", nil))
	if emptyRecorder.Code != http.StatusMovedPermanently {
		t.Fatalf("expected double-slash route to canonicalize, got %d body=%s", emptyRecorder.Code, emptyRecorder.Body.String())
	}
	if location := emptyRecorder.Header().Get("Location"); location != "/examples/" {
		t.Fatalf("expected canonical redirect to /examples/, got %q", location)
	}

	missingRecorder := httptest.NewRecorder()
	handler.ServeHTTP(missingRecorder, httptest.NewRequest(http.MethodGet, "/examples/does-not-exist/", nil))
	if missingRecorder.Code != http.StatusNotFound {
		t.Fatalf("expected missing example route to pass through as 404, got %d body=%s", missingRecorder.Code, missingRecorder.Body.String())
	}
}

func TestExamplesHandlerReachableMuxFallbackBranches(t *testing.T) {
	root := t.TempDir()
	examplesDir := filepath.Join(root, "examples")
	staticDir := filepath.Join(root, "static")
	if err := os.MkdirAll(filepath.Join(examplesDir, "nested"), 0755); err != nil {
		t.Fatalf("mkdir nested dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(staticDir, "bin"), 0755); err != nil {
		t.Fatalf("mkdir static bin dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(examplesDir, "nested", "info.txt"), []byte("nested asset"), 0644); err != nil {
		t.Fatalf("write nested asset: %v", err)
	}

	handler := (launcher{repoRoot: root, examplesDir: examplesDir, staticDir: staticDir}).newExamplesHandler("127.0.0.1", "8090")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/examples", nil))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `<div id="app"></div>`) {
		t.Fatalf("expected exact /examples route to render app shell, got code=%d body=%s", recorder.Code, recorder.Body.String())
	}

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/examples/static/index.html", nil))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"catalogHref":"/examples/"`) {
		t.Fatalf("expected static index route to render app shell, got code=%d body=%s", recorder.Code, recorder.Body.String())
	}

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/nested/info.txt", nil))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected root fallback to examples server 404, got code=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestExamplesHelperFunctionsAdditionalBranches(t *testing.T) {
	root := t.TempDir()
	noHTMLDir := filepath.Join(root, "no-html")
	if err := os.MkdirAll(noHTMLDir, 0755); err != nil {
		t.Fatalf("mkdir no-html dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(noHTMLDir, "readme.txt"), []byte("plain"), 0644); err != nil {
		t.Fatalf("write text file: %v", err)
	}
	if htmlFile, ok, err := firstHTMLFileName(noHTMLDir); err != nil || ok || htmlFile != "" {
		t.Fatalf("expected no-html directory to return no match, got file=%q ok=%t err=%v", htmlFile, ok, err)
	}

	if _, ok, err := (&launcher{examplesDir: root, staticDir: root}).buildExampleCatalogEntry(noHTMLDir, "no-html", nil); err != nil || ok {
		t.Fatalf("expected buildExampleCatalogEntry to skip directories without html, got ok=%t err=%v", ok, err)
	}

	if title, err := detectHTMLTitle(filepath.Join(noHTMLDir, "missing.html")); err == nil || title != "" {
		t.Fatalf("expected detectHTMLTitle read failure, got title=%q err=%v", title, err)
	}

	htmlPath := filepath.Join(root, "untitled.html")
	if err := os.WriteFile(htmlPath, []byte("<html><body>no title</body></html>"), 0644); err != nil {
		t.Fatalf("write html without title: %v", err)
	}
	if title, err := detectHTMLTitle(htmlPath); err != nil || title != "" {
		t.Fatalf("expected html without title to return empty title, got %q err=%v", title, err)
	}

	if got := defaultExampleTitle("example", ""); got != "Example - GoWebComponents" {
		t.Fatalf("expected empty html filename fallback title, got %q", got)
	}

	if wasm, ok, err := detectExampleWasmBinary(htmlPath); err != nil || ok || wasm != "" {
		t.Fatalf("expected html without wasm reference to return no wasm, got wasm=%q ok=%t err=%v", wasm, ok, err)
	}

	if _, _, err := detectExampleWasmBinary(filepath.Join(root, "missing.html")); err == nil {
		t.Fatal("expected missing html for wasm detection to fail")
	}

	if tag := renderExamplesLoaderScriptTag("/static/bin/app.wasm", "Title", "Message", "/fallback", "Retry"); !strings.HasPrefix(tag, "<script>") || !strings.Contains(tag, "loadCachedWasm") {
		t.Fatalf("expected loader script tag wrapper, got %q", tag)
	}
	if script := renderExamplesBootstrapDataScript(examplesShellDocument{RoutePath: "/examples/route/", CatalogHref: "/examples/", ExampleSlug: "route"}); !strings.Contains(script, "__GWC_BOOTSTRAP__") || !strings.Contains(script, "catalogURL") {
		t.Fatalf("expected bootstrap data script, got %q", script)
	}
	if _, ok, err := firstHTMLFileName(filepath.Join(root, "missing-dir")); err == nil || ok {
		t.Fatalf("expected missing directory html lookup to fail, got ok=%t err=%v", ok, err)
	}
	if got := defaultExampleTitle("fallback slug", "---"); got != "fallback slug" {
		t.Fatalf("expected punctuation-only html filename to fall back to dir name, got %q", got)
	}
}

func TestExamplesCatalogHelpersCoverGeneratedAndFallbackBranches(t *testing.T) {
	root := t.TempDir()
	examplesDir := filepath.Join(root, "examples")
	staticDir := filepath.Join(root, "static")
	exampleDir := filepath.Join(examplesDir, "97-multi-client-presence")
	if err := os.MkdirAll(filepath.Join(staticDir, "bin"), 0755); err != nil {
		t.Fatalf("mkdir static bin dir: %v", err)
	}
	if err := os.MkdirAll(exampleDir, 0755); err != nil {
		t.Fatalf("mkdir example dir: %v", err)
	}
	htmlPath := filepath.Join(exampleDir, "presence.html")
	html := "<html><head><title> Presence\n Multi Client </title></head><body><script src=\"/static/bin/presence.wasm\"></script></body></html>"
	if err := os.WriteFile(htmlPath, []byte(html), 0644); err != nil {
		t.Fatalf("write example html: %v", err)
	}
	if err := os.WriteFile(filepath.Join(staticDir, "bin", "presence.wasm"), []byte("wasm"), 0644); err != nil {
		t.Fatalf("write wasm artifact: %v", err)
	}

	launcher := launcher{examplesDir: examplesDir, staticDir: staticDir}
	entry, ok, err := launcher.buildExampleCatalogEntry(exampleDir, "97-multi-client-presence", func(dirPath string, dirName string, htmlFile string) string {
		return "/custom/" + dirName + "/" + htmlFile
	})
	if err != nil {
		t.Fatalf("build example catalog entry: %v", err)
	}
	if !ok {
		t.Fatal("expected example catalog entry to be built")
	}
	if entry.Href != "/custom/97-multi-client-presence/presence.html" {
		t.Fatalf("expected custom href, got %#v", entry)
	}
	if entry.Title != "Presence Multi Client" {
		t.Fatalf("expected normalized multiline title, got %#v", entry)
	}
	if !entry.UsesWasm || entry.WasmBinary != "presence.wasm" || !entry.MultiClient {
		t.Fatalf("expected wasm multi-client entry, got %#v", entry)
	}
	if !hasAnyTag(entry.Tags, "interop", "multi-client", "wasm") {
		t.Fatalf("expected multi-client tags, got %#v", entry)
	}

	page, ok, err := launcher.resolveGeneratedExamplePage("/examples/97-multi-client-presence/")
	if err != nil {
		t.Fatalf("resolve generated example page: %v", err)
	}
	if !ok || page.ManifestHref != "" || page.GeneratedFrom != "presence.html" {
		t.Fatalf("expected manifest-less generated example page, got %#v", page)
	}

	missingStaticDir := filepath.Join(root, "missing-static")
	if err := os.MkdirAll(missingStaticDir, 0755); err != nil {
		t.Fatalf("mkdir missing static dir: %v", err)
	}
	if wasmBinary, usesWasm, err := detectAvailableExampleWasmBinary(missingStaticDir, htmlPath); err != nil || usesWasm || wasmBinary != "" {
		t.Fatalf("expected missing static wasm binary to disable wasm usage, got wasm=%q usesWasm=%t err=%v", wasmBinary, usesWasm, err)
	}
}

func TestResolveExampleCatalogEntryAndHandlerSurfaceStatErrors(t *testing.T) {
	root := t.TempDir()
	staticDir := filepath.Join(root, "static")
	if err := os.MkdirAll(staticDir, 0755); err != nil {
		t.Fatalf("mkdir static dir: %v", err)
	}

	launcher := launcher{repoRoot: root, examplesDir: filepath.Join(root, "bad:examples"), staticDir: staticDir}
	if entry, ok, err := launcher.resolveExampleCatalogEntry("01-counter"); err == nil || ok {
		t.Fatalf("expected invalid example path to return stat error, got entry=%#v ok=%t err=%v", entry, ok, err)
	}

	handler := launcher.newExamplesHandler("127.0.0.1", "8090")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/examples/01-counter/", nil))
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected invalid example route to surface 500, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "examples_route_failed") {
		t.Fatalf("expected invalid example route error payload, got %s", recorder.Body.String())
	}
}

func TestExamplesBootstrapHelperErrorBranches(t *testing.T) {
	originalRenderBootstrapData := renderExamplesUIBootstrapScript
	originalBootstrapScriptFunc := renderExamplesBootstrapScriptFunc
	t.Cleanup(func() {
		renderExamplesUIBootstrapScript = originalRenderBootstrapData
		renderExamplesBootstrapScriptFunc = originalBootstrapScriptFunc
	})

	renderExamplesUIBootstrapScript = func(bootstrap ui.SSRBootstrap, nonce string) (string, error) {
		return "", errors.New("bootstrap failed")
	}
	if got := renderExamplesBootstrapDataScript(examplesShellDocument{RoutePath: "/examples/failure/", CatalogHref: "/examples/", ExampleSlug: "failure"}); got != "" {
		t.Fatalf("expected bootstrap data script failure to return empty string, got %q", got)
	}

	renderExamplesBootstrapScriptFunc = func(wasmURL string, failureTitle string, failureMessage string, failureHref string, failureLinkLabel string) string {
		return "   "
	}
	if got := renderExamplesLoaderScriptTag("/static/bin/app.wasm", "Title", "Message", "/fallback", "Retry"); got != "" {
		t.Fatalf("expected blank bootstrap script body to suppress script tag, got %q", got)
	}
}

func TestExamplesHandlerPassesThroughNonWasmExampleDirectory(t *testing.T) {
	root := t.TempDir()
	examplesDir := filepath.Join(root, "examples")
	staticDir := filepath.Join(root, "static")
	exampleDir := filepath.Join(examplesDir, "88-plain-html")
	if err := os.MkdirAll(exampleDir, 0755); err != nil {
		t.Fatalf("mkdir example dir: %v", err)
	}
	if err := os.MkdirAll(staticDir, 0755); err != nil {
		t.Fatalf("mkdir static dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(exampleDir, "index.html"), []byte("<html><body>plain example</body></html>"), 0644); err != nil {
		t.Fatalf("write plain html: %v", err)
	}

	handler := (launcher{repoRoot: root, examplesDir: examplesDir, staticDir: staticDir}).newExamplesHandler("127.0.0.1", "8090")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/examples/88-plain-html/", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected non-wasm example directory to pass through, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "plain example") || strings.Contains(body, "loadCachedWasm") {
		t.Fatalf("expected plain html passthrough without generated wasm shell, got %s", body)
	}
}

func TestExamplesRenderingDefaultsAndGeneratedManifest(t *testing.T) {
	if html := renderExamplesAppShellHTML("   ", "   "); !strings.Contains(html, `"catalogHref":"/examples/"`) || !strings.Contains(html, `"path":"/examples/"`) {
		t.Fatalf("expected app shell defaults for blank inputs, got %s", html)
	}

	root := t.TempDir()
	examplesDir := filepath.Join(root, "examples")
	staticDir := filepath.Join(root, "static")
	if err := os.MkdirAll(filepath.Join(examplesDir, "01-counter"), 0755); err != nil {
		t.Fatalf("mkdir example dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(staticDir, "bin"), 0755); err != nil {
		t.Fatalf("mkdir static bin dir: %v", err)
	}
	html := `<html><head><title>Counter</title></head><body><script src="/static/bin/counter.wasm"></script></body></html>`
	if err := os.WriteFile(filepath.Join(examplesDir, "01-counter", "index.html"), []byte(html), 0644); err != nil {
		t.Fatalf("write index.html: %v", err)
	}
	if err := os.WriteFile(filepath.Join(examplesDir, "01-counter", "manifest.webmanifest"), []byte("{}\n"), 0644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(staticDir, "bin", "counter.wasm"), []byte("wasm"), 0644); err != nil {
		t.Fatalf("write wasm: %v", err)
	}

	launcher := launcher{examplesDir: examplesDir, staticDir: staticDir}
	page, ok, err := launcher.resolveGeneratedExamplePage("/examples/01-counter/")
	if err != nil {
		t.Fatalf("resolve generated example page: %v", err)
	}
	if !ok || page.ManifestHref != "./manifest.webmanifest" {
		t.Fatalf("expected generated page manifest href, got %#v", page)
	}

	page, ok, err = launcher.resolveGeneratedExamplePage("   ")
	if err != nil {
		t.Fatalf("resolve blank generated page: %v", err)
	}
	if ok || page != (generatedExamplePage{}) {
		t.Fatalf("expected blank generated page path to skip, got %#v", page)
	}
}

func captureExamplesStdout() (func() (string, error), func(), error) {
	originalStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		return nil, nil, err
	}
	os.Stdout = writer

	readOutput := func() (string, error) {
		if err := writer.Close(); err != nil {
			return "", err
		}
		bytes, err := io.ReadAll(reader)
		if err != nil {
			return "", err
		}
		return string(bytes), nil
	}
	restore := func() {
		os.Stdout = originalStdout
		_ = writer.Close()
		_ = reader.Close()
	}
	return readOutput, restore, nil
}

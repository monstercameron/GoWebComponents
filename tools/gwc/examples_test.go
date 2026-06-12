package main

import (
	"bytes"
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

func stageExampleWasmFixtures(parseT *testing.T, parseBinaryNames ...string) string {
	parseT.Helper()
	parseWasmDir := parseT.TempDir()
	for _, parseBinaryName := range parseBinaryNames {
		if strings.TrimSpace(parseBinaryName) == "" {
			continue
		}
		if parseErr := os.WriteFile(filepath.Join(parseWasmDir, parseBinaryName), []byte("wasm"), 0644); parseErr != nil {
			parseT.Fatalf("write wasm fixture %q: %v", parseBinaryName, parseErr)
		}
	}
	return parseWasmDir
}

func TestFilterExampleLinksMatchesKeywordsAcrossNameAndHref(parseT *testing.T) {
	parseLinks := []exampleLink{
		{Name: "01-counter", Href: "/examples/public/counter/"},
		{Name: "17-ssr-routing", Href: "/examples/public/static-server-side-rendering-routing/"},
		{Name: "86-atlas-commerce-os", Href: "/examples/server/atlas-commerce-os/"},
	}

	parseGot := filterExampleLinks(parseLinks, "atlas commerce")
	if len(parseGot) != 1 {
		parseT.Fatalf("expected one atlas match, got %d: %#v", len(parseGot), parseGot)
	}
	if parseGot[0].Name != "86-atlas-commerce-os" {
		parseT.Fatalf("expected atlas example match, got %q", parseGot[0].Name)
	}

	parseGot = filterExampleLinks(parseLinks, "17 examples")
	if len(parseGot) != 1 || parseGot[0].Name != "17-ssr-routing" {
		parseT.Fatalf("expected ssr-routing match from name and href, got %#v", parseGot)
	}

	parseGot = filterExampleLinks(parseLinks, "")
	if len(parseGot) != len(parseLinks) {
		parseT.Fatalf("expected empty query to return all links, got %d", len(parseGot))
	}
}

func TestRenderExamplesListingHTMLIncludesSearchState(parseT *testing.T) {
	parseHtml := renderExamplesListingHTML([]exampleLink{{Name: "01-counter", Href: "/examples/public/counter/"}}, `atlas "search"`)
	for _, parseExpected := range []string{"name=\"q\"", "Filtered examples for", "atlas &quot;search&quot;", "01-counter"} {
		if !strings.Contains(parseHtml, parseExpected) {
			parseT.Fatalf("expected examples HTML to contain %q", parseExpected)
		}
	}
}

func TestRenderExamplesListingHTMLShowsNoMatchesState(parseT *testing.T) {
	parseHtml := renderExamplesListingHTML(nil, "nomatch")
	if !strings.Contains(parseHtml, "No examples matched this search yet.") {
		parseT.Fatalf("expected no-match helper text, got %s", parseHtml)
	}
}

func TestRenderExamplesAppShellHTMLIncludesWasmCatalogBootstrap(parseT *testing.T) {
	parseHtml := renderExamplesAppShellHTML("/", "/")
	for _, parseExpected := range []string{"gwc-examples-site.wasm", "/examples/list", "GoWebComponents Examples", "gwc-examples-runtime-v1", "loadCachedWasm", "wasm source:", "__GWC_BOOTSTRAP__", "catalogURL"} {
		if !strings.Contains(parseHtml, parseExpected) {
			parseT.Fatalf("expected app shell HTML to contain %q", parseExpected)
		}
	}
	for _, parseExpected2 := range []string{"\"catalogHref\":\"/\"", "\"path\":\"/\""} {
		if !strings.Contains(parseHtml, parseExpected2) {
			parseT.Fatalf("expected root app shell HTML to contain %q", parseExpected2)
		}
	}
	for _, parseUnexpected := range []string{"&#39;caches&#39;", "&#39;/static/bin/gwc-examples-site.wasm&#39;", "result =&gt; go.run"} {
		if strings.Contains(parseHtml, parseUnexpected) {
			parseT.Fatalf("expected app shell loader script to remain raw JavaScript, found escaped fragment %q", parseUnexpected)
		}
	}
}

func TestStaticExamplesShellIncludesBootstrapContract(parseT *testing.T) {
	parseRepoRoot, parseErr := resolveRepoRoot()
	if parseErr != nil {
		parseT.Fatalf("resolve repo root: %v", parseErr)
	}
	parseStaticShellPath := filepath.Join(parseRepoRoot, "examples", "static", "index.html")
	parseContent, parseErr := os.ReadFile(parseStaticShellPath)
	if parseErr != nil {
		parseT.Fatalf("read static shell: %v", parseErr)
	}
	parseHtml := string(parseContent)
	for _, parseExpected := range []string{"__GWC_BOOTSTRAP__", "\"mode\":\"static\"", "\"catalogURL\":\"catalog.json\"", "\"wasmBase\":\"bin/\"", "\"catalogHref\":\"./index.html#/examples\"", "gwc-examples-site.wasm", "wasm source:"} {
		if !strings.Contains(parseHtml, parseExpected) {
			parseT.Fatalf("expected static shell HTML to contain %q", parseExpected)
		}
	}
}

func TestBuildExamplesListingUsesCatalogEntries(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseExamplesDir := filepath.Join(parseRoot, "examples")
	parseStaticDir := filepath.Join(parseRoot, "static")
	for _, parseDirName := range []string{"02-second", "01-first", "notes"} {
		if parseErr := os.MkdirAll(filepath.Join(parseExamplesDir, parseDirName), 0755); parseErr != nil {
			parseT.Fatalf("mkdir example dir %q: %v", parseDirName, parseErr)
		}
	}
	if parseErr2 := os.MkdirAll(filepath.Join(parseStaticDir, "bin"), 0755); parseErr2 != nil {
		parseT.Fatalf("mkdir static bin dir: %v", parseErr2)
	}
	if parseErr3 := os.WriteFile(filepath.Join(parseExamplesDir, "01-first", "index.html"), []byte("<title>First</title>"), 0644); parseErr3 != nil {
		parseT.Fatalf("write first html: %v", parseErr3)
	}
	if parseErr4 := os.WriteFile(filepath.Join(parseExamplesDir, "02-second", "index.html"), []byte("<title>Second</title>"), 0644); parseErr4 != nil {
		parseT.Fatalf("write second html: %v", parseErr4)
	}

	parseLauncher := launcher{examplesDir: parseExamplesDir, staticDir: parseStaticDir}
	parseLinks, parseErr5 := parseLauncher.buildExamplesListing()
	if parseErr5 != nil {
		parseT.Fatalf("build examples listing: %v", parseErr5)
	}
	if len(parseLinks) != 2 {
		parseT.Fatalf("expected two discoverable examples, got %#v", parseLinks)
	}
	if parseLinks[0] != (exampleLink{Name: "01-first", Href: "/examples/01-first/"}) {
		parseT.Fatalf("expected first sorted example link, got %#v", parseLinks[0])
	}
	if parseLinks[1] != (exampleLink{Name: "02-second", Href: "/examples/02-second/"}) {
		parseT.Fatalf("expected second sorted example link, got %#v", parseLinks[1])
	}
}

func TestBuildExamplesListingIncludesThreeDigitExamples(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseExamplesDir := filepath.Join(parseRoot, "examples")
	parseStaticDir := filepath.Join(parseRoot, "static")
	for _, parseDirName := range []string{"01-first", "203-third", "notes"} {
		if parseErr := os.MkdirAll(filepath.Join(parseExamplesDir, parseDirName), 0755); parseErr != nil {
			parseT.Fatalf("mkdir example dir %q: %v", parseDirName, parseErr)
		}
	}
	if parseErr := os.MkdirAll(filepath.Join(parseStaticDir, "bin"), 0755); parseErr != nil {
		parseT.Fatalf("mkdir static bin dir: %v", parseErr)
	}
	if parseErr := os.WriteFile(filepath.Join(parseExamplesDir, "01-first", "index.html"), []byte("<title>First</title>"), 0644); parseErr != nil {
		parseT.Fatalf("write first html: %v", parseErr)
	}
	if parseErr := os.WriteFile(filepath.Join(parseExamplesDir, "203-third", "index.html"), []byte("<title>Third</title>"), 0644); parseErr != nil {
		parseT.Fatalf("write third html: %v", parseErr)
	}

	parseLauncher := launcher{examplesDir: parseExamplesDir, staticDir: parseStaticDir}
	parseLinks, parseErr := parseLauncher.buildExamplesListing()
	if parseErr != nil {
		parseT.Fatalf("build examples listing: %v", parseErr)
	}
	if len(parseLinks) != 2 {
		parseT.Fatalf("expected two discoverable examples, got %#v", parseLinks)
	}
	if parseLinks[1] != (exampleLink{Name: "203-third", Href: "/examples/203-third/"}) {
		parseT.Fatalf("expected three-digit example link, got %#v", parseLinks[1])
	}
}

func TestWriteStaticExamplesCatalogFileWritesStaticHrefCatalog(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseExamplesDir := filepath.Join(parseRoot, "examples")
	parseStaticDir := filepath.Join(parseRoot, "static")
	if parseErr := os.MkdirAll(filepath.Join(parseExamplesDir, "01-first"), 0755); parseErr != nil {
		parseT.Fatalf("mkdir example dir: %v", parseErr)
	}
	if parseErr2 := os.MkdirAll(filepath.Join(parseStaticDir, "bin"), 0755); parseErr2 != nil {
		parseT.Fatalf("mkdir static dir: %v", parseErr2)
	}
	parseHtml := `<html><head><title>First Example</title></head><body><script src="/static/bin/app.wasm"></script></body></html>`
	if parseErr3 := os.WriteFile(filepath.Join(parseExamplesDir, "01-first", "index.html"), []byte(parseHtml), 0644); parseErr3 != nil {
		parseT.Fatalf("write example html: %v", parseErr3)
	}
	if parseErr4 := os.WriteFile(filepath.Join(parseStaticDir, "bin", "app.wasm"), []byte("wasm"), 0644); parseErr4 != nil {
		parseT.Fatalf("write wasm binary: %v", parseErr4)
	}

	parseLauncher := launcher{examplesDir: parseExamplesDir, staticDir: parseStaticDir}
	parseBlankErr := parseLauncher.writeStaticExamplesCatalogFile("   ")
	if parseBlankErr == nil || !strings.Contains(parseBlankErr.Error(), "static catalog output path is required") {
		parseT.Fatalf("expected blank output path error, got %v", parseBlankErr)
	}

	parseTargetPath := filepath.Join(parseRoot, "out", "catalog.json")
	if parseErr5 := parseLauncher.writeStaticExamplesCatalogFile(parseTargetPath); parseErr5 != nil {
		parseT.Fatalf("write static examples catalog file: %v", parseErr5)
	}
	parseContent, parseErr6 := os.ReadFile(parseTargetPath)
	if parseErr6 != nil {
		parseT.Fatalf("read static catalog file: %v", parseErr6)
	}
	if !strings.HasSuffix(string(parseContent), "\n") {
		parseT.Fatalf("expected static catalog file to end with newline, got %q", string(parseContent))
	}
	var parsePayload examplesCatalogPayload
	if parseErr7 := json.Unmarshal(parseContent, &parsePayload); parseErr7 != nil {
		parseT.Fatalf("decode static catalog file: %v", parseErr7)
	}
	if parsePayload.TotalExamples != 1 || len(parsePayload.Examples) != 1 {
		parseT.Fatalf("expected one static catalog entry, got %#v", parsePayload)
	}
	parseEntry := parsePayload.Examples[0]
	if parseEntry.Href != "../01-first/index.html" {
		parseT.Fatalf("expected static href, got %#v", parseEntry)
	}
	if !parseEntry.UsesWasm || parseEntry.WasmBinary != "app.wasm" {
		parseT.Fatalf("expected wasm metadata in static catalog entry, got %#v", parseEntry)
	}
	if parseEntry.Title != "First Example" {
		parseT.Fatalf("expected html title to be preserved, got %#v", parseEntry)
	}
}

func TestWriteStaticExamplesCatalogFileRelativeAndFailurePaths(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseExamplesDir := filepath.Join(parseRoot, "examples")
	parseStaticDir := filepath.Join(parseRoot, "static")
	if parseErr := os.MkdirAll(filepath.Join(parseExamplesDir, "01-first"), 0755); parseErr != nil {
		parseT.Fatalf("mkdir example dir: %v", parseErr)
	}
	if parseErr2 := os.MkdirAll(filepath.Join(parseStaticDir, "bin"), 0755); parseErr2 != nil {
		parseT.Fatalf("mkdir static dir: %v", parseErr2)
	}
	if parseErr3 := os.WriteFile(filepath.Join(parseExamplesDir, "01-first", "index.html"), []byte("<title>First</title>"), 0644); parseErr3 != nil {
		parseT.Fatalf("write html: %v", parseErr3)
	}

	parseOriginalWD, parseErr4 := os.Getwd()
	if parseErr4 != nil {
		parseT.Fatalf("get working dir: %v", parseErr4)
	}
	if parseErr5 := os.Chdir(parseRoot); parseErr5 != nil {
		parseT.Fatalf("chdir root: %v", parseErr5)
	}
	defer func() {
		_ = os.Chdir(parseOriginalWD)
	}()

	parseExamplesLauncher := launcher{examplesDir: parseExamplesDir, staticDir: parseStaticDir}
	if parseErr6 := parseExamplesLauncher.writeStaticExamplesCatalogFile(filepath.Join("out", "catalog.json")); parseErr6 != nil {
		parseT.Fatalf("write relative static catalog: %v", parseErr6)
	}
	if _, parseErr7 := os.Stat(filepath.Join(parseRoot, "out", "catalog.json")); parseErr7 != nil {
		parseT.Fatalf("expected relative static catalog file: %v", parseErr7)
	}

	parseFailingLauncher := launcher{examplesDir: filepath.Join(parseRoot, "missing"), staticDir: parseStaticDir}
	if parseErr8 := parseFailingLauncher.writeStaticExamplesCatalogFile(filepath.Join(parseRoot, "broken", "catalog.json")); parseErr8 == nil {
		parseT.Fatal("expected static catalog build failure")
	}

	parseBlockedParent := filepath.Join(parseRoot, "blocked")
	if parseErr9 := os.WriteFile(parseBlockedParent, []byte("file"), 0644); parseErr9 != nil {
		parseT.Fatalf("write blocked parent: %v", parseErr9)
	}
	if parseErr10 := parseExamplesLauncher.writeStaticExamplesCatalogFile(filepath.Join(parseBlockedParent, "catalog.json")); parseErr10 == nil || !strings.Contains(parseErr10.Error(), "create static catalog directory") {
		parseT.Fatalf("expected static catalog directory creation failure, got %v", parseErr10)
	}

	parseOriginalMarshal := examplesCatalogMarshalIndent
	parseT.Cleanup(func() { examplesCatalogMarshalIndent = parseOriginalMarshal })
	examplesCatalogMarshalIndent = func(parseV any, parsePrefix string, parseIndent string) ([]byte, error) {
		return nil, errors.New("encode failed")
	}
	if parseErr11 := parseExamplesLauncher.writeStaticExamplesCatalogFile(filepath.Join(parseRoot, "encode", "catalog.json")); parseErr11 == nil || !strings.Contains(parseErr11.Error(), "encode static catalog") {
		parseT.Fatalf("expected static catalog encode failure, got %v", parseErr11)
	}
	examplesCatalogMarshalIndent = parseOriginalMarshal

	parseDirectoryTarget := filepath.Join(parseRoot, "directory-target")
	if parseErr12 := os.MkdirAll(parseDirectoryTarget, 0755); parseErr12 != nil {
		parseT.Fatalf("mkdir directory target: %v", parseErr12)
	}
	if parseErr13 := parseExamplesLauncher.writeStaticExamplesCatalogFile(parseDirectoryTarget); parseErr13 == nil || !strings.Contains(parseErr13.Error(), "write static catalog") {
		parseT.Fatalf("expected static catalog write failure when target is a directory, got %v", parseErr13)
	}
}

func TestBuildBrowserTestEnvPreservesWorkerOverrideAndStripsWasmEnv(parseT *testing.T) {
	parseT.Setenv("GOOS", "js")
	parseT.Setenv("GOARCH", "wasm")
	parseT.Setenv("PLAYWRIGHT_WORKERS", "9")

	parseEnv := buildBrowserTestEnv()
	parseJoined := strings.Join(parseEnv, "\n")
	if strings.Contains(parseJoined, "GOOS=js") || strings.Contains(parseJoined, "GOARCH=wasm") {
		parseT.Fatalf("expected browser env to strip wasm-specific variables, got %#v", parseEnv)
	}
	if !strings.Contains(parseJoined, "PLAYWRIGHT_WORKERS=9") {
		parseT.Fatalf("expected browser env to preserve explicit worker override, got %#v", parseEnv)
	}
	if strings.Contains(parseJoined, "PLAYWRIGHT_WORKERS=4") {
		parseT.Fatalf("expected browser env not to append default workers when already set, got %#v", parseEnv)
	}
}

func TestBuildBrowserTestEnvAddsDefaultWorkersWhenUnset(parseT *testing.T) {
	parseT.Setenv("GOOS", "js")
	parseT.Setenv("GOARCH", "wasm")
	parseOriginalWorkers, parseHadWorkers := os.LookupEnv("PLAYWRIGHT_WORKERS")
	if parseErr := os.Unsetenv("PLAYWRIGHT_WORKERS"); parseErr != nil {
		parseT.Fatalf("unset PLAYWRIGHT_WORKERS: %v", parseErr)
	}
	parseT.Cleanup(func() {
		if !parseHadWorkers {
			_ = os.Unsetenv("PLAYWRIGHT_WORKERS")
			return
		}
		_ = os.Setenv("PLAYWRIGHT_WORKERS", parseOriginalWorkers)
	})

	parseEnv := buildBrowserTestEnv()
	parseJoined := strings.Join(parseEnv, "\n")
	if !strings.Contains(parseJoined, "PLAYWRIGHT_WORKERS=4") {
		parseT.Fatalf("expected browser env to add default workers when unset, got %#v", parseEnv)
	}
	if strings.Contains(parseJoined, "GOOS=js") || strings.Contains(parseJoined, "GOARCH=wasm") {
		parseT.Fatalf("expected browser env to strip wasm variables, got %#v", parseEnv)
	}
}

func TestJoinHostPortDefaultsMissingValues(parseT *testing.T) {
	if parseGot := joinHostPort("", ""); parseGot != defaultHost+":"+defaultPort {
		parseT.Fatalf("expected default host and port, got %q", parseGot)
	}
	if parseGot2 := joinHostPort("127.0.0.1", ""); parseGot2 != "127.0.0.1:"+defaultPort {
		parseT.Fatalf("expected explicit host with default port, got %q", parseGot2)
	}
	if parseGot3 := joinHostPort("", "8123"); parseGot3 != defaultHost+":8123" {
		parseT.Fatalf("expected default host with explicit port, got %q", parseGot3)
	}
}

func TestApplyDevHeadersSetsWasmAndCacheHeaders(parseT *testing.T) {
	parseWasmRecorder := httptest.NewRecorder()
	parseWasmRequest := httptest.NewRequest(http.MethodGet, "/static/app.wasm", nil)
	applyDevHeaders(parseWasmRecorder, parseWasmRequest)
	if parseGot := parseWasmRecorder.Header().Get("Content-Type"); parseGot != "application/wasm" {
		parseT.Fatalf("expected wasm content type, got %q", parseGot)
	}
	if parseGot2 := parseWasmRecorder.Header().Get("Cache-Control"); parseGot2 != "no-store, no-cache, must-revalidate" {
		parseT.Fatalf("expected no-store cache header, got %q", parseGot2)
	}

	parseHtmlRecorder := httptest.NewRecorder()
	parseHtmlRequest := httptest.NewRequest(http.MethodGet, "/index.html", nil)
	applyDevHeaders(parseHtmlRecorder, parseHtmlRequest)
	if parseGot3 := parseHtmlRecorder.Header().Get("Content-Type"); parseGot3 != "" {
		parseT.Fatalf("expected non-wasm content type to remain unset, got %q", parseGot3)
	}
	if parseGot4 := parseHtmlRecorder.Header().Get("Cache-Control"); parseGot4 != "no-store, no-cache, must-revalidate" {
		parseT.Fatalf("expected cache header for non-wasm response, got %q", parseGot4)
	}
}

func TestExamplesCatalogOmitsUnavailableWasmArtifacts(parseT *testing.T) {
	parseRepoRoot, parseErr := resolveRepoRoot()
	if parseErr != nil {
		parseT.Fatalf("resolve repo root: %v", parseErr)
	}
	parseLauncher := launcher{
		repoRoot:    parseRepoRoot,
		examplesDir: filepath.Join(parseRepoRoot, "examples"),
		staticDir:   filepath.Join(parseRepoRoot, "examples", "static"),
	}
	parseCatalog, parseErr := parseLauncher.buildExamplesCatalog()
	if parseErr != nil {
		parseT.Fatalf("build examples catalog: %v", parseErr)
	}
	for _, parseEntry := range parseCatalog.Examples {
		if parseEntry.Name != "88-web-components" {
			continue
		}
		if parseEntry.UsesWasm {
			parseT.Fatalf("expected 88-web-components to stop advertising an unavailable wasm artifact, got %#v", parseEntry)
		}
		if parseEntry.WasmBinary != "" {
			parseT.Fatalf("expected 88-web-components wasmBinary to be empty when the artifact is unavailable, got %#v", parseEntry)
		}
		return
	}
	parseT.Fatalf("expected 88-web-components entry to appear in catalog")
}

func TestExamplesCatalogJSONReportsWasmAndMultiClientEntries(parseT *testing.T) {
	parseRepoRoot, parseErr := resolveRepoRoot()
	if parseErr != nil {
		parseT.Fatalf("resolve repo root: %v", parseErr)
	}
	parseLauncher := launcher{
		repoRoot:    parseRepoRoot,
		examplesDir: filepath.Join(parseRepoRoot, "examples"),
		staticDir:   filepath.Join(parseRepoRoot, "examples", "static"),
		examplesWasmDir: stageExampleWasmFixtures(parseT,
			"counter.wasm",
			"multi-client-presence.wasm",
		),
	}
	parseHandler := parseLauncher.newExamplesHandler("127.0.0.1", "8090")
	parseRequest := httptest.NewRequest(http.MethodGet, "/examples/catalog.json", nil)
	parseRecorder := httptest.NewRecorder()

	parseHandler.ServeHTTP(parseRecorder, parseRequest)

	if parseRecorder.Code != http.StatusOK {
		parseT.Fatalf("expected catalog endpoint to succeed, got %d with body %s", parseRecorder.Code, parseRecorder.Body.String())
	}

	var parsePayload examplesCatalogPayload
	if parseErr2 := json.Unmarshal(parseRecorder.Body.Bytes(), &parsePayload); parseErr2 != nil {
		parseT.Fatalf("decode catalog payload: %v", parseErr2)
	}
	if parsePayload.TotalExamples == 0 || parsePayload.WasmExamples == 0 {
		parseT.Fatalf("expected non-empty catalog summary, got %#v", parsePayload)
	}

	var isFoundPresence bool
	var isFoundUI bool
	for _, parseEntry := range parsePayload.Examples {
		if parseEntry.Name == "97-multi-client-presence" {
			isFoundPresence = true
			if !parseEntry.MultiClient {
				parseT.Fatalf("expected multi-client presence example to be marked multi-client")
			}
			if !parseEntry.UsesWasm || parseEntry.WasmBinary == "" {
				parseT.Fatalf("expected multi-client presence example to advertise wasm binary, got %#v", parseEntry)
			}
		}
		if parseEntry.Name == "21-ui-render" {
			isFoundUI = true
			if !hasAnyTag(parseEntry.Tags, "ui") {
				parseT.Fatalf("expected ui-render example to expose ui tag, got %#v", parseEntry)
			}
		}
	}
	if !isFoundPresence {
		parseT.Fatalf("expected multi-client presence example to appear in catalog payload")
	}
	if !isFoundUI {
		parseT.Fatalf("expected ui-render example to appear in catalog payload")
	}
}

func TestStaticExamplesCatalogUsesHTMLEntrypointsForStaticHosting(parseT *testing.T) {
	parseRepoRoot, parseErr := resolveRepoRoot()
	if parseErr != nil {
		parseT.Fatalf("resolve repo root: %v", parseErr)
	}
	parseLauncher := launcher{
		repoRoot:    parseRepoRoot,
		examplesDir: filepath.Join(parseRepoRoot, "examples"),
		staticDir:   filepath.Join(parseRepoRoot, "examples", "static"),
	}
	parseCatalog, parseErr := parseLauncher.buildStaticExamplesCatalog()
	if parseErr != nil {
		parseT.Fatalf("build static examples catalog: %v", parseErr)
	}

	var isFoundCounter bool
	var isFoundHTMLForms bool
	for _, parseEntry := range parseCatalog.Examples {
		if parseEntry.Name != "01-counter" {
			if parseEntry.Name == "53-html-forms" {
				isFoundHTMLForms = true
				if !hasAnyTag(parseEntry.Tags, "html", "forms") {
					parseT.Fatalf("expected html-forms example to expose html/forms tags, got %#v", parseEntry)
				}
			}
			continue
		}
		isFoundCounter = true
		if parseEntry.Href != "../01-counter/counter.html" {
			parseT.Fatalf("expected static catalog href to target html entrypoint, got %q", parseEntry.Href)
		}
		if !hasAnyTag(parseEntry.Tags, "ui") {
			parseT.Fatalf("expected counter example to expose ui tag, got %#v", parseEntry)
		}
	}
	if !isFoundCounter {
		parseT.Fatalf("expected 01-counter entry in static catalog")
	}
	if !isFoundHTMLForms {
		parseT.Fatalf("expected 53-html-forms entry in static catalog")
	}
}

func TestExamplesRouteGeneratesWasmHostPage(parseT *testing.T) {
	parseRepoRoot, parseErr := resolveRepoRoot()
	if parseErr != nil {
		parseT.Fatalf("resolve repo root: %v", parseErr)
	}
	parseLauncher := launcher{
		repoRoot:        parseRepoRoot,
		examplesDir:     filepath.Join(parseRepoRoot, "examples"),
		staticDir:       filepath.Join(parseRepoRoot, "examples", "static"),
		examplesWasmDir: stageExampleWasmFixtures(parseT, "counter.wasm"),
	}
	parseHandler := parseLauncher.newExamplesHandler("127.0.0.1", "8090")
	parseRequest := httptest.NewRequest(http.MethodGet, "/examples/public/counter/", nil)
	parseRecorder := httptest.NewRecorder()

	parseHandler.ServeHTTP(parseRecorder, parseRequest)

	if parseRecorder.Code != http.StatusOK {
		parseT.Fatalf("expected generated clean route to succeed, got %d with body %s", parseRecorder.Code, parseRecorder.Body.String())
	}
	parseBody := parseRecorder.Body.String()
	for _, parseExpected := range []string{"<div id=\"app\"></div>", "/static/bin/counter.wasm", "Go();", "gwc-examples-runtime-v1", "loadCachedWasm", "__GWC_BOOTSTRAP__", "01-counter"} {
		if !strings.Contains(parseBody, parseExpected) {
			parseT.Fatalf("expected generated example page to contain %q, got %s", parseExpected, parseBody)
		}
	}
	for _, parseUnexpected := range []string{"&#39;caches&#39;", "&#39;/static/bin/counter.wasm&#39;", "result =&gt; go.run"} {
		if strings.Contains(parseBody, parseUnexpected) {
			parseT.Fatalf("expected generated example page loader script to remain raw JavaScript, found escaped fragment %q", parseUnexpected)
		}
	}
}

func TestExamplesRootServesAppShellWithoutRedirect(parseT *testing.T) {
	parseRepoRoot, parseErr := resolveRepoRoot()
	if parseErr != nil {
		parseT.Fatalf("resolve repo root: %v", parseErr)
	}
	parseLauncher := launcher{
		repoRoot:    parseRepoRoot,
		examplesDir: filepath.Join(parseRepoRoot, "examples"),
		staticDir:   filepath.Join(parseRepoRoot, "examples", "static"),
	}
	parseHandler := parseLauncher.newExamplesHandler("127.0.0.1", "8090")
	parseRequest := httptest.NewRequest(http.MethodGet, "/", nil)
	parseRecorder := httptest.NewRecorder()

	parseHandler.ServeHTTP(parseRecorder, parseRequest)

	if parseRecorder.Code != http.StatusOK {
		parseT.Fatalf("expected root route to serve app shell, got %d with body %s", parseRecorder.Code, parseRecorder.Body.String())
	}
	if parseLocation := parseRecorder.Header().Get("Location"); parseLocation != "" {
		parseT.Fatalf("expected root route to avoid redirects, got location %q", parseLocation)
	}
	parseBody := parseRecorder.Body.String()
	for _, parseExpected := range []string{"<div id=\"app\"></div>", "\"catalogHref\":\"/\"", "\"path\":\"/\""} {
		if !strings.Contains(parseBody, parseExpected) {
			parseT.Fatalf("expected root app shell to contain %q, got %s", parseExpected, parseBody)
		}
	}
}

func TestExamplesCatalogRouteServesAppShellWithoutRedirect(parseT *testing.T) {
	parseRepoRoot, parseErr := resolveRepoRoot()
	if parseErr != nil {
		parseT.Fatalf("resolve repo root: %v", parseErr)
	}
	parseLauncher := launcher{
		repoRoot:    parseRepoRoot,
		examplesDir: filepath.Join(parseRepoRoot, "examples"),
		staticDir:   filepath.Join(parseRepoRoot, "examples", "static"),
	}
	parseHandler := parseLauncher.newExamplesHandler("127.0.0.1", "8090")
	parseRequest := httptest.NewRequest(http.MethodGet, "/examples/", nil)
	parseRecorder := httptest.NewRecorder()

	parseHandler.ServeHTTP(parseRecorder, parseRequest)

	if parseRecorder.Code != http.StatusOK {
		parseT.Fatalf("expected /examples/ route to serve app shell, got %d with body %s", parseRecorder.Code, parseRecorder.Body.String())
	}
	if parseLocation := parseRecorder.Header().Get("Location"); parseLocation != "" {
		parseT.Fatalf("expected /examples/ route to avoid redirects, got location %q", parseLocation)
	}
	parseBody := parseRecorder.Body.String()
	for _, parseExpected := range []string{"<div id=\"app\"></div>", "\"catalogHref\":\"/examples/\"", "\"path\":\"/examples/\""} {
		if !strings.Contains(parseBody, parseExpected) {
			parseT.Fatalf("expected /examples/ app shell to contain %q, got %s", parseExpected, parseBody)
		}
	}
}

func TestExamplesRouteRedirectsToTrailingSlash(parseT *testing.T) {
	parseRepoRoot, parseErr := resolveRepoRoot()
	if parseErr != nil {
		parseT.Fatalf("resolve repo root: %v", parseErr)
	}
	parseLauncher := launcher{
		repoRoot:    parseRepoRoot,
		examplesDir: filepath.Join(parseRepoRoot, "examples"),
		staticDir:   filepath.Join(parseRepoRoot, "examples", "static"),
	}
	parseHandler := parseLauncher.newExamplesHandler("127.0.0.1", "8090")
	parseRequest := httptest.NewRequest(http.MethodGet, "/examples/public/counter", nil)
	parseRecorder := httptest.NewRecorder()

	parseHandler.ServeHTTP(parseRecorder, parseRequest)

	if parseRecorder.Code != http.StatusFound {
		parseT.Fatalf("expected clean example route redirect, got %d with body %s", parseRecorder.Code, parseRecorder.Body.String())
	}
	if parseLocation := parseRecorder.Header().Get("Location"); parseLocation != "/examples/public/counter/" {
		parseT.Fatalf("expected redirect to trailing slash route, got %q", parseLocation)
	}
}

func TestExamplesHTMLRouteRedirectsToCleanRoute(parseT *testing.T) {
	parseRepoRoot, parseErr := resolveRepoRoot()
	if parseErr != nil {
		parseT.Fatalf("resolve repo root: %v", parseErr)
	}
	parseLauncher := launcher{
		repoRoot:    parseRepoRoot,
		examplesDir: filepath.Join(parseRepoRoot, "examples"),
		staticDir:   filepath.Join(parseRepoRoot, "examples", "static"),
	}
	parseHandler := parseLauncher.newExamplesHandler("127.0.0.1", "8090")
	parseRequest := httptest.NewRequest(http.MethodGet, "/examples/public/counter/counter.html", nil)
	parseRecorder := httptest.NewRecorder()

	parseHandler.ServeHTTP(parseRecorder, parseRequest)

	if parseRecorder.Code != http.StatusFound {
		parseT.Fatalf("expected html route redirect, got %d with body %s", parseRecorder.Code, parseRecorder.Body.String())
	}
	if parseLocation := parseRecorder.Header().Get("Location"); parseLocation != "/examples/public/counter/" {
		parseT.Fatalf("expected redirect to clean example route, got %q", parseLocation)
	}
}

func TestExamplesNonHTMLAssetRoutePassesThrough(parseT *testing.T) {
	parseRepoRoot, parseErr := resolveRepoRoot()
	if parseErr != nil {
		parseT.Fatalf("resolve repo root: %v", parseErr)
	}
	parseLauncher := launcher{
		repoRoot:    parseRepoRoot,
		examplesDir: filepath.Join(parseRepoRoot, "examples"),
		staticDir:   filepath.Join(parseRepoRoot, "examples", "static"),
	}
	parseHandler := parseLauncher.newExamplesHandler("127.0.0.1", "8090")
	parseRequest := httptest.NewRequest(http.MethodGet, "/examples/public/progressive-web-app-offline-cache/sw.js", nil)
	parseRecorder := httptest.NewRecorder()

	parseHandler.ServeHTTP(parseRecorder, parseRequest)

	if parseRecorder.Code != http.StatusOK {
		parseT.Fatalf("expected sw.js route to pass through, got %d with body %s", parseRecorder.Code, parseRecorder.Body.String())
	}
	if !strings.Contains(parseRecorder.Body.String(), "OFFLINE_URL") {
		parseT.Fatalf("expected service worker contents to be served, got %s", parseRecorder.Body.String())
	}
}

func TestDescribeDevPlanIncludesResolvedModesAndURL(parseT *testing.T) {
	parsePlan := describeDevPlan(devConfig{
		appPath:  `C:\repo\app\main.go`,
		rootPath: `C:\repo\app`,
		host:     "127.0.0.1",
		port:     "8123",
	})
	if parsePlan.ProjectRoot != `C:\repo\app` {
		parseT.Fatalf("expected project root to be preserved, got %q", parsePlan.ProjectRoot)
	}
	if parsePlan.AppMode != "client-only-wasm" {
		parseT.Fatalf("expected client-only app mode, got %q", parsePlan.AppMode)
	}
	if parsePlan.ServerMode != "livereload-wasm" {
		parseT.Fatalf("expected livereload server mode, got %q", parsePlan.ServerMode)
	}
	if parsePlan.ListeningURL != "http://127.0.0.1:8123" {
		parseT.Fatalf("expected listening URL, got %q", parsePlan.ListeningURL)
	}

	parseServerPlan := describeDevPlan(devConfig{
		appPath: `C:\repo\app\cmd\web\main.go`,
		host:    "127.0.0.1",
		port:    "8124",
	})
	if parseServerPlan.AppMode != "server-app" || parseServerPlan.ServerMode != "server-entrypoint" {
		parseT.Fatalf("expected server mode classification, got %#v", parseServerPlan)
	}
}

func TestPrintDevPlanJSONIncludesResolvedSummaryFields(parseT *testing.T) {
	parseStdout, parseRestoreStdout, parseErr := captureExamplesStdout()
	if parseErr != nil {
		parseT.Fatalf("capture stdout: %v", parseErr)
	}
	defer parseRestoreStdout()

	parseErr = printDevPlanJSON(devConfig{
		appPath:  `C:\repo\app\main.go`,
		rootPath: `C:\repo\app`,
		htmlPath: `C:\repo\app\index.html`,
		wasmPath: "main.wasm",
		host:     "127.0.0.1",
		port:     "8125",
		hot:      true,
	})
	if parseErr != nil {
		parseT.Fatalf("print plan json: %v", parseErr)
	}

	parseOutput, parseErr := parseStdout()
	if parseErr != nil {
		parseT.Fatalf("read captured stdout: %v", parseErr)
	}
	var parsePayload map[string]any
	if parseErr2 := json.Unmarshal([]byte(parseOutput), &parsePayload); parseErr2 != nil {
		parseT.Fatalf("unmarshal payload: %v\n%s", parseErr2, parseOutput)
	}
	for parseKey, parseExpected := range map[string]string{
		"projectRoot":  `C:\repo\app`,
		"appMode":      "client-only-wasm",
		"serverMode":   "livereload-wasm",
		"listeningURL": "http://127.0.0.1:8125",
	} {
		if parsePayload[parseKey] != parseExpected {
			parseT.Fatalf("expected %s to be %q, got %#v", parseKey, parseExpected, parsePayload[parseKey])
		}
	}
}

func TestResolveDevConfigPrefersScaffoldMetadata(parseT *testing.T) {
	parseTempApp := parseT.TempDir()
	if parseErr := os.WriteFile(filepath.Join(parseTempApp, "main.go"), []byte("package main\nfunc main() {}\n"), 0644); parseErr != nil {
		parseT.Fatalf("write main.go: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(filepath.Join(parseTempApp, "index.html"), []byte("<html></html>\n"), 0644); parseErr2 != nil {
		parseT.Fatalf("write index.html: %v", parseErr2)
	}
	parseMetadata := `{
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
	if parseErr3 := os.WriteFile(filepath.Join(parseTempApp, "gwc-start.json"), []byte(parseMetadata), 0644); parseErr3 != nil {
		parseT.Fatalf("write gwc-start.json: %v", parseErr3)
	}

	parseOriginalGetwd := devGetwd
	parseT.Cleanup(func() { devGetwd = parseOriginalGetwd })
	devGetwd = func() (string, error) { return parseTempApp, nil }

	parseLauncher := launcher{}
	parseConfig, parseErr4 := parseLauncher.resolveDevConfig(devConfig{})
	if parseErr4 != nil {
		parseT.Fatalf("resolve dev config: %v", parseErr4)
	}
	if parseConfig.appPath != filepath.Join(parseTempApp, "main.go") {
		parseT.Fatalf("expected metadata app path, got %#v", parseConfig)
	}
	if parseConfig.rootPath != parseTempApp {
		parseT.Fatalf("expected metadata root path, got %#v", parseConfig)
	}
	if parseConfig.htmlPath != filepath.Join(parseTempApp, "index.html") {
		parseT.Fatalf("expected metadata html path, got %#v", parseConfig)
	}
	if parseConfig.wasmPath != "build/app.wasm" {
		parseT.Fatalf("expected metadata wasm path, got %#v", parseConfig)
	}
	if parseConfig.host != "0.0.0.0" || parseConfig.port != "8140" {
		parseT.Fatalf("expected metadata host/port, got %#v", parseConfig)
	}
	for parseKey, parseExpected := range map[string]string{
		"app":  "gwc-start.json",
		"root": "gwc-start.json",
		"html": "gwc-start.json",
		"wasm": "gwc-start.json",
		"host": "gwc-start.json",
		"port": "gwc-start.json",
	} {
		if parseConfig.resolution[parseKey] != parseExpected {
			parseT.Fatalf("expected dev resolution %q to be %q, got %#v", parseKey, parseExpected, parseConfig.resolution)
		}
	}
}

func TestResolveBuildConfigTracksResolutionSources(parseT *testing.T) {
	parseTempApp := parseT.TempDir()
	if parseErr := os.WriteFile(filepath.Join(parseTempApp, "main.go"), []byte("package main\nfunc main() {}\n"), 0644); parseErr != nil {
		parseT.Fatalf("write main.go: %v", parseErr)
	}
	parseMetadata := `{
  "projectName": "metadata-build-app",
  "modulePath": "example.com/metadata-build-app",
  "tooling": {
    "appPath": "main.go",
    "wasmPath": "build/app.wasm",
    "defaultBuildProfile": "ci"
  }
}
`
	if parseErr2 := os.WriteFile(filepath.Join(parseTempApp, "gwc-start.json"), []byte(parseMetadata), 0644); parseErr2 != nil {
		parseT.Fatalf("write gwc-start.json: %v", parseErr2)
	}

	parseOriginalGetwd := buildGetwd
	parseT.Cleanup(func() { buildGetwd = parseOriginalGetwd })
	buildGetwd = func() (string, error) { return parseTempApp, nil }

	parseConfig, parseErr3 := resolveBuildConfig(buildConfig{})
	if parseErr3 != nil {
		parseT.Fatalf("resolve build config: %v", parseErr3)
	}
	if parseConfig.appPath != filepath.Join(parseTempApp, "main.go") || parseConfig.rootPath != parseTempApp || parseConfig.outputPath != filepath.Join(parseTempApp, "build", "app.wasm") || parseConfig.profile != "ci" {
		parseT.Fatalf("expected metadata-resolved build config, got %#v", parseConfig)
	}
	for parseKey, parseExpected := range map[string]string{
		"app":     "gwc-start.json",
		"root":    "gwc-start.json",
		"output":  "gwc-start.json",
		"profile": "gwc-start.json",
	} {
		if parseConfig.resolution[parseKey] != parseExpected {
			parseT.Fatalf("expected build resolution %q to be %q, got %#v", parseKey, parseExpected, parseConfig.resolution)
		}
	}
}

func TestResolveTestConfigTracksExplicitAndFallbackSources(parseT *testing.T) {
	parseTempRoot := parseT.TempDir()
	parseAppPath := filepath.Join(parseTempRoot, "main.go")
	if parseErr := os.WriteFile(parseAppPath, []byte("package main\n"), 0644); parseErr != nil {
		parseT.Fatalf("write main.go: %v", parseErr)
	}

	parseOriginalGetwd := testGetwd
	parseT.Cleanup(func() { testGetwd = parseOriginalGetwd })
	testGetwd = func() (string, error) { return parseTempRoot, nil }

	parseConfig, parseErr2 := resolveTestConfig(testConfig{appPath: parseAppPath})
	if parseErr2 != nil {
		parseT.Fatalf("resolve test config: %v", parseErr2)
	}
	if parseConfig.rootPath != parseTempRoot || parseConfig.appPath != parseAppPath {
		parseT.Fatalf("expected resolved test config paths, got %#v", parseConfig)
	}
	if parseConfig.resolution["root"] != "convention fallback" || parseConfig.resolution["app"] != "explicit flag" {
		parseT.Fatalf("expected test resolution sources, got %#v", parseConfig.resolution)
	}
}

func TestBuildDoctorReportPassesWithHealthyTooling(parseT *testing.T) {
	parseTempRepo := parseT.TempDir()
	if parseErr := os.MkdirAll(filepath.Join(parseTempRepo, "test", "playwrightgo"), 0755); parseErr != nil {
		parseT.Fatalf("create playwrightgo dir: %v", parseErr)
	}

	parseTempApp := parseT.TempDir()
	if parseErr2 := os.WriteFile(filepath.Join(parseTempApp, "main.go"), []byte("package main\n"), 0644); parseErr2 != nil {
		parseT.Fatalf("write main.go: %v", parseErr2)
	}
	if parseErr3 := os.WriteFile(filepath.Join(parseTempApp, "index.html"), []byte("<html></html>\n"), 0644); parseErr3 != nil {
		parseT.Fatalf("write index.html: %v", parseErr3)
	}
	if parseErr4 := os.WriteFile(filepath.Join(parseTempApp, "gwc-start.json"), []byte("{\n  \"projectName\": \"doctor-app\",\n  \"modulePath\": \"example.com/doctor-app\"\n}\n"), 0644); parseErr4 != nil {
		parseT.Fatalf("write gwc-start.json: %v", parseErr4)
	}

	parseOriginalLookPath := doctorLookPath
	parseOriginalCommandOutput := doctorCommandOutput
	parseOriginalGetwd := doctorGetwd
	parseOriginalListen := doctorListen
	parseOriginalResolveWasmExec := doctorResolveWasmExec
	parseT.Cleanup(func() {
		doctorLookPath = parseOriginalLookPath
		doctorCommandOutput = parseOriginalCommandOutput
		doctorGetwd = parseOriginalGetwd
		doctorListen = parseOriginalListen
		doctorResolveWasmExec = parseOriginalResolveWasmExec
	})

	doctorLookPath = func(parseName2 string) (string, error) {
		return filepath.Join("C:\\tools", parseName2), nil
	}
	doctorCommandOutput = func(parseName3 string, parseArgs ...string) (string, error) {
		switch parseName3 {
		case "go":
			return "go version go1.25.0 windows/amd64", nil
		default:
			return "ok", nil
		}
	}
	doctorGetwd = func() (string, error) { return parseTempApp, nil }
	doctorListen = func(parseNetwork string, parseAddress string) (net.Listener, error) {
		return net.Listen("tcp", "127.0.0.1:0")
	}
	doctorResolveWasmExec = func() (string, error) { return filepath.Join("C:\\Go", "lib", "wasm", "wasm_exec.js"), nil }

	parseLauncher := launcher{repoRoot: parseTempRepo}
	parseReport := parseLauncher.buildDoctorReport(doctorConfig{host: "127.0.0.1", port: "8123"})
	if !parseReport.OK {
		parseT.Fatalf("expected healthy doctor report, got %#v", parseReport)
	}
	parseStatuses := map[string]string{}
	for _, parseCheck := range parseReport.Checks {
		parseStatuses[parseCheck.Name] = parseCheck.Status
	}
	for _, parseName := range []string{"Go toolchain", "wasm_exec.js", "Browser tests", "Scaffold metadata", "Project detection", "Port availability"} {
		if parseStatuses[parseName] != "pass" {
			parseT.Fatalf("expected %s to pass, got %#v", parseName, parseStatuses[parseName])
		}
	}
	if parseReport.CWD != parseTempApp {
		parseT.Fatalf("expected cwd %q, got %q", parseTempApp, parseReport.CWD)
	}
	for parseKey, parseExpected := range map[string]string{
		"app":  "convention fallback",
		"root": "gwc-start.json",
		"html": "convention fallback",
		"host": "convention fallback",
		"port": "explicit flag",
	} {
		if parseReport.Resolution[parseKey] != parseExpected {
			parseT.Fatalf("expected doctor resolution %q to be %q, got %#v", parseKey, parseExpected, parseReport.Resolution)
		}
	}
}

func TestBuildDoctorReportFailsWhenPortIsUnavailable(parseT *testing.T) {
	parseOriginalGetwd := doctorGetwd
	parseOriginalListen := doctorListen
	parseOriginalLookPath := doctorLookPath
	parseOriginalCommandOutput := doctorCommandOutput
	parseOriginalResolveWasmExec := doctorResolveWasmExec
	parseT.Cleanup(func() {
		doctorGetwd = parseOriginalGetwd
		doctorListen = parseOriginalListen
		doctorLookPath = parseOriginalLookPath
		doctorCommandOutput = parseOriginalCommandOutput
		doctorResolveWasmExec = parseOriginalResolveWasmExec
	})

	doctorGetwd = func() (string, error) { return parseT.TempDir(), nil }
	doctorLookPath = func(parseName string) (string, error) { return parseName, nil }
	doctorCommandOutput = func(parseName2 string, parseArgs ...string) (string, error) { return parseName2 + " ok", nil }
	doctorResolveWasmExec = func() (string, error) { return "wasm_exec.js", nil }
	doctorListen = func(parseNetwork string, parseAddress string) (net.Listener, error) {
		return nil, errors.New("address already in use")
	}

	parseLauncher := launcher{repoRoot: parseT.TempDir()}
	parseReport := parseLauncher.buildDoctorReport(doctorConfig{host: "127.0.0.1", port: "8090"})
	if parseReport.OK {
		parseT.Fatalf("expected failing report when port is unavailable, got %#v", parseReport)
	}
	for _, parseCheck := range parseReport.Checks {
		if parseCheck.Name == "Port availability" {
			if parseCheck.Status != "fail" {
				parseT.Fatalf("expected port availability to fail, got %#v", parseCheck)
			}
			return
		}
	}
	parseT.Fatal("expected port availability check to be present")
}

func TestBuildDoctorReportIncludesGoldenPathAuditWhenRequested(parseT *testing.T) {
	parseTempRepo := parseT.TempDir()
	if parseErr := os.MkdirAll(filepath.Join(parseTempRepo, "test", "playwrightgo"), 0755); parseErr != nil {
		parseT.Fatalf("create playwrightgo dir: %v", parseErr)
	}
	parseTempApp := parseT.TempDir()
	if parseErr2 := os.WriteFile(filepath.Join(parseTempApp, "main.go"), []byte("package main\n"), 0644); parseErr2 != nil {
		parseT.Fatalf("write main.go: %v", parseErr2)
	}
	if parseErr3 := os.WriteFile(filepath.Join(parseTempApp, "index.html"), []byte("<html></html>\n"), 0644); parseErr3 != nil {
		parseT.Fatalf("write index.html: %v", parseErr3)
	}
	if parseErr4 := os.WriteFile(filepath.Join(parseTempApp, "gwc-start.json"), []byte("{\n  \"projectName\": \"audit-app\"\n}\n"), 0644); parseErr4 != nil {
		parseT.Fatalf("write gwc-start.json: %v", parseErr4)
	}

	parseOriginalLookPath := doctorLookPath
	parseOriginalCommandOutput := doctorCommandOutput
	parseOriginalGetwd := doctorGetwd
	parseOriginalListen := doctorListen
	parseOriginalResolveWasmExec := doctorResolveWasmExec
	parseT.Cleanup(func() {
		doctorLookPath = parseOriginalLookPath
		doctorCommandOutput = parseOriginalCommandOutput
		doctorGetwd = parseOriginalGetwd
		doctorListen = parseOriginalListen
		doctorResolveWasmExec = parseOriginalResolveWasmExec
	})
	doctorLookPath = func(parseName3 string) (string, error) { return parseName3, nil }
	doctorCommandOutput = func(parseName4 string, parseArgs ...string) (string, error) { return parseName4 + " version", nil }
	doctorGetwd = func() (string, error) { return parseTempApp, nil }
	doctorResolveWasmExec = func() (string, error) { return "wasm_exec.js", nil }
	doctorListen = func(parseNetwork string, parseAddress string) (net.Listener, error) {
		return net.Listen("tcp", "127.0.0.1:0")
	}

	parseReport := (launcher{repoRoot: parseTempRepo}).buildDoctorReport(doctorConfig{host: "127.0.0.1", port: "8125", audit: true})
	if !parseReport.OK {
		parseT.Fatalf("expected doctor audit report to pass, got %#v", parseReport)
	}
	if parseReport.Audit == nil || !parseReport.Audit.OK {
		parseT.Fatalf("expected golden-path audit section, got %#v", parseReport)
	}
	parseStatuses := map[string]string{}
	for _, parseCheck := range parseReport.Audit.Checks {
		parseStatuses[parseCheck.Name] = parseCheck.Status
	}
	for _, parseName := range []string{"App entrypoint", "HTML shell", "Starter metadata anchor"} {
		if parseStatuses[parseName] != "pass" {
			parseT.Fatalf("expected audit check %q to pass, got %#v", parseName, parseReport.Audit.Checks)
		}
	}
	for _, parseName2 := range []string{"State and ownership boundaries", "Local versus shared state ownership", "Route shape and delivery", "Mutation and resilience", "Startup cost and ownership evidence", "Runtime evidence"} {
		if parseStatuses[parseName2] != "pass" {
			parseT.Fatalf("expected audit check %q to pass, got %#v", parseName2, parseReport.Audit.Checks)
		}
	}
}

func TestBuildDoctorGoldenPathAuditFlagsStateAndOwnershipViolations(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	if parseErr := os.MkdirAll(filepath.Join(parseRoot, "client", "app"), 0755); parseErr != nil {
		parseT.Fatalf("mkdir client app: %v", parseErr)
	}
	if parseErr2 := os.MkdirAll(filepath.Join(parseRoot, "server"), 0755); parseErr2 != nil {
		parseT.Fatalf("mkdir server dir: %v", parseErr2)
	}
	parseClientLeak := `//go:build js && wasm

package app

import (
	"database/sql"

	"github.com/monstercameron/GoWebComponents/examples/server/ai-chat-wizard/server/app"
	"github.com/monstercameron/GoWebComponents/fetch"
)

func leak() {
	_ = sql.ErrNoRows
	_ = app.Run
	_ = fetch.UseCachedResource
	println(localStorage)
}
`
	if parseErr3 := os.WriteFile(filepath.Join(parseRoot, "client", "app", "leak.go"), []byte(parseClientLeak), 0644); parseErr3 != nil {
		parseT.Fatalf("write client leak: %v", parseErr3)
	}
	parseServerLeak := `package server

import "syscall/js"

func leak() { _ = js.Null() }
`
	if parseErr4 := os.WriteFile(filepath.Join(parseRoot, "server", "leak.go"), []byte(parseServerLeak), 0644); parseErr4 != nil {
		parseT.Fatalf("write server leak: %v", parseErr4)
	}

	parseAudit := buildDoctorGoldenPathAudit(parseRoot)
	parseStatuses := map[string]doctorCheck{}
	for _, parseCheck := range parseAudit.Checks {
		parseStatuses[parseCheck.Name] = parseCheck
	}
	parseOwnershipCheck := parseStatuses["State and ownership boundaries"]
	if parseOwnershipCheck.Status != "fail" || !strings.Contains(parseOwnershipCheck.Summary, "client/app/leak.go imports server-only package database/sql") || !strings.Contains(parseOwnershipCheck.Summary, "server/leak.go imports browser-only package syscall/js") {
		parseT.Fatalf("expected ownership boundary failures, got %#v", parseOwnershipCheck)
	}
	if parseOwnershipCheck.RuleID != "audit.state_boundaries" || parseOwnershipCheck.Severity != "error" || len(parseOwnershipCheck.Locations) == 0 {
		parseT.Fatalf("expected machine-readable ownership metadata, got %#v", parseOwnershipCheck)
	}
	parseStateCheck := parseStatuses["Local versus shared state ownership"]
	if parseStateCheck.Status != "warn" || !strings.Contains(parseStateCheck.Summary, "client/app/leak.go mixes fetch.UseCachedResource with direct browser storage access") {
		parseT.Fatalf("expected mixed state ownership warning, got %#v", parseStateCheck)
	}
	if parseStateCheck.RuleID != "audit.state_ownership" || parseStateCheck.Severity != "warning" || len(parseStateCheck.Locations) == 0 {
		parseT.Fatalf("expected machine-readable state ownership metadata, got %#v", parseStateCheck)
	}
}

func TestBuildDoctorGoldenPathAuditFlagsRouteShapeWarnings(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseRouteSource := `package main

import "github.com/monstercameron/GoWebComponents/router"

var (
	homeRoute = router.MustDefineRoute("/")
	pricingRoute = router.MustDefineRoute("/pricing")
	capabilitiesRoute = router.MustDefineRoute("/capabilities")
)

func registerRoutes(r *router.Router) {
	r.Register(homeRoute.Pattern(), nil)
	r.Register(pricingRoute.Pattern(), nil)
	r.Register(capabilitiesRoute.Pattern(), nil)
}
`
	if parseErr := os.WriteFile(filepath.Join(parseRoot, "main.go"), []byte(parseRouteSource), 0644); parseErr != nil {
		parseT.Fatalf("write main.go: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(filepath.Join(parseRoot, "index.html"), []byte("<html></html>\n"), 0644); parseErr2 != nil {
		parseT.Fatalf("write index.html: %v", parseErr2)
	}
	if parseErr3 := os.WriteFile(filepath.Join(parseRoot, "pricing.html"), []byte("<html></html>\n"), 0644); parseErr3 != nil {
		parseT.Fatalf("write pricing.html: %v", parseErr3)
	}

	parseAudit := buildDoctorGoldenPathAudit(parseRoot)
	parseStatuses := map[string]doctorCheck{}
	for _, parseCheck := range parseAudit.Checks {
		parseStatuses[parseCheck.Name] = parseCheck
	}
	parseRouteCheck := parseStatuses["Route shape and delivery"]
	if parseRouteCheck.Status != "warn" ||
		!strings.Contains(parseRouteCheck.Summary, "multiple HTML entry shells detected") ||
		!strings.Contains(parseRouteCheck.Summary, "no ui.Lazy split signal") ||
		!strings.Contains(parseRouteCheck.Summary, "marketing-style routes were detected with no static/prerender delivery hint") {
		parseT.Fatalf("expected route shape warnings, got %#v", parseRouteCheck)
	}
}

func TestBuildDoctorGoldenPathAuditFlagsMutationResilienceWarnings(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	if parseErr := os.MkdirAll(filepath.Join(parseRoot, "client", "app"), 0755); parseErr != nil {
		parseT.Fatalf("mkdir client app: %v", parseErr)
	}
	parseSource := `package app

func submitSettings() {
	SetSelectedModel()
	DeleteConversation()
}
`
	if parseErr2 := os.WriteFile(filepath.Join(parseRoot, "client", "app", "mutations.go"), []byte(parseSource), 0644); parseErr2 != nil {
		parseT.Fatalf("write mutations.go: %v", parseErr2)
	}

	parseAudit := buildDoctorGoldenPathAudit(parseRoot)
	parseStatuses := map[string]doctorCheck{}
	for _, parseCheck := range parseAudit.Checks {
		parseStatuses[parseCheck.Name] = parseCheck
	}
	parseMutationCheck := parseStatuses["Mutation and resilience"]
	if parseMutationCheck.Status != "warn" || !strings.Contains(parseMutationCheck.Summary, "client/app/mutations.go exposes mutation-shaped code with no retry/idempotency/conflict/offline signal") {
		parseT.Fatalf("expected mutation resilience warning, got %#v", parseMutationCheck)
	}
}

func TestBuildDoctorGoldenPathAuditFlagsStartupEvidenceWarnings(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	if parseErr := os.MkdirAll(filepath.Join(parseRoot, "client", "app"), 0755); parseErr != nil {
		parseT.Fatalf("mkdir client app: %v", parseErr)
	}
	parseImports := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k"}
	var parseBuilder strings.Builder
	parseBuilder.WriteString("//go:build js && wasm\n\npackage app\n\nimport (\n")
	for _, parseImportPath := range parseImports {
		parseBuilder.WriteString(fmt.Sprintf("\t%q\n", parseImportPath))
	}
	parseBuilder.WriteString(")\n\n")
	parseBuilder.WriteString("const payload = `")
	parseBuilder.WriteString(strings.Repeat("x", 70*1024))
	parseBuilder.WriteString("`\n")
	if parseErr2 := os.WriteFile(filepath.Join(parseRoot, "client", "app", "heavy.go"), []byte(parseBuilder.String()), 0644); parseErr2 != nil {
		parseT.Fatalf("write heavy.go: %v", parseErr2)
	}

	parseAudit := buildDoctorGoldenPathAudit(parseRoot)
	parseStatuses := map[string]doctorCheck{}
	for _, parseCheck := range parseAudit.Checks {
		parseStatuses[parseCheck.Name] = parseCheck
	}
	parseStartupCheck := parseStatuses["Startup cost and ownership evidence"]
	if parseStartupCheck.Status != "warn" ||
		!strings.Contains(parseStartupCheck.Summary, "client/app/heavy.go is") ||
		!strings.Contains(parseStartupCheck.Summary, "client/app/heavy.go imports 11 packages") {
		parseT.Fatalf("expected startup evidence warning, got %#v", parseStartupCheck)
	}
}

func TestBuildDoctorGoldenPathAuditCollectsRuntimeEvidence(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	if parseErr := os.MkdirAll(filepath.Join(parseRoot, "bin", "release"), 0755); parseErr != nil {
		parseT.Fatalf("mkdir runtime artifact dir: %v", parseErr)
	}
	parseWasmPath := filepath.Join(parseRoot, "bin", "release", "app.wasm")
	if parseErr2 := os.WriteFile(parseWasmPath, []byte(strings.Repeat("w", 6*1024*1024)), 0644); parseErr2 != nil {
		parseT.Fatalf("write wasm artifact: %v", parseErr2)
	}
	parseStartupReport := `{
  "startup": {
    "readyMs": 3200,
    "interactionMs": 780
  }
}`
	if parseErr3 := os.WriteFile(filepath.Join(parseRoot, "bin", "release", "wasm-startup-report.json"), []byte(parseStartupReport), 0644); parseErr3 != nil {
		parseT.Fatalf("write startup report: %v", parseErr3)
	}

	parseAudit := buildDoctorGoldenPathAudit(parseRoot)
	parseStatuses := map[string]doctorCheck{}
	for _, parseCheck := range parseAudit.Checks {
		parseStatuses[parseCheck.Name] = parseCheck
	}
	parseRuntimeCheck := parseStatuses["Runtime evidence"]
	if parseRuntimeCheck.Status != "warn" ||
		!strings.Contains(parseRuntimeCheck.Summary, "bin/release/app.wasm weighs") ||
		!strings.Contains(parseRuntimeCheck.Summary, "bin/release/wasm-startup-report.json reports readyMs=3200") ||
		!strings.Contains(parseRuntimeCheck.Summary, "interactionMs=780") {
		parseT.Fatalf("expected runtime evidence warning, got %#v", parseRuntimeCheck)
	}
}

func TestBuildDoctorStandaloneChecksAdditionalBranches(parseT *testing.T) {
	parseOriginalResolveWasmExec := doctorResolveWasmExec
	parseT.Cleanup(func() { doctorResolveWasmExec = parseOriginalResolveWasmExec })
	doctorResolveWasmExec = func() (string, error) { return "", errors.New("missing wasm exec") }
	parseWasmCheck := buildDoctorWasmExecCheck()
	if parseWasmCheck.Status != "fail" || !strings.Contains(parseWasmCheck.Summary, "could not be resolved") {
		parseT.Fatalf("expected failing wasm_exec check, got %#v", parseWasmCheck)
	}

	parseRoot := parseT.TempDir()
	parseMissingPackage := buildDoctorPlaywrightCheck(parseRoot)
	if parseMissingPackage.Status != "warn" {
		parseT.Fatalf("expected missing browser package to warn, got %#v", parseMissingPackage)
	}
	if !strings.Contains(strings.ToLower(parseMissingPackage.Summary), "no playwright-go browser suite") {
		parseT.Fatalf("expected missing browser suite warning, got %#v", parseMissingPackage)
	}
	if parseErr := os.MkdirAll(filepath.Join(parseRoot, "test"), 0755); parseErr != nil {
		parseT.Fatalf("mkdir test dir: %v", parseErr)
	}
	parseMissingDeps := buildDoctorPlaywrightCheck(parseRoot)
	if parseMissingDeps.Status != "warn" || !strings.Contains(strings.ToLower(parseMissingDeps.Summary), "no playwright-go browser suite") {
		parseT.Fatalf("expected missing playwright-go suite warning, got %#v", parseMissingDeps)
	}

	parseBlankMetadata := buildDoctorMetadataCheck("")
	if parseBlankMetadata.Status != "warn" {
		parseT.Fatalf("expected blank cwd metadata warning, got %#v", parseBlankMetadata)
	}
	if parseErr2 := os.WriteFile(filepath.Join(parseRoot, "gwc-start.json"), []byte(`{"tooling":`), 0644); parseErr2 != nil {
		parseT.Fatalf("write invalid metadata: %v", parseErr2)
	}
	parseInvalidMetadata := buildDoctorMetadataCheck(parseRoot)
	if parseInvalidMetadata.Status != "fail" || !strings.Contains(parseInvalidMetadata.Summary, "parse scaffold metadata") {
		parseT.Fatalf("expected invalid metadata failure, got %#v", parseInvalidMetadata)
	}

	parseBlankProjectDetection := buildDoctorProjectDetectionCheck("")
	if parseBlankProjectDetection.Status != "warn" {
		parseT.Fatalf("expected blank cwd project detection warning, got %#v", parseBlankProjectDetection)
	}
	parseMissingProjectDetection := buildDoctorProjectDetectionCheck(parseT.TempDir())
	if parseMissingProjectDetection.Status != "warn" {
		parseT.Fatalf("expected missing app project detection warning, got %#v", parseMissingProjectDetection)
	}
}

func TestProjectHasGoTestsSkipsIgnoredDirectories(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	if parseErr := os.MkdirAll(filepath.Join(parseRoot, "node_modules", "pkg"), 0755); parseErr != nil {
		parseT.Fatalf("mkdir ignored dir: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(filepath.Join(parseRoot, "node_modules", "pkg", "ignored_test.go"), []byte("package pkg\n"), 0644); parseErr2 != nil {
		parseT.Fatalf("write ignored test file: %v", parseErr2)
	}
	if parseErr3 := os.WriteFile(filepath.Join(parseRoot, "main_test.go"), []byte("package main\n"), 0644); parseErr3 != nil {
		parseT.Fatalf("write root test file: %v", parseErr3)
	}

	hasTests, parseErr4 := projectHasGoTests(parseRoot)
	if parseErr4 != nil {
		parseT.Fatalf("project has go tests: %v", parseErr4)
	}
	if !hasTests {
		parseT.Fatal("expected root test file to be detected")
	}

	parseEmptyRoot := parseT.TempDir()
	hasTests, parseErr4 = projectHasGoTests(parseEmptyRoot)
	if parseErr4 != nil {
		parseT.Fatalf("project has go tests for empty root: %v", parseErr4)
	}
	if hasTests {
		parseT.Fatal("expected empty root to report no tests")
	}
}

func TestRunDoctorJSONEmitsMachineReadableReport(parseT *testing.T) {
	parseTempRepo := parseT.TempDir()
	if parseErr := os.MkdirAll(filepath.Join(parseTempRepo, "test", "playwrightgo"), 0755); parseErr != nil {
		parseT.Fatalf("create playwrightgo dir: %v", parseErr)
	}
	parseTempApp := parseT.TempDir()
	if parseErr2 := os.WriteFile(filepath.Join(parseTempApp, "main.go"), []byte("package main\n"), 0644); parseErr2 != nil {
		parseT.Fatalf("write main.go: %v", parseErr2)
	}

	parseStdout, parseRestoreStdout, parseErr3 := captureExamplesStdout()
	if parseErr3 != nil {
		parseT.Fatalf("capture stdout: %v", parseErr3)
	}
	defer parseRestoreStdout()

	parseOriginalLookPath := doctorLookPath
	parseOriginalCommandOutput := doctorCommandOutput
	parseOriginalGetwd := doctorGetwd
	parseOriginalListen := doctorListen
	parseOriginalResolveWasmExec := doctorResolveWasmExec
	parseT.Cleanup(func() {
		doctorLookPath = parseOriginalLookPath
		doctorCommandOutput = parseOriginalCommandOutput
		doctorGetwd = parseOriginalGetwd
		doctorListen = parseOriginalListen
		doctorResolveWasmExec = parseOriginalResolveWasmExec
	})
	doctorLookPath = func(parseName string) (string, error) { return parseName, nil }
	doctorCommandOutput = func(parseName2 string, parseArgs ...string) (string, error) { return parseName2 + " version", nil }
	doctorGetwd = func() (string, error) { return parseTempApp, nil }
	doctorResolveWasmExec = func() (string, error) { return "wasm_exec.js", nil }
	doctorListen = func(parseNetwork string, parseAddress string) (net.Listener, error) {
		return net.Listen("tcp", "127.0.0.1:0")
	}

	parseLauncher := launcher{repoRoot: parseTempRepo}
	if parseErr4 := parseLauncher.run([]string{"doctor", "-json", "-port", "8127"}); parseErr4 != nil {
		parseT.Fatalf("run doctor json: %v", parseErr4)
	}

	parseOutput, parseErr3 := parseStdout()
	if parseErr3 != nil {
		parseT.Fatalf("read captured stdout: %v", parseErr3)
	}
	var parseReport doctorReport
	if parseErr5 := json.Unmarshal([]byte(parseOutput), &parseReport); parseErr5 != nil {
		parseT.Fatalf("unmarshal doctor report: %v\n%s", parseErr5, parseOutput)
	}
	if !parseReport.OK {
		parseT.Fatalf("expected ok doctor report, got %#v", parseReport)
	}
	if len(parseReport.Checks) == 0 {
		parseT.Fatalf("expected doctor checks in JSON output, got %#v", parseReport)
	}
}

func TestBuildDoctorGoldenPathAuditReportsPassingAnchors(parseT *testing.T) {
	parseTempApp := parseT.TempDir()
	if parseErr := os.WriteFile(filepath.Join(parseTempApp, "main.go"), []byte("package main\n"), 0644); parseErr != nil {
		parseT.Fatalf("write main.go: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(filepath.Join(parseTempApp, "index.html"), []byte("<!DOCTYPE html>\n"), 0644); parseErr2 != nil {
		parseT.Fatalf("write index.html: %v", parseErr2)
	}
	if parseErr3 := os.WriteFile(filepath.Join(parseTempApp, "gwc-start.json"), []byte("{\n  \"projectName\": \"audit-app\",\n  \"modulePath\": \"example.com/audit-app\"\n}\n"), 0644); parseErr3 != nil {
		parseT.Fatalf("write gwc-start.json: %v", parseErr3)
	}

	parseReport := buildDoctorGoldenPathAudit(parseTempApp)
	if parseReport.Mode != "golden-path" {
		parseT.Fatalf("expected golden-path mode, got %#v", parseReport)
	}
	if !parseReport.OK {
		parseT.Fatalf("expected passing audit report, got %#v", parseReport)
	}
	if len(parseReport.Checks) != 9 {
		parseT.Fatalf("expected nine baseline audit checks, got %#v", parseReport.Checks)
	}
	for _, parseCheck := range parseReport.Checks {
		if parseCheck.Status != "pass" {
			parseT.Fatalf("expected passing audit check, got %#v", parseCheck)
		}
	}
}

func TestBuildDoctorOwnershipBoundaryCheckFlagsClientServerLeak(parseT *testing.T) {
	parseTempApp := parseT.TempDir()
	parseClientDir := filepath.Join(parseTempApp, "client")
	if parseErr := os.MkdirAll(parseClientDir, 0755); parseErr != nil {
		parseT.Fatalf("mkdir client dir: %v", parseErr)
	}
	parseClientFile := []byte("//go:build js && wasm\n\npackage client\n\nimport (\n\t\"database/sql\"\n\t\"example.com/app/server/auth\"\n)\n")
	if parseErr2 := os.WriteFile(filepath.Join(parseClientDir, "main.go"), parseClientFile, 0644); parseErr2 != nil {
		parseT.Fatalf("write client main.go: %v", parseErr2)
	}

	parseCheck := buildDoctorOwnershipBoundaryCheck(parseTempApp)
	if parseCheck.Status != "fail" {
		parseT.Fatalf("expected ownership boundary failure, got %#v", parseCheck)
	}
	for _, parseExpected := range []string{"client/main.go imports server-only package database/sql", "client/main.go imports server package example.com/app/server/auth"} {
		if !strings.Contains(parseCheck.Summary, parseExpected) {
			parseT.Fatalf("expected ownership summary to contain %q, got %#v", parseExpected, parseCheck)
		}
	}
}

func TestBuildDoctorStateOwnershipCheckFlagsMixedPersistenceHeuristic(parseT *testing.T) {
	parseTempApp := parseT.TempDir()
	parseClientDir := filepath.Join(parseTempApp, "client")
	if parseErr := os.MkdirAll(parseClientDir, 0755); parseErr != nil {
		parseT.Fatalf("mkdir client dir: %v", parseErr)
	}
	parseClientFile := []byte("//go:build js && wasm\n\npackage client\n\nimport \"github.com/monstercameron/GoWebComponents/fetch\"\n\nfunc mixedOwnership() {\n\t_ = fetch.UseCachedResource(\n\t_ = localStorage\n}\n")
	if parseErr2 := os.WriteFile(filepath.Join(parseClientDir, "state.go"), parseClientFile, 0644); parseErr2 != nil {
		parseT.Fatalf("write state.go: %v", parseErr2)
	}

	parseCheck := buildDoctorStateOwnershipCheck(parseTempApp)
	if parseCheck.Status != "warn" {
		parseT.Fatalf("expected state ownership warning, got %#v", parseCheck)
	}
	if !strings.Contains(parseCheck.Summary, "client/state.go mixes fetch.UseCachedResource with direct browser storage access") {
		parseT.Fatalf("expected mixed ownership summary, got %#v", parseCheck)
	}
}

func TestRunDoctorAuditJSONIncludesGoldenPathReport(parseT *testing.T) {
	parseTempRepo := parseT.TempDir()
	if parseErr := os.MkdirAll(filepath.Join(parseTempRepo, "test", "playwrightgo"), 0755); parseErr != nil {
		parseT.Fatalf("create playwrightgo dir: %v", parseErr)
	}
	parseTempApp := parseT.TempDir()
	if parseErr2 := os.WriteFile(filepath.Join(parseTempApp, "main.go"), []byte("package main\n"), 0644); parseErr2 != nil {
		parseT.Fatalf("write main.go: %v", parseErr2)
	}
	if parseErr3 := os.WriteFile(filepath.Join(parseTempApp, "index.html"), []byte("<!DOCTYPE html>\n"), 0644); parseErr3 != nil {
		parseT.Fatalf("write index.html: %v", parseErr3)
	}
	if parseErr4 := os.MkdirAll(filepath.Join(parseTempApp, "client", "app"), 0755); parseErr4 != nil {
		parseT.Fatalf("mkdir client app: %v", parseErr4)
	}
	parseMutationSource := `package app

func submitSettings() {
	SetSelectedModel()
	DeleteConversation()
}
`
	if parseErr5 := os.WriteFile(filepath.Join(parseTempApp, "client", "app", "mutations.go"), []byte(parseMutationSource), 0644); parseErr5 != nil {
		parseT.Fatalf("write mutations.go: %v", parseErr5)
	}
	if parseErr6 := os.WriteFile(filepath.Join(parseTempApp, "gwc-start.json"), []byte("{\n  \"projectName\": \"audit-app\",\n  \"modulePath\": \"example.com/audit-app\"\n}\n"), 0644); parseErr6 != nil {
		parseT.Fatalf("write gwc-start.json: %v", parseErr6)
	}

	parseStdout, parseRestoreStdout, parseErr7 := captureExamplesStdout()
	if parseErr7 != nil {
		parseT.Fatalf("capture stdout: %v", parseErr7)
	}
	defer parseRestoreStdout()

	parseOriginalLookPath := doctorLookPath
	parseOriginalCommandOutput := doctorCommandOutput
	parseOriginalGetwd := doctorGetwd
	parseOriginalListen := doctorListen
	parseOriginalResolveWasmExec := doctorResolveWasmExec
	parseT.Cleanup(func() {
		doctorLookPath = parseOriginalLookPath
		doctorCommandOutput = parseOriginalCommandOutput
		doctorGetwd = parseOriginalGetwd
		doctorListen = parseOriginalListen
		doctorResolveWasmExec = parseOriginalResolveWasmExec
	})
	doctorLookPath = func(parseName string) (string, error) { return parseName, nil }
	doctorCommandOutput = func(parseName2 string, parseArgs ...string) (string, error) { return parseName2 + " version", nil }
	doctorGetwd = func() (string, error) { return parseTempApp, nil }
	doctorResolveWasmExec = func() (string, error) { return "wasm_exec.js", nil }
	doctorListen = func(parseNetwork string, parseAddress string) (net.Listener, error) {
		return net.Listen("tcp", "127.0.0.1:0")
	}

	parseLauncher := launcher{repoRoot: parseTempRepo}
	if parseErr8 := parseLauncher.run([]string{"doctor", "-audit", "-json", "-port", "8128"}); parseErr8 != nil {
		parseT.Fatalf("run doctor audit json: %v", parseErr8)
	}

	parseOutput, parseErr7 := parseStdout()
	if parseErr7 != nil {
		parseT.Fatalf("read captured stdout: %v", parseErr7)
	}
	var parseReport doctorReport
	if parseErr9 := json.Unmarshal([]byte(parseOutput), &parseReport); parseErr9 != nil {
		parseT.Fatalf("unmarshal doctor report: %v\n%s", parseErr9, parseOutput)
	}
	if parseReport.Audit == nil {
		parseT.Fatalf("expected audit report in JSON output, got %#v", parseReport)
	}
	if parseReport.Audit.Mode != "golden-path" || !parseReport.Audit.OK {
		parseT.Fatalf("expected passing golden-path audit report, got %#v", parseReport.Audit)
	}
	parseChecks := map[string]doctorCheck{}
	for _, parseCheck := range parseReport.Audit.Checks {
		parseChecks[parseCheck.Name] = parseCheck
	}
	parseAppCheck := parseChecks["App entrypoint"]
	if parseAppCheck.RuleID != "audit.app_entrypoint" || parseAppCheck.Severity != "info" || len(parseAppCheck.Locations) != 1 || parseAppCheck.Locations[0] != "main.go" {
		parseT.Fatalf("expected machine-readable app entrypoint metadata, got %#v", parseAppCheck)
	}
	parseMutationCheck := parseChecks["Mutation and resilience"]
	if parseMutationCheck.RuleID != "audit.mutation_resilience" ||
		parseMutationCheck.Severity != "warning" ||
		len(parseMutationCheck.Locations) != 1 ||
		parseMutationCheck.Locations[0] != "client/app/mutations.go" ||
		!strings.Contains(parseMutationCheck.Remediation, "retry posture") {
		parseT.Fatalf("expected machine-readable mutation audit metadata, got %#v", parseMutationCheck)
	}
}

func TestRunDoctorAuditFailurePropagatesIntoExitStatusAndText(parseT *testing.T) {
	parseTempRepo := parseT.TempDir()
	if parseErr := os.MkdirAll(filepath.Join(parseTempRepo, "test", "playwrightgo"), 0755); parseErr != nil {
		parseT.Fatalf("create playwrightgo dir: %v", parseErr)
	}
	parseTempApp := parseT.TempDir()
	if parseErr2 := os.WriteFile(filepath.Join(parseTempApp, "index.html"), []byte("<!DOCTYPE html>\n"), 0644); parseErr2 != nil {
		parseT.Fatalf("write index.html: %v", parseErr2)
	}

	parseStdout, parseRestoreStdout, parseErr3 := captureExamplesStdout()
	if parseErr3 != nil {
		parseT.Fatalf("capture stdout: %v", parseErr3)
	}
	defer parseRestoreStdout()

	parseOriginalLookPath := doctorLookPath
	parseOriginalCommandOutput := doctorCommandOutput
	parseOriginalGetwd := doctorGetwd
	parseOriginalListen := doctorListen
	parseOriginalResolveWasmExec := doctorResolveWasmExec
	parseT.Cleanup(func() {
		doctorLookPath = parseOriginalLookPath
		doctorCommandOutput = parseOriginalCommandOutput
		doctorGetwd = parseOriginalGetwd
		doctorListen = parseOriginalListen
		doctorResolveWasmExec = parseOriginalResolveWasmExec
	})
	doctorLookPath = func(parseName string) (string, error) { return parseName, nil }
	doctorCommandOutput = func(parseName2 string, parseArgs ...string) (string, error) { return parseName2 + " version", nil }
	doctorGetwd = func() (string, error) { return parseTempApp, nil }
	doctorResolveWasmExec = func() (string, error) { return "wasm_exec.js", nil }
	doctorListen = func(parseNetwork string, parseAddress string) (net.Listener, error) {
		return net.Listen("tcp", "127.0.0.1:0")
	}

	parseLauncher := launcher{repoRoot: parseTempRepo}
	parseErr3 = parseLauncher.run([]string{"doctor", "-audit", "-port", "8129"})
	if parseErr3 == nil || !strings.Contains(parseErr3.Error(), "doctor found required checks") {
		parseT.Fatalf("expected doctor audit failure summary, got %v", parseErr3)
	}

	parseOutput, parseErr3 := parseStdout()
	if parseErr3 != nil {
		parseT.Fatalf("read captured stdout: %v", parseErr3)
	}
	for _, parseExpected := range []string{"audit[golden-path]: FAIL", "App entrypoint", "No launcher-detectable app entrypoint was found"} {
		if !strings.Contains(parseOutput, parseExpected) {
			parseT.Fatalf("expected audit output to contain %q, got:\n%s", parseExpected, parseOutput)
		}
	}
}

func TestRunDoctorPrintsPassingReportWithoutError(parseT *testing.T) {
	parseTempRepo := parseT.TempDir()
	if parseErr := os.MkdirAll(filepath.Join(parseTempRepo, "test", "playwrightgo"), 0755); parseErr != nil {
		parseT.Fatalf("create playwrightgo dir: %v", parseErr)
	}
	parseTempApp := parseT.TempDir()
	if parseErr2 := os.WriteFile(filepath.Join(parseTempApp, "main.go"), []byte("package main\n"), 0644); parseErr2 != nil {
		parseT.Fatalf("write main.go: %v", parseErr2)
	}

	parseStdout, parseRestoreStdout, parseErr3 := captureExamplesStdout()
	if parseErr3 != nil {
		parseT.Fatalf("capture stdout: %v", parseErr3)
	}
	defer parseRestoreStdout()

	parseOriginalLookPath := doctorLookPath
	parseOriginalCommandOutput := doctorCommandOutput
	parseOriginalGetwd := doctorGetwd
	parseOriginalListen := doctorListen
	parseOriginalResolveWasmExec := doctorResolveWasmExec
	parseT.Cleanup(func() {
		doctorLookPath = parseOriginalLookPath
		doctorCommandOutput = parseOriginalCommandOutput
		doctorGetwd = parseOriginalGetwd
		doctorListen = parseOriginalListen
		doctorResolveWasmExec = parseOriginalResolveWasmExec
	})
	doctorLookPath = func(parseName string) (string, error) { return parseName, nil }
	doctorCommandOutput = func(parseName2 string, parseArgs ...string) (string, error) { return parseName2 + " version", nil }
	doctorGetwd = func() (string, error) { return parseTempApp, nil }
	doctorResolveWasmExec = func() (string, error) { return "wasm_exec.js", nil }
	doctorListen = func(parseNetwork string, parseAddress string) (net.Listener, error) {
		return net.Listen("tcp", "127.0.0.1:0")
	}

	parseLauncher := launcher{repoRoot: parseTempRepo}
	if parseErr4 := parseLauncher.runDoctor([]string{"-port", "8129"}); parseErr4 != nil {
		parseT.Fatalf("run doctor: %v", parseErr4)
	}
	parseOutput, parseErr3 := parseStdout()
	if parseErr3 != nil {
		parseT.Fatalf("read captured stdout: %v", parseErr3)
	}
	for _, parseExpected := range []string{"GWC doctor: PASS", "Go toolchain", "Port availability"} {
		if !strings.Contains(parseOutput, parseExpected) {
			parseT.Fatalf("expected doctor output to contain %q, got:\n%s", parseExpected, parseOutput)
		}
	}
}

func TestRunVerifyJSONIncludesAuditAndSeverityGate(parseT *testing.T) {
	parseTempApp := parseT.TempDir()
	parseOriginalVerifyExecuteBuild := verifyExecuteBuild
	parseT.Cleanup(func() { verifyExecuteBuild = parseOriginalVerifyExecuteBuild })
	verifyExecuteBuild = func(parseConfig buildConfig) (buildSummary, error) {
		return buildSummary{
			Profile:     buildProfile{Name: parseConfig.profile},
			AppPath:     parseConfig.appPath,
			ProjectRoot: parseConfig.rootPath,
			OutputPath:  filepath.Join(parseConfig.rootPath, "bin", "app.wasm"),
		}, nil
	}
	if parseErr := os.MkdirAll(filepath.Join(parseTempApp, "client", "app"), 0755); parseErr != nil {
		parseT.Fatalf("mkdir client app: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(filepath.Join(parseTempApp, "go.mod"), []byte("module example.com/verify-audit\n\ngo 1.25.0\n"), 0644); parseErr2 != nil {
		parseT.Fatalf("write go.mod: %v", parseErr2)
	}
	if parseErr3 := os.WriteFile(filepath.Join(parseTempApp, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0644); parseErr3 != nil {
		parseT.Fatalf("write main.go: %v", parseErr3)
	}
	if parseErr4 := os.WriteFile(filepath.Join(parseTempApp, "index.html"), []byte("<!DOCTYPE html>\n"), 0644); parseErr4 != nil {
		parseT.Fatalf("write index.html: %v", parseErr4)
	}
	if parseErr5 := os.WriteFile(filepath.Join(parseTempApp, "gwc-start.json"), []byte("{\n  \"projectName\": \"verify-audit\",\n  \"modulePath\": \"example.com/verify-audit\"\n}\n"), 0644); parseErr5 != nil {
		parseT.Fatalf("write gwc-start.json: %v", parseErr5)
	}
	parseMutationSource := `package app

func submitSettings() {
	SetSelectedModel()
	DeleteConversation()
}
`
	if parseErr6 := os.WriteFile(filepath.Join(parseTempApp, "client", "app", "mutations.go"), []byte(parseMutationSource), 0644); parseErr6 != nil {
		parseT.Fatalf("write mutations.go: %v", parseErr6)
	}

	parseStdout, parseRestoreStdout, parseErr7 := captureExamplesStdout()
	if parseErr7 != nil {
		parseT.Fatalf("capture stdout: %v", parseErr7)
	}
	defer parseRestoreStdout()

	parseErr7 = (launcher{}).run([]string{"verify", "-app", filepath.Join(parseTempApp, "main.go"), "-root", parseTempApp, "-skip-tests", "-audit", "-audit-min-severity", "warning", "-json"})
	if parseErr7 == nil || !strings.Contains(parseErr7.Error(), "warning-severity") {
		parseT.Fatalf("expected warning-threshold audit failure, got %v", parseErr7)
	}

	parseOutput, parseReadErr := parseStdout()
	if parseReadErr != nil {
		parseT.Fatalf("read captured stdout: %v", parseReadErr)
	}
	var parseSummary verifySummary
	if parseErr8 := json.Unmarshal([]byte(parseOutput), &parseSummary); parseErr8 != nil {
		parseT.Fatalf("unmarshal verify summary: %v\n%s", parseErr8, parseOutput)
	}
	if parseSummary.OK {
		parseT.Fatalf("expected failing verify summary, got %#v", parseSummary)
	}
	if parseSummary.Audit == nil || parseSummary.AuditMinSeverity != "warning" {
		parseT.Fatalf("expected warning-gated audit summary, got %#v", parseSummary)
	}
	parseChecks := map[string]doctorCheck{}
	for _, parseCheck := range parseSummary.Audit.Checks {
		parseChecks[parseCheck.Name] = parseCheck
	}
	parseMutationCheck := parseChecks["Mutation and resilience"]
	if parseMutationCheck.RuleID != "audit.mutation_resilience" ||
		parseMutationCheck.Severity != "warning" ||
		len(parseMutationCheck.Locations) != 1 ||
		parseMutationCheck.Locations[0] != "client/app/mutations.go" ||
		!strings.Contains(parseMutationCheck.Remediation, "retry posture") {
		parseT.Fatalf("expected machine-readable mutation audit metadata in verify output, got %#v", parseMutationCheck)
	}
}

func TestRunVerifyAuditErrorThresholdAllowsWarnings(parseT *testing.T) {
	parseTempApp := parseT.TempDir()
	parseOriginalVerifyExecuteBuild := verifyExecuteBuild
	parseT.Cleanup(func() { verifyExecuteBuild = parseOriginalVerifyExecuteBuild })
	verifyExecuteBuild = func(parseConfig buildConfig) (buildSummary, error) {
		return buildSummary{
			Profile:     buildProfile{Name: parseConfig.profile},
			AppPath:     parseConfig.appPath,
			ProjectRoot: parseConfig.rootPath,
			OutputPath:  filepath.Join(parseConfig.rootPath, "bin", "app.wasm"),
		}, nil
	}
	if parseErr := os.MkdirAll(filepath.Join(parseTempApp, "client", "app"), 0755); parseErr != nil {
		parseT.Fatalf("mkdir client app: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(filepath.Join(parseTempApp, "go.mod"), []byte("module example.com/verify-audit-pass\n\ngo 1.25.0\n"), 0644); parseErr2 != nil {
		parseT.Fatalf("write go.mod: %v", parseErr2)
	}
	if parseErr3 := os.WriteFile(filepath.Join(parseTempApp, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0644); parseErr3 != nil {
		parseT.Fatalf("write main.go: %v", parseErr3)
	}
	if parseErr4 := os.WriteFile(filepath.Join(parseTempApp, "index.html"), []byte("<!DOCTYPE html>\n"), 0644); parseErr4 != nil {
		parseT.Fatalf("write index.html: %v", parseErr4)
	}
	if parseErr5 := os.WriteFile(filepath.Join(parseTempApp, "gwc-start.json"), []byte("{\n  \"projectName\": \"verify-audit-pass\",\n  \"modulePath\": \"example.com/verify-audit-pass\"\n}\n"), 0644); parseErr5 != nil {
		parseT.Fatalf("write gwc-start.json: %v", parseErr5)
	}
	parseMutationSource := `package app

func submitSettings() {
	SetSelectedModel()
	DeleteConversation()
}
`
	if parseErr6 := os.WriteFile(filepath.Join(parseTempApp, "client", "app", "mutations.go"), []byte(parseMutationSource), 0644); parseErr6 != nil {
		parseT.Fatalf("write mutations.go: %v", parseErr6)
	}

	parseStdout, parseRestoreStdout, parseErr7 := captureExamplesStdout()
	if parseErr7 != nil {
		parseT.Fatalf("capture stdout: %v", parseErr7)
	}
	defer parseRestoreStdout()

	if parseErr8 := (launcher{}).run([]string{"verify", "-app", filepath.Join(parseTempApp, "main.go"), "-root", parseTempApp, "-skip-tests", "-audit", "-audit-min-severity", "error", "-json"}); parseErr8 != nil {
		parseT.Fatalf("expected warning-only audit to pass error threshold, got %v", parseErr8)
	}

	parseOutput, parseReadErr := parseStdout()
	if parseReadErr != nil {
		parseT.Fatalf("read captured stdout: %v", parseReadErr)
	}
	var parseSummary verifySummary
	if parseErr9 := json.Unmarshal([]byte(parseOutput), &parseSummary); parseErr9 != nil {
		parseT.Fatalf("unmarshal verify summary: %v\n%s", parseErr9, parseOutput)
	}
	if !parseSummary.OK || parseSummary.Audit == nil || parseSummary.AuditMinSeverity != "error" {
		parseT.Fatalf("expected passing verify summary with error-only threshold, got %#v", parseSummary)
	}
}

func TestRunDoctorAuditJSONEmitsAuditSection(parseT *testing.T) {
	parseTempRepo := parseT.TempDir()
	if parseErr := os.MkdirAll(filepath.Join(parseTempRepo, "test", "playwrightgo"), 0755); parseErr != nil {
		parseT.Fatalf("create playwrightgo dir: %v", parseErr)
	}
	parseTempApp := parseT.TempDir()
	if parseErr2 := os.WriteFile(filepath.Join(parseTempApp, "main.go"), []byte("package main\n"), 0644); parseErr2 != nil {
		parseT.Fatalf("write main.go: %v", parseErr2)
	}

	parseStdout, parseRestoreStdout, parseErr3 := captureExamplesStdout()
	if parseErr3 != nil {
		parseT.Fatalf("capture stdout: %v", parseErr3)
	}
	defer parseRestoreStdout()

	parseOriginalLookPath := doctorLookPath
	parseOriginalCommandOutput := doctorCommandOutput
	parseOriginalGetwd := doctorGetwd
	parseOriginalListen := doctorListen
	parseOriginalResolveWasmExec := doctorResolveWasmExec
	parseT.Cleanup(func() {
		doctorLookPath = parseOriginalLookPath
		doctorCommandOutput = parseOriginalCommandOutput
		doctorGetwd = parseOriginalGetwd
		doctorListen = parseOriginalListen
		doctorResolveWasmExec = parseOriginalResolveWasmExec
	})
	doctorLookPath = func(parseName string) (string, error) { return parseName, nil }
	doctorCommandOutput = func(parseName2 string, parseArgs ...string) (string, error) { return parseName2 + " version", nil }
	doctorGetwd = func() (string, error) { return parseTempApp, nil }
	doctorResolveWasmExec = func() (string, error) { return "wasm_exec.js", nil }
	doctorListen = func(parseNetwork string, parseAddress string) (net.Listener, error) {
		return net.Listen("tcp", "127.0.0.1:0")
	}

	if parseErr4 := (launcher{repoRoot: parseTempRepo}).run([]string{"doctor", "-audit", "-json", "-port", "8128"}); parseErr4 != nil {
		parseT.Fatalf("run doctor audit json: %v", parseErr4)
	}

	parseOutput, parseErr3 := parseStdout()
	if parseErr3 != nil {
		parseT.Fatalf("read captured stdout: %v", parseErr3)
	}
	var parseReport doctorReport
	if parseErr5 := json.Unmarshal([]byte(parseOutput), &parseReport); parseErr5 != nil {
		parseT.Fatalf("unmarshal doctor audit report: %v\n%s", parseErr5, parseOutput)
	}
	if parseReport.Audit == nil || parseReport.Audit.Mode != "golden-path" {
		parseT.Fatalf("expected golden-path audit JSON payload, got %#v", parseReport)
	}
	isParseFoundRuleMetadata := false
	for _, parseCheck := range parseReport.Audit.Checks {
		if parseCheck.RuleID != "" && parseCheck.Severity != "" {
			isParseFoundRuleMetadata = true
			break
		}
	}
	if !isParseFoundRuleMetadata {
		parseT.Fatalf("expected machine-readable rule metadata in audit JSON, got %#v", parseReport.Audit)
	}
}

func TestRunDoctorAuditAdvisoryPolicySupportsNamedSuppressions(parseT *testing.T) {
	parseTempRepo := parseT.TempDir()
	if parseErr := os.MkdirAll(filepath.Join(parseTempRepo, "test", "playwrightgo"), 0755); parseErr != nil {
		parseT.Fatalf("create playwrightgo dir: %v", parseErr)
	}
	parseTempApp := parseT.TempDir()

	parseStdout, parseRestoreStdout, parseErr2 := captureExamplesStdout()
	if parseErr2 != nil {
		parseT.Fatalf("capture stdout: %v", parseErr2)
	}
	defer parseRestoreStdout()

	parseOriginalLookPath := doctorLookPath
	parseOriginalCommandOutput := doctorCommandOutput
	parseOriginalGetwd := doctorGetwd
	parseOriginalListen := doctorListen
	parseOriginalResolveWasmExec := doctorResolveWasmExec
	parseT.Cleanup(func() {
		doctorLookPath = parseOriginalLookPath
		doctorCommandOutput = parseOriginalCommandOutput
		doctorGetwd = parseOriginalGetwd
		doctorListen = parseOriginalListen
		doctorResolveWasmExec = parseOriginalResolveWasmExec
	})
	doctorLookPath = func(parseName string) (string, error) { return parseName, nil }
	doctorCommandOutput = func(parseName2 string, parseArgs ...string) (string, error) { return parseName2 + " version", nil }
	doctorGetwd = func() (string, error) { return parseTempApp, nil }
	doctorResolveWasmExec = func() (string, error) { return "wasm_exec.js", nil }
	doctorListen = func(parseNetwork string, parseAddress string) (net.Listener, error) {
		return net.Listen("tcp", "127.0.0.1:0")
	}

	if parseErr3 := (launcher{repoRoot: parseTempRepo}).run([]string{"doctor", "-audit", "-audit-policy", "advisory", "-audit-suppress", "App entrypoint", "-json", "-port", "8126"}); parseErr3 != nil {
		parseT.Fatalf("run doctor audit advisory json: %v", parseErr3)
	}

	parseOutput, parseErr2 := parseStdout()
	if parseErr2 != nil {
		parseT.Fatalf("read captured stdout: %v", parseErr2)
	}
	var parseReport doctorReport
	if parseErr4 := json.Unmarshal([]byte(parseOutput), &parseReport); parseErr4 != nil {
		parseT.Fatalf("unmarshal doctor advisory report: %v\n%s", parseErr4, parseOutput)
	}
	if parseReport.Audit == nil || parseReport.Audit.Policy != "advisory" {
		parseT.Fatalf("expected advisory audit policy, got %#v", parseReport)
	}
	if len(parseReport.Audit.Suppressed) != 1 || parseReport.Audit.Suppressed[0] != "App entrypoint" {
		parseT.Fatalf("expected named suppression to be recorded, got %#v", parseReport.Audit)
	}
	for _, parseCheck := range parseReport.Audit.Checks {
		if parseCheck.Name == "App entrypoint" && parseCheck.Status != "suppressed" {
			parseT.Fatalf("expected app entrypoint to be suppressed, got %#v", parseCheck)
		}
	}
}

func TestRunDoctorAuditWriteAndReadBaseline(parseT *testing.T) {
	parseTempRepo := parseT.TempDir()
	if parseErr := os.MkdirAll(filepath.Join(parseTempRepo, "test", "playwrightgo"), 0755); parseErr != nil {
		parseT.Fatalf("create playwrightgo dir: %v", parseErr)
	}
	parseTempApp := parseT.TempDir()
	parseBaselinePath := filepath.Join(parseTempApp, "audit-baseline.json")

	parseOriginalLookPath := doctorLookPath
	parseOriginalCommandOutput := doctorCommandOutput
	parseOriginalGetwd := doctorGetwd
	parseOriginalListen := doctorListen
	parseOriginalResolveWasmExec := doctorResolveWasmExec
	parseT.Cleanup(func() {
		doctorLookPath = parseOriginalLookPath
		doctorCommandOutput = parseOriginalCommandOutput
		doctorGetwd = parseOriginalGetwd
		doctorListen = parseOriginalListen
		doctorResolveWasmExec = parseOriginalResolveWasmExec
	})
	doctorLookPath = func(parseName string) (string, error) { return parseName, nil }
	doctorCommandOutput = func(parseName2 string, parseArgs ...string) (string, error) { return parseName2 + " version", nil }
	doctorGetwd = func() (string, error) { return parseTempApp, nil }
	doctorResolveWasmExec = func() (string, error) { return "wasm_exec.js", nil }
	doctorListen = func(parseNetwork string, parseAddress string) (net.Listener, error) {
		return net.Listen("tcp", "127.0.0.1:0")
	}

	if parseErr2 := (launcher{repoRoot: parseTempRepo}).runDoctor([]string{"-audit", "-audit-policy", "advisory", "-audit-write-baseline", parseBaselinePath, "-port", "8124"}); parseErr2 != nil {
		parseT.Fatalf("write audit baseline: %v", parseErr2)
	}
	parseContent, parseErr3 := os.ReadFile(parseBaselinePath)
	if parseErr3 != nil {
		parseT.Fatalf("read baseline file: %v", parseErr3)
	}
	var parseBaseline doctorAuditBaseline
	if parseErr4 := json.Unmarshal(parseContent, &parseBaseline); parseErr4 != nil {
		parseT.Fatalf("unmarshal baseline: %v\n%s", parseErr4, string(parseContent))
	}
	isParseFoundAppEntrypoint := false
	for _, parseCheck := range parseBaseline.Checks {
		if parseCheck.Name == "App entrypoint" {
			isParseFoundAppEntrypoint = true
		}
	}
	if !isParseFoundAppEntrypoint {
		parseT.Fatalf("expected baseline to capture current app entrypoint finding, got %#v", parseBaseline)
	}

	parseStdout, parseRestoreStdout, parseErr3 := captureExamplesStdout()
	if parseErr3 != nil {
		parseT.Fatalf("capture stdout: %v", parseErr3)
	}
	defer parseRestoreStdout()

	if parseErr5 := (launcher{repoRoot: parseTempRepo}).run([]string{"doctor", "-audit", "-audit-policy", "advisory", "-audit-baseline", parseBaselinePath, "-json", "-port", "8122"}); parseErr5 != nil {
		parseT.Fatalf("run doctor with audit baseline: %v", parseErr5)
	}
	parseOutput, parseErr3 := parseStdout()
	if parseErr3 != nil {
		parseT.Fatalf("read captured stdout: %v", parseErr3)
	}
	var parseReport doctorReport
	if parseErr6 := json.Unmarshal([]byte(parseOutput), &parseReport); parseErr6 != nil {
		parseT.Fatalf("unmarshal doctor report with baseline: %v\n%s", parseErr6, parseOutput)
	}
	if parseReport.Audit == nil || parseReport.Audit.BaselinePath == "" {
		parseT.Fatalf("expected audit baseline path in report, got %#v", parseReport)
	}
	if len(parseReport.Audit.Suppressed) == 0 {
		parseT.Fatalf("expected baseline-suppressed audit checks, got %#v", parseReport.Audit)
	}
}

func TestRunDoctorAuditReturnsErrorWhenAuditFindingsFail(parseT *testing.T) {
	parseTempRepo := parseT.TempDir()
	if parseErr := os.MkdirAll(filepath.Join(parseTempRepo, "test", "playwrightgo"), 0755); parseErr != nil {
		parseT.Fatalf("create playwrightgo dir: %v", parseErr)
	}

	parseOriginalLookPath := doctorLookPath
	parseOriginalCommandOutput := doctorCommandOutput
	parseOriginalGetwd := doctorGetwd
	parseOriginalListen := doctorListen
	parseOriginalResolveWasmExec := doctorResolveWasmExec
	parseT.Cleanup(func() {
		doctorLookPath = parseOriginalLookPath
		doctorCommandOutput = parseOriginalCommandOutput
		doctorGetwd = parseOriginalGetwd
		doctorListen = parseOriginalListen
		doctorResolveWasmExec = parseOriginalResolveWasmExec
	})
	doctorLookPath = func(parseName string) (string, error) { return parseName, nil }
	doctorCommandOutput = func(parseName2 string, parseArgs ...string) (string, error) { return parseName2 + " version", nil }
	doctorGetwd = func() (string, error) { return parseT.TempDir(), nil }
	doctorResolveWasmExec = func() (string, error) { return "wasm_exec.js", nil }
	doctorListen = func(parseNetwork string, parseAddress string) (net.Listener, error) {
		return net.Listen("tcp", "127.0.0.1:0")
	}

	parseErr2 := (launcher{repoRoot: parseTempRepo}).runDoctor([]string{"-audit", "-port", "8133"})
	if parseErr2 == nil || !strings.Contains(parseErr2.Error(), "doctor found required checks") {
		parseT.Fatalf("expected audit failure to fail doctor, got %v", parseErr2)
	}
}

func TestRunDoctorReturnsErrorWhenChecksFail(parseT *testing.T) {
	parseOriginalLookPath := doctorLookPath
	parseOriginalGetwd := doctorGetwd
	parseT.Cleanup(func() {
		doctorLookPath = parseOriginalLookPath
		doctorGetwd = parseOriginalGetwd
	})

	doctorLookPath = func(parseName string) (string, error) {
		return "", errors.New("missing")
	}
	doctorGetwd = func() (string, error) { return parseT.TempDir(), nil }

	parseLauncher := launcher{repoRoot: parseT.TempDir()}
	parseErr := parseLauncher.runDoctor([]string{"-port", "8131"})
	if parseErr == nil {
		parseT.Fatal("expected doctor to fail when required checks fail")
	}
	if !strings.Contains(parseErr.Error(), "doctor found required checks") {
		parseT.Fatalf("expected doctor failure summary, got %v", parseErr)
	}
}

func TestRunDoctorHandlesHelpAndInvalidFlags(parseT *testing.T) {
	parseLauncher := launcher{repoRoot: parseT.TempDir()}
	if parseErr := parseLauncher.runDoctor([]string{"-help"}); parseErr != nil {
		parseT.Fatalf("expected doctor help to succeed, got %v", parseErr)
	}
	parseErr2 := parseLauncher.runDoctor([]string{"-definitely-invalid"})
	if parseErr2 == nil || !strings.Contains(parseErr2.Error(), "flag provided but not defined") {
		parseT.Fatalf("expected invalid doctor flag error, got %v", parseErr2)
	}
}

func TestRunExamplesDoesNotPrintListeningURLsWhenBindFails(parseT *testing.T) {
	parseTempExamples := parseT.TempDir()
	parseListener, parseErr := net.Listen("tcp", "127.0.0.1:0")
	if parseErr != nil {
		parseT.Fatalf("listen: %v", parseErr)
	}
	defer parseListener.Close()

	parseStdout, parseRestoreStdout, parseErr := captureExamplesStdout()
	if parseErr != nil {
		parseT.Fatalf("capture stdout: %v", parseErr)
	}
	defer parseRestoreStdout()

	parseLauncher := launcher{examplesDir: parseTempExamples, staticDir: parseTempExamples}
	parseErr = parseLauncher.runExamples([]string{"-host", "127.0.0.1", "-port", strconv.Itoa(parseListener.Addr().(*net.TCPAddr).Port)})
	if parseErr == nil {
		parseT.Fatal("expected bind failure")
	}

	parseOutput, parseReadErr := parseStdout()
	if parseReadErr != nil {
		parseT.Fatalf("read captured stdout: %v", parseReadErr)
	}
	if strings.Contains(parseOutput, "GWC examples server listening on") || strings.Contains(parseOutput, "Examples: http://") {
		parseT.Fatalf("expected bind failure to avoid misleading listening output, got %q", parseOutput)
	}
	if !strings.Contains(parseErr.Error(), "bind") {
		parseT.Fatalf("expected bind error, got %v", parseErr)
	}
}

func TestRunExamplesPrintsListeningURLsOnServe(parseT *testing.T) {
	parseTempExamples := parseT.TempDir()
	parseStdout, parseRestoreStdout, parseErr := captureExamplesStdout()
	if parseErr != nil {
		parseT.Fatalf("capture stdout: %v", parseErr)
	}
	defer parseRestoreStdout()

	parseOriginalExamplesListen := examplesListen
	parseOriginalExamplesServe := examplesServe
	parseT.Cleanup(func() {
		examplesListen = parseOriginalExamplesListen
		examplesServe = parseOriginalExamplesServe
	})

	parseRealListener, parseErr := net.Listen("tcp", "127.0.0.1:0")
	if parseErr != nil {
		parseT.Fatalf("listen: %v", parseErr)
	}
	examplesListen = func(parseNetwork string, parseAddress string) (net.Listener, error) {
		return parseRealListener, nil
	}
	examplesServe = func(parseServer *http.Server, parseListener net.Listener) error {
		if parseServer == nil || parseListener == nil {
			parseT.Fatal("expected server and listener to be provided")
		}
		if !strings.Contains(parseServer.Addr, "127.0.0.1:") {
			parseT.Fatalf("expected server addr to include provided host, got %q", parseServer.Addr)
		}
		if parseServer.Handler == nil {
			parseT.Fatal("expected examples handler to be configured")
		}
		return http.ErrServerClosed
	}

	parseLauncher := launcher{examplesDir: parseTempExamples, staticDir: parseTempExamples}
	if parseErr2 := parseLauncher.runExamples([]string{"-host", "127.0.0.1", "-port", strconv.Itoa(parseRealListener.Addr().(*net.TCPAddr).Port)}); parseErr2 != nil {
		parseT.Fatalf("run examples: %v", parseErr2)
	}
	parseOutput, parseErr := parseStdout()
	if parseErr != nil {
		parseT.Fatalf("read captured stdout: %v", parseErr)
	}
	for _, parseExpected := range []string{"GWC examples server listening on http://127.0.0.1:", "Examples: http://127.0.0.1:", "Counter:  http://127.0.0.1:"} {
		if !strings.Contains(parseOutput, parseExpected) {
			parseT.Fatalf("expected examples output to contain %q, got %q", parseExpected, parseOutput)
		}
	}
}

func TestRunExamplesTreatsServerClosedAsCleanShutdown(parseT *testing.T) {
	parseTempExamples := parseT.TempDir()
	parseStdout, parseRestoreStdout, parseErr := captureExamplesStdout()
	if parseErr != nil {
		parseT.Fatalf("capture stdout: %v", parseErr)
	}
	defer parseRestoreStdout()

	parseOriginalExamplesListen := examplesListen
	parseOriginalExamplesServe := examplesServe
	parseT.Cleanup(func() {
		examplesListen = parseOriginalExamplesListen
		examplesServe = parseOriginalExamplesServe
	})

	parseRealListener, parseErr := net.Listen("tcp", "127.0.0.1:0")
	if parseErr != nil {
		parseT.Fatalf("listen: %v", parseErr)
	}
	examplesListen = func(parseNetwork string, parseAddress string) (net.Listener, error) {
		return parseRealListener, nil
	}
	examplesServe = func(parseServer *http.Server, parseListener net.Listener) error {
		return http.ErrServerClosed
	}

	parseLauncher := launcher{examplesDir: parseTempExamples, staticDir: parseTempExamples}
	if parseErr2 := parseLauncher.runExamples([]string{"-host", "127.0.0.1", "-port", strconv.Itoa(parseRealListener.Addr().(*net.TCPAddr).Port)}); parseErr2 != nil {
		parseT.Fatalf("expected clean shutdown, got %v", parseErr2)
	}
	parseOutput, parseErr := parseStdout()
	if parseErr != nil {
		parseT.Fatalf("read stdout: %v", parseErr)
	}
	if !strings.Contains(parseOutput, "GWC examples server listening on") {
		parseT.Fatalf("expected listening output before clean shutdown, got %q", parseOutput)
	}
}

func TestRunExamplesHandlesHelpInvalidFlagAndServeError(parseT *testing.T) {
	parseLauncher := launcher{examplesDir: parseT.TempDir(), staticDir: parseT.TempDir()}
	if parseErr := parseLauncher.runExamples([]string{"-help"}); parseErr != nil {
		parseT.Fatalf("expected examples help to succeed, got %v", parseErr)
	}
	parseErr2 := parseLauncher.runExamples([]string{"-definitely-invalid"})
	if parseErr2 == nil || !strings.Contains(parseErr2.Error(), "flag provided but not defined") {
		parseT.Fatalf("expected invalid examples flag error, got %v", parseErr2)
	}

	parseOriginalExamplesListen := examplesListen
	parseOriginalExamplesServe := examplesServe
	parseT.Cleanup(func() {
		examplesListen = parseOriginalExamplesListen
		examplesServe = parseOriginalExamplesServe
	})

	parseRealListener, parseErr2 := net.Listen("tcp", "127.0.0.1:0")
	if parseErr2 != nil {
		parseT.Fatalf("listen: %v", parseErr2)
	}
	examplesListen = func(parseNetwork string, parseAddress string) (net.Listener, error) {
		return parseRealListener, nil
	}
	examplesServe = func(parseServer *http.Server, parseListener net.Listener) error {
		return errors.New("serve failed")
	}

	parseErr2 = parseLauncher.runExamples([]string{"-host", "127.0.0.1", "-port", strconv.Itoa(parseRealListener.Addr().(*net.TCPAddr).Port)})
	if parseErr2 == nil || !strings.Contains(parseErr2.Error(), "serve failed") {
		parseT.Fatalf("expected serve failure to bubble, got %v", parseErr2)
	}
}

func TestRunExamplesWritesStaticCatalogAndReportsOutputPath(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseExamplesDir := filepath.Join(parseRoot, "examples")
	parseStaticDir := filepath.Join(parseRoot, "static")
	if parseErr := os.MkdirAll(filepath.Join(parseExamplesDir, "01-first"), 0755); parseErr != nil {
		parseT.Fatalf("mkdir example dir: %v", parseErr)
	}
	if parseErr2 := os.MkdirAll(filepath.Join(parseStaticDir, "bin"), 0755); parseErr2 != nil {
		parseT.Fatalf("mkdir static bin dir: %v", parseErr2)
	}
	if parseErr3 := os.WriteFile(filepath.Join(parseExamplesDir, "01-first", "index.html"), []byte("<title>First</title>"), 0644); parseErr3 != nil {
		parseT.Fatalf("write html: %v", parseErr3)
	}

	parseStdout, parseRestoreStdout, parseErr4 := captureExamplesStdout()
	if parseErr4 != nil {
		parseT.Fatalf("capture stdout: %v", parseErr4)
	}
	defer parseRestoreStdout()

	parseTargetPath := filepath.Join(parseRoot, "exports", "catalog.json")
	parseLauncher := launcher{examplesDir: parseExamplesDir, staticDir: parseStaticDir}
	if parseErr5 := parseLauncher.runExamples([]string{"-export-static-catalog", parseTargetPath}); parseErr5 != nil {
		parseT.Fatalf("run examples export static catalog: %v", parseErr5)
	}
	if _, parseErr6 := os.Stat(parseTargetPath); parseErr6 != nil {
		parseT.Fatalf("expected static catalog file to exist: %v", parseErr6)
	}
	parseOutput, parseErr4 := parseStdout()
	if parseErr4 != nil {
		parseT.Fatalf("read captured stdout: %v", parseErr4)
	}
	if !strings.Contains(parseOutput, "Wrote static examples catalog to "+parseTargetPath) {
		parseT.Fatalf("expected export confirmation output, got %q", parseOutput)
	}
}

// TestRunExamplesBuildPublicSiteStagesWasmBinaries verifies that the launcher-owned public site build writes the site shell and first embedded example wasm binaries into their runtime locations.
func TestRunExamplesBuildPublicSiteStagesWasmBinaries(parseT *testing.T) {
	buildRoot := parseT.TempDir()
	buildExamplesDir := filepath.Join(buildRoot, "examples")
	buildStaticDir := filepath.Join(buildExamplesDir, "static")
	buildWasmDir := filepath.Join(buildRoot, "bin", "examples")
	buildCatalogPath := filepath.Join(buildExamplesDir, "public-examples-site", "assets", "data", "catalog.json")

	if buildErr := os.MkdirAll(filepath.Join(buildExamplesDir, "public-examples-site"), 0755); buildErr != nil {
		parseT.Fatalf("mkdir public-examples-site: %v", buildErr)
	}
	if buildErr := os.MkdirAll(filepath.Join(buildExamplesDir, "public", "counter"), 0755); buildErr != nil {
		parseT.Fatalf("mkdir counter example: %v", buildErr)
	}
	if buildErr := os.MkdirAll(filepath.Join(buildExamplesDir, "public", "text-input"), 0755); buildErr != nil {
		parseT.Fatalf("mkdir text-input example: %v", buildErr)
	}
	if buildErr := os.WriteFile(filepath.Join(buildExamplesDir, "public-examples-site", "main.go"), []byte("package main\nfunc main() {}\n"), 0644); buildErr != nil {
		parseT.Fatalf("write public-examples-site main.go: %v", buildErr)
	}
	if buildErr := os.WriteFile(filepath.Join(buildExamplesDir, "public", "counter", "main.go"), []byte("package main\nfunc main() {}\n"), 0644); buildErr != nil {
		parseT.Fatalf("write counter main.go: %v", buildErr)
	}
	if buildErr := os.WriteFile(filepath.Join(buildExamplesDir, "public", "counter", "README.md"), []byte("# Counter\n"), 0644); buildErr != nil {
		parseT.Fatalf("write counter README: %v", buildErr)
	}
	if buildErr := os.WriteFile(filepath.Join(buildExamplesDir, "public", "text-input", "main.go"), []byte("package main\nfunc main() {}\n"), 0644); buildErr != nil {
		parseT.Fatalf("write text-input main.go: %v", buildErr)
	}
	if buildErr := os.MkdirAll(filepath.Join(buildExamplesDir, "public-examples-site", "assets", "code", "stale-example"), 0755); buildErr != nil {
		parseT.Fatalf("mkdir stale mirror dir: %v", buildErr)
	}
	if buildErr := os.WriteFile(filepath.Join(buildExamplesDir, "public-examples-site", "assets", "code", "stale-example", "old.go"), []byte("package stale\n"), 0644); buildErr != nil {
		parseT.Fatalf("write stale mirror file: %v", buildErr)
	}
	if buildErr := os.MkdirAll(filepath.Dir(buildCatalogPath), 0755); buildErr != nil {
		parseT.Fatalf("mkdir catalog dir: %v", buildErr)
	}
	buildCatalogJSON := `{"modules":["all","core","forms","state"],"statuses":["all","experimental","stable"],"levels":["all","Beginner","Core","Intermediate","Advanced"],"filters":["All","Concept","API","Example"],"sortOptions":[{"value":"relevance","label":"Relevance"},{"value":"alpha","label":"A-Z"},{"value":"level","label":"Level"}],"items":[{"id":1,"title":"Overview","status":"stable","module":"core","type":"Concept","level":"Core","tags":["overview"],"blurb":"Base concept entry.","readTime":"5 min read","content":{"kind":"article","sourcePath":"assets/docs/overview.md","sections":[{"heading":"Overview","paragraphs":["Base concept entry."]}]}}]}`
	if buildErr := os.WriteFile(buildCatalogPath, []byte(buildCatalogJSON), 0644); buildErr != nil {
		parseT.Fatalf("write catalog fixture: %v", buildErr)
	}

	buildOriginalExecuteExamplesBuild := executeExamplesBuild
	parseT.Cleanup(func() {
		executeExamplesBuild = buildOriginalExecuteExamplesBuild
	})

	buildCalls := make([]buildConfig, 0, 3)
	executeExamplesBuild = func(buildConfig buildConfig) (buildSummary, error) {
		buildCalls = append(buildCalls, buildConfig)
		if buildErr := os.MkdirAll(filepath.Dir(buildConfig.outputPath), 0755); buildErr != nil {
			return buildSummary{}, buildErr
		}
		if buildErr := os.WriteFile(buildConfig.outputPath, []byte(filepath.Base(buildConfig.outputPath)), 0644); buildErr != nil {
			return buildSummary{}, buildErr
		}
		return buildSummary{
			OK:          true,
			Profile:     buildProfile{Name: buildConfig.profile},
			AppPath:     buildConfig.appPath,
			ProjectRoot: buildConfig.rootPath,
			PackageDir:  filepath.Dir(buildConfig.appPath),
			OutputPath:  buildConfig.outputPath,
			Resolution:  buildConfig.resolution,
		}, nil
	}

	buildLauncher := launcher{
		repoRoot:        buildRoot,
		examplesDir:     buildExamplesDir,
		staticDir:       buildStaticDir,
		examplesWasmDir: buildWasmDir,
	}
	if buildErr := buildLauncher.runExamples([]string{"build-public-site"}); buildErr != nil {
		parseT.Fatalf("run examples build-public-site: %v", buildErr)
	}

	if len(buildCalls) != 3 {
		parseT.Fatalf("expected three build calls, got %d", len(buildCalls))
	}

	buildCoreWasmPath := filepath.Join(buildWasmDir, "public-examples-site.wasm")
	if _, buildErr := os.Stat(buildCoreWasmPath); buildErr != nil {
		parseT.Fatalf("expected public-examples-site wasm to exist: %v", buildErr)
	}

	buildCounterWasmPath := filepath.Join(buildWasmDir, "counter.wasm")
	if _, buildErr := os.Stat(buildCounterWasmPath); buildErr != nil {
		parseT.Fatalf("expected counter wasm to exist: %v", buildErr)
	}

	buildStaticCorePath := filepath.Join(buildStaticDir, "bin", "public-examples-site.wasm")
	if _, buildErr := os.Stat(buildStaticCorePath); buildErr != nil {
		parseT.Fatalf("expected static public-examples-site wasm copy to exist: %v", buildErr)
	}

	buildStaticCounterPath := filepath.Join(buildStaticDir, "bin", "counter.wasm")
	if _, buildErr := os.Stat(buildStaticCounterPath); buildErr != nil {
		parseT.Fatalf("expected static counter wasm copy to exist: %v", buildErr)
	}

	buildEmbeddedCounterPath := filepath.Join(buildExamplesDir, "public-examples-site", "assets", "bins", "counter.wasm")
	buildEmbeddedCounterBytes, buildErr := os.ReadFile(buildEmbeddedCounterPath)
	if buildErr != nil {
		parseT.Fatalf("expected embedded counter wasm copy to exist: %v", buildErr)
	}
	if string(buildEmbeddedCounterBytes) != "counter.wasm" {
		parseT.Fatalf("unexpected embedded counter wasm copy contents: %q", string(buildEmbeddedCounterBytes))
	}

	buildTextInputWasmPath := filepath.Join(buildWasmDir, "text-input.wasm")
	if _, buildErr := os.Stat(buildTextInputWasmPath); buildErr != nil {
		parseT.Fatalf("expected text-input wasm to exist: %v", buildErr)
	}

	buildStaticTextInputPath := filepath.Join(buildStaticDir, "bin", "text-input.wasm")
	if _, buildErr := os.Stat(buildStaticTextInputPath); buildErr != nil {
		parseT.Fatalf("expected static text-input wasm copy to exist: %v", buildErr)
	}

	buildEmbeddedTextInputPath := filepath.Join(buildExamplesDir, "public-examples-site", "assets", "bins", "text-input.wasm")
	buildEmbeddedTextInputBytes, buildErr := os.ReadFile(buildEmbeddedTextInputPath)
	if buildErr != nil {
		parseT.Fatalf("expected embedded text-input wasm copy to exist: %v", buildErr)
	}
	if string(buildEmbeddedTextInputBytes) != "text-input.wasm" {
		parseT.Fatalf("unexpected embedded text-input wasm copy contents: %q", string(buildEmbeddedTextInputBytes))
	}

	buildCounterPreviewIndexPath := filepath.Join(buildExamplesDir, "public-examples-site", "assets", "examples", "counter", "index.html")
	buildCounterPreviewIndexBytes, buildErr := os.ReadFile(buildCounterPreviewIndexPath)
	if buildErr != nil {
		parseT.Fatalf("expected counter preview index.html to exist: %v", buildErr)
	}
	if !strings.Contains(string(buildCounterPreviewIndexBytes), "./app.wasm") || !strings.Contains(string(buildCounterPreviewIndexBytes), "loadCachedWasmWithProgress") || !strings.Contains(string(buildCounterPreviewIndexBytes), "clearPreviewCacheOverflow") || !strings.Contains(string(buildCounterPreviewIndexBytes), "previewCacheRetainCount = 2") {
		parseT.Fatalf("expected counter preview host to boot app.wasm, got %q", string(buildCounterPreviewIndexBytes))
	}

	buildCounterPreviewWasmPath := filepath.Join(buildExamplesDir, "public-examples-site", "assets", "examples", "counter", "app.wasm")
	buildCounterPreviewWasmBytes, buildErr := os.ReadFile(buildCounterPreviewWasmPath)
	if buildErr != nil {
		parseT.Fatalf("expected counter preview wasm to exist: %v", buildErr)
	}
	if string(buildCounterPreviewWasmBytes) != "counter.wasm" {
		parseT.Fatalf("unexpected counter preview wasm copy contents: %q", string(buildCounterPreviewWasmBytes))
	}

	buildTextInputPreviewIndexPath := filepath.Join(buildExamplesDir, "public-examples-site", "assets", "examples", "text-input", "index.html")
	if _, buildErr := os.Stat(buildTextInputPreviewIndexPath); buildErr != nil {
		parseT.Fatalf("expected text-input preview index.html to exist: %v", buildErr)
	}

	buildLoggerScriptPath := filepath.Join(buildStaticDir, "script", "example-logger.js")
	buildLoggerScriptBytes, buildErr := os.ReadFile(buildLoggerScriptPath)
	if buildErr != nil {
		parseT.Fatalf("expected example logger script to exist: %v", buildErr)
	}
	if !strings.Contains(string(buildLoggerScriptBytes), "window.__gwcExampleLogger") {
		parseT.Fatalf("expected example logger script contents, got %q", string(buildLoggerScriptBytes))
	}

	buildMirroredCounterPath := filepath.Join(buildExamplesDir, "public-examples-site", "assets", "code", "counter", "main.go")
	if _, buildErr := os.Stat(buildMirroredCounterPath); buildErr != nil {
		parseT.Fatalf("expected mirrored counter main.go to exist: %v", buildErr)
	}

	buildMirroredTextInputPath := filepath.Join(buildExamplesDir, "public-examples-site", "assets", "code", "text-input", "main.go")
	if _, buildErr := os.Stat(buildMirroredTextInputPath); buildErr != nil {
		parseT.Fatalf("expected mirrored text-input main.go to exist: %v", buildErr)
	}

	buildMirroredCounterReadmePath := filepath.Join(buildExamplesDir, "public-examples-site", "assets", "code", "counter", "README.md")
	if _, buildErr := os.Stat(buildMirroredCounterReadmePath); buildErr != nil {
		parseT.Fatalf("expected mirrored counter README to exist: %v", buildErr)
	}

	buildStaleMirrorPath := filepath.Join(buildExamplesDir, "public-examples-site", "assets", "code", "stale-example", "old.go")
	if _, buildErr := os.Stat(buildStaleMirrorPath); !os.IsNotExist(buildErr) {
		parseT.Fatalf("expected stale mirror file to be removed, got err=%v", buildErr)
	}

	buildCatalogBytes, buildErr := os.ReadFile(buildCatalogPath)
	if buildErr != nil {
		parseT.Fatalf("expected regenerated catalog to exist: %v", buildErr)
	}
	for _, buildSnippet := range []string{`"title": "Counter"`, `"embedPath": "assets/bins/counter.wasm"`, `"previewPath": "assets/examples/counter/index.html"`, `"title": "Text Input"`, `"sourcePath": "assets/code/text-input/main.go"`} {
		if !strings.Contains(string(buildCatalogBytes), buildSnippet) {
			parseT.Fatalf("expected regenerated catalog to contain %q, got %s", buildSnippet, string(buildCatalogBytes))
		}
	}
}

// TestRunExamplesBuildPublicSiteHelp verifies that the public site build action exposes ordinary flag help without starting a server.
func TestRunExamplesBuildPublicSiteHelp(parseT *testing.T) {
	buildLauncher := launcher{examplesDir: parseT.TempDir(), staticDir: parseT.TempDir()}
	if buildErr := buildLauncher.runExamples([]string{"build-public-site", "-help"}); buildErr != nil {
		parseT.Fatalf("expected build-public-site help to succeed, got %v", buildErr)
	}
}

func TestRunExamplesErrorsWhenExamplesDirectoryIsMissing(parseT *testing.T) {
	parseLauncher := launcher{examplesDir: filepath.Join(parseT.TempDir(), "missing")}
	parseErr := parseLauncher.runExamples(nil)
	if parseErr == nil {
		parseT.Fatal("expected missing examples directory to fail")
	}
	if !strings.Contains(parseErr.Error(), "examples directory not found") {
		parseT.Fatalf("expected missing examples directory error, got %v", parseErr)
	}
}

func TestLauncherRunHandlesUsageHelpAndUnknownCommand(parseT *testing.T) {
	parseStdout, parseRestoreStdout, parseErr := captureExamplesStdout()
	if parseErr != nil {
		parseT.Fatalf("capture stdout: %v", parseErr)
	}
	defer parseRestoreStdout()

	parseAppLauncher := launcher{examplesDir: parseT.TempDir(), staticDir: parseT.TempDir()}
	for _, parseArgs := range [][]string{{}, {"help"}, {"-h"}, {"--help"}} {
		if parseErr2 := parseAppLauncher.run(parseArgs); parseErr2 != nil {
			parseT.Fatalf("expected usage/help args %v to succeed, got %v", parseArgs, parseErr2)
		}
	}
	parseErr = parseAppLauncher.run([]string{"unknown-command"})
	if parseErr == nil || !strings.Contains(parseErr.Error(), `unknown command "unknown-command"`) {
		parseT.Fatalf("expected unknown command error, got %v", parseErr)
	}
	parseOutput, parseErr := parseStdout()
	if parseErr != nil {
		parseT.Fatalf("read captured stdout: %v", parseErr)
	}
	if parseCount := strings.Count(parseOutput, "GWC launcher"); parseCount < 4 {
		parseT.Fatalf("expected usage output for each help path, got count=%d output=%q", parseCount, parseOutput)
	}
}

func TestLauncherRunDispatchesEachSubcommand(parseT *testing.T) {
	parseOriginalRunTestCommand := runTestCommand
	parseOriginalRunExamplesCommand := runExamplesCommand
	parseOriginalRunBuildCommand := runBuildCommand
	parseOriginalRunBenchmarkCommand := runBenchmarkCommand
	parseOriginalRunReleaseCommand := runReleaseCommand
	parseOriginalRunDevCommand := runDevCommand
	parseOriginalRunServeCommand := runServeCommand
	parseOriginalRunFilesCommand := runFilesCommand
	parseOriginalRunLintCommand := runLintCommand
	parseOriginalRunInitCommand := runInitCommand
	parseOriginalRunInspectCommand := runInspectCommand
	parseOriginalRunUpgradeCommand := runUpgradeCommand
	parseOriginalRunMigrateCommand := runMigrateCommand
	parseOriginalRunPrerenderCommand := runPrerenderCommand
	parseOriginalRunExportCommand := runExportCommand
	parseOriginalRunDeployCommand := runDeployCommand
	parseOriginalRunTailwindCommand := runTailwindCommand
	parseOriginalRunDoctorCommand := runDoctorCommand
	parseOriginalRunEnvCommand := runEnvCommand
	parseOriginalRunVerifyCommand := runVerifyCommand
	parseOriginalRunStartCommand := runStartCommand
	parseOriginalRunBootstrapCommand := runBootstrapCommand
	parseT.Cleanup(func() {
		runTestCommand = parseOriginalRunTestCommand
		runExamplesCommand = parseOriginalRunExamplesCommand
		runBuildCommand = parseOriginalRunBuildCommand
		runBenchmarkCommand = parseOriginalRunBenchmarkCommand
		runReleaseCommand = parseOriginalRunReleaseCommand
		runDevCommand = parseOriginalRunDevCommand
		runServeCommand = parseOriginalRunServeCommand
		runFilesCommand = parseOriginalRunFilesCommand
		runLintCommand = parseOriginalRunLintCommand
		runInitCommand = parseOriginalRunInitCommand
		runInspectCommand = parseOriginalRunInspectCommand
		runUpgradeCommand = parseOriginalRunUpgradeCommand
		runMigrateCommand = parseOriginalRunMigrateCommand
		runPrerenderCommand = parseOriginalRunPrerenderCommand
		runExportCommand = parseOriginalRunExportCommand
		runDeployCommand = parseOriginalRunDeployCommand
		runTailwindCommand = parseOriginalRunTailwindCommand
		runDoctorCommand = parseOriginalRunDoctorCommand
		runEnvCommand = parseOriginalRunEnvCommand
		runVerifyCommand = parseOriginalRunVerifyCommand
		runStartCommand = parseOriginalRunStartCommand
		runBootstrapCommand = parseOriginalRunBootstrapCommand
	})

	parseTests := []struct {
		name        string
		args        []string
		installStub func(parseT2 *testing.T, parseCalled2 *bool)
	}{
		{name: "test", args: []string{"test", "-json"}, installStub: func(parseT3 *testing.T, parseCalled3 *bool) {
			runTestCommand = func(parseL launcher, parseArgs2 []string) error {
				*parseCalled3 = true
				if fmt.Sprint(parseArgs2) != fmt.Sprint([]string{"-json"}) {
					parseT3.Fatalf("unexpected args: %#v", parseArgs2)
				}
				return nil
			}
		}},
		{name: "examples", args: []string{"examples", "-port", "9000"}, installStub: func(parseT4 *testing.T, parseCalled4 *bool) {
			runExamplesCommand = func(parseL2 launcher, parseArgs3 []string) error {
				*parseCalled4 = true
				if fmt.Sprint(parseArgs3) != fmt.Sprint([]string{"-port", "9000"}) {
					parseT4.Fatalf("unexpected args: %#v", parseArgs3)
				}
				return nil
			}
		}},
		{name: "build", args: []string{"build", "-json"}, installStub: func(parseT5 *testing.T, parseCalled5 *bool) {
			runBuildCommand = func(parseL3 launcher, parseArgs4 []string) error { *parseCalled5 = true; return nil }
		}},
		{name: "bench", args: []string{"bench", "-json"}, installStub: func(parseT6 *testing.T, parseCalled6 *bool) {
			runBenchmarkCommand = func(parseL4 launcher, parseArgs5 []string) error { *parseCalled6 = true; return nil }
		}},
		{name: "release", args: []string{"release", "-json"}, installStub: func(parseT7 *testing.T, parseCalled7 *bool) {
			runReleaseCommand = func(parseL5 launcher, parseArgs6 []string) error { *parseCalled7 = true; return nil }
		}},
		{name: "dev", args: []string{"dev", "-dry-run"}, installStub: func(parseT8 *testing.T, parseCalled8 *bool) {
			runDevCommand = func(parseL6 launcher, parseArgs7 []string) error { *parseCalled8 = true; return nil }
		}},
		{name: "serve", args: []string{"serve", "-root", "."}, installStub: func(parseT9 *testing.T, parseCalled9 *bool) {
			runServeCommand = func(parseL7 launcher, parseArgs8 []string) error { *parseCalled9 = true; return nil }
		}},
		{name: "files", args: []string{"files", "-ext", "js"}, installStub: func(parseT10 *testing.T, parseCalled10 *bool) {
			runFilesCommand = func(parseL8 launcher, parseArgs9 []string) error { *parseCalled10 = true; return nil }
		}},
		{name: "lint", args: []string{"lint", "-json"}, installStub: func(parseT10 *testing.T, parseCalled10 *bool) {
			runLintCommand = func(parseL8 launcher, parseArgs9 []string) error { *parseCalled10 = true; return nil }
		}},
		{name: "review", args: []string{"review", "-json"}, installStub: func(parseT10 *testing.T, parseCalled10 *bool) {
			runLintCommand = func(parseL8 launcher, parseArgs9 []string) error { *parseCalled10 = true; return nil }
		}},
		{name: "init", args: []string{"init", "-json"}, installStub: func(parseT11 *testing.T, parseCalled11 *bool) {
			runInitCommand = func(parseL9 launcher, parseArgs10 []string) error { *parseCalled11 = true; return nil }
		}},
		{name: "inspect", args: []string{"inspect", "-json"}, installStub: func(parseT12 *testing.T, parseCalled12 *bool) {
			runInspectCommand = func(parseL10 launcher, parseArgs11 []string) error { *parseCalled12 = true; return nil }
		}},
		{name: "upgrade", args: []string{"upgrade", "-json"}, installStub: func(parseT13 *testing.T, parseCalled13 *bool) {
			runUpgradeCommand = func(parseL11 launcher, parseArgs12 []string) error { *parseCalled13 = true; return nil }
		}},
		{name: "migrate", args: []string{"migrate", "-json"}, installStub: func(parseT14 *testing.T, parseCalled14 *bool) {
			runMigrateCommand = func(parseL12 launcher, parseArgs13 []string) error { *parseCalled14 = true; return nil }
		}},
		{name: "prerender", args: []string{"prerender", "-json"}, installStub: func(parseT15 *testing.T, parseCalled15 *bool) {
			runPrerenderCommand = func(parseL13 launcher, parseArgs14 []string) error { *parseCalled15 = true; return nil }
		}},
		{name: "export", args: []string{"export", "-json"}, installStub: func(parseT16 *testing.T, parseCalled16 *bool) {
			runExportCommand = func(parseL14 launcher, parseArgs15 []string) error { *parseCalled16 = true; return nil }
		}},
		{name: "deploy", args: []string{"deploy", "-json"}, installStub: func(parseT17 *testing.T, parseCalled17 *bool) {
			runDeployCommand = func(parseL15 launcher, parseArgs16 []string) error { *parseCalled17 = true; return nil }
		}},
		{name: "tailwind", args: []string{"tailwind", "-json"}, installStub: func(parseT18 *testing.T, parseCalled18 *bool) {
			runTailwindCommand = func(parseL16 launcher, parseArgs17 []string) error { *parseCalled18 = true; return nil }
		}},
		{name: "doctor", args: []string{"doctor", "-json"}, installStub: func(parseT19 *testing.T, parseCalled19 *bool) {
			runDoctorCommand = func(parseL17 launcher, parseArgs18 []string) error { *parseCalled19 = true; return nil }
		}},
		{name: "env", args: []string{"env", "-json"}, installStub: func(parseT20 *testing.T, parseCalled20 *bool) {
			runEnvCommand = func(parseL18 launcher, parseArgs19 []string) error { *parseCalled20 = true; return nil }
		}},
		{name: "verify", args: []string{"verify", "-json"}, installStub: func(parseT21 *testing.T, parseCalled21 *bool) {
			runVerifyCommand = func(parseL19 launcher, parseArgs20 []string) error { *parseCalled21 = true; return nil }
		}},
		{name: "start", args: []string{"start", "--help"}, installStub: func(parseT22 *testing.T, parseCalled22 *bool) {
			runStartCommand = func(parseL20 launcher, parseArgs21 []string) error { *parseCalled22 = true; return nil }
		}},
		{name: "bootstrap", args: []string{"bootstrap", "-examples"}, installStub: func(parseT23 *testing.T, parseCalled23 *bool) {
			runBootstrapCommand = func(parseL21 launcher, parseArgs22 []string) error { *parseCalled23 = true; return nil }
		}},
	}

	parseAppLauncher := launcher{}
	for _, parseTest := range parseTests {
		parseT.Run(parseTest.name, func(parseT24 *testing.T) {
			runTestCommand = parseOriginalRunTestCommand
			runExamplesCommand = parseOriginalRunExamplesCommand
			runBuildCommand = parseOriginalRunBuildCommand
			runBenchmarkCommand = parseOriginalRunBenchmarkCommand
			runReleaseCommand = parseOriginalRunReleaseCommand
			runDevCommand = parseOriginalRunDevCommand
			runServeCommand = parseOriginalRunServeCommand
			runFilesCommand = parseOriginalRunFilesCommand
			runLintCommand = parseOriginalRunLintCommand
			runInitCommand = parseOriginalRunInitCommand
			runInspectCommand = parseOriginalRunInspectCommand
			runUpgradeCommand = parseOriginalRunUpgradeCommand
			runMigrateCommand = parseOriginalRunMigrateCommand
			runPrerenderCommand = parseOriginalRunPrerenderCommand
			runExportCommand = parseOriginalRunExportCommand
			runDeployCommand = parseOriginalRunDeployCommand
			runTailwindCommand = parseOriginalRunTailwindCommand
			runDoctorCommand = parseOriginalRunDoctorCommand
			runEnvCommand = parseOriginalRunEnvCommand
			runVerifyCommand = parseOriginalRunVerifyCommand
			runStartCommand = parseOriginalRunStartCommand
			runBootstrapCommand = parseOriginalRunBootstrapCommand

			isParseCalled := false
			parseTest.installStub(parseT24, &isParseCalled)
			if parseErr := parseAppLauncher.run(parseTest.args); parseErr != nil {
				parseT24.Fatalf("run dispatch failed: %v", parseErr)
			}
			if !isParseCalled {
				parseT24.Fatal("expected dispatch stub to be called")
			}
		})
	}
}

func TestMainHandlesResolveRepoRootRunAndSuccessPaths(parseT *testing.T) {
	parseOriginalResolveRepoRoot := mainResolveRepoRoot
	parseOriginalRunLauncher := mainRunLauncher
	parseOriginalExit := mainExit
	parseOriginalArgs := mainArgs
	parseOriginalPrintError := mainPrintError
	parseT.Cleanup(func() {
		mainResolveRepoRoot = parseOriginalResolveRepoRoot
		mainRunLauncher = parseOriginalRunLauncher
		mainExit = parseOriginalExit
		mainArgs = parseOriginalArgs
		mainPrintError = parseOriginalPrintError
	})

	type exitSignal struct{ code int }

	parseT.Run("resolve repo root failure", func(parseT2 *testing.T) {
		parsePrinted := ""
		mainResolveRepoRoot = func() (string, error) { return "", errors.New("no repo") }
		mainRunLauncher = func(parseL launcher, parseArgs []string) error {
			parseT2.Fatal("expected run launcher not to be called")
			return nil
		}
		mainArgs = func() []string { return []string{"gwc"} }
		mainPrintError = func(parseErr error) { parsePrinted = parseErr.Error() }
		mainExit = func(parseCode2 int) { panic(exitSignal{code: parseCode2}) }

		defer func() {
			parseRecovered := recover()
			parseSignal, parseOk := parseRecovered.(exitSignal)
			if !parseOk || parseSignal.code != 1 {
				parseT2.Fatalf("expected exit code 1, got %#v", parseRecovered)
			}
			if parsePrinted != "no repo" {
				parseT2.Fatalf("expected printed resolveRepoRoot error, got %q", parsePrinted)
			}
		}()
		main()
	})

	parseT.Run("run failure", func(parseT3 *testing.T) {
		parsePrinted2 := ""
		mainResolveRepoRoot = func() (string, error) { return `C:\repo`, nil }
		mainArgs = func() []string { return []string{"gwc", "examples", "-port", "9999"} }
		mainPrintError = func(parseErr2 error) { parsePrinted2 = parseErr2.Error() }
		mainRunLauncher = func(parseL2 launcher, parseArgs2 []string) error {
			if parseL2.repoRoot != `C:\repo` || parseL2.examplesDir != filepath.Join(`C:\repo`, "examples") || parseL2.staticDir != filepath.Join(`C:\repo`, "examples", "static") {
				parseT3.Fatalf("expected launcher to be initialized from repo root, got %#v", parseL2)
			}
			if strings.Join(parseArgs2, " ") != "examples -port 9999" {
				parseT3.Fatalf("expected main args to be forwarded, got %#v", parseArgs2)
			}
			return errors.New("run failed")
		}
		mainExit = func(parseCode3 int) { panic(exitSignal{code: parseCode3}) }

		defer func() {
			parseRecovered2 := recover()
			parseSignal2, parseOk2 := parseRecovered2.(exitSignal)
			if !parseOk2 || parseSignal2.code != 1 {
				parseT3.Fatalf("expected exit code 1, got %#v", parseRecovered2)
			}
			if parsePrinted2 != "run failed" {
				parseT3.Fatalf("expected printed run error, got %q", parsePrinted2)
			}
		}()
		main()
	})

	parseT.Run("success", func(parseT4 *testing.T) {
		isParsePrinted3 := false
		isParseExited := false
		isParseCalled := false
		mainResolveRepoRoot = func() (string, error) { return `C:\repo`, nil }
		mainArgs = func() []string { return []string{"gwc", "doctor"} }
		mainPrintError = func(parseErr3 error) { isParsePrinted3 = true }
		mainExit = func(parseCode4 int) { isParseExited = true }
		mainRunLauncher = func(parseL3 launcher, parseArgs3 []string) error {
			isParseCalled = true
			if len(parseArgs3) != 1 || parseArgs3[0] != "doctor" {
				parseT4.Fatalf("expected success args to be forwarded, got %#v", parseArgs3)
			}
			return nil
		}

		main()
		if !isParseCalled {
			parseT4.Fatal("expected main to invoke launcher run on success")
		}
		if isParsePrinted3 || isParseExited {
			parseT4.Fatalf("expected success path to avoid error printing and exit, printed=%t exited=%t", isParsePrinted3, isParseExited)
		}
	})
}

func TestPrintLauncherErrorSupportsMachineReadableDiagnostics(parseT *testing.T) {
	parseT.Run("json requested emits structured configuration diagnostic", func(parseT2 *testing.T) {
		var parseOutput bytes.Buffer
		printLauncherError(&parseOutput, []string{"verify", "-json"}, fmt.Errorf("resolve app path: missing main.go"))

		var parseDiagnostic launcherFailureDiagnostic
		if parseErr := json.Unmarshal(parseOutput.Bytes(), &parseDiagnostic); parseErr != nil {
			parseT2.Fatalf("unmarshal launcher diagnostic: %v\n%s", parseErr, parseOutput.String())
		}
		if parseDiagnostic.OK {
			parseT2.Fatalf("expected failing diagnostic, got %#v", parseDiagnostic)
		}
		if parseDiagnostic.Command != "verify" || parseDiagnostic.Phase != "configuration" || parseDiagnostic.Category != "configuration" || parseDiagnostic.Code != "invalid_configuration" {
			parseT2.Fatalf("expected structured configuration diagnostic, got %#v", parseDiagnostic)
		}
		if !strings.Contains(parseDiagnostic.Message, "resolve app path") {
			parseT2.Fatalf("expected original failure message to be preserved, got %#v", parseDiagnostic)
		}
	})

	parseT.Run("json requested distinguishes code failure", func(parseT3 *testing.T) {
		var parseOutput2 bytes.Buffer
		printLauncherError(&parseOutput2, []string{"build", "-json"}, fmt.Errorf("go build failed: exit status 1"))

		var parseDiagnostic2 launcherFailureDiagnostic
		if parseErr2 := json.Unmarshal(parseOutput2.Bytes(), &parseDiagnostic2); parseErr2 != nil {
			parseT3.Fatalf("unmarshal launcher diagnostic: %v\n%s", parseErr2, parseOutput2.String())
		}
		if parseDiagnostic2.Category != "code" || parseDiagnostic2.Code != "code_failure" {
			parseT3.Fatalf("expected code failure classification, got %#v", parseDiagnostic2)
		}
	})

	parseT.Run("json requested distinguishes invalid runner override", func(parseT4 *testing.T) {
		var parseOutput3 bytes.Buffer
		printLauncherError(&parseOutput3, []string{"test", "-json"}, fmt.Errorf("configured browserWorkspace does not contain a Playwright-Go suite: C:\\broken\\browser"))

		var parseDiagnostic3 launcherFailureDiagnostic
		if parseErr3 := json.Unmarshal(parseOutput3.Bytes(), &parseDiagnostic3); parseErr3 != nil {
			parseT4.Fatalf("unmarshal launcher diagnostic: %v\n%s", parseErr3, parseOutput3.String())
		}
		if parseDiagnostic3.Code != "invalid_runner_override" || parseDiagnostic3.Override != "browserWorkspace" {
			parseT4.Fatalf("expected invalid runner override diagnostic, got %#v", parseDiagnostic3)
		}
	})

	parseT.Run("plain text output remains unchanged without json", func(parseT5 *testing.T) {
		var parseOutput4 bytes.Buffer
		printLauncherError(&parseOutput4, []string{"release"}, errors.New("boom"))
		if parseGot := parseOutput4.String(); parseGot != "gwc: boom\n" {
			parseT5.Fatalf("expected plain text launcher error, got %q", parseGot)
		}
	})
}

func TestResolveRepoRootErrorBranches(parseT *testing.T) {
	parseOriginalCaller := resolveRepoRootCaller
	parseT.Cleanup(func() { resolveRepoRootCaller = parseOriginalCaller })

	resolveRepoRootCaller = func(parseSkip int) (uintptr, string, int, bool) {
		return 0, "", 0, false
	}
	if parseGot, parseErr := resolveRepoRoot(); parseErr == nil || !strings.Contains(parseErr.Error(), "unable to resolve launcher source path") {
		parseT.Fatalf("expected caller failure, got path=%q err=%v", parseGot, parseErr)
	}

	parseFakeFile := filepath.Join(parseT.TempDir(), "tools", "gwc", "main.go")
	if parseErr2 := os.MkdirAll(filepath.Dir(parseFakeFile), 0755); parseErr2 != nil {
		parseT.Fatalf("mkdir fake source dir: %v", parseErr2)
	}
	if parseErr3 := os.WriteFile(parseFakeFile, []byte("package main\n"), 0644); parseErr3 != nil {
		parseT.Fatalf("write fake source file: %v", parseErr3)
	}
	resolveRepoRootCaller = func(parseSkip2 int) (uintptr, string, int, bool) {
		return 0, parseFakeFile, 1, true
	}
	if parseGot2, parseErr4 := resolveRepoRoot(); parseErr4 == nil || !strings.Contains(parseErr4.Error(), "unable to resolve repo root") {
		parseT.Fatalf("expected missing go.mod error, got path=%q err=%v", parseGot2, parseErr4)
	}
}

func TestExamplesHealthzRouteIncludesLauncherMetadata(parseT *testing.T) {
	parseLauncher := launcher{repoRoot: `C:\repo`, examplesDir: parseT.TempDir(), staticDir: parseT.TempDir()}
	parseHandler := parseLauncher.newExamplesHandler("127.0.0.1", "8090")
	parseRequest := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	parseRecorder := httptest.NewRecorder()

	parseHandler.ServeHTTP(parseRecorder, parseRequest)

	if parseRecorder.Code != http.StatusOK {
		parseT.Fatalf("expected healthz to succeed, got %d with body %s", parseRecorder.Code, parseRecorder.Body.String())
	}
	var parsePayload map[string]any
	if parseErr := json.Unmarshal(parseRecorder.Body.Bytes(), &parsePayload); parseErr != nil {
		parseT.Fatalf("decode healthz payload: %v", parseErr)
	}
	for parseKey, parseExpected := range map[string]string{
		"service": "gowebcomponents-gwc-examples",
		"root":    `C:\repo`,
		"host":    "127.0.0.1",
		"port":    "8090",
	} {
		if parsePayload[parseKey] != parseExpected {
			parseT.Fatalf("expected %s=%q, got %#v", parseKey, parseExpected, parsePayload[parseKey])
		}
	}
	if parsePayload["ok"] != true {
		parseT.Fatalf("expected ok=true, got %#v", parsePayload)
	}
}

func TestExamplesListRouteRendersFilteredCatalogHTML(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseExamplesDir := filepath.Join(parseRoot, "examples")
	parseStaticDir := filepath.Join(parseRoot, "static")
	if parseErr := os.MkdirAll(filepath.Join(parseExamplesDir, "01-alpha"), 0755); parseErr != nil {
		parseT.Fatalf("mkdir alpha dir: %v", parseErr)
	}
	if parseErr2 := os.MkdirAll(filepath.Join(parseExamplesDir, "02-beta"), 0755); parseErr2 != nil {
		parseT.Fatalf("mkdir beta dir: %v", parseErr2)
	}
	if parseErr3 := os.WriteFile(filepath.Join(parseExamplesDir, "01-alpha", "index.html"), []byte("<title>Alpha</title>"), 0644); parseErr3 != nil {
		parseT.Fatalf("write alpha html: %v", parseErr3)
	}
	if parseErr4 := os.WriteFile(filepath.Join(parseExamplesDir, "02-beta", "index.html"), []byte("<title>Beta</title>"), 0644); parseErr4 != nil {
		parseT.Fatalf("write beta html: %v", parseErr4)
	}
	if parseErr5 := os.MkdirAll(parseStaticDir, 0755); parseErr5 != nil {
		parseT.Fatalf("mkdir static dir: %v", parseErr5)
	}

	parseLauncher := launcher{examplesDir: parseExamplesDir, staticDir: parseStaticDir}
	parseHandler := parseLauncher.newExamplesHandler("127.0.0.1", "8090")
	parseRequest := httptest.NewRequest(http.MethodGet, "/examples/list?q=beta", nil)
	parseRecorder := httptest.NewRecorder()

	parseHandler.ServeHTTP(parseRecorder, parseRequest)

	if parseRecorder.Code != http.StatusOK {
		parseT.Fatalf("expected examples list to succeed, got %d with body %s", parseRecorder.Code, parseRecorder.Body.String())
	}
	parseBody := parseRecorder.Body.String()
	if !strings.Contains(parseBody, "Filtered examples for") || !strings.Contains(parseBody, "/examples/02-beta/") {
		parseT.Fatalf("expected filtered examples list html, got %s", parseBody)
	}
	if strings.Contains(parseBody, "/examples/01-alpha/") {
		parseT.Fatalf("expected filtered examples list to omit alpha entry, got %s", parseBody)
	}
}

func TestPrintHelpersEmitExpectedLauncherOutput(parseT *testing.T) {
	parseStdout, parseRestoreStdout, parseErr := captureExamplesStdout()
	if parseErr != nil {
		parseT.Fatalf("capture stdout: %v", parseErr)
	}
	defer parseRestoreStdout()

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
		OutDir:       `C:\repo\bin`,
		ManifestPath: `C:\repo\bin\manifest.json`,
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
			{Name: "browser", Skipped: true, Summary: "No browser test workspace was found for the requested root."},
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

	parseOutput, parseErr := parseStdout()
	if parseErr != nil {
		parseT.Fatalf("read captured stdout: %v", parseErr)
	}
	for _, parseExpected := range []string{
		"GWC launcher",
		"bench      Discover native/js-wasm benchmark packages, capture raw benchmark output, compare files with benchstat, and write docs/benchmarks JSON output",
		"files      List project files with repeatable extension and directory filters",
		"GWC build",
		"ldflags:      -s -w",
		"artifact[gzip]: app.wasm.gz (10 bytes)",
		"artifact[wasm]: app.wasm (20 bytes)",
		"tests:        go test ./...",
		"tests:        skipped",
		"[ok] unit: Native Go tests passed.",
		"[skipped] browser: No browser test workspace was found for the requested root.",
		"GWC doctor: FAIL",
		"hint: pick another port",
	} {
		if !strings.Contains(parseOutput, parseExpected) {
			parseT.Fatalf("expected output to contain %q, got:\n%s", parseExpected, parseOutput)
		}
	}
	if strings.Index(parseOutput, "artifact[gzip]:") > strings.Index(parseOutput, "artifact[wasm]:") {
		parseT.Fatalf("expected release artifacts to be sorted alphabetically, got:\n%s", parseOutput)
	}
}

func TestRenderExamplesShellHTMLFallbackIncludesManifestNoscriptAndBodyData(parseT *testing.T) {
	parseDocument := examplesShellDocument{
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
	parseHtml := renderExamplesShellHTMLFallback(parseDocument, `<script id="bootstrap"></script>`)
	for _, parseExpected := range []string{
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
		if !strings.Contains(parseHtml, parseExpected) {
			parseT.Fatalf("expected fallback html to contain %q, got %s", parseExpected, parseHtml)
		}
	}
}

func TestRenderExamplesShellHTMLFallsBackWhenRenderToStringFails(parseT *testing.T) {
	parseOriginalRenderToString := renderExamplesToString
	parseOriginalRenderBootstrapData := renderExamplesUIBootstrapScript
	parseT.Cleanup(func() {
		renderExamplesToString = parseOriginalRenderToString
		renderExamplesUIBootstrapScript = parseOriginalRenderBootstrapData
	})

	renderExamplesToString = func(parseNode ui.Node) (string, error) {
		return "", errors.New("render failed")
	}
	renderExamplesUIBootstrapScript = func(parseBootstrap ui.SSRBootstrap, parseNonce string) (string, error) {
		return `<script id="bootstrap"></script>`, nil
	}

	parseHtml := renderExamplesShellHTML(examplesShellDocument{
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
	for _, parseExpected := range []string{`<script id="bootstrap"></script>`, `manifest.webmanifest`, `Enable JavaScript.`, `data-mode="fallback"`} {
		if !strings.Contains(parseHtml, parseExpected) {
			parseT.Fatalf("expected fallback render html to contain %q, got %s", parseExpected, parseHtml)
		}
	}
}

func TestStatusWriterDefaultsStatusOnWrite(parseT *testing.T) {
	parseRecorder := httptest.NewRecorder()
	parseWriter := &statusWriter{ResponseWriter: parseRecorder}
	if _, parseErr := parseWriter.Write([]byte("ok")); parseErr != nil {
		parseT.Fatalf("write body: %v", parseErr)
	}
	if parseWriter.status != http.StatusOK {
		parseT.Fatalf("expected implicit status 200, got %d", parseWriter.status)
	}
	parseWriter.WriteHeader(http.StatusAccepted)
	if parseWriter.status != http.StatusAccepted {
		parseT.Fatalf("expected explicit status to be preserved, got %d", parseWriter.status)
	}
}

func TestPrintDevPlanCoversResolvedAndDefaultFields(parseT *testing.T) {
	parseStdout, parseRestoreStdout, parseErr := captureExamplesStdout()
	if parseErr != nil {
		parseT.Fatalf("capture stdout: %v", parseErr)
	}
	defer parseRestoreStdout()

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

	parseOutput, parseErr := parseStdout()
	if parseErr != nil {
		parseT.Fatalf("read captured stdout: %v", parseErr)
	}
	for _, parseExpected := range []string{
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
		if !strings.Contains(parseOutput, parseExpected) {
			parseT.Fatalf("expected dev plan output to contain %q, got:\n%s", parseExpected, parseOutput)
		}
	}
}

func TestBuildDoctorToolCheckFailureModes(parseT *testing.T) {
	parseOriginalLookPath := doctorLookPath
	parseOriginalCommandOutput := doctorCommandOutput
	parseT.Cleanup(func() {
		doctorLookPath = parseOriginalLookPath
		doctorCommandOutput = parseOriginalCommandOutput
	})

	doctorLookPath = func(string) (string, error) {
		return "", errors.New("missing")
	}
	parseMissing := buildDoctorToolCheck("go", "Go toolchain", "version", "install go")
	if parseMissing.Status != "fail" {
		parseT.Fatalf("expected missing tool check to fail, got %#v", parseMissing)
	}
	if !strings.Contains(parseMissing.Summary, "go was not found on PATH") {
		parseT.Fatalf("expected missing tool summary, got %#v", parseMissing)
	}
	if parseMissing.Hint != "install go" {
		parseT.Fatalf("expected missing tool hint to be preserved, got %#v", parseMissing)
	}

	doctorLookPath = func(string) (string, error) {
		return `C:\Go\bin\go.exe`, nil
	}
	doctorCommandOutput = func(parseName string, parseArgs ...string) (string, error) {
		return "version command blocked", errors.New("exit status 1")
	}
	parseCommandFailure := buildDoctorToolCheck("go", "Go toolchain", "version", "install go")
	if parseCommandFailure.Status != "fail" {
		parseT.Fatalf("expected version command failure to report fail, got %#v", parseCommandFailure)
	}
	for _, parseExpected := range []string{"C:\\Go\\bin\\go.exe", "version command blocked"} {
		if !strings.Contains(parseCommandFailure.Summary, parseExpected) {
			parseT.Fatalf("expected command failure summary to contain %q, got %#v", parseExpected, parseCommandFailure)
		}
	}

	doctorCommandOutput = func(parseName2 string, parseArgs2 ...string) (string, error) {
		return "", errors.New("exit status 2")
	}
	parseEmptyOutputFailure := buildDoctorToolCheck("go", "Go toolchain", "version", "install go")
	if parseEmptyOutputFailure.Status != "fail" {
		parseT.Fatalf("expected empty output failure to report fail, got %#v", parseEmptyOutputFailure)
	}
	if !strings.Contains(parseEmptyOutputFailure.Summary, "exit status 2") {
		parseT.Fatalf("expected fallback error text in summary, got %#v", parseEmptyOutputFailure)
	}
}

func TestRunExamplesHelpReturnsNil(parseT *testing.T) {
	parseLauncher := launcher{examplesDir: parseT.TempDir(), staticDir: parseT.TempDir()}
	if parseErr := parseLauncher.run([]string{"examples", "-help"}); parseErr != nil {
		parseT.Fatalf("expected examples help to succeed, got %v", parseErr)
	}
}

func TestRunDoctorHelpReturnsNil(parseT *testing.T) {
	parseLauncher := launcher{repoRoot: parseT.TempDir()}
	if parseErr := parseLauncher.run([]string{"doctor", "-help"}); parseErr != nil {
		parseT.Fatalf("expected doctor help to succeed, got %v", parseErr)
	}
}

func TestRunDevHelpReturnsNil(parseT *testing.T) {
	parseLauncher := launcher{repoRoot: parseT.TempDir()}
	if parseErr := parseLauncher.run([]string{"dev", "-help"}); parseErr != nil {
		parseT.Fatalf("expected dev help to succeed, got %v", parseErr)
	}
}

func TestResolveGeneratedExamplePageReturnsFalseForNonWasmExample(parseT *testing.T) {
	parseRepoRoot, parseErr := resolveRepoRoot()
	if parseErr != nil {
		parseT.Fatalf("resolve repo root: %v", parseErr)
	}
	parseLauncher := launcher{
		repoRoot:    parseRepoRoot,
		examplesDir: filepath.Join(parseRepoRoot, "examples"),
		staticDir:   filepath.Join(parseRepoRoot, "examples", "static"),
	}

	parsePage, parseOk, parseErr := parseLauncher.resolveGeneratedExamplePage("/examples/public/web-components/")
	if parseErr != nil {
		parseT.Fatalf("resolve generated example page: %v", parseErr)
	}
	if parseOk {
		parseT.Fatalf("expected non-wasm example not to generate a wasm host page, got %#v", parsePage)
	}
}

func TestResolveExampleCatalogEntryHandlesMissingAndNonDirectoryTargets(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseExamplesDir := filepath.Join(parseRoot, "examples")
	parseStaticDir := filepath.Join(parseRoot, "static")
	if parseErr := os.MkdirAll(parseExamplesDir, 0755); parseErr != nil {
		parseT.Fatalf("mkdir examples dir: %v", parseErr)
	}
	if parseErr2 := os.MkdirAll(parseStaticDir, 0755); parseErr2 != nil {
		parseT.Fatalf("mkdir static dir: %v", parseErr2)
	}
	if parseErr3 := os.WriteFile(filepath.Join(parseExamplesDir, "99-file"), []byte("not a directory"), 0644); parseErr3 != nil {
		parseT.Fatalf("write marker file: %v", parseErr3)
	}

	parseLauncher := launcher{examplesDir: parseExamplesDir, staticDir: parseStaticDir}
	parseEntry, parseOk, parseErr4 := parseLauncher.resolveExampleCatalogEntry("missing")
	if parseErr4 != nil {
		parseT.Fatalf("resolve missing catalog entry: %v", parseErr4)
	}
	if parseOk {
		parseT.Fatalf("expected missing example to return ok=false, got %#v", parseEntry)
	}

	parseEntry, parseOk, parseErr4 = parseLauncher.resolveExampleCatalogEntry("99-file")
	if parseErr4 != nil {
		parseT.Fatalf("resolve non-directory catalog entry: %v", parseErr4)
	}
	if parseOk {
		parseT.Fatalf("expected non-directory example target to return ok=false, got %#v", parseEntry)
	}
}

func TestExamplesHandlerReturnsJSONErrorsForBrokenCatalogSources(parseT *testing.T) {
	parseLauncher := launcher{repoRoot: parseT.TempDir(), examplesDir: filepath.Join(parseT.TempDir(), "missing"), staticDir: parseT.TempDir()}
	parseHandler := parseLauncher.newExamplesHandler("127.0.0.1", "8090")

	for parsePath, parseExpectedError := range map[string]string{
		"/examples/list":         "examples_listing_failed",
		"/examples/catalog.json": "examples_catalog_failed",
	} {
		parseRecorder := httptest.NewRecorder()
		parseHandler.ServeHTTP(parseRecorder, httptest.NewRequest(http.MethodGet, parsePath, nil))
		if parseRecorder.Code != http.StatusInternalServerError {
			parseT.Fatalf("expected %s to fail with 500, got %d", parsePath, parseRecorder.Code)
		}
		if !strings.Contains(parseRecorder.Body.String(), parseExpectedError) {
			parseT.Fatalf("expected %s response to contain %q, got %s", parsePath, parseExpectedError, parseRecorder.Body.String())
		}
	}
}

func TestExamplesHandlerPassThroughBranches(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseExamplesDir := filepath.Join(parseRoot, "examples")
	parseStaticDir := filepath.Join(parseRoot, "static")
	if parseErr := os.MkdirAll(filepath.Join(parseExamplesDir, "assets"), 0755); parseErr != nil {
		parseT.Fatalf("mkdir assets dir: %v", parseErr)
	}
	if parseErr2 := os.MkdirAll(parseStaticDir, 0755); parseErr2 != nil {
		parseT.Fatalf("mkdir static dir: %v", parseErr2)
	}
	if parseErr3 := os.WriteFile(filepath.Join(parseExamplesDir, "assets", "readme.txt"), []byte("example asset"), 0644); parseErr3 != nil {
		parseT.Fatalf("write example asset: %v", parseErr3)
	}
	if parseErr4 := os.WriteFile(filepath.Join(parseStaticDir, "plain.txt"), []byte("static asset"), 0644); parseErr4 != nil {
		parseT.Fatalf("write static asset: %v", parseErr4)
	}

	parseLauncher := launcher{repoRoot: parseRoot, examplesDir: parseExamplesDir, staticDir: parseStaticDir}
	parseHandler := parseLauncher.newExamplesHandler("127.0.0.1", "8090")

	parseRecorder := httptest.NewRecorder()
	parseHandler.ServeHTTP(parseRecorder, httptest.NewRequest(http.MethodGet, "/examples/assets/readme.txt", nil))
	if parseRecorder.Code != http.StatusOK || !strings.Contains(parseRecorder.Body.String(), "example asset") {
		parseT.Fatalf("expected examples pass-through asset, got code=%d body=%s", parseRecorder.Code, parseRecorder.Body.String())
	}

	parseRecorder = httptest.NewRecorder()
	parseHandler.ServeHTTP(parseRecorder, httptest.NewRequest(http.MethodGet, "/static/plain.txt", nil))
	if parseRecorder.Code != http.StatusOK || !strings.Contains(parseRecorder.Body.String(), "static asset") {
		parseT.Fatalf("expected static pass-through asset, got code=%d body=%s", parseRecorder.Code, parseRecorder.Body.String())
	}

	parseRecorder = httptest.NewRecorder()
	parseHandler.ServeHTTP(parseRecorder, httptest.NewRequest(http.MethodGet, "/not-root.txt", nil))
	if parseRecorder.Code != http.StatusNotFound {
		parseT.Fatalf("expected root fallback pass-through 404 for unknown asset, got %d", parseRecorder.Code)
	}
}

func TestExamplesHandlerExactAndErrorRoutes(parseT *testing.T) {
	parseRepoRoot, parseErr := resolveRepoRoot()
	if parseErr != nil {
		parseT.Fatalf("resolve repo root: %v", parseErr)
	}
	parseRealLauncher := launcher{
		repoRoot:    parseRepoRoot,
		examplesDir: filepath.Join(parseRepoRoot, "examples"),
		staticDir:   filepath.Join(parseRepoRoot, "examples", "static"),
	}
	parseHandler := parseRealLauncher.newExamplesHandler("127.0.0.1", "8090")

	for _, parsePath := range []string{"/examples", "/examples/static/index.html"} {
		parseRecorder := httptest.NewRecorder()
		parseHandler.ServeHTTP(parseRecorder, httptest.NewRequest(http.MethodGet, parsePath, nil))
		if parseRecorder.Code != http.StatusOK {
			parseT.Fatalf("expected %s to serve app shell, got %d with body %s", parsePath, parseRecorder.Code, parseRecorder.Body.String())
		}
		if !strings.Contains(parseRecorder.Body.String(), `<div id="app"></div>`) {
			parseT.Fatalf("expected %s to render app shell, got %s", parsePath, parseRecorder.Body.String())
		}
	}

	parseBrokenRoot := parseT.TempDir()
	parseBrokenExamplesPath := filepath.Join(parseBrokenRoot, "examples-file")
	if parseErr2 := os.WriteFile(parseBrokenExamplesPath, []byte("not a directory"), 0644); parseErr2 != nil {
		parseT.Fatalf("write examples file: %v", parseErr2)
	}
	parseBrokenLauncher := launcher{repoRoot: parseBrokenRoot, examplesDir: parseBrokenExamplesPath, staticDir: filepath.Join(parseBrokenRoot, "static")}
	parseBrokenHandler := parseBrokenLauncher.newExamplesHandler("127.0.0.1", "8090")
	parseRecorder2 := httptest.NewRecorder()
	parseBrokenHandler.ServeHTTP(parseRecorder2, httptest.NewRequest(http.MethodGet, "/examples/public/counter/", nil))
	if parseRecorder2.Code != http.StatusNotFound {
		parseT.Fatalf("expected file-backed examples root to fall through with 404, got code=%d body=%s", parseRecorder2.Code, parseRecorder2.Body.String())
	}
}

func TestExamplesHandlerAppliesHeadersToServedAssets(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseExamplesDir := filepath.Join(parseRoot, "examples")
	parseStaticDir := filepath.Join(parseRoot, "static")
	if parseErr := os.MkdirAll(filepath.Join(parseExamplesDir, "01-counter"), 0755); parseErr != nil {
		parseT.Fatalf("mkdir example dir: %v", parseErr)
	}
	if parseErr2 := os.MkdirAll(filepath.Join(parseStaticDir, "bin"), 0755); parseErr2 != nil {
		parseT.Fatalf("mkdir static bin dir: %v", parseErr2)
	}
	if parseErr3 := os.WriteFile(filepath.Join(parseExamplesDir, "01-counter", "notes.txt"), []byte("example-notes"), 0644); parseErr3 != nil {
		parseT.Fatalf("write example asset: %v", parseErr3)
	}
	if parseErr4 := os.WriteFile(filepath.Join(parseStaticDir, "bin", "app.wasm"), []byte("wasm-bytes"), 0644); parseErr4 != nil {
		parseT.Fatalf("write wasm asset: %v", parseErr4)
	}

	parseHandler := (launcher{repoRoot: parseRoot, examplesDir: parseExamplesDir, staticDir: parseStaticDir}).newExamplesHandler("127.0.0.1", "8090")

	parseExampleRecorder := httptest.NewRecorder()
	parseHandler.ServeHTTP(parseExampleRecorder, httptest.NewRequest(http.MethodGet, "/examples/public/counter/notes.txt", nil))
	if parseExampleRecorder.Code != http.StatusOK {
		parseT.Fatalf("expected example asset to be served, got %d body=%s", parseExampleRecorder.Code, parseExampleRecorder.Body.String())
	}
	if parseGot := parseExampleRecorder.Header().Get("Cache-Control"); parseGot != "no-store, no-cache, must-revalidate" {
		parseT.Fatalf("expected cache header on example asset, got %q", parseGot)
	}

	parseWasmRecorder := httptest.NewRecorder()
	parseHandler.ServeHTTP(parseWasmRecorder, httptest.NewRequest(http.MethodGet, "/static/bin/app.wasm", nil))
	if parseWasmRecorder.Code != http.StatusOK {
		parseT.Fatalf("expected wasm asset to be served, got %d body=%s", parseWasmRecorder.Code, parseWasmRecorder.Body.String())
	}
	if parseGot2 := parseWasmRecorder.Header().Get("Content-Type"); parseGot2 != "application/wasm" {
		parseT.Fatalf("expected wasm content type, got %q", parseGot2)
	}
	if parseGot3 := parseWasmRecorder.Header().Get("Cache-Control"); parseGot3 != "no-store, no-cache, must-revalidate" {
		parseT.Fatalf("expected cache header on wasm asset, got %q", parseGot3)
	}
}

func TestExamplesHandlerFallsBackForEmptyAndMissingRoutes(parseT *testing.T) {
	parseRepoRoot, parseErr := resolveRepoRoot()
	if parseErr != nil {
		parseT.Fatalf("resolve repo root: %v", parseErr)
	}
	parseHandler := (launcher{
		repoRoot:    parseRepoRoot,
		examplesDir: filepath.Join(parseRepoRoot, "examples"),
		staticDir:   filepath.Join(parseRepoRoot, "examples", "static"),
	}).newExamplesHandler("127.0.0.1", "8090")

	parseEmptyRecorder := httptest.NewRecorder()
	parseHandler.ServeHTTP(parseEmptyRecorder, httptest.NewRequest(http.MethodGet, "/examples//", nil))
	if parseEmptyRecorder.Code != http.StatusMovedPermanently &&
		parseEmptyRecorder.Code != http.StatusTemporaryRedirect &&
		parseEmptyRecorder.Code != http.StatusPermanentRedirect {
		parseT.Fatalf("expected double-slash route to canonicalize, got %d body=%s", parseEmptyRecorder.Code, parseEmptyRecorder.Body.String())
	}
	if parseLocation := parseEmptyRecorder.Header().Get("Location"); parseLocation != "/examples/" {
		parseT.Fatalf("expected canonical redirect to /examples/, got %q", parseLocation)
	}

	parseMissingRecorder := httptest.NewRecorder()
	parseHandler.ServeHTTP(parseMissingRecorder, httptest.NewRequest(http.MethodGet, "/examples/does-not-exist/", nil))
	if parseMissingRecorder.Code != http.StatusNotFound {
		parseT.Fatalf("expected missing example route to pass through as 404, got %d body=%s", parseMissingRecorder.Code, parseMissingRecorder.Body.String())
	}
}

func TestExamplesHandlerReachableMuxFallbackBranches(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseExamplesDir := filepath.Join(parseRoot, "examples")
	parseStaticDir := filepath.Join(parseRoot, "static")
	if parseErr := os.MkdirAll(filepath.Join(parseExamplesDir, "nested"), 0755); parseErr != nil {
		parseT.Fatalf("mkdir nested dir: %v", parseErr)
	}
	if parseErr2 := os.MkdirAll(filepath.Join(parseStaticDir, "bin"), 0755); parseErr2 != nil {
		parseT.Fatalf("mkdir static bin dir: %v", parseErr2)
	}
	if parseErr3 := os.WriteFile(filepath.Join(parseExamplesDir, "nested", "info.txt"), []byte("nested asset"), 0644); parseErr3 != nil {
		parseT.Fatalf("write nested asset: %v", parseErr3)
	}

	parseHandler := (launcher{repoRoot: parseRoot, examplesDir: parseExamplesDir, staticDir: parseStaticDir}).newExamplesHandler("127.0.0.1", "8090")

	parseRecorder := httptest.NewRecorder()
	parseHandler.ServeHTTP(parseRecorder, httptest.NewRequest(http.MethodGet, "/examples", nil))
	if parseRecorder.Code != http.StatusOK || !strings.Contains(parseRecorder.Body.String(), `<div id="app"></div>`) {
		parseT.Fatalf("expected exact /examples route to render app shell, got code=%d body=%s", parseRecorder.Code, parseRecorder.Body.String())
	}

	parseRecorder = httptest.NewRecorder()
	parseHandler.ServeHTTP(parseRecorder, httptest.NewRequest(http.MethodGet, "/examples/static/index.html", nil))
	if parseRecorder.Code != http.StatusOK || !strings.Contains(parseRecorder.Body.String(), `"catalogHref":"/examples/"`) {
		parseT.Fatalf("expected static index route to render app shell, got code=%d body=%s", parseRecorder.Code, parseRecorder.Body.String())
	}

	parseRecorder = httptest.NewRecorder()
	parseHandler.ServeHTTP(parseRecorder, httptest.NewRequest(http.MethodGet, "/nested/info.txt", nil))
	if parseRecorder.Code != http.StatusNotFound {
		parseT.Fatalf("expected root fallback to examples server 404, got code=%d body=%s", parseRecorder.Code, parseRecorder.Body.String())
	}
}

func TestExamplesHelperFunctionsAdditionalBranches(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseNoHTMLDir := filepath.Join(parseRoot, "no-html")
	if parseErr := os.MkdirAll(parseNoHTMLDir, 0755); parseErr != nil {
		parseT.Fatalf("mkdir no-html dir: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(filepath.Join(parseNoHTMLDir, "readme.txt"), []byte("plain"), 0644); parseErr2 != nil {
		parseT.Fatalf("write text file: %v", parseErr2)
	}
	if parseHtmlFile, parseOk, parseErr3 := firstHTMLFileName(parseNoHTMLDir); parseErr3 != nil || parseOk || parseHtmlFile != "" {
		parseT.Fatalf("expected no-html directory to return no match, got file=%q ok=%t err=%v", parseHtmlFile, parseOk, parseErr3)
	}

	if _, parseOk2, parseErr4 := (&launcher{examplesDir: parseRoot, staticDir: parseRoot}).buildExampleCatalogEntry(parseNoHTMLDir, "no-html", nil); parseErr4 != nil || parseOk2 {
		parseT.Fatalf("expected buildExampleCatalogEntry to skip directories without html, got ok=%t err=%v", parseOk2, parseErr4)
	}

	if parseTitle, parseErr5 := detectHTMLTitle(filepath.Join(parseNoHTMLDir, "missing.html")); parseErr5 == nil || parseTitle != "" {
		parseT.Fatalf("expected detectHTMLTitle read failure, got title=%q err=%v", parseTitle, parseErr5)
	}

	parseHtmlPath := filepath.Join(parseRoot, "untitled.html")
	if parseErr6 := os.WriteFile(parseHtmlPath, []byte("<html><body>no title</body></html>"), 0644); parseErr6 != nil {
		parseT.Fatalf("write html without title: %v", parseErr6)
	}
	if parseTitle2, parseErr7 := detectHTMLTitle(parseHtmlPath); parseErr7 != nil || parseTitle2 != "" {
		parseT.Fatalf("expected html without title to return empty title, got %q err=%v", parseTitle2, parseErr7)
	}

	if parseGot := defaultExampleTitle("example", ""); parseGot != "Example - GoWebComponents" {
		parseT.Fatalf("expected empty html filename fallback title, got %q", parseGot)
	}

	if parseWasm, parseOk3, parseErr8 := detectExampleWasmBinary(parseHtmlPath); parseErr8 != nil || parseOk3 || parseWasm != "" {
		parseT.Fatalf("expected html without wasm reference to return no wasm, got wasm=%q ok=%t err=%v", parseWasm, parseOk3, parseErr8)
	}

	if _, _, parseErr9 := detectExampleWasmBinary(filepath.Join(parseRoot, "missing.html")); parseErr9 == nil {
		parseT.Fatal("expected missing html for wasm detection to fail")
	}

	if parseTag := renderExamplesLoaderScriptTag("/static/bin/app.wasm", "Title", "Message", "/fallback", "Retry"); !strings.HasPrefix(parseTag, "<script>") || !strings.Contains(parseTag, "loadCachedWasm") {
		parseT.Fatalf("expected loader script tag wrapper, got %q", parseTag)
	}
	if parseScript := renderExamplesBootstrapDataScript(examplesShellDocument{RoutePath: "/examples/route/", CatalogHref: "/examples/", ExampleSlug: "route"}); !strings.Contains(parseScript, "__GWC_BOOTSTRAP__") || !strings.Contains(parseScript, "catalogURL") {
		parseT.Fatalf("expected bootstrap data script, got %q", parseScript)
	}
	if _, parseOk4, parseErr10 := firstHTMLFileName(filepath.Join(parseRoot, "missing-dir")); parseErr10 == nil || parseOk4 {
		parseT.Fatalf("expected missing directory html lookup to fail, got ok=%t err=%v", parseOk4, parseErr10)
	}
	if parseGot2 := defaultExampleTitle("fallback slug", "---"); parseGot2 != "fallback slug" {
		parseT.Fatalf("expected punctuation-only html filename to fall back to dir name, got %q", parseGot2)
	}
}

func TestExamplesCatalogHelpersCoverGeneratedAndFallbackBranches(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseExamplesDir := filepath.Join(parseRoot, "examples")
	parseStaticDir := filepath.Join(parseRoot, "static")
	parseExampleDir := filepath.Join(parseExamplesDir, "97-multi-client-presence")
	if parseErr := os.MkdirAll(filepath.Join(parseStaticDir, "bin"), 0755); parseErr != nil {
		parseT.Fatalf("mkdir static bin dir: %v", parseErr)
	}
	if parseErr2 := os.MkdirAll(parseExampleDir, 0755); parseErr2 != nil {
		parseT.Fatalf("mkdir example dir: %v", parseErr2)
	}
	parseHtmlPath := filepath.Join(parseExampleDir, "presence.html")
	parseHtml := "<html><head><title> Presence\n Multi Client </title></head><body><script src=\"/static/bin/presence.wasm\"></script></body></html>"
	if parseErr3 := os.WriteFile(parseHtmlPath, []byte(parseHtml), 0644); parseErr3 != nil {
		parseT.Fatalf("write example html: %v", parseErr3)
	}
	if parseErr4 := os.WriteFile(filepath.Join(parseStaticDir, "bin", "presence.wasm"), []byte("wasm"), 0644); parseErr4 != nil {
		parseT.Fatalf("write wasm artifact: %v", parseErr4)
	}

	parseLauncher := launcher{examplesDir: parseExamplesDir, staticDir: parseStaticDir}
	parseEntry, parseOk, parseErr5 := parseLauncher.buildExampleCatalogEntry(parseExampleDir, "97-multi-client-presence", func(parseDirPath string, parseDirName string, parseHtmlFile string) string {
		return "/custom/" + parseDirName + "/" + parseHtmlFile
	})
	if parseErr5 != nil {
		parseT.Fatalf("build example catalog entry: %v", parseErr5)
	}
	if !parseOk {
		parseT.Fatal("expected example catalog entry to be built")
	}
	if parseEntry.Href != "/custom/97-multi-client-presence/presence.html" {
		parseT.Fatalf("expected custom href, got %#v", parseEntry)
	}
	if parseEntry.Title != "Presence Multi Client" {
		parseT.Fatalf("expected normalized multiline title, got %#v", parseEntry)
	}
	if !parseEntry.UsesWasm || parseEntry.WasmBinary != "presence.wasm" || !parseEntry.MultiClient {
		parseT.Fatalf("expected wasm multi-client entry, got %#v", parseEntry)
	}
	if !hasAnyTag(parseEntry.Tags, "interop", "multi-client", "wasm") {
		parseT.Fatalf("expected multi-client tags, got %#v", parseEntry)
	}

	parsePage, parseOk, parseErr5 := parseLauncher.resolveGeneratedExamplePage("/examples/public/multi-client-presence/")
	if parseErr5 != nil {
		parseT.Fatalf("resolve generated example page: %v", parseErr5)
	}
	if !parseOk || parsePage.ManifestHref != "" || parsePage.GeneratedFrom != "presence.html" {
		parseT.Fatalf("expected manifest-less generated example page, got %#v", parsePage)
	}

	parseMissingStaticDir := filepath.Join(parseRoot, "missing-static")
	if parseErr6 := os.MkdirAll(parseMissingStaticDir, 0755); parseErr6 != nil {
		parseT.Fatalf("mkdir missing static dir: %v", parseErr6)
	}
	if parseWasmBinary, parseUsesWasm, parseErr7 := detectAvailableExampleWasmBinary(parseMissingStaticDir, parseHtmlPath); parseErr7 != nil || parseUsesWasm || parseWasmBinary != "" {
		parseT.Fatalf("expected missing static wasm binary to disable wasm usage, got wasm=%q usesWasm=%t err=%v", parseWasmBinary, parseUsesWasm, parseErr7)
	}
}

func TestResolveExampleCatalogEntryAndHandlerSurfaceStatErrors(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseStaticDir := filepath.Join(parseRoot, "static")
	if parseErr := os.MkdirAll(parseStaticDir, 0755); parseErr != nil {
		parseT.Fatalf("mkdir static dir: %v", parseErr)
	}

	parseLauncher := launcher{repoRoot: parseRoot, examplesDir: filepath.Join(parseRoot, "bad:examples"), staticDir: parseStaticDir}
	if parseEntry, parseOk, parseErr2 := parseLauncher.resolveExampleCatalogEntry("01-counter"); parseErr2 == nil || parseOk {
		parseT.Fatalf("expected invalid example path to return stat error, got entry=%#v ok=%t err=%v", parseEntry, parseOk, parseErr2)
	}

	parseHandler := parseLauncher.newExamplesHandler("127.0.0.1", "8090")
	parseRecorder := httptest.NewRecorder()
	parseHandler.ServeHTTP(parseRecorder, httptest.NewRequest(http.MethodGet, "/examples/public/counter/", nil))
	if parseRecorder.Code != http.StatusInternalServerError {
		parseT.Fatalf("expected invalid example route to surface 500, got %d body=%s", parseRecorder.Code, parseRecorder.Body.String())
	}
	if !strings.Contains(parseRecorder.Body.String(), "examples_route_failed") {
		parseT.Fatalf("expected invalid example route error payload, got %s", parseRecorder.Body.String())
	}
}

func TestExamplesBootstrapHelperErrorBranches(parseT *testing.T) {
	parseOriginalRenderBootstrapData := renderExamplesUIBootstrapScript
	parseOriginalBootstrapScriptFunc := renderExamplesBootstrapScriptFunc
	parseT.Cleanup(func() {
		renderExamplesUIBootstrapScript = parseOriginalRenderBootstrapData
		renderExamplesBootstrapScriptFunc = parseOriginalBootstrapScriptFunc
	})

	renderExamplesUIBootstrapScript = func(parseBootstrap ui.SSRBootstrap, parseNonce string) (string, error) {
		return "", errors.New("bootstrap failed")
	}
	if parseGot := renderExamplesBootstrapDataScript(examplesShellDocument{RoutePath: "/examples/failure/", CatalogHref: "/examples/", ExampleSlug: "failure"}); parseGot != "" {
		parseT.Fatalf("expected bootstrap data script failure to return empty string, got %q", parseGot)
	}

	renderExamplesBootstrapScriptFunc = func(parseWasmURL string, parseFailureTitle string, parseFailureMessage string, parseFailureHref string, parseFailureLinkLabel string) string {
		return "   "
	}
	if parseGot2 := renderExamplesLoaderScriptTag("/static/bin/app.wasm", "Title", "Message", "/fallback", "Retry"); parseGot2 != "" {
		parseT.Fatalf("expected blank bootstrap script body to suppress script tag, got %q", parseGot2)
	}
}

func TestExamplesHandlerPassesThroughNonWasmExampleDirectory(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseExamplesDir := filepath.Join(parseRoot, "examples")
	parseStaticDir := filepath.Join(parseRoot, "static")
	parseExampleDir := filepath.Join(parseExamplesDir, "88-plain-html")
	if parseErr := os.MkdirAll(parseExampleDir, 0755); parseErr != nil {
		parseT.Fatalf("mkdir example dir: %v", parseErr)
	}
	if parseErr2 := os.MkdirAll(parseStaticDir, 0755); parseErr2 != nil {
		parseT.Fatalf("mkdir static dir: %v", parseErr2)
	}
	if parseErr3 := os.WriteFile(filepath.Join(parseExampleDir, "index.html"), []byte("<html><body>plain example</body></html>"), 0644); parseErr3 != nil {
		parseT.Fatalf("write plain html: %v", parseErr3)
	}

	parseHandler := (launcher{repoRoot: parseRoot, examplesDir: parseExamplesDir, staticDir: parseStaticDir}).newExamplesHandler("127.0.0.1", "8090")
	parseRecorder := httptest.NewRecorder()
	parseHandler.ServeHTTP(parseRecorder, httptest.NewRequest(http.MethodGet, "/examples/88-plain-html/", nil))
	if parseRecorder.Code != http.StatusOK {
		parseT.Fatalf("expected non-wasm example directory to pass through, got %d body=%s", parseRecorder.Code, parseRecorder.Body.String())
	}
	parseBody := parseRecorder.Body.String()
	if !strings.Contains(parseBody, "plain example") || strings.Contains(parseBody, "loadCachedWasm") {
		parseT.Fatalf("expected plain html passthrough without generated wasm shell, got %s", parseBody)
	}
}

func TestExamplesRenderingDefaultsAndGeneratedManifest(parseT *testing.T) {
	if parseHtml := renderExamplesAppShellHTML("   ", "   "); !strings.Contains(parseHtml, `"catalogHref":"/examples/"`) || !strings.Contains(parseHtml, `"path":"/examples/"`) {
		parseT.Fatalf("expected app shell defaults for blank inputs, got %s", parseHtml)
	}

	parseRoot := parseT.TempDir()
	parseExamplesDir := filepath.Join(parseRoot, "examples")
	parseStaticDir := filepath.Join(parseRoot, "static")
	if parseErr := os.MkdirAll(filepath.Join(parseExamplesDir, "01-counter"), 0755); parseErr != nil {
		parseT.Fatalf("mkdir example dir: %v", parseErr)
	}
	if parseErr2 := os.MkdirAll(filepath.Join(parseStaticDir, "bin"), 0755); parseErr2 != nil {
		parseT.Fatalf("mkdir static bin dir: %v", parseErr2)
	}
	parseHtml2 := `<html><head><title>Counter</title></head><body><script src="/static/bin/counter.wasm"></script></body></html>`
	if parseErr3 := os.WriteFile(filepath.Join(parseExamplesDir, "01-counter", "index.html"), []byte(parseHtml2), 0644); parseErr3 != nil {
		parseT.Fatalf("write index.html: %v", parseErr3)
	}
	if parseErr4 := os.WriteFile(filepath.Join(parseExamplesDir, "01-counter", "manifest.webmanifest"), []byte("{}\n"), 0644); parseErr4 != nil {
		parseT.Fatalf("write manifest: %v", parseErr4)
	}
	if parseErr5 := os.WriteFile(filepath.Join(parseStaticDir, "bin", "counter.wasm"), []byte("wasm"), 0644); parseErr5 != nil {
		parseT.Fatalf("write wasm: %v", parseErr5)
	}

	parseLauncher := launcher{examplesDir: parseExamplesDir, staticDir: parseStaticDir}
	parsePage, parseOk, parseErr6 := parseLauncher.resolveGeneratedExamplePage("/examples/public/counter/")
	if parseErr6 != nil {
		parseT.Fatalf("resolve generated example page: %v", parseErr6)
	}
	if !parseOk || parsePage.ManifestHref != "./manifest.webmanifest" {
		parseT.Fatalf("expected generated page manifest href, got %#v", parsePage)
	}

	parsePage, parseOk, parseErr6 = parseLauncher.resolveGeneratedExamplePage("   ")
	if parseErr6 != nil {
		parseT.Fatalf("resolve blank generated page: %v", parseErr6)
	}
	if parseOk || parsePage != (generatedExamplePage{}) {
		parseT.Fatalf("expected blank generated page path to skip, got %#v", parsePage)
	}
}

func captureExamplesStdout() (func() (string, error), func(), error) {
	parseOriginalStdout := os.Stdout
	parseReader, parseWriter, parseErr := os.Pipe()
	if parseErr != nil {
		return nil, nil, parseErr
	}
	os.Stdout = parseWriter
	var parseOutput bytes.Buffer
	parseReadDone := make(chan error, 1)
	go func() {
		_, parseCopyErr := io.Copy(&parseOutput, parseReader)
		parseReadDone <- parseCopyErr
	}()

	parseReadOutput := func() (string, error) {
		if parseErr2 := parseWriter.Close(); parseErr2 != nil {
			return "", parseErr2
		}
		if parseErr3 := <-parseReadDone; parseErr3 != nil {
			return "", parseErr3
		}
		return parseOutput.String(), nil
	}
	parseRestore := func() {
		os.Stdout = parseOriginalStdout
		_ = parseWriter.Close()
		_ = parseReader.Close()
	}
	return parseReadOutput, parseRestore, nil
}

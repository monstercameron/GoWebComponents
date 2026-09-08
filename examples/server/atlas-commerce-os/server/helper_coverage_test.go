package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v6/examples/server/atlas-commerce-os/shared/bootfallback"
	"github.com/monstercameron/GoWebComponents/v6/examples/server/atlas-commerce-os/shared/repository"
)

// TestServerLoggingHelperBranches covers direct logging helper and extraction branches.
func TestServerLoggingHelperBranches(parseT *testing.T) {
	parseHeaders := http.Header{}
	parseHeaders.Set("Location", "/app/products/prod-123?atlas_notice=product-saved")
	if parseResourceID := extractServerResourceID(parseHeaders, "/app/products/"); parseResourceID != "prod-123" {
		parseT.Fatalf("expected extracted resource id prod-123, got %q", parseResourceID)
	}
	parseHeaders.Set("Location", "/app/products/prod-123/edit")
	if parseResourceID := extractServerResourceID(parseHeaders, "/app/products/"); parseResourceID != "" {
		parseT.Fatalf("expected nested resource path to be rejected, got %q", parseResourceID)
	}
	parseHeaders.Set("Location", "/app/receiving/rcv-123?atlas_notice=receiving-reconciled")
	if parseResultState := extractServerResultState(parseHeaders); parseResultState != "receiving-reconciled" {
		parseT.Fatalf("expected result state receiving-reconciled, got %q", parseResultState)
	}
	parseHeaders.Set("Location", "://bad-location")
	if parseResultState := extractServerResultState(parseHeaders); parseResultState != "" {
		parseT.Fatalf("expected invalid location to suppress result state, got %q", parseResultState)
	}

	parseServer := &atlasServer{cfg: config{LogsEnabled: true}}
	parseLogBuffer := captureServerLogBuffer(parseT)

	parseServer.logServerRequestEvent(httptest.NewRequest(http.MethodGet, "/assets/logo.svg", nil), http.StatusOK, -5*time.Millisecond, http.Header{})
	parseServer.logServerRequestEvent(httptest.NewRequest(http.MethodDelete, "/api/app/unknown", nil), http.StatusNoContent, 5*time.Millisecond, http.Header{})
	if strings.TrimSpace(parseLogBuffer.String()) != "" {
		parseT.Fatalf("expected skipped helper branches to emit no logs, got %q", parseLogBuffer.String())
	}

	parseMutationHeaders := http.Header{}
	parseMutationHeaders.Set("Location", "/app/products/prod-123?atlas_notice=product-saved")
	parseServer.logServerRequestEvent(httptest.NewRequest(http.MethodPost, "/api/app/products", nil), http.StatusSeeOther, 25*time.Millisecond, parseMutationHeaders)
	parseEntry := decodeServerLogEntryByEvent(parseT, parseLogBuffer, "mutation.result")
	if parseType := strings.TrimSpace(parseEntry["mutation_type"].(string)); parseType != "internal_product_create" {
		parseT.Fatalf("expected internal_product_create mutation type, got %#v", parseEntry["mutation_type"])
	}
	if parseMutationID := strings.TrimSpace(parseEntry["mutation_id"].(string)); parseMutationID != "prod-123" {
		parseT.Fatalf("expected mutation id prod-123, got %#v", parseEntry["mutation_id"])
	}
	if parseResultState := strings.TrimSpace(parseEntry["result_state"].(string)); parseResultState != "product-saved" {
		parseT.Fatalf("expected product-saved result state, got %#v", parseEntry["result_state"])
	}

	parseLogBuffer.Reset()
	parseServer.logServerRequestEvent(httptest.NewRequest(http.MethodPatch, "/api/app/unknown", nil), http.StatusBadRequest, 10*time.Millisecond, http.Header{})
	parseUnknownEntry := decodeServerLogEntryByEvent(parseT, parseLogBuffer, "mutation.result")
	if parseType := strings.TrimSpace(parseUnknownEntry["mutation_type"].(string)); parseType != "mutation_unknown" {
		parseT.Fatalf("expected unknown mutation result log fields, got %#v", parseUnknownEntry)
	}

	parseServer.writeServerLogEvent("bad.event", map[string]any{"invalid": func() {}})
	if !strings.Contains(parseLogBuffer.String(), "server_log_encode_failed") {
		parseT.Fatalf("expected marshal failure fallback log, got %q", parseLogBuffer.String())
	}
}

// TestInventoryCMSHelperBranches covers inventory sort, label, and filter helper branches.
func TestInventoryCMSHelperBranches(parseT *testing.T) {
	if parseLabel := fallbackWarehouseLabel(repository.InventoryRow{WarehouseID: "nj"}); parseLabel != "nj" {
		parseT.Fatalf("expected warehouse id fallback label, got %q", parseLabel)
	}

	parseItemRows := []repository.InventoryRow{
		{Title: "Desk", WarehouseID: "zulu", WarehouseName: "", Available: 5, CoverDays: 12, Inbound: 2, WeeklyUnits: 7, RegionalShare: 3, UpdatedAt: "2026-03-24T10:00:00Z"},
		{Title: "Arm", WarehouseID: "alpha", WarehouseName: "Atlanta", Available: 8, CoverDays: 5, Inbound: 4, WeeklyUnits: 9, RegionalShare: 4, UpdatedAt: "2026-03-25T10:00:00Z"},
	}
	sortWarehouseItemRows(parseItemRows, "warehouse", "")
	if parseItemRows[0].WarehouseID != "alpha" {
		parseT.Fatalf("expected warehouse sort to default ascending, got %+v", parseItemRows)
	}
	sortWarehouseItemRows(parseItemRows, "available", "")
	if parseItemRows[0].Available != 8 {
		parseT.Fatalf("expected available sort to default descending, got %+v", parseItemRows)
	}
	sortWarehouseItemRows(parseItemRows, "cover", "")
	if parseItemRows[0].CoverDays != 5 {
		parseT.Fatalf("expected cover sort to default ascending, got %+v", parseItemRows)
	}
	if parseDirection := normalizeWarehouseItemDirection("updated", ""); parseDirection != "desc" {
		parseT.Fatalf("expected updated direction fallback desc, got %q", parseDirection)
	}

	parseInventoryRows := []repository.InventoryRow{
		{Title: "Desk", Available: 9, DemandScore: 10, WeeklyRevenue: 500, UpdatedAt: "2026-03-24T10:00:00Z"},
		{Title: "Arm", Available: 4, DemandScore: 20, WeeklyRevenue: 900, UpdatedAt: "2026-03-25T10:00:00Z"},
	}
	sortWarehouseInventoryRows(parseInventoryRows, "available")
	if parseInventoryRows[0].Available != 4 {
		parseT.Fatalf("expected available inventory sort ascending, got %+v", parseInventoryRows)
	}
	sortWarehouseInventoryRows(parseInventoryRows, "demand")
	if parseInventoryRows[0].DemandScore != 20 {
		parseT.Fatalf("expected demand inventory sort descending, got %+v", parseInventoryRows)
	}
	sortWarehouseInventoryRows(parseInventoryRows, "revenue")
	if parseInventoryRows[0].WeeklyRevenue != 900 {
		parseT.Fatalf("expected revenue inventory sort descending, got %+v", parseInventoryRows)
	}
	sortWarehouseInventoryRows(parseInventoryRows, "updated")
	if parseInventoryRows[0].UpdatedAt != "2026-03-25T10:00:00Z" {
		parseT.Fatalf("expected default inventory sort by updated descending, got %+v", parseInventoryRows)
	}

	parseFiltered := filterWarehouseInventoryRows([]repository.InventoryRow{
		{SKU: "DESK-001", Title: "Desk", Category: "workspace", MarketPressure: "hot lane", Status: "risk"},
		{SKU: "LAMP-001", Title: "Lamp", Category: "lighting", MarketPressure: "steady", Status: "balanced"},
	}, map[string]string{"q": "desk", "status": "risk"})
	if len(parseFiltered) != 1 || parseFiltered[0].SKU != "DESK-001" {
		parseT.Fatalf("expected one filtered warehouse row, got %+v", parseFiltered)
	}
}

// TestServerRouteHelperBranches covers remaining pure route and snippet helper branches.
func TestServerRouteHelperBranches(parseT *testing.T) {
	// A missing module no longer emits a script at all. The old expectation was a
	// console.warn containing "without hydration" — a warning in a console nobody
	// had open, on a page that rendered blank. The server knows the module is
	// missing before it writes the response, so the explanation is now
	// server-rendered markup and the script is simply unnecessary. The assertion
	// therefore moves from "warns" to "no script, and the page says so".
	if parseSnippet := wasmRuntimeSnippet(false); parseSnippet != "" {
		parseT.Fatalf("expected no boot script when the module is missing, got %q", parseSnippet)
	}
	parseMissingMarkup := atlasBootFallbackMarkup(false)
	for _, parseWant := range []string{
		`id="` + bootfallback.HostID + `"`,
		bootfallback.KindAttr + `="` + bootfallback.KindBinaryMissing + `"`,
		atlasWASMAssetURL,
		atlasWASMBuildCommand,
	} {
		if !strings.Contains(parseMissingMarkup, parseWant) {
			parseT.Fatalf("expected visible missing-module fallback to contain %q, got %q", parseWant, parseMissingMarkup)
		}
	}
	if !strings.Contains(parseMissingMarkup, `role="alert"`) {
		parseT.Fatalf("expected the missing-module fallback to be an alert, got %q", parseMissingMarkup)
	}
	// Nothing on this path may be hidden: no script will run to reveal it.
	if strings.Contains(parseMissingMarkup, " hidden") {
		parseT.Fatalf("missing-module fallback must not ship hidden, got %q", parseMissingMarkup)
	}

	// With the module present the snippet keeps both instantiation paths: the fast
	// streaming one and the ArrayBuffer one that survives a wrong Content-Type.
	parseSnippet := wasmRuntimeSnippet(true)
	for _, parseWant := range []string{"instantiateStreaming", "instantiateFromBuffer", "arrayBuffer", atlasWASMAssetURL} {
		if !strings.Contains(parseSnippet, parseWant) {
			parseT.Fatalf("expected boot snippet to contain %q, got %q", parseWant, parseSnippet)
		}
	}
	// The present-module fallback must ship hidden: a script reveals it.
	if parsePresentMarkup := atlasBootFallbackMarkup(true); !strings.Contains(parsePresentMarkup, ` hidden>`) ||
		!strings.Contains(parsePresentMarkup, `id="`+bootfallback.ReasonIDPrefix+bootfallback.KindUnsupported+`"`) {
		parseT.Fatalf("expected hidden reason blocks when the module is present, got %q", parsePresentMarkup)
	}

	if parseLabel := roleLabel("ops_lead"); parseLabel != "Operations Lead" {
		parseT.Fatalf("expected ops_lead label, got %q", parseLabel)
	}
	if parseLabel := roleLabel("custom_role"); parseLabel != "custom role" {
		parseT.Fatalf("expected fallback role label, got %q", parseLabel)
	}
	if parseDescription := roleDescription("warehouse_supervisor"); !strings.Contains(parseDescription, "Warehouse-focused operator role") {
		parseT.Fatalf("expected warehouse supervisor description, got %q", parseDescription)
	}
	if parseDescription := roleDescription("custom_role"); parseDescription != "Mock Atlas internal role." {
		parseT.Fatalf("expected fallback role description, got %q", parseDescription)
	}

	if parsePath := sanitizeNextPath(""); parsePath != "/app/dashboard" {
		parseT.Fatalf("expected empty next path fallback, got %q", parsePath)
	}
	if parsePath := sanitizeNextPath("//evil.example"); parsePath != "/app/dashboard" {
		parseT.Fatalf("expected double-slash next path rejection, got %q", parsePath)
	}
	if parsePath := sanitizeNextPath("/%zz"); parsePath != "/app/dashboard" {
		parseT.Fatalf("expected invalid next path rejection, got %q", parsePath)
	}
	if parsePath := sanitizeNextPath("/app/inventory?sku=desk-001"); parsePath != "/app/inventory?sku=desk-001" {
		parseT.Fatalf("expected safe next path to pass through, got %q", parsePath)
	}
}

// TestServedDocumentScriptInventory is the ratchet on Atlas's JavaScript surface.
//
// The example's claim is that a browser app can be written in Go, so the number of
// scripts in the served document is a load-bearing fact and not an implementation
// detail. This test pins it: two <script> elements plus one inline snippet, and
// nothing else. wasm_exec.js is the Go toolchain's own glue and cannot be Go;
// __ATLAS_BOOTSTRAP__ is JSON data, not code; the snippet instantiates the module
// and reveals a server-rendered failure message. Anything a fourth script would do
// belongs in the wasm module.
//
// It also pins the two fallbacks, because both regressed silently before: a page
// that cannot boot must say so in the DOM, and a page with scripting disabled must
// have <noscript> content rather than an empty <div id="app">.
func TestServedDocumentScriptInventory(parseT *testing.T) {
	parseServer, parseCleanup := newTestAtlasServer(parseT)
	defer parseCleanup()

	parseReq := httptest.NewRequest(http.MethodGet, "/shop", nil)
	parseRes := httptest.NewRecorder()
	parseServer.routes().ServeHTTP(parseRes, parseReq)
	if parseRes.Code != http.StatusOK {
		parseT.Fatalf("expected %d, got %d", http.StatusOK, parseRes.Code)
	}
	parseBody := parseRes.Body.String()

	// Exactly one external script, and it is the toolchain's.
	if parseCount := strings.Count(parseBody, "<script src="); parseCount != 1 {
		parseT.Fatalf("expected exactly one external script (wasm_exec.js), got %d in %q", parseCount, parseBody)
	}
	if !strings.Contains(parseBody, `<script src="`+atlasWASMExecURL+`">`) {
		parseT.Fatalf("expected wasm_exec.js to be the only external script, got %q", parseBody)
	}
	// The retired logger. It set window.__gwcExampleLogger to four console
	// wrappers that nothing in Atlas called; Go talks to console through
	// syscall/js directly.
	if strings.Contains(parseBody, "example-logger.js") {
		parseT.Fatalf("example-logger.js must not be linked: nothing in Atlas calls it, got %q", parseBody)
	}

	// STYLESHEET INVENTORY — the same ratchet, applied to CSS, and for the same
	// reason: "written in Go" is not a claim you can make while linking a 10,646-line
	// Tailwind build. There must be no <link rel="stylesheet"> of any kind, and
	// neither retired file may reappear by name.
	if strings.Contains(parseBody, `rel="stylesheet"`) {
		parseT.Fatalf("the document must link no stylesheet; shared/design is the only source of CSS, got %q", parseBody)
	}
	for _, parseRetired := range []string{"tailwind.css", "example-shell.css"} {
		if strings.Contains(parseBody, parseRetired) {
			parseT.Fatalf("%s must not be linked: it is replaced by shared/design, got %q", parseRetired, parseBody)
		}
	}
	// And the replacement is actually present, because "no stylesheet" on its own is
	// also what a completely unstyled page looks like. Assert the css package's SSR
	// extraction block, one token from the :root block, and the dark override — i.e.
	// that design.Install() ran, not merely that some CSS exists.
	if !strings.Contains(parseBody, `<style data-gwc-css="`) {
		parseT.Fatalf("expected the inlined design layer as <style data-gwc-css>, got %q", parseBody)
	}
	for _, parseExpected := range []string{
		"--atlas-paper:",
		"--atlas-ink:",
		"prefers-color-scheme:dark",
		"color-scheme:light dark",
	} {
		if !strings.Contains(parseBody, parseExpected) {
			parseT.Fatalf("expected the design global layer to contain %q, got %q", parseExpected, parseBody)
		}
	}
	// The old example-shell.css lock, in both its parts. `color-scheme: dark` pinned
	// the UA to dark regardless of preference and `!important` made the gradients
	// unbeatable from inside the app — together, the reason Atlas reported
	// THEME: light while rendering near-black. Neither may come back through the
	// design system either.
	// Matched with the trailing semicolon on purpose: the DECLARATION
	// `color-scheme:dark;` is the lock, while `@media (prefers-color-scheme:dark)`
	// contains the same characters and is the correct mechanism. A bare substring
	// match here fails on the fix.
	if strings.Contains(parseBody, "color-scheme:dark;") {
		parseT.Fatalf("color-scheme must be `light dark`, not a hard dark lock, got %q", parseBody)
	}
	if strings.Contains(parseBody, "!important") {
		parseT.Fatalf("the emitted design layer must contain no !important, got %q", parseBody)
	}
	// The mount point stays empty. Fallback markup is a SIBLING of #app, never a
	// child, because markup inside the mount is diffed against the client tree
	// during hydration and reported as a mismatch.
	if !strings.Contains(parseBody, `<div id="app"></div>`) {
		parseT.Fatalf("expected an empty mount point, got %q", parseBody)
	}

	// Both fallbacks present, and ordered so the snippet can find what it reveals.
	parseFallbackIndex := strings.Index(parseBody, `id="`+bootfallback.HostID+`"`)
	parseSnippetIndex := strings.Index(parseBody, "instantiateFromBuffer")
	if parseFallbackIndex < 0 || parseSnippetIndex < 0 {
		parseT.Fatalf("expected both the fallback markup and the boot snippet, got %q", parseBody)
	}
	if parseFallbackIndex > parseSnippetIndex {
		parseT.Fatal("fallback markup must precede the boot snippet, or getElementById finds nothing on a synchronous failure")
	}
	if !strings.Contains(parseBody, `<noscript><div `+bootfallback.NoScriptAttr+`="true"`) {
		parseT.Fatalf("expected noscript content for scripting-disabled clients, got %q", parseBody)
	}
	for _, parseKind := range []string{bootfallback.KindUnsupported, bootfallback.KindLoaderMissing, bootfallback.KindBootFailed} {
		if !strings.Contains(parseBody, `id="`+bootfallback.ReasonIDPrefix+parseKind+`"`) {
			parseT.Fatalf("expected a specific fallback message for %q, got %q", parseKind, parseBody)
		}
	}
}

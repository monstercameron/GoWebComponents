//go:build js && wasm && production
// +build js,wasm,production

package utils

import (
	"syscall/js"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/hotreload"
)

// storeUtilsProductionTestGlobal replaces one browser global for the lifetime of one test.
func storeUtilsProductionTestGlobal(parseT *testing.T, parseName string, parseValue js.Value) func() {
	parseT.Helper()
	parseOriginal := js.Global().Get(parseName)
	js.Global().Set(parseName, parseValue)
	return func() {
		js.Global().Set(parseName, parseOriginal)
	}
}

// buildUtilsProductionTestConsole creates one console stub that records structured payloads.
func buildUtilsProductionTestConsole() (js.Value, js.Value, func()) {
	parseConsole := js.Global().Get("Object").New()
	parseEntries := js.Global().Get("Array").New()
	parseFuncs := make([]js.Func, 0, 6)
	for _, parseLevel := range []string{"log", "info", "warn", "error", "debug", "trace"} {
		parseLevel2 := parseLevel
		parseFunc := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
			parseEntry := js.Global().Get("Object").New()
			parseEntry.Set("level", parseLevel2)
			if len(parseArgs) > 0 {
				parseEntry.Set("payload", parseArgs[0])
			}
			parseEntries.Call("push", parseEntry)
			return nil
		})
		parseConsole.Set(parseLevel2, parseFunc)
		parseFuncs = append(parseFuncs, parseFunc)
	}
	return parseConsole, parseEntries, func() {
		for _, parseFunc := range parseFuncs {
			parseFunc.Release()
		}
	}
}

// TestProductionHelpersWasmCoverConfiguration verifies the production compatibility helpers stay disabled and return stable stub state.
func TestProductionHelpersWasmCoverConfiguration(parseT *testing.T) {
	hotreload.Disable()
	parseT.Cleanup(hotreload.Disable)

	if isDebugBuild() {
		parseT.Fatal("expected production build to disable debug mode")
	}
	debugf("interop", "production no-op %d", 1)
	if shouldCollectMemStats() {
		parseT.Fatal("expected production build to disable memstats collection")
	}

	ConfigureMemStatsSampleRate(500)
	if parseRate := GetMemStatsSampleRate(); parseRate != 0 {
		parseT.Fatalf("expected production memstats sampling to stay disabled, got %d", parseRate)
	}

	EnableDebug()
	DisableDebug()
	ConfigureDebugNamespace("router", true)
	ConfigureDebugNamespaces(map[string]bool{"router": true, "cache": false})
	ConfigureDebugNamespacesExclusive(map[string]bool{"interop": true})
	EnableAllDebug()
	DisableAllDebug()
	parseDebugStatus := GetDebugStatus()
	if parseDebugStatus["global"] || parseDebugStatus["hotReload"] {
		parseT.Fatalf("expected all-false debug status, got %#v", parseDebugStatus)
	}

	EnableHotReload(true)
	InstallHotReloadBridge("session", "draft")
	if hotreload.IsEnabled() || IsHotReloadEnabled() {
		parseT.Fatal("expected production hot reload helpers to remain disabled")
	}
	EnableHotReload(false)
	if hotreload.IsEnabled() || IsHotReloadEnabled() {
		parseT.Fatal("expected production hot reload disable path to remain disabled")
	}

	EnableGoroutineMonitoring()
	DisableGoroutineMonitoring()
	ConfigureGoroutineThreshold(5)
	ResetGoroutineBaseline()
	parseStats := GetGoroutineStats()
	if parseStats["current"] != 0 || parseStats["baseline"] != 0 || parseStats["growth"] != 0 || parseStats["detected"] != 0 {
		parseT.Fatalf("expected zeroed production goroutine stats, got %#v", parseStats)
	}
}

// TestProductionConsoleHelpersWasmCoverStructuredLogging verifies production console helpers preserve stable envelopes and fall back cleanly.
func TestProductionConsoleHelpersWasmCoverStructuredLogging(parseT *testing.T) {
	parseConsole, parseEntries, parseConsoleCleanup := buildUtilsProductionTestConsole()
	defer parseConsoleCleanup()
	parseRestoreConsole := storeUtilsProductionTestGlobal(parseT, "console", parseConsole)
	defer parseRestoreConsole()

	WriteConsole("plain %s", "message")
	WriteConsoleStructured(" WARN ", "scope", "structured", map[string]interface{}{
		"count":   3,
		"message": "nested",
		"scope":   "inner",
	})
	if parseEntries.Length() != 2 {
		parseT.Fatalf("expected two console entries, got %d", parseEntries.Length())
	}
	parseStructured := parseEntries.Index(1)
	if parseStructured.Get("level").String() != "warn" {
		parseT.Fatalf("expected normalized warn level, got %q", parseStructured.Get("level").String())
	}
	parsePayload := parseStructured.Get("payload")
	if parsePayload.Get("message").String() != "structured" || parsePayload.Get("scope").String() != "scope" {
		parseT.Fatalf("expected stable message/scope envelope, got payload=%#v", parsePayload)
	}
	if parsePayload.Get("field_message").String() != "nested" || parsePayload.Get("field_scope").String() != "inner" || parsePayload.Get("count").Int() != 3 {
		parseT.Fatalf("expected conflicting fields to be renamed, got payload=%#v", parsePayload)
	}

	js.Global().Set("console", js.Undefined())
	WriteConsoleStructured("", "", "fallback without console", nil)
	consoleFallback("info", "scope", "fallback with fields", map[string]interface{}{"count": 2})
}

// TestProductionResolveDocumentURLWasmCoverBaseResolution verifies production URL resolution follows document and window base rules.
func TestProductionResolveDocumentURLWasmCoverBaseResolution(parseT *testing.T) {
	parseDocument := js.Global().Get("Object").New()
	parseDocument.Set("baseURI", "https://example.test/app/index.html")
	parseRestoreDocument := storeUtilsProductionTestGlobal(parseT, "document", parseDocument)
	defer parseRestoreDocument()
	parseWindow := js.Global().Get("Object").New()
	parseLocation := js.Global().Get("Object").New()
	parseLocation.Set("href", "https://fallback.test/root/page.html")
	parseWindow.Set("location", parseLocation)
	parseRestoreWindow := storeUtilsProductionTestGlobal(parseT, "window", parseWindow)
	defer parseRestoreWindow()

	if parseResolved := ResolveDocumentURL(" "); parseResolved != "" {
		parseT.Fatalf("expected blank relative path to stay empty, got %q", parseResolved)
	}
	if parseResolved := ResolveDocumentURL("../asset.js"); parseResolved != "https://example.test/asset.js" {
		parseT.Fatalf("expected baseURI resolution, got %q", parseResolved)
	}

	parseDocument.Set("baseURI", " ")
	if parseResolved := ResolveDocumentURL("asset.js"); parseResolved != "https://fallback.test/root/asset.js" {
		parseT.Fatalf("expected window.location fallback, got %q", parseResolved)
	}

	parseLocation.Set("href", " ")
	if parseResolved := ResolveDocumentURL("asset.js"); parseResolved != "asset.js" {
		parseT.Fatalf("expected missing base to fall back to relative path, got %q", parseResolved)
	}

	parseDocument.Set("baseURI", "::::")
	if parseResolved := ResolveDocumentURL("asset.js"); parseResolved != "asset.js" {
		parseT.Fatalf("expected invalid base to fall back to relative path, got %q", parseResolved)
	}
	parseDocument.Set("baseURI", "https://example.test/app/index.html")
	if parseResolved := ResolveDocumentURL("%"); parseResolved != "%" {
		parseT.Fatalf("expected invalid relative path to fall back to original value, got %q", parseResolved)
	}
}

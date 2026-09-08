//go:build js && wasm && !production
// +build js,wasm,!production

package utils

import (
	"runtime"
	"sync/atomic"
	"syscall/js"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/hotreload"
)

// storeUtilsTestGlobal replaces a browser global for the lifetime of one test.
func storeUtilsTestGlobal(parseT *testing.T, parseName string, parseValue js.Value) func() {
	parseT.Helper()
	parseOriginal := js.Global().Get(parseName)
	js.Global().Set(parseName, parseValue)
	return func() {
		js.Global().Set(parseName, parseOriginal)
	}
}

// buildUtilsTestConsole creates one console stub that records every call payload.
func buildUtilsTestConsole() (js.Value, js.Value, func()) {
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

// TestDebugHelpersWasmCoverConfiguration verifies the debug and hot-reload compatibility helpers keep their shared state coherent.
func TestDebugHelpersWasmCoverConfiguration(parseT *testing.T) {
	debugEnabled = false
	for parseKey := range debugNamespaces {
		delete(debugNamespaces, parseKey)
	}
	hotreload.Disable()
	ConfigureMemStatsSampleRate(100)
	parseT.Cleanup(func() {
		debugEnabled = false
		for parseKey := range debugNamespaces {
			delete(debugNamespaces, parseKey)
		}
		hotreload.Disable()
		ConfigureMemStatsSampleRate(100)
	})

	EnableDebug()
	if !debugEnabled {
		parseT.Fatal("expected EnableDebug to set global debug mode")
	}
	DisableDebug()
	if debugEnabled {
		parseT.Fatal("expected DisableDebug to clear global debug mode")
	}

	ConfigureDebugNamespace("router", true)
	if !debugNamespaces["router"] {
		parseT.Fatal("expected ConfigureDebugNamespace to store the namespace flag")
	}
	ConfigureDebugNamespaces(map[string]bool{"cache": true, "router": false})
	if !debugNamespaces["cache"] || debugNamespaces["router"] {
		parseT.Fatalf("unexpected namespace map after ConfigureDebugNamespaces: %#v", debugNamespaces)
	}
	ConfigureDebugNamespacesExclusive(map[string]bool{"interop": true})
	if debugEnabled || len(debugNamespaces) != 1 || !debugNamespaces["interop"] {
		parseT.Fatalf("expected exclusive namespace config, got debug=%t namespaces=%#v", debugEnabled, debugNamespaces)
	}

	ConfigureMemStatsSampleRate(0)
	if parseRate := GetMemStatsSampleRate(); parseRate != 1 {
		parseT.Fatalf("expected sample rate floor of 1, got %d", parseRate)
	}

	EnableAllDebug()
	if !debugEnabled {
		parseT.Fatal("expected EnableAllDebug to delegate to EnableDebug")
	}
	InstallHotReloadBridge("theme", "sidebar")
	EnableHotReload(true)
	if !hotreload.IsEnabled() || !IsHotReloadEnabled() {
		parseT.Fatal("expected hot reload compatibility helpers to enable hotreload")
	}

	parseStatus := GetDebugStatus()
	if !parseStatus["global"] || !parseStatus["hotReload"] || !parseStatus["interop"] {
		parseT.Fatalf("unexpected debug status snapshot: %#v", parseStatus)
	}

	DisableAllDebug()
	if debugEnabled || len(debugNamespaces) != 0 {
		parseT.Fatalf("expected DisableAllDebug to clear global and namespace debug state, got debug=%t namespaces=%#v", debugEnabled, debugNamespaces)
	}
	EnableHotReload(false)
	if hotreload.IsEnabled() {
		parseT.Fatal("expected hot reload compatibility helpers to disable hotreload")
	}
}

// TestGoroutineHelpersWasmCoverMonitoring verifies the monitoring helpers update thresholds, stats, and cleanup state consistently.
func TestGoroutineHelpersWasmCoverMonitoring(parseT *testing.T) {
	DisableGoroutineMonitoring()
	goroutineMonitorContext = nil
	goroutineMonitorCancel = nil
	goroutineMonitoringEnabled = false
	debugEnabled = false
	maxGoroutineThreshold = 1000
	baselineGoroutineCount = runtime.NumGoroutine()
	atomic.StoreInt64(&goroutineLeakDetected, 0)
	parseT.Cleanup(func() {
		DisableGoroutineMonitoring()
		goroutineMonitorContext = nil
		goroutineMonitorCancel = nil
		goroutineMonitoringEnabled = false
		debugEnabled = false
		maxGoroutineThreshold = 1000
		baselineGoroutineCount = runtime.NumGoroutine()
		atomic.StoreInt64(&goroutineLeakDetected, 0)
	})

	parseMonitorContext, parseMonitorCancel := buildGoroutineMonitorContext()
	if parseMonitorContext == nil || parseMonitorCancel == nil {
		parseT.Fatal("expected buildGoroutineMonitorContext to return a usable context and cancel func")
	}
	parseMonitorCancel()

	EnableGoroutineMonitoring()
	if !goroutineMonitoringEnabled || goroutineMonitorContext == nil || goroutineMonitorCancel == nil {
		parseT.Fatal("expected EnableGoroutineMonitoring to install monitoring state")
	}
	DisableGoroutineMonitoring()
	if goroutineMonitoringEnabled || goroutineMonitorContext != nil || goroutineMonitorCancel != nil {
		parseT.Fatal("expected DisableGoroutineMonitoring to clear monitoring state")
	}

	ConfigureGoroutineThreshold(0)
	if maxGoroutineThreshold != 1000 {
		parseT.Fatalf("expected threshold fallback to 1000, got %d", maxGoroutineThreshold)
	}
	ConfigureGoroutineThreshold(3)
	if maxGoroutineThreshold != 3 {
		parseT.Fatalf("expected explicit threshold to be stored, got %d", maxGoroutineThreshold)
	}

	baselineGoroutineCount = runtime.NumGoroutine() - 100
	checkGoroutineLeaks()
	maxGoroutineThreshold = 1
	checkGoroutineLeaks()
	if atomic.LoadInt64(&goroutineLeakDetected) == 0 {
		parseT.Fatal("expected leak counter increment when threshold is exceeded")
	}

	ResetGoroutineBaseline()
	parseStats := GetGoroutineStats()
	if parseStats["threshold"] != int64(maxGoroutineThreshold) || parseStats["leaksDetected"] == 0 {
		parseT.Fatalf("unexpected goroutine stats snapshot: %#v", parseStats)
	}
}

// TestConsoleHelpersWasmCoverStructuredLogging verifies structured console writes preserve envelope keys and URL resolution follows browser base rules.
func TestConsoleHelpersWasmCoverStructuredLogging(parseT *testing.T) {
	parseConsole, parseEntries, parseConsoleCleanup := buildUtilsTestConsole()
	defer parseConsoleCleanup()
	parseRestoreConsole := storeUtilsTestGlobal(parseT, "console", parseConsole)
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

	parseDocument := js.Global().Get("Object").New()
	parseDocument.Set("baseURI", "https://example.test/app/index.html")
	parseRestoreDocument := storeUtilsTestGlobal(parseT, "document", parseDocument)
	defer parseRestoreDocument()
	parseWindow := js.Global().Get("Object").New()
	parseLocation := js.Global().Get("Object").New()
	parseLocation.Set("href", "https://fallback.test/root/page.html")
	parseWindow.Set("location", parseLocation)
	parseRestoreWindow := storeUtilsTestGlobal(parseT, "window", parseWindow)
	defer parseRestoreWindow()

	if parseResolved := ResolveDocumentURL("../asset.js"); parseResolved != "https://example.test/asset.js" {
		parseT.Fatalf("expected baseURI resolution, got %q", parseResolved)
	}
	parseDocument.Set("baseURI", " ")
	if parseResolved := ResolveDocumentURL("asset.js"); parseResolved != "https://fallback.test/root/asset.js" {
		parseT.Fatalf("expected window.location fallback, got %q", parseResolved)
	}
	parseLocation.Set("href", "::::")
	if parseResolved := ResolveDocumentURL("asset.js"); parseResolved != "asset.js" {
		parseT.Fatalf("expected invalid base to fall back to the relative path, got %q", parseResolved)
	}
	if parseResolved := ResolveDocumentURL(" "); parseResolved != "" {
		parseT.Fatalf("expected blank relative path to stay empty, got %q", parseResolved)
	}

	consoleFallback("info", "", "plain fallback", nil)
	consoleFallback("info", "scope", "structured fallback", map[string]interface{}{"count": 2})
	debugf("interop", "coverage probe %d", 1)
}

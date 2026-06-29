//go:build js && wasm && production
// +build js,wasm,production

package utils

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/monstercameron/GoWebComponents/v4/deprecation"
	"github.com/monstercameron/GoWebComponents/v4/hotreload"
	"github.com/monstercameron/GoWebComponents/v4/interop"
)

// isDebugBuild returns false for production builds
// This allows the compiler to completely eliminate debug calls
func isDebugBuild() bool {
	return false
}

// debugf is a production-optimized no-op; the compiler inlines and eliminates the call entirely.
func debugf(parseNamespace, format string, parseA ...interface{}) {
	// No-op in production builds - compiler will eliminate this function call
}

// shouldCollectMemStats always returns false in production builds
// This completely eliminates memory stats collection overhead
func shouldCollectMemStats() bool {
	return false
}

// ConfigureMemStatsSampleRate is a production stub that no-ops memory stats configuration.
func ConfigureMemStatsSampleRate(parseRate int64) {
	// No-op in production
}

// GetMemStatsSampleRate always returns 0 in production builds.
func GetMemStatsSampleRate() int64 {
	return 0 // Always disabled in production
}

// EnableDebug is a production stub that does nothing.
func EnableDebug() {}

// DisableDebug is a production stub that does nothing.
func DisableDebug() {}

// ConfigureDebugNamespace is a production stub that does nothing.
func ConfigureDebugNamespace(parseNamespace string, isEnabled bool) {}

// ConfigureDebugNamespaces is a production stub that does nothing.
func ConfigureDebugNamespaces(parseNamespaces map[string]bool) {}

// ConfigureDebugNamespacesExclusive is a production stub that does nothing.
func ConfigureDebugNamespacesExclusive(parseNamespaces map[string]bool) {}

// EnableAllDebug is a production stub that does nothing.
func EnableAllDebug() {}

// DisableAllDebug is a production stub that does nothing.
func DisableAllDebug() {}

// WaitForever blocks indefinitely so js/wasm programs stay alive for events. It
// delegates to interop.KeepAlive, the shared keep-alive primitive also used by
// ui.Run.
func WaitForever() {
	interop.KeepAlive()
}

// GetDebugStatus always returns all-false in production builds.
func GetDebugStatus() map[string]bool {
	return map[string]bool{
		"global":    false,
		"hotReload": false,
	}
}

// EnableHotReload enables or disables hot reload.
//
// Deprecated: use hotreload.Enable / hotreload.Disable directly.
func EnableHotReload(isEnabled bool) {
	deprecation.Warn("utils.EnableHotReload", "hotreload.Enable/hotreload.Disable")
	if isEnabled {
		hotreload.Enable()
		return
	}
	hotreload.Disable()
}

// IsHotReloadEnabled reports whether hot reload is currently enabled.
//
// Deprecated: use hotreload.Enabled.
func IsHotReloadEnabled() bool {
	deprecation.Warn("utils.IsHotReloadEnabled", "hotreload.Enabled")
	return hotreload.Enabled()
}

// InstallHotReloadBridge configures the hot reload bridge with the given atom IDs.
//
// Deprecated: use hotreload.Configure directly.
func InstallHotReloadBridge(parseAtomIDs ...string) {
	deprecation.Warn("utils.InstallHotReloadBridge", "hotreload.Configure")
	hotreload.Configure(hotreload.Config{AtomIDs: parseAtomIDs})
}

// EnableGoroutineMonitoring is a production stub that does nothing.
func EnableGoroutineMonitoring() {}

// DisableGoroutineMonitoring is a production stub that does nothing.
func DisableGoroutineMonitoring() {}

// ConfigureGoroutineThreshold is a production stub that does nothing.
func ConfigureGoroutineThreshold(parseThreshold int) {}

// GetGoroutineStats always returns zero values in production builds.
func GetGoroutineStats() map[string]int64 {
	return map[string]int64{
		"current":  0,
		"baseline": 0,
		"growth":   0,
		"detected": 0,
	}
}

// ResetGoroutineBaseline is a production stub that does nothing.
func ResetGoroutineBaseline() {}

// WriteConsole formats and writes a log entry to the browser console.
func WriteConsole(format string, parseArgs ...interface{}) {
	parseMsg := fmt.Sprintf(format, parseArgs...)
	WriteConsoleStructured("log", "", parseMsg, nil)
}

// WriteConsoleStructured writes a structured log entry to the browser console when available.
func WriteConsoleStructured(parseLevel, parseScope, parseMessage string, parseFields map[string]interface{}) {
	parseNormalizedLevel := strings.ToLower(strings.TrimSpace(parseLevel))
	if parseNormalizedLevel == "" {
		parseNormalizedLevel = "log"
	}

	parseGlobal, parseErr := interop.GetGlobalThis()
	if parseErr != nil {
		consoleFallback(parseNormalizedLevel, parseScope, parseMessage, parseFields)
		return
	}
	parseConsole := parseGlobal.Get("console")
	if !parseConsole.Present() {
		consoleFallback(parseNormalizedLevel, parseScope, parseMessage, parseFields)
		return
	}

	parseEntry := map[string]interface{}{"message": parseMessage}
	if parseScope != "" {
		parseEntry["scope"] = parseScope
	}
	for parseKey, parseValue := range parseFields {
		// Preserve the envelope keys so downstream tooling can rely on a stable message/scope shape.
		if parseKey == "message" || parseKey == "scope" {
			parseEntry["field_"+parseKey] = parseValue
			continue
		}
		parseEntry[parseKey] = parseValue
	}
	if _, parseErr2 := parseConsole.Call(parseNormalizedLevel, parseEntry); parseErr2 == nil {
		return
	}
	consoleFallback(parseNormalizedLevel, parseScope, parseMessage, parseFields)
}

// ResolveDocumentURL resolves a relative asset path against the current document URL.
func ResolveDocumentURL(parseRelative string) string {
	if strings.TrimSpace(parseRelative) == "" {
		return ""
	}

	parseGlobal, parseErr := interop.GetGlobalThis()
	if parseErr != nil {
		return parseRelative
	}

	parseBase := ""
	parseDocument := parseGlobal.Get("document")
	if parseDocument.Present() {
		parseBaseURI := parseDocument.Get("baseURI")
		if parseBaseURI.Present() {
			parseBase = strings.TrimSpace(parseBaseURI.String())
		}
	}
	if parseBase == "" {
		parseWindow := parseGlobal.Get("window")
		if parseWindow.Present() {
			parseLocation := parseWindow.Get("location")
			if parseLocation.Present() {
				parseHref := parseLocation.Get("href")
				if parseHref.Present() {
					parseBase = strings.TrimSpace(parseHref.String())
				}
			}
		}
	}
	if parseBase == "" {
		return parseRelative
	}

	parseBaseURL, parseErr := url.Parse(parseBase)
	if parseErr != nil {
		return parseRelative
	}
	parseRelativeURL, parseErr := url.Parse(parseRelative)
	if parseErr != nil {
		return parseRelative
	}
	return parseBaseURL.ResolveReference(parseRelativeURL).String()
}

func consoleFallback(parseLevel, parseScope, parseMessage string, parseFields map[string]interface{}) {
	parsePrefix := ""
	if parseScope != "" {
		parsePrefix = "[" + parseScope + "] "
	}
	if len(parseFields) == 0 {
		fmt.Printf("%s%s: %s\n", parsePrefix, strings.ToUpper(parseLevel), parseMessage)
		return
	}
	fmt.Printf("%s%s: %s %v\n", parsePrefix, strings.ToUpper(parseLevel), parseMessage, parseFields)
}

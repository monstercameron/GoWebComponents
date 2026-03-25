//go:build js && wasm && production
// +build js,wasm,production

package utils

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/monstercameron/GoWebComponents/hotreload"
	"github.com/monstercameron/GoWebComponents/interop"
)

var hotReloadEnabled bool

// isDebugBuild returns false for production builds
// This allows the compiler to completely eliminate debug calls
func isDebugBuild() bool {
	return false
}

// debugf is a production-optimized no-op; the compiler inlines and eliminates the call entirely.
func debugf(namespace, format string, a ...interface{}) {
	// No-op in production builds - compiler will eliminate this function call
}

// shouldCollectMemStats always returns false in production builds
// This completely eliminates memory stats collection overhead
func shouldCollectMemStats() bool {
	return false
}

// ConfigureMemStatsSampleRate is a production stub that no-ops memory stats configuration.
func ConfigureMemStatsSampleRate(rate int64) {
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
func ConfigureDebugNamespace(namespace string, enabled bool) {}

// ConfigureDebugNamespaces is a production stub that does nothing.
func ConfigureDebugNamespaces(namespaces map[string]bool) {}

// ConfigureDebugNamespacesExclusive is a production stub that does nothing.
func ConfigureDebugNamespacesExclusive(namespaces map[string]bool) {}

// EnableAllDebug is a production stub that does nothing.
func EnableAllDebug() {}

// DisableAllDebug is a production stub that does nothing.
func DisableAllDebug() {}

// GetDebugStatus always returns all-false in production builds.
func GetDebugStatus() map[string]bool {
	return map[string]bool{
		"global":    false,
		"hotReload": false,
	}
}

// EnableHotReload enables or disables hot reload in production builds.
func EnableHotReload(enabled bool) {
	if enabled {
		hotreload.Enable()
		return
	}
	hotreload.Disable()
}

// IsHotReloadEnabled reports whether hot reload is currently enabled.
func IsHotReloadEnabled() bool {
	return hotreload.IsEnabled()
}

// InstallHotReloadBridge configures the hot reload bridge with the given atom IDs.
func InstallHotReloadBridge(atomIDs ...string) {
	hotreload.Configure(hotreload.Config{AtomIDs: atomIDs})
}

// EnableGoroutineMonitoring is a production stub that does nothing.
func EnableGoroutineMonitoring() {}

// DisableGoroutineMonitoring is a production stub that does nothing.
func DisableGoroutineMonitoring() {}

// ConfigureGoroutineThreshold is a production stub that does nothing.
func ConfigureGoroutineThreshold(threshold int) {}

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
func WriteConsole(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	WriteConsoleStructured("log", "", msg, nil)
}

// WriteConsoleStructured writes a structured log entry to the browser console when available.
func WriteConsoleStructured(level, scope, message string, fields map[string]interface{}) {
	normalizedLevel := strings.ToLower(strings.TrimSpace(level))
	if normalizedLevel == "" {
		normalizedLevel = "log"
	}

	global, err := interop.GetGlobalThis()
	if err != nil {
		consoleFallback(normalizedLevel, scope, message, fields)
		return
	}
	console := global.Get("console")
	if !console.Present() {
		consoleFallback(normalizedLevel, scope, message, fields)
		return
	}

	entry := map[string]interface{}{"message": message}
	if scope != "" {
		entry["scope"] = scope
	}
	for key, value := range fields {
		entry[key] = value
	}
	if _, err := console.Call(normalizedLevel, entry); err == nil {
		return
	}
	consoleFallback(normalizedLevel, scope, message, fields)
}

// ResolveDocumentURL resolves a relative asset path against the current document URL.
func ResolveDocumentURL(relative string) string {
	if strings.TrimSpace(relative) == "" {
		return ""
	}

	global, err := interop.GetGlobalThis()
	if err != nil {
		return relative
	}

	base := ""
	document := global.Get("document")
	if document.Present() {
		baseURI := document.Get("baseURI")
		if baseURI.Present() {
			base = strings.TrimSpace(baseURI.String())
		}
	}
	if base == "" {
		window := global.Get("window")
		if window.Present() {
			location := window.Get("location")
			if location.Present() {
				href := location.Get("href")
				if href.Present() {
					base = strings.TrimSpace(href.String())
				}
			}
		}
	}
	if base == "" {
		return relative
	}

	baseURL, err := url.Parse(base)
	if err != nil {
		return relative
	}
	relativeURL, err := url.Parse(relative)
	if err != nil {
		return relative
	}
	return baseURL.ResolveReference(relativeURL).String()
}

func consoleFallback(level, scope, message string, fields map[string]interface{}) {
	prefix := ""
	if scope != "" {
		prefix = "[" + scope + "] "
	}
	if len(fields) == 0 {
		fmt.Printf("%s%s: %s\n", prefix, strings.ToUpper(level), message)
		return
	}
	fmt.Printf("%s%s: %s %v\n", prefix, strings.ToUpper(level), message, fields)
}

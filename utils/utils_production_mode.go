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

// Production-optimized debugf that becomes a no-op
// The compiler will inline this and eliminate the call entirely
func debugf(namespace, format string, a ...interface{}) {
	// No-op in production builds - compiler will eliminate this function call
}

// shouldCollectMemStats always returns false in production builds
// This completely eliminates memory stats collection overhead
func shouldCollectMemStats() bool {
	return false
}

// Production stubs for memory stats configuration
func SetMemStatsSampleRate(rate int64) {
	// No-op in production
}

func GetMemStatsSampleRate() int64 {
	return 0 // Always disabled in production
}

func SetDebug(enabled bool) {}

func SetDebugNamespace(namespace string, enabled bool) {}

func SetDebugNamespaces(namespaces map[string]bool) {}

func SetDebugNamespacesExclusive(namespaces map[string]bool) {}

func EnableAllDebug() {}

func DisableAllDebug() {}

func GetDebugStatus() map[string]bool {
	return map[string]bool{
		"global":    false,
		"hotReload": false,
	}
}

func EnableHotReload(enabled bool) {
	if enabled {
		hotreload.Enable()
		return
	}
	hotreload.Disable()
}

func IsHotReloadEnabled() bool {
	return hotreload.IsEnabled()
}

func InstallHotReloadBridge(atomIDs ...string) {
	hotreload.Configure(hotreload.Config{AtomIDs: atomIDs})
}

func EnableGoroutineMonitoring() {}

func DisableGoroutineMonitoring() {}

func SetGoroutineThreshold(threshold int) {}

func GetGoroutineStats() map[string]int64 {
	return map[string]int64{
		"current":  0,
		"baseline": 0,
		"growth":   0,
		"detected": 0,
	}
}

func ResetGoroutineBaseline() {}

func ConsoleLog(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	ConsoleStructured("log", "", msg, nil)
}

func ConsoleStructured(level, scope, message string, fields map[string]interface{}) {
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

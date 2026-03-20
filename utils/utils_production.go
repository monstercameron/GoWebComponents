//go:build js && wasm && production
// +build js,wasm,production

package utils

import "github.com/monstercameron/GoWebComponents/hotreload"

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
	return hotreload.Enabled()
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

func ConsoleLog(format string, args ...interface{}) {}

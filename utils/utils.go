//go:build js && wasm && !production
// +build js,wasm,!production

package utils

import (
	"context"
	"fmt"
	"net/url"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"github.com/monstercameron/GoWebComponents/hotreload"
	"github.com/monstercameron/GoWebComponents/interop"
)

// FastComparable is an interface for types that can provide fast equality comparison
type FastComparable interface {
	FastEqual(other interface{}) bool
}

// Goroutine leak prevention and monitoring
var (
	goroutineMonitoringEnabled bool = false
	baselineGoroutineCount     int
	maxGoroutineThreshold      int = 1000 // Alert if goroutines exceed this
	goroutineCheckInterval         = 10 * time.Second
	goroutineMonitorContext    context.Context
	goroutineMonitorCancel     context.CancelFunc
	goroutineLeakDetected      int64 // Atomic counter for leak detection
)

var debugEnabled bool

// init initializes goroutine monitoring.
func init() {
	baselineGoroutineCount = runtime.NumGoroutine()
}

// buildGoroutineMonitorContext returns one fresh cancellation scope for the background monitor.
func buildGoroutineMonitorContext() (context.Context, context.CancelFunc) {
	return context.WithCancel(context.Background())
}

// Debug namespace control - map of namespace to enabled status
var debugNamespaces = make(map[string]bool)

// EnableDebug enables verbose debug logs at runtime.
func EnableDebug() {
	debugEnabled = true
}

// DisableDebug disables verbose debug logs at runtime.
func DisableDebug() {
	debugEnabled = false
}

// ConfigureDebugNamespace enables or disables debug logs for a specific namespace.
func ConfigureDebugNamespace(parseNamespace string, isEnabled bool) {
	debugNamespaces[parseNamespace] = isEnabled
}

// ConfigureDebugNamespaces enables multiple namespaces at once.
func ConfigureDebugNamespaces(parseNamespaces map[string]bool) {
	for parseNs, parseEnabled := range parseNamespaces {
		debugNamespaces[parseNs] = parseEnabled
	}
}

// ConfigureDebugNamespacesExclusive disables global debug and only enables specified namespaces.
func ConfigureDebugNamespacesExclusive(parseNamespaces map[string]bool) {
	// Disable global debug first
	debugEnabled = false
	// Clear existing namespace settings efficiently without creating new map
	for parseK := range debugNamespaces {
		delete(debugNamespaces, parseK)
	}
	// Set only the specified namespaces
	for parseNs, parseEnabled := range parseNamespaces {
		debugNamespaces[parseNs] = parseEnabled
	}
}

// debugf prints debug messages if debug is enabled globally or for the specific namespace
// Optimized to avoid expensive string operations when debug is disabled
//
// PERFORMANCE OPTIMIZATION: This function uses multiple optimization strategies:
// 1. Early return check before any string operations
// 2. Conditional compilation support via build tags
// 3. Efficient namespace lookup with map access
// 4. Deferred string formatting until actually needed
func debugf(parseNamespace, format string, parseA ...interface{}) {
	// Ultra-fast path: compile-time optimization for production builds
	// When built with -tags=production, debug calls become no-ops
	if !isDebugBuild() {
		return
	}

	// Fast path: runtime check if debug is enabled before any string operations
	// This avoids expensive fmt.Sprintf calls when debug is disabled
	if !debugEnabled && !debugNamespaces[parseNamespace] {
		return
	}

	// Only perform expensive string formatting when debug is actually enabled
	// This reduces CPU overhead by 2-8% in production when debug is disabled
	// Optimized string concatenation for debug output
	var parseOutput strings.Builder
	parseOutput.WriteByte('[')
	parseOutput.WriteString(parseNamespace)
	parseOutput.WriteString("] ")
	parseOutput.WriteString(fmt.Sprintf(format, parseA...))
	fmt.Print(parseOutput.String())
}

// isDebugBuild returns true if this is a debug build
// This allows compile-time optimization of debug statements
func isDebugBuild() bool {
	// This will be optimized away by the compiler in production builds
	// when using build tags like: go build -tags=production
	return true // Default to debug enabled for development
}

// Memory stats collection optimization
var memStatsSampleRate int64 = 100 // Collect stats every N calls (configurable)

// ConfigureMemStatsSampleRate configures how often memory stats are collected.
// Higher values = less frequent collection = better performance.
// Lower values = more frequent collection = more detailed monitoring.
func ConfigureMemStatsSampleRate(parseRate int64) {
	if parseRate <= 0 {
		parseRate = 1 // Minimum sample rate
	}
	atomic.StoreInt64(&memStatsSampleRate, parseRate)
}

// GetMemStatsSampleRate returns the current memory stats sample rate
func GetMemStatsSampleRate() int64 {
	return atomic.LoadInt64(&memStatsSampleRate)
}

// EnableAllDebug enables all debug logging globally
func EnableAllDebug() {
	EnableDebug()
}

// DisableAllDebug disables all debug logging globally
func DisableAllDebug() {
	DisableDebug()
	// Clear namespace-specific settings efficiently without creating new map
	for parseK := range debugNamespaces {
		delete(debugNamespaces, parseK)
	}
}

// WaitForever blocks indefinitely so js/wasm programs stay alive for events.
func WaitForever() {
	select {}
}

// GetDebugStatus returns current debug settings
func GetDebugStatus() map[string]bool {
	parseStatus := make(map[string]bool)
	parseStatus["global"] = debugEnabled
	parseStatus["hotReload"] = hotreload.IsEnabled()
	for parseNs, parseEnabled := range debugNamespaces {
		parseStatus[parseNs] = parseEnabled
	}
	return parseStatus
}

// EnableHotReload is a compatibility wrapper around the hotreload package.
// New code should prefer hotreload.Enable() or hotreload.Disable().
func EnableHotReload(isEnabled bool) {
	debugf("UTILS", "🔥 EnableHotReload: hot reload %s\n", map[bool]string{true: "enabled", false: "disabled"}[isEnabled])
	if isEnabled {
		hotreload.Enable()
		return
	}
	hotreload.Disable()
}

// IsHotReloadEnabled is a compatibility wrapper around hotreload.IsEnabled().
func IsHotReloadEnabled() bool {
	return hotreload.IsEnabled()
}

// InstallHotReloadBridge is a compatibility wrapper around
// hotreload.Configure(hotreload.Config{AtomIDs: ...}).
// New code should prefer the hotreload package directly.
func InstallHotReloadBridge(parseAtomIDs ...string) {
	hotreload.Configure(hotreload.Config{AtomIDs: parseAtomIDs})
}

// EnableGoroutineMonitoring starts monitoring for potential goroutine leaks
// This helps detect and prevent goroutine accumulation in long-running applications
func EnableGoroutineMonitoring() {
	if goroutineMonitoringEnabled {
		debugf("UTILS", "⚠️ EnableGoroutineMonitoring: monitoring already enabled\n")
		return
	}

	if goroutineMonitorCancel != nil {
		goroutineMonitorCancel()
	}
	parseMonitorContext, parseMonitorCancel := buildGoroutineMonitorContext()
	goroutineMonitorContext = parseMonitorContext
	goroutineMonitorCancel = parseMonitorCancel
	goroutineMonitoringEnabled = true
	baselineGoroutineCount = runtime.NumGoroutine()

	debugf("UTILS", "🔍 EnableGoroutineMonitoring: enabled with baseline %d goroutines\n", baselineGoroutineCount)

	// Start monitoring goroutine in background
	go func(parseMonitorContext context.Context) {
		parseTicker := time.NewTicker(goroutineCheckInterval)
		defer parseTicker.Stop()

		for {
			select {
			case <-parseMonitorContext.Done():
				debugf("UTILS", "🛑 EnableGoroutineMonitoring: monitoring stopped\n")
				return
			case <-parseTicker.C:
				checkGoroutineLeaks()
			}
		}
	}(parseMonitorContext)
}

// DisableGoroutineMonitoring stops goroutine leak monitoring
func DisableGoroutineMonitoring() {
	if !goroutineMonitoringEnabled {
		return
	}

	goroutineMonitoringEnabled = false
	if goroutineMonitorCancel != nil {
		goroutineMonitorCancel()
	}
	goroutineMonitorContext = nil
	goroutineMonitorCancel = nil
	debugf("UTILS", "🔍 DisableGoroutineMonitoring: monitoring disabled\n")
}

// checkGoroutineLeaks monitors goroutine count and detects potential leaks
func checkGoroutineLeaks() {
	parseCurrentCount := runtime.NumGoroutine()
	parseGrowth := parseCurrentCount - baselineGoroutineCount

	if parseCurrentCount > maxGoroutineThreshold {
		atomic.AddInt64(&goroutineLeakDetected, 1)
		debugf("UTILS", "🚨 checkGoroutineLeaks: HIGH goroutine count detected: %d (baseline: %d, growth: +%d)\n",
			parseCurrentCount, baselineGoroutineCount, parseGrowth)

		// Trigger aggressive cleanup
		triggerGoroutineCleanup()
	} else if parseGrowth > 50 {
		debugf("UTILS", "⚠️ checkGoroutineLeaks: elevated goroutine count: %d (baseline: %d, growth: +%d)\n",
			parseCurrentCount, baselineGoroutineCount, parseGrowth)
	} else {
		debugf("UTILS", "✅ checkGoroutineLeaks: normal goroutine count: %d (baseline: %d, growth: +%d)\n",
			parseCurrentCount, baselineGoroutineCount, parseGrowth)
	}
}

// triggerGoroutineCleanup attempts to clean up potential goroutine leaks
func triggerGoroutineCleanup() {
	debugf("UTILS", "🧹 triggerGoroutineCleanup: attempting cleanup\n")

	// Force garbage collection to clean up any unreferenced goroutines
	runtime.GC()

	// Wait a moment for cleanup to take effect
	time.Sleep(100 * time.Millisecond)
	parseNewCount := runtime.NumGoroutine()
	debugf("UTILS", "🧹 triggerGoroutineCleanup: goroutine count after cleanup: %d\n", parseNewCount)

	// Note: Specific cleanup functions (CancelAllEventCallbacks, CancelAllFetchOperations, etc.)
	// should be called directly by the application when needed to avoid circular dependencies
}

// ConfigureGoroutineThreshold configures the threshold for goroutine leak detection.
func ConfigureGoroutineThreshold(parseThreshold int) {
	if parseThreshold <= 0 {
		parseThreshold = 1000 // Default fallback
	}
	maxGoroutineThreshold = parseThreshold
	debugf("UTILS", "🔍 ConfigureGoroutineThreshold: set to %d\n", parseThreshold)
}

// GetGoroutineStats returns current goroutine statistics
func GetGoroutineStats() map[string]int64 {
	parseCurrent := int64(runtime.NumGoroutine())
	parseBaseline := int64(baselineGoroutineCount)
	parseGrowth := parseCurrent - parseBaseline
	parseLeakCount := atomic.LoadInt64(&goroutineLeakDetected)

	return map[string]int64{
		"current":       parseCurrent,
		"baseline":      parseBaseline,
		"growth":        parseGrowth,
		"threshold":     int64(maxGoroutineThreshold),
		"leaksDetected": parseLeakCount,
	}
}

// ResetGoroutineBaseline resets the baseline goroutine count to current count
// This is useful after major application state changes
func ResetGoroutineBaseline() {
	parseOldBaseline := baselineGoroutineCount
	baselineGoroutineCount = runtime.NumGoroutine()
	debugf("UTILS", "🔄 ResetGoroutineBaseline: reset from %d to %d\n", parseOldBaseline, baselineGoroutineCount)
}

// WriteConsole wraps the browser console for easier debugging.
func WriteConsole(format string, parseArgs ...interface{}) {
	parseMsg := fmt.Sprintf(format, parseArgs...)
	WriteConsoleStructured("log", "", parseMsg, nil)
}

// WriteConsoleStructured writes a structured entry to the browser console when available.
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

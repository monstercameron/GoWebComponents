//go:build js && wasm

package ui

import (
	"runtime/debug"
	"strconv"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
)

// Interactive GC pacing defaults for browser apps.
//
// Go's default GOGC=100 collects every time the heap doubles over the live
// set. An interactive wasm UI has a tiny live set (a few MB), so the default
// schedules a stop-the-world collection every few MB of render churn —
// measured as 1–5 collections with pauses up to ~9ms landing INSIDE
// interaction windows, the dominant tail-latency source vs JS frameworks
// (V8 collects tiny nursery garbage incrementally). Trading heap headroom
// for pause frequency is the right default in a browser tab: collections
// happen at ~4x live instead of 2x, with a hard memory ceiling as backstop.
//
// Overrides (localStorage): "gwc:gogc" = "off" keeps Go defaults, or a
// positive integer sets a custom percent. The applied value is reported as
// an info diagnostic so it is visible in devtools.
const (
	interactiveGCPercent     = 300
	interactiveGCMemoryLimit = 512 << 20 // bytes; backstop so the heap cannot grow unbounded
)

// applyInteractiveGCPacing configures the collector once at app init.
func applyInteractiveGCPacing() {
	parsePercent := interactiveGCPercent
	if parseStorage := js.Global().Get("localStorage"); parseStorage.Truthy() {
		if parseOverride := parseStorage.Call("getItem", "gwc:gogc"); parseOverride.Truthy() {
			parseValue := parseOverride.String()
			if parseValue == "off" {
				runtime.ReportDiagnostic("runtime", runtime.DiagnosticInfo, "interactive GC pacing disabled by gwc:gogc=off (Go defaults in effect)")
				return
			}
			if parseCustom, parseErr := strconv.Atoi(parseValue); parseErr == nil && parseCustom > 0 {
				parsePercent = parseCustom
			}
		}
	}
	debug.SetGCPercent(parsePercent)
	debug.SetMemoryLimit(interactiveGCMemoryLimit)
	runtime.ReportDiagnostic("runtime", runtime.DiagnosticInfo,
		"interactive GC pacing applied: GOGC="+strconv.Itoa(parsePercent)+", memory limit 512MB (override via localStorage gwc:gogc)")
}

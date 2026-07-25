//go:build js && wasm

package ui

import (
	"syscall/js"
	"time"

	"github.com/monstercameron/GoWebComponents/v5/internal/runtime"
)

// defaultScheduleTimer schedules a managed setTimeout/setInterval and returns a
// cancel func that clears it and releases the js.Func.
func defaultScheduleTimer(parseFn func(), parseDelay time.Duration, parseRepeat bool) func() {
	parseGlobal := js.Global()
	parseMillis := int(parseDelay / time.Millisecond)
	parseCallback := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) any {
		defer runtime.RecoverContainedPanic("ui", "UseTimer callback")
		parseFn()
		return nil
	})
	parseSetName, parseClearName := "setTimeout", "clearTimeout"
	if parseRepeat {
		parseSetName, parseClearName = "setInterval", "clearInterval"
	}
	parseHandle := parseGlobal.Call(parseSetName, parseCallback, parseMillis)
	return func() {
		parseGlobal.Call(parseClearName, parseHandle)
		parseCallback.Release()
	}
}

//go:build js && wasm

package interop

import "syscall/js"

// RequestAnimationFrame schedules parseCallback to be called before the next
// browser repaint, passing the high-resolution timestamp (in milliseconds) at
// which the frame begins. It mirrors the browser requestAnimationFrame API.
//
// RAF is one-shot: parseCallback fires exactly once and then the loop stops.
// Callers that want a continuous animation must re-call RequestAnimationFrame
// from inside parseCallback on each frame.
//
// The returned cancel func cancels the pending frame if it has not fired yet,
// releasing the associated js.Func. Calling cancel after the callback has
// already fired is a no-op. Either way cancel must eventually be called to
// avoid resource leaks when animation stops early (e.g. on component unmount).
func RequestAnimationFrame(parseCallback func(parseTimestampMillis float64)) (parseCancel func()) {
	var parseJSFunc js.Func
	parseFired := false

	parseJSFunc = js.FuncOf(func(this js.Value, parseArgs []js.Value) any {
		defer RecoverContainedPanic("RequestAnimationFrame")
		parseFired = true
		parseJSFunc.Release()
		var parseTs float64
		if len(parseArgs) > 0 {
			parseTs = parseArgs[0].Float()
		}
		parseCallback(parseTs)
		return nil
	})

	parseHandle := js.Global().Call("requestAnimationFrame", parseJSFunc)

	return func() {
		if !parseFired {
			parseFired = true
			js.Global().Call("cancelAnimationFrame", parseHandle)
			parseJSFunc.Release()
		}
	}
}

//go:build !js || !wasm
// +build !js !wasm

package runtime

// wrapEventHandler is a core package helper.
func (parseRt *Runtime) wrapEventHandler(parseOwner *Fiber, parseFn interface{}) interface{} {
	switch parseTyped := parseFn.(type) {
	case func():
		return func() {
			defer parseRt.recoverEventPanic(parseOwner)
			parseRt.recordFirstInteraction("event")
			parseTyped()
		}
	case func(string):
		return func(parseValue string) {
			defer parseRt.recoverEventPanic(parseOwner)
			parseRt.recordFirstInteraction("event")
			parseTyped(parseValue)
		}
	case func() error:
		return func() error {
			defer parseRt.recoverEventPanic(parseOwner)
			parseRt.recordFirstInteraction("event")
			return parseTyped()
		}
	default:
		return parseFn
	}
}

// recoverEventPanic is a core package helper.
func (parseRt *Runtime) recoverEventPanic(parseOwner *Fiber) {
	if parseRecovered := recover(); parseRecovered != nil {
		if panicPhaseMayRecoverWithBoundary(PanicPhaseEvent) {
			if _, parseHandled := parseRt.recoverBoundaryError(parseOwner, parseRecovered, boundaryPhaseEvent); parseHandled {
				return
			}
		}
		panicFinalUnhandledPanic(parseOwner, boundaryPhaseEvent, parseRecovered)
	}
}

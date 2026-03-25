//go:build !js || !wasm
// +build !js !wasm

package runtime

// wrapEventHandler is a core package helper.
func (parseRuntime *Runtime) wrapEventHandler(parseEventOwner *Fiber, parseEventFn interface{}) interface{} {
	switch parseEventTyped := parseEventFn.(type) {
	case func():
		return func() {
			defer parseRuntime.recoverEventPanic(parseEventOwner)
			parseRuntime.recordFirstInteraction("event")
			parseEventTyped()
		}
	case func(string):
		return func(parseEventValue string) {
			defer parseRuntime.recoverEventPanic(parseEventOwner)
			parseRuntime.recordFirstInteraction("event")
			parseEventTyped(parseEventValue)
		}
	case func() error:
		return func() error {
			defer parseRuntime.recoverEventPanic(parseEventOwner)
			parseRuntime.recordFirstInteraction("event")
			return parseEventTyped()
		}
	default:
		return parseEventFn
	}
}

// recoverEventPanic is a core package helper.
func (parseRuntime *Runtime) recoverEventPanic(parseEventOwner *Fiber) {
	if parseEventRecovered := recover(); parseEventRecovered != nil {
		if panicPhaseMayRecoverWithBoundary(PanicPhaseEvent) {
			if _, parseEventHandled := parseRuntime.recoverBoundaryError(parseEventOwner, parseEventRecovered, boundaryPhaseEvent); parseEventHandled {
				return
			}
		}
		panicFinalUnhandledPanic(parseEventOwner, boundaryPhaseEvent, parseEventRecovered)
	}
}

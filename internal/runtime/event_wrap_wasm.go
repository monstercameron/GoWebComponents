//go:build js && wasm

package runtime

import "syscall/js"

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
	case func(js.Value):
		return func(parseEventJSValue js.Value) {
			defer parseRuntime.recoverEventPanic(parseEventOwner)
			parseRuntime.recordFirstInteraction("event")
			parseEventTyped(parseEventJSValue)
		}
	case func() error:
		return func() error {
			defer parseRuntime.recoverEventPanic(parseEventOwner)
			parseRuntime.recordFirstInteraction("event")
			return parseEventTyped()
		}
	case func(js.Value) error:
		return func(parseEventJSValue js.Value) error {
			defer parseRuntime.recoverEventPanic(parseEventOwner)
			parseRuntime.recordFirstInteraction("event")
			return parseEventTyped(parseEventJSValue)
		}
	case func(GoEvent):
		return func(parseEventValue GoEvent) {
			defer parseRuntime.recoverEventPanic(parseEventOwner)
			parseRuntime.recordFirstInteraction("event")
			parseEventTyped(parseEventValue)
		}
	case func(GoEvent) error:
		return func(parseEventValue GoEvent) error {
			defer parseRuntime.recoverEventPanic(parseEventOwner)
			parseRuntime.recordFirstInteraction("event")
			return parseEventTyped(parseEventValue)
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

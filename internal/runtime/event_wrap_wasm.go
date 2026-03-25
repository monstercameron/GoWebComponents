//go:build js && wasm
// +build js,wasm

package runtime

import "syscall/js"

func (rt *Runtime) wrapEventHandler(owner *Fiber, fn interface{}) interface{} {
	switch typed := fn.(type) {
	case func():
		return func() {
			defer rt.recoverEventPanic(owner)
			rt.recordFirstInteraction("event")
			typed()
		}
	case func(string):
		return func(value string) {
			defer rt.recoverEventPanic(owner)
			rt.recordFirstInteraction("event")
			typed(value)
		}
	case func(js.Value):
		return func(value js.Value) {
			defer rt.recoverEventPanic(owner)
			rt.recordFirstInteraction("event")
			typed(value)
		}
	case func() error:
		return func() error {
			defer rt.recoverEventPanic(owner)
			rt.recordFirstInteraction("event")
			return typed()
		}
	case func(js.Value) error:
		return func(value js.Value) error {
			defer rt.recoverEventPanic(owner)
			rt.recordFirstInteraction("event")
			return typed(value)
		}
	case func(GoEvent):
		return func(event GoEvent) {
			defer rt.recoverEventPanic(owner)
			rt.recordFirstInteraction("event")
			typed(event)
		}
	case func(GoEvent) error:
		return func(event GoEvent) error {
			defer rt.recoverEventPanic(owner)
			rt.recordFirstInteraction("event")
			return typed(event)
		}
	default:
		return fn
	}
}

func (rt *Runtime) recoverEventPanic(owner *Fiber) {
	if recovered := recover(); recovered != nil {
		if panicPhaseMayRecoverWithBoundary(PanicPhaseEvent) {
			if _, handled := rt.recoverBoundaryError(owner, recovered, boundaryPhaseEvent); handled {
				return
			}
		}
		panicFinalUnhandledPanic(owner, boundaryPhaseEvent, recovered)
	}
}

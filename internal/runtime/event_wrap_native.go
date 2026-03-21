//go:build !js || !wasm
// +build !js !wasm

package runtime

func (rt *Runtime) wrapEventHandler(owner *Fiber, fn interface{}) interface{} {
	switch typed := fn.(type) {
	case func():
		return func() {
			defer rt.recoverEventPanic(owner)
			typed()
		}
	case func(string):
		return func(value string) {
			defer rt.recoverEventPanic(owner)
			typed(value)
		}
	case func() error:
		return func() error {
			defer rt.recoverEventPanic(owner)
			return typed()
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

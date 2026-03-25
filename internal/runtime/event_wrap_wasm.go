//go:build js && wasm
// +build js,wasm

package runtime

import "syscall/js"

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
	case func(js.Value):
		return func(parseValue2 js.Value) {
			defer parseRt.recoverEventPanic(parseOwner)
			parseRt.recordFirstInteraction("event")
			parseTyped(parseValue2)
		}
	case func() error:
		return func() error {
			defer parseRt.recoverEventPanic(parseOwner)
			parseRt.recordFirstInteraction("event")
			return parseTyped()
		}
	case func(js.Value) error:
		return func(parseValue3 js.Value) error {
			defer parseRt.recoverEventPanic(parseOwner)
			parseRt.recordFirstInteraction("event")
			return parseTyped(parseValue3)
		}
	case func(GoEvent):
		return func(parseEvent GoEvent) {
			defer parseRt.recoverEventPanic(parseOwner)
			parseRt.recordFirstInteraction("event")
			parseTyped(parseEvent)
		}
	case func(GoEvent) error:
		return func(parseEvent2 GoEvent) error {
			defer parseRt.recoverEventPanic(parseOwner)
			parseRt.recordFirstInteraction("event")
			return parseTyped(parseEvent2)
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

package ui

import (
	"time"

	"github.com/monstercameron/GoWebComponents/interop"
)

// Trigger hooks produce the boolean a UseDefer latch consumes. Each returns false until its
// condition first occurs, then true; on native/SSR builds (no effect lifecycle / no DOM)
// they stay false, so deferred content is simply absent from the server render and mounts in
// the browser once the trigger fires.

// UseTimerTrigger returns true once delay has elapsed since mount — the timer trigger for a
// deferred view (Angular `@defer (on timer(...))`).
func UseTimerTrigger(parseDelay time.Duration) bool {
	parseFired := UseState(false)
	UseEffect(func() func() {
		parseTimer, parseErr := interop.SetTimeout(parseDelay, func() { parseFired.Set(true) })
		if parseErr != nil {
			return nil
		}
		return func() { _ = parseTimer.Cancel() }
	})
	return parseFired.Get()
}

// UseIdle returns true once the browser has reached an idle point after mount — the idle
// trigger (`@defer (on idle)`). It is approximated with a deferred (next-task) timeout, which
// schedules after the current render/commit work, where a true requestIdleCallback is
// unavailable.
func UseIdle() bool {
	return UseTimerTrigger(0)
}

// UseInteraction returns true after the first pointer interaction with the referenced
// element — the interaction trigger (`@defer (on interaction)`), for deferring a heavy view
// until the user actually touches it.
func UseInteraction(parseRef DOMRef) bool {
	parseFired := UseState(false)
	UsePointerEvents(parseRef, PointerHandlers{
		OnPointerDown: func(Event) { parseFired.Set(true) },
	})
	return parseFired.Get()
}

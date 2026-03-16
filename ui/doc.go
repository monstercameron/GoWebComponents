// Package ui provides the preferred public component API for GoWebComponents.
//
// It exposes:
//   - CreateElement for component composition
//   - Render for browser mounting
//   - CreateContext, UseContext, and Provider components for subtree-scoped values
//   - UseState, UseReducer, UseForm, UseEffect, UseMemo, UseRef, UsePrevious, UseDeferredValue, UseDebounced, UseThrottled, UseChannel, UseTask, UseLazyNode, UseTransition, StartTransition, and UseId for local stateful logic
//   - AsyncBoundary, ErrorBoundary, and Lazy for explicit async subtree loading, panic recovery, and fallback rendering
//   - UseEvent for typed event handler wrapping
//
// UseReducer is intended for components whose local state behaves like a small
// state machine with several coordinated transitions. Prefer UseState.Update for
// simpler structs; reach for UseReducer when local actions are clearer than many
// ad hoc field mutations spread across handlers.
//
// UseForm is intended for multi-field local forms that need touched/dirty state,
// structured field errors, validation, and a submission lifecycle without having
// to rebuild that bookkeeping in every example.
//
// UsePrevious is intended for cases where a component needs to compare the
// current value against the last committed one without introducing extra state.
// Typical uses include change detection, transition styling, and one-step diff
// logic inside a component:
//
//	func SearchStatus(query string) ui.Node {
//		previous := ui.UsePrevious(query)
//		message := "First search"
//		if previous.Ok() && previous.Get() != query {
//			message = "Updated from " + previous.Get() + " to " + query
//		}
//
//		return html.P(html.Props{}, html.Text(message))
//	}
//
// It is not intended as a substitute for real state management. Prefer
// UseState, UseMemo, or state.UseAtom when the value needs to drive updates of
// its own rather than only inform render-time comparisons.
//
// UseChannel is intended for components that consume streamed values from
// goroutines, timers, or worker-style producers. It exposes the latest received
// value together with availability and closure state without forcing every
// component to hand-roll channel bookkeeping.
//
// UseTask is intended for explicit background jobs that should be started and
// cancelled by UI actions. It exposes typed task state and keeps cancellation in
// terms of context.Context rather than custom ad hoc flags in each component.
//
// UseDebounced and UseThrottled are small input-oriented convenience hooks.
// They are intended for search boxes, live filtering, or fast-changing UI state
// where callers want delayed or rate-limited derived values without open-coding
// timer cleanup in every component.
//
// CreateContext and UseContext are intended for subtree-scoped values such as
// theme, auth/session state, app configuration, or service-style helpers that
// should not be threaded manually through many intermediate component props.
// Missing providers currently resolve to the context default value.
//
// AsyncBoundary is an explicit async rendering primitive for loading and error
// fallbacks around a subtree. Lazy builds on top of it by resolving a ui.Node
// asynchronously and routing the result through the same boundary semantics.
// The first implementation is intentionally explicit: callers pass Pending,
// Error, and fallback nodes rather than relying on implicit promise throwing.
// ErrorBoundary is the sibling recovery primitive for unexpected panics during
// render, effect, cleanup, and event-handler execution. It renders a fallback
// subtree instead of letting a child failure tear down the entire app tree.
// StartTransition and UseTransition provide a small non-urgent scheduling lane
// for background tree refreshes, while UseDeferredValue keeps rendering the
// last committed value until a transition catches up. The current scheduler
// deliberately does not expose a separate layout-effect hook yet; DOM-read-
// before-paint scenarios should continue to use explicit event sequencing and
// the existing UseEffect surface until a stronger concrete need appears.
//
// The ui package is the recommended replacement for the older dom/hooks/render
// split when authoring new components.
package ui

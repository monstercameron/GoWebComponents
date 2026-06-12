// Package ui provides the preferred public component API for GoWebComponents.
//
// It exposes:
//   - Component as a concise alias for CreateElement
//   - CreateElement for component composition
//   - If and Match for lazy conditional node rendering
//   - Render for browser mounting
//   - RenderToString and RenderToStream for request-time SSR output
//   - HydrateIsland and HydrationIsland for progressive, independently resumed SSR islands
//   - Portal for rendering a subtree into a selector or explicit host node outside the current DOM parent
//   - CreateContext, UseContext, and Provider components for subtree-scoped values
//   - UseState, UseReducer, UseForm, UseEffect, UseMemo, UseRef, UsePrevious, UseDeferredValue, UseDebounced, UseThrottled, UseChannel, UseTask, UseWorkerTask, UseLazyNode, UseTransition, StartTransition, UseId, UseFocusManager, UseFocusTrap, UseCompositeNavigation, and UseAnnouncer for local stateful logic
//   - AsyncBoundary, ErrorBoundary, and Lazy for explicit async subtree loading, panic recovery, and fallback rendering
//   - UseEvent for typed event handler wrapping
//   - AccessibleOverlay for portal-backed dialog and overlay semantics with focus trapping and dismissal behavior
//
// UseReducer is intended for components whose local state behaves like a small
// state machine with several coordinated transitions. Prefer UseState.Update for
// simpler structs; reach for UseReducer when local actions are clearer than many
// ad hoc field mutations spread across handlers.
//
// UseForm is intended for multi-field local forms that need touched/dirty state,
// structured field errors, validation, a submission lifecycle, and a small set
// of transport-oriented helpers such as server error mapping and CSRF naming
// conventions without having to rebuild that bookkeeping in every example.
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
// UseWorkerTask is the worker-backed sibling for browser-only CPU-heavy jobs. It
// reuses a dedicated browser worker across runs, reports typed progress payloads,
// and tears the worker down with the owning component instead of requiring each
// feature to hand-roll worker lifecycle and request-correlation logic.
//
// UseDebounced and UseThrottled are small input-oriented convenience hooks.
// They are intended for search boxes, live filtering, or fast-changing UI state
// where callers want delayed or rate-limited derived values without open-coding
// timer cleanup in every component.

// UseFocusManager and UseFocusTrap provide the first public accessibility-
// oriented focus helpers. They are intended for returning focus after dialogs
// close, moving focus to the next meaningful control after route or async UI
// changes, and keeping keyboard focus inside modal-style overlays.

// UseCompositeNavigation is intended for reusable composite widgets such as
// tabs, listboxes, menus, and command-palette results. It provides roving
// tabindex, arrow-key movement, Home/End handling, and simple typeahead so
// applications do not have to hand-roll the same keyboard loop repeatedly.

// UseAnnouncer provides an application-local live-region helper for polite and
// assertive announcements. It is intended for validation feedback, route or
// async status updates, and other spoken status changes that should be wired to
// ordinary HTML rather than a separate runtime subsystem.
//
// CreateContext and UseContext are intended for subtree-scoped values such as
// theme, auth/session state, app configuration, or service-style helpers that
// should not be threaded manually through many intermediate component props.
// Missing providers currently resolve to the context default value.
//
// AsyncBoundary is an explicit async rendering primitive for loading and error
// fallbacks around a subtree. Lazy builds on top of it by resolving a ui.Node
// asynchronously and routing the result through the same boundary semantics.
// Components can also call SuspendUntil or Await during render to suspend the
// subtree until a data dependency signals readiness.
// RenderToStream turns those suspended async boundaries into fallback-first
// shell markup and later boundary replacement chunks, while RenderToString keeps
// the synchronous all-at-once HTML path.
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
// The public composition model deliberately centers on ordinary children,
// explicit props structs, context, portals, and router layout routes. There is
// no first-class slot API today. If a component needs named insertion points,
// model them as explicit props or child subtrees rather than depending on an
// undocumented slot convention.
//
// The ui package is the recommended replacement for the older dom/hooks/render
// split when authoring new components.
package ui

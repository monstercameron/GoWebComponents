// Package state provides shared atom-based state management for GoWebComponents.
//
// This package implements atom-based state management inspired by SolidJS and Jotai,
// allowing components to subscribe to global state that persists across the entire
// application and automatically triggers re-renders when values change.
//
// Basic usage:
//
//	import "github.com/monstercameron/GoWebComponents/state"
//
//	// In any component
//	func UserProfile() ui.Node {
//	    username := state.UseAtom("currentUser", "Guest")
//	    login := ui.UseEvent(func() {
//	        username.Set("John Doe")
//	    })
//
//	    return html.Div(html.Props{},
//	        html.H1(html.Props{}, html.Text(fmt.Sprintf("Welcome, %s", username.Get()))),
//	        html.Button(html.Props{OnClick: login}, html.Text("Login")),
//	    )
//	}
//
//	// In a different component - shares the same state!
//	func NavBar() ui.Node {
//	    username := state.UseAtom("currentUser", "Guest")
//	    return html.Nav(html.Props{}, html.Span(html.Props{}, html.Text(username.Get())))
//	}
//
// Key features:
//
//   - Global state accessible from any component by ID
//   - Automatic subscription and re-rendering when state changes
//   - Type-safe with Go generics
//   - Thread-safe for concurrent access
//   - Subscription-scoped updates - only subscribed components re-render
//   - UseComputed for typed derived values inside components
//   - UseDerived for read-only shared derived atoms with explicit source dependencies
//   - Snapshot export/import for in-memory restore and optional browser persistence helpers
//
// Atoms vs Component State:
//
//   - Use state.UseAtom for data that needs to be shared across components
//   - Use state.UseComputed for memoized derived values based on atoms, props, or local state
//   - Use state.UseDerived for shared read-only derived atoms keyed by ID and explicit source atom dependencies
//   - Use ui.UseState for local component state
//   - Atoms persist across component unmounts
//   - Atoms trigger updates in all subscribed components
//
// Snapshot persistence notes:
//
//   - ExportSnapshot and ImportSnapshot preserve exact Go values for same-process restore.
//   - SaveSnapshot and LoadSnapshot encode snapshots as JSON for browser storage.
//   - SavePersistentSnapshot and LoadPersistentSnapshot use IndexedDB-first durable storage with explicit fallback behavior.
//   - JSON persistence is only stable for JSON-compatible atom values; numeric and struct-heavy
//     atoms may need caller-owned codecs if exact round-tripping is required.
package state

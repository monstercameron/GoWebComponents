// Package state provides global state management with fine-grained reactivity
// for GoWebComponents.
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
//   - Fine-grained reactivity - only subscribed components re-render
//
// Atoms vs Component State:
//
//   - Use state.UseAtom for data that needs to be shared across components
//   - Use ui.UseState for local component state
//   - Atoms persist across component unmounts
//   - Atoms trigger updates in all subscribed components
package state

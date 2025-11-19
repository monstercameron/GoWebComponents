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
//	func UserProfile(props dom.Attrs) *fiber.Element {
//	    username, setUsername := state.UseAtom("currentUser", "Guest")
//
//	    return dom.Div(nil,
//	        dom.H1(nil, fmt.Sprintf("Welcome, %s", username())),
//	        dom.Button(map[string]interface{}{
//	            "onclick": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
//	                setUsername("John Doe")
//	                return nil
//	            }),
//	        }, "Login"),
//	    )
//	}
//
//	// In a different component - shares the same state!
//	func NavBar(props dom.Attrs) *fiber.Element {
//	    username, _ := state.UseAtom("currentUser", "Guest")
//	    return dom.Nav(nil, dom.Span(nil, username()))
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
//   - Use hooks.UseState for local component state
//   - Atoms persist across component unmounts
//   - Atoms trigger updates in all subscribed components
package state

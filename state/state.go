//go:build js && wasm
// +build js,wasm

package state

import (
	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

// Type alias for Element
type Element = runtime.Element

// UseAtom provides SolidJS-style fine-grained reactivity with global atoms.
// Atoms are accessible from anywhere in the component tree by ID and
// automatically trigger re-renders in all subscribed components when updated.
//
// The first component to call UseAtom with a specific ID initializes the atom
// with the provided initial value. Subsequent calls from other components will
// use the existing value and subscribe to updates.
//
// Type parameter T can be any Go type. The hook uses generic type parameters
// for type safety.
//
// Returns:
//   - A getter function that returns the current atom value
//   - A setter function that updates the atom and triggers re-renders in all subscribers
//
// Example - Theme Management:
//
//	// In theme switcher component
//	func ThemeSwitcher(props dom.Attrs) *fiber.Element {
//	    theme, setTheme := state.UseAtom("appTheme", "light")
//
//	    toggle := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
//	        if theme() == "light" {
//	            setTheme("dark")
//	        } else {
//	            setTheme("light")
//	        }
//	        return nil
//	    })
//
//	    return dom.Button(map[string]interface{}{"onclick": toggle},
//	        fmt.Sprintf("Switch to %s mode", theme()))
//	}
//
//	// In header component - automatically updates when theme changes
//	func Header(props dom.Attrs) *fiber.Element {
//	    theme, _ := state.UseAtom("appTheme", "light")
//
//	    bgColor := "white"
//	    if theme() == "dark" {
//	        bgColor = "#333"
//	    }
//
//	    return dom.Header(map[string]interface{}{
//	        "style": map[string]string{"background-color": bgColor},
//	    }, dom.H1(nil, "My App"))
//	}
//
// Example - User Authentication:
//
//	type User struct {
//	    ID       int
//	    Username string
//	    Email    string
//	}
//
//	func LoginButton(props dom.Attrs) *fiber.Element {
//	    user, setUser := state.UseAtom("currentUser", User{})
//
//	    login := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
//	        setUser(User{ID: 1, Username: "johndoe", Email: "john@example.com"})
//	        return nil
//	    })
//
//	    if user().ID == 0 {
//	        return dom.Button(map[string]interface{}{"onclick": login}, "Login")
//	    }
//	    return dom.Span(nil, fmt.Sprintf("Welcome, %s", user().Username))
//	}
//
// Thread Safety:
// UseAtom is thread-safe and can be safely called from multiple goroutines.
// The internal atom registry uses mutex-based synchronization.
//
// Cleanup:
// When a component unmounts, it is automatically unsubscribed from all atoms
// to prevent memory leaks and unnecessary updates.
//
// Best Practices:
//   - Use descriptive atom IDs (e.g., "currentUser", "appTheme", "shoppingCart")
//   - Initialize atoms with appropriate default values
//
// Consider using structured types (structs) for complex state
//   - Avoid storing large amounts of data in atoms (use for coordination, not caching)
func UseAtom[T any](id string, initialValue T) (func() T, func(T)) {
	return runtime.GoUseAtomGlobal(id, initialValue)
}

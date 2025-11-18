# State Package

**Location:** `/state`

```
GoWebComponents/
├── dom/
├── hooks/
├── state/            ← YOU ARE HERE
│   ├── doc.go
│   └── state.go
├── render/
├── router/
├── fetch/
├── internal/
├── examples/
└── ...
```

## Overview

The `state` package provides global state management through **atoms** - named pieces of state that can be shared across components. Unlike `hooks.UseState` which is local to a component, atoms enable multiple components to subscribe to and update the same state.

## Core API

### `UseAtom[T](key string, initialValue T) (func() T, func(T))`

Creates or connects to a global state atom identified by a unique key.

```go
// Component A - Controller
func CounterController(props dom.Attrs) *dom.Element {
    count, setCount := state.UseAtom("global-counter", 0)

    handleIncrement := hooks.GoUseFunc(func(e dom.GoEvent) {
        setCount(count() + 1)
    })

    return dom.Button(dom.Attrs{
        "onclick": handleIncrement,
    }, dom.Text("Increment"))
}

// Component B - Display
func CounterDisplay(props dom.Attrs) *dom.Element {
    count, _ := state.UseAtom("global-counter", 0)

    return dom.P(nil,
        dom.Text(fmt.Sprintf("Count: %d", count())))
}
```

**Key Features:**

- **Shared State**: Multiple components can subscribe to the same atom
- **Type-Safe**: Generic type parameter ensures type safety
- **Automatic Re-renders**: All subscribed components re-render when atom updates
- **Initial Value**: If atom doesn't exist, creates it with `initialValue`

## Use Cases

### 1. Application Theme

```go
// In theme toggle component
theme, setTheme := state.UseAtom("app-theme", "light")

toggleTheme := hooks.GoUseFunc(func(e dom.GoEvent) {
    if theme() == "light" {
        setTheme("dark")
    } else {
        setTheme("light")
    }
})

// In any other component
theme, _ := state.UseAtom("app-theme", "light")
className := theme() + "-mode"
```

### 2. User Session

```go
type User struct {
    ID       int
    Username string
    Email    string
}

// In login component
user, setUser := state.UseAtom("current-user", User{})

// In navbar
user, _ := state.UseAtom("current-user", User{})
return dom.Span(nil, dom.Text(user().Username))

// In profile page
user, _ := state.UseAtom("current-user", User{})
// Access user().Email, user().ID, etc.
```

### 3. Shopping Cart

```go
type CartItem struct {
    ProductID int
    Quantity  int
    Price     float64
}

// Add to cart button
cart, setCart := state.UseAtom("shopping-cart", []CartItem{})

addToCart := hooks.GoUseFunc(func(e dom.GoEvent) {
    currentCart := cart()
    newCart := append(currentCart, CartItem{
        ProductID: productID,
        Quantity: 1,
        Price: product.Price,
    })
    setCart(newCart)
})

// Cart display component
cart, _ := state.UseAtom("shopping-cart", []CartItem{})
total := calculateTotal(cart())
```

### 4. Modal/Dialog State

```go
// In any component
isOpen, setIsOpen := state.UseAtom("modal-open", false)

openModal := hooks.GoUseFunc(func(e dom.GoEvent) {
    setIsOpen(true)
})

// In modal component
isOpen, setIsOpen := state.UseAtom("modal-open", false)

if !isOpen() {
    return nil // Don't render modal
}

return dom.Dialog(/* ... modal content ... */)
```

## Atom Patterns

### Read-Only Access

Components that only need to read an atom can ignore the setter:

```go
theme, _ := state.UseAtom("app-theme", "light")
// Only uses theme(), never updates it
```

### Write-Only Access

Components that only update can ignore the getter (less common):

```go
_, setTheme := state.UseAtom("app-theme", "light")
// Only calls setTheme(), doesn't read current value
```

### Derived State

Compute values based on atom state:

```go
cart, _ := state.UseAtom("shopping-cart", []CartItem{})

total := hooks.UseMemo(func() interface{} {
    sum := 0.0
    for _, item := range cart() {
        sum += item.Price * float64(item.Quantity)
    }
    return sum
}, []interface{}{cart()}).(float64)
```

## Best Practices

### 1. Use Descriptive Keys

```go
// Good
state.UseAtom("user-preferences", prefs)
state.UseAtom("shopping-cart-items", items)

// Avoid
state.UseAtom("data", something)
state.UseAtom("x", value)
```

### 2. Initialize with Proper Types

```go
// Struct
state.UseAtom("user", User{})

// Slice
state.UseAtom("items", []Item{})

// Map
state.UseAtom("cache", map[string]interface{}{})

// Primitive
state.UseAtom("count", 0)
state.UseAtom("enabled", false)
state.UseAtom("message", "")
```

### 3. Consider Component State First

Use atoms for:

- ✅ State needed by multiple distant components
- ✅ Global application state (theme, user, cart)
- ✅ State that persists across route changes

Use `hooks.UseState` for:

- ✅ Form input values
- ✅ Component-specific UI state (expanded, selected)
- ✅ Temporary local state

## When to Use Atoms vs. Props

| Scenario                         | Use Atoms | Use Props |
| -------------------------------- | --------- | --------- |
| Parent-child communication       | ❌        | ✅        |
| Sibling communication            | ✅        | ❌        |
| Distant components (no parent)   | ✅        | ❌        |
| Global app state                 | ✅        | ❌        |
| Component configuration          | ❌        | ✅        |
| One-time values                  | ❌        | ✅        |
| Frequently changing shared state | ✅        | ❌        |

## Implementation Details

- Atoms are stored in a global registry by key
- Each atom maintains a list of subscribed components
- When an atom updates, all subscribed components are marked for re-render
- Type safety enforced through generics at compile time

## Related Packages

- **[/hooks](../hooks/)** - Provides `UseState` for local state
- **[/render](../render/)** - Handles component re-rendering
- **[/internal/runtime](../internal/runtime/)** - Atom implementation

## Examples

See atom usage in:

- **[/examples/09-atoms](../examples/09-atoms/)** - Complete atom demonstration with multiple components sharing state

## Documentation

See [doc.go](./doc.go) for the official Go package documentation.

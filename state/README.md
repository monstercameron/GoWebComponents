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

### Shared Derived Atoms

Use `UseDerived` when a value should stay shared and recompute from other atoms
instead of living as a component-local memo:

```go
count := state.UseAtom("count", 2)

double := state.UseDerived("double-count", func() int {
    return count.Get() * 2
}, "count")

fmt.Println(double.Get())
```

Derived atom notes:

- Derived atoms are read-only in the current design.
- Dependencies are explicit atom IDs, not inferred at runtime.
- Chained derived atoms are supported.
- Simple dependency cycles are rejected as a safe failure mode.

### Snapshot Persistence

The package now exposes a small snapshot surface for persistence and hot-reload style restore:

```go
snapshot := state.ExportSnapshot()

// Restore later in the same process with exact Go values.
_ = state.ImportSnapshot(snapshot)

// Optional browser storage persistence for JSON-compatible atoms.
_ = state.SaveSnapshot("app-state", snapshot.Select("app-theme", "shopping-cart"), state.LocalStorage)

restored, ok, err := state.LoadSnapshot("app-state", state.LocalStorage)
if err == nil && ok {
    _ = state.ImportSnapshot(restored)
}
```

Serialization constraints:

- `ExportSnapshot` and `ImportSnapshot` preserve exact in-memory Go values in the current process.
- `SaveSnapshot` and `LoadSnapshot` use JSON, so persisted atoms should be JSON-compatible.
- If exact round-tripping of complex structs is required across browser reloads, callers should provide their own typed codec layer before storage.

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
cart := state.UseAtom("shopping-cart", []CartItem{})

total := state.UseComputed(func() float64 {
    sum := 0.0
    for _, item := range cart.Get() {
        sum += item.Price * float64(item.Quantity)
    }
    return sum
}, cart.Get())

fmt.Println(total.Get())
```

Use `UseComputed` for component-local memoized derivation and `UseDerived` when
the derived value itself should behave like shared global state.

### Theme-Derived Labels

Use `UseComputed` when multiple render decisions should flow from a shared atom:

```go
theme := state.UseAtom("app-theme", "light")

themeLabel := state.UseComputed(func() string {
    if theme.Get() == "dark" {
        return "Dark theme active"
    }
    return "Light theme active"
}, theme.Get())

className := state.UseComputed(func() string {
    if theme.Get() == "dark" {
        return "panel panel-dark"
    }
    return "panel panel-light"
}, theme.Get())

fmt.Println(themeLabel.Get())
fmt.Println(className.Get())
```

### Filtered Collections

Use `UseComputed` to keep list filtering logic typed and colocated with the atoms it depends on:

```go
type Todo struct {
    Title     string
    Completed bool
}

todos := state.UseAtom("todos", []Todo{})
showCompleted := state.UseAtom("show-completed", false)

visibleTodos := state.UseComputed(func() []Todo {
    items := todos.Get()
    if showCompleted.Get() {
        return items
    }

    filtered := make([]Todo, 0, len(items))
    for _, todo := range items {
        if !todo.Completed {
            filtered = append(filtered, todo)
        }
    }
    return filtered
}, todos.Get(), showCompleted.Get())

fmt.Println(len(visibleTodos.Get()))
```

### Derived Totals and Summary Text

You can expose both numeric aggregates and render-ready summary strings from the same shared state:

```go
cart := state.UseAtom("shopping-cart", []CartItem{})

total := state.UseComputed(func() float64 {
    sum := 0.0
    for _, item := range cart.Get() {
        sum += item.Price * float64(item.Quantity)
    }
    return sum
}, cart.Get())

summary := state.UseComputed(func() string {
    return fmt.Sprintf("%d items, total $%.2f", len(cart.Get()), total.Get())
}, cart.Get(), total.Get())

fmt.Println(summary.Get())
```

## Best Practices

### Derived State Guidance

- Prefer `UseComputed` for render-only derivation inside a single component.
- Prefer `UseDerived` when multiple components should subscribe to the same derived value.
- Keep derived atom dependency lists explicit and small.
- Avoid long dependency chains when a simpler direct derivation will do.
- Do not model writable state through derived atoms; keep writes on source atoms.

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

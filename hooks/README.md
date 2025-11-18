# Hooks Package

**Location:** `/hooks`

```
GoWebComponents/
├── dom/
├── hooks/            ← YOU ARE HERE
│   ├── doc.go
│   └── hooks.go
├── state/
├── render/
├── router/
├── fetch/
├── internal/
├── examples/
└── ...
```

## Overview

The `hooks` package provides React-style hooks for managing component state, side effects, memoization, and event handlers. Hooks enable functional components to have stateful logic and lifecycle management.

## Core Hooks

### `UseState[T](initialValue T) (func() T, func(T))`

Manages local component state with automatic re-rendering on updates.

```go
count, setCount := hooks.UseState(0)

// Get current value
currentCount := count()

// Update value (triggers re-render)
setCount(currentCount + 1)
```

**Type Inference:** Go automatically infers the type from `initialValue`, so you rarely need to specify `[T]` explicitly.

### `UseEffect(effect func(), deps []interface{})`

Runs side effects after render. Perfect for data fetching, subscriptions, or DOM manipulation.

```go
// Run once on mount
hooks.UseEffect(func() {
    fmt.Println("Component mounted!")
}, nil)

// Run when count changes
hooks.UseEffect(func() {
    fmt.Printf("Count is now: %d\n", count())
}, []interface{}{count()})

// Run on every render (deps omitted)
hooks.UseEffect(func() {
    fmt.Println("Component rendered")
}, nil)
```

**Dependency Rules:**
- `nil` or empty slice: Run once on mount
- Slice with values: Run when any dependency changes
- Omitted: Run after every render

### `UseMemo(compute func() interface{}, deps []interface{}) interface{}`

Memoizes expensive computations, only recalculating when dependencies change.

```go
// Expensive calculation
fibResult := hooks.UseMemo(func() interface{} {
    return calculateFibonacci(n())
}, []interface{}{n()}).(int)

// Type assertion needed since it returns interface{}
result := fibResult
```

### `GoUseFunc(callback interface{}) js.Func`

Wraps Go functions for use as JavaScript event handlers with automatic cleanup.

```go
handleClick := hooks.GoUseFunc(func(event dom.GoEvent) {
    event.PreventDefault()
    setCount(count() + 1)
})

// Use in element
dom.Button(dom.Attrs{
    "onclick": handleClick,
}, dom.Text("Increment"))
```

**Memory Safety:** Automatically manages `js.Func` lifecycle to prevent memory leaks.

## Advanced Usage

### Combining Hooks

```go
func UserProfile(props dom.Attrs) *dom.Element {
    // State management
    user, setUser := hooks.UseState(map[string]interface{}{})
    loading, setLoading := hooks.UseState(true)
    error, setError := hooks.UseState("")

    // Side effect: Fetch user data
    hooks.UseEffect(func() {
        setLoading(true)
        // Fetch logic here
        setLoading(false)
    }, nil)

    // Memoized computation
    displayName := hooks.UseMemo(func() interface{} {
        userData := user()
        return fmt.Sprintf("%s %s", 
            userData["firstName"], 
            userData["lastName"])
    }, []interface{}{user()}).(string)

    // Event handler
    handleRefresh := hooks.GoUseFunc(func(e dom.GoEvent) {
        // Refresh logic
    })

    // Render logic...
}
```

### Custom Hooks Pattern

You can create reusable custom hooks:

```go
func UseCounter(initial int) (func() int, func(), func()) {
    count, setCount := hooks.UseState(initial)

    increment := func() {
        setCount(count() + 1)
    }

    decrement := func() {
        setCount(count() - 1)
    }

    return count, increment, decrement
}

// Usage
func Counter(props dom.Attrs) *dom.Element {
    count, inc, dec := UseCounter(0)
    
    // Use count(), inc(), dec() in your component
}
```

## Hook Rules

1. **Only call hooks at the top level** - Don't call hooks inside loops, conditions, or nested functions
2. **Only call hooks from components** - Hooks must be called from component functions
3. **Consistent hook order** - Always call hooks in the same order on every render

## Event Handler Patterns

### Simple Handler
```go
handleClick := hooks.GoUseFunc(func(e dom.GoEvent) {
    fmt.Println("Clicked!")
})
```

### Input Handler
```go
handleInput := hooks.GoUseFunc(func(e dom.GoEvent) {
    setText(e.GetValue())
})
```

### Form Submit Handler
```go
handleSubmit := hooks.GoUseFunc(func(e dom.GoEvent) {
    e.PreventDefault()
    // Process form
})
```

### Keyboard Handler
```go
handleKeyPress := hooks.GoUseFunc(func(e dom.GoEvent) {
    if e.GetKey() == "Enter" {
        // Handle enter key
    }
})
```

## Related Packages

- **[/dom](../dom/)** - Provides `GoEvent` type and element creation
- **[/state](../state/)** - Global state management with `UseAtom`
- **[/fetch](../fetch/)** - `UseFetch` hook for data fetching
- **[/internal/runtime](../internal/runtime/)** - Hook implementation details

## Examples

See practical hook usage in:
- **[/examples/01-counter](../examples/01-counter/)** - `UseState` basics
- **[/examples/02-text-input](../examples/02-text-input/)** - Input handling
- **[/examples/07-goroutines](../examples/07-goroutines/)** - `UseEffect` with goroutines
- **[/examples/09-atoms](../examples/09-atoms/)** - State management patterns

## Documentation

See [doc.go](./doc.go) for the official Go package documentation.

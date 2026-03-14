# Hooks Package

Location: `hooks/`

The `hooks` package exposes the browser-facing hook API for GoWebComponents. It is a thin layer over the runtime hooks in `internal/runtime/` and is built for `js/wasm`.

## Available Hooks

### `UseState[T](initialValue T) (func() T, func(interface{}))`

Local component state.

- The getter returns the current value.
- The setter accepts either a direct value or a functional update.

```go
count, setCount := hooks.UseState(0)

setCount(1)
setCount(func(prev int) int { return prev + 1 })
```

### `UseEffect(effect func() func(), deps ...interface{})`

Post-render side effects with optional cleanup.

```go
hooks.UseEffect(func() func() {
    fmt.Println("mounted or deps changed")
    return func() {
        fmt.Println("cleanup")
    }
}, userID())
```

### `UseMemo(compute func() interface{}, deps ...interface{}) interface{}`

Memoized computation.

```go
filtered := hooks.UseMemo(func() interface{} {
    return expensiveFilter(items(), query())
}, items(), query()).([]Item)
```

### `UseCallback(fn interface{}, deps ...interface{}) interface{}`

Stable callback identity across renders when dependencies do not change.

### `UseRef(initialValue interface{}) *runtime.RefValue`

Mutable reference that survives renders without triggering rerenders.

### `UseId() string`

Stable unique ID for accessibility wiring.

### `UseFetch(url string, options ...interface{}) (func() runtime.FetchState, func())`

Hook-based fetch state and manual `refetch()` entrypoint.

### `GoUseFunc(fn interface{}) interface{}`

Wraps Go functions for DOM event handlers.

Common supported shapes include:

- `func()`
- `func(string)`
- `func(js.Value)`
- `func(dom.GoEvent)`
- error-returning equivalents for supported signatures

## Hook Rules

- Call hooks at the top level of a component.
- Do not call hooks conditionally.
- Keep hook call order stable between renders.

## Related Packages

- [../dom/README.md](../dom/README.md)
- [../state/README.md](../state/README.md)
- [../fetch/README.md](../fetch/README.md)
- [../internal/README.md](../internal/README.md)

## Examples

- [../examples/01-counter](../examples/01-counter)
- [../examples/02-text-input](../examples/02-text-input)
- [../examples/07-goroutines](../examples/07-goroutines)
- [../examples/09-atoms](../examples/09-atoms)

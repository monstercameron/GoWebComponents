# Counter Example

**Level:** Beginner  
**Concepts:** `ui.UseState`, `ui.UseEvent`, declarative event wiring, rerender on state change

## Current Status

This example is the smallest current interactive app in the integrated example set.

It demonstrates the modern `ui` package hook surface rather than the older `hooks.*` API shape, and it is useful as the first stop when you want to see a minimal browser-mounted GoWebComponents component with local state.

## What This Example Demonstrates

- Using `ui.UseState` for local numeric state
- Using `ui.UseEvent` for typed click handlers without raw `syscall/js` wiring
- Increment, decrement, and reset flows through `count.Update(...)` and `count.Set(...)`
- State-driven rerendering in a minimal component
- A small but intentionally styled integrated example shell

## Key Code

```go
count := ui.UseState(0)
currentCount := count.Get()

increment := ui.UseEvent(func() {
    count.Update(func(prev int) int { return prev + 1 })
})
```

## Learning Objectives

1. Understand how to create and update local state with `ui.UseState`
2. Learn how event handlers are attached through `ui.UseEvent`
3. See how state updates trigger rerendering without manual DOM mutation
4. Use a very small example as the entrypoint into the larger examples catalog

## Running

From the repo root, preferred current paths are:

```powershell
go run ./tools/gwc examples
```

Then open `http://127.0.0.1:8090/examples/public/counter/counter.html`.

## Related Docs

- `examples/README.md`
- `docs/REFERENCE_MANUAL/01-getting-started.md`
- `docs/REFERENCE_MANUAL/02-gwc-workflows.md`

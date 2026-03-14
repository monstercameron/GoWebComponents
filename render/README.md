# Render Package

Location: `render/`

The `render` package bootstraps the browser runtime and mounts elements into the DOM. It is built for `js/wasm`.

## Primary Entry Points

### `To(element *runtime.Element, selector string)`

Mounts an element into the first DOM node that matches a CSS selector.

```go
render.To(dom.CreateElement(App, nil), "#app")
```

### `ToElement(element *runtime.Element, domElement js.Value)`

Mounts an element into a specific DOM element you already looked up through `syscall/js`.

```go
container := js.Global().Get("document").Call("getElementById", "app")
render.ToElement(dom.CreateElement(App, nil), container)
```

## What It Does

- initializes the global runtime with the wasm DOM, event, scheduler, and browser-state adapters
- renders the initial element tree
- routes later state/atom/effect-driven updates through the runtime

The runtime itself lives in `internal/runtime/`. The `render` package is the browser-facing mount layer.

## Related Packages

- [../dom/README.md](../dom/README.md)
- [../hooks/README.md](../hooks/README.md)
- [../state/README.md](../state/README.md)
- [../internal/README.md](../internal/README.md)

# HTML Sugar

This document defines the current additive authoring sugar that lives in the `html` package.

## Scope

- The sugar layer lives in `html`, not `ui`.
- The explicit typed builders such as `html.Div(html.Props{}, ...)` remain the stable default host-element API.
- The sugar layer is additive and mechanical. It does not introduce JSX, hidden reactivity, or a second rendering model.
- The sugar layer is intended for primitive DOM authoring ergonomics. It is not a replacement for explicit typed props on business components.

## Current shipped helpers

- Mixed child normalization: `Children(...)`
- Companion mixed-argument host tags in `html/shorthand`, including `Div(...)`, `Span(...)`, `Button(...)`, `Input(...)`, `Label(...)`, `Form(...)`, `P(...)`, `Pre(...)`, `Code(...)`, `Section(...)`, `Ul(...)`, `Li(...)`, `H1(...)`, `H2(...)`, `H3(...)`, `Img(...)`, `Select(...)`, and `Option(...)`
- Text helpers: `Text(...)`, `Textf(...)`, `TextIf(...)`
- Class helpers: `When(...)`, `ClassNames(...)`
- Conditional node helpers: `If(...)`, `IfElse(...)`, `Unless(...)`
- Collection helpers: `Map(...)`, `MapKeyed(...)`, `FlatMap(...)`, `FilterMap(...)`, `Join(...)`
- Optional helpers: `Maybe(...)`, `OrElse(...)`, `Coalesce(...)`
- Switch helpers: `Switch(...)`, `Case(...)`, `Default(...)`
- Option model: `PropsOf(...)`, `WithProps(...)`, plus option helpers such as `Class(...)`, `ID(...)`, `OnClick(...)`, `Attr(...)`, `Data(...)`, and `Aria(...)`
- Event wrappers: `Prevent(...)` and `Stop(...)`
- Temporal event wrappers: `Debounce(...)` and `Throttle(...)`

## Child contract

`Children(...)` accepts these inputs:

- `ui.Node`
- `string`
- `fmt.Stringer`
- `func() string`
- nested `[]ui.Node`
- nested `[]string`
- nested `[]interface{}`
- other slice or array values that recursively contain supported child forms

Behavior:

- `nil` children are skipped
- scalar fallbacks are stringified with `fmt.Sprint(...)`
- order is preserved
- normalized strings and string-like values become text nodes

## Conditional semantics

- `If(false, ...)` returns `nil`
- `Unless(true, ...)` returns `nil`
- `IfElse(...)` returns one of the provided nodes directly
- `TextIf(false, ...)` returns `nil`
- `Maybe(nil, ...)` returns `nil`

No conditional helper inserts an empty fragment automatically.

## Option model

`PropsOf(...)` applies `PropOption` values from left to right.

- duplicate scalar options use last-write-wins semantics
- repeated `Style(...)`, `Dataset(...)`, `AriaSet(...)`, and `Attrs(...)` calls merge maps and later keys overwrite earlier keys
- typed helpers still win by explicit call order because they all compile down through the same `PropsOf(...)` path

Current first-pass option helpers include:

- string props: `ID`, `Class`, `Title`, `Value`, `Placeholder`, `Type`, `Href`, `Src`, `Role`
- form and linkage props: `For`, `Name`, `Rows`
- boolean props: `Disabled`, `Checked`, `Selected`, `Required`, `ReadOnly`, `AutoFocus`
- conditional boolean props: `DisabledIf`, `ReadOnlyIf`, `SelectedIf`
- structured props: `Style`, `Data`, `Dataset`, `Aria`, `AriaSet`, `Attr`, `Attrs`, `TabIndex`
- event props: `OnClick`, `OnInput`, `OnChange`, `OnSubmit`, `OnKeyDown`, `OnKeyUp`, `OnFocus`, `OnBlur`

Event options accept either an existing `ui.Handler` or a callback supported by `ui.UseEvent(...)`, including zero-argument callbacks and typed event callbacks.

`Prevent(...)` and `Stop(...)` wrap either zero-argument callbacks or typed-event callbacks and preserve ordinary callback execution while adding explicit browser-control behavior before invoking the wrapped callback. Wrappers compose from the outside in, so `Prevent(Stop(fn))` prevents default first, then stops propagation, then invokes `fn`.

## Second-pass additions

- `html/shorthand` ships mixed variadic host tags without changing the existing `html.Div(...)`-style typed builder signatures.
- `FromProps(...)` in `html/shorthand` injects a full props struct into a mixed argument list and preserves explicit zero values before later shorthand options are applied.
- `MapKeyed(...)` applies reconciliation keys explicitly by writing the computed key onto each realized node.
- `FlatMap(...)` flattens `[]ui.Node` results from each item in order.
- `FilterMap(...)` keeps only nodes whose render callback reports `true`.
- `Join(...)` inserts separators only between realized nodes and skips nil inputs.
- `Maybe(...)`, `OrElse(...)`, and `Coalesce(...)` are pointer-based in order to keep absence semantics explicit and avoid conflating legitimate zero values with missing data.
- `Switch(...)` matches branches in order using `reflect.DeepEqual(...)`; the first matching `Case(...)` wins and the first `Default(...)` is used as fallback.
- `Debounce(...)` waits for a quiet period before invoking the wrapped callback.
- `Throttle(...)` invokes immediately and then delivers at most one trailing invocation with the latest pending event per interval.

## Decisions for first pass

- `TextIf(...)` is included because it composes directly with the shipped text normalization path.
- mixed variadic host-element wrappers ship in the companion `html/shorthand` package rather than changing the existing typed builder signatures to `...interface{}` and breaking common `[]ui.Node` expansion callsites
- automatic string-to-text lifting still remains out of the existing typed builders for compatibility reasons; the non-breaking mixed-input path is `html/shorthand` plus the stable explicit `html` builders
- `ClassIf(...)` is not included because `When(...)` already covers the same readability case.
- `ClassNames(...)` recursively flattens nested class-part slices.
- `Map(...)` is intentionally `func(T) ui.Node` only in the first pass.
- `MapKeyed(...)` is explicit rather than magical: callers provide the key function and the sugar layer writes the resulting key onto each realized node.
- `Fragment(...)` remains exposed through `html.Fragment(...)`; no short alias such as `F` is added.
- `Switch(...)`, `Case(...)`, and `Default(...)` are part of the reopened second pass and use first-match-wins semantics with an optional `Default(...)` fallback.
- `Maybe(...)`, `OrElse(...)`, and `Coalesce(...)` are part of the reopened second pass and stay pointer-based rather than inventing zero-value absence rules.
- `StyleMap(...)` is not added because `Style(...)` already covers the intended map-based case.
- positional anchor sugar such as `A("/settings", "Settings")` is not added in the first pass
- button and link content use the general child normalization path rather than a dedicated primitive-specific shortcut
- primitive DOM helpers may use option-style sugar; business components should continue to use explicit typed props structs
- router-aware, resource-aware, and binding-style sugar stays outside the generic html sugar surface
- temporal event helpers are part of the reopened second pass and intentionally stay closure-scoped rather than adding separate component lifecycle ownership
- primitive visual options such as variant and size stay outside the first-pass html sugar surface
- app-specific conveniences such as pluralization, truncation, and test-id helpers stay out of the first pass

## Import guidance

Preferred import:

```go
import html "github.com/monstercameron/GoWebComponents/html"
```

Second-pass companion import when mixed argument host tags are the goal:

```go
import h "github.com/monstercameron/GoWebComponents/html/shorthand"
```

Dot-importing `html` is acceptable only in very small example packages where identifier collisions are intentionally controlled.
Dot-importing `html/shorthand` is also acceptable only in tiny example packages; in larger packages prefer a short alias such as `h` so the narrower surface remains obvious in review.

## Migration guidance

Teams can adopt the sugar incrementally:

- keep existing `html.Div(html.Props{}, ...)` and other typed builders
- adopt `html/shorthand` only when a component materially benefits from one-call mixed props plus children such as `h.Div(h.Class("panel"), "hello")`
- see `examples/03-toggle` for a compact event-driven mixed-argument `html/shorthand` component path and `examples/02-text-input` for a form-heavy input path
- use `PropsOf(...)` where repetitive props literals are noisy
- use `Children(...)` only where mixed strings, nodes, or nested slices materially reduce boilerplate
- continue to keep business-component props explicit and typed

The current sugar layer is designed to coexist with the explicit builder model rather than replace it.
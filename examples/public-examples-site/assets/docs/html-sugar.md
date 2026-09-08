# HTML Sugar

This document defines the current additive authoring sugar that spans the core `html` package and the companion `html/shorthand` package.

## At A Glance

- Keep `html.Div(html.Props{}, ...)` and the other typed builders as the stable default host-element API.
- Use `html` helpers such as `Children(...)`, `PropsOf(...)`, `ClassNames(...)`, `If(...)`, `Map(...)`, and related utilities when they reduce boilerplate without changing the authoring model.
- Use `html/shorthand` when one-call mixed props plus children materially improves readability for primitive DOM authoring.
- Keep business-component props explicit and typed; the sugar layer is for host-element ergonomics, not for replacing component contracts.

## Quick Authoring Chooser

Use this rule of thumb:

- choose typed `html.*` builders when explicit prop structs make the callsite clearer or when you already have a concrete `html.Props` value
- choose `html/shorthand` when the element reads better as one mixed argument list of options and children
- choose `html.Children(...)`, `html.Map(...)`, `html.If(...)`, and related helpers when you want less boilerplate without changing tag style
- avoid introducing sugar that hides routing, data loading, or business-specific conventions behind generic DOM helpers

## Scope

- The sugar layer lives in `html`, not `ui`.
- The explicit typed builders such as `html.Div(html.Props{}, ...)` remain the stable default host-element API.
- The sugar layer is additive and mechanical. It does not introduce JSX, hidden reactivity, or a second rendering model.
- The sugar layer is intended for primitive DOM authoring ergonomics. It is not a replacement for explicit typed props on business components.

## Example Shape

The intended split is straightforward: keep explicit typed builders available, and reach for `html/shorthand` only when the mixed argument list makes the markup easier to scan.

```go
import (
	h "github.com/monstercameron/GoWebComponents/v6/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

func SearchPanel(query string, pending bool, onInput func(ui.InputEvent)) ui.Node {
	return h.Section(
		h.Class("panel"),
		h.H2("Search"),
		h.Input(
			h.Type("search"),
			h.Value(query),
			h.OnInput(onInput),
			h.Placeholder("Filter docs"),
		),
		h.P(
			h.Class("muted"),
			h.IfElse(pending, h.Text("Updating..."), h.Textf("%d characters", len(query))),
		),
	)
}
```

This is still ordinary Go plus explicit helpers. The sugar is additive; it does not create a second component model.

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

Preferred base import:

```go
import html "github.com/monstercameron/GoWebComponents/v6/html"
```

Preferred companion import when mixed argument host tags are the goal:

```go
import h "github.com/monstercameron/GoWebComponents/v6/html/shorthand"
```

Dot-importing `html` is acceptable only in very small example packages where identifier collisions are intentionally controlled.
Dot-importing `html/shorthand` is also acceptable only in tiny example packages; in larger packages prefer a short alias such as `h` so the narrower surface remains obvious in review.

The repo root README now uses dot-imported shorthand for the smallest starter example because it optimizes for legibility in a minimal snippet. That should not be treated as the default style for larger packages.

## Migration guidance

Teams can adopt the sugar incrementally:

- keep existing `html.Div(html.Props{}, ...)` and other typed builders
- adopt `html/shorthand` only when a component materially benefits from one-call mixed props plus children such as `h.Div(h.Class("panel"), "hello")`
- see `examples/public/toggle` for a compact event-driven mixed-argument `html/shorthand` component path and `examples/public/text-input` for a form-heavy input path
- use `PropsOf(...)` where repetitive props literals are noisy
- use `Children(...)` only where mixed strings, nodes, or nested slices materially reduce boilerplate
- continue to keep business-component props explicit and typed

The current sugar layer is designed to coexist with the explicit builder model rather than replace it.

## Recent additions (v3.3–v3.4)

- **`MapKeyedComponent(items, key, render)`** renders each list item as its own keyed component, so the
  `render` func may call hooks and `On*` handlers directly — no hand-extracted row component, no
  "hooks in a variable-length loop" gotcha. Per-row state is isolated and follows the key across
  reorders and removals.
- **SVG chart primitives** join `Svg`/`Path`/`Circle`/`Rect`: `G`, `Line`, `Polyline`, `Polygon`,
  `Ellipse`, `TSpan`, `Defs`, `Use`, `LinearGradient`, `RadialGradient`, `GradientStop`, `ClipPath`,
  `Mask`, `SvgPattern`, `SvgImage`, `ForeignObject`, `Symbol`, `Marker` — charts as real Go nodes (no
  JS shim). Use `RawHTMLUnsafe` for the SVG `<text>` element.
- **Typed CSS tokens, layers, and reactive theming.** Author a `:root` palette with `css.Root` /
  `css.Theme.RootRules` (`--color-*`/`--space-*`/`--text-*`/`--radius-*`), reference tokens with
  `css.Var`, scope overrides with `css.DataTheme` / `css.Layer`, and switch live with the reactive
  `ui.UseTheme` / `ui.SetTheme` hooks. See reference-manual chapters 04 (hooks) and 05 (HTML/CSS
  authoring) for the full surface.

## Review Checklist

- does the example or doc keep the boundary clear between typed `html` builders and the mixed-argument `html/shorthand` companion surface
- are sugar helpers improving readability rather than hiding business logic or creating a second rendering model
- are business-component props still explicit structs instead of ad hoc mixed argument lists
- is import style chosen intentionally for the package size, with aliases preferred over dot imports in non-trivial packages
- do examples show current public helpers rather than deprecated or hypothetical sugar
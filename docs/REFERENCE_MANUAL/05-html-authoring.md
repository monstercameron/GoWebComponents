# 05 HTML Authoring

Use this chapter when you are authoring DOM trees with `html` or `html/shorthand`.

It is the right chapter for:

- choosing between typed builders and shorthand helpers
- composing `html.Props` and prop options cleanly
- building semantic layout and typed forms
- using generic tags, markdown helpers, or browser custom elements

Use another chapter instead when:

- you need local hook patterns or rendering flow: go to [04 UI Rendering And Hooks](04-ui-rendering-and-hooks.md)
- you need form ownership and validation flow in depth: go to [11 Forms Accessibility And I18n](11-forms-accessibility-and-i18n.md)
- you need browser interop for custom events or custom-element event wiring: go to [10 Browser Interop And Workers](10-browser-interop-and-workers.md)

## Overview

The `html` layer is additive, explicit, and Go-first.

Use it to:

- build DOM with typed element constructors such as `html.Div`, `html.Input`, and `html.Tag`
- apply props through `html.Props` or `html.PropsOf(...)`
- use ergonomic sugar through `html/shorthand` when mixed arguments read better
- render semantic sectioning markup explicitly
- integrate browser-defined custom elements without dropping into ad hoc string templates

The key split is:

- `html`: stable typed builders and explicit props
- `html/shorthand`: additive mixed-argument host-tag sugar

## Stability Note

Practical defaults:

- typed `html.*` builders are the stable default host-element API
- `html/shorthand` is additive sugar and should be used when it improves readability, not because it changes the rendering model
- generic DOM authoring helpers such as `Textf`, `If`, `When`, `PropsOf`, and `ClassNames` are useful, but some sugar-layer helpers are still treated as more flexible and evolving than the core typed builders
- custom-element consumption is supported; export-side custom-element wrapping remains experimental

## Minimal Example

If you want the safest default style, start with typed builders and explicit props.

```go
package main

import (
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

// renderProfileCard renders a small DOM tree with typed builders and explicit props.
func renderProfileCard() ui.Node {
	return html.Main(html.Props{Class: "mx-auto max-w-xl space-y-4 p-6"},
		html.H1(html.Props{}, html.Text("Typed html builders")),
		html.P(html.Props{Class: "text-slate-600"}, html.Text("Start here when explicit props make the callsite clearer.")),
		html.Button(html.Props{Type: "button", Class: "rounded-xl border px-4 py-2"}, html.Text("Continue")),
	)
}

// main mounts the typed-html example into the browser DOM.
func main() {
	ui.Render(ui.CreateElement(renderProfileCard, nil), "#app")
	utils.WaitForever()
}
```

Why this is the safest starting point:

- every element is explicit
- props stay typed instead of stringly-typed
- the callsite reads like ordinary Go, not a second templating language

## Production-Shaped Example

When callsites become noisy, use `PropsOf(...)` and the shorthand package deliberately, but keep business-component props explicit.

```go
package formview

import (
	h "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/ui"
)

type teamFormProps struct {
	Name          string
	Team          string
	HandleUserName ui.Handler
	HandleUserTeam ui.Handler
}

// renderTeamForm renders a production-shaped host tree with shorthand host tags and explicit component props.
func renderTeamForm(getProps teamFormProps) ui.Node {
	return h.Section(
		h.Class("space-y-4 rounded-2xl border border-slate-200 bg-white p-5"),
		h.H2("Team settings"),
		h.Div(
			h.Label(h.For("team-name"), "Name"),
			h.Input(
				h.ID("team-name"),
				h.Value(getProps.Name),
				h.OnInput(getProps.HandleUserName),
				h.Placeholder("Cam"),
			),
		),
		h.Div(
			h.Label(h.For("team-role"), "Team"),
			h.Select(
				h.ID("team-role"),
				h.Value(getProps.Team),
				h.OnChange(getProps.HandleUserTeam),
				h.Option(h.Value("platform"), "Platform"),
				h.Option(h.Value("design"), "Design"),
				h.Option(h.Value("ops"), "Operations"),
			),
		),
	)
}
```

Why this is the better midpoint:

- host markup gets terser
- component props stay typed and obvious
- you avoid pretending shorthand is a replacement for explicit business-component contracts

## Scale-Up Example

In a larger codebase, pick one host-authoring style per package and centralize repeated visual prop combinations behind helper builders, not raw prop maps scattered everywhere.

```go
package sharedview

import (
	"github.com/monstercameron/GoWebComponents/html"
)

// BuildPanelProps returns one shared panel treatment for host containers.
func BuildPanelProps() html.Props {
	return html.Props{
		Class: "rounded-2xl border border-slate-200 bg-white p-5 shadow-sm",
	}
}

// BuildFieldLabelProps returns one reusable label style for form-heavy packages.
func BuildFieldLabelProps() html.Props {
	return html.Props{
		Class: "block text-sm font-semibold text-slate-700",
	}
}
```

```go
package accountview

import (
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"

	"my-app/internal/sharedview"
)

// RenderAccountSection composes shared host props without inventing a second component prop model.
func RenderAccountSection() ui.Node {
	return html.Section(sharedview.BuildPanelProps(),
		html.Label(sharedview.BuildFieldLabelProps(), html.Text("Account name")),
		html.Input(html.Props{Type: "text", Placeholder: "Acme"}),
	)
}
```

Why this scales:

- packages stay consistent
- repeated host styles stop leaking as copy-pasted strings
- component props remain explicit while host-element styling gets reusable

## API Family Reference

Use this table before choosing a style.

| API family | Representative APIs | Stability | Use it when | Prefer something else when |
| --- | --- | --- | --- | --- |
| Typed host builders | `Div`, `Section`, `Main`, `Input`, `Button`, `Select`, `Textarea` in `html` | `Stable` default | explicit props make the callsite clearer | the callsite becomes repetitive and shorthand is materially easier to scan |
| Prop composition | `Props`, `PropsOf`, `WithProps`, `Class`, `ID`, `Data`, `Aria`, `Style` | core prop model is stable | you need typed, composable props | you are trying to hide business logic behind prop helpers |
| Shorthand host tags | `Div`, `Button`, `Input`, `Label`, `Select` in `html/shorthand` | additive sugar | mixed props plus children read better | the package is already clearer with explicit `html.Props` |
| Conditional and collection helpers | `If`, `IfElse`, `Unless`, `Map`, `MapKeyed`, `FlatMap`, `FilterMap`, `Join`, `Maybe`, `Switch` | additive sugar | you want less boilerplate for host-tree assembly | ordinary Go control flow is clearer |
| Text helpers | `Text`, `Textf`, `TextIf` | additive and practical | you want explicit text-node construction and formatting | you are building a higher-level formatting abstraction |
| Semantic structure | `Header`, `Nav`, `Main`, `Article`, `Aside`, `Footer`, `Section` | `Stable` host surface | document structure and accessibility semantics matter | a generic `Div` tree is genuinely enough |
| Generic tags | `Tag` | `Stable` escape hatch | you need an uncommon or custom tag with plain attrs and children | the element needs property assignment or custom-event integration |
| Custom-element host | `CustomElement` | supported consumption surface | a browser-defined custom element needs attributes, properties, slots, or typed event integration | the element is simple enough for `Tag` |
| Markdown helpers | `RenderMarkdown`, `ResolveMarkdownHref` | specialized public helper | you need markdown rendered into the current tree | the content is already structured Go markup |

## Semantic Layout Guidance

Prefer semantic wrappers when the page structure matters to users, accessibility tools, or future maintainers.

Good uses:

- `Header` for global or section introductions
- `Nav` for real navigation groups
- `Main` for the primary page body
- `Article` for standalone content blocks
- `Aside` for supporting context
- `Footer` for closing metadata and related links

Do not turn this into ceremony:

- if the subtree is purely structural glue inside a small feature, `Div` is still valid
- use semantic wrappers where they communicate meaning, not to satisfy a document-style quota

## Forms And Typed Props

The `html` surface is especially useful for forms because the prop model stays typed.

Use it when you need:

- `Value`, `OnInput`, `OnChange`, `Rows`, `Placeholder`
- typed checkbox, select, and textarea markup
- labels and `For(...)` wiring that stay explicit

Practical rule:

- use `html` or shorthand for the markup layer
- keep form state and validation ownership in `ui.UseForm[T]` or the server-owned form path

That keeps markup and workflow concerns separate.

## Generic Tags And Custom Elements

Use `html.Tag(...)` when:

- the tag only needs normal attrs and children
- you want an uncommon standard tag or a simple custom host

Use `html.CustomElement(...)` when:

- the browser-defined element needs property assignment
- the element uses reflected attributes plus client-only properties
- you need slot wiring or custom-event integration

Keep the boundary clear:

- `Tag` is the plain escape hatch
- `CustomElement` is the stable consumption path for browser-defined custom elements
- export-side custom-element wrapping is still experimental

## Markdown Guidance

Use `RenderMarkdown(...)` when markdown is a real content source and it is useful to keep the rendering path inside the current tree composition model.

Do not use it as a default content strategy when:

- the page is mostly application UI, not authored content
- structured Go markup is already clearer than markdown transformation

## Typed CSS

Styling is authored with the **typed CSS** layer instead of free-form class strings. The `css` package
is the raw layer (typed values, properties, variants, and SCSS-style selector composition); `css/u` is
a Tailwind-shaped utility layer built on top of it. Both are designed to be dot-imported so styles read
like bare utilities next to `html/shorthand` elements.

```go
import (
    "github.com/monstercameron/GoWebComponents/css"
    . "github.com/monstercameron/GoWebComponents/css/u"
    . "github.com/monstercameron/GoWebComponents/html/shorthand"
)

func Badge() ui.Node {
    return Span(
        css.Class(Flex, ItemsCenter, Pad(Spacing2), Rounded(RadiusXl), Bg(Hex("#0ea5e9")), Fg(White),
            Hover(Bg(Hex("#0284c7"))),
        ),
        Text("live"),
    )
}
```

Key properties:

- **Type-checked values.** `Hex`, `RGBA`, `Spacing`, `Rem`, and friends are constructors, so a typo is a
  compile error rather than a silently-dead rule. Values like `Hex` are validated by construction.
- **Content-hashed, deduped classes.** Each folded rule becomes a deterministic class keyed by its
  content; identical rules collapse to one class, and the stylesheet is injected once. The same source
  always produces the same class names, which keeps builds reproducible.
- **`css.Class(...)` is the clsx-style entry point.** It accepts a mix of literal strings, typed rules,
  variant slices, and pre-folded sheets in a single call.
- **Variants and selectors.** Pseudo-classes and composed selectors (`Hover(...)`, `Active(...)`,
  `Focus(...)`, and `&`-template selectors in the raw layer) are first-class, so you rarely need a hand-
  written stylesheet.
- **SSR + hydration.** Server-rendered rules are emitted into the SSR buffer; on boot the client calls
  `css.SeedFromDocument()` so those rules are adopted rather than re-injected.
- **Safe output.** Emitted CSS is hardened against `</style>`/`<script>` breakout, so interpolated
  values cannot escape the stylesheet.

See the [typed-css](../../examples/public/typed-css/) example for a fully bare-authored view.

### Global rules, design tokens, and cascade layers (v3.3+)

The hashed-class model above scopes every rule under a generated class. For the *global* stylesheet
layer — element selectors, a `:root` token palette, semantic classes, and cascade ordering — use:

- **`css.Global(selector, rules...)`** — emit an un-prefixed top-level rule (`body`, `*`, `.nav-link`,
  `:root`). **`css.Root(rules...)`** is shorthand for `Global(":root", ...)`.
- **Design tokens.** Author a `:root` custom-property palette and reference it with `css.Var`:
  ```go
  css.Root(css.Raw("--accent", "#4f46e5"), css.Raw("--radius", "12px"))
  css.New(css.Bg(css.Var("accent")), css.Raw("border-radius", string(css.Var("radius"))))
  ```
  A runtime `element.style.setProperty("--accent", …)` then reskins every reference without
  regenerating classes. **`css.Theme.RootRules()` / `css.EmitThemeTokens(theme)`** emit a typed
  `Theme`'s scales (`--color-*`, `--space-*`, `--text-*`, `--radius-*`) as that palette automatically.
- **Ancestor-attribute variants.** `css.Within(selector, rules...)` → `<selector> &`; `css.DataTheme("light", rules...)`
  → `[data-theme="light"] &` (pairs with `ui.UseTheme`, see [04 UI Rendering And Hooks](04-ui-rendering-and-hooks.md)).
- **Cascade layers.** `css.Layer(name, rules...)`, `css.LayerGlobal(name, selector, rules...)`, and
  `css.DeclareLayers("base","components","overrides")` give declared override precedence via `@layer`
  instead of order/`!important` accidents.
- **Base + extraction.** `css.Preflight()` / `css.PreflightInLayer(name)` emit an opt-in modern reset;
  `css.CriticalCSS()` returns the `<style>` block to inline for SSR (paired with `SeedFromDocument`).

All global/token/layer output is hardened against `</style>` breakout like the rest of `css`.

## Markup Nodes (Raw HTML & SVG)

When you need to render a fragment of existing markup — sanitized rich text from a CMS, or a trusted
inline SVG — use the markup-node helpers in `html`. They **parse markup into real DOM nodes** and commit
them through the normal reconciler; the framework never assigns `innerHTML`.

```go
// Sanitized: the <b> survives, an embedded <script> is stripped.
nodes := html.RawHTML(`<p>hello <b>world</b><script>steal()</script></p>`)

// Trusted: inline SVG kept as real, correctly-namespaced nodes.
svg := html.RawHTMLUnsafe(`<svg width="20" height="20"><circle cx="10" cy="10" r="5"/></svg>`)
```

- `RawHTML` runs the fragment through the `sanitize` allowlist before producing nodes — use it for any
  untrusted or semi-trusted content.
- `RawHTMLUnsafe` skips sanitization for content you fully control; it still parses to real nodes (no
  `innerHTML`), and SVG is committed in the SVG namespace.

See the [raw-html](../../examples/public/raw-html/) example.

### Typed SVG elements (v3.3+)

Beyond `Svg`/`Path`/`Circle`/`Rect`, `html/shorthand` provides the SVG chart primitives so charts can
be authored as real Go nodes (no JS shim): `G`, `Line`, `Polyline`, `Polygon`, `Ellipse`, `TSpan`,
`Defs`, `Use`, `LinearGradient`, `RadialGradient`, `GradientStop`, `ClipPath`, `Mask`, `SvgPattern`,
`SvgImage`, `ForeignObject`, `Symbol`, `Marker`. All commit in the SVG namespace. (The SVG `<text>`
element is intentionally not a helper — use `RawHTMLUnsafe` for it to avoid HTML/SVG name collisions.)

### Interactive lists: `MapKeyedComponent` (v3.3+)

`On*` handlers and hooks must sit at stable hook positions, so they cannot be called directly inside a
variable-length `Map`/`MapKeyed` loop. **`shorthand.MapKeyedComponent(items, key, render)`** renders
each item as its own keyed component (its own fiber), so the `render` func may legally use `UseState`,
`UseEvent`, and `On*` handlers per row. Per-row state is isolated and follows the key across reorders
and removals.

```go
MapKeyedComponent(rows, func(r Row) any { return r.ID }, func(r Row) ui.Node {
    open := ui.UseState(false)                       // legal: each row is its own component
    return Button(OnClick(func() { open.Set(!open.Get()) }), r.Name)
})
```

## Design Notes And Boundaries

Keep these rules in mind when authoring with `html`:

- typed builders are the stable default host-element API
- shorthand is additive ergonomics, not a second rendering model
- business-component props should stay explicit and typed
- host-layer sugar should not hide routing, data ownership, or business workflow
- custom-element consumption is supported; custom-element export remains experimental
- import style should be chosen intentionally; aliases are safer than dot imports in non-trivial packages

## Common Failure Modes

- mixing typed builders and shorthand randomly inside the same package with no style discipline
- using shorthand as an excuse to hide business-component contracts
- reaching for `Tag` when the real need is `CustomElement` with explicit property mapping
- turning prop helpers into a dumping ground for domain logic
- using markdown helpers where structured Go markup would be clearer and safer
- relying on server markup to carry custom-element client-only property state

## Validation

Use the smallest focused examples for the HTML authoring surface you are adopting.

Semantic and typed host structure:

```powershell
go run ./tools/gwc dev -app .\examples\public\semantic-html\main.go
go run ./tools/gwc dev -app .\examples\public\html-forms\main.go
```

Generic tags and escape hatches:

```powershell
go run ./tools/gwc dev -app .\examples\public\html-tag\main.go
```

Browser custom-element consumption:

```powershell
go run ./tools/gwc dev -app .\examples\public\web-components\main.go
```

## Topic Pagination
Topic 5 of 16. Use previous and next to move through the ordered manual chapters; the first and last topics wrap.
- Previous topic: [04 UI Rendering And Hooks](04-ui-rendering-and-hooks.md)
- Topic index: [Reference Manual](README.md)
- Next topic: [06 State And Reactivity](06-state-and-reactivity.md)

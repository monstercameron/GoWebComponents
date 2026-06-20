# F3 Design — `css` typed CSS package

Typed, type-safe CSS for GWC where the raw-CSS layer is the foundation and Tailwind-shaped utilities are
a typed convenience on top. No external Tailwind build step. Existing class strings keep working during
migration. **Emission v1 = runtime `<style>` injection.** New **top-level `css/`** package; `css` imports
`html` (never the reverse — no cycle, since `html` only imports `ui`/`runtime`).

Status: **design** (Feature 3 of `sqlite-state-typedcss.md`). Independent of F1/F2. Implements into an
`examples/public/` demo + tests in both lanes.

---

## The interop contract (settled — driven by a language constraint)

`html.PropOption` is `interface { apply(*Props) }` with an **unexported** method (`html/sugar.go:288`).
A type declared in package `css` therefore **cannot** satisfy `html.PropOption`. That rules out "the css
value *is* a PropOption" and decides the idiomatic crossing: `css` delegates to the existing `html.Class`
setter. **Zero edits to `html` or `shorthand`.**

Two entry points, one per existing call-site contract:

```go
// Raw class name(s). Satisfies fmt.Stringer, so it already flows through
// html.ClassNames(...) / appendClassFragments (sugar.go:972), ClassMap keys, SSR, attr strings.
type Sheet string
func (s Sheet) String() string { return string(s) }
func New(rules ...Rule) Sheet                          // register + hash -> class name(s)

// PropOption for Div(...)/shorthand arg lists. Delegates to the existing html.Class setter,
// which splitArgs already accepts (shorthand.go:798). No plumbing anywhere.
func Class(rules ...Rule) html.PropOption {
    return html.Class(New(rules...).String())
}
```

Reads as a clean parallel to today — typing lives in the *builder*, exactly like typed `PropOption`s put
it in the attribute builders:

```go
// today
Div(html.Class("flex gap-2 hover:bg-slate-900"), …)
// typed, same shape
Div(css.Class(css.Display.Flex, css.Gap(css.Px(8)), css.Hover(css.Bg(css.Slate900))), …)
```

`html.Class` (literal strings) and `css.Class` (typed rules) sit side by side. Migration is per-call-site:
a literal stays a literal until someone rewrites it as `css.*`; both compile to the same `Props.Class`.

---

## Package & file layout

```
css/
  doc.go            // package godoc + authoring examples
  css.go            // Sheet, New, Class; the public crossing into html
  rule.go           // Rule type + composition (the core value model)
  prop.go           // typed properties: Display, Position, … (typed enums)
  value.go          // typed values & units: Px, Rem, Percent, color tokens
  variant.go        // Hover, Focus, Media, Keyframes, DefineVariant
  registry.go       // process-wide hash->class dedup registry + the Sink seam
  sink_wasm.go      //go:build js && wasm     — runtime <style> injection sink
  sink_native.go    //go:build !(js && wasm)  — buffer sink (SSR + tests read it back)
  theme.go          // Layer 2 — Theme struct, DefaultTheme, UseTheme (the tailwind.config analog)
  u/                // Layer 2 — typed-native Tailwind engine (full-parity, generated)
    spec.go         // machine-readable utility-family spec table (property + theme scale + flags)
    variant.go      // hand-written variant wrappers backbone (Hover/Md/Dark/Group/Peer …)
    util_gen.go     // GENERATED typed utilities from spec.go + Theme (checked in)
    util.go         // hand-written helpers + DefineUtility surface
  css_test.go       // native lane: hashing/dedup/serialization determinism
  css_wasm_test.go  // wasm lane: injection + SSR-harvest + hydration suppression

tools/gwc/         // existing toolchain
  cssgen.go         // native-lane generator: spec table + Theme -> css/u/util_gen.go
```

Module path: `github.com/monstercameron/GoWebComponents/css`. House style: `parse`-prefixed params,
`Use*` for any component-bound hook, dual-build `*_wasm.go` / `*_native.go`.

---

## Layer 1 — typed raw CSS (foundation)

### Rule (the core value model)
A `Rule` is one typed declaration or a variant-wrapped group. `New(rules...)` folds them into a single
hashed class.

```go
type Rule struct { /* opaque: property, value, and optional variant scope */ }

// Properties are typed namespaces of typed values:
var Display = displayProp{}           // css.Display.Flex, css.Display.Grid, css.Display.None
func Gap(v Length) Rule                // css.Gap(css.Px(8))
func Bg(c Color) Rule                  // css.Bg(css.Slate900)

// Values carry units in the type, not the string:
type Length string
func Px(n int) Length                  // "8px"
func Rem(n float64) Length             // "0.5rem"
type Color string                      // tokens: css.Slate900, css.Transparent, css.Hex("#0af")
```

Emits **real scoped CSS** (hashed class + stylesheet rule), so it expresses what inline `Style` can't:
`:hover`, media, keyframes. Inline `Style(map[string]string)` (sugar.go:605) stays as-is for one-off
dynamic styling; `css.*` is the path whenever a pseudo/media/keyframe or dedup is wanted.

### Variants
`css.Hover(rules...)`, `css.Focus(...)`, `css.Media(css.MinW(768), rules...)`, `css.Keyframes(...)` wrap
inner rules in a selector/at-rule scope. Open for extension via `css.DefineVariant("group-hover", …)`.

---

## Registry, dedup & emission (the perf-correctness core)

`New(...)` is called inside render loops, so it must be cheap and idempotent:

1. Serialize the rule-set deterministically (sorted properties, canonical value strings).
2. Content-hash → short stable class name (e.g. `c-1a2b3c`).
3. Look up the hash in a **process-wide registry**. Hit → return the existing class, **no re-emit**.
   Miss → register the compiled CSS text and push it to the active `Sink`.

```go
type Sink interface {
    Emit(parseClass string, parseCSS string) // idempotent per class; called once per new hash
}
```

- **`sink_wasm.go` (v1, ships):** appends a rule to a single managed `<style>` element in `<head>`.
- **`sink_native.go`:** writes into an in-memory buffer the SSR path and tests read back.
- The `Sink` is the documented seam for a **later** build-time extraction pass (`tools/gwc`) — design
  it in now as an interface, but only the runtime sink is built for v1.

Dedup means N identical `css.New(...)` calls across a render → one registry entry, one injected rule.

---

## SSR & hydration (no FOUC, no double-inject)

- During SSR the native buffer Sink collects every emitted rule; `ui/ssr_*` serializes the buffer into a
  `<style data-gwc-css>` block in the rendered `<head>` so styles are present on first paint.
- On client hydration the wasm Sink **pre-seeds its registry** from the server-rendered class names (read
  off the `data-gwc-css` block) so already-present rules are recognized as hits and **not re-injected**.
- Open item to confirm against the real `ui/ssr_*` API: the exact harvest hook and whether the registry
  pre-seed reads the DOM block or a serialized manifest. Resolve during implementation against that code.

---

## Dynamic values (the static/runtime boundary)

A rule built from a runtime value (`css.Gap(stateValue)`) would mint a new class per distinct value and
grow the registry unboundedly. Boundary:

- **Static rule** (compile-time-known values) → hashed class via `New`. The common case.
- **Runtime value** → fall back to an inline CSS custom property using the existing `StyleVar`
  (sugar.go:1247): the class references `var(--x)`, and the live value is set inline per element. Keeps
  the registry bounded while staying dynamic. `css` exposes a helper that pairs a `--var`-referencing
  rule with the matching `StyleVar` PropOption so the two stay in sync.

---

## Layer 2 — Tailwind engine port (typed-native, full parity)

Decision (locked with user): **typed-native only — no string parser** — and **aim for full Tailwind
parity**. The complete Tailwind utility + variant + theme model, reimplemented as typed Go over Layer 1.
There is no `tw("...")` class-string front-end: every utility is a typed Go symbol, so the whole surface
is autocompletable and compile-checked, and there is no Tailwind-grammar parser to own or keep
spec-compatible. "Parity" here is *coverage* (every utility/variant exists as a typed call), not
string-compatibility.

```go
u.Flex                       // display:flex
u.Gap(3)                     // theme spacing index 3 -> gap:.75rem
u.Bg(u.Slate900)             // theme color token
u.Text(u.Lg)                 // theme type scale
u.Hover(u.Bg(u.Slate800))    // variant = function composition
u.Md(u.Hover(u.Flex))        // variants stack by nesting: md:hover:flex
u.MtN(4)                      // negative margin (-mt-4)
u.Important(u.Block)         // !important
u.Gap(css.Px(7))             // arbitrary value = the Layer-1 typed value (gap-[7px])
```

Each `u.*` resolves its argument against the active `Theme` and returns Layer-1 `Rule`(s); emission,
dedup, and SSR all reuse the Layer-1 registry/sink unchanged. JIT is free: only called utilities emit.

### Full parity ⇒ the surface is code-generated, not hand-written
Full coverage is hundreds of utilities across the variant set — too large and too drift-prone to
hand-author. The `u` package is **generated** from a machine-readable spec:

- A spec table maps each Tailwind utility family → CSS property template + which `Theme` scale it draws
  from (e.g. `gap` ← spacing scale, `bg` ← color scale), plus negative-value and arbitrary-value support
  flags.
- A generator (under `tools/gwc`, native lane) emits typed `u.*` functions/vars + the variant wrappers
  (`Hover`, `Focus`, `Md`, `Lg`, `Dark`, `GroupHover`, `Peer*`, …) from the table + the `Theme`.
- Tracking upstream Tailwind = editing the spec table and regenerating, not rewriting Go by hand.
- Generated files are checked in (`u/*_gen.go`) so consumers need no codegen step; regeneration is a
  contributor task. Honest scope note: full parity is a large, perpetually-maintained table — the *core*
  (spacing/color/type/flex/grid/border/effects + responsive/hover/focus/dark/group/peer) lands first and
  the long tail fills in against the table over time.

### Theme is the centerpiece (the `tailwind.config.js` analog, in typed Go)
```go
type Theme struct {
    Spacing     map[int]css.Length      // 0..96 scale
    Colors      map[string]css.Color    // slate-900, etc.
    FontSizes   map[string]css.Length
    Breakpoints map[string]css.Length   // sm/md/lg/xl/2xl drive the responsive variants
    // … radii, shadows, z, etc.
}
func DefaultTheme() Theme                 // Tailwind's default scales
func UseTheme(parseTheme Theme)           // swap/extend; variants + utilities resolve against it
```

### Open at every layer (required, beyond the generated built-ins)
- `css.DefineUtility(name, rules...)` — user-defined utilities composing with built-ins (the plugin path).
- `css.Theme(tokens)` / `UseTheme` — branded token scales drive the whole generated surface.
- `css.DefineVariant(name, selector)` — new variant selectors (container queries, data-attr states).
- The `Sink` interface — custom emission targets without touching authoring.

Built on Layer 1; the generator is the new work item full parity introduces.

---

## Open items to confirm at implementation time

1. Exact `ui/ssr_*` harvest hook + hydration pre-seed mechanism (DOM block vs serialized manifest).
2. Class-name scheme/length and collision handling for the content hash.
3. Whether `css.Class(...)` should also accept already-built `Sheet`s/utility bundles for free composition.
4. Color/spacing/type token sets for the v1 curated scale.

## Tasks

deep-study the authoring surface (done — see interop section) → implement Layer 1 (rule/prop/value/
variant + registry + wasm sink) → SSR harvest + hydration suppression → Layer 2 theme + the `cssgen`
generator + spec table for the core utility families → generate `u` and fill the parity tail against the
table → a custom-utility/theme example → tests in native (hash/dedup determinism, generator output) and
wasm (injection + SSR + hydration) lanes.

Sequencing note: full parity is a long tail, so the generator + core families gate a *usable* engine;
the remaining utility families are incremental table entries, not blocking work.

# Framework Bench — one app, every framework

> **Companion analysis:** the
> [GWC Competitive Teardown](../competitive-analysis/GWC_COMPETITIVE_TEARDOWN.md) uses
> this bench as its corpus — a ranked, multidimensional gap analysis of where GWC is
> weak vs. the field and exactly which ideas to steal.


A research study: implement **one identical app** — a Bill/Tip Splitter — across as
many UI frameworks and languages as practical, so GoWebComponents (GWC) can be
compared apples-to-apples against the field on real, equivalent code.

Every implementation matches [`SPEC.md`](./SPEC.md) exactly: same DOM structure,
same behavior, same formulas, same visual design. Only the framework idioms differ.

## Why a Bill Splitter

It is small but exercises a broad surface in every framework:

- **local component state** — bill, tip %, people
- **shared/global state** — theme + round-up, read by ≥3 components (each framework's
  native mechanism: atoms, context, stores, signals, services)
- **derived values** — tip, total, per-person, rounding math
- **a keyed list** — per-person breakdown
- **a presets list** — tip buttons with active state
- **conditionals** — empty state, rounding note
- **forms** — number inputs, stepper, custom tip

## Layout

```
framework-bench/
  SPEC.md            # canonical contract — the single source of truth
  README.md          # this file
  .gitignore         # ignores all deps + build artifacts (node_modules, dist, bin, obj, target…)
  shared/
    styles.css       # canonical design; imported verbatim by every non-GWC app
  apps/
    gwc/             # Go → wasm  (the anchor; GoWebComponents)
    react/           # TypeScript
    vue/             # TypeScript
    svelte/          # TypeScript
    solid/           # TypeScript
    angular/         # TypeScript
    blazor/          # C# → wasm
```

Only source is committed. Each app's dependencies and build outputs are reproducible
from its manifest and are git-ignored, so the repo stays clean.

## Apps so far (23)

Grouped by language family; the wasm/compiled column matters most for comparing
against GWC's "compile the whole UI" thesis.

**JS / TS (compiles to JS):**

| App | Shared-state mechanism | Reactivity | Run |
|-----|------------------------|------------|-----|
| **react** | Context | VDOM diff | `npm i && npm run dev` |
| **preact** | `@preact/signals` | VDOM + signals | `npm i && npm run dev` |
| **vue** | `reactive()` store module | proxy tracking | `npm i && npm run dev` |
| **svelte** | `writable` stores | compiled reactivity | `npm i && npm run dev` |
| **solid** | module `createSignal` | fine-grained signals | `npm i && npm run dev` |
| **qwik** | `useStore` + context | resumable signals | `npm i && npm run dev` |
| **angular** | injectable signals service | signals | `npm i && npm run dev` |
| **lit** | store + ReactiveController | web components | `npm i && npm run dev` |
| **alpine** | `Alpine.store` | proxy tracking | static serve (CDN, no build) |

**Compiled to wasm / native — the most apples-to-apples with GWC:**

| App | Language | Shared-state mechanism | Run |
|-----|----------|------------------------|-----|
| **gwc** | Go | `state.UseAtom` keyed atoms | `gwc dev …` (see [apps/gwc](./apps/gwc/README.md)) |
| **go-app** | Go | context state (`SetState`/`ObserveState`) | `go run .` (port 8100) |
| **vugu** | Go | root fields | `vgrun .` |
| **vecty** | Go | component fields + `Rerender` | wasm build + serve |
| **blazor** | C# | DI singleton + `OnChange` | `dotnet run` |
| **leptos** | Rust | `provide_context` signals | `trunk serve` |
| **dioxus** | Rust | `use_context_provider` signals | `dx serve` |
| **yew** | Rust | `use_reducer` + `ContextProvider` | `trunk serve` |
| **sycamore** | Rust | `provide_context` signals | `trunk serve` |
| **flutter** | Dart | `ChangeNotifier` | `flutter run -d chrome` |

**Server-driven / other paradigm:**

| App | Language | Model | Run |
|-----|----------|-------|-----|
| **htmx** | Go (server) | hypermedia; state on server | `go run .` (see [apps/htmx](./apps/htmx/README.md)) |
| **livewire** | PHP (Laravel) | server components | needs a Laravel app |
| **elm** | Elm | The Elm Architecture (one Model) | `elm make` |
| **pyscript** | Python | imperative DOM (no framework) | static serve |

`alpine` / `pyscript` are build-less: serve the bench root statically and open
`/apps/<name>/index.html` so `../../shared/styles.css` resolves.

### Pinned versions & refinement status (verified against docs, 2026-06)

A doc-checked currency pass brought each app to its current stable idiom. Verified
online against each framework's docs/registry:

| App | Pinned | Notes from the refinement pass |
|-----|--------|--------------------------------|
| react | 19.1 | React 19 is the production standard; Context + `useState` idiom unchanged |
| preact | 10.25 / signals 2.0 | current |
| vue | 3.5 | Composition API; Vapor is opt-in compiler mode, not used |
| svelte | **5.16** | **rewritten to runes** (`$state`/`$derived`) + `mount()`; shared state is now a `.svelte.ts` runes module (was Svelte 4 stores/`$:`) |
| solid | 1.9.11 | current stable (Solid 2.0 still experimental) |
| qwik | 1.19.2 | v1 stable; **Qwik 2 (`@qwik.dev/core`) is still beta**, so pinned v1 |
| angular | 20 | **switched to zoneless** (`provideZonelessChangeDetection`); signals-only, no zone.js |
| lit | 3.2 | current |
| leptos | **0.7** | **rewritten to the `signal()`/`RwSignal::new` API + `leptos::prelude`** (was 0.6 `create_rw_signal`) |
| dioxus | 0.6 | current stable (0.7 in development) |
| yew | **0.23** | **rewritten to `#[component]`** (0.23 renamed `#[function_component]`); hooks/html! unchanged |
| sycamore | 0.9 | **rewritten to the `Indexed(list=…, view=…)` API** + reactive class closures (was an `iterable=*create_signal` hack) |
| blazor | **.NET 10** | LTS; bumped from net8.0 |
| go-app | v10.1.8 | current; `ObserveState`/`SetState` confirmed |
| flutter | Flutter 3.44 / Dart 3.12 | sdk lower bound + intl 0.20 bumped; canvas renderer (see caveat) |
| elm | 0.19.1 | unchanged (genuinely current — stable since 2019) |
| pyscript | **2026.3.1** | imperative DOM via the `document` FFI + delegated listeners — a valid current pattern for a re-render model (`@when` suits persistent-element apps) |
| livewire | **v4** | **rewritten as a v4 single-file component** (`resources/views/components/bill-splitter.blade.php`: `new class extends Component` + Blade in one file) |
| vugu | v0.3.x | **directives fixed** (`:data-theme`/`:disabled`, single-quoted Go-string attrs, `vg-for`/`vg-content`/`@click` per docs) |
| dioxus | 0.6 | confirmed idiomatic against 0.6 docs (`use_signal`/`use_context_provider`, `signal()` reads) |
| htmx | 2.0 + idiomorph | morph swap |

All apps are now at their current stable idiom per each framework's docs (verified
online June 2026). **The remaining gap is build verification, not currency:** only
`gwc` + `htmx` (Go) compile here. The doc-checked-but-not-built Rust apps (sycamore,
vugu list/event macro details, dioxus signal forms) are the likeliest to still need
small fixes once a real toolchain runs them — that's what the per-app CI lane is for.

### Paradigm caveats (where "identical markup/design" bends)

- **flutter** renders to a **canvas**, not the DOM — it cannot use the shared
  markup or `styles.css`. It reproduces the *visual design* with widgets + the same
  tokens; DOM/class parity is impossible by construction.
- **elm** has no separate shared-state primitive — *all* state is one `Model` by
  design. `theme`/`roundUp` are just fields.
- **htmx / livewire** keep state on the **server**; there is no client framework.
- **pyscript** has no reactive framework; the DOM is rebuilt imperatively.

## Verification status

> **Honesty note (important):** only two apps are compiled in this repo's
> environment — `gwc` (`go build` for both `GOOS=js GOARCH=wasm` and native) and the
> `htmx` server (`go build`), both passing. **Everything else is a faithful scaffold
> written to `SPEC.md` but not build-verified** — the toolchains (npm, dotnet, cargo,
> dart/flutter, elm, pyodide, php/laravel) are not available where these were
> authored. The Go peers (`go-app`, `vugu`, `vecty`) are each their **own Go module**
> (separate `go.mod`) so the parent GoWebComponents build never tries to compile them;
> run `go mod tidy` inside each to resolve deps. The intended next step is a per-app
> CI lane that installs each toolchain and records pass/fail here.

## Design parity

Non-GWC apps import the *same* `shared/styles.css`, so their design is byte-identical.
**GWC is the documented exception:** it reproduces the same design through typed
`css/u` utilities (generated, compile-checked, no Tailwind toolchain) rather than a
shared stylesheet — a deliberate divergence that is itself a comparison dimension.

## Adding a framework

1. `mkdir apps/<name>`, implement `SPEC.md` exactly.
2. Non-GWC apps import `../../shared/styles.css`; reproduce the `bs-*` DOM tree.
3. Use the framework's *native* shared-state primitive for `theme` + `roundUp`.
4. Add a row to the table above; keep deps/artifacts covered by `.gitignore`.

## Planned long tail (later passes)

Still open: vanilla JS, Marko, Qwik City (SSR), F# (Fable/Elmish), Scala
(Laminar), Kotlin (Compose HTML), ClojureScript (re-frame), Phoenix LiveView
(Elixir), Hotwire (Ruby), Reflex (Python SSR). Each added under `apps/` with the
same contract.

## Comparison dimensions to collect (per app, once build-verified)

LOC (impl only), dependency count, cold-build time, dev-server start time, raw +
compressed bundle size, first-paint, and a subjective ergonomics note. These will be
tabulated once the per-app build lanes exist.

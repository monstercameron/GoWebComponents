# GoWebComponents — Competitive Teardown & "Steal-This" Roadmap

> Goal: find every dimension where GWC is weak against the field, identify the best
> idea each competitor has, and lay out exactly what to take to make GWC the best
> framework on the market. Written 2026-06 against current stable releases, with the
> 23-app [framework-bench](../framework-bench/) as the concrete comparison corpus.

> **Companion:** [Volume II — DevX Killer Features](./GWC_DEVX_KILLER_FEATURES.md) goes
> granular on the *daily-felt* DevX gaps and killer-demo features (with API sketches),
> including the two highest-leverage 2026 plays: `//gwc:server` (tRPC with zero schema
> tax) and AI-native DevX ("the framework your AI gets right the first time").

This is deliberately adversarial about GWC. It assumes the strengths are real and
focuses the entire document on **gaps + the specific ideas worth robbing**. A short
"strengths / moat" section exists only so the recommendations don't accidentally
erode what already works.

---

## 0. Thesis

GWC's bet — *write the entire UI in Go, one language end-to-end, batteries included* —
is sound and underexploited. But the framework currently competes as a **2018-era
React clone** (VDOM + component re-render + hooks + atoms) compiled to a **heavy wasm
runtime**, at the exact moment the market has moved decisively to:

1. **fine-grained signals** (everyone except React),
2. **resumability / zero-hydration** (Qwik),
3. **type-safe isomorphic server functions** (Leptos, Qwik, SolidStart, TanStack Start),
4. **end-to-end type-safe routing + data** (TanStack), and
5. **copy-paste headless component distribution** (shadcn/ui).

The single most important insight in this document: **three of GWC's biggest
"weaknesses" are actually its biggest latent advantages**, because being *Go on both
the client and the server* lets GWC implement signals, server functions, and a
component registry *better than the JS frameworks can* — they fight a language/runtime
boundary that GWC simply doesn't have. The work is to flip them.

---

## 1. GWC's real moat (don't erode these while fixing the rest)

| Strength | Why it matters | Who can't match it |
|---|---|---|
| **One language, client + server** | Shared structs, zero schema drift, no JS build | React/Vue/Svelte/Solid (TS↔TS but separate runtimes); only Leptos/Dioxus (Rust) and Blazor (C#) are peers |
| **Browser SQLite + durable reactive `kvstate`** (encryption at rest, cross-tab sync, conflict resolution, CBOR/JSON codecs, debounced/onunload writes) | Real offline-first persistence with almost no wiring | Genuinely ahead of *every* JS framework; this is a differentiator nobody talks about |
| **Crash containment by default** | Panics in render/effects/events/async caught at every boundary as structured diagnostics | Only React error boundaries come close, and they're opt-in |
| **Batteries-included in one module** | Routing, state, fetch, SSR, streaming, islands, i18n, a11y, PWA, flags, devtools, typed CSS — no assembly | Beats the "assemble 12 npm libs" world for cohesion |
| **Typed, compile-checked CSS (`css`/`css/u`)** | A typo is a compile error; no Tailwind toolchain; SSR `<style>` + hydration seeding | Stronger than Tailwind-string approaches; comparable to vanilla-extract |
| **Conservative, documented stability tiers** | Enterprise-legible; explicit non-goals | Most JS frameworks churn harder |

Keep these. Every recommendation below should compose with them, not replace them.

---

## 2. The gap analysis (by dimension, ranked by strategic leverage)

Each dimension: **where GWC is → best-in-class idea → the gap → steal-this**, with a
rough **impact × effort**.

---

### D1 — Reactivity model: GWC is on the losing side of the signals war

**Where GWC is.** Component re-render over a fiber/VDOM with `UseState`/`UseReducer`/
atoms. Fine-grained updates exist (`ui.ReactiveRegion` + `UseSelector`) but the docs
explicitly call them *"an opt-in hot-path optimization, not the default reactivity
model."* That is the React-18 model.

**Best-in-class.** **Signals / fine-grained reactivity** — Solid, Svelte 5 runes,
Angular (signals + zoneless), Vue, Preact, Qwik. Components run **once** to wire a
reactive graph; only the exact DOM nodes subscribed to a changed signal update.
Update cost is proportional to *what changed*, not to *how many components contain it*.
The TC39 **Signals proposal** (Stage 1, designed with input from Angular/Solid/Vue/
Svelte/Preact/Qwik/MobX authors) is standardizing `Signal.State` + `Signal.Computed`.
Measured: Svelte 5 median INP ~24ms vs React ~68ms (~3×).

**The gap — and why it's worse for GWC than for React.** Every DOM mutation in GWC
crosses the **`syscall/js` boundary**, which is *much* slower than JS touching the DOM
directly. A VDOM diff + component re-render produces *many* boundary crossings.
**Fine-grained reactivity is therefore a double win for GWC**: it minimizes both
re-execution *and* the number of expensive wasm↔JS calls. Signals are more valuable to
a wasm framework than to a JS one — and GWC is the one framework treating them as an
afterthought.

**Steal this.**
- Promote fine-grained reactivity from "advanced optimization" to **the default
  authoring path**. Ship a first-class `state.Signal[T]` (Get/Set/Update) and
  `state.Computed[T]` that auto-track dependencies, and make `UseComputed`/derived
  values + region updates the *normal* way to render dynamic leaves — so a component
  body runs once and only signal-bound nodes re-render.
- You already have the primitives (`UseComputed`, `UseDerived`, `UseSelector`,
  `ReactiveRegion`). The work is **ergonomics + defaults + docs**, not a rewrite:
  make the fine-grained path shorter to type than the re-render path.
- Mirror the TC39 shape (`Signal.State`/`Signal.Computed`) so the mental model
  transfers and you can claim standards alignment.

**Impact: very high · Effort: high (but incremental — primitives exist).** This is the
top correctness-of-architecture bet.

---

### D2 — Startup & bundle size: the structural weakness, and the only real answer (resumability + server-first)

**Where GWC is.** Go→wasm has a **hard ~2MB floor** (GC + runtime); the bench
`counter` is ~6.3MB raw / ~1.24MB brotli. SSR + hydration means the **entire wasm
payload must download *and* execute to make the page interactive** — the worst case of
the hydration tax.

**Best-in-class.**
- **Qwik resumability**: serializes state *and* the location of event handlers /
  component boundaries into HTML, so the app **resumes** on the client with **zero
  hydration** — a button is interactive before any framework code runs, and code is
  lazy-loaded only on the interaction that needs it. Hydration costs 2–10s on mid
  mobile; resumability is ~O(1).
- **Astro/islands + server-first**: ship HTML, hydrate only interactive islands.
- **Rust→wasm** (Leptos/Dioxus): ~500KB vs Go's 2MB+ floor.

**The gap.** GWC cannot win cold-load size against JS (~50KB) or Rust (~500KB) — that
floor is structural. So **stop trying to win on raw size**; win on *not needing the
wasm for first paint*.

**Steal this (this is a *combination* play, and it's GWC's escape hatch):**
1. **Resumability mode.** Steal Qwik's core idea: server-render Go → HTML, serialize
   handler/boundary locations, and **resume** instead of hydrate. Lazy-load wasm chunks
   per interaction. This directly neutralizes the "6MB wasm" objection for content/
   commerce/marketing apps — the wasm never blocks first interaction.
2. **Server-first by default for content-heavy routes.** You already have **islands** —
   double down so the *default* for a content route is server-rendered Go HTML with
   wasm only in interactive islands. (You also already ship an **htmx-style server
   path** in the bench; a first-class "GWC server-driven mode" — like Phoenix LiveView/
   Livewire but in Go — could power whole app classes with *no client wasm at all*.)
3. **Keep brotli + tinygo profile** as the size knobs they already are, but treat them
   as secondary to (1) and (2).

**Impact: very high · Effort: very high (resumability is hard).** But it's the only
honest answer to the size critique, and it would be a genuine headline feature.

---

### D3 — Isomorphic server functions: GWC's single biggest *unbuilt* advantage

**Where GWC is.** **Nothing.** The model is "client wasm + a separate server you wire
by hand with `fetch`." The design notes even list *"server-interactive UI"* as
**experimental / deferred**.

**Best-in-class.** Type-safe server functions are now table stakes for serious
frameworks: **Leptos `#[server]`**, **Qwik `server$()`**, **SolidStart `"use server"`**,
**TanStack Start** (build replaces the impl with an RPC stub in the client bundle;
server code never ships), **RSC `"use server"`**. You write a function; calling it from
the client transparently becomes a validated, type-safe RPC.

**The gap — and why GWC should win it outright.** Every JS implementation fights a
boundary: client is TS, server is a *different runtime* (Node/edge), and they need
build-time codegen/stubbing + runtime validation to stay type-safe across the wire.
**GWC is Go on both sides, in one module.** A GWC server function can:
- be an **ordinary Go function** annotated `//gwc:server`,
- share the **exact same structs** as the client (zero schema, zero codegen for
  types — the types are *literally the same types*),
- generate a tiny client stub (POST + decode) and a server handler automatically,
- and even be **isomorphic** (the same Go runs on server *or* client — Leptos's
  pioneering idea) because it's all Go.

No JS framework can match "the request and response types are the same Go struct with
no boundary." This is the **#1 "rob them blind" opportunity** and it plays directly to
GWC's one-language moat.

**Steal this.**
- Ship `//gwc:server` functions: annotated Go func → auto client stub + server route +
  shared types + request validation. Borrow Solid's "`use server` ergonomics" reputation
  and Qwik's "get the RequestEvent without bloating the signature" pattern.
- Add **server actions for forms** (progressive-enhancement: works without wasm, better
  with it) — steal Remix/RSC actions.
- Wire it to the data layer (D5) so a server function result lands in the query cache.

**Impact: very high · Effort: medium-high.** Highest leverage-to-effort ratio in the
whole document. **Do this.**

---

### D4 — Routing: code-first and explicitly *not* type-safe/file-based

**Where GWC is.** `router` with a runtime `RouteContract` (`MustHref`, params).
Design notes explicitly make **file-based routing and route-manifest generation a
non-goal**, leaving route registries/prerender lists app-owned.

**Best-in-class.** **TanStack Router**: end-to-end type safety — params, **typed search
params with schema validation (Zod)**, loaders, context, and `Link` are all typed at
*compile time*; file-based route generation; automatic per-route code-splitting; SSR
via TanStack Start. "No other router offers this level of type safety." SvelteKit/
SolidStart/Next file-based routing for zero-config structure.

**The gap.** GWC routing is stringly-typed-ish and manual. **Typed search params with
validation** is the feature developers rave about and GWC has no answer for. Go's type
system could make routes *more* type-safe than TS, but GWC currently doesn't.

**Steal this.**
- **Typed routes + typed search params**: a `go:generate`-based (or `gwc`-CLI) route
  codegen that turns a route tree into typed `Link`/params/search structs with
  validation. This keeps "no compiler *required*" (it's opt-in generate) while
  delivering TanStack-grade safety.
- **Optional file-based routing** via the `gwc` CLI (scan `routes/`, generate the
  registry). Keep code-first as the floor; offer file-based as sugar. The non-goal was
  "core-owned manifest"; a CLI generator sidesteps that cleanly.
- **Per-route code-splitting** is doubly important given wasm size (D2) — split wasm by
  route.

**Impact: high · Effort: medium.** Typed search params alone is a beloved, attainable win.

---

### D5 — Data layer: solid primitives, but the cache is *Experimental* and missing the SWR magic

**Where GWC is.** `fetch.UseFetch/UseResource[T]/UseCachedResource[T]/Fetch/Upload` +
a durable **offline mutation queue** (genuinely good). But `UseCachedResource` and the
whole advanced cache lifecycle are flagged **Experimental**.

**Best-in-class.** **TanStack Query**: caching, **request dedupe**, retries,
**background refetch**, **window-focus refetch**, GC, **stale-while-revalidate** (show
stale instantly, refetch silently), **optimistic mutations with automatic rollback**,
query invalidation by key, infinite queries. It's the cross-framework gold standard and
has *no external deps*.

**The gap.** GWC has the pieces but the killer ergonomics — SWR-by-default, automatic
dedupe, window-focus/reconnect refetch, optimistic + rollback as a one-liner, invalidate-
by-key — are either missing or Experimental. Server state management is where most app
bugs live; this needs to be **Stable and best-in-class**.

**Steal this.**
- Build a **Query-equivalent that's Stable**: `query.Use[T](key, fetcher, opts)` with
  SWR, dedupe, background + focus + reconnect refetch, GC, and **invalidate-by-key**.
- **Optimistic mutations with rollback** as a first-class one-liner, wired to the
  offline mutation queue you already have (that's a differentiator — Query doesn't ship
  durable offline replay).
- Integrate with server functions (D3): calling a `//gwc:server` fn populates the cache.

**Impact: high · Effort: medium.** The design is well-understood — copy it faithfully,
then beat it with the offline queue + server-fn integration.

---

### D6 — Component ecosystem: the biggest *practical* adoption blocker — and shadcn shows the escape

**Where GWC is.** A small `a11y` package (Menu/Listbox/Combobox) — a real start, but a
handful of primitives. Effectively **no component ecosystem**, which the design notes
honestly admit ("weaker on market depth").

**Best-in-class.**
- **Headless primitives**: Radix (30+ accessible unstyled components; `react-slot` ~131M
  weekly downloads via shadcn), **Base UI** (MUI-maintained, the actively-developed
  Radix successor), **React Aria** (deepest a11y), **Ark UI** (same primitives **across
  React/Vue/Solid**, logic built as **XState state machines**), Melt (Svelte), Kobalte
  (Solid).
- **Distribution model: shadcn/ui** — a **CLI that copies component *source* into your
  repo**. You own the code, style it freely, no version-lock. This is the dominant way
  teams consume primitives in 2026.

**The gap.** No GWC team can build a real app without re-implementing Dialog, Combobox,
Tabs, Popover, Tooltip, DatePicker, etc. — each a multi-week a11y minefield. This is the
**#1 reason a team picks React over GWC today.**

**Steal this (two moves):**
1. **A headless primitive library** — port the Radix/Ark *behavior* (focus management,
   keyboard nav, ARIA, RTL, controlled/uncontrolled) as Go components. Ark's
   **state-machine** approach is the one to copy — machines are language-agnostic and
   easy to verify; a Go XState-equivalent gives provably-correct interaction logic.
2. **Steal shadcn's distribution model *verbatim*** — this is the cheat code. `gwc add
   dialog` copies a styled-with-`css/u`, a11y-correct Go component **into the user's
   repo**. You already have the `gwc` CLI + scaffold. A **registry of copy-paste Go
   components** sidesteps the entire "Go has no npm component ecosystem" problem using a
   model the market *already validated*. This could close the ecosystem gap faster than
   any other single investment.

**Impact: very high (adoption) · Effort: high (but parallelizable; each component is
independent).** The shadcn-style `gwc add` registry is the highest-impact ecosystem play.

---

### D7 — Authoring ergonomics & compile-time guarantees: verbose builders, no template DSL

**Where GWC is.** Builder functions (`html/shorthand`). Type-safe (Go's checker) but
**verbose** and visually noisy vs JSX. Internal convention prefixes nearly every
identifier with `parse*` — a real contributor-friction wart (flagged in earlier work).

**Best-in-class.** JSX (compile-checked, ubiquitous, great tooling), SFC templates
(Vue/Svelte — scoped styles, terse), macros (**Leptos `view!`**, **Dioxus `rsx!`** —
JSX-like ergonomics *in Rust*), Lit tagged templates. JSX+TS catches template errors at
compile time; Svelte/Solid **compilers** also optimize output.

**The gap.** Go has no macro system and no JSX, so GWC can't match `rsx!`/JSX terseness,
and there's no template-level static analysis beyond the type checker. Authoring is the
single most-felt daily ergonomic difference.

**Steal this (turn the weakness into a tooling strength):**
- **Invest hard in `gwc check`** (the hook-context analyzer already built is the seed):
  add rules-of-hooks, prop validation, a11y lint, dead-prop detection, exhaustive-deps —
  i.e., recreate what JSX+ESLint+TS give, as Go static analysis. *Tooling can substitute
  for a template compiler.*
- **Offer an optional template DSL** (`.gwc`/Vugu-style or a `go:generate` JSX-to-Go) for
  teams that want JSX ergonomics — keep Go-funcs as the no-codegen floor (respecting the
  "no compiler *required*" non-goal).
- **Kill the `parse*` convention** for public/new code; document it as internal-only.
  It actively deters external contributors and reads as non-idiomatic Go.

**Impact: medium-high · Effort: medium.** The analyzer path is the pragmatic, on-brand win.

---

### D8 — DX inner loop: the wasm rebuild tax

**Where GWC is.** `gwc dev` + `hotreload` (state-preserving). But each change triggers a
**Go→wasm rebuild (seconds) + reload**.

**Best-in-class.** **Vite HMR < 50ms** — feels instant; module-level hot replacement, no
full reload, excellent error overlay. The whole JS ecosystem is now built on this
expectation (Rolldown/Turbopack push it further).

**The gap.** Seconds-per-change vs sub-50ms is the difference between flow and
frustration — the most-felt DX gap after authoring. `hotreload` preserving atom state is
good, but the rebuild latency dominates.

**Steal this.**
- **Attack rebuild latency**: aggressive incremental compilation, a warm build daemon,
  precompiled framework objects, and per-component recompilation where possible.
- **True component-level hot-swap** (not full reload) — `hotreload` already preserves
  state; push toward swapping a component's code in place.
- **A first-class in-page error overlay** — you already produce structured console
  diagnostics; render them as a Vite/React-style overlay with source mapping.

**Impact: high (retention) · Effort: high (wasm toolchain is the constraint).**

---

### D9 — Animation & transitions: thin and Experimental

**Where GWC is.** `transitions` exist but are **Experimental**; no motion system.

**Best-in-class.** **Svelte** built-ins (`transition:fade/fly/scale`, **`animate:flip`**
for list reordering, `spring`/`tweened` motion stores) — *no library needed*. **Motion**
(ex-Framer Motion) for React/Vue/JS. The **View Transitions API** for route/state
morphs.

**The gap.** No enter/exit transitions, no FLIP list animation, no spring physics, no
View Transitions integration. Modern UIs feel dead without these.

**Steal this.**
- Built-in **enter/exit transitions** + **FLIP** for keyed lists (steal Svelte's
  `animate:flip` — it's the single most-loved animation primitive) + **spring/tween**
  motion values driven by signals (D1).
- First-class **View Transitions API** integration for route changes.

**Impact: medium · Effort: medium.**

---

### D10 — Cross-platform reach: web-only

**Where GWC is.** **Web only.**

**Best-in-class.** **Dioxus**: one Rust codebase → web (wasm), desktop (WebView/native
WGPU), **mobile** (`dx serve --platform ios/android`). React Native, Flutter, Tauri also
own multi-target.

**The gap.** GWC can't follow a team to desktop/mobile — a ceiling on TAM.

**Steal this (opportunistic, not urgent).**
- **Desktop via WebView** (wails/go-webview) is low-hanging: the same Go UI in a native
  shell. Steal Dioxus's "one codebase, `serve --platform`" CLI ergonomics.
- Mobile is a much larger bet; flag as future.

**Impact: medium (TAM) · Effort: desktop low / mobile very high.** Desktop is a cheap
"GWC runs on the desktop too" headline.

---

### D11 — Forms & validation

**Where GWC is.** `UseForm` + a11y form behavior + the offline mutation queue (good
bones). Validation/touched semantics are treated as a public contract (mature).

**Best-in-class.** React Hook Form / **TanStack Form** (typed, framework-agnostic),
Felte, Angular reactive forms; **server actions** (Remix/RSC) for progressive-enhanced
form posts that work without JS.

**The gap.** No typed schema-validation story to match, and no **server-action form**
(works without wasm). The latter pairs perfectly with D3.

**Steal this.** Typed schema validation + **server-action forms** (progressive
enhancement: HTML form works with zero wasm, wasm upgrades it). Directly leverages
server functions (D3) — a combined win.

**Impact: medium · Effort: medium.**

---

### D12 — DevTools maturity

**Where GWC is.** A `devtools` package (inspection/diagnostics/snapshots) — more than
most young frameworks.

**Best-in-class.** React/Vue DevTools (component tree, props, perf flamegraphs), Redux
DevTools (**time-travel**), signal-graph inspectors (Solid/Angular).

**The gap.** Likely no time-travel, no visual reactive-graph/signal inspector, weaker
component-tree UX than React/Vue DevTools.

**Steal this.** **Time-travel** (you already have snapshot export/import — wire it into
devtools as undo/replay) and a **reactive-graph visualizer** once signals land (D1).
Snapshot infra makes time-travel unusually cheap for GWC — a quick differentiator.

**Impact: medium · Effort: low-medium (snapshot infra already exists).**

---

### D13 — Ecosystem, docs, community, distribution

**Where GWC is.** Excellent first-party docs (reference manual, capability matrix,
stability tiers). Small community, **no pkg.go.dev discoverability push**, no component
registry, no template marketplace.

**Best-in-class.** Massive ecosystems, `create-x` starters, shadcn registries, huge
Q&A/hiring pools, deploy-platform adapters (Vercel/Netlify/SvelteKit adapters).

**The gap.** Network effects. Partly unwinnable short-term, but **distribution mechanics
are copyable**.

**Steal this.** The `gwc add` component registry (D6) + a `gwc create` starter gallery
(you have starters) + **deployment adapters** (one-command deploy to common hosts) +
a real **pkg.go.dev presence** (per-package docs). Distribution is a product surface, not
just headcount.

**Impact: medium-high (compounding) · Effort: medium.**

---

## 3. The "rob them blind" roadmap (prioritized)

Sequenced by **leverage** and by how directly each plays to GWC's one-language moat.

### Tier 0 — Asymmetric wins (do these; competitors *structurally cannot* match them)
1. **Isomorphic server functions (`//gwc:server`)** — D3. Shared Go types across the
   wire, no codegen, optional isomorphic execution. Nobody can match "the request and
   response are the same Go struct." **Highest leverage-to-effort.**
2. **`gwc add` headless component registry (shadcn model)** — D6. Copy-paste, `css/u`-
   styled, a11y-correct Go components via the CLI. Closes the #1 adoption gap with a
   market-validated distribution model.
3. **Stable, best-in-class query/data layer** — D5. SWR + dedupe + focus refetch +
   optimistic/rollback, *plus* the durable offline queue nobody else ships. Wire to (1).

### Tier 1 — Architecture corrections (the market has moved; GWC must follow)
4. **Fine-grained signals as the default** — D1. Promote the existing primitives to
   first-class; double win for wasm (fewer `syscall/js` crossings).
5. **Resumability + server-first mode** — D2. The only honest answer to wasm size;
   neutralizes the heaviest objection. Combine with islands and a Phoenix-LiveView-style
   server-driven Go mode.
6. **Typed routing + typed search params (+ optional file-based via CLI)** — D4.

### Tier 2 — Polish that compounds
7. **Animation system** (FLIP, enter/exit, spring, View Transitions) — D9.
8. **DX loop**: faster rebuilds, component hot-swap, in-page error overlay — D8.
9. **Server-action forms + typed validation** — D11.
10. **Time-travel devtools + signal-graph inspector** — D12.
11. **`gwc check` as a JSX-ESLint-equivalent** + kill `parse*` convention — D7.
12. **Desktop-via-WebView + deploy adapters + pkg.go.dev push** — D10/D13.

---

## 4. What NOT to copy (strategic discipline)

- **React's VDOM / re-render model** — don't double down; go fine-grained instead.
- **RSC's `use client`/`use server` inconsistency** — the research notes its API
  warts; GWC's one-language server functions can be *cleaner*. Take the idea, not the
  mess.
- **npm-style dependency sprawl** — the `gwc add` copy-paste model avoids version hell;
  keep it.
- **A *required* compiler/template step** — keep Go-funcs as the no-codegen floor; make
  DSLs/codegen opt-in (consistent with stated non-goals).
- **Chasing every meta-framework knob** — keep the batteries-included, explicit-
  ownership identity. Win on cohesion + one-language, not on imitating all of Next.js.

---

## 5. One-paragraph verdict

GWC's weaknesses cluster into two buckets: (a) **architecture the market has moved past**
(VDOM re-render, hydration, no signals) and (b) **table-stakes surfaces it hasn't built
yet** (server functions, a query layer, a component registry, typed routing, animation).
Bucket (b) is mostly *known, copyable engineering* — and in the three highest-leverage
cases (**server functions, component registry, data layer**), GWC's one-language design
lets it ship a *better* version than the frameworks it's copying. Bucket (a) is harder
(signals + resumability), but the primitives for signals already exist and resumability
is the only credible answer to the wasm-size critique. Do **Tier 0** first: it's where
GWC can be not just competitive but *uniquely best*.

---

## Sources

Reactivity / signals: [jsmanifest — Signals won the war](https://jsmanifest.com/signals-runes-fine-grained-reactivity) ·
[TC39 Signals / InfoWorld](https://www.infoworld.com/article/4129648/reactive-state-management-with-javascript-signals.html) ·
[SolidJS fine-grained](https://strapi.io/blog/solidjs-explained-fine-grained-reactive-framework) ·
[React 19 vs Svelte 5 latency](https://www.sitepoint.com/react-19-compiler-vs-svelte-5-virtual-dom-latency-benchmark/).
Resumability: [Qwik resumable docs](https://qwik.dev/docs/concepts/resumable/) ·
[Builder — resumability vs hydration](https://www.builder.io/blog/resumability-vs-hydration) ·
[The New Stack](https://thenewstack.io/javascript-on-demand-how-qwik-differs-from-react-hydration/).
Server functions: [Leptos server fns](https://book.leptos.dev/server/25_server_functions.html) ·
[TanStack Start server functions](https://tanstack.com/start/latest/docs/framework/react/guide/server-functions) ·
[Bridging the server/client divide](https://benw.is/posts/bridging-the-divide).
Routing: [TanStack Router](https://tanstack.com/router/latest) ·
[Type Safety](https://tanstack.com/router/latest/docs/guide/type-safety) ·
[Search Params](https://tanstack.com/router/latest/docs/guide/search-params).
Data: [TanStack Query](https://tanstack.com/query/latest).
Headless / shadcn: [GreatFrontend — top headless UI 2026](https://www.greatfrontend.com/blog/top-headless-ui-libraries-for-react-in-2026) ·
[LogRocket — Radix vs React Aria vs Ark vs Base](https://blog.logrocket.com/headless-ui-alternatives/).
Animation: [Svelte vs Framer Motion / Motion](https://motion.dev/) ·
[Strapi — Svelte vs React](https://strapi.io/blog/svelte-vs-react-comparison).
Cross-platform: [Dioxus](https://dioxuslabs.com/) ·
[Dioxus 0.6 release](https://dioxuslabs.com/blog/release-060/).
DX / HMR: [Vite — why](https://vite.dev/guide/why) ·
[Vite 8 / Rolldown](https://dev.to/gabrielenache/vite-8-is-here-and-it-changes-everything-about-frontend-builds-45ci).
Bundle size: [WASM Go vs Rust vs AssemblyScript](https://ecostack.dev/posts/wasm-tinygo-vs-rust-vs-assemblyscript/) ·
[WASM value proposition](https://nickb.dev/blog/the-webassembly-value-proposition-is-write-once-not-performance/).

# devx-maxxing — A Developer-Experience Rubric for Application Frameworks

Status: research item (living document)
Owner: Cam
Scope: GoWebComponents, but written framework-agnostic so it can score any
application UI framework (React/Next, Solid, Svelte/Kit, Vue/Nuxt, Qwik, Leptos,
Blazor, etc.) on equal terms.

## Purpose

Most "DX" claims are vibes. This rubric turns developer experience into something
**measurable, comparable, and falsifiable**. It enumerates every dimension that
shapes how it feels to build, ship, and maintain a real application on a
framework, and gives each one a concrete **1–10 maturity ladder** plus the
evidence you must collect to assign a score.

Goals:

1. **Audit** GoWebComponents honestly — find the DX cliffs before users do.
2. **Benchmark** against incumbents on identical criteria (no goalpost moving).
3. **Prioritize** roadmap work by largest score gap × dimension weight.
4. **Prevent drift** — a rescored snapshot over time shows whether DX is
   actually improving or just accreting features.

## How to use this rubric

1. For each dimension, gather the **evidence** listed (don't score from memory).
2. Assign a score **1–10** using the ladder. The ladder anchors every other
   rung (2/4/6/8/10); use the odd values (3/5/7/9) for "clearly past the lower
   anchor but not yet the higher one." When torn between two rungs, score the
   lower unless the higher rung's *entry condition* is fully met.
3. Record the evidence link/command and a one-line justification per score.
4. Multiply each score by the dimension **weight** (see weighting) for a
   composite. Always publish the per-dimension scores too — the composite hides
   cliffs.
5. Re-score on a cadence (per minor release). Track deltas, not absolutes.

### The universal maturity ladder

Every dimension uses the same 1–10 scale so scores are comparable. Even rungs are
**anchored** (each dimension defines what they mean concretely); odd rungs are
deliberate in-between scores.

| Level | Band         | Meaning                                                                       |
|-------|--------------|-------------------------------------------------------------------------------|
| 1     | Hostile      | Present but actively works against the developer — harmful default, a trap.   |
| 2     | Absent       | Missing, or so broken it's effectively not there.                             |
| 3     | —            | Between absent and rudimentary.                                               |
| 4     | Rudimentary  | Exists but manual, undocumented, or requires expert workarounds.              |
| 5     | —            | Between rudimentary and functional.                                           |
| 6     | Functional   | Works for the common case; rough edges, gaps at the margins.                  |
| 7     | —            | Between functional and strong.                                                |
| 8     | Strong       | Polished, documented, discoverable; handles edge cases; few surprises.        |
| 9     | —            | Between strong and exemplary.                                                 |
| 10    | Exemplary    | Best-in-class; the framework *teaches* the right thing by default — the one others copy. |

A simple gut-check: **1–2** = I fought it or it wasn't there; **4** = I had to
read source; **6** = I read the docs and it worked; **8** = it just worked and the
error told me what to do when it didn't; **10** = it made me a better developer.

### Two lenses, scored separately

Every dimension is graded through two lenses, because they diverge:

- **First-hour lens** — a newcomer's experience (discoverability, defaults,
  time-to-success).
- **Tenth-kiloline lens** — a team six months in at scale (maintainability,
  refactor safety, performance under growth, escape hatches).

A framework can be a 10 in the first hour and a 3 at scale (and vice versa). Note
both when they differ.

---

## Dimension catalog

Grouped into six phases of the development lifecycle. Each dimension has:
**Definition · Why it matters · 1–10 ladder (anchored at 2/4/6/8/10) · Evidence
to collect · Red flags.**

---

## Phase A — First run & onboarding

### A1. Installation & prerequisites
- **Definition:** Getting the framework and its toolchain onto a clean machine.
- **Why:** The first failure here is the highest-leverage churn point; people
  quit before they ever write code.
- **Ladder:**
  - 2 — Undocumented prereqs; install fails silently or with cryptic errors.
  - 4 — Manual multi-step setup; version pitfalls discovered by trial.
  - 6 — One documented install path; prereqs listed; works on the main OS.
  - 8 — Single command; a `doctor`/preflight verifies the environment and names
    exactly what's missing.
  - 10 — Cross-platform parity, pinned/declared toolchain, reproducible, and the
    preflight auto-suggests the fix command.
- **Evidence:** clean-VM install timing on Win/macOS/Linux; presence of a
  `doctor` command; how a missing dep is reported.
- **Red flags:** "works on my machine"; global mutable state; undocumented env
  vars; OS-specific gaps presented as universal.

### A2. Time-to-first-render (hello world)
- **Definition:** Minutes from "decided to try it" to a running app in a browser.
- **Why:** Proxy for the entire happy path; sets emotional tone.
- **Ladder:**
  - 2 — No runnable hello world; copy-paste from README fails.
  - 4 — Works after manual fixes/missing steps.
  - 6 — README snippet runs as written; >10 min.
  - 8 — `<5 min`, one command, live in the browser.
  - 10 — `<2 min`, live-reloading, and the starter is *idiomatic* (teaches good
    structure, not a toy).
- **Evidence:** stopwatch the golden-path README on a clean checkout; does the
  copy-pasted snippet compile and run unmodified?
- **Red flags:** hello world that doesn't resemble real apps; hidden setup the
  README omits.

### A3. Scaffolding, starters & templates
- **Definition:** Generating a new project or feature with sensible defaults.
- **Why:** Determines whether teams converge on good structure or each reinvent.
- **Ladder:**
  - 2 — None; copy an example folder by hand.
  - 4 — A single bare template.
  - 6 — `create`/`start` command with a couple of templates.
  - 8 — Interactive scaffolder; multiple curated starters (app shapes: SPA, SSR,
    PWA, library); each starter builds and tests green out of the box.
  - 10 — Composable generators (add-a-route, add-a-component), starters are CI-
    verified, and the generated code matches the docs' idioms exactly.
- **Evidence:** list of starters; do they build/test clean; drift between
  generated code and current API.
- **Red flags:** starters that no longer compile; generated code using
  deprecated APIs.

---

## Phase B — Authoring experience (writing the app)

### B1. Component model & composition
- **Definition:** How UI units are declared, nested, parameterized, and reused.
- **Why:** The single most-touched surface; small ergonomic taxes compound.
- **Ladder:**
  - 2 — No real composition; copy-paste reuse.
  - 4 — Components exist but props/children are stringly-typed or awkward.
  - 6 — Typed props, children, basic composition.
  - 8 — Typed props + children + context/slots + clear parent/child data flow;
    composition patterns documented.
  - 10 — Composition is uniform (no special cases), refactor-safe, and the model
    scales from a button to an app without a paradigm shift.
- **Evidence:** declare a component with typed props + children + a slot; refactor
  a child's prop and see if the compiler catches all call sites.
- **Red flags:** prop drilling with no escape hatch; "magic" that breaks under
  composition; different rules for leaf vs. container components.

### B2. API design & ergonomic consistency
- **Definition:** Naming, argument order, return shapes, and predictability
  across the public surface.
- **Why:** Consistency is *learnability* — you guess the next API correctly.
- **Ladder:**
  - 2 — Inconsistent naming/ordering; surprising side effects.
  - 4 — Mostly consistent within a package, divergent across packages.
  - 6 — Documented conventions; a few legacy warts.
  - 8 — Uniform naming/arg-order/error conventions enforced by lint or review;
    deprecations are signposted.
  - 10 — The API is *predictable enough to guess* — a developer who learns one
    package can use the next without docs.
- **Evidence:** a conventions doc; grep for naming inconsistencies; can you guess
  an unfamiliar function's signature from a sibling's?
- **Red flags:** boolean-trap parameters; same concept named three ways; silent
  no-ops on typos.

### B3. Type safety & compile-time guarantees
- **Definition:** How much wrongness is caught at build time vs. runtime.
- **Why:** Compile-time errors are the cheapest possible feedback loop.
- **Ladder:**
  - 2 — Stringly-typed everything; errors surface in the browser at runtime.
  - 4 — Types exist but heavy `any`/casts at the boundaries.
  - 6 — Core APIs typed; some unchecked surfaces (styles, routes, queries).
  - 8 — End-to-end typing incl. styling, routes, and data; a typo is a compile
    error, not a silent no-op.
  - 10 — Types *guide* authoring (autocomplete-as-documentation), illegal states
    are unrepresentable, and shared types span server↔client by construction.
- **Evidence:** introduce a typo in a style token / route name / query key — does
  it fail to compile? Are server and client types the same source?
- **Red flags:** runtime "unknown class" / "no such route"; casts to escape the
  type system; types that lie (compile but crash).

### B4. Styling experience
- **Definition:** Authoring, composing, theming, and shipping CSS/visual style.
- **Why:** A perennial DX swamp; scoping, theming, and dead-code are hard.
- **Ladder:**
  - 2 — Global stylesheets, manual class strings, collisions.
  - 4 — Scoped styles but stringly-typed; no theme system.
  - 6 — Scoped + a utility or token layer; some compile checks.
  - 8 — Typed/compile-checked styles, theme tokens, pseudo/media variants, dead-
    style elimination, SSR-safe extraction.
  - 10 — Styling is type-safe end to end, autocompletes, folds to atomic output,
    handles dynamic values without class explosion, and theme switching is free.
- **Evidence:** typo a color token; check generated CSS size and dedup; verify
  SSR style extraction + hydration; dynamic value handling.
- **Red flags:** unbounded class generation; FOUC on hydration; theme = global
  override hacks.

### B5. State management
- **Definition:** Local component state and shared/global app state.
- **Why:** Determines testability, predictability, and refactor blast radius.
- **Ladder:**
  - 2 — Ad hoc mutable globals.
  - 4 — Local state only; shared state is DIY.
  - 6 — Local hooks + a basic shared store.
  - 8 — Local state + atoms/derived/computed + subscriptions + snapshot
    import/export; clear update semantics.
  - 10 — Above + time-travel/snapshots, persistence/restore, fine-grained
    reactivity (no over-render), and a documented mental model for when to use
    which tool.
- **Evidence:** derive state, persist/restore it, observe render granularity
  (does an unrelated update re-render the world?).
- **Red flags:** "everything re-renders"; two competing state systems with no
  guidance; mutation without subscription.

### B6. Data fetching & async
- **Definition:** Loading, caching, mutating, and streaming server data.
- **Why:** Real apps are mostly async glue; this is where bugs live.
- **Ladder:**
  - 2 — Raw fetch, manual loading/error flags everywhere.
  - 4 — A fetch helper; no caching.
  - 6 — Typed resources + basic cache + loading/error states.
  - 8 — Tag-aware query cache, mutations w/ invalidation, realtime
    (WS/SSE) hooks, suspense/async boundaries, retries.
  - 10 — Above + request dedup, optimistic updates, streaming, and async errors
    contained (no white screen) with structured diagnostics.
- **Evidence:** model a list+detail+mutation with cache invalidation; kill the
  network mid-request and watch the failure mode.
- **Red flags:** waterfalls by default; cache you can't invalidate; an async
  panic that blanks the page.

### B7. Routing
- **Definition:** Mapping URLs to UI, params, guards, loaders, nesting.
- **Why:** Routing shapes the whole app's data + navigation architecture.
- **Ladder:**
  - 2 — Manual `switch` on location.
  - 4 — Basic path matching, no params/typing.
  - 6 — Params + query + redirects.
  - 8 — Guards, loaders, metadata, nested layouts, typed params, hydration-aware.
  - 10 — Above + code-split/lazy routes, revalidation, SSR route-data reuse, and
    routes are type-checked (no string drift between link and route).
- **Evidence:** add a nested layout with a guarded, lazy, param'd route; mistype
  a link target and see if it compiles.
- **Red flags:** untyped string routes; loaders that can't cancel on navigation;
  layout state lost on nav.

### B8. Forms & validation
- **Definition:** Input binding, validation, submission, error display.
- **Why:** High-friction, high-frequency surface; easy to get tediously verbose.
- **Ladder:**
  - 2 — Manual value/onChange/error wiring per field.
  - 4 — A form helper, no validation integration.
  - 6 — Binding + basic validation hooks.
  - 8 — Typed form models, field-level + form-level validation, submission state,
    accessible error wiring.
  - 10 — Above + shared client/server validation (one schema), async validation,
    and dirty/touched/reset semantics that "just work."
- **Evidence:** build a multi-field form with cross-field validation reused on
  the server.
- **Red flags:** validation logic duplicated client/server; a11y of errors
  ignored; uncontrolled/controlled confusion.

---

## Phase C — The inner loop (the feedback cycle)

### C1. Local dev server & hot reload
- **Definition:** Edit → see result. Reload speed and state preservation.
- **Why:** This loop runs thousands of times/day; latency here is pure tax.
- **Ladder:**
  - 2 — Manual rebuild + manual browser refresh.
  - 4 — Watch + full page reload.
  - 6 — Auto rebuild + auto reload, multi-second.
  - 8 — Fast reload, serves + live-reloads on save, sub-second-ish.
  - 10 — State-preserving hot reload (component state survives edits), near-
    instant, with a clear indicator of reload status.
- **Evidence:** time save→repaint; does component state survive an edit?
- **Red flags:** lost state on every edit; reload that silently no-ops; needing a
  manual restart for some file types.

### C2. Build speed & incremental compilation
- **Definition:** Cold and warm build times; incremental rebuild granularity.
- **Why:** Compile latency gates the inner loop and CI cost.
- **Ladder:**
  - 2 — Always full rebuild; minutes.
  - 4 — No caching; slow but works.
  - 6 — Some caching; warm builds tolerable.
  - 8 — Incremental, cached, parallel; warm builds fast.
  - 10 — Near-instant warm builds, build cache shared across CI/dev, and the tool
    reports what was rebuilt and why.
- **Evidence:** cold vs warm build of a representative app; rebuild after a
  one-line change.
- **Red flags:** rebuild-the-world on a comment change; opaque build time.

### C3. Error messages & diagnostics
- **Definition:** Quality of compile-time and runtime error reporting.
- **Why:** Error quality *is* DX — a great error is a free tutorial.
- **Ladder:**
  - 2 — Stack traces into framework internals; no app context.
  - 4 — Generic message, no location.
  - 6 — Message + file/line.
  - 8 — Message + location + likely cause + suggested fix; errors are
    structured/coded.
  - 10 — Errors are coded, searchable, linkable to docs, name the offending
    symbol, propose the exact fix, and are *agent-readable* (machine-parseable).
- **Evidence:** trigger 5 common mistakes; grade each message; is there an
  error-code index?
- **Red flags:** errors that point at framework internals; "undefined is not a
  function" equivalents; silent failures.

### C4. Debugging experience
- **Definition:** Inspecting state, stepping, source mapping, time-travel.
- **Why:** When the error message isn't enough, this is the next line.
- **Ladder:**
  - 2 — `print` debugging only; no source maps.
  - 4 — Source maps sometimes; breakpoints land in generated code.
  - 6 — Source maps to original code; basic logging.
  - 8 — Source-debug build profile, component tree inspector, state inspection.
  - 10 — Above + time-travel/snapshots, commit profiling, and a debug build that
    correlates browser stacks to original source reliably.
- **Evidence:** set a breakpoint in original source; inspect a component's live
  state; correlate a browser stack to source.
- **Red flags:** breakpoints in transpiled/generated output; no way to see live
  component state.

### C5. Editor / IDE integration
- **Definition:** Autocomplete, go-to-def, inline docs, lint, refactors.
- **Why:** The IDE is where the code is actually written; integration multiplies
  every other dimension.
- **Ladder:**
  - 2 — No language support; plain text.
  - 4 — Syntax highlight only.
  - 6 — Autocomplete + go-to-def via the base language server.
  - 8 — Inline docs on hover, framework-aware lints, safe rename across the app.
  - 10 — Framework-specific tooling (hooks-rules analyzer, route/style
    completion), quick-fixes, and autocomplete that doubles as documentation.
- **Evidence:** hover an API for docs; rename a component; does a custom lint
  catch a misuse (e.g., conditional hook)?
- **Red flags:** autocomplete that suggests private internals; renames that miss
  call sites; no framework-specific lints.

---

## Phase D — Quality & confidence

### D1. Testing story
- **Definition:** Unit, component, integration, browser/e2e, and test ergonomics.
- **Why:** Confidence to refactor is downstream of test ergonomics.
- **Ladder:**
  - 2 — No testing guidance; components untestable in isolation.
  - 4 — Unit tests of pure logic only.
  - 6 — Component tests via a harness; some browser coverage.
  - 8 — Native unit + headless component + browser/e2e lanes, with reusable test
    helpers shipped.
  - 10 — Above + fast deterministic component tests w/o a browser, first-class
    fixtures, and a one-command lane runner used in CI and locally.
- **Evidence:** test a stateful component without a browser; run the full lane
  suite; is there a consumer-facing testkit?
- **Red flags:** must boot a real browser for any component test; flaky e2e as
  the only option; no test helpers for framework constructs.

### D2. Observability, logging & devtools
- **Definition:** In-app inspection, logging, diagnostics, profiling surfaces.
- **Why:** Production and dev both need a window into runtime behavior.
- **Ladder:**
  - 2 — `console.log` only.
  - 4 — Structured logging.
  - 6 — A devtools panel or inspector.
  - 8 — Embeddable inspector + diagnostics + profiling hints + snapshots.
  - 10 — Above + browser-extension bridge, RUM/OTel export, scheduler
    instrumentation, and redaction for sensitive data.
- **Evidence:** open the inspector; export a profiling snapshot; pipe diagnostics
  to OTel.
- **Red flags:** no runtime visibility; logging that leaks secrets; profiling
  that perturbs the thing it measures.

### D3. Performance & profiling
- **Definition:** Render/runtime performance and the tools to measure it.
- **Why:** DX includes *finding* and *fixing* slowness, not just being fast.
- **Ladder:**
  - 2 — No benchmarks; perf is folklore.
  - 4 — Ad hoc timing.
  - 6 — Some microbenchmarks.
  - 8 — Checked-in benchmark suite (native + target), reproducible commands,
    no-drift reporting.
  - 10 — Above + a render benchmark vs. a named baseline framework, regression
    gates, and honest documented trade-offs (no inflated "native-speed" claims).
- **Evidence:** run the bench suite; reproduce a vs-baseline comparison; is there
  a drift policy?
- **Red flags:** hard-coded perf numbers in prose that can drift; "native speed"
  claims without a reproducible benchmark.

### D4. Accessibility
- **Definition:** A11y primitives, semantic defaults, and guidance.
- **Why:** Hard to retrofit; a framework that makes a11y the default is rare.
- **Ladder:**
  - 2 — No a11y consideration.
  - 4 — Developer fully on their own.
  - 6 — Some semantic helpers.
  - 8 — Headless accessible components (menu, combobox, listbox, dialog), focus
    + announcer primitives, ARIA contracts documented.
  - 10 — Above + a11y baked into form/error wiring, focus management on
    navigation, and lint/checks for common violations.
- **Evidence:** build a combobox keyboard-navigable + screen-reader correct using
  shipped primitives; check focus on route change.
- **Red flags:** divs-as-buttons in examples; no focus management; a11y as an
  afterthought doc.

### D5. Security
- **Definition:** Safe-by-default rendering, escaping, CSP, secret handling.
- **Why:** DX includes not making it easy to ship a vulnerability.
- **Ladder:**
  - 2 — Raw HTML injection is the easy path.
  - 4 — Manual escaping required.
  - 6 — Auto-escaping by default; raw-HTML is opt-in and named scary.
  - 8 — Above + CSP guidance, XSS-safe styling/attrs, documented threat model.
  - 10 — Above + secret-redaction in logs/telemetry, secure SSR data transport,
    and security review docs (`SECURITY.md`) with a reporting path.
- **Evidence:** how hard is it to inject raw HTML; is escaping the default; is
  there a SECURITY.md + redaction?
- **Red flags:** unescaped interpolation by default; secrets in client bundle;
  no disclosure policy.

### D6. SSR / hydration / rendering modes
- **Definition:** Server rendering, streaming, hydration, islands, CSR fallback.
- **Why:** First-paint, SEO, and perceived speed live here; also a bug minefield.
- **Ladder:**
  - 2 — CSR only.
  - 4 — Basic render-to-string, no hydration.
  - 6 — SSR + whole-tree hydration.
  - 8 — Streaming SSR (shell-first), async suspension, hydration with mismatch
    diagnostics, selective islands.
  - 10 — Above + hydrate-on-visible/interaction/idle islands with budgets, route-
    data reuse, multiple transport encodings, and DOM-reuse hydration.
- **Evidence:** stream a shell before async resolves; hydrate an island on
  scroll; force a hydration mismatch and read the diagnostic.
- **Red flags:** hydration mismatches that silently corrupt; no streaming; islands
  that re-download the whole app.

---

## Phase E — Delivery

### E1. Build output, bundle size & optimization
- **Definition:** Artifact size, compression, dead-code elimination, profiles.
- **Why:** First-load cost is a real user-facing tax; DX = tools to manage it.
- **Ladder:**
  - 2 — One opaque artifact; no size visibility.
  - 4 — Size known only by inspecting files.
  - 6 — A size-measure command; basic minification.
  - 8 — Multiple build profiles (dev/debug/prod), tree-shaking/DCE, precompressed
    (br/gzip) output, measured raw + compressed sizes.
  - 10 — Above + a size-constrained profile (e.g. TinyGo/opt-z), code-splitting,
    size budgets/gates in CI, and honest documented size trade-offs.
- **Evidence:** measure raw + brotli/gzip of a representative app; try a
  size-optimized profile; is size gated in CI?
- **Red flags:** shipping the raw uncompressed artifact; no size visibility;
  hidden size regressions.

### E2. Deployment & release tooling
- **Definition:** Producing and shipping a production build.
- **Why:** The last mile; friction here delays every release.
- **Ladder:**
  - 2 — Hand-assemble the deploy.
  - 4 — A documented manual sequence.
  - 6 — A `build` command producing a deployable bundle.
  - 8 — A `release` command: optimized build + compression + manifest + asset
    layout, host-agnostic.
  - 10 — Above + reproducible builds, release metadata/provenance, CI workflow
    templates, and a verify step before publish.
- **Evidence:** one-command release; inspect the manifest; reproduce the build.
- **Red flags:** snowflake deploys; no manifest; "works only from my laptop."

### E3. PWA / offline
- **Definition:** Service workers, caching strategy, installability, offline.
- **Why:** Increasingly expected; notoriously fiddly by hand.
- **Ladder:**
  - 2 — None.
  - 4 — Manual service-worker wiring.
  - 6 — SW registration helper.
  - 8 — Cache-storage plans, installability observation, offline read support.
  - 10 — Above + offline mutation queue/replay, update-prompt flow, and tested
    offline scenarios.
- **Evidence:** install the app, go offline, perform a mutation, reconnect, watch
  replay.
- **Red flags:** stale-cache lock-in with no update path; offline that loses
  writes.

---

## Phase F — Longevity & ecosystem

### F1. Documentation
- **Definition:** Reference, guides, conceptual docs, and their findability.
- **Why:** Docs are the framework's user interface for everything not in code.
- **Ladder:**
  - 2 — README only / out of date.
  - 4 — API list, no guides.
  - 6 — Reference + getting-started.
  - 8 — Layered docs (getting-started → guides → reference → design notes),
    role-based entry points, examples linked to APIs.
  - 10 — Above + searchable, versioned, drift-checked against code, an API
    browser, and an error-code index — docs that *can't* silently lie.
- **Evidence:** find how to do 3 real tasks from a cold start; check a doc claim
  against current code; is there an API browser + error index?
- **Red flags:** docs describing removed APIs; no conceptual layer; code samples
  that don't compile.

### F2. Examples & recipes
- **Definition:** Runnable examples covering real features and combinations.
- **Why:** Developers learn by copying working code more than by reading prose.
- **Ladder:**
  - 2 — None.
  - 4 — One toy example.
  - 6 — A handful of feature examples.
  - 8 — A broad catalog (state, forms, routing, async, SSR, hydration,
    diagnostics) mapped to APIs, browsable locally.
  - 10 — Above + CI-verified examples (they build/test green), a showcase app
    that dogfoods every API, and an API→example index.
- **Evidence:** browse the catalog; confirm examples build; is there an
  API-to-example map; does a flagship app use every feature?
- **Red flags:** stale examples; features with zero examples; a showcase that
  avoids the hard APIs.

### F3. Versioning, stability & migration
- **Definition:** SemVer discipline, deprecation policy, migration tooling.
- **Why:** Upgrade cost determines whether teams stay current or freeze.
- **Ladder:**
  - 2 — Breaking changes any release, no notice.
  - 4 — A changelog, no policy.
  - 6 — SemVer + changelog.
  - 8 — Documented stability tiers (stable/experimental), deprecation windows,
    migration notes per breaking change.
  - 10 — Above + codemods/automated migrations, API-baseline tests that catch
    accidental breaks, and a published support window.
- **Evidence:** read the stability policy; is there an API-baseline test; are
  experimental surfaces labeled; migration guide quality.
- **Red flags:** silent breaking changes; experimental APIs presented as stable;
  no changelog discipline.

### F4. Extensibility & plugins
- **Definition:** Adding capabilities without forking; hook points; escape hatches.
- **Why:** No framework covers everything; the escape hatch *is* the ceiling.
- **Ladder:**
  - 2 — Closed; fork to extend.
  - 4 — Undocumented internals you can poke.
  - 6 — Some public extension points.
  - 8 — A plugin model with manifests, capability-checked registration, and
    documented subsystem hooks.
  - 10 — Above + extensions compose safely, can't silently break core invariants,
    and escape hatches exist at every layer (you're never trapped).
- **Evidence:** write a plugin/extension; reach a lower layer when the high-level
  API doesn't fit.
- **Red flags:** no escape hatch (capability ceiling); plugins that monkey-patch
  internals; extension breaks on minor upgrade.

### F5. Interoperability
- **Definition:** Talking to the host platform and external libraries.
- **Why:** Real apps integrate; isolation is a dead end.
- **Ladder:**
  - 2 — Sealed; no host/library access.
  - 4 — Raw, untyped host bridge.
  - 6 — Typed bridges for common host APIs.
  - 8 — Typed interop for storage, clipboard, events, observers, cross-tab,
    lazy module loading; clear native vs. browser boundaries.
  - 10 — Above + server-component/native boundaries, third-party lib integration
    patterns, and interop that degrades gracefully off-platform.
- **Evidence:** call a host API (storage, clipboard, observer) with types;
  integrate an external lib; check native vs browser stubs.
- **Red flags:** stringly-typed host calls; no story for existing JS/native libs;
  interop that panics on the server.

### F6. Internationalization (i18n / l10n)
- **Definition:** Locale handling, pluralization, formatting, SSR alignment.
- **Why:** Retrofitting i18n is brutal; default support is a major DX multiplier.
- **Ladder:**
  - 2 — None.
  - 4 — Manual string maps.
  - 6 — A locale provider + lookup.
  - 8 — Pluralization, number/date formatting, locale switching.
  - 10 — Above + SSR-aligned locale bootstrap (no hydration mismatch), typed
    message keys, and lazy locale loading.
- **Evidence:** switch locale at runtime; SSR a localized page and hydrate without
  mismatch; pluralization correctness.
- **Red flags:** locale mismatch between server and client; untyped string keys;
  formatting hand-rolled.

### F7. Community, support & governance
- **Definition:** Ecosystem size, responsiveness, contribution path, hiring pool.
- **Why:** When docs fail, humans are the fallback; also a hiring/longevity risk.
- **Ladder:**
  - 2 — Single maintainer, no responses, no contrib guide.
  - 4 — Sparse activity; issues languish.
  - 6 — A contributing guide; some responsiveness.
  - 8 — Active maintenance, clear contribution + governance, a security reporting
    path, healthy issue triage.
  - 10 — Above + sizable ecosystem (third-party components, answers), predictable
    releases, and a public roadmap.
- **Evidence:** issue response times; CONTRIBUTING/SECURITY presence; third-party
  package count; roadmap visibility.
- **Red flags:** bus-factor 1; dead issues; no disclosure path; thin hiring pool
  (state this honestly — it's a real adoption cost).

### F8. Learning curve & conceptual coherence
- **Definition:** How few, how orthogonal, and how transferable the core concepts
  are.
- **Why:** A small coherent mental model beats a large feature list for DX.
- **Ladder:**
  - 2 — Incoherent; every feature its own paradigm.
  - 4 — Steep; lots of special cases.
  - 6 — Learnable with effort; some surprising interactions.
  - 8 — A small set of orthogonal concepts that compose predictably; transferable
    knowledge (e.g. React mental model carries over).
  - 10 — The model is so coherent that learning one part predicts the rest, and
    advanced use is the same concepts taken further (no "second framework" to
    learn for SSR/state/etc.).
- **Evidence:** count core concepts; does SSR/state/routing reuse the same model
  or introduce new ones; can a developer predict unfamiliar APIs?
- **Red flags:** "now learn the SSR way / the state way / the routing way";
  concepts that interact in surprising ways; leaky abstractions.

### F9. Maintenance & dependency footprint
- **Definition:** Long-term cost: dependency tree, toolchain churn, build
  reproducibility, single-language surface.
- **Why:** DX over years is dominated by *not* fighting the environment.
- **Ladder:**
  - 2 — Sprawling, unpinned deps; frequent breakage.
  - 4 — Large dep tree; manual pinning.
  - 6 — Pinned deps; occasional churn.
  - 8 — Minimal/declared deps, single toolchain, reproducible builds, vendored
    externals.
  - 10 — Above + one-language stack (no parallel ecosystem to maintain), shared
    types server↔client, and upgrades that don't cascade.
- **Evidence:** count direct deps; is the toolchain single or split; are externals
  vendored/pinned; does an upgrade cascade?
- **Red flags:** npm-style transitive sprawl; two ecosystems (back/front) to keep
  in sync; unpinned tool payloads.

---

## Weighting (tune per project)

Not all dimensions matter equally for every team. Suggested default weights
(1 = nice-to-have, 3 = critical). Adjust before computing a composite and record
the weights used so comparisons stay honest.

| Weight 3 (critical) | Weight 2 (important) | Weight 1 (contextual) |
|---|---|---|
| A2 first-render, B1 component model, B2 API consistency, B3 type safety, C1 hot reload, C3 error messages, D1 testing, F1 documentation, F8 conceptual coherence | A1 install, A3 scaffolding, B4 styling, B5 state, B6 data, B7 routing, C2 build speed, C5 IDE, D6 SSR, E1 bundle size, F3 versioning, F4 extensibility | B8 forms, C4 debugging, D2 observability, D3 perf tooling, D4 a11y, D5 security, E2 deploy, E3 PWA, F2 examples, F5 interop, F6 i18n, F7 community, F9 maintenance |

Composite = Σ(score × weight) / Σ(10 × weight), reported as a percentage,
**always alongside the per-dimension table** (the composite hides cliffs — a
single 1–2 on a weight-3 dimension can sink real-world DX while the average still
looks fine).

## Scoring sheet template

Fill each cell with a 1–10 score per lens.

```
Dimension                          | First-hour | At-scale | Weight | Evidence
-----------------------------------|-----------|----------|--------|---------
A1 Installation & prerequisites    |           |          |        |
A2 Time-to-first-render            |           |          |        |
A3 Scaffolding & starters          |           |          |        |
B1 Component model                 |           |          |        |
B2 API consistency                 |           |          |        |
B3 Type safety                     |           |          |        |
B4 Styling                         |           |          |        |
B5 State management                |           |          |        |
B6 Data fetching & async           |           |          |        |
B7 Routing                         |           |          |        |
B8 Forms & validation              |           |          |        |
C1 Hot reload / inner loop         |           |          |        |
C2 Build speed                     |           |          |        |
C3 Error messages                  |           |          |        |
C4 Debugging                       |           |          |        |
C5 IDE integration                 |           |          |        |
D1 Testing                         |           |          |        |
D2 Observability & devtools        |           |          |        |
D3 Performance & profiling         |           |          |        |
D4 Accessibility                   |           |          |        |
D5 Security                        |           |          |        |
D6 SSR / hydration                 |           |          |        |
E1 Bundle size & output            |           |          |        |
E2 Deployment & release            |           |          |        |
E3 PWA / offline                   |           |          |        |
F1 Documentation                   |           |          |        |
F2 Examples & recipes              |           |          |        |
F3 Versioning & migration          |           |          |        |
F4 Extensibility & plugins         |           |          |        |
F5 Interoperability                |           |          |        |
F6 i18n                            |           |          |        |
F7 Community & governance          |           |          |        |
F8 Conceptual coherence            |           |          |        |
F9 Maintenance & deps              |           |          |        |
```

## Anti-gaming rules

To keep scores honest over time:

1. **Evidence or it didn't happen.** Every score links a command, file, or
   screenshot. No score from memory.
2. **Score the default path, not the expert path.** If reaching the 8 (Strong)
   rung requires tribal knowledge, cap it at 4 (Rudimentary / expert-only).
3. **The lower lens wins the headline.** If first-hour is 9 but at-scale is 3,
   the dimension's risk is 3 — note both, lead with the gap.
4. **No partial credit for "planned."** Roadmap items score ≤2 (Absent) until
   shipped and documented.
5. **Re-score, don't re-baseline.** Compare against the prior snapshot; a feature
   that regressed DX can lower a score even if the feature count grew.

## Platform-honest anchors (when the literal 10 needs an upstream change)

A few dimensions cannot reach the literal 10-rung through framework work alone —
the ceiling is the host platform (Go/wasm toolchain) or an emergent property
(community size). For these, score against a **platform-honest 10** and record the
residual gap explicitly so re-scores stay truthful rather than aspirational:

- **A2 Time-to-first-render:** Go module cold-fetch + full wasm compile dominate a
  clean-machine first run — an upstream toolchain cost. Platform-honest 10 =
  prebuilt `gwc` binary releases + documented warm-cache path + hot-reload-on
  idiomatic default starter + published cold/warm timings. Literal `<2 min` cold
  needs a CDN-cached starter / module proxy (the stretch route).
- **C2 Build speed:** Go/wasm has no incremental wasm linker (full-world link per
  build); prod code-splitting is a runtime HTTP mechanism, not a linker one.
  Platform-honest 10 = persistent build daemon keeping the cache hot + a per-build
  "what rebuilt & why" report + published, gated cold/warm times. Literal 10
  (incremental link) is an upstream Go ask.
- **C4 Debugging:** No production DWARF→browser-source-map tool exists for Go/wasm.
  Platform-honest 10 = shipped installable browser-extension panel (live tree,
  props/state, commit profiling) + snapshot time-travel + a documented
  stack↔Go-symbol correlation workaround. Native source maps are an upstream ask.
- **F7 Community:** The 10-rung's "sizable ecosystem + hiring pool" is emergent and
  outside a solo maintainer's control. Honest ceiling ≈ 7 (active maintenance,
  CONTRIBUTING/SECURITY, responsiveness, public roadmap, triage SLA). Do the
  enabling work; record the cap, don't claim 10.

Use a platform-honest anchor ONLY where a genuine upstream/structural blocker is
named — never as a way to excuse ordinary engineering gaps.

## Next steps for devx-maxxing

1. Score GoWebComponents against this rubric (first pass), evidence-linked.
2. Score 2–3 reference frameworks (React/Next, Solid, Svelte/Kit) on the same
   sheet for calibration.
3. Rank gaps by (target − current) × weight; feed the top items into `todos.md`.
4. Re-score each minor release; chart the deltas.

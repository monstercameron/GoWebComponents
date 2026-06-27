# devx-maxxing — Road to god-tier (DevX × Features)

Plan to lift every dimension toward the **10 (Exemplary)** rung of
[`DEVX_MAXXING.md`](./DEVX_MAXXING.md). Baseline: 2026-06-27 `cab0d205` snapshot in
[`SCORES.md`](./SCORES.md). **Planning artifact** — each item names the concrete
mechanism that closes the gap, the verification that proves it, and (where
relevant) the honest ceiling.

**Status: adversarial SIGN-OFF achieved (2026-06-27, plan v3).** Projected composite
if fully implemented: **99.5%** (637/640) — every DevX dimension at its honest maximum.
See [`SCORES.md`](./SCORES.md) for the refinement-loop history and watch-items.

## V4 implementation status (live)

Shipped so far on the `v4` branch (each an atomic, tested commit — native +, where
relevant, wasm coverage; `go vet` clean):

| Plan item | Shipped | Commit |
|---|---|---|
| **FC1 (frontier)** local-first sync | `localfirst` LWW-Register CRDT engine: `Clock`/`Replica` (optimistic+offline pending+convergent Merge)/`Authority`; offline→reconnect→converge proven in-process AND over the real serverfn transport | `feat(localfirst): convergent CRDT core` + `test(localfirst): converge over serverfn` |
| **FB1 (keystone)** server functions | `serverfn` runtime (`Handle`/`Call`, typed `*ServerError`, no-build-tag net/http so server=sockets, browser=fetch) + `gwc server gen` (`//gwc:server` → client stubs + `RegisterServerFunctions`); e2e: native server + wasm client compile, HTTP round-trip through generated registration | `feat(serverfn): server-function runtime` + `feat(gwc): \`gwc server gen\`` |
| **FA1** fine-grained signals | `state.Signal[T]` / `NewSignal` / `NewComputed` (+ `.Text` fine-grained binding) | `feat(state): fine-grained Signal[T]…` |
| **FB7** two-way bind | `html.BindTo` / `BindFunc` (structural `Binding`; binds Signals/atoms) | `feat(html): BindTo/BindFunc…` |
| **C3** diagnostics | hookcheck conditional-hook detection + symbol + specific remediation | `feat(hookcheck): detect conditional hooks…` |
| **FC2** zero-npm security | `gwc supplychain` (zero-npm proof, dep budget, checksum verify, SBOM-shaped JSON) | `feat(gwc): add \`gwc supplychain\`…` |
| **D5** vuln scanning | `gwc vuln` (govulncheck reachability scan: REACHABLE vs imported-only; `-strict`) | `feat(gwc): add \`gwc vuln\`…` |
| **FA3/A3/B1/D4** component registry | `gwc add` shadcn-model catalog (disclosure, tabs — WAI-ARIA, self-contained, native+wasm verified; templates are a compiled package) | `feat(gwc): add \`gwc add\`…` |
| **FB3** AI-native docs | `gwc llms` (llms.txt + llms-full.txt, `-check` staleness gate) | `feat(gwc): add \`gwc llms\`…` |
| **B8** shared validation | `validate` package (struct-tag, wasm+native, `Fields()`→`ui.FieldErrors`) + `Form.ValidateStruct()` | `feat(validate)…` + `feat(ui): Form.ValidateStruct` |
| **B7/FA4** typed routes | `gwc routes gen` typed `Link*` constructors from route contracts (+ `check` gate) | `feat(gwc): add \`gwc routes gen\`…` |
| **FA4/B3** typed search params | `router.DecodeQuery[T]`/`EncodeQuery` (`query:` tags → validated typed struct via `validate`); `validate` gains `omitempty` | `feat(router): typed, validated search params` + `feat(validate): omitempty` |
| **FA5/B1** transitions | `anim.DiffKeyedRects` (keyed-list FLIP classify+invert), `anim.Transition` (pure enter/exit state machine), `anim.StaggerDelay` | `feat(anim): keyed-list FLIP + enter/exit transitions` |
| **F6/B3** typed i18n keys | `gwc i18n gen` typed message accessors from a base-locale bundle (namespace+key+`{param}` all compile-checked) (+ `check` gate) | `feat(gwc): add \`gwc i18n gen\`…` |
| **FA2/B6** query layer | `query` package: `Fetch` (request dedupe), `SWR`, `Mutate` (optimistic + rollback), invalidate-by-key/prefix, injectable clock; `ui.UseQuery`/`UseMutation` hooks (wasm e2e through real render path) | `feat(query): stable query/data layer…` + `feat(ui): UseQuery/UseMutation…` |
| groundwork | `ui.Run`+`interop.KeepAlive`; `gwc` singleton guard; `gwc check` hook-context analyzer; release go-get smoke fix | (4 commits) |

The reactivity, binding, and validation work also advance the meshed Part I dimensions
(B5/D3, B1/B2, B8/B3); the two `gwc` commands advance D5/F9 and F1/C3/C5.

## v4 — the combined plan: god-tier DevX × god-tier features

This revision merges the **feature/capability roadmap** from the competitive analysis
([`../competitive-analysis/`](../competitive-analysis/README.md), Vols I–III) into this
DevX plan, so both halves mesh into one target:

- **Part I — DevX dimensions → 10** (everything below the Part I header). Polishes every
  *existing* surface to the exemplary rung. The **quality** layer.
- **Part II — capability roadmap** (the new section). Adds the *features* engineers expect
  (parity), the killer features that earn evangelism (delight), and the frontier moves no
  competitor can follow (moat). The **capability** layer.

They are **not independent**. Every Part II feature also raises one or more Part I
dimensions — that is the mesh (see the capability↔dimension table in Part II, and the `→
meshes` / `extends` markers on each feature, which point back to existing items so the
two plans never duplicate or conflict). God-tier DevX on a framework that lacks the
capabilities is a flawless paint job on a car with no engine; god-tier capabilities with
mediocre DevX is an engine no one enjoys driving. The target is **both, woven together**.

> **North star:** every DevX dimension at its honest maximum (Part I) × the capabilities
> that make those surfaces matter *and* define a category no one else can enter (Part II).
> The keystone is **`//gwc:server`** (FB1) — the data, forms, security, and frontier work
> all build on it; the three category bets are **local-first sync (FC1)**, **zero-npm
> security (FC2)**, and **agent-native UI (FC3)**.

---

# Part I — DevX dimensions → 10

Legend: **[min→target]** lower-lens score → target. **(platform-cap)** = the true
10-rung needs an upstream Go/wasm toolchain change; a **platform-honest 10** is
defined and is the planning target. **(baseline-fix)** = baseline score was stale;
corrected here.

---

## Platform-honest anchors (agreed with reviewer)

Three dimensions cannot hit the literal 10-rung purely through framework work:

- **C2 Build speed (platform-cap):** Go/wasm does a full-world link every build;
  there is no incremental wasm linker and prod code-splitting is a *runtime HTTP*
  mechanism, not a linker one. **Platform-honest 10 = persistent build daemon
  keeping the Go build cache hot across saves + per-build "what rebuilt & why"
  report + published, CI-gated cold/warm times.** True-10 (incremental wasm link)
  is filed as an upstream Go ask, tracked, not blocking.
- **C4 Debugging (platform-cap):** No production DWARF→browser-source-map tool
  exists for Go/wasm; dev builds strip DWARF (`tools/livereload/livereload.go`
  ~1104) on purpose. **Platform-honest 10 = shipped, installable browser-extension
  panel (packaged + store/side-load) with live component tree, props/state, commit
  profiling; a snapshot step-back/time-travel replay engine; and a documented
  browser-stack↔Go-symbol correlation workaround (symbol-preserving build + panic
  stack mapper).** Native source maps filed upstream.
- **F7 Community (structural):** The rubric's 10-rung requires an emergent
  *sizable ecosystem + hiring pool* a solo maintainer cannot ship. **Honest
  target = 7** (active maintenance, CONTRIBUTING/SECURITY, responsiveness, public
  roadmap, triage SLA). We do the enabling work and **record the cap in SCORES.md
  rather than claim 10.**

A short note to this effect has been added to the rubric so re-scores stay honest.

## Baseline corrections (from review round 1)

- **D5 Security [8→9] (baseline-fix):** `release.yml` already runs **govulncheck +
  CycloneDX SBOM**. Baseline "no govulncheck/SBOM in CI" was stale. Rescore to 9.
  Remaining gap to 10: run govulncheck on **every PR**, not just release (item
  below).
- **F4 (baseline-fix):** `plugin/conformance/conformance.go` is already shipped;
  the remaining gap is docs + escape-hatch coverage, not infrastructure.

---

## Priority tier 1 — the cliffs

### F7 Community & governance **[3→7] (structural ceiling)**
- Public **roadmap** (GitHub Projects or `ROADMAP.md`), linked from README.
- **GitHub Discussions** on; document a triage SLA in `CONTRIBUTING.md` (label
  taxonomy, "good first issue", response-time target).
- `docs/adr/` architecture decision records + a maintainer-onboarding guide to
  lower bus-factor risk.
- Publish 2–3 first-party companion packages (component kit, charts) as separate
  repos to seed the third-party path.
- **Honest cap:** stop at 7; record in SCORES that 8–10 require real adoption
  outside the maintainer's control.

### C4 Debugging **[4→10 platform-honest] (platform-cap)**
- Ship the **installable browser-extension panel** as a packaged artifact
  (Chrome `.crx` via Web Store + Firefox `.xpi` via AMO, plus a one-command
  side-load). It consumes the existing `gwc.devtools.extension.v1` bridge payload
  (`devtools.BuildExtensionPanelPayload`) and shows live component tree +
  props/state + commit profiling. Apply the existing `devtools/trace_capture.go`
  **redaction at the bridge-payload projection** so the panel can't leak secrets.
- Build a **snapshot step-back replay engine** in `devtools` (record committed
  snapshots, scrub/step backward) — scoped as a new subsystem, not a doc.
- Document a **stack-correlation workaround** for default builds: symbol-preserving
  build flags + a panic-stack→Go-symbol mapper so browser stacks name Go funcs.
- **Upstream ask (tracked, non-blocking):** native Go/wasm source maps.

### C2 Build speed **[5→10 platform-honest] (platform-cap)**
- **Persistent build daemon** in `gwc dev`: keep the Go build cache hot across
  saves, watch + rebuild, and print a per-build **"what was rebuilt and why"**
  report (the 10-rung's explicit ask). **Mechanism:** derive the report from
  `go build -x`/`-debug-actiongraph` output (or a build-action manifest diff)
  rather than a bare list of changed files, so "why" is the actual cache-miss
  reason, not a vague file dump.
- Publish **cold/warm build-time benchmarks** in `docs/benchmarks/` (drift-guarded)
  and add a CI gate on warm-cycle regression.
- Drop the incorrect "extend prod route-chunks to the dev linker" claim (prod
  splitting is runtime HTTP import, not a linker artifact).
- **Upstream ask (tracked, non-blocking):** incremental wasm linking.

### F9 Maintenance & dependency footprint **[6→10]**
- **Module split — exact plan:** move `playwright-go`, `bubbletea`,
  `charmbracelet/*`, `anthropic-sdk-go`, `openai-go` into a separate **`tools`
  module** (own `go.mod`); relocate every importer (incl. `tools/gwc/main.go`'s
  playwright use, test harnesses) into that module. Library module retains only
  runtime deps (target **<6**: e.g. `gorilla/websocket`, a sqlite driver, `x/text`,
  `golang-jwt` — audit and justify each).
- Verify the `replace`-directive chain + `go mod tidy` so existing checkouts and
  `go get` for consumers don't break; document the migration steps (not just the
  end state). **Decide the version strategy up front:** if any library-consumer
  `require` line changes (it will, since transitive playwright/bubbletea deps
  disappear from consumers' `go.sum`), the split needs a `/v4` major bump; if it's
  a pure internal reorg with no public import-path change, it can land in a minor.
- Add a **dependency-budget CI check** failing if the library module's direct-dep
  count or transitive footprint grows.
- Document reproducible-build guidance (`go.mod` `toolchain` pin + vendored
  `third_party/`).

---

## Priority tier 2 — the 7s (and the platform reframes)

### A2 Time-to-first-render **[6→10 platform-honest]** (platform-cap)
- **Honest assessment:** `<2 min` on a *cold* machine is unreachable for Go/wasm
  (Go module cold-fetch + full wasm compile dominate — an upstream toolchain cost
  comparable to C2/C4). **A2 is therefore added to the rubric's platform-honest
  anchors.** Platform-honest 10 = prebuilt `gwc` binary releases (removes the
  `go run ./tools/gwc` compile cost) + documented warm-module-cache path + default
  starter is hot-reload-on + idiomatic + **published measured cold/warm timings**.
- **Route to a literal <2 min (stretch):** ship a CDN-cached starter tarball with a
  pre-populated module cache (or a framework module proxy) so even a cold machine
  clears <2 min. Tracked as the path to literal 10; the maintenance tradeoff (a
  baked artifact) is noted.

### A1 Installation & prerequisites **[7→10]**
- `gwc doctor --fix`: auto-apply (or one-key apply) the `Remediation` fields the
  doctor already emits.
- CI on **Win/macOS/Linux** proving cross-platform parity; document it.

### B2 API design & ergonomic consistency **[7→10]**
- Bless **`html/shorthand`** as the single authoring path in all docs/examples/
  starters; demote `html` to "stable builders" with a one-line rationale.
- **Before** any deprecation window closes, add a `gwc migrate` codemod for the
  `html` → `html/shorthand` import-path change (current migrate handles version
  renames, not surface changes).
- Add an API-consistency lint to `gwc lint`: enumerate the concrete rules
  (verbSubject naming, receiver/arg order, `Use*` hook prefix, option-struct vs
  variadic) and scope it to public packages to avoid internal false positives.

### B4 Styling **[7→10]**
- `css/u` **already re-exports the full `css` surface** (`css/u/exports.go`); the
  utility *names* are a curated layer over it. So the 10 gap is: **document the
  intentional utility subset + a completeness matrix + the raw-`css` escape** for
  anything outside it.
- Drop "dead-style report via static analysis" — the runtime registry model can't
  know never-registered classes without a wasm static pass. Instead: **prove the
  registry doesn't double-emit** (already largely true; add a test) and file
  build-time tree-shaking as a stretch/upstream item, not a 10-blocker.

### B6 Data fetching & async **[7→10]**
- Graduate **Experimental** cached-resource lifecycle surfaces to **Stable**,
  pinned by an **API-baseline test**.
- Verify + document request dedup; ship an optimistic-update-with-rollback example.

### B7 Routing **[7→10]**
- Graduate Experimental loader/guard surfaces to Stable (API-baseline pinned).
- **Typed-route codegen — concrete:** a `go generate`-invoked `gwc routes gen`
  reads the route-contract definitions and emits typed link constructors to a
  generated file; a CI step fails if the generated output is **stale** (regen +
  `git diff --exit-code`). This closes the last compile-time link-target gap that
  runtime `RouteContract` (`router/contracts.go`) can't.

### B8 Forms & validation **[7→10]**
- **Shared client/server validation — concrete:** define a **pure-Go struct-tag
  validator** (no syscall/net/file deps so it compiles to wasm *and* native);
  one validated struct flows through both an HTTP handler and `UseForm`. Stub any
  wasm-incompatible validators behind build tags. Add async validation + an example
  wiring the same struct on both sides (the rubric's exact 10-rung ask).

### C1 Hot reload **[7→10] (DX-negative fix)**
- **Keep state-preserving reload opt-in** (auto-injection silently restores stale
  state across schema changes → ghost bugs). Instead:
  - Make it **discoverable**: a dev-overlay hint + a `gwc dev --hotreload` flag.
  - When enabled, **detect state-schema change** and show a visible "state reset"
    indicator + warning rather than silently restoring into a mismatched type.
- Add the visible **reload-status indicator** to the dev overlay.
- Near-instant warm reload depends on C2's daemon.

### C5 IDE integration **[7→10]**
- Ship a **VS Code extension** scoped to concrete capabilities: commands
  (dev/build/test/lint), route/style/token completion, and `gwc lint --json`
  surfaced via `languages.createDiagnosticCollection` with quick-fixes; set
  `go.toolsEnvVars` for wasm gopls **via the Extension API without clobbering**
  user settings. GoDoc-sourced hover docs.
- **JetBrains plugin is a separate later deliverable** — not on the same timeline.

### D2 Observability & devtools **[7→10]**
- Ship the installable extension panel (shared with C4).
- **OTel — concrete:** route runtime spans through the existing `telemetry`
  package's **browser-fetch OTLP transport** to a collector; the extension panel
  reads via the `gwc.devtools.extension.v1` bridge. Wire the existing
  `scheduler.NewInstrumented(...).DevtoolsSection` into the panel for the 10-rung's
  scheduler-instrumentation ask. Redact at the bridge projection (see C4).
- Runnable end-to-end OTel example.

### D3 Performance & profiling **[7→10]**
- **Diagnose, don't re-baseline:** root-cause the
  `BenchmarkFineGrainedSelectorDashboardReactiveTextUpdate16-20` 123% regression
  (114 regressions in `DRIFT_NOTE.md`) and fix the cause; then restore green drift.
- Add a CI **merge gate**: a `gwc bench` step that exits non-zero on drift beyond
  tolerance (today the drift note exists but no workflow blocks on it).

### D5 Security **[9→10] (baseline-fix)**
- govulncheck + CycloneDX SBOM already run on release. To reach 10: run
  **govulncheck on every PR** (not just release) as a merge gate.

### D6 SSR / hydration **[7→10]**
- Graduate Experimental loader/guard + bootstrap-transport surfaces to Stable.
- Hydration-mismatch **debugging example**; assert-and-recover path so a mismatch
  never silently corrupts (documented + tested).

### E1 Bundle size & output **[7→10]**
- Generate a **CI size-budget gate into every starter** (default budget +
  `gwc wasm measure` check in the scaffold CI), not just the framework repo.
- Document the tinygo profile's API-compat boundary so the size-optimized path is
  a predictable first-class choice.

### E2 Deployment & release **[7→10] (clarify "reproducible")**
- Source-archive provenance (`git archive` + SHA-256 + `attest-build-provenance`)
  is **already shipped** → the deploy/provenance story is strong.
- **Decision (closed): target byte-identical binary reproducibility** (provenance
  is already shipped, so this is the only remaining gap to 10). Specify `-trimpath
  -buildvcs=false`, `SOURCE_DATE_EPOCH`, and a CI-pinned `toolchain`; verify
  byte-identical output from a clean checkout in CI.
- Ship verified **CI deploy templates** (static/CDN, Pages) with a deploy smoke test.

### E3 PWA / offline **[7→10]**
- **Zero-config service worker:** `gwc` emits the SW from the release manifest so
  it isn't hand-written app code.
- CI end-to-end offline scenario (install → offline → mutate → reconnect → replay).

### F3 Versioning & migration **[7→10] (commit the numbers)**
- Publish a concrete policy: **deprecated APIs removed after 2 minor releases;
  12-month support window after a major.**
- **Update `api-stability-and-support-policy.md`** (and its mirror under
  `examples/public-examples-site/assets/docs/`) to state the 12-month post-major
  support window — the shipped doc today commits only "≥2 minor + 90 days" and
  disclaims LTS, so the doc and policy must be reconciled, not just asserted here.
- Expand `gwc migrate` codemods to cover every breaking change; make the API
  compat guard a **merge gate**.

### F4 Extensibility & plugins **[7→10]**
- Enumerate the **layer model** explicitly and document a named public escape at
  each: CSS sink (`css.SetSink`), render tree/reconciler, router, state atoms, SSR
  transport, `interop` raw-browser, `html.RawHTML`. Document the **sealed
  `pluginruntime` kernel as a hard ceiling**.
- Conformance kit is shipped; add docs + examples that reach the lowest public
  layer when the high-level API doesn't fit.

### F5 Interoperability **[7→10]**
- Ship a **third-party-JS integration example** (wrap an npm lib via `ImportModule`
  + typed bridge).
- Native-stub parity test proving every interop surface degrades gracefully
  off-platform.

### F6 i18n **[7→10]**
- **Typed message keys — concrete:** `gwc i18n gen` reads locale bundles and emits
  typed key constants/accessor; CI staleness check (regen + `git diff`).
- **Lazy locale loading:** verify whether the runtime already supports it; then
  document (if present) or build (if not) + ship an example.

### F8 Conceptual coherence **[8→10]**
- Resolve the two-HTML-surface friction (shared with B2); document the wasm/native
  build-tag split as one coherent contributor mental model in design notes.

---

## Priority tier 3 — the 9s, close the last point

### D1 Testing **[9→10]**
- Make the **headless component lane the documented default** (no browser needed).
- Enumerate and ship the missing **first-class fixtures**: async-boundary,
  suspense, hydration-boundary, and error-boundary test fixtures. These **must be
  native-testable (no browser)** to satisfy the 10-rung's fast-headless condition;
  any fixture that is genuinely wasm-only requires an explicit architectural note
  saying why.

### F1 Documentation **[9→10]**
- **Compile-test inline code samples:** extract fenced `go` blocks and compile them
  in CI (doc-sample test harness) so samples can't silently rot.
- Add full-text search + versioned docs site. Fix or explicitly scope the TinyGo
  doclint exception (`commands.go` ~332) so TinyGo doc claims are verifiable.

### F2 Examples & recipes **[8→10] (concrete actions, not just "protect")**
- Enumerate which of the 95+ examples lack CI coverage; bring all into a CI lane.
- Build a machine-verified **API→example index** (distinct from the capability
  matrix).
- Verify a showcase app **dogfoods every public API**, or document the covered
  subset honestly.

---

## Priority tier 3b — close the remaining 8s to 10

These sat at 8 with strong evidence and earlier had no items — the simulated
re-scan projected 94.8% precisely because of them. Each now gets a concrete item.

### A3 Scaffolding & starters **[8→10]**
- Add **composable sub-generators** beyond `gwc start`: `gwc add route` and
  `gwc add component` that emit idiomatic code into an existing app.
- CI verifies generator output **builds, tests green, and matches current docs
  idioms** (regenerate + compare drift check).

### B1 Component model & composition **[8→10]**
- Publish a **uniform-composition guarantee**: one worked guide/example proving the
  same rules hold from a leaf button → container → full app, with **no special
  cases** (keyed lists, context, slots, fragments, boundaries all one model).
- Document the wasm/native build-tag split as a **platform-adapter** concern that
  does not leak into the component model (shared with F8), so advanced use needs no
  second paradigm.

### B3 Type safety **[8→10]**
- Clarify + prove **shared-types-by-construction**: the *same* Go structs are the
  server, client, and validation types (the build-tag split is platform adapters,
  not domain types). Ship an example where **one struct** drives SSR render, client
  render, and `UseForm` validation, pinned by an API-baseline test.
- Typed-route codegen (B7) + typed i18n keys (F6) remove the last stringly-typed
  surfaces, so a typo anywhere is a compile error.

### B5 State management **[8→10]**
- **Extend the existing "Choosing The Smallest Owner" section** (ch.06) into a full
  state-tool decision guide: exact conditions favoring `UseState` vs `UseReducer` vs
  context vs `state` atoms vs derived/computed vs **SQLite/KV** (the existing
  section omits the SQLite/KV tier), each with an example.
- Make **fine-grained reactivity** the discoverable, documented default where
  over-render matters — not an opt-in buried in the API.

### C3 Error messages & diagnostics **[8→10]**
- **Real engineering required (the analyzer does NOT already do this):**
  `tools/hookcheck/hookcheck.go` today detects only **loop-site** hook violations,
  and the non-panic `Diagnostic` struct carries **no symbol/source-line field**
  (only the panic path has a `TopFrame`). To reach the 10-rung's "name the
  offending symbol + propose the exact fix":
  1. **Extend `hookcheck`** to also detect **conditionally-called** hooks (if/switch
     branches), not just loops.
  2. **Add `Symbol string` + `SourceLocation token.Position`** fields to the
     `Diagnostic` struct and plumb them through for every code with an AST-level
     origin (e.g., "`UseState` called conditionally at `MyComponent` L42").
  3. Make each code's `Remediation` the **specific corrective action** (with the
     symbol), not a category-level hint.
- **Verification:** a test asserting symbol + location presence on the AST-origin
  codes. Codes that are genuinely runtime-only (no AST origin) keep the panic-path
  `TopFrame` and are documented as such — the 10-rung is met for the AST-origin set
  and honestly scoped for the rest.

### D4 Accessibility **[8→10]**
- Add **focus management on route change** as a built-in default (focus the new
  view's heading/landmark), not an app-wired concern.
- Add an **a11y lint** to `gwc lint` (or wire axe rules into the native component
  lane) for common violations beyond the browser-only axe-core gate, so issues
  surface pre-browser.

---

## Cross-cutting enablers (build first — each moves several dimensions)

1. **Installable browser-extension devtools panel** → C4, D2.
2. **Module split (runtime vs. tooling)** → F9; clarifies F4/F5 boundaries.
3. **VS Code extension** → C5; helps A2/C1 onboarding.
4. **Dev-loop build daemon + build report** → C2, C1.
5. **Pure-Go shared validation schema** → B8; one source of truth helps F9.
6. **Experimental→Stable graduation + API-baseline pins** → B6, B7, D6, F3.
7. **Codegen pair: typed routes + typed i18n keys** → B7, F6, deeper B3 safety.
8. **Starter-embedded CI gates (size, perf, a11y, deploy smoke, govulncheck)** →
   E1, D3, E2, E3, D5.

---

---

# Part II — Capability roadmap (god-tier features)

Part I makes the *surfaces* exemplary; Part II adds what those surfaces *do*. Organized by
the three competitive-analysis volumes (parity → delight → moat). Each item names the
mechanism, the **DevX dimensions it also lifts** (`→ meshes`), and verification. Items
that **extend an existing Part I item** are marked so the two plans don't duplicate or
conflict — where Part I already started the work, Part II points at it and goes further
rather than re-specifying it.

## Tier F-A — Parity capabilities (table stakes the market already pays for)

### FA1 — Fine-grained signals as a first-class primitive → meshes **B5, D3**
- Ship `state.Signal[T]` / `state.Computed[T]` (TC39-shaped) that auto-track inside
  computed/regions, so a component body runs **once** and only signal-bound nodes
  re-render. **Extends B5** ("make fine-grained reactivity the discoverable default"): B5
  documents it; FA1 makes it a real primitive, not an opt-in optimization.
- Double win on wasm — fewer `syscall/js` boundary crossings → directly improves **D3**.
- Verify: a render-count test proving no component re-exec on a leaf-signal update + a
  bench showing fewer DOM ops vs the re-render path.

### FA2 — Stable query/data layer (SWR + dedupe + optimistic/rollback) → meshes **B6**
- **Extends B6.** Graduate the cache to Stable (already a B6 ask) *and* add window-focus/
  reconnect refetch, request dedupe, invalidate-by-key, and one-line optimistic mutations
  wired to the **existing durable offline queue** (a differentiator no JS query lib ships).
- Verify: dedupe + SWR + rollback integration tests; the offline-queue example (shared
  with E3/FC1).

### FA3 — `gwc add`: headless component registry (shadcn model) → meshes **A3, B1, D4**
- A curated catalog of a11y-correct, `css/u`-styled Go components copied into the user's
  repo via the CLI (`gwc add dialog|combobox|tabs|popover|...`), behavior ported from
  Radix/Ark **state machines**. **Extends A3** (`gwc add component` generator ships the
  mechanism; FA3 ships the *catalog* on top). Lifts **B1** (compound-component/slots proof
  point) and **D4** (a11y primitives shipped, not hand-rolled).
- Verify: each component carries native + browser a11y tests; `gwc add` smoke in CI.

### FA4 — Typed routing + typed search params → meshes **B7, B3**
- **This is B7's typed-route codegen, extended** with typed, *validated* search params
  (the TanStack-Router feature engineers rave about). Same `gwc routes gen` + staleness
  gate. Removes the last stringly-typed query surface → deepens **B3** ("a typo anywhere
  is a compile error").

### FA5 — Animation & transitions system → meshes **B1** (new surface)
- Built-in enter/exit transitions, **FLIP** for keyed lists, signal-driven (FA1) spring/
  tween motion, and first-class **View Transitions API** for route changes.
- Verify: transition fixtures in native + browser lanes; reduced-motion honored by default.

### FA6 — `ui.Defer` deferrable views (lazy-load wasm per view) → meshes **E1, A2, C2**
- Angular-`@defer`-style blocks (viewport / idle / interaction / timer triggers;
  placeholder/loading/error sub-blocks) that lazy-load **wasm chunks** + per-route
  splitting. Turns the bundle-size weakness into a feature: ship only above-the-fold
  interactivity. Directly attacks **E1** and improves first interaction (**A2**); the
  split manifest feeds **C2**'s build report.
- Verify: a split-bundle size budget gate in the starter CI (shared with E1).

## Tier F-B — Killer DevX features (the delight layer)

### FB1 — `//gwc:server`: isomorphic server functions (THE keystone) → meshes **B3, B6, B8, D5, F5**
- Annotated Go func → auto client stub + server route + **the same Go structs both
  sides** (no codegen, no router, no schema tax). The build **strips the server body and
  secrets from the wasm bundle**.
- The keystone almost everything else builds on: **B6** data, **B8** server-action forms,
  **D5** security (server-only code compile-eliminated from the client — a whole bug class
  gone; `gwc check` flags any leak), **F5** interop, plus the frontier (FC1/FC3/FC5).
  Reuses **B8**'s pure-Go shared validator for input validation.
- Verify: a leak test (analyzer flags server-only imports reaching client code) + a
  shared-struct request/response round-trip test.

### FB2 — Optimistic UI + Actions one-liners → meshes **B6, B8**
- `ui.UseOptimistic` + `ui.UseAction` + form pending/error state, on top of FB1.
  **Extends B6** (which already asks for an optimistic example) into first-class hooks.

### FB3 — AI-native DevX → meshes **F1, C3, C5**
- Auto-generate `llms.txt` / `llms-full.txt` / `skill.md` from the capability matrix +
  markdown doc delivery (content negotiation); a **`gwc mcp` docs+API+analyzer server**;
  and **`gwc check --fix` as the canonical agent post-edit hook** plus an `AGENTS.md`
  ruleset. **Extends F1** with AI-readability and turns **C3** diagnostics + **C5** IDE
  integration into *agent guardrails*. GWC's typed/explicit/analyzable design is
  accidentally perfect here — market it: "the framework your AI gets right the first time."
- Verify: llms.txt generation test; MCP-server smoke; docs-markdown negotiation test.

### FB4 — Elm-grade errors + in-page overlay → meshes **C3, C4**
- Push every framework error to plain-English cause + source frame + "did you mean" +
  copy-pasteable fix + docs anchor; render a Vite-grade in-page **error overlay** in dev.
- **Extends C3** (the diagnostic symbol/location plumbing) and **C4** (the overlay is the
  panel's dev-surface).

### FB5 — `gwc workbench` (Storybook + stories-as-tests) → meshes **D1, A3**
- Isolated component dev + a stories format whose stories *are* the wasm/browser tests
  (built on the existing testkit + SSR). **Extends D1** — stories become the authoring UI
  for D1's missing first-class fixtures (async/suspense/hydration/error boundaries).

### FB6 — Snapshot time-travel devtools → meshes **C4, D2**
- **This is C4's "snapshot step-back replay engine," surfaced in the panel** as
  undo/scrub. One subsystem, two consumers — reconcile with C4, don't build twice.

### FB7 — Micro-DX ergonomics → meshes **B1, B2**
- `ui.Inspect` (Svelte `$inspect`), two-way `h.Bind` (kills the `value=`+`onInput=parse`
  boilerplate), named **slots/snippets**, and `Show`/`Switch`/`Index` control flow. Each
  small; collectively the daily-delight layer. **Extends B2** (ergonomic consistency) and
  **B1** (slots/compound composition).

## Tier F-C — Frontier category moves (the moat — where GWC stops being "a Go React")

### FC1 — Built-in local-first sync engine → meshes **B5, B6, E3, D6**
- Server-authoritative Go ↔ client Go SQLite, query-driven "shapes" (`//gwc:sync`),
  optimistic + offline + real-time — built on **FB1** + the *already-shipped* `db/sqlite`,
  `kvstate` (conflict resolution + cross-tab sync), and `fetch.OpenMutationQueue`. GWC is
  ~80% of the way there; this finishes it. Lifts **B5** (the SQLite/KV state tier), **B6**
  (live queries), **E3** (offline replay is the same engine), **D6** (SSR seeds from the
  synced store). Rides "the year of the sync engine" (Zero 1.0 / Electric / TanStack DB).
- Verify: an offline → mutate → reconnect → converge integration test (shared with E3).

### FC2 — Zero-npm as a security product → meshes **D5, F9**
- `gwc audit` (prove zero-npm, emit the CycloneDX SBOM already wired in `release.yml`,
  verify module checksums) + capability sandboxing of third-party components. **Extends
  F9** (the runtime/tooling module split) and **D5** (security) into a *marketed* supply-
  chain-immunity story — structurally immune to the 2026 npm attack wave (Axios, Shai-
  Hulud, the 42-package TanStack compromise). The strongest claim is also true: there's no
  npm to compromise.
- Verify: `gwc audit` CI gate (shared with F9's dependency-budget check) + a
  capability-violation test.

### FC3 — Agent-native / generative-UI runtime → meshes **D2, F4**
- A typed "renderable schema" agents emit (server-side via FB1), validated against the
  component allow-list by **`gwc check`**, then streamed + rendered natively. Expose the
  component catalog + analyzer as **MCP tools**, built on the **already-shipped agent
  runtime bridge** (WS+MCP). GWC's typed component model *is* the "allow-listed components,
  not raw code" safety property the whole A2UI/MCP-UI space converged on. Lifts **F4**
  (the public extensibility layers) + **D2** (observe/devtools the agent session).
- Verify: agent emits a tree → validated → rendered, end-to-end test.

### FC4 — Ride the wasm platform leap → meshes **E1, C2, F5**
- Bless the **TinyGo leaf-app profile** (2.4 MB → ~200 KB) with a compatibility lint
  (**extends E1**'s tinygo compat-boundary doc); track **WasmGC** (Wasm 3.0) as the
  structural size fix; adopt the **component model** (WASI 0.2 / TinyGo wasip2) for
  polyglot + edge (feeds **F5** interop and FC5).
- Verify: a tinygo size budget; a WIT-component smoke when the toolchain is ready.

### FC5 — One-binary, whole-stack, edge-portable → meshes **E2, A2**
- `gwc build --single-binary` → one static artifact serving SSR + wasm + the `//gwc:server`
  API + the sync engine; edge SSR via WASI runtimes (FC4). **Extends E2** (deployment): the
  ops dream only a Go full-stack framework can offer — `scp` one binary, run it.
- Verify: the single binary serves a full-stack app in a CI smoke; a copy-and-run deploy test.

### FC6 — Multiplayer / collaboration → meshes **FC1**
- `sync.Presence` / `sync.UseCursors` + CRDT-backed docs (Automerge-Go on the server
  authority), on top of FC1 + `UseWebSocket`. Falls out of FC1 nearly for free.
- Verify: a two-client presence + concurrent-edit merge test.

## Capability ↔ DevX-dimension mesh (how Part II raises Part I)

| Feature | Lifts these Part I dimensions | Relationship |
|---|---|---|
| FA1 signals | B5, D3 | extends B5 |
| FA2 query layer | B6 | extends B6 |
| FA3 component registry | A3, B1, D4 | extends A3 |
| FA4 typed routing + search params | B7, B3 | extends B7 |
| FA5 animation | B1 | new surface |
| FA6 `ui.Defer` | E1, A2, C2 | new surface |
| **FB1 `//gwc:server` (keystone)** | **B3, B6, B8, D5, F5** | new; uses B8 validator |
| FB2 optimistic/actions | B6, B8 | extends B6 |
| FB3 AI-native DevX | F1, C3, C5 | extends F1 |
| FB4 Elm-grade errors | C3, C4 | extends C3 |
| FB5 workbench | D1, A3 | extends D1 |
| FB6 time-travel | C4, D2 | **= C4 replay engine** |
| FB7 micro-DX | B1, B2 | extends B2 |
| FC1 local-first sync | B5, B6, E3, D6 | new; uses shipped db/kvstate/queue |
| FC2 zero-npm security | D5, F9 | extends F9 |
| FC3 agent-native UI | D2, F4 | new; uses agent bridge |
| FC4 wasm platform | E1, C2, F5 | extends E1 |
| FC5 one-binary | E2, A2 | extends E2 |

**Build order (keystone-first):** **FB1** → FA1, FA2 → FA3, FA4 → FB3 → **FC1, FC2, FC3**
→ the rest. FB1 unblocks the data/forms/security/frontier chain; FC1+FC2+FC3 are the three
category bets. This order is also a clean merge into the Part I **cross-cutting enablers**
list above — FB1 sits alongside enabler #6 (Experimental→Stable graduation), FA3/FB5
alongside #3 (VS Code) and A3 generators, FC2 alongside #2 (module split), FA1/FA2/FA4
alongside #7 (codegen pair).

---

## Honest final ceiling (planning-level)

This now has two axes: **DevX dimensions** (Part I) and **capabilities** (Part II).

**Part I (DevX):** with every item above executed (including tier 3b), the planning-level
scores are:

- **30 dimensions at literal 10.**
- **3 at a platform-honest 10** — A2, C2, C4 (literal 10 needs upstream Go/wasm
  toolchain changes; the platform-honest anchor and the stretch route to literal
  10 are recorded for each).
- **1 at its structural ceiling, F7 = 7** (a sizable ecosystem + hiring pool is
  emergent and outside a solo maintainer's control; the enabling work is done and
  the cap is recorded, not papered over).

That is the DevX target: **every dimension at its honest maximum.** The only gaps to a
literal all-10 composite are three upstream toolchain asks (A2 cold module-fetch, C2
incremental wasm link, C4 wasm source maps) and one emergent-community constraint — all
explicitly named, none hidden. No dimension is left silently below target.

**Part II (capabilities):** with the capability roadmap executed, GWC moves from "a Go
React with exemplary DevX" to a framework that (a) has the **parity** features the market
expects (signals, query layer, component registry, typed routing, animation), (b) ships
the **delight** features that earn evangelism (`//gwc:server`, AI-native DevX, optimistic
UI, workbench, Elm-grade errors), and (c) **owns three frontier categories** the JS
ecosystem is structurally barred from entering (local-first sync, zero-npm security,
agent-native UI). Because every Part II feature is wired to the Part I dimensions it
lifts, the two plans are one: the capabilities are not bolted on — they are the *reason*
the polished surfaces matter.

**Combined god-tier definition:** every DevX dimension at its honest maximum **×** the
capability set that makes those surfaces matter and defines a category no competitor can
enter. A framework that is both the most *pleasant* to use (Part I) and the most
*capable* and *uncopyable* (Part II) — keystoned by `//gwc:server`, moated by FC1/FC2/FC3.
That is the god-tier framework, not just god-tier DevX.

> **Honest scope note:** Part I is an adversarially-signed-off plan against a concrete
> rubric with verification per item. Part II is a *capability roadmap* synthesized from
> the competitive analysis (Vols I–III) — each item names a mechanism + verification, but
> the larger frontier bets (FC1/FC3/FC5) are multi-quarter efforts whose effort/risk is
> recorded in the source volumes. The mesh is real; the sequencing is keystone-first.

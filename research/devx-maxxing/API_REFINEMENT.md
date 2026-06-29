# API Surface Refinement — GoWebComponents

Big-picture audit of the **current public API** (distinct from the [DEVX_MAXXING](./DEVX_MAXXING.md)
rubric scoring). Goal: find concrete, actionable refinements — inconsistencies, ergonomic friction,
missing features, footguns, and duplication — across the core packages
(ui, html, html/shorthand, state, router, fetch, query, serverfn, css/u, validate, i18n, interop).

Sourced from a sequential adversarial API-design review (2026-06-28). **Findings are leads, not
verdicts** — each is verified against the code before any change. Status column tracks that.

**Landed 2026-06-28 — 9 additive, non-breaking refinements, each adversarially reviewed.** The
review loop caught a REAL bug in 3 of them (all fixed): #6 permanent-suspension on fetcher panic,
#4 nil-`*StatusError` panic, #17 custom-rule panic crashing Struct.
- #3 `ui.UseQuery(..., deps ...any)` (stale-closure fix)
- #13 `router.Query`/`SearchParams` `Int`/`Bool`
- #2 `GoEvent` coords + modifier accessors (wasm)
- #10 `DOMRef.Blur`/`Click`/`ScrollIntoView` (new runtime capability interfaces)
- #19 `i18n.Runtime.NS(ns)` namespace handle
- #6 `ui.UseSuspenseQuery` (first real consumer of Await/SuspendUntil; hardened the suspense path)
- #4 `serverfn.StatusError` + constructors (4xx taxonomy; round-trips as `ServerError.Status`)
- #17 `validate.RegisterRule` (custom tag rules; panic-contained)
- #18 `validate.Check` + `CrossRule`/`Fail` (cross-field validation merged with tags)

All built native+wasm, tested (wasm tests via `node wasm_exec_node.js`), gofmt-clean, API baselines
regenerated where exports changed. Verification also corrected 2 audit errors (#7 name-clash, #2
wasm-only). **The additive runway is now exhausted.** Remaining are direction/BREAKING calls
deferred to the maintainer (#5 typed AtomKey — biggest DevX win, #1 distinct event types, #8
Child/Children, #15/#16 deprecations) + 2 minor items (#11 `UseForm.SetField` dev-panic, #20
`Hydrate`/`HydrateInto` internal dedup; #14 `UseDerived` auto-scope is behavioral).

## Findings (ranked by impact × tractability)

| # | pkg | finding | type | proposed change | breaking | status |
|---|-----|---------|------|-----------------|----------|--------|
| 1 | ui | event type aliases all = `runtime.GoEvent`; `func(KeyboardEvent)` ≡ `func(MouseEvent)` (no compile distinction) | footgun | distinct typed wrappers w/ event-specific accessors | yes(minor) | **SKIPPED — trap.** The useful half (event accessors) shipped in #2. The type-distinction half gives ~nothing: dispatch is `interface{}`/reflection-based (`UseEvent(any)`, `OnClick ui.Handler`), so there's no Props-level signature to compile-check, and making the types distinct structs would BREAK the reflection dispatch (it builds a GoEvent) for a marginal gain. Not worth the invasive rework. |
| 2 | ui | `GoEvent` lacks mouse coords + modifier-key accessors | missing | add accessors to `GoEvent` (wasm-only type) | no | **DONE** (internal/runtime/events_pointer.go: GetClientX/Y, PageX/Y, OffsetX/Y, Button(−1 absent)/Buttons, Shift/Ctrl/Alt/MetaKey; wasm test via node — PASS) |
| 3 | ui | `UseQuery` re-subs only on `key`; stale-closure fetcher footgun | footgun | add `deps ...any` variadic to dep list | no | **DONE** (ui/query_hook.go; source-compatible variadic + doc example) |
| 4 | serverfn | `Handle` maps every error → HTTP 500; no 4xx path | missing | typed `StatusError` mapped to status code | no | **DONE** (serverfn/serverfn.go: `StatusError` + `NewStatusError`/`BadRequest`/`Unauthorized`/`Forbidden`/`NotFound`/`Conflict`/`UnprocessableEntity`; `statusForError` via `errors.As` (chain-aware) → `Handle`. Status round-trips to client as `ServerError.Status` (already wired). Review caught a nil-`*StatusError` panic (errors.As matches-but-nil → bare `Error()` deref) — fixed: nil-safe `Error()` + `statusForError` 500-on-nil. 6 tests: direct 404, wrapped 409 (+msg-no-leak), plain→500, zero-status→500, nil-typed→500-no-panic. API baseline updated; ch07 docs. Additive/opt-in.) |
| 5 | state | atom IDs are global raw strings; silent cross-pkg collision + type-mismatch zero reads | footgun | typed `AtomKey[T]` package vars | ~~yes~~ **NO** | **DONE — delivered ADDITIVELY** (state/atom_key.go: `AtomKey[T]` + `NewAtomKey(id,initial)` + `UseAtomKey(key)` hook + `key.Global()`/`ID()`/`Default()`). User greenlit the breaking version, but 48 files use `UseAtom` + downstream breakage — the typed-key win (declare id+type+default once, compile-checked key names, type-consistency across Use/Global) is fully achievable additively, so I did that instead (the better call). `UseAtom(string,T)` stays for local/simple use. Reviewer ✅ (seeding no-clobber/G39 holds, non-breaking). 5 tests incl. G39 pre-render-write + same-id-share + ch06 docs (subsection + decision-guide row). Centralizes contract, does NOT namespace ids; same-id-different-type + pointer-T-Default aliasing documented.) |
| 6 | ui | no data-fetching suspense path (`UseQuery` forces status switch) | missing | `UseSuspenseQuery[T]` over `SuspendUntil` | no | **DONE** (ui/query_hook.go: box+goroutine, reads only mutex-guarded Snapshot; serves settled success/error immediately, suspends Idle/Loading. First real consumer of Await/SuspendUntil. Review found+fixed: Bug1 deferred close/rerender on fetcher panic; Bug2 `query.runFetch` panic→error recovery. 5 tests incl. resume-after-fetch + error→ErrorBoundary; ch04 docs. NOTE: `-race` unavailable on win/arm64 — race-freedom reasoned (cache-mutex synchronizes all result reads).) |
| 7 | ui | `ErrorBoundary` is a `var` (CreateElement) while peers are funcs | inconsistency | ~~add `func ErrorBoundary`~~ — **NOT a quick win**: Go forbids a func with the same name as the existing `var ErrorBoundary` (ui.go:205). Needs a renamed helper (`ErrorBoundaryEl`/`NewErrorBoundary`) → a naming decision, not additive. | no | NEEDS-DECISION |
| 8 | ui | `Child Node` + `Children []Node` duality in props → reflection extraction everywhere | ergonomic | standardize on `Children []Node` (+ adapter) | ~~yes~~ **NO** | **DONE — additive.** Removing `Child` breaks 32 files; instead extracted the duplicated ~50-line extraction (ErrorBoundary + context Provider were near-identical) into ONE shared `coalesceChildren` (ui/children_shared.go); both helpers now delegate. Kills the boilerplate (the actual pain) non-breakingly. +`TestCoalesceChildrenUnifiedForms`; existing extractor tests pass. |
| 9 | ui | `UseTask` hardcodes `context.Background()` | ergonomic | accept a parent `context.Context` | ~~yes~~ **NO** | **DONE — additive.** Body extracted to `useTaskWithParent(parent, run)`; `UseTask(run)` delegates with `Background` (behavior unchanged — existing `TestUseTaskTransitions...` passes); new `UseTaskCtx(parent, run)` derives the task context from `parent` (nil→Background) so a request/deadline/value context propagates + cancelling the parent cancels the task. wasm test `TestUseTaskCtxParentCancellationPropagates` passes via node. The other WithCancel (UseLazyNode) untouched. |
| 10 | ui | `DOMRef` exposes only `Focus()` | missing | add `Blur/ScrollIntoView/Click` | no | **DONE** (runtime Blurrer/Clicker/ScrollIntoViewer caps + WASMDOMNode impls + DOMRef wrappers; native dispatch test PASS) |
| 11 | ui | `UseForm.SetField(string,…)` stringly-typed, silent miss | footgun | typo-catching helpers | no | **DONE** (ui/form.go + form_native.go: `MustSetField(name,value)` panics on unknown-field/type-mismatch (loud vs silent no-op); `HasField(name) bool` to assert names. Both wasm+native. Tested BOTH lanes: native `TestFormHasField`/`TestFormMustSetField` (all branches + both panics + no-mutation), wasm `TestFormHasFieldWasm` (node, live render). ch11 docs. Review-hardened: native `HasField` `Lock`→`RLock` (read-only); `readNamedField` `CanInterface()` guard so an unexported-but-existing field name returns false instead of panicking (both files); +tests for the guard, zero-form contract, wasm `MustSetField` happy-path; regenerated `api_compat_guard` baseline (additions-only). Full typed setters = codegen, deferred.) |
| 12 | html | `Props.Value=""` / `TabIndex=0` silently omitted | footgun | `*string`/`*int` or sentinel; or lint | yes | **DOC-FIX, not `*string`.** Verified: `html.Value("")` (PropOption) and `Raw["value"]=""` ALREADY force an intentional empty value, and the Props struct comment documents it — the escape hatch exists. `*string` would be breaking + awkward in Go (`&"literal"` is illegal) + redundant. Real gap is discoverability → ch05 doc note added. |
| 13 | router | `Query` lacks `Int/Bool` (Params has them) | inconsistency | add `Query.Int/Bool` | no | **DONE** (router/router_api.go: Query+SearchParams Int/Bool; wasm test) |
| 14 | state | `UseSelector` auto-scopes id; `UseDerived` global-keys (collides) | inconsistency | auto-scope UseDerived or mandate prefix | no | **DECLINED — contradicts design.** ch06 documents `UseDerived` as the SHARED-by-id primitive ("several components need the same read-only derived shared value") — the global key IS the intended sharing mechanism. Auto-scoping would break its documented purpose. The "collision" is the same intentional id-sharing as `UseAtom`; the typed-key answer to it is `AtomKey` (#5). Same audit misread as #15/#16. |
| 15 | state | `state.UseComputed` ≈ `ui.UseMemo` (dup concept) | duplication | deprecate `UseComputed` → `UseMemo` | no | **DECLINED** — verified: ch06 lists `UseComputed` as a *Stable, first-class primitive* ("a typed render-time derived value") alongside UseDerived/UseSelector. Deprecating contradicts the framework's documented design on one reviewer's say-so. Not a unilateral call. |
| 16 | fetch | `UseFetch` (untyped) superseded by `UseResource[T]` | duplication | deprecate `UseFetch` | no | **DECLINED** — verified: `UseFetch(url)→untyped Resource` vs `UseResource[T](loader)→typed` are different ergonomics (URL-string quick GET vs typed loader), not strict duplication. Deprecation is a maintainer direction call; not done unilaterally. |
| 17 | validate | no custom-rule extension point | missing | `RegisterRule(name, fn)` | no | **DONE** (validate/validate.go: `RuleFunc(value any, arg string)(ok,msg)` + concurrency-safe `RegisterRule`; dispatched in applyRule's `default` for non-built-in names. Guards: blank/nil/built-in-name ignored (can't shadow core rules); unknown-unregistered still ignored (forward-compat). Review-hardened: a panicking RuleFunc is CONTAINED (`callRuleFunc` recover → field fails "validation rule panicked", never crashes Struct/500s). 7 tests (pass/fail+arg, can't-shadow, forward-compat, non-string field, replace, panic-contained, omitempty-skip, guards) + baseline regen + ch11 docs.) |
| 18 | validate | no cross-field validation | missing | `validate.Func[T]` composing w/ tags | no | **DONE** (validate/validate.go: generic `Check[T](value, ...CrossRule[T])` runs `Struct` tags + cross-field closures merged into one `Result`; `CrossRule[T] func(T) *FieldError`, `Fail(field,msg)` helper. Nil rules skipped, panicking rule contained (`callCrossRule`). 4 tests (tag+cross merge, same-field conflict Fields-vs-All, multiple cross failures, nil-skip+panic-contain) + baseline regen + ch11 docs. Reviewer ✅ no bugs.) |
| 19 | i18n | `T(namespace, key, args)` positional; forgot-namespace = silent miss | ergonomic | `Runtime.NS(ns).T(key,…)` handle | no | **DONE** (i18n/i18n.go: `Namespace` type + `Runtime.NS`/`Namespace.T`/`Name`; test PASS) |
| 20 | ui | `Hydrate`/`HydrateInto` share ~200 lines | duplication | extract `hydrateWithOptions` | no(internal) | **DONE** (ui/ui.go: `hydrateWithBootstrap(opts, bridgeFn, mountFn)` holds the common bootstrap/observability/seed/atom/strict logic; `Hydrate`/`HydrateInto` are 3-line wrappers differing only in bridge + mount. Reviewer ✅ behavior-preserving: exact step order, observer/bridge identity, both error paths preserved. native+wasm hydration tests pass.) |

## Missing-feature gaps worth noting
- **`useImperativeHandle` equivalent — ABSENT.** No typed imperative handle from child → parent ref.
- **Named slots — ABSENT** (explicit props only; acknowledged in `ui/doc.go`).
- **Component display names / dev labels — ABSENT.**
- **StrictMode double-invocation — ABSENT.**
- **First-class event delegation — ABSENT** (per-element handlers only).
- **List virtualization — ABSENT** (defer/lazy-node primitives exist; no windowing helper).
- **`MapWithKey(items, keyFn, render)` helper — ABSENT** (only the `Props.Key` field).

## Larger design questions (need a real decision)
1. **Atom IDs: typed `AtomKey[T]` vs global strings** — highest-leverage state change; breaking (Jotai-vs-Recoil precedent favors typed).
2. **Event types: distinct wrappers vs aliases** — enables handler type-checking + idiomatic accessors; breaks every handler signature → needs a shim period.
3. **`Child`/`Children` unification** — simplifies internal extraction; breaks 3 props structs.
4. **`UseSuspenseQuery` (the React `use(promise)` / Solid `createResource` shape)** — all pieces exist (SWR + SuspendUntil + AsyncBoundary), not yet composited.
5. **`serverfn` error taxonomy / status codes** — close the loop with the existing `ServerActionOutcome` taxonomy.

## Genuinely strong (do not churn)
query/cache layer · serverfn shape (single Handle/Call, shared types, httptest-able) · `UseForm[T]`
(touched/dirty/async-validation/submit-lifecycle) · css (typed rules → hashed class → SSR/wasm parity
+ hardening) · a11y primitives (focus trap/manager/composite-nav/announcer) · `GlobalAtom` ·
`router.DecodeQuery[T]` round-trip · SSR island / parallel hydration.

---

## Round 2 — second-tier packages (2026-06-28)

A second audit pass over the packages round 1 skimmed. Confirmed `anim`, `events`, `flags`,
`agentui`, `localfirst`, `devtools` are **already clean** (no findings). Shipped 8 refinements:

| # | pkg | refinement | type | status |
|---|-----|-----------|------|--------|
| R2-3 | css | `VarLength`/`VarDuration`/`VarAngle`/`VarNumber` — typed custom-property refs (Var was Color-only, so `W(Var(...))` wouldn't compile) | additive | **DONE** (+test, native+wasm) |
| R2-1 | css | `Animation(Length,string)` → `Animation(Duration,Easing)` — match `Transition`'s typing | breaking (0 callers) | **DONE** (fixed in place; test updated) |
| R2-9 | timetravel | `History.Snapshots() []Snapshot[T]` — read the whole timeline without moving the cursor | additive | **DONE** (+baseline regen) |
| R2-4 | telemetry | `ExportOTLPHTTP(ctx, …)` — was unbounded; a stuck collector blocked forever | additive* (0 callers) | **DONE** (NewRequestWithContext) |
| R2-5 | interop | `WorkerPool.Size()`/`QueueLimit()` — drop the `Get` prefix nothing else uses; `Get*` deprecated | additive | **DONE** |
| R2-6 | interop | `WaitResult` type + `WaitOK`/`WaitNotEqual`/`WaitTimedOut` — `WaitInt32` returned magic strings | source-compat | **DONE** (literal compares still work) |
| R2-10 | pwa | `ServiceWorkerType*`/`UpdateViaCache*` string constants — options were bare strings | additive | **DONE** |
| R2-2 | kvstate | `HydrateLazy` was a silent no-op vs its doc ("defers until first read") | bug→doc-honesty | **DONE** (doc corrected: it's "skip eager load"; true lazy-on-read not implementable in the async-engine model — stated) |

**Declined (round-2):**
- **R2-7** css `Opacity(float64)`/`OpacityNum(Number)` dup → deprecating is a direction call; dual float64/typed API is plausibly deliberate. Low value.
- **R2-8** pwa `ServiceWorkerSubscription`/`InstallabilitySubscription` dup → merging two 1-field PUBLIC types is an API change for negligible gain.
- **R2-11** head `UseHead(Document)` client hook → genuine missing feature, but applying `<meta>` elements is client-DOM work that can't be verified here (node has no real DOM; same runtime-block as Playwright items). Real follow-up, not shipped unverified.

---

## Round 3 — third-tier packages (2026-06-28)

Audited the packages neither prior round touched. **Findings now sparse (as expected at this
maturity):** confirmed `sanitize`, `textutil`, `plugin`, `servercomponents`, `hotreload`, `head`,
`scheduler`, `db/sqlite`, `agentbridge`, `security` (docs-only) **already clean**. Shipped 3:

| # | pkg | refinement | type | status |
|---|-----|-----------|------|--------|
| R3-1 | logging | `Logger.With(args...) Logger` — attach base fields (component/req-id) once instead of repeating per call; chained, per-call overrides; mirrors slog/zerolog/zap | additive | **DONE** (+test; WithContext carries base args) |
| R3-2 | prerender | `Export` returned a ZEROED summary on mid-loop error — callers couldn't find already-written files to clean up. Now returns the partial summary | additive (footgun fix) | **DONE** (+test) |
| R3-3 | a11y | `Listbox` never emitted `aria-multiselectable` — WAI-ARIA 1.2 §5.5 gap for multi-select. Added `ListboxProps.MultiSelect` → `aria-multiselectable="true"` | additive (ARIA compliance) | **DONE** (+test; single-select omits it) |

No api-baseline regen needed (these packages aren't gated). **Not worth it (declined, won't re-raise):**
sanitize `dropWithContents` customization (hard security boundary), textutil more case-helpers
(intentionally minimal), db/sqlite pragma "injection" (developer-supplied), plugin validator panic
recovery (app-owned code, unlike agentbridge's untrusted-agent handlers), agentbridge connection-state
observable (no consumer), scheduler latency in devtools summary (already in Snapshot).

**Audit status: three passes complete.** Core + mid + third tier reviewed; findings dropped from
20 → 11 → 3. The public API is now comprehensively covered.

---

## Table-clearing pass (2026-06-28) — Playwright set up + remaining items resolved

Owner directed: set up Playwright + clear every remaining table item. **Playwright confirmed working
in-environment** (existing e2e lane builds wasm fixtures + drives real Chromium; verified via the
dom-ref e2e). Outcome:

| Item | Resolution |
|------|-----------|
| R2-7 `Opacity(float64)` | **Deprecated** → `OpacityNum(Num(n))`; fixed the one internal caller (`u.Opacity`) |
| #16 `UseFetch` | **Deprecated** → `UseResource[T]`/`ui.UseQuery` |
| #15 `UseComputed` | **Deprecated** → `ui.UseMemo`; ch06 Stable-list + decision-guide updated |
| #7 `ErrorBoundary` func form | **Added `ui.NewErrorBoundary(props)`** (func entry mirroring AsyncBoundary; var name was taken). +test, api-compat baseline regen |
| R2-8 pwa `*Subscription` dup | **Unified** via `pwa.Subscription` + the two names as type aliases (non-breaking) |
| #14 `UseDerived` scoping | **Doc-clarified**: the global id is the intentional shared-by-id design (vs UseSelector's scoped projection); auto-scoping would break it |
| #12 `Props.Value=""` | Already resolved (ch05 doc-fix; escape hatch `html.Value("")` exists; `*string` is breaking+awkward+redundant) |
| #11 `head.UseHead` | **SHIPPED + Playwright-verified** — `head/use_head_wasm.go` applies title/description/canonical/OG/twitter to the live document via syscall/js (native no-op stub); `examples/public/use-head` fixture + `TestUseHeadE2E` (real Chromium: title override + meta + live re-apply on page change) PASS |
| #1 distinct event types | **DECLINED (firm).** 240+ handler sites use the aliases; making them distinct requires reworking the core reflection dispatch (GoUseFunc) for a marginal gain (accessors shipped in #2; UseEvent/OnClick are untyped, so no compile-check at the wiring boundary). Net-negative — would destabilize the most-depended-on subsystem. Would only do as a dedicated staged effort, not a "clear the table" item. |

**Net: 8/9 table items cleared (the 9th, #1, declined on evidence). Playwright is now a working
verification lane** — `head.UseHead` and any future DOM-level feature can be browser-verified here.

---

## Polish pass — 50 cleaning/polishing issues (2026-06-28)

Broadened scan (single sequential sonnet discovery agent) across BOTH core public APIs and
DevX/tooling surfaces, finer-grained than the 3 leverage-ranked passes above. **All 50 fixed**,
ordered quick-wins first; native+wasm build clean; unit tests pass (only the network-dependent
`TestDefaultStarterTemplatesScaffoldTidyTestAndBuild` fails — it `go mod tidy`s a temp scaffold
needing `playwright-go`, no network in sandbox). `anim` apidump baseline regenerated for the new
`EasingFunc` alias.

Breaking renames (#5 GetOutlet, #6 GetCurrentRouterPath, #12 WhenDisabled, #42 Build*Envelope) were
done ADDITIVELY: new canonical name + deprecated alias delegating to it (no removals). #24 anim/css
`Easing` collision resolved with an `anim.EasingFunc = anim.Easing` alias, not a rename.

Core API (#1–25): doc comments on PersistentSnapshotOptions, ComputedSignal multi-source caveat,
Spring zero-value div-by-zero, RouteContract zero-value, Envelope wire fields, 19 devtools types +
Severity/Classification/LogLevel; String() on ShadowToken & BreakerState; Topic() on TopicHandle;
events counter → atomic.Uint64; itoa → strconv.FormatUint; validateParams redundant-branch cleanup;
fetch Options/Result header-type asymmetry documented; AttrSel doc-name fix; fetch/doc.go +
state/doc.go stop promoting deprecated symbols; agentui DefaultLimits + RegisterAgentCommand footgun
docs; UseOutlet/GetCurrentPath/Disabled/New*Envelope additive renames.

Tooling (#26–50): help-text `?`, cross-platform .bat/.sh hints, gwc routes parse-skip stderr
warning, cssgen + snapshot-diff error/field docs, `gwc scaffold hook -type` flag, inspect.go
build*→parse* param-convention rename (233 ids, scoped regex), test/render & test/hooks package
docs, FirstChild/LastChild doc split, WithQueuedScheduler doc, and reference-manual ch.6/7/16
deprecation markers + testkit→test facade index fix.

# devx-maxxing — Feature & UX-Perfection Audit (aggressive)

- **Date:** 2026-06-27 · **Commit:** `1af9f6cc` (`v4`)
- **Method:** adversarial sonnet auditor + per-feature deep-dive sweep, verifying
  each of the 19 Part-II capabilities against the ACTUAL code (packages, exported
  signatures, `_test.go` presence, CI wiring), not the plan's "shipped" table.
- **Companion to:** [`SCORES.md`](./SCORES.md) (Review 2 = 80.8%) and
  [`IMPROVEMENT_PLAN.md`](./IMPROVEMENT_PLAN.md) Part II.

> **UPDATE — Review 3 (2026-06-27 @ `3c94979e`):** after 51 fix commits,
> **7/19 features are now genuinely UX-perfect** — FA4, FB2, FB4, FB7, FC2, FC4, FC5.
> The other 12 are improved but still shipped-but-rough (see `SCORES.md` R3 snapshot
> for the precise residual gaps). Composite rose 80.8% → 82.0%. The verdicts below are
> the ORIGINAL Review-2 findings; Part III of the plan tracks the fixes + what remains.
>
> **UPDATE — Review 4 (2026-06-28 @ `d2e234e0`):** after the v6 Part-IV round (18 commits),
> **18/19 features are now UX-perfect.** The 3 Tier-0 scope-reduced "fixes" were redone
> properly (FB3 real remediations, FB1 transitive graph, FC6 RGA text CRDT). Only **FB1**
> stays ⚠️ (3rd-party server-lib detection limited to a 17-entry curated list). Composite
> 82.0% → **86.9%**. See `SCORES.md` R4 snapshot + Part IV of the plan.

## Bottom line

**Of 19 capability features: 0 are UX-perfect.** Every feature ships a real,
tested *core primitive* — and almost every one is missing the **last mile**: the
ergonomic hook, the wired-in default, the CI gate, the loading/error state, or the
integration into the surface a developer actually touches. The pattern is
consistent enough to name: **library primitives shipped, product polish didn't.**

- **0 UX-perfect**
- **13 shipped-but-rough** (core works; critical spec clauses unmet)
- **3 spec-inflated** (FA6, FC2, FC5 — central claim missing under a "shipped" label)
- **3 missing/tracking-only** (FC4 wasm-platform, FC6 collaboration, FB6 browser extension)

**The single most damaging gap a new adopter hits first:** `gwc check --fix` does
not exist — yet the framework's own `AGENTS.md` + FB3 narrative position it as THE
canonical agent post-edit hook. An AI agent told to run `gwc check --fix` after
generating code gets an error. The framework's AI story breaks on its own first
instruction.

## Feature verdict table

✅ done · ⚠️ partial/rough · ❌ missing. "UX-perfect?" judged against each feature's own spec.

| Feature | Shipped | Complete per spec | UX-perfect | Verify test | The exact gap |
|---|---|---|---|---|---|
| FA1 signals | ✅ | ⚠️ | ⚠️ | ⚠️ | `Signal`/`Computed`/`.Text` + render-count proof all real. But `NewComputed` is *static* tracking (must pass sources); `.Text` on a computed binds only `sourceIDs[0]` (silent missed-update footgun); bench has no enforced assertion; non-comparable signal values always re-notify. |
| FA2 query layer | ✅ | ⚠️ | ⚠️ | ⚠️ | SWR+dedupe+optimistic+rollback+injectable-clock all real and tested. But **window-focus/reconnect refetch is absent** (only an advisory doc comment); **offline queue lives in `fetch/`, not wired to `query`**; no combined dedupe+SWR+rollback integration test; no async `UseMutation`. |
| FA3 `gwc add` | ✅ | ⚠️ | ⚠️ | ⚠️ | Copy-into-repo mechanism + provenance + compile-safe templates real. But **catalog = 2 components** (disclosure, tabs) for a "shadcn model"; **no native a11y render tests** for the catalog templates; no browser axe test; no dedicated `gwc add` CI smoke; tabs arrow-keys are manual. |
| FA4 typed routing+search | ✅ | ⚠️ | ⚠️ | ⚠️ | `gwc routes gen` typed Links + `DecodeQuery`/`EncodeQuery` validated round-trip real & tested. But **CI staleness gate not wired** (`routes check`/`i18n check` in no workflow); never run against a real package (no `routes_gen.go` exists); `EncodeQuery(any)` not generic (silent empty on misuse). |
| FA5 animation | ✅ | ⚠️ | ⚠️ | ⚠️ | Spring+FLIP+enter/exit SM+stagger+ViewTransition+easing all real. But **reduced-motion NOT honored by default** (`UseSpring`/`ViewTransition` animate regardless; caller must gate); **View Transitions not auto-wired into router**; no `UseTween`; browser tests only for spring (none for enter/exit or FLIP). |
| FA6 `ui.Defer` | ✅ | ❌ | ❌ | ⚠️ | **Severely under-spec.** `Defer(shown bool, placeholder, content)` is a boolean gate. **0 of 4 triggers** built in (no `UseIdle`/`UseInteraction`/`UseTimerTrigger`); **no loading/error sub-blocks** (1 placeholder slot vs 3+); **wasm-chunk lazy-load absent and not documented as capped**; doc comment references a non-existent `UseInViewport`. Angular `@defer` parity <½. |
| FB1 `//gwc:server` (keystone) | ✅ | ⚠️ | ⚠️ | ⚠️ | Runtime + `gwc server gen` + shared structs + round-trip tests real & excellent. But the **import-graph leak analyzer is NOT in `gwc check`** — it's an AST heuristic in `gwc doctor -audit` (rule `audit.state_boundaries`), not a `go/analysis` pass; the spec's "`gwc check` flags any leak" is unwired. No wasm-compile smoke for the generated client stub. |
| FB2 optimistic/actions | ⚠️ | ❌ | ❌ | ⚠️ | **`ui.UseOptimistic` and `ui.UseAction` DO NOT EXIST** (research docs only). `query.MutateAsync` is real & tested, and `UseForm` has rich pending/error state, but `ui.UseMutation` wraps the *blocking* `Mutate`. The "one-liner" is a manual `MutateAsync`+`UseForceUpdate` wiring exercise. |
| FB3 AI-native DevX | ⚠️ | ❌ | ❌ | ⚠️ | `gwc mcp` server + `gwc llms`(+`--check`) + `AGENTS.md` real. But **`gwc check --fix` ABSENT** (the canonical agent hook); **no MCP server smoke test**; **no HTTP content negotiation** (llms is file-only); agentui catalog not an MCP tool. |
| FB4 Elm-grade errors | ✅ | ⚠️ | ⚠️ | ⚠️ | `erroroverlay` + `hookcheck` conditional detection real. But **hookcheck not wired into `gwc lint --json`** (separate path, no Symbol field) → the VS Code extension never gets symbol-named fixes; **"did you mean" only for CLI typos**, not diagnostics; field names diverge from spec (`Hook`/`Pos`/`Func` ≠ `Symbol`/`SourceLocation`). |
| FB5 workbench | ✅ | ⚠️ | ⚠️ | ⚠️ | `RunStories` headless runner + gallery real & tested. But **all 4 first-class boundary fixtures absent** (async/suspense/hydration/error) — the spec's stated reason for the feature; stories cover only button+card. |
| FB6 time-travel | ✅ | ⚠️ | ❌ | ⚠️ | `timetravel.History[T]` engine (undo/redo/scrub/ring-evict) is COMPLETE & tested; in-page `devpanel` real. But the **installable browser extension is entirely absent** — `devtools/extension_bridge.go` is manifest *data structures* only; no `.crx`/`.xpi`, no `chrome.devtools.panels`, no store/side-load. The C4 platform-honest-10 deliverable does not exist. |
| FB7 micro-DX | ✅ | ⚠️ | ⚠️ | ✅ | `UseInspect`, `BindTo`/`BindFunc`, named slots, `Show`/`Switch` all real & well-tested. **`Index` (Solid-style position-stable list) is the one absent piece** of the spec's trio. |
| FC1 local-first sync | ✅ | ⚠️ | ⚠️ | ✅ | LWW-CRDT (Clock/Replica/Authority) + offline→reconnect→converge over real serverfn transport + PresenceSet + decision guide: all real, thoroughly tested, genuinely strong. Gap: **LWW only** (no op-based/Automerge merge — concurrent same-field edits silently last-write-wins); spec API names (`sync.Presence`/`UseCursors`) diverge from shipped `localfirst.PresenceSet`. |
| FC2 zero-npm security | ⚠️ | ⚠️ | ⚠️ | ❌ | `gwc supplychain` (zero-npm proof, dep budget, checksum count, SBOM-shaped JSON) real & tested. But **not in any CI workflow** (the gate that enforces zero-npm never runs); **`gwc audit` is spec fiction** (the command is `supplychain`); **capability sandboxing absent**; no capability-violation test. |
| FC3 agent-native UI | ✅ | ⚠️ | ⚠️ | ✅ | `agentui` allow-list Registry/Validate/Render + e2e (JSON→validate→render→DOM) real, tested, well-designed. But **`gwc check` does NOT validate agentui schemas** (no import link); **catalog not exposed as an MCP tool**; **not built on `agentbridge`** (fully decoupled); no streaming. |
| FC4 wasm platform leap | ⚠️ | ❌ | ❌ | ❌ | **Mostly a tracking note dressed as shipped.** TinyGo profile exists, but **`GOOS=wasip1` compile unverified** (claimed in CHANGELOG, no test/CI); no TinyGo size budget (200 KB claim unproven); no compat lint; **0 `.wit` files**; WasmGC/component-model is future. |
| FC5 one-binary whole-stack | ✅ | ⚠️ | ⚠️ | ⚠️ | `wholestack.Handler`/`ListenAndServe` real & unit-tested (serves SSR+wasm+serverfn from one handler). But **`gwc build --single-binary` CLI flag absent**; no CI full-stack smoke; no copy-and-run deploy test. The library primitive exists; the "one command, scp one binary" story doesn't. |
| FC6 multiplayer | ⚠️ | ❌ | ❌ | ⚠️ | **Presence-only, not collaboration.** `PresenceSet` (TTL heartbeat + facepile) real & tested. But **`sync.UseCursors` absent** (cursor = opaque JSON string); **no CRDT-backed shared docs / Automerge-Go**; concurrent field edits are last-write-wins. A Figma/collab-editor use case cannot be built on this. |

## Not UX-perfect — the punch list

### Critical (breaks the feature's headline promise)
1. **FB3 — `gwc check --fix` does not exist.** The framework's `AGENTS.md` + AI-native narrative tell agents to run it; it errors. Fix: implement `--fix` applying `gwc fmt` + remediations; make it the documented post-edit hook.
2. **FB6/C4 — "installable browser extension" is nonexistent.** Only manifest data structures. Fix: build a real web extension consuming the `gwc.devtools.extension.v1` bridge (live tree + props/state + commit profiling), packaged + side-loadable.
3. **FA6 — `ui.Defer` is a boolean gate, not Angular `@defer`.** Fix: add trigger hooks (`UseIdle`/`UseInteraction`/`UseTimerTrigger`), loading + error sub-blocks, and document the wasm-chunk limitation at the API.
4. **FC4 — wasm-platform "shipped" claim is a research note.** Fix: reclassify as tracked; add a real `GOOS=wasip1` compile CI step + TinyGo size budget + compat lint before calling it shipped.
5. **FC6 — "collaboration" is presence-only.** Fix: implement op-based CRDT merge (or Automerge-Go) + a typed cursor/selection type before claiming multiplayer editing.

### Major (core works, a load-bearing clause is missing)
6. **FB2 — `ui.UseOptimistic`/`ui.UseAction` absent.** The "one-liner" requires manual `MutateAsync`+`UseForceUpdate`. Fix: ship both hooks + an async `UseMutation`.
7. **FB1 — leak analyzer not in `gwc check`.** It lives in `gwc doctor -audit` as a heuristic. Fix: a real `go/analysis` import-graph pass wired into `gwc check`.
8. **FA2 — no focus/reconnect refetch; offline queue not bridged.** Fix: built-in focus/online listeners in `SWR`/`UseQuery`; a `UseDurableMutation` bridging `MutateAsync`↔`fetch.MutationQueue`.
9. **FA5 — reduced-motion not honored by default; View Transitions not auto-wired.** Fix: `UseSpring`/`ViewTransition` internally consult `UsePrefersReducedMotion`; router invokes `ViewTransition` on navigation.
10. **FC2 — `gwc supplychain` not in CI; capability sandboxing absent.** Fix: add it as a merge gate; implement (or descope) sandboxing + a violation test.
11. **FB4 — hookcheck symbol diagnostics not in `gwc lint`/VS Code.** Fix: wire `hookcheck` into the `gwc lint --json` path (add a Symbol field); add "did you mean" to diagnostic messages.
12. **FA4 — CI staleness gates unwired** for `routes check` + `i18n check`. Fix: add a per-PR step; ship a real `routes_gen.go` example.
13. **FC3 — `gwc check` doesn't validate agentui; catalog not an MCP tool; not on agentbridge.** Fix: wire `DefaultRegistry().Validate` into `gwc check`; expose the catalog as an MCP tool.
14. **FC5 — `gwc build --single-binary` flag + CI smoke absent.** Fix: add the flag (embed wasm in a Go server binary) + a copy-and-run deploy test.

### Minor (polish / coverage gaps)
15. **FB7 — `Index` control-flow helper missing** (Show/Switch shipped).
16. **FA3 — 2-component catalog; no native a11y tests for catalog templates; no `gwc add` CI smoke.**
17. **FA1 — `.Text` on multi-source computed silently misses updates from non-primary sources; bench unasserted.**
18. **FB5 — boundary fixtures (async/suspense/hydration/error) absent from workbench.**
19. **FA5/FA3 — browser-lane tests missing** for enter/exit + FLIP, and for `gwc add` components.
20. **FB3 — MCP server smoke test + HTTP markdown content negotiation absent.**
21. **FC1/FC6 — spec API names (`sync.Presence`/`UseCursors`) diverge from shipped `localfirst.*`.**

## Missing entirely (no implementation)
`gwc check --fix` · installable browser extension · `ui.UseOptimistic`/`ui.UseAction` ·
FA6 trigger hooks + loading/error sub-blocks + wasm-chunk loading · import-graph leak
analyzer in `gwc check` · Automerge/op-based CRDT · `sync.UseCursors` typed cursors ·
`GOOS=wasip1` CI verification · TinyGo size budget + compat lint · `.wit`/component-model ·
`gwc supplychain` CI gate · capability sandboxing + violation test · `gwc build
--single-binary` flag + CI smoke · `Index` helper · MCP server smoke test · llms-over-HTTP ·
browser a11y tests for `gwc add` · workbench boundary fixtures · `routes/i18n check` CI gates.

## Genuinely close to done (credit where earned — still not 10)
- **FA1 signals** — best-in-class core; only the computed-`.Text` footgun + unasserted bench.
- **FC1 local-first** — exceptional CRDT core + transport test; gap is op-based merge + API naming.
- **FB1 `//gwc:server`** — the keystone is real and excellent; gap is the leak-analyzer wiring.
- **FB7 micro-DX** — everything but `Index`.
- **FC3 agentui core** — the allow-list runtime is solid; gaps are all integration wiring.

## Theme for the next milestone
The work to date built **primitives**. The points left on the table are almost all
**integration + defaults + CI gates + the ergonomic hook layer** — comparatively
cheap, high-leverage polish that converts "a tested package exists" into "a developer
hits zero rough edges." Prioritize: (1) the 5 Critical headline-breakers, (2) wiring
the already-built analyzers/commands into `gwc check`/`gwc lint`/CI, (3) the missing
ergonomic hooks (`UseOptimistic`/`UseAction`, FA6 triggers, focus-refetch).

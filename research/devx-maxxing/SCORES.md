# devx-maxxing — Score Ledger

Running history of GoWebComponents DevX scores against
[`DEVX_MAXXING.md`](./DEVX_MAXXING.md). Each snapshot is stamped with the date
and the GWC git commit it was measured against, so improvement over time is
auditable. Scores use `min(first-hour, at-scale)` per the anti-gaming rule.

Target: god-tier DevX — **10/10 on every dimension** (at least at the planning
level; see [`IMPROVEMENT_PLAN.md`](./IMPROVEMENT_PLAN.md)).

| Snapshot date | Commit | Composite | Notes |
|---------------|--------|-----------|-------|
| 2026-06-27 | `cab0d205` | ~~81.3% (520/640)~~ → **72.5%** (464/640) corrected | **Review 1 baseline.** The 520 sum was an addition error (its own per-dimension contributions sum to 464); review 2 caught it. Use 72.5% as the true baseline for trend tracking. |
| 2026-06-27 | `cab0d205` | **99.5%** (637/640) projected | **Planning target** — projected composite if [`IMPROVEMENT_PLAN.md`](./IMPROVEMENT_PLAN.md) is fully implemented. Adversarial **SIGN-OFF achieved** after a 3-revision refinement loop. Not yet built; this is the ceiling, not a measurement. |
| 2026-06-27 | `1af9f6cc` (`v4`) | **80.8%** (517/640) measured | **Review 2.** Adversarial re-score after 84 `v4` commits. **+8.3 pp** over the corrected 72.5% baseline. Big jumps in data/routing/i18n/versioning/type-safety; several plan items claimed-but-not-actually-shipped (see below). |
| 2026-06-27 | `3c94979e` (`v4`) | **82.0%** (525/640) measured | **Review 3.** Adversarial re-verify after 51 commits fixing the feature/UX audit. **+1.2 pp.** 4 dims moved (B6 9→10, C3 9→10, C4 6→8, D2 8→9). **7/19 features now UX-perfect** (was 0). Modest composite gain because the sprint fixed Part-II features, not the high-weight Part-I runway (A2/C2/F7/F5/D3/D4). |
| 2026-06-28 | `d2e234e0` (`v4`) | **86.9%** (556/640) measured | **Review 4.** Adversarial re-verify after 18 commits doing the v6 Part-IV round (Tier-0 redos + Tier-1 high-weight movers). **+4.9 pp** — biggest jump. All 3 Tier-0 redos genuinely fixed. 11 dims moved. **18/19 features UX-perfect** (FB1 the lone ⚠️). Targeting the high-weight Part-I runway is what moved it. |
| 2026-06-28 | working tree | **88.75%** (568/640) measured | **Review 5.** Self-coded batch (main agent codes, subagent reviews) + 1 refinement round. Moves: A2 7→9, C2 7→9, A3 9→10 (all adversarially verified). Also done this batch but not in the R5 reviewer's file scope: windows/arm64 (A2, batch-1 verified), bench-drift push-trigger (D3), FB1 extensibility (lone ⚠️→✅, 19/19 features). **+1.9 pp.** |
| 2026-06-28 | working tree | **90.00%** (576/640) measured | **Review 6.** Batch 2: upstream-asks doc completes A2 9→**10** & C2 9→**10** (platform-honest, all anchor conditions verified); C1 8→**9** (client-side reload indicator confirmed — `updateGWCIcon`/popups/phase colors/save→paint + `-hot` default). **+1.25 pp.** C1→10 path identified (wire `hotreload.Enable()` into the default starter). E1 (size-budget gate, coded) pending next review. |
| 2026-06-28 | working tree | **91.1%** (583/640) measured | **Review 7.** E1 8→**10** (verified: `gwc wasm measure -max-gzip-bytes` gate + self-contained starter-CI size budget step). C1 9→**10** (reviewer-prescribed close: `hotreload.Enable()` injected into starter `main()` + `hot-reload` added to the common default presets `minimal-client`/`routed-spa` → state-preserving reload on the default scaffold path; tested). **+1.1 pp.** |
| 2026-06-28 | working tree | **91.6%** (586/640) measured | **Review 8.** Re-verification of under-credited dims: A1 9→**10** (`gwc doctor --fix` fully implemented in `doctorfix.go` — auto-applies edits + generates scaffold metadata; R6 note was stale), F2 8→**9** (`docs/capabilities` IS a machine-verified API→example index w/ CI existence guards). C4/F3/F8 held at 9 (cheap shortest-paths identified). **+0.5 pp.** F1 doc-sample gate coded, pending review. |
| 2026-06-28 | working tree | **92.0%** (589/640) measured | **Review 9.** F3 9→**10** (API-baseline PR merge gate `api-baseline.yml` across 9 packages + support window in VERSIONING.md), C4 9→**10** platform-honest (documented `SetWASMStackFrameMapper` stack-correlation workaround). F1 held at **9** — parse-level gate is real but a renamed API still parses; 10 needs type-level compilation of doc samples. **+0.47 pp.** |
| 2026-06-28 | working tree | **92.5%** (592/640) measured | **Review 10.** B2 8→**9** — built `gwc-consistency` lint (compat-alias-needs-deprecation + adjacent-bool guard; dropped an over-broad pure-delegate rule after a dump showed 208 false-positives); fixed all 8 real compat-alias violations + `css.Property` + production-stub variants to the deprecation protocol; merge-gate test. Held at 9: naming-convention lint declined (high-FP vs the noun-form DSL, ~zero real catches). **+0.47 pp.** |
| 2026-06-28 | working tree | **93.0%** (595/640) measured | **Review 11.** Under-credit corrections (evidence-verified): D6 8→**9** (all island strategies + budget queue + 3 bootstrap encodings + route-data reuse + DOM-reuse hydration present & native-tested; "graduate Experimental" gap was stale — no Experimental markers; only a browser-lane island e2e missing), E3 8→**9** (offline Playwright e2e + adversarial tests + full mutation queue now present; gap to 10 = composite update-prompt hook). C5 confirmed 8 (genuine completion/quick-fix builds needed). **+0.47 pp.** |
| 2026-06-28 | working tree | **93.1%** (596/640) measured | **Review 12.** E3 9→**10** — built `pwa.UpdatePrompt` (`pwa/update_prompt.go`): composes SubscribeLifecycle + SkipWaiting + ReloadOnControllerChange into one "update available → apply → reload" controller (the 10-rung update-prompt flow as one object). Native test drives the full flow (waiting-worker→Available→OnChange, Apply arms reload before SkipWaiting, transition-only firing). **+0.16 pp.** |
| 2026-06-28 | working tree | ~~95.3% (610/640)~~ → **95.5%** (611/640) corrected | **Review 15.** Scan of the eight unreviewed 9s found SIX under-credited to 10 (banked on file-cited evidence, no code): **B5** (time-travel+persistence+fine-grained+decision-guide), **B7** (lazy+revalidator+SSR-reuse+typed routes), **D2** (installable MV3 extension+OTel+scheduler flamegraph+redaction), **D3** (baseline+drift-gate-push/PR+vs-React+honest docs), **F4** (conformance kit+rollback+escape hatches), **F5** (3rd-party-JS bridge+fallback+native parity). B4 (typed-token codegen) + F6 (LazyBundle) confirmed at 9 needing real builds. No bugs found. R16 cross-check caught a **−1 ledger slip**: the six moves carry weights B5×2+B7×2+D2+D3+F4×2+F5 = **+9** (→611), recorded as +8 (610). Correction is *up*, not down — evidence was all real, the label under-counted. **+1.41 pp.** |
| 2026-06-28 | working tree | **94.1%** (602/640) measured | **Review 14.** Surface scan found B8 under-credited + 2 real bugs. **B8 8→10** (shared `validate` struct + `Form.ValidateStruct` + async + dirty/touched/reset all present; shipped the discoverability layer: `examples/public/shared-form-validation` + capability row + ch11/ch16 docs). **E2 8→9** (fixed real bug: reproducibility gate was silently skipped in CI under `-short` → dedicated non-short `release.yml` step, byte-identical verified). Also fixed: F8 ch05 false-claim (shorthand-vs-scaffold), and a defect in my own shared-form example (string→int age coercion). **+0.47 pp.** |
| 2026-06-28 | working tree | **96.3%** (616/640) measured | **Review 18.** C5 IDE integration 8→**9** (w2, +2) — closed the three stated gaps. (1) Fixability signal: `lintIssueRecord.Fixable` set from golangci's `issue.Replacement` (`tools/gwc/lint.go`). (2) `gwc lint --fix` delegates to golangci's own verified `run --fix` (no editor-computed edits) — `runLintFix`/`buildLintFixArgs`. (3) Editor quick-fixes: host-independent `toCodeActions` (node-tested in `tools/vscode-gwc/diagnostics.js`) + `CodeActionProvider` + `gwc.fix` command (`extension.js`, pkg v0.2.0). (4) Completion story confirmed REAL by reviewer: typed routes/i18n/CSS-token codegen emit ordinary typed Go symbols → gopls autocompletes + typo is a compile error (no custom LSP needed). Docs: extension README + ch02 "Editor integration". Reviewer RAN all Go+node tests (pass), confirmed wiring not dead code. **Held at 9, not 10:** true 10 needs marketplace/vsix distribution + `.vscode/extensions.json` auto-recommend (extension is unpublished) + file-scoped `--fix --path` — distribution is publisher-account/runtime-blocked here; refused to fake it with a recommend-an-unpublished-extension file. **+0.31 pp.** |
| 2026-06-28 | working tree | **95.9%** (614/640) measured | **Review 17.** B4 Styling 9→**10** (w2, +2) — shipped typed CSS-token codegen, the one stated B4 gap. `css/u/scale.go`: `type ColorToken` + 21 constants covering 100% of `css.DefaultTheme` (mirrors the existing `Radius`/`TextScale`/`Spacing` typed scales) + `resolveColor`. `css/u/util.go`: typed `BgC`/`TextC`/`BorderC(ColorToken)`; deprecated the stringly `BgToken`/`TextToken`. `tools/gwc/cssgen.go`: `gwc css gen\|check -theme theme.json` generates typed `u.ColorToken/TextScale/Radius/Spacing` constants for a **custom** theme (mirrors `gwc i18n gen`/`gwc routes gen`), wired into `main.go` + help + a codegen-staleness CI gate (`supply-chain.yml`). Canonical `examples/public/typed-css-tokens-demo/` + ch05 docs + capability row. Adversarial reviewer RAN it: all tests pass, generated pkg compiles against css/u, **typo-won't-compile proven** (`u.BgC(u.ColorBrand5000)` → `undefined`; typed-string var → `cannot use ... as u.ColorToken`), staleness gate **exits 1 on drift**. Sole residual = untyped-string-literal bypass, a Go-language ceiling that equally applied to Radius/TextScale at 9 (not the stated gap; canonical typed path fully protected). **+0.31 pp.** |
| 2026-06-28 | working tree | **95.6%** (612/640) measured | **Review 16.** F6 9→**10** — built `i18n.LazyBundle` (`i18n/lazy_bundle.go` + test): mutex-guarded load-once-per-locale orchestration over a user `LocaleLoader` (dedup, concurrency-safe, retryable-on-error), completing the F6 10-rung (SSR-aligned bootstrap `ToSSRBootstrap` + typed keys `gwc i18n gen` were already present). Adversarial reviewer **ran the tests** (both `TestLazyBundle*` PASS, `GOOS=js` build clean) and independently re-verified all six R15 bankings hold (every cited file/fn real, no stubs) — and caught the R15 −1 arithmetic slip (corrected to 611). +F6 (w1) → **612/640**. Cheap under-credited harvest now exhausted; remaining sub-10 dims are real builds / runtime-blocked / governance decisions / F7 structural cap. **+0.16 pp.** |
| 2026-06-28 | working tree | **93.6%** (599/640) measured | **Review 13.** F1 9→**10** — expanded the doc compile-gate from 1 to **15** `gwc:build`-tagged runnable samples (README + reference manual), all compile against the real module via `doc-samples.yml`; 40 illustrative fragments stay parse-only by design. **The gate caught a real bug**: the C4 stack-correlation doc imported internal-only `runtime.SetWASMStackFrameMapper` → shipped public `ui.SetWASMStackFrameMapper` (the C4 workaround was previously unusable). F2 held at **9** (per-capability indexed CI-built examples now complete incl. new `feature-flags` example, but the literal "single showcase dogfoods EVERY API" clause is unmet — portfolio-site uses ~6 of 19 packages). **+0.47 pp.** |

### Refinement loop (2026-06-27, sequential sonnet auditors)
Converged to adversarial sign-off over six subagent passes:
1. **Baseline scan** → 81.3%, evidence-linked, 34 dimensions.
2. **Critique r1** → NO; 21 dimensions insufficient/hand-wavy/DX-negative; answered 3 open questions (C2/C4 platform-capped, F7 structural).
3. **Critique r2 (plan v2)** → conditional; 17 RESOLVED / 4 PARTIAL; gated on 2 one-line fixes (F3 policy-doc, D1 native-testable fixtures) + 2 nitpicks.
4. **Simulated re-scan (plan v3, full re-rate with proposals as context)** → caught the plan claiming "31 at 10" while A2/A3/B1/B5/C3 had no items; projected 94.8%; NO.
5. **Final re-scan** → 99.5%; one blocker: C3 falsely claimed `hookcheck` already locates conditional hooks.
6. **Sign-off confirm (C3 corrected)** → **YES.** Every dimension at honest maximum.

### Planning-level target shape (signed off)
- **30 dimensions at literal 10.**
- **3 platform-honest 10:** A2 (Go module cold-fetch), C2 (incremental wasm link), C4 (wasm source maps) — each with a filed upstream ask + a stretch route to literal 10.
- **1 structural ceiling:** F7 = 7 (sizable ecosystem + hiring pool is emergent; cap recorded, not papered over).

### Implementation-time watch-items (NOT plan defects — verify at build time)
1. **C3** conditional-hook detection must not flag legal early-return guards as violations (edge-case taxonomy).
2. **F9** module-split major-bump decision can only be finalized after `go mod tidy` reveals the real consumer `go.sum` delta.
3. **D3** the 123% `BenchmarkFineGrainedSelectorDashboardReactiveTextUpdate16-20` regression is real and may surface a deeper architectural cost.
4. **C4** Chrome Web Store / Firefox AMO review timelines gate the "installable" ship date.
5. **E2** byte-identical reproducibility needs explicit cross-OS verification (Windows PE timestamps / path separators in debug info).

---

## Snapshot — 2026-06-27 @ `cab0d205` (baseline)

- Measured by: adversarial DevX auditor subagent (sonnet), evidence-linked.
- Branch: `master`. Commit date: 2026-06-25. Measured: 2026-06-27.
- Composite (lower-lens, weighted): **520 / 640 = 81.3%**.

| Dim | First-hour | At-scale | min | Evidence summary |
|-----|-----------|---------|-----|------------------|
| A1 Installation & prereqs | 7 | 7 | 7 | `gwc doctor` structured checks + remediation hints; no auto-fix; Go 1.26 prereq. |
| A2 Time-to-first-render | 6 | 6 | 6 | 2-cmd golden path but Go-install + tool-compile + wasm build push clean machine to 5–15 min. |
| A3 Scaffolding & starters | 8 | 8 | 8 | `gwc start` TUI, 8 curated starters, each w/ CI + scaffold test. |
| B1 Component model | 8 | 8 | 8 | Typed props/children/context/slots; React 1:1; error/async boundaries. |
| B2 API consistency | 7 | 7 | 7 | CONVENTIONS.md enforced; two HTML surfaces; some legacy aliases. |
| B3 Type safety | 8 | 8 | 8 | Props, CSS tokens, route contracts all compile-checked. |
| B4 Styling | 8 | 7 | 7 | 3-layer typed CSS, theme, dynamic vars, SSR extraction; `css/u` is curated subset. |
| B5 State management | 8 | 8 | 8 | Full ladder + snapshot versioning + SQLite/KV; fine-grained opt-in. |
| B6 Data fetching & async | 8 | 7 | 7 | UseQuery/optimistic/infinite/WS/SSE/offline-queue; some Experimental. |
| B7 Routing | 8 | 7 | 7 | Typed params/contracts/loaders/guards/nested/lazy; some Experimental. |
| B8 Forms & validation | 7 | 7 | 7 | UseForm typed + a11y errors + CSRF; no single shared client/server schema. |
| C1 Hot reload | 7 | 7 | 7 | `gwc dev` live-reload; state-preserving reload is opt-in (`hotreload.Enable()`). |
| C2 Build speed | 5 | 5 | 5 | Full wasm relink every change; no incremental wasm; times unpublished. |
| C3 Error messages | 8 | 8 | 8 | 30 `GWC-*` codes, generated, drift-guarded, remediation text, agent-readable. |
| C4 Debugging | 4 | 4 | 4 | No browser source maps (Go/wasm gap); no shipped browser-extension panel. |
| C5 IDE integration | 7 | 7 | 7 | Committed `.vscode` config, `gwc lint` hooks/deprecations; no shipped extension/gopls plugin. |
| D1 Testing | 9 | 9 | 9 | 4 lanes (native/wasm/hydration/browser), consumer testkit, headless component tests. |
| D2 Observability & devtools | 7 | 7 | 7 | In-app panel/snapshots/bundles, OTel export; no shipped browser extension. |
| D3 Performance & profiling | 8 | 7 | 7 | `gwc bench` 249 benches + vs-React + drift guard; drift OUT of tolerance at snapshot. |
| D4 Accessibility | 8 | 8 | 8 | Headless a11y components, focus primitives, axe-core CI gate. |
| D5 Security | 8 | 8 | 8 | No raw-HTML default sink, sanitize pkg, CSP nonce, SECURITY.md; no govulncheck/SBOM in CI. |
| D6 SSR / hydration | 8 | 7 | 7 | Streaming shell-first, islands, mismatch diagnostics, bootstrap transports; some Experimental. |
| E1 Bundle size & output | 7 | 7 | 7 | Profiles + gzip/brotli + measure; ~6.3MB raw/1.24MB br counter; no consumer size gate. |
| E2 Deployment & release | 7 | 7 | 7 | `gwc release` manifest + compression sidecars; no reproducible-build guarantee. |
| E3 PWA / offline | 7 | 7 | 7 | SW helpers, manifest, install observe, mutation-queue replay; SW script is app code. |
| F1 Documentation | 9 | 9 | 9 | 16-ch reference manual, API browser, drift-checked matrix + error index, doclint CI. |
| F2 Examples & recipes | 8 | 8 | 8 | 95+ examples + 2 showcase apps + capability matrix; not all serving paths guaranteed. |
| F3 Versioning & migration | 7 | 7 | 7 | SemVer, CHANGELOG, stability tiers, `gwc migrate`, API compat guard; rapid cadence. |
| F4 Extensibility & plugins | 7 | 7 | 7 | `plugin.Host` app extensions, css utility/variant/sink hooks; kernel sealed. |
| F5 Interoperability | 8 | 7 | 7 | Typed interop (storage/clipboard/observers/workers/cross-tab), native stubs. |
| F6 i18n | 8 | 7 | 7 | Plurals/format/locale-routing/SSR-bootstrap; keys not compile-checked; lazy load undocumented. |
| F7 Community & governance | 3 | 3 | 3 | Bus factor 1; no community channel/roadmap; CONTRIBUTING/SECURITY exist. |
| F8 Conceptual coherence | 8 | 8 | 8 | React model scales button→SSR; friction: two HTML surfaces, wasm/native split. |
| F9 Maintenance & deps | 6 | 6 | 6 | 20+ direct deps incl. AI/TUI tooling in same module as the library. |

**Lowest dimensions (priority targets):** F7 (3), C4 (4), C2 (5), F9 (6), A2 (6).

> **Baseline arithmetic correction (found in Review 2):** the per-dimension `min`
> scores in this table weight-sum to **464/640 = 72.5%**, not the 520/640 = 81.3%
> originally recorded. The 520 was an addition error. **72.5% is the true Review-1
> baseline.** All trend deltas use 72.5%.

---

## Snapshot — 2026-06-27 @ `1af9f6cc` (`v4`) — Review 2

- Measured by: adversarial DevX auditor subagent (sonnet), evidence-linked, against
  **actual `v4` code** (84 commits past baseline), verifying every "shipped" claim
  in [`IMPROVEMENT_PLAN.md`](./IMPROVEMENT_PLAN.md) rather than trusting it.
- Composite (lower-lens, weighted): **517 / 640 = 80.8%** — **+8.3 pp** over the
  corrected 72.5% baseline. Working tree clean.

| Dim | R1 min | R2 min | Δ | Evidence verified |
|-----|-------|-------|---|-------------------|
| A1 Installation | 7 | 7 | 0 | `gwc doctor` unchanged; `--fix` auto-remediation + cross-platform CI not shipped. |
| A2 Time-to-first-render | 6 | 6 | 0 | Prebuilt binary + published cold/warm timings not shipped; cold cost unchanged. |
| A3 Scaffolding | 8 | 8 | 0 | `gwc add` ships 2 WAI-ARIA components; `gwc add route`/`component` generators not shipped. |
| B1 Component model | 8 | 9 | +1 | `shorthand/slots.go` typed named slots; `workbench.RunStories` proves one model leaf→app. |
| B2 API consistency | 7 | 8 | +1 | shorthand blessed in ch.05; conventions surfaced inline; dedicated consistency lint not shipped. |
| B3 Type safety | 8 | 9 | +1 | `validate` + `routes gen` + `i18n gen` + `serverfn` shared structs + API baselines close stringly-typed surfaces. |
| B4 Styling | 7 | 7 | 0 | `css/u` completeness matrix + double-emit test not shipped. |
| B5 State management | 8 | 9 | +1 | `state.Signal[T]`/`NewComputed`; `html.BindTo`; ch.06 guide now covers SQLite/KV tier. |
| B6 Data fetching | 7 | 9 | +2 | `query` SWR+dedupe+optimistic+rollback, baseline-pinned Stable; e2e through serverfn. |
| B7 Routing | 7 | 9 | +2 | `gwc routes gen` typed Links + `router.DecodeQuery` typed search params. |
| B8 Forms & validation | 7 | 8 | +1 | `validate` dependency-free shared validator (wasm+native); `Form.ValidateStruct()`. |
| C1 Hot reload | 7 | 8 | +1 | `hotreload.SchemaFingerprint` ghost-bug fix + error overlay; `--hotreload` flag/status indicator not verified. |
| C2 Build speed | 5 | 6 | +1 | `gwc buildreport` cache-miss classification; persistent daemon + published timings not shipped. |
| C3 Error messages | 8 | 9 | +1 | `hookcheck` conditional detection + symbol naming (tested); but NOT wired into `gwc lint --json`. |
| C4 Debugging | 4 | 6 | +2 | `timetravel.History[T]` step-back engine + panel; NO installable browser extension (bridge is data-only). |
| C5 IDE integration | 7 | 8 | +1 | `tools/vscode-gwc` v0.1.0 surfaces `gwc lint --json` inline; route/style completion not in extension. |
| D1 Testing | 9 | 9 | 0 | `workbench.RunStories` browserless story runner; first-class boundary fixture pkg not shipped. |
| D2 Observability | 7 | 8 | +1 | `query/devtools`, `timetravel/devtools`, `localfirst/facepile`; installable browser ext not shipped. |
| D3 Performance | 7 | 7 | 0 | CI drift merge-gate not found; 123% regression doesn't repro on arm64 (deferred to amd64). |
| D4 Accessibility | 8 | 8 | 0 | WAI-ARIA components shipped; focus-on-route-change is example pattern, not built-in default; no a11y lint. |
| D5 Security | 8 | 9 | +1 | `gwc vuln` govulncheck reachability scan; release-only (not per-PR yet). |
| D6 SSR / hydration | 7 | 8 | +1 | `localfirst` SSR-seed-on-hydration test; serverfn route-data reuse; baselines graduate transports. |
| E1 Bundle size | 7 | 8 | +1 | `shorthand.Defer`/`UseDefer` lazy subtree + size-budget template; CI gate in starters not found. |
| E2 Deployment | 7 | 8 | +1 | `reprobuild` byte-identical SHA + `wholestack` one-binary; CI deploy templates not found. |
| E3 PWA / offline | 7 | 8 | +1 | `localfirst` durable offline queue (tested); zero-config SW + CI offline e2e not shipped. |
| F1 Documentation | 9 | 9 | 0 | `gwc llms` AI-readable docs + staleness gate; inline Go-sample compile-test not shipped. |
| F2 Examples | 8 | 8 | 0 | Catalog unchanged; machine-verified API→example index not shipped. |
| F3 Versioning | 7 | 9 | +2 | `VERSIONING.md` concrete policy (baseline-diff = boundary, 12-mo support, 2-minor deprecation). |
| F4 Extensibility | 7 | 8 | +1 | `agentui.Registry.Catalog` MCP introspection; ch.15 names escape per layer; conformance kit shipped. |
| F5 Interoperability | 7 | 7 | 0 | Third-party-JS example + native-stub parity test not found. |
| F6 i18n | 7 | 9 | +2 | `gwc i18n gen` typed keys (namespace+key+`{param}` compile-checked) + staleness gate. |
| F7 Community | 3 | 4 | +1 | GOVERNANCE/CONTRIBUTING/SECURITY/VERSIONING present; no public roadmap / Discussions / triage SLA. |
| F8 Coherence | 8 | 9 | +1 | ch.15 §70 "wasm/native build-tag split = one mental model" formally documented. |
| F9 Maintenance | 6 | 7 | +1 | `gwc supplychain` zero-npm + dep budget; module split deferred (major-bump call). |

**Composite: 517 / 640 = 80.8%** — Weight-3 sum 228, Weight-2 sum 192, Weight-1 sum 97.

### Claims that did NOT hold up (shipped-but-not-really)
The auditor verified against code and found these plan/claim items unmet:
1. **C2 persistent build daemon** — only the buildreport classifier shipped, not the warm-cache daemon; no published CI-gated timings.
2. **C4 installable browser extension** — `devtools/extension_bridge.go` is manifest *data structures* only; no `.crx`/`.xpi`, no store listing, no side-load. (Time-travel engine IS real.)
3. **C3 hookcheck not wired into `gwc lint`** — standalone `hookcheck` CLI is complete, but `gwc lint --json` (which the VS Code extension consumes) uses a separate path with no Symbol field. The symbol-named diagnostics don't reach `gwc lint` users.
4. **D3 CI drift merge-gate** — not present in `.github/workflows/`.
5. **D4 focus-on-route-change built-in default** — exists as an example pattern, not framework-wired; a11y lint not added.
6. **E3 zero-config service worker** — SW still app-wired; no CI offline e2e.
7. **F1 inline Go-sample compile-testing** — `doclint` checks path refs only, not Go code blocks.
8. **F5 third-party-JS integration example** — not found in v4.
9. **C1 `--hotreload` flag + reload-status indicator** — not verified in `dev.go`.

### Biggest real jumps
B6 +2 (query layer), B7 +2 (typed routes/search params), F3 +2 (concrete versioning policy),
F6 +2 (typed i18n keys), C4 +2 (time-travel engine). The keystone `serverfn` + `query` +
`validate` + `localfirst` work is genuine and tested.

### Feature/UX audit (2026-06-27 @ `1af9f6cc`)
See [`FEATURE_UX_AUDIT.md`](./FEATURE_UX_AUDIT.md) — aggressive per-feature sweep of
all 19 Part-II capabilities. Verdict: **0 of 19 are UX-perfect.** Every feature ships
a tested core primitive but is missing the last mile (ergonomic hook, wired-in default,
CI gate, loading/error state, or integration into the surface a dev touches). Single
worst gap: `gwc check --fix` doesn't exist though `AGENTS.md` tells agents to run it.

---

## Snapshot — 2026-06-27 @ `3c94979e` (`v4`) — Review 3

- Adversarial re-verify (sequential, no fan-out) after 51 commits fixing the
  feature/UX audit punch list. Composite **525/640 = 82.0%** (+1.2 pp over R2).
- **Dimension moves:** B6 9→10 (UseOptimistic/UseAction/UseAsyncMutation +
  UseRevalidateOnFocus complete the hook layer); C3 9→10 (Symbol field in `gwc lint`
  + server-leak diagnostics wired into `gwc check`); C4 6→8 (real MV3 browser
  extension now exists, was data-structures-only); D2 8→9 (working extension panel).
- **7/19 features now genuinely UX-perfect** (was 0): FA4, FB2, FB4, FB7, FC2, FC4, FC5.
- **Still shipped-but-rough (12):** `gwc check --fix` is gofmt-only (doesn't apply
  diagnostic remediations); FB6 extension has no `.crx` packaging (manual load-unpacked);
  FA6 still a boolean latch (no loading/error sub-blocks); FB1 leak analyzer is a
  6-entry deny-list not a graph pass (misses 3rd-party server imports); FA2 no
  `UseDurableMutation` bridge; FA5 router doesn't auto-call ViewTransition; FC6 PN-Counter
  fixes ints only (strings still LWW); FC3 catalog not an MCP tool; FA3 no a11y/CI tests.
- **No regressions.** Three honest scope-reductions flagged: gofmt-only `--fix`,
  deny-list "analyzer", int-only CRDT.

---

## Snapshot — 2026-06-28 @ `d2e234e0` (`v4`) — Review 4

- Adversarial re-verify (sequential, no fan-out) after 18 commits doing the Part-IV
  v6 round. Composite **556/640 = 86.9%** (+4.9 pp over R3) — the biggest single jump,
  because this round finally targeted the **high-weight Part-I runway**, not features.
- **All 3 Tier-0 redos genuinely fixed** (were scope-reduced in R3): FB3 `check --fix`
  now applies structured server-leak remediations to a fixed point (not gofmt-only);
  FB1 is a real transitive import-graph walk (+ 17 curated 3rd-party server-SDK prefixes);
  FC6 ships a real RGA collaborative-**text** CRDT (concurrent same-position inserts both
  survive — LWW gone for strings).
- **11 dimensions moved:** A1 7→9 (`doctor --fix` + cross-OS CI), A2 6→7 (prebuilt
  binaries), A3 8→9 (catalog CI smoke), B4 7→9 (completeness matrix + no-double-emit),
  C2 6→7 (`gwc warm` daemon), C4 8→9 (one-command extension packager), D1 9→10 (all 4
  boundary fixtures), D3 7→9 (bench drift merge-gate), D4 8→10 (router focus default +
  a11y lint), D5 9→10 (blocking vuln gate), F4 8→9 (agentui catalog MCP tool), F5 7→9
  (3rd-party-JS bridge example + parity test), F7 4→7 (roadmap + triage SLA + Discussions).
- **18/19 features now UX-perfect.** FB1 is the lone ⚠️: the transitive walk is real, but
  unknown server libs not in the 17-entry curated prefix list pass silently (acknowledged
  deterministic-design ceiling).
- **No faked gates, no broken tests.** Four honest residual gaps flagged below.

### R4 residual gaps / sloppiness (next round)
1. **`release.yml` is missing `windows/arm64`** — builds 5 targets, claim implied 6. Directly
   relevant: this machine is Windows ARM64 (Snapdragon X2), so the prebuilt `gwc` binary
   wouldn't cover the maintainer's own hardware. One-line workflow fix.
2. **Cold/warm build timings unpublished** — the ONLY thing keeping both A2 (w3, at 7) and
   C2 (w2, at 7) below platform-honest 10. `gwc warm -once -json` emits timings; nothing
   persists/CI-gates them. Closing this = ~+9 composite points (the single best next move).
3. **`bench-drift.yml` runs on PRs only**, not direct pushes to main — a direct-push commit
   bypasses the drift gate.
4. **FB1 curated 3rd-party list is fragile** — a new ORM/cloud SDK not in the 17 prefixes is
   missed until added. Documented as deterministic-design tradeoff, not a regression.

### Strategic note (why +1.2pp despite 51 commits)
The sprint targeted **Part-II feature correctness**, which is mostly weight-1/2 and was
already near-ceiling per-dimension. The biggest remaining composite runway is in the
**high-weight Part-I dimensions left untouched**: A2 (6, w3), C2 (6, w2), F7 (4, w1),
F5 (7, w1), D3 (7, w1), D4 (8, w1), A1 (7, w2), B4 (7, w2), D5 (9→10 needs the
govulncheck gate flipped from advisory to blocking). To move the composite materially,
target those — not more feature polish.

### Why 80.8%, not the 99.5% planning ceiling
The framework genuinely advanced (+8.3 pp), but the gap to target is the nine unmet claims
above plus the structurally-/platform-capped dimensions (A2/C2/C4 platform-honest still need
their shipped halves completed; F7 needs community). The plan's "shipped" table is ahead of
what the code actually delivers on those nine items.

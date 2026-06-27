# devx-maxxing — Score Ledger

Running history of GoWebComponents DevX scores against
[`DEVX_MAXXING.md`](./DEVX_MAXXING.md). Each snapshot is stamped with the date
and the GWC git commit it was measured against, so improvement over time is
auditable. Scores use `min(first-hour, at-scale)` per the anti-gaming rule.

Target: god-tier DevX — **10/10 on every dimension** (at least at the planning
level; see [`IMPROVEMENT_PLAN.md`](./IMPROVEMENT_PLAN.md)).

| Snapshot date | Commit | Composite | Notes |
|---------------|--------|-----------|-------|
| 2026-06-27 | `cab0d205` | **81.3%** (520/640) | Baseline. Adversarial measurement. Working tree had uncommitted WIP from other agents. |
| 2026-06-27 | `cab0d205` | **99.5%** (637/640) projected | **Planning target** — projected composite if [`IMPROVEMENT_PLAN.md`](./IMPROVEMENT_PLAN.md) is fully implemented. Adversarial **SIGN-OFF achieved** after a 3-revision refinement loop. Not yet built; this is the ceiling, not a measurement. |

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

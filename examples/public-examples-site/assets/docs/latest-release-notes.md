# Latest Release Notes

This docs-site page mirrors the first section returned by
`tools/changelogcheck.LatestEntry(CHANGELOG.md)`.

## v4.2.0 - 2026-07-05

The current latest changelog section is `v4.2.0 - 2026-07-05` — a second
performance release from an overnight benchmark-refine-regress loop. GWC's
production build now runs a **0.85–0.95 same-run geomean band against React
19's production bundle, with 8–10 outright scenario wins per run**.

**Highlights:**

- **Interactive GC pacing default** — browser apps init with GOGC=300 + a
  512MB memory-limit backstop (override via `localStorage["gwc:gogc"]`);
  in-window GC pauses that dominated small-update scenarios are largely gone.
- **Production-vs-production benchmarking** — the scored harness now builds
  GWC with `-profile benchmark`; previously a dev build (timers + threading
  guard active) was compared against React's production bundle.
- **Serialized mounts extended** — flat lists and mixed text+element chains
  mount from one template parse; deep-render ~8–9ms → ~2.8ms.
- **`ui.Typed`, `ui.UseMemoOf`/`UseEffectOf`, `Props.DataAttr`, `Props.Text`**
  — reflection-free component dispatch and zero-allocation hooks/element
  construction fields (native mirror allocations roughly halved).
- **Deletion teardown and keyed-append fast paths** — no registry-wide scans
  for never-subscribed fibers, fewer subtree walks, no key boxing on list
  growth.

It is checked by the blocking `tools/changelogcheck` release gate before a versioned
release can ship.

See the repository root `CHANGELOG.md` for the full entry body.

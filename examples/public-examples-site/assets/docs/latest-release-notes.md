# Latest Release Notes

This docs-site page mirrors the first section returned by
`tools/changelogcheck.LatestEntry(CHANGELOG.md)`.

## v4.1.1 - 2026-07-04

The current latest changelog section is `v4.1.1 - 2026-07-04` — release-gate
follow-ups for v4.1.0 (this docs mirror + a CI-flaky fetch test ceiling). The
substantive release is **v4.1.0**: a performance release from a day-long
optimization campaign against the Example 201 browser benchmark (React 19
side-by-side). Same-run geomean vs React improved from ~0.27x to **0.71x**,
with two scenarios beating React outright (core-stress-update 1.10x,
content-update 1.02x).

**Performance (highlights):**

- **Synchronous discrete-event flush** — event handlers commit their render
  pass inside the event's own task instead of paying a per-interaction
  `setTimeout(0)` macrotask hop; goroutine state writes still coalesce.
- **Hook-slot machinery 2x** — the dev threading guard re-samples its
  `runtime.Stack` traceback every 4096th call (was every 64th), `GoUseState`
  caches its getter/setter closures per slot across renders, and
  `ui.CreateElement` stops rebuilding renderers on component-handle cache hits.
- **Typed no-map element fast lane** — compact host elements carry a
  deterministic attribute slice instead of a Props map; positional diffing,
  attribute-level commits, byte-identical SSR. Core native update 155µs → 55µs.
- **LIS-hybrid child reordering** and **serialized subtree mounts** (one
  `template.innerHTML` parse per eligible subtree instead of one bridge call
  per node).

**Fixed:** the bounded diagnostics ring rebuilt itself entirely on every
report once full (quadratic under mismatch storms; 478ms → 6.4ms for 2000
warnings), and mockdom's `textContent` now includes descendant text like a
browser.

**Added:** native mirror/hooks/hydration benchmarks, a browser boot/TTI probe,
GC counters in the phase probe, and `mockdom.CreateHTMLSubtree`.

It is checked by the blocking `tools/changelogcheck` release gate before a versioned
release can ship.

See the repository root `CHANGELOG.md` for the full entry body.

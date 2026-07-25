# Latest Release Notes

This docs-site page mirrors the first section returned by
`tools/changelogcheck.LatestEntry(CHANGELOG.md)`.

## v5.0.0 - 2026-07-25

The current latest changelog section is `v5.0.0 - 2026-07-25` — a **major**
release. The module path is now `github.com/monstercameron/GoWebComponents/v5`,
because `ui.ParallelRegion` and the surrounding parallel-region API are removed
and `ui.Hydrate`/`ui.HydrateInto` lose a parameter. Rendering across workers
measured ~11%, which did not pay for its complexity.

**Highlights:**

- **The v5 thesis, measured** — with background work relocated to a domain
  worker, the harness reports a loaded p95 frame of 16.70 ms against a 16.70 ms
  idle frame, against 1633.30 ms for the same workloads on the render thread.
- **`ui.PostAsync`** — the supported way for a worker reply, callback, or
  goroutine to change rendered state; writes are applied at one defined point per
  frame, so several together produce one render.
- **Cancellable bulk commands** — `SetYield`/`YieldEvery` let a long run return
  to its worker's message loop, which is what makes a cancel able to reach it.
- **Two-artifact packaging** — `app.wasm` renders, `services.wasm` owns the
  engine, with a test that fails the build if the engine ever leaks.
- **Three fixed hangs and one dropped update** — a `Deliver` deadlock inside the
  worker message callback, rollback on a cancelled context, a commit failure that
  never cleaned up, and lane deferral discarding the work it deferred.

It also records what is still open — M2 and M7 missed, exactly-once incomplete,
Initial Render still behind React — rather than reporting a clean bill.

It is checked by the blocking `tools/changelogcheck` release gate before a versioned
release can ship.

See the repository root `CHANGELOG.md` for the full entry body.

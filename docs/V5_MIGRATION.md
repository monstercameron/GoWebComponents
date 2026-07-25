# Migrating an app to GWC v5

Plan item **P3.13**, first half. The second half — porting CashFlux and running
its e2e suite green — lives in that application's own repository and is not part
of this branch.

This guide is written from the v5 work as built, so every API named here exists
and every claim has a test behind it. Where something is unverified, it says so.

---

## The packages

Everything below is importable from any module. Verified by building an external
module against them, not by inspection — three of these lived under `internal/`
until it turned out that made the two-artifact split impossible for anyone
outside this repository.

| Package | What it is | Runs on |
|---|---|---|
| `html` | `VirtualList` | render thread |
| `projection` | resident reads, typed commands, failure model, SSR bootstrap, worker client | render thread |
| `gcpacing` | GC profiles | both |
| `trace` | cross-thread timeline | both |
| `domain` | command runtime: replay, resume, cancel | worker |
| `delta` | O(change) publication engine | worker |
| `db/offthread` | SQLite client — links no engine | render thread |
| `db/offthread/server` | SQLite server — links the engine | worker |
| `db/durability` | storage tier + single-writer model | worker |
| `compute` | CPU-bound job pool | render thread |
| `escalate` | migration assistant | build-time tool |

`internal/services` stays internal deliberately: it is the substrate `compute`
wraps, and its types reach you through `compute`'s own aliases.

## Do you need to migrate?

Probably not all at once, and possibly not at all.

v5 is additive. An app that keeps using `sqlite.Open` and renders with `html.Map`
compiles and behaves exactly as it did — that is P3.5's criterion (a), checked by
a dependency-graph test asserting `db/sqlite` never comes to depend on the new
code.

Migrate a screen when it has a symptom:

| Symptom | The item that addresses it |
|---|---|
| Typing stutters while a query or import runs | P3.5 — move the database off-thread |
| A list of thousands of rows is slow to update | P4.1 — `html.VirtualList` |
| Every keystroke re-queries | P3.7 — a resident projection |
| A crash mid-import restarts it from zero | P3.4 — a bulk command |
| Server-rendered content flashes empty on hydrate | P3.11 — bootstrap the projection |

Migrating a screen with none of these buys nothing and costs a rewrite.

---

## Step 1 — virtualize long lists

The cheapest change with the largest measured effect, and it needs no
architecture work at all.

```go
html.VirtualList(html.VirtualListProps{
    ItemCount:      len(rows),
    ItemHeight:     32,
    ViewportHeight: 640,
    Key:            func(i int) any { return rows[i].ID },
    Render:         func(i int) ui.Node { return renderRow(rows[i]) },
})
```

Measured at 10,000 rows: **137× faster to build, 131× fewer allocations**, and
flat from 1,000 to 100,000 rows.

Two things to know before you use it:

- **`ItemHeight` is fixed.** Variable heights need a measurement pass per row,
  which is the O(dataset) cost virtualization exists to remove. Rows that
  genuinely vary should set `Unvirtualized: true` and stay short.
- **Supply `Key`.** Without it the reconciler keys on position, and scrolling by
  one row re-renders the whole window.

---

## Step 2 — find what has to move

```go
findings, _ := escalate.AnalyzeSource(fileName, source, escalate.Options{
    Receivers: []string{"s.db"},
})
summary := escalate.Summarize(findings)
```

It reports every `Query`, `QueryRow`, and `Exec`, classifies each read or write,
and proposes a command signature. `summary.NeedsHuman` is the number that matters
for planning: sites whose SQL is built at runtime, which the tool refuses to
classify rather than guess at.

**It is not a codemod, and that is not caution.** Mechanically wrapping each call
site in a command produces one command per site — a chatty protocol with exactly
the round-trip problem moving off-thread was meant to solve. Deciding which sites
collapse into one command is the migration.

---

## Step 3 — split the binary

Two `main` packages from one module:

- `app` — renders. Must not import `db/sqlite`, or the SQLite engine and its wasm
  interpreter land in `app.wasm`.
- `services` — the domain worker. Imports `db/offthread/server`, `domain`,
  `delta`.

Copy the structure from `examples/v5-two-artifact`, including its test. The test
is the point: it fails the build if `app.wasm` ever links the engine, checked on
the dependency graph rather than on a size that could be small for unrelated
reasons.

Sizes, brotli: `app.wasm` 1.26 MB, `services.wasm` 1.96 MB. See
`docs/V5_SIZE_STORY.md` for the full accounting, including what is not measured.

---

## Step 4 — declare commands

```go
var CreateOrder = projection.Define[CreateOrderArgs, CreateOrderResult]("createOrder")

type CreateOrderArgs struct{ CustomerID string `json:"customerId"` }
type CreateOrderResult struct{ ID string `json:"id"` }
```

Declared once at package scope and invoked through the variable, so a
misspelling is an undefined identifier rather than a runtime failure in the
worker after a user clicked something. That is checked by actually compiling a
deliberately misspelled program and requiring it to fail.

The compiler cannot check that the *name string* matches a handler the worker
has, so verify the set at startup:

```go
if err := registry.Verify(CreateOrder, CancelOrder, ArchiveOrder); err != nil {
    return err   // names every mismatch, not just the first
}
```

### Give commands stable IDs

The domain runtime keys replay protection on the command ID. An ID generated
fresh per attempt gets **no replay protection at all** — the single most likely
way to misuse this. Derive it from the operation, not the attempt.

---

## Step 5 — read from a projection, not the database

```go
proj, _ := projection.New(decodeOrder, projection.Options{})
proj.Apply(opsFromWorker)

visible := proj.Filter(func(o Order) bool { return strings.Contains(o.Ref, query) })
```

Filtering, sorting, and windowing are local. A filter over 20,000 resident rows
costs **18 µs** — against a worker round trip that cannot beat a millisecond.

Three things that catch people:

- **`Ready()` is not `Len() > 0`.** A projection with zero rows may be loaded (a
  search with no matches) or unloaded. Branch on `Ready()`, or a route with no
  results shows a spinner forever.
- **`Complete()` is not `Ready()`.** A server can send the first screenful.
  Conflating them stops pagination at the fold.
- **Residency is capped** at 20,000 rows by default, a number set by measurement
  (M12). Past the cap, `DroppedByResidency()` is non-zero and every local count
  is a lower bound rather than an answer.

---

## Step 6 — handle failure by kind

```go
if err != nil {
    switch kind := projection.Classify(err); {
    case kind == projection.FailureRejected:
        show(err)                    // the domain answered; retrying repeats it
    case kind.MayHaveApplied():
        reconcile()                  // do NOT roll back — see below
    default:
        retry()                      // definitely did not apply
    }
}
```

`MayHaveApplied()` is the question a UI actually has, and it drives the one rule
worth memorizing:

> **Never roll back an optimistic update whose outcome is unknown.**

Undoing a change the domain actually made leaves the UI showing state the domain
does not have, and the next published delta silently contradicts it. Keep the
optimistic state and let the projection correct you. `projection.Optimistic`
implements exactly this and rolls back only for failures known not to have
applied.

Retry refuses may-have-applied failures **by default**. Enable
`RetryMayHaveApplied` only once commands carry stable IDs, because that is what
makes a repeat a no-op.

---

## Step 7 — bootstrap server-rendered routes

```go
// server
bs, _ := projection.BuildBootstrap(keys, rows, encodeOrder, true /* complete */)

// client
proj.Hydrate(bs)   // Ready() is now true, on first render
```

Without this the page paints, the client mounts into an empty projection, and the
content vanishes and returns. Server rendering made the first paint fast and the
experience worse.

---

## What to expect, honestly

**Measured and met:** M4 (virtualization, 137× at 10k rows), M11 (worker index at
0.30× payload), M12's memory half (which sets the residency default).

**Measured and missed:** M5. `app.wasm` is 1.73 MB gzipped against a 1.6 MB
target — though 1.26 MB brotli, which is what browsers negotiate. Recorded as
missed because the metric says gzip.

**Not settled, and not claimed:** M7 (GC pause) and M12's pause half. Both need a
browser: Go's js/wasm runtime marks single-threaded without native Go's parallel
assist, so a native pause number is not the browser's. Native tests here assert
the *mechanism* and abstain on the number.

**Not what this fixes:** if a screen is slow because of what it renders rather
than what it queries, moving the database off-thread does nothing. P5.1 measured
component bodies at ~11% of a render pass; the win in v5 comes from moving domain
work, not from rendering faster.

---

## Order of operations

1. Virtualize lists — cheapest, largest measured effect, no architecture change.
2. Run the escalation assistant — find the work before committing to it.
3. Split the binary — the dependency test makes the split real.
4. Declare commands with stable IDs.
5. Move reads to a projection.
6. Handle failure by kind.
7. Bootstrap server-rendered routes.

Steps 1 and 2 are worth doing even if you stop there.

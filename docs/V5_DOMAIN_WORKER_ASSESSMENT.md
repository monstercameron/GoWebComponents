# Does the secondary domain wasm make sense?

An assessment of v5's two-artifact packaging (`app.wasm` + `services.wasm`), made
by running it rather than reading it. Measured 2026-07-26 on windows/arm64,
headless Chromium, against `examples/v5-two-artifact` served with the correct
`application/wasm` MIME type.

**Short answer: the transport makes sense and works. The data path is half-built —
commands flow out, state never flows back — and the reference example does not
demonstrate the property the split exists to provide.**

---

## What works, and works well

The command path is genuinely good, and its failure modes have clearly been paid
for once already:

- **Request correlation.** Every message carries an id, so several commands can be
  in flight without a reply reaching the wrong caller. `WorkerClient` owns this
  rather than the application.
- **The ready handshake.** `services.wasm` posts `{ready:true}` only after
  `onmessage` exists, and the app waits for it. Without this a command posted into
  a scope with no handler is dropped silently — a hang, not an error.
- **Three distinguishable failures.** A refusal (`DeliverRejection`), a dead worker
  (`WorkerDied`), and an unknown command are separate paths. The unknown-command
  case *answers* rather than ignoring, so a typo cannot hang its caller.
- **The `<undefined>` trap is handled.** `js.Value.String()` on an absent field
  returns the literal string `"<undefined>"`, which is non-empty — so a naive
  emptiness check turns every SUCCESS into a rejection. The code checks presence.
- **Handlers run on their own goroutine**, so a blocking handler cannot stop the
  worker from reading the cancel meant to stop it.

None of that is obvious, all of it is load-bearing, and it is the part of v5 that
most deserves to be copied.

## What does not work

### 1. State never flows back from the worker

This is the finding that matters.

`projection.Projection[T]` is populated by `Apply(ops []Op)`. In
`examples/v5-two-artifact/app/main.go`, the only references to the projection are
`Rows()` — **`Apply` is never called**. On the other side,
`services/main.go` declares `engine *delta.Engine`, assigns `delta.New()`, and
never uses it again; `postMessage` appears exactly twice, for a command reply and
for the ready signal. There is no delta publication anywhere.

So the render thread has no way to learn what the domain owns. The example inserts
a row into SQLite in the worker, returns `{id}`, logs `addRow committed 1`, and
displays nothing — because the projection it renders from was never fed.

Observed before any change (12s after load, no errors, no failed requests):

```
#app  children=1  innerHTML="<div></div>"  (11 chars)  innerText=""
console: "addRow committed 1"
```

The transport succeeded. The screen stayed empty. That is the exact failure shape
`PRODUCTION_READINESS.md` describes for M10 — "every one of them fails by
producing nothing" — still present in the example that work was meant to fix.

### 2. `projection.Projection` has no change signal

Its entire public surface is `Rows()`, `Len()`, `DroppedByResidency()` and
`Apply()`. There is no `Subscribe`, no notification, no reactive integration. So
even once a projection IS fed, nothing tells the runtime to re-render.

An app must therefore hold its own state and mirror the projection into it by
hand. That is a real gap: the projection is the API for getting worker-owned data
onto the render thread, and it cannot participate in rendering.

### 3. `ui.Render` was called with a snapshot, not a component

The example evaluated `parseProjection.Rows()` once, at boot — while the first
command was still waiting on the worker handshake, so the slice was empty — and
passed the result to `ui.Render`. Even with a populated projection and a change
signal, that structure could never update.

Fixed here: the tree is now `ui.CreateElement(func() ui.Node { ... })`, reads the
projection inside the component body, and depends on a global atom bumped from the
reply. With that change the page renders `Waiting for the domain worker…` instead
of an empty div — an honest empty state. It still shows no rows, because of
finding 1.

## What the split actually costs

| artifact | raw | gzip | brotli |
|---|---:|---:|---:|
| `app.wasm` | 6.23 MB | 1.69 MB | 1.44 MB |
| `services.wasm` | 9.56 MB | 2.68 MB | 2.29 MB |
| **total downloaded** | 15.8 MB | **4.37 MB** | 3.73 MB |

`docs/plans/v5-plan.md` records the split as "2.37 → 1.73 MB (−27%)". That is
`app.wasm` alone. The user downloads both halves — `worker.js` fetches
`services.wasm` at +262 ms on a cold load — so the total went from 2.37 MB to
4.37 MB, an **84% increase**, while the headline metric shows a 27% decrease.

Both numbers are true and they answer different questions. The split is a
**main-thread CPU** optimisation: the render thread never parses or instantiates
the database engine, which is real and valuable. It is not a size optimisation,
and M5 measuring only `app.wasm` makes it read like one. Note also that the Go
runtime is necessarily duplicated — the ~541 KB js/wasm floor is paid twice — so
splitting can never reduce total bytes.

`wasm_exec.js` is also fetched twice (page + `importScripts` in the worker),
though the second is cache-served.

## M10, re-measured

**1,078 ms** time-to-first-command against a 400 ms target, consistent across
runs (1076, 1078). Matches the 1,084 ms already recorded. Note what that number
measures: a command COMPLETING. Since nothing renders as a result, it is currently
time-to-first-invisible-effect.

## Verdict

The two-artifact model is architecturally sound and its hard parts — correlation,
handshake, failure taxonomy — are well built. But as shipped it demonstrates only
half of itself: **an app can tell the domain to do something; the domain cannot
tell the app what it now knows.**

Until a projection can be fed from the worker and can signal a change, the split
buys main-thread isolation for command execution only, and any real app must
hand-roll the publication path. That is a large amount of undifferentiated work to
leave to every consumer, and it is the reason a server-backed application (see
`shared/api` in the Atlas example) currently gets more from `serverfn` than from
the worker split.

### What would close the gap

1. Publish deltas from `services.wasm` — use the `delta.Engine` that is already
   instantiated, and post ops alongside command replies.
2. Call `Projection.Apply` on receipt in the app.
3. Give `Projection` a change signal that the runtime can subscribe to, so a
   component re-renders without an app-authored version counter.
4. Make the example demonstrate all three, and assert the rendered rows — not the
   console line. The current packaging test checks which symbols land in which
   binary, which is why this gap survived: it is invisible to a symbol check.

---

## Resolution — 1, 2 and 4 are now done

The example demonstrates the full loop. Verified in headless Chromium: a row
inserted into SQLite **inside the worker** now appears on the render thread.

```
#app  <div><div class="row">first — 0</div></div>
```

The closed path, end to end:

```
app     addRow command                    (off the event loop, via runCommand)
worker  domain.Runtime.Execute -> INSERT  (exactly-once)
worker  delta.Engine.Publish   -> 1 op    (content-hashed versions)
worker  postMessage({ops})                (no request id: unsolicited)
app     projection.Apply(ops)             (inside ui.PostAsync)
app     GlobalAtom.Set -> UseAtom re-render
```

Three things worth carrying out of doing it:

**Publication is unsolicited and must be routed by shape.** The reply protocol
correlates by request id, and an id that matches no waiter is how worker DEATH is
signalled. A published delta has nobody waiting for it, so tagging it `{ops:…}`
and checking that before the id paths is not a style choice — giving it id 0 would
report every publication as a dead worker.

**Versions must be derived from content.** `delta.Engine` treats an unchanged
version as "this row did not change". A counter republishes unchanged rows; a
constant never republishes changed ones. Hashing the payload makes a full
re-publish of an unchanged table produce zero ops and zero messages, which is what
makes publishing after *every* command affordable.

**`GlobalAtom.Get()` is not a reactive read, and nothing says so at the call
site.** This cost the most time here. `Get()` reads the registry and returns
without subscribing the rendering fiber; `Set` re-renders "every component
subscribed **via UseAtom**" — which, with a bare `Get()`, is none. The symptom is
a component that would render correct data if it ever re-rendered, and never
re-renders. The working pairing is `GlobalAtom.Set` off-fiber (a worker reply has
no fiber) and `state.UseAtom(sameID, …)` inside the component. Both are correct
APIs; the trap is that the wrong combination compiles, runs, and produces a
permanently stale screen with no error.

---

## Is the thesis achieved? Measured, not argued

The thesis: *heavier background work should take longer to COMPLETE, never longer
to PAINT.*

Method: main-thread **event-loop lag** — schedule a 0 ms timer repeatedly and
measure how late it fires. If the main thread is busy, the timer cannot run, so
lag *is* main-thread blocking. Unlike `requestAnimationFrame` it is not throttled
in headless, which matters: an rAF probe returns one sample per window there and
reads as a clean result when it is really no result. ~900 samples per 4 s window.

The load is a real domain command: N SQLite inserts plus ~20k arithmetic
operations each, executed in `services.wasm` on the worker thread.

| load on the worker | p50 | p95 | max | frames >50 ms |
|---|---:|---:|---:|---:|
| idle (control) | 4.2 | 5.1 | 6.1 | 0 |
| 50 rows | 4.2 | 5.2 | 25.7 | 0 |
| 200 rows | 4.2 | 5.1 | 18.1 | 0 |
| 800 rows | 4.2 | 5.1 | **53.8** | **1** |

**p50 and p95 do not move at all.** 4.2 ms and 5.1 ms whether the domain is idle
or executing 800 inserts with sustained CPU. That is the thesis, and it holds: the
work genuinely left the render thread, and typical responsiveness is untouched by
how much of it there is.

**The tail moves, and it scales with the RESULT, not the work.** Max lag goes
6.1 → 25.7 → 18.1 → 53.8 ms as the published delta grows. What the main thread is
doing during that spike is not domain work — it is `json.Unmarshal` of the ops,
`projection.Apply`, and the re-render that follows. Those are unavoidably on the
render thread, they are proportional to how much state came back, and at 800 rows
one of them crosses the 50 ms long-frame threshold.

So, precisely:

- **Domain work: off the render thread.** Achieved.
- **Database I/O: off the render thread.** Achieved — SQLite lives in the worker.
- **Network I/O: off the render thread** by construction; the browser does the
  transfer. But the response *decode and state application* are main-thread, and
  fall in the same category as delta application below.
- **Result ingestion: still on the render thread, and unsliced.** This is where the
  remaining long frames live.

This explains M2 directly. M2 counts frames over 50 ms and its target is zero; the
plan attributes the misses to unsliced commit. The measurement here says the
trigger is *result size*, and my earlier CPU profile agrees — every long frame had
`styleAndLayoutDuration = 0.0 ms` and was attributed to Go script time in one
uninterrupted turn. That Go script time is delta decode plus apply plus
reconciliation. Reconciliation is sliced against the frame budget; **delta
application is not sliced at all.**

### What that implies

The two-artifact split solved the half it set out to solve. The remaining work is
not "make the worker faster" or "move more into the worker" — it is to stop
applying an arbitrarily large delta in one uninterrupted turn:

1. Slice `Projection.Apply` against the frame budget, the way reconciliation
   already is.
2. Or bound the ops per message in the publisher, so a 20,000-row import arrives
   as many small applications rather than one large one. The engine already
   computes minimal deltas; nothing currently caps their size.

Either would move the max lag toward the p95, which is already excellent.

---

## The P0.2 gate harness does not measure in headless Chromium

Found while trying to re-measure M2 after the `Projection.Apply` fix. This
invalidates any gate number taken from a headless run of
`examples/testing/v5-load-harness`, including ones I reported minutes earlier.

Three runs produced, identically:

```
M2count=0  M2worstMs=0  M7maxGC=0  M7trunc=false  M1equiv=true
workloads=["import","reindex","decode"]
```

That reads like M1, M2 and M7 all passing. It is not.

**Negative control.** Six deliberate 180 ms main-thread blocks were injected while
the harness measured — blatant, unmissable jank, far past the 50 ms threshold.
**M2 still reported 0.** The metric's source is `long-animation-frame` (the LoAF
API), which does not fire in this headless environment, so the counter cannot go
above zero no matter what the page does.

**Corroborating evidence in the M1 sample.** The frame distribution reports
`mad: 7.28e-12` with idle p95 16.7999 ms and loaded p95 16.7000 ms over ~1440
samples each. A median absolute deviation of 7×10⁻¹² means every frame is exactly
16.7 ms — the 60 Hz interval, emitted on a fixed schedule. A real frame
distribution under load has variance. So M1's equivalence is true by construction
rather than by measurement, and M7's `0` is "no GC pause observed", not "GC pauses
are small".

**What this means.**

- A headless "pass" on M1/M2/M7 from this harness is not evidence. The instrument
  reports zero because nothing is being sampled.
- The gate does not fail loudly when it cannot measure. M3 *does* — its verdict
  says "no interactions recorded; probe is not exercising input" — which is
  exactly the right behaviour and exactly what M2 and M7 lack. M3 was the only
  honest metric in the run.
- The recorded M2 = 21 and M7 ≈ 5.5 ms in `docs/plans/v5-plan.md` must therefore
  have come from a headed browser or a differently configured run. Whatever
  produced them, it was not this path, and the plan does not say which.

**Required before any of these gates can be trusted again:**

1. Make M2 and M7 refuse to report a number when their source produced no samples,
   the way M3 already does. A metric that cannot distinguish "zero" from "not
   measured" is worse than a missing metric, because it reads as a pass.
2. Record in the harness output whether LoAF actually delivered entries, and how
   many, so a reader can tell a real zero from a dead probe.
3. Re-measure M1/M2/M7 in a headed browser, and state in the plan which mode each
   recorded number came from.

Until then the honest status of M2 is **unknown**, not met and not missed — the
21 that the plan records cannot be reproduced or refuted through this path.

### Still open

Item 3. `Projection` still has no change signal, so the example carries an atom
bumped by hand. That counter is a stand-in for a missing framework capability, and
it is commented as such at both the declaration and the write — if `Projection`
grows a subscription, both should be deleted in the same commit.

Item 4's second half also stands: the packaging test asserts which symbols land in
which binary and would still pass if the screen were blank. A test that asserts
the RENDERED ROW is what would have caught this in the first place.

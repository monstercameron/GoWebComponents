# v5 Plan — Adversarial Review Log

Ten rounds of hostile review against `v5-plan.md`. Kept separate so the plan
stays a plan; this is the record of how it got there, and it is more useful for
deciding *how to review* than for deciding what to build.

Scores are not comparable across rounds 1–9 (mechanism completeness) and round
10 (plan quality) — the axis changed, deliberately.

| Round | Score | The finding that mattered |
|---|---|---|
| 1 | 6/10 | The thesis contradicted its own cited source. The plan quoted `MICROBENCH_REPORT_arm64.md` for "the diff is at the floor" while ignoring that report's own **🎯 Central finding** — `cloneElementProps` at 46–56% of allocations, its *"single highest-leverage optimization."* → **Phase A** |
| 2 | 6/10 | A Go microbenchmark was conflated with a browser end-to-end figure; P2.3's global-state scope was short by three globals and a `fetch`/`agentbridge` coupling |
| 3 | 6/10 | **P1.2 widens the T12 race before P2.5 closes it.** Making time-slicing live multiplies mid-tree yields, each a window for a bug that commits a childless root. Shipping the improvement before the fix increases exposure to a blank screen |
| 4 | 6/10 | *"Converging in depth, not breadth — each round surfaces a fresh instance of the same two defect classes."* → stopped fixing instances, bounded the global audit once (32 call sites, 9 packages) and added **§13 consistency invariants** |
| 5 | 6/10 | Three **false-pass bugs in the benchmark harness**, all in the layer that decides pass/fail, all in the one file with no test coverage. A blind observer counted as zero; a dropped field let an overflowed GC window pass M7 |
| **DevX** | **5/10** | **The most valuable round.** *"Right about the architecture for the workload it was built to fix, and wrong to make it the price of admission for every app that isn't running that workload."* → **D4 rewritten to opt-in**, P3.5 became additive, Phase 3 found unable to pass its own exit test |
| 6 | 6/10 | `OffThread()` as a functional option **cannot** carry a compile-time guarantee — one function, one static return type. → two constructors |
| 7 | 5/10 | Keyed diffing claimed O(change) compute "by construction" while its own example passed the whole collection. **Two properties the same example could not hold simultaneously** |
| 8 | 5/10 | Filtering is not a mutation — per-keystroke re-subscription makes every keystroke a first-subscribe, putting M3's own probe on the O(N) path. And "a command is the transaction boundary" fails the plan's own 50k-row import |
| 9 | 5/10 | *"The third time a headline property is named in a table before its mechanism exists."* Residency put a dataset back on the render heap; M11 measured the wrong side of the boundary |
| **10** | **7/10** | **Altitude change.** Reviewed as a plan: blocking/advisory status per metric, a staffing assumption, a contingency for the largest bet, and acceptance criteria for the items that had none |

## What this cycle actually taught

**Rounds 1–5 and the DevX round were worth their cost.** They caught a thesis
that contradicted its own evidence, a phase that could not pass its own exit
test, three false-pass bugs in the instrument, and an architecture being made
mandatory for apps that would never need it. None of those were findable by
re-reading.

**Rounds 6–9 were a treadmill.** The score never moved. Each round asked "what
is the mechanism?", each answer specified one at implementation depth, and each
specification opened three more questions a hostile reviewer could legitimately
ask. The plan grew to **1,023 lines, 44% blockquoted mechanism.**

The signal was in the *shape of the questions*, not the score. Once a reviewer
starts asking **how does this work** rather than **is this the right thing to
build**, the document has stopped being a plan — and the correct response is to
answer with an acceptance criterion and an open question, not a design.

That is now **R8** and invariant 17. Had it existed at round 6, four rounds
would have produced fourteen open questions instead of six hundred lines of
premature design.

**The score stopped being the signal well before it stopped moving.** Five
rounds at 6/10 read as "not converging" when it actually meant "measuring the
wrong axis." The number only moved when the axis changed.

## Harness bugs found (all fixed, all regression-tested)

Kept because they are the most concrete artifact of the cycle — a benchmark that
reports false passes is worse than no benchmark, since it converts *we don't
know* into *we verified*.

1. `gate()` counted zero long frames as a pass when **no observer existed**
2. `pauseSampleTruncated` computed in `probe.go`, dropped in `metrics.js` — an
   overflowed GC window passed M7 with its true worst pause evicted
3. `gate()` read `p99` while ignoring the `tailReliable` flag added for exactly
   that purpose
4. `integration.test.mjs` maintained its **own copy** of `buildReport`, so
   reverting a production fix left the tests green
5. `buildReport` read the observer source from **window 0 only** — a mid-run
   observer failure reported the stronger instrument

Coverage went from 19 tests (pure math only) to **42** across three suites,
including seam tests driving the real `diffGoProbe` → `buildReport` → `gate`
pipeline. Bugs 1–3 and 5 were all in code that had no test until they were found.

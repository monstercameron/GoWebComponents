# Atlas Commerce OS as a v5 performance subject — measured baseline

`docs/PRODUCTION_READINESS.md` records the gap plainly: *"No real application
exercises v5, so nothing has validated it outside two synthetic harnesses."*
Atlas is now that application. This document is what measuring it found.

**This was a measurement pass, not an optimization pass.** No framework behaviour
was changed. Every number below carries the command that reproduces it.

The discipline standard is the one `PRODUCTION_READINESS.md` sets: two reviews
named `cloneElementProps` as 46–56% of framework allocations and measurement put
it at 3.14% flat. Nothing here is asserted without a number, hot spots that were
expected and did not appear are reported as such, and where a profile is
dominated by harness scaffolding it is named. (The harness's own share of the
native allocation profile is **0.015%** — one line, `renderAtlasPerfPayload`, at
1 MB of 6810 MB.)

---

## 0. What was measured, and the state it was measured in

Two lanes, because they answer different questions.

| Lane | Subject | What it can see | What it cannot |
|---|---|---|---|
| **Native (Go pprof)** | `ui.RenderToString(ui.CreateElement(atlas.App, payload))` over 7 real captured route payloads | component bodies, hooks, element construction, props, fiber creation, SSR walk, serialization, and every allocation they make | DOM commit, deletion, order repair, layout effects, the browser scheduler |
| **Browser (Playwright + CDP)** | the real Atlas wasm client under Chromium | hydration cost, Long Animation Frames with per-script attribution, interaction latency, and a V8 CPU profile bucketed by Go package | Go-side phase totals and GC pause — see §7 |

### Working-tree state (this is a dirty tree, mid-conversion)

Several agents were editing `shared/atlas`, `server/`, and `client/` throughout
this pass, converting markup to the design system. The tree did not compile for
part of it. Absolute numbers will move; the framework-vs-example *split* is the
stable part and is what should be quoted.

```
git status --short | wc -l   ->  89 modified paths
go version go1.26.3 windows/arm64
```

| file | sha256 (first 16) |
|---|---|
| `internal/runtime/reconciler.go` | `a088b82824e393ad` |
| `ui/ui_native.go` | `98e81b0ae7bb3e75` |
| `html/html.go` | `91b71e29cddac37c` |
| `css/css.go` | `f602f84d1e948d9b` |
| `css/rule.go` | `98c14b47842a80a1` |
| `css/registry.go` | `b460852e88fd3687` |
| `shared/atlas/page.go` | `8ef4985c9ac133cc` |
| `shared/atlas/inventory_cms.go` | `faafacf1d7cc6f80` |
| `shared/design/tokens.go` | `8d42b9ff3d0dc5af` |
| `examples/static/bin/atlas-commerce-os.wasm` | `013657611c0d44f3` |

The wasm bundle was rebuilt from this tree (staged to a temp name in the same
directory and `mv`-d into place, so a concurrent reader never sees a partial
file). The previous bundle, built earlier the same day, predated the design-system
conversion; §3.3 reports both because the pair is informative.

**Thermal caveat, per the plan.** This chassis is fanless and the plan records the
same build measuring 4 long frames cold and 33 hot. Every table below is a
min/median/max over repeats or a two-run reproduction, never a single reading.

### The harness

| Path | What it is |
|---|---|
| `examples/testing/atlas-perf/atlas_perf_harness_test.go` | fixture loading, serialized render, subject validation |
| `examples/testing/atlas-perf/atlas_perf_capture_test.go` | captures real bootstrap payloads from a running server (opt-in) |
| `examples/testing/atlas-perf/atlas_perf_render_test.go` | shape + timing + benchmarks + CSS-fold benchmarks + attribution recipe |
| `examples/testing/atlas-perf/atlas_perf_wasm_size_test.go` | raw/gzip/brotli against M5 |
| `examples/testing/atlas-perf/fixtures/*.json` | 7 captured route payloads, verbatim |
| `test/playwrightgo/examples/atlas_perf_browser_test.go` | hydration, LoAF, interaction latency, CDP CPU profiles, shell-stability probe |

Fixtures are **real**: each is the verbatim `<script id="__ATLAS_BOOTSTRAP__">`
the running server sent, with its shape recorded at capture time.

```
ATLAS_PERF_CAPTURE=1 go test ./examples/testing/atlas-perf/ -run TestAtlasPerfCaptureFixtures -v
  captured operator-comments  /app/comments   bootstrap=32176 bytes  items=88
  captured operator-dashboard /app/dashboard  bootstrap=34252 bytes  comments=88 transfers=3 receiving=2 orders=3
```

---

## 1. A structural fact that changes how to read everything else

**Atlas does not server-render markup.** Verified, not assumed:

```
grep -c RenderToString examples/server/atlas-commerce-os/server/*.go   # 0 in every file
curl -s http://127.0.0.1:8531/ | wc -c                                  # 7324 bytes
```

The server sends an empty `<div id="app"></div>` plus a JSON bootstrap payload
and paints everything from wasm. "SSR bootstrap + hydration" in Atlas means SSR
of the **data**, not of HTML.

Consequences:

1. The native lane is **a harness of our own construction**, not a reproduction of
   a server code path. It is still the right instrument for attributing render
   work — it exercises the identical component bodies, hooks, element
   construction and reconciliation the browser runs — but nothing in it is on a
   production request path today.
2. Every byte of Atlas's first paint is behind a ~20 MB wasm download and
   compile. That dominates cold boot and is not a reconciler problem (§3).

---

## 2. Ranked framework hot spots

Ordered by size of the effect, on the **native render path**, which is the lane
that isolates render work from boot work.

Reproduce all of §2:

```
go test ./examples/testing/atlas-perf/ -run '^$' -bench BenchmarkAtlasPerfAllRoutes \
  -benchtime 6000x -cpuprofile atlas_cpu.out -memprofile atlas_mem.out
go tool pprof -top -cum -nodecount=45 atlas_cpu.out
go tool pprof -sample_index=alloc_space -top -cum -nodecount=35 atlas_mem.out
```

Two independent runs, machine cooled between them. Both are quoted.

### 2.1 Typed-CSS class folding — 46% of render CPU, 63% of render allocations

**Confidence: high.** Reproduced twice with near-identical numbers, and it is the
largest single item by a wide margin in both profiles.

| symbol | CPU cum (run 1 / run 2) | alloc_space cum (run 1 / run 2) |
|---|---:|---:|
| `internal/runtime.renderToStringScoped` (the whole render path) | 55.72% / 61.73% | 99.81% / — |
| **`shared/design.Class`** | **25.69% / 28.68%** | **63.24% / 63.26%** |
| `css.New` | 23.41% / 26.15% | 44.07% / 44.16% |
| `css.canonicalize` | 22.88% / 25.31% | 44.06% / 44.16% |
| `css.canonicalize` (flat) | 2.41% / 2.30% | 24.58% / 24.39% |
| `strings.(*Builder).WriteString` (flat, mostly inside `canonicalize`) | — | 24.71% |

As a share **of the render path itself**, `design.Class` is **46.1%** and
**46.5%** of CPU in the two runs, and **63%** of allocations in both.

The mechanism is in `css/css.go`. `css.New` has a memo cache, and its comment
says *"Repeated New(...) in a render loop becomes a map lookup."* It does not:
`canonicalize(rules)` runs on **every** call, before the cache can be consulted,
and it builds a bucket map, sorts the groups, and serializes them into a
`strings.Builder` to produce the cache key.

Measured directly:

```
go test ./examples/testing/atlas-perf/ -run '^$' -bench BenchmarkAtlasPerfDesignClass -benchtime 2000x
```

| case | warm (cache hit) | cold (`css.Reset()` first) | what the cache saves |
|---|---:|---:|---:|
| single-bundle leaf | 2 624 ns, 2 682 B, 17 allocs | 5 435 ns, 3 992 B, 30 allocs | 52% time, 33% bytes |
| two-bundle control | 17 841 ns, 22 304 B, 64 allocs | 20 727 ns, 28 520 B, 93 allocs | 14% time, 22% bytes |
| four-bundle container | 23 880 ns, 26 682 B, 80 allocs | 30 984 ns, 34 352 B, 113 allocs | 23% time, 22% bytes |

**A warm fold of a four-bundle container still costs 23.9 µs and 26.7 KB.** The
fold cache removes under a quarter of the cost, because the expensive part runs
before it.

The call pattern makes this bite. `shared/design/doc.go` already prescribes the
fix — *"Static styles should also be hoisted to package vars so they fold exactly
once at init"* — and the application does not follow it:

```
grep -rn "design\.Class(" --include=*.go examples/server/atlas-commerce-os/shared/atlas/ | grep -v _test | wc -l   # 398
grep -rn "^var .* = design.Class(" --include=*.go examples/server/atlas-commerce-os/shared/atlas/ | wc -l          # 0
```

**398 call sites, none hoisted.** Many are inside per-row builders
(`commentTableRow`, `inventoryQueueTableRow`, `internalNavLink`), so one render
of the 88-row inbox folds the same rule sets dozens of times.

Attribution: the *cost* is framework (`css.canonicalize`); the *call frequency* is
example. Both halves are fixable independently and either one is worth roughly
half the render path.

That folding really is on the render path is checked as a build-graph-independent
fact, not inferred:

```
go test ./examples/testing/atlas-perf/ -run TestAtlasPerfDesignFoldReach -v
  typed-CSS classes emitted by one Atlas render of operator-comments:
    before=0 after=48 delta=48 styleBlock=17.4 KB
  VERDICT: typed-CSS folding IS on the Atlas render path — 48 folds per render of this route.
```

### 2.2 Allocation volume, not reconciler logic, is what the render path spends on

**Confidence: high.** Flat CPU aggregated by package over 277 symbols:

```
go tool pprof -top -nodecount=100000 atlas_cpu.out   # then fold by package prefix
```

| bucket | flat share |
|---|---:|
| Go runtime (GC mark/sweep, mallocgc, preemption, locks) | **73.76%** |
| `encoding/json` | 8.89% |
| stdlib other (`strings`, `slices`, `sort`, `reflect`) | 6.07% |
| framework `html` | 3.85% |
| framework `css` | 3.33% |
| example `shared/atlas` | 1.03% |
| framework `internal/runtime` | 0.43% |
| framework `ui` | 0.00% |
| **harness scaffolding** | **0.00%** |

Total samples were 14.95 s over a 10.23 s wall — 146% CPU — because background
GC workers ran in parallel the whole time (`runtime.gcBgMarkWorker` 24.08% cum,
`runtime.bgsweep` 8.83% cum, `runtime.preemptM` 11.17% cum).

Per render of the 88-row inbox: **3.06 MB allocated across 23 465 allocations**
to produce 46.6 KB of markup over 1 983 elements — roughly **1.5 KB and 12
allocations per element**.

The reconciler's own flat cost is 0.43%. Reconciliation logic is not where the
time is; feeding the allocator is.

### 2.3 The example's payload decode is a real render-path cost

**Confidence: high.** `encoding/json.Unmarshal` is **9.57% / 10.51% cum** of CPU
and `encoding/json.Marshal` is **8.63% / 9.49% cum** of allocations — *inside the
render*, not in payload setup.

The cause is `shared/atlas/page.go`:

```go
func decode[T any](parseValue any) T {
	parseEncoded, parseErr := json.Marshal(parseValue)   // marshal…
	_ = json.Unmarshal(parseEncoded, &parseResult)       // …then unmarshal, every call
}
```

`atlas.PayloadFromSSRBootstrap` does the same round trip once (`decodeInto`), and
`decode[T]` repeats it on **every read of the page data during render**. Measured
in isolation:

```
go test ./examples/testing/atlas-perf/ -run '^$' -bench BenchmarkAtlasPerfPayloadDecode -benchtime 50x
  operator-dashboard   1 190 034 ns/op   449 590 B/op   9 555 allocs/op
  operator-comments    1 115 586 ns/op   404 278 B/op   8 916 allocs/op
  public-landing          27 410 ns/op     9 072 B/op     197 allocs/op
```

**This is example cost, not framework cost.** It is listed because it is the
second-largest identifiable item and because attributing it to the reconciler
would be the exact error this document is written to avoid.

### 2.4 Component invocation goes through reflection

**Confidence: medium-high** (clear in the profile, mechanism not traced to a
specific fix). `reflect.Value.Call` is **8.83% / 11.81% cum** of CPU and 10.80%
cum of allocations, under `ui.buildComponentRenderer.func5`. Every component body
call on the SSR path pays a reflective dispatch. Whether a typed fast lane is
available here was not investigated.

### 2.5 Per-route cost and shape

```
go test ./examples/testing/atlas-perf/ -run TestAtlasPerfRouteShapes -v
go test ./examples/testing/atlas-perf/ -run '^$' -bench BenchmarkAtlasPerfRouteRender -benchtime 50x
```

Timing is a 20-render batch per sample, 15 samples — a single render of the
landing route reported `min=0.00 med=0.00 max=0.51 ms`, which was the Windows
clock quantum, not the renderer.

| fixture | route | min / med / max (ms) | spread | ns/op | B/op | allocs/op | markup | elements |
|---|---|---|---:|---:|---:|---:|---:|---:|
| operator-comments | `/app/comments` | 4.24 / 5.23 / 9.10 | 2.1× | 5 606 176 | 3 060 404 | 23 465 | 46 566 B | 1 983 |
| operator-dashboard | `/app/dashboard` | 2.41 / 2.92 / 3.63 | 1.5× | 3 075 556 | 1 517 214 | 13 450 | 13 791 B | 646 |
| operator-settings | `/app/settings` | 1.08 / 1.45 / 3.06 | 2.8× | 2 345 988 | 1 115 945 | 6 767 | 16 650 B | 481 |
| operator-inventory | `/app/inventory` | 1.41 / 1.69 / 4.73 | 3.4× | 2 310 454 | 1 193 826 | 8 976 | 13 541 B | 562 |
| public-product | `/shop/frame-desk` | 0.72 / 0.84 / 1.68 | 2.3× | 1 948 926 | 789 131 | 5 178 | 19 296 B | 429 |
| public-catalog | `/shop` | 0.42 / 0.46 / 0.89 | 2.1× | 906 324 | 430 572 | 2 737 | 11 409 B | 293 |
| public-landing | `/` | 0.13 / 0.18 / 0.85 | 6.8× | 163 600 | 176 360 | 1 136 | 7 108 B | 132 |

Cost tracks **element count and class count**, not markup bytes: `/app/settings`
emits more bytes than `/app/dashboard` and costs half as much, because it has 481
elements against 646.

### 2.6 `ui.UseId()` is not per-render on native — repeated SSR renders differ

**Confidence: high.** This is a correctness observation the perf harness tripped
over, not a perf finding.

`TestAtlasPerfRouteShapes` asserts that rendering the same payload twice produces
the same markup. It does not, for every route:

```
NOT IDEMPOTENT: operator-dashboard repeat render is 13785 bytes vs 13785; first difference at offset 6644:
  run1: id="gwc-native:81-workflow-sheet-title"
  run2: id="gwc-native:82-workflow-sheet-title"

NOT IDEMPOTENT: operator-inventory repeat render is 13508 bytes vs 13499
  run1: id="gwc-native:97-label"   run2: id="gwc-native:108-label"
```

Source, `ui/ui_native.go:490`:

```go
// UseId returns a stable generated identifier for the current component instance.
func UseId() string {
	nativeIDMu.Lock(); defer nativeIDMu.Unlock()
	nativeIDCounter++
	return fmt.Sprintf("gwc-native:%d", nativeIDCounter)
}
```

The counter is a **process global that is never reset per render**, while
`RenderToString` otherwise gives each render its own atom scope. The IDs it
produces are load-bearing — they appear in `aria-labelledby` — and they differ
between two renders of identical input. The byte-length also changes when the
counter crosses a digit boundary, which is why one route's two renders differed
in length.

Not exercised here, and worth checking before it is called harmless: whether an
SSR-then-hydrate flow could ever compare a server ID against a client ID.

---

## 3. The browser lane

```
go test -tags playwrightgo ./test/playwrightgo/examples/ -run TestAtlasPerf -v -timeout 40m
# or against an already-running server (skips a ~25 s boot per test):
ATLAS_PERF_BASE_URL=http://127.0.0.1:8531 go test -tags playwrightgo ./test/playwrightgo/examples/ -run TestAtlasPerf -v -timeout 40m
```

Long Animation Frames **are** available in this headless Chromium
(`instrument=long-animation-frame` in every run), so per-script attribution and
the style/layout split are real here. That is worth recording, because
`example201_spike_probe_test.go` documents `longtask` being unavailable and the
201 long-task metric therefore reading zero for 114 scenario rows.

### 3.1 Cold boot is bundle work, not render work

```
go test -tags playwrightgo ./test/playwrightgo/examples/ -run TestAtlasPerfBrowserWarmCacheHydration -v
```

Cold-cache load, then three reloads in the same context (V8 caches the compiled
wasm module):

| route | cold hydrated | warm hydrated min/med/max | warm post-fetch tail min/med/max | share of cold that vanishes when warm |
|---|---:|---:|---:|---:|
| `/` | 1000 ms | 182 / 247 / 272 | 123 / 134 / 199 | 75% |
| `/shop` | 1116 ms | 234 / 275 / 362 | 151 / 156 / 238 | 75% |
| `/app/dashboard` | 1277 ms | 249 / 299 / 396 | 173 / 240 / 295 | 77% |
| `/app/comments` | 1384 ms | 311 / 384 / 464 | 236 / 307 / 366 | 72% |

The HTML document is 9–13 ms to `responseEnd`. The 20.2 MB bundle transfers from
localhost in 59–89 ms. **Three quarters of a cold hydration is wasm compilation**
— the frame that disappears on a warm load is a single 300–390 ms Long Animation
Frame at t≈80–120 ms with `styleAndLayoutDuration = 0.0` and no script attribution,
which is the shape of module compile.

The residue — "post-fetch tail", from the wasm request's end to shell paint — is
**134–307 ms warm**, and that is the closest browser-side bound available on
instantiate + package init + payload decode + first render + first commit.

### 3.2 Where cold-boot time actually goes

V8 CPU profile across a cold hydration, bucketed by Go package (Go's js/wasm name
section survives, so wasm frames carry Go symbols):

```
go test -tags playwrightgo ./test/playwrightgo/examples/ -run TestAtlasPerfBrowserHydrationCPUProfile -v
```

`/app/comments`, 2 768 samples over 1 610 ms:

| bucket | self time |
|---|---:|
| Go runtime (GC, alloc, scheduler, wasm trampoline) | 19.98% |
| V8 `(program)` — wasm compile/JIT | 13.19% |
| **framework `internal/runtime`** | **10.69%** |
| **example (`shared/atlas` + `shared/design` + client)** | **10.22%** |
| `encoding/json` | 8.09% |
| stdlib other | 6.68% |
| V8 `(idle)` | 5.20% |
| `regexp` package-init compilation | 4.41% |
| Go↔JS bridge (`syscall/js` + `wasm_exec` glue) | 4.37% |
| `strings`/`sort`/`fmt`/`strconv` | 3.54% |
| **framework `ui` + `html`** | **2.75%** |
| Go package initializers | 2.64% |
| framework other v5 | 2.53% |
| **framework `css`** | **2.53%** |
| `reflect` | 2.53% |

Framework total ≈ **18.5%**. Example ≈ **10.2%**. Everything else — 71% — is wasm
compile, the Go runtime, package initialization and JSON.

The single largest named Go function on a cold boot is
`shared/design.init` at **3.36%** (4.48% on the landing route): the design
system's package-level bundle variables, paid once per page load.

`css` folding is only 2.53% here against 46% in the native lane. Both are correct
and they are not in conflict: a cold hydration folds each rule set once into a
cold cache and is diluted by ~700 ms of boot; the native lane renders repeatedly
with everything else warm and so shows what a *steady-state render* costs. §5.4
says which one bears on which gate.

### 3.3 Two builds, one day apart — a paired observation, not a causal claim

The bundle first measured was built before the design-system conversion reached
`page.go`; the bundle finally measured was built from the tree in §0, in which
`internalNavLink` and 398 other sites call `design.Class` per render.

| route | warm hydrated med, pre-conversion | post-conversion | warm tail med, pre | post |
|---|---:|---:|---:|---:|
| `/` | 177 ms | 247 ms | 108 ms | 134 ms |
| `/shop` | 182 ms | 275 ms | 120 ms | 156 ms |
| `/app/dashboard` | 199 ms | 299 ms | 120 ms | 240 ms |
| `/app/comments` | 245 ms | 384 ms | 144 ms | 307 ms |

**+40% to +113%.** This is stated as an observation, not a proven cause: the two
builds differ in more than the CSS conversion, hours of benchmarking separate
them on a fanless machine, and no per-change arm was run. It is recorded because
it points the same way as §2.1, which *is* isolated and reproduced.

### 3.4 Long frames on a real app: script time, never style/layout

```
go test -tags playwrightgo ./test/playwrightgo/examples/ -run TestAtlasPerfBrowserInteractions -v
```

| scenario | long frames > 50 ms | worst | blocking | style+layout |
|---|---:|---:|---:|---:|
| route swap `/app/dashboard` → `/app/comments` (88 rows) | 3 | 67.1 ms | 21.9 ms | **0.0 ms** |
| route swap `/app/comments` → `/app/inventory` | 2 | 186.9 ms | 136.8 ms | **0.0 ms** |
| 4 keystrokes into a text field | **0** | — | — | — |
| route swap `/app/inventory` → `/app/settings` | 0 | — | — | — |
| 40 keystrokes into the inventory filter | **0** | — | — | **0.0 ms** |

Per-script attribution of the worst frames:

```
frame 186.9 ms  blocking=136.8  styleLayout=0.0   script 185.0 ms  invoker=TimerHandler:setTimeout
frame  67.1 ms  blocking=13.9   styleLayout=0.0   script  52.0 ms  invoker=#document.onclick
frame  53.4 ms  blocking=0.0    styleLayout=0.0   script  31.0 ms  invoker=ReadableStreamDefaultReader.read.then
                                                  script   6.0 ms  invoker=ViewTransitionCallback  forcedStyleLayout=2.0
```

Interaction latency, real Playwright input (`n=12`, below the 200 the tail
guard needs for p95, so p95 is not quoted):

```
median 48 ms   max 96 ms
worst: pointerdown  total=96  inputDelay=3  processing=1  presentationDelay=92
```

The worst interaction is **96% presentation delay**. Input handling is
essentially free; the paint that follows it is late.

---

## 4. The four hypotheses

### H1 — "Initial Render is M6's dominant loss; does Atlas show the same shape?" — **CONFIRMED, with a different cause than expected**

Atlas's cold hydration is dominated by initial render, and it is 4–8× the cost of
any steady-state interaction. But the thing inside it that dominates is **not**
element construction, fiber creation, or first commit:

- 75% of cold hydration is wasm module compilation (§3.1).
- Of what remains, framework `internal/runtime` is 10.69% of samples,
  `ui` + `html` together 2.75%, `css` 2.53% (§3.2).
- `shared/design.init` alone (3.36%) costs more than `ui` + `html` combined.

**Wasm instantiation, not the framework, is Atlas's initial-render cost.** A
reconciler change of even 30% would move a cold Atlas boot by roughly 3%.

### H2 — "Unsliced commit is the long-frame source" — **NOT SUPPORTED by this evidence**

Every long frame measured on a real Atlas interaction had
`styleAndLayoutDuration = 0.0 ms` — including the 186.9 ms one. LoAF separates
script time from style and layout, and all the time was script.

The worst frames are attributed to `TimerHandler:setTimeout`, i.e. the
framework's own scheduled continuation, not to the click handler; one is
attributed to `ReadableStreamDefaultReader.read.then`, i.e. the route swap's
payload fetch and decode.

This does not *refute* the plan's claim that deletion, DOM commit, order repair
and layout effects are unsliced — that is a true statement about the code. It
refutes the inference that they are what M2 is measuring on this application:
DOM-commit work would show as style and layout, and there is none. The long
frames are Go work inside one uninterrupted script turn.

### H3 — "The 88-item inbox produces long frames; does virtualization apply?" — **PARTLY CONFIRMED; virtualization is not the lever**

All 88 rows do reach the DOM — 89 `<tr>` (88 + header), verified:

```
go test -tags playwrightgo ./test/playwrightgo/examples/ -run TestAtlasPerfBrowserCommentsInboxShape -v
  /app/comments DOM: documentElements=1015 shellElements=974
  row candidates: [data-comment-id]=0 tr=89 li=0 article=0
```

It is the most expensive route in both lanes — 5.23 ms median native (2.9× the
dashboard), 384 ms warm hydration, and the swap into it produces long frames.

But **1 015 DOM elements is a small tree**, and the cost is not proportional to
it: the same route allocates 3.06 MB and 23 465 allocations natively, ~12
allocations per element, and 63% of that is CSS folding (§2.1). Virtualizing to
20 visible rows would cut roughly 78% of the row work while leaving the
per-element folding cost per rendered row unchanged.

Virtualization would help this route. It is not the first thing to reach for,
because a 1 000-node tree should not cost 5 ms.

### H4 — "Does typed-CSS folding cost anything measurable?" — **CONFIRMED, and it is the largest single item on the render path**

See §2.1. **46% of render CPU, 63% of render allocations**, reproduced twice.

Recorded because it is a real hazard for reading this document later: at the
start of this pass the answer was **no**. `shared/atlas` did not import
`shared/design` at all; only `server/server.go` did, and only to call
`design.Install()` once at boot. The conversion landed during the pass. Any
re-measurement must re-establish the reach fact (`TestAtlasPerfDesignFoldReach`)
before quoting the cost.

---

## 5. What I would fix first, and what I would not

### 5.1 Fix first — make `css.New` cheap on a repeat fold. Framework.

The single highest-value change available. `canonicalize` costs 23.9 µs and
26.7 KB per four-bundle call *before* the memo cache is consulted, so the cache
saves under a quarter of the cost. Some key that does not require rebuilding the
canonical string on every call would remove most of 46% of CPU and 63% of
allocations on the render path — and since flat time is 73.76% Go runtime, cutting
allocations also cuts the GC pressure behind it.

This is not a design proposal; the mechanism belongs to whoever implements it.
What the measurement supports is the target and its size.

### 5.2 Fix second — hoist `design.Class` to package vars in `shared/atlas`. Example.

398 sites, 0 hoisted, and `shared/design/doc.go` already prescribes it. Requires
no framework change and is independently verifiable with the same benchmark.
Mechanical for the many sites whose arguments are constant; the tone- and
step-parameterised ones (`design.StatusChip(tone)`, `design.Display(step)`) need
a small keyed table instead.

Do both 5.1 and 5.2. Either alone leaves half the cost.

### 5.3 Fix third — stop the JSON round trip in `page.go`'s `decode[T]`. Example.

9.5% of render CPU and 8.6–9.5% of render allocations, for a marshal-then-unmarshal
of data that was already decoded once by `PayloadFromSSRBootstrap`. Decoding the
payload into typed structs once, at bootstrap, removes it.

### 5.4 Would NOT fix — reconciliation, on this evidence

`internal/runtime`'s flat CPU on the native render path is **0.43%**. In the
browser it is 10.69% of cold-boot samples, against 19.98% Go runtime and 13.19%
V8 compile. The plan's conclusion — *"Closing it means making reconciliation and
commit faster"* — is not what a real application's profile says. Reconciliation
is not free, but it is not the thing standing between Atlas and its budgets, and
the three items above are each larger and cheaper to fix.

Specifically, `cloneElementProps` does not appear in the top 25 of the allocation
profile at all. `PRODUCTION_READINESS.md`'s refutation of it now reproduces on a
real application, not just on the microbenchmarks.

### 5.5 Would NOT fix yet — bundle size, by reconciler-side work

Atlas's client, built exactly as M5 measures (`-trimpath -ldflags="-s -w"`,
maximum compression):

```
GOOS=js GOARCH=wasm go build -trimpath -ldflags="-s -w" -o "$TMPDIR/atlas_perf_client.wasm" ./examples/server/atlas-commerce-os/client
ATLAS_PERF_WASM="$TMPDIR/atlas_perf_client.wasm" ATLAS_PERF_WASM_STRIPPED=1 \
  go test ./examples/testing/atlas-perf/ -run TestAtlasPerfWasmSize -v
```

(The size test **skips** unless a bundle is named. Brotli at maximum compression
over a 20 MB bundle costs ~60 s and must not be a side effect of
`go test ./...`; with that gate the whole package runs in 4 s.)

| | raw | gzip | brotli |
|---|---:|---:|---:|
| Atlas client | 19.51 MB | **4.55 MB** | **3.17 MB** (30.4% under gzip) |
| M5 target | — | 1.60 MB | — |
| two-artifact `app.wasm` (recorded) | 6.33 MB | 1.73 MB | 1.26 MB |
| Go js/wasm floor (recorded) | 1.80 MB | 0.54 MB | — |

**Atlas is 2.85× the M5 gzip budget** — 4.55 MB against 1.60 MB, over by 3.03 MB,
which is 303% of the 1.02 MB addressable budget after the 541 KB Go floor. The
synthetic two-artifact example misses M5 by 8%; a real application misses it by
185%. That is the size story's most important missing number.

The stripped build is only 3% smaller than the development build (20.46 MB vs
21.15 MB raw), so `-s -w` is nearly a no-op here and the size is real code.

Since 75% of Atlas's cold hydration is compiling this bundle (§3.1), **bundle size
is Atlas's initial-render problem**, and route splitting (P6.2) and streaming
instantiation (P6.3) address it where reconciler work does not.

---

## 6. Two things the harness found by failing

Both cost real debugging time and are recorded so the next reader does not
re-derive them.

### 6.1 Reading layout changes what Atlas renders

`assertAtlasSpecOperatorSessionVisible` uses Playwright `InnerText`, which forces
style and layout. On a settled `/app/dashboard`, an anchor count taken
immediately before it read **33**; immediately after, **9**, with zero console
errors and zero page errors. A six-second sampling probe that reads no layout
shows the same shell sitting at 33 for the whole window:

```
go test -tags playwrightgo ./test/playwrightgo/examples/ -run TestAtlasPerfBrowserShellStability -v
  t=3394 ms  anchors=29  /app/ links=28  shellRoot=true  #app children=1  path=/app/dashboard
  (no change across 30 samples)
```

Any measurement taken after a layout read describes a different tree.

### 6.2 The operator rail is not reliably present after a client-side route swap

Driving repeated in-app route swaps could not be made to work. After swapping to
`/app/comments`, `page.WaitForFunction` polling for `a[href^='/app/']` succeeds,
and a single-shot query moments later finds none. Playwright Locators and
main-world `page.Evaluate` **agree** it is absent, so this is not a
selector-engine artefact. Anchor counts on a swapped-to route were observed at
29, then 9, then 0 across 189 polls in 10 s. It reproduces with the V8 profiler
removed, so it is not instrument-induced.

**Cause unidentified.** It is recorded as an open question rather than explained,
and it is why §3.4's route-swap numbers come from single swaps with settles
rather than from a loop, and why the steady-state CPU profile drives keystrokes
instead of navigation.

If this is a real product defect it is more important than anything else in this
document, and it deserves its own investigation.

---

## 7. What could not be measured, and why

| | Why |
|---|---|
| **Go-side phase totals on Atlas** (render / diff / commit / effect / cleanup) | They come from `gwcruntime.GetGlobalRuntime().Inspect().Profiling.PhaseTotals`, which needs an in-app probe registered from `main` — the pattern `examples/testing/v5-load-harness/probe.go` uses. Atlas's client registers none, and this pass does not own `examples/server/atlas-commerce-os/client`. **Adding that probe is the single highest-value follow-up**: it would split the browser's 10.69% `internal/runtime` bucket into the four phases the plan reasons about, and would settle H2 directly instead of by inference from LoAF. |
| **GC pause on Atlas (M7)** | Same probe. `MemStats.PauseNs` is only reachable from inside the wasm app. |
| **M3 p95** | The interaction probe yields n=12 per run, far below the 200 the tail-reliability guard requires. The max (96 ms) is gated independently and is quotable; the p95 is not, and the harness refuses to print it. Raising the input rate would fix this. |
| **A production-build browser profile** | Every browser number here is from a **development** build (no `-tags production`). `runtime.traceback2` and `runtime.printArgs` appear in the steady-state profile at 3.42% and 1.71%, which is stack-trace formatting on the render path — consistent with a dev-only guard. Production numbers will differ and a production arm was not run. |
| **`-race`** | Unavailable on windows/arm64; cgo has no usable toolchain. Not attempted, per the brief. |
| **Bench-drift comparison** | `docs/benchmarks/latest.json` is amd64 from 2026-03-25 with 249 benchmarks; a local arm64 run produces 197. The baseline was deliberately **not** regenerated — writing an arm64 baseline would corrupt the gate for the amd64 runner. Atlas was measured fresh instead. |
| **Parallel-render throughput** | `internal/runtime/reconciler.go` keeps `currentFiber` in package globals, so native rendering is single-flight per process. Every render in the native lane holds a mutex. Nothing here says anything about concurrency or contention. |
| **Whether hoisting `design.Class` actually recovers 46%** | The counterfactual was not built. The claim is that `design.Class` *costs* 46% of the render path, measured twice; that removing it recovers all of that is an inference and should be verified with the same benchmark after the change. |

---

## 8. Reproduction, end to end

```sh
# 0. one Atlas server; ports 8080/8096/8097/8199/8231/8299/8402/8733 were taken here
ATLAS_ADDR=127.0.0.1:8531 go run ./examples/server/atlas-commerce-os/server &

# 1. fixtures (only needed when the server's payloads change)
ATLAS_PERF_CAPTURE=1 go test ./examples/testing/atlas-perf/ -run TestAtlasPerfCaptureFixtures -v

# 2. native: shapes, idempotence, CSS-fold reach
go test ./examples/testing/atlas-perf/ -run 'TestAtlasPerfRouteShapes|TestAtlasPerfDesignFoldReach' -v

# 3. native: throughput and allocations
go test ./examples/testing/atlas-perf/ -run '^$' \
  -bench 'BenchmarkAtlasPerfRouteRender|BenchmarkAtlasPerfPayloadDecode|BenchmarkAtlasPerfDesignClass' -benchtime 50x

# 4. native: profiles + attribution
go test ./examples/testing/atlas-perf/ -run '^$' -bench BenchmarkAtlasPerfAllRoutes \
  -benchtime 6000x -cpuprofile atlas_cpu.out -memprofile atlas_mem.out
go tool pprof -top -cum -nodecount=45 atlas_cpu.out
go tool pprof -sample_index=alloc_space -top -cum -nodecount=35 atlas_mem.out
go test ./examples/testing/atlas-perf/ -run TestAtlasPerfAttributionRecipe -v

# 5. browser (needs examples/static/bin/atlas-commerce-os.wasm to match the tree)
GOOS=js GOARCH=wasm go build -o /tmp/atlas.wasm ./examples/server/atlas-commerce-os/client
mv /tmp/atlas.wasm examples/static/bin/atlas-commerce-os.wasm    # stage-and-move; the path is shared
ATLAS_PERF_BASE_URL=http://127.0.0.1:8531 \
  go test -tags playwrightgo ./test/playwrightgo/examples/ -run TestAtlasPerf -v -timeout 40m

# 6. size
GOOS=js GOARCH=wasm go build -trimpath -ldflags="-s -w" -o /tmp/atlas_stripped.wasm ./examples/server/atlas-commerce-os/client
ATLAS_PERF_WASM=/tmp/atlas_stripped.wasm ATLAS_PERF_WASM_STRIPPED=1 \
  go test ./examples/testing/atlas-perf/ -run TestAtlasPerfWasmSize -v

# 7. stop the server BY PID, never by image name
netstat -ano | grep 127.0.0.1:8531 | grep LISTENING
taskkill /F /T /PID <pid>
```

Cool the machine between profiling runs and take at least two. Every headline
number in this document was reproduced.

# Multithreaded Runtime

Last updated: 2026-03-26

This document describes the proposed framework-owned multithreaded rendering runtime for GoWebComponents.

It is intentionally deeper and more architectural than [WORKERS.md](WORKERS.md) or [FINE_GRAINED_REACTIVITY.md](FINE_GRAINED_REACTIVITY.md).

Implementation backlog: [MULTITHREADED_RUNTIME_TODO.md](MULTITHREADED_RUNTIME_TODO.md)

Use it when you need to answer questions such as:

- how a worker-backed renderer should fit the current runtime
- why the existing `Fiber` runtime should not simply be made concurrent
- what the worker-owned render boundary should look like
- how state, scheduling, hydration, diagnostics, and fallback should work

## Status

This page is a design document, not a claim that the runtime already ships in this form.

Shipped today:

- app-facing worker primitives in `interop`
- explicit worker pools, message channels, shared buffers, and worker-safe atomics
- explicit fine-grained regions through `ui.ReactiveRegion(...)`

Not shipped today:

- a framework-owned worker-backed render runtime
- worker-rendered component trees
- worker-produced DOM patch application as a public runtime feature

Rule: treat this document as the intended architecture for a future worker-backed runtime slice, not as current framework behavior.

## Short Version

The multithreaded runtime should not be "the current runtime, but on more threads."

It should be a new opt-in render path with these rules:

- the main thread remains the only DOM owner
- workers own pure render and region-local diff work
- state truth remains on the main thread
- concurrency starts at explicit anchored region boundaries
- the cross-thread contract is a serializable render IR and patch IR, not the current `Element` or `Fiber` graph

In practical terms, the right shape is:

```text
owner component
-> explicit parallel region
-> main thread snapshots region inputs
-> one worker shard renders and diffs that region
-> worker emits patch IR
-> main thread validates version and epoch
-> main thread commits DOM updates
```

## Why The Current Runtime Is Not The Right Base

The current renderer is optimized for one runtime, one UI thread, and one mutable in-memory tree.

That is the correct design for the shipped runtime, but it is the wrong transport for worker rendering.

### 1. The current render graph is not serializable

The current runtime relies on:

- `runtime.Element` values with `Type interface{}` and `Props map[string]interface{}`
- live component implementations and wrappers
- hook state stored in mutable `Hooks` objects
- DOM handles stored directly on fibers
- event handlers and closures stored in props or hook state

That means the current runtime tree contains:

- functions
- pointers
- interface-typed values
- DOM adapter handles
- mutable ownership edges

Those are runtime-local structures, not a safe cross-thread protocol.

### 2. The current scheduler assumes one active mutable runtime

The current renderer uses:

- a package-global current fiber cursor
- a process-global runtime singleton
- package-global scheduler locking
- mutable `alternate` links between current and work-in-progress fibers

That is fine for the shipped main-thread runtime.

It is not the right basis for worker-owned render jobs that need:

- stable snapshot input
- isolated worker-local state
- versioned output
- deterministic stale-result dropping

### 3. The current renderer is host-aware all the way through

The shipped `performUnitOfWork(...)` path can:

- create DOM nodes
- claim hydrated DOM
- compare hydrated DOM
- drive commit and cleanup directly

Workers cannot own those responsibilities because browser DOM stays on the main thread.

### 4. The current fine-grained path already suggests the correct boundary

The best clue in the current codebase is the explicit fine-grained region model:

- `ui.ReactiveRegion(...)` is explicit
- region boundaries are anchored and narrow
- hook-driven rerenders still take priority
- the current runtime already distinguishes full-owner rerender from narrow region rerender

That means the multithreaded runtime should grow from explicit region boundaries, not from whole-tree speculative concurrency.

## Design Goals

The multithreaded runtime should:

- preserve browser correctness by keeping DOM ownership on the main thread
- reduce main-thread render and diff work for hot anchored display regions
- keep the default authoring model component-first and hook-first
- reuse the existing worker substrate instead of inventing a second thread API
- make stale worker results cheap to drop
- degrade safely to local main-thread rendering
- support structured diagnostics and strong testability from the first slice

## Non-Goals

The first multithreaded runtime slice should not try to:

- make every component render on workers
- move hooks, effects, refs, or DOM interop into workers
- make workers claim hydrated DOM
- create a second general-purpose concurrent `Fiber` runtime
- replace the current runtime for ordinary app code
- require cross-origin isolation for all apps

Rule: the first slice is an opt-in acceleration path for measured hotspots, not a new default renderer.

## Core Premise

The browser concurrency model is the design constraint:

- workers can compute
- workers cannot own DOM
- shared memory is optional
- message passing is always available

That means the runtime split must be:

```text
main thread:
  state authority
  component ownership
  DOM ownership
  hydration ownership
  event ownership
  final commit

worker:
  pure region render
  region-local diff
  IR caching
  patch generation
```

## Unit Of Concurrency

The unit of concurrency should be the explicit parallel region.

That region must satisfy all of these conditions:

- it is explicitly marked by the app
- it is anchored to a stable local DOM area
- its hot values already come from explicit shared sources
- its surrounding owner component can stay structurally stable
- its first shipped behavior can be described as "display-oriented"

Examples that fit the first slice:

- dashboard counters
- inspector readouts
- live status chips
- compact keyed rows with stable host structure
- editor side panels that display derived state frequently

Examples that do not fit the first slice:

- forms with local draft ownership
- route shells
- auth gates
- portal-owned overlays
- error boundary ownership
- components whose hooks must rerun for correctness

## Public Authoring Model

The public API should remain explicit, similar in spirit to `ui.ReactiveRegion(...)`, but worker-safe.

The critical rule is that anonymous closures are not the cross-thread contract.

Instead, the public shape should be based on region registration plus explicit region instances.

Illustrative API shape:

```go
type ParallelSource interface {
    ParallelSourceIDs() []string
}

type ParallelRegionSpec[T any] struct {
    ID      string
    Props   T
    Sources []ParallelSource
}

func RegisterParallelRegion[T any](id string, render func(T) ui.Node)

func ParallelRegion[T any](spec ParallelRegionSpec[T]) ui.Node
```

Important properties of this shape:

- `ID` names the renderer implementation
- `Props` is serializable worker input, not a live closure
- `Sources` define the state dependency boundary explicitly
- the owner component still chooses where the region appears

Rule: if a region cannot be described by stable ID plus serializable props plus explicit shared sources, it is not yet a worker-renderable region.

## Region Registration

The runtime needs a registry-based model because workers cannot receive live Go function values from the main thread.

Both the main-thread bundle and worker bundle should register the same region IDs.

Example:

```go
func init() {
    ui.RegisterParallelRegion("dashboard.hot-panel", renderDashboardHotPanel)
}
```

That registry should support:

- stable logical IDs
- renderer lookup by ID
- protocol version reporting
- capability reporting for region types if needed later

The registry should not support:

- implicit reflection-based discovery
- closure serialization
- worker-side execution of arbitrary unregistered component functions

## Runtime Split

The multithreaded runtime should be structured as two cooperating runtimes.

### Main-thread runtime responsibilities

- render the owner component tree normally
- mount the region shell and initial local content
- snapshot declared source values
- assign region instances to worker shards
- deliver mount and update jobs
- own version and epoch validation
- own DOM node indexing
- apply patch IR
- handle fallback to local rendering
- own event binding and hydration

### Worker runtime responsibilities

- resolve a region renderer by registered ID
- render the region from serializable props plus source snapshot
- build worker-local render IR
- diff against previous region IR on the same worker
- write patch IR
- emit diagnostics and timing

### Shared responsibilities

- protocol version negotiation
- region lifecycle signaling
- transport capability negotiation
- restart and dispose coordination

## Main-thread Region Coordinator

The main-thread runtime needs one coordinator that owns every live parallel region instance.

Per region instance, it should track at least:

- `regionID`
- `rendererID`
- `epoch`
- `assignedWorkerShard`
- `attached`
- `sourceIDs`
- `lastSnapshotVersion`
- `lastDispatchedVersion`
- `lastCommittedVersion`
- `fallbackMode`
- `domIndex`
- `diagnostics`

The coordinator owns the rule that the main thread is authoritative.

That means:

- local owner rerender can replace or remove the region at any time
- stale worker output is ignored
- worker failures never grant workers the right to mutate DOM directly

## Worker-Affinity Scheduler

The current `interop.OpenWorkerPool(...)` is the right substrate, but not the final scheduler shape.

Region work needs sticky ownership.

### Why sticky ownership matters

If region updates are routed to arbitrary workers:

- every update becomes a cold render
- previous IR is lost or must be transferred expensively
- caches churn
- diff quality falls
- worker-local state becomes meaningless

Instead, each region instance should hash to one worker shard and stay there until:

- the region is disposed
- the worker dies and the region is reassigned
- the scheduler explicitly rebalances during a controlled restart

### Scheduler model

The scheduler should expose:

- region mount
- region update
- region cancel
- region dispose
- worker health events

It should not expose:

- arbitrary pool-wide request dispatch as the region protocol

Rule: `WorkerPool` owns worker count and bounded queueing; the region scheduler owns shard affinity and region lifecycle.

## State And Snapshot Model

State truth should remain on the main thread.

The worker runtime should never become the source of truth for shared state.

### Source declaration

Parallel regions should declare source IDs explicitly, just like the current fine-grained path.

Those sources should come from:

- `state.Atom[...]`
- `state.Derived[...]`
- `state.Select(...)`

### Snapshot rule

For each region update:

- the main thread gathers the declared source IDs
- the main thread reads the current values
- the main thread stamps the snapshot with a monotonic input version
- the worker receives a region-local snapshot payload

Illustrative snapshot:

```json
{
  "region_id": "region-42",
  "epoch": 3,
  "input_version": 18,
  "sources": {
    "dashboard-model-hot": 91,
    "dashboard-model-status": "healthy"
  },
  "props": {
    "title": "Orders"
  }
}
```

### Versioning rules

The main thread should track:

- source version
- dispatched input version
- committed patch version

The worker should track:

- last accepted input version
- last produced patch version
- last worker-local IR version

Rule: worker output older than the main thread's latest accepted version is dropped before commit.

## Control Plane

The control plane should use `MessagePort`.

That channel is for:

- mount
- update
- cancel
- dispose
- ready
- diagnostic
- patch-ready
- restart coordination

It should remain event-driven even when shared memory is enabled.

The main thread should not rely on blocking waits for control flow.

Illustrative envelope:

```json
{
  "protocol": "gwc.parallel.v1",
  "kind": "update",
  "region_id": "region-42",
  "renderer_id": "dashboard.hot-panel",
  "epoch": 3,
  "input_version": 18,
  "transport": "shared-buffer"
}
```

Recommended control-plane message kinds:

- `ready`
- `capabilities`
- `mount`
- `update`
- `cancel`
- `dispose`
- `patch-ready`
- `diagnostic`
- `restart`
- `pong`

## Data Plane

The data plane should support three transport tiers.

### Tier 1: structured-clone payloads

Use for:

- small props
- small snapshots
- early prototypes
- no-shared-memory environments

Advantages:

- easiest to debug
- works everywhere workers work

Disadvantages:

- repeated cloning cost
- poor fit for larger IR payloads

### Tier 2: binary `[]byte` payloads

Use for:

- compact IR payloads
- patch streams that should avoid repeated map and slice allocation churn

Advantages:

- works without cross-origin isolation
- easier to version and validate than arbitrary nested JSON

Disadvantages:

- still copied during transport

### Tier 3: `SharedBuffer`

Use for:

- larger IR pages
- patch pages
- worker-owned region-local scratch arenas
- low-overhead patch publishing

Advantages:

- lower copy cost
- better fit for repeated region updates

Disadvantages:

- requires cross-origin isolation
- more complicated coordination

Rule: the control plane remains `MessagePort`-driven in every tier. Shared memory optimizes payload movement, not ownership of browser-visible control flow.

## Shared-Memory Layout

The shared-memory path should use fixed-format pages, not ad hoc offsets.

One patch page header can look like this:

```text
bytes 0..3    magic            "GPR1"
bytes 4..7    protocol version
bytes 8..11   page kind
bytes 12..15  status flags
bytes 16..23  region epoch
bytes 24..31  input version
bytes 32..35  op count
bytes 36..39  string table offset
bytes 40..43  node table offset
bytes 44..47  patch op offset
bytes 48..51  payload length
bytes 52..63  reserved
```

Important constraints:

- every page must be self-describing
- every page must carry protocol version
- every page must carry region epoch and input version
- every reader must validate bounds before reading
- malformed pages must fail closed

`WaitInt32(...)` and `NotifyInt32(...)` can be used between workers if needed for worker-side scratch coordination, but main-thread patch readiness should still be signaled through the port protocol.

## Render IR

The worker-safe render IR should be a compact region-local tree format.

It must be:

- serializable
- deterministic
- cheap to diff
- independent of DOM node handles
- independent of hook objects

### Node identity

Each region-local node should have a stable `node_id`.

That ID must be stable across renders for structurally equivalent nodes inside the same region instance.

Suggested sources of stability:

- keyed child identity
- deterministic local path order
- stable renderer-owned anchors

### Node record

Each node record should include:

- `node_id`
- `kind`
- `tag_id`
- `flags`
- `key_hash`
- `first_child_index`
- `next_sibling_index`
- `prop_start`
- `prop_count`
- `text_id`

### Node kinds

The first slice only needs:

- text
- host element
- region root marker

Later slices may add:

- event slot placeholder
- keyed fragment marker

### String table

Strings should live in a separate deduplicated table:

- text content
- attribute names
- attribute values
- tag names if not preindexed

That reduces patch size and simplifies binary encoding.

### Prop encoding

Props should be canonicalized before entering the worker IR.

The worker should not receive arbitrary `map[string]interface{}` with live Go values.

Instead, the main thread should normalize to a renderable prop subset:

- strings
- booleans
- numeric scalar values when the host contract permits them
- canonical style maps
- canonical attribute names

## Patch IR

The worker should emit ordered host operations, not a new DOM tree.

First-slice patch ops should include:

- `set_text`
- `set_attr`
- `remove_attr`
- `set_style`
- `remove_style`
- `insert_node`
- `remove_node`
- `replace_subtree`
- `move_keyed_child`

Each op should be explicit and self-contained.

Example conceptual op:

```text
op: set_text
target_node_id: 17
text_id: 42
```

### Patch ordering rules

Patch order must preserve browser-visible coherence.

The coordinator should apply a region patch as one ordered unit:

- validate epoch and input version first
- resolve every target node ID
- reject the whole patch if structural prerequisites are invalid
- apply the patch
- publish new committed version

Rule: after commit returns, the DOM must reflect either the old region version or the new region version, not a partially applied mix.

## DOM Indexing

The main thread needs a region-local DOM index.

That index maps:

- `regionID + nodeID -> DOM node`

The index should be updated only by the main-thread commit path.

Workers should never observe DOM node handles.

The first slice can rebuild portions of the index on subtree replacement.

Later slices can optimize for incremental updates if needed.

## Local-First Initial Render

The initial frame should render locally on the main thread.

That rule is important for:

- first paint
- loading states
- hydration safety
- non-worker environments
- easier fallback

Recommended flow:

```text
owner component renders
-> region renders local initial frame
-> DOM mounts normally
-> region coordinator attaches worker
-> worker builds baseline IR from the same input
-> later updates take the worker path
```

This avoids the worst product failure mode: a blank or delayed first paint because a worker has not responded yet.

## Worker-Side Render Runtime

The worker runtime should be intentionally small.

Per worker shard, it should keep:

- renderer registry
- region states by `regionID`
- previous render IR per region
- string tables or scratch arenas
- last accepted input version per region
- diagnostics and timing counters

Per region state, it should keep:

- `rendererID`
- `epoch`
- `lastInputVersion`
- `lastIR`
- optional scratch memory handles

Worker job flow:

```text
mount:
  resolve renderer
  render region
  build baseline IR
  store IR
  optionally emit baseline diagnostic

update:
  reject stale version
  render region from snapshot
  build next IR
  diff prev IR -> next IR
  emit patch IR
  store next IR

dispose:
  delete region-local state
```

## Interaction With Hooks And Fine-Grained Reactivity

The worker-backed runtime should not replace the current fine-grained runtime.

It should sit above it as a stricter, more expensive, more explicit optimization tier.

Recommended escalation order:

1. normal hook rerender
2. `state.Select(...)`
3. `ui.ReactiveRegion(...)`
4. worker-backed `ParallelRegion(...)`

Why this order matters:

- most problems do not need workers
- explicit selectors already remove a lot of churn
- current fine-grained regions avoid owner rerenders without worker complexity
- worker-backed rendering only makes sense for proven hotspots

Correctness rule:

- when hook-driven owner rerender and worker-region patch results conflict, the owner rerender wins

That matches the existing fine-grained policy.

## Transitions

Transition semantics should stay main-thread-owned.

If a source update is deferred through `StartTransition(...)`:

- the source snapshot should not be published to the worker until the deferred update is accepted
- worker output should inherit the same input version ordering rules

Rule: worker rendering changes where render work happens, not the semantic priority model for state updates.

## SSR And Hydration

SSR and hydration should remain main-thread render concerns.

### SSR

Server rendering should stay on the ordinary server render path.

No worker-backed runtime is needed on the server for the first slice.

### Hydration

Hydration should remain main-thread-owned because it needs:

- DOM claim
- mismatch detection
- subtree fallback
- event wiring

Worker-backed regions should attach only after hydration is complete for the owning subtree.

Recommended hydration flow:

```text
SSR markup exists
-> main thread hydrates normally
-> region remains on local runtime during hydration
-> hydration settles
-> coordinator attaches eligible region to worker shard
-> later updates use worker path
```

Rule: workers never claim DOM nodes and never decide hydration mismatch recovery.

## Event Model

The first slice can remain display-only.

That is the safest path because interactive regions create new problems:

- handler identity
- event-slot binding
- local state ownership
- focus management
- form control value synchronization

If interactive worker-rendered regions are added later, the model should be event-slot based:

- worker describes event slot IDs in the render IR
- main thread binds those slots to DOM nodes
- main thread dispatches semantic events back through the region control plane

Rule: event ownership stays on the main thread even if render topology came from a worker.

## Error Handling And Fallback

Failure should always degrade to correctness.

The coordinator should support these fallback reasons:

- worker unavailable
- unknown renderer ID
- unsupported region feature
- malformed control-plane message
- malformed patch page
- stale epoch or input version
- worker restart before patch completion
- transport capability mismatch
- explicit runtime safety rejection

Fallback behavior:

- keep or return to local main-thread rendering for that region
- mark the region as locally owned until a controlled reattach
- record a diagnostic

Rule: worker failure is a performance failure first, not a correctness failure first.

## Diagnostics And Observability

The worker-backed runtime should not be opaque.

Per region, the runtime should surface:

- assigned worker shard
- renderer ID
- epoch
- latest input version
- latest committed patch version
- transport mode
- worker render duration
- diff duration
- patch byte size
- round-trip duration
- dropped stale patch count
- local fallback reason

Devtools should be able to answer:

- is this region currently local or worker-backed
- which worker owns it
- how often it updates
- whether patches are being dropped as stale
- whether shared memory is actually active
- why a region fell back

## Security And Deployment

The multithreaded runtime inherits the worker deployment constraints.

Important rules:

- shared memory is optional, not mandatory
- cross-origin isolation is required only for the shared-memory tier
- the runtime must keep a structured-clone fallback path
- diagnostics must not leak source snapshots containing secrets

The runtime must not assume:

- all deployments can use `SharedArrayBuffer`
- all routes can keep cross-origin isolation
- all apps want worker-backed rendering on every page

## Testing Strategy

The test surface should be explicit from the start.

### Positive-path tests

Cover:

- local-first mount
- worker attach after first frame
- sticky worker-affinity updates
- shared-memory and non-shared-memory paths
- text and host-prop patch commits
- keyed child updates inside allowed region shapes

### Negative-path tests

Cover:

- unknown region IDs
- invalid renderer registration
- malformed control-plane messages
- malformed patch pages
- invalid op ordering
- stale epoch and version output
- worker disposal during active jobs

### Edge-case tests

Cover:

- owner rerender versus worker patch conflict
- transition-deferred source updates
- hydration attach timing
- rapid dispose and remount churn
- repeated source writes collapsing to one visible result

### Fuzz tests

Cover:

- control-plane decoding
- binary page header parsing
- patch IR decoding
- patch application ordering
- shared-buffer page readers

Rule: malformed worker output must never panic the coordinator or partially corrupt the DOM.

## Benchmarking Strategy

The worker-backed runtime is only justified if it wins for real workloads.

Benchmark comparisons should include:

- hook-only rerender
- current `ui.ReactiveRegion(...)`
- worker-backed region with structured-clone transport
- worker-backed region with shared-memory transport

Representative workloads:

- dashboard counters
- inspector panels
- keyed readout lists
- editor sidebars with hot derived state

Measure at least:

- main-thread render time
- main-thread commit time
- worker render and diff time
- round-trip latency
- bytes transferred
- total allocations

Rule: if the worker path is not measurably better for a hotspot shape, stay on the current fine-grained runtime.

## Rollout Plan

The runtime should land in narrow phases.

### Phase 1

- registry-based parallel region API
- local-first mount
- display-only regions
- structured-clone transport
- worker-local baseline IR
- text and attribute patches

### Phase 2

- sticky shard scheduler
- binary patch payloads
- keyed child patch support
- diagnostics and devtools surfacing

### Phase 3

- shared-memory transport
- larger-region patch arenas
- richer host subtree support

### Phase 4

- optional event-slot model
- limited interactive regions if benchmarked and justified

Rule: do not start with generalized concurrent component rendering.

## Deferred Areas

These are intentionally out of scope for the first multithreaded runtime slice:

- general worker-owned hook execution
- effect execution on workers
- ref ownership on workers
- portal ownership on workers
- route-shell rendering on workers
- error-boundary ownership on workers
- worker-owned hydration
- full concurrent `Fiber` parity

## Open Questions

Questions that still need concrete design decisions:

- what exact exported names should the public region API use
- whether region props should be generic typed values only or support one lower-level raw payload path
- whether region registration should happen through `init()`-time registration only or also support explicit boot-time registration
- how much keyed child movement the first patch IR should support before falling back to subtree replacement
- whether region-local DOM indexing should be rebuilt eagerly or lazily after subtree replacement
- whether a binary IR should be introduced in phase 1 or after the structured-clone prototype proves the product value

## Final Rule

The right mental model is not "parallelize the current runtime."

The right mental model is:

- keep the current runtime as the default renderer
- keep workers as the compute substrate
- add one new explicit worker-backed region runtime above the current fine-grained path
- preserve main-thread ownership of browser truth

That gives the project a realistic path to multithreaded UI acceleration without turning the current runtime into an unsafe or overcomplicated concurrent system.

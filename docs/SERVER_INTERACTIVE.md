# Server-Interactive Experiments

This page records the minimum scope for any future server-owned interactive rendering experiment.

Use it as a containment boundary, not as a claim that the feature is part of the shipped core runtime.

## Current Position

Server-owned interactive rendering is not a current product direction for core. If it is explored at all, it should remain an explicitly separate experiment or sibling package.

The first exploration should stay small enough that it can be judged on its own merits without dragging the whole framework into a websocket-backed runtime.

## Runtime Assumptions Audit

The current runtime still assumes a browser-owned, in-process render loop in several places:

- hooks and context read from a global `currentFiber`, so render state is implicit instead of session-scoped
- `GoUseState`, `GoUseEffect`, `GoUseMemo`, `GoUseCallback`, `GoUseRef`, `GoUseId`, `GoUseAtom`, `GoUseFunc`, and `GoUseContextValue` all assume they are running during a local synchronous render
- state setters call back into the same runtime and schedule local rerenders instead of emitting transport messages
- the scheduler uses local idle callbacks and timeouts, and `ScheduleUpdateForFiber` marks fibers dirty inside the same process
- commit logic writes directly through `DOMAdapter`, mutating live nodes, moving portal children, and finalizing hydration against existing browser DOM
- event handling wraps browser-native event objects and `syscall/js` values, so `GoUseFunc` expects a direct DOM listener instead of a serialized event payload
- the atom registry keeps values and subscriptions in one runtime instance, so it currently behaves like shared client memory rather than a per-session server source of truth
- hydration and SSR reuse the browser DOM as the source of truth for reconciliation, which is incompatible with a server-owned renderer unless that flow is replaced with a transport-driven patch model
- browser state helpers still assume history, hash, storage, and reload APIs that only exist in the client runtime

To make a server-interactive mode viable, the experiment would need to abstract at least:

1. render context and fiber ownership
2. event transport and acknowledgement
3. session-scoped state and subscription tracking
4. DOM patch emission and browser-side application
5. navigation and reconnect state outside the browser-only helpers

## Transport Shape Evaluation

The experiment should not treat every interaction as a new HTML document.

### Full HTML Streaming

Full HTML streaming is a good fit for initial page delivery, reconnect fallback, or SSR progress updates. It is a poor fit for every interactive change because it repaints too much, makes state preservation awkward, and turns a small user action into a document-sized response.

Use it for:

- bootstrap HTML
- reconnect recovery when the interactive channel is lost
- fallback rendering if the interactive protocol fails completely

Do not use it as the normal click-by-click transport for the experiment.

### Tree Patch Messages

Tree-patch messages are the best framework-level abstraction because they preserve a logical view of the UI instead of leaking raw DOM details into application code.

They are a better long-term contract than whole-document streaming because they can represent component updates, subtree replacement, and state ownership without pretending the browser should always reparse HTML.

The downside is that the browser still needs a concrete patch applier, so the experiment would still need to define a message vocabulary instead of hand-waving the transport away.

### DOM-Op Diffs

DOM-op diffs are the lowest-risk first experiment because the current runtime already thinks in DOM operations when it commits work.

That makes the wire model easy to measure and easy to compare against the existing client-owned commit path, but it should stay behind a transport boundary so the protocol does not become permanently coupled to browser implementation details.

### Recommendation

For the first experiment, use WebSocket transport with JSON-shaped messages and DOM-op style patches.

That choice is pragmatic because:

- the repo already uses WebSocket tooling for live reload
- the browser needs a bidirectional channel for user intent and server responses
- the current runtime commit model already maps naturally to append, remove, set-attribute, and set-text operations

Keep tree-patch messages as the conceptual model that the server owns, but do not start with full HTML streaming as the interactive protocol.

## Latency And Offline Expectations

The first experiment should stay honest about what latency-sensitive UX it can support.

### Must Stay Responsive

These interactions should still feel usable with a moderate round trip to the server:

- primary buttons, toggles, and small menu actions
- form submit actions
- route-like state changes that naturally wait for server confirmation
- bounded workflow steps such as approve, reject, save, and dismiss

### Should Avoid Server Ownership

These interaction classes are a poor fit for a server-owned first experiment:

- freeform typing that needs a character-by-character response
- drag, resize, and pointer-tracking interactions
- canvas, game, or animation-heavy surfaces
- continuous scrubbing, scrolling effects, or any control that expects near-zero input lag

### Reconnect Expectations

If the connection drops, the experiment should:

- keep the last known UI visible instead of blanking the surface
- mark the session as reconnecting or stale
- resubscribe or resync from the server before accepting more actions
- treat the server as authoritative after reconnect, even if the browser had a stale local view

### Offline Expectations

The first experiment should not promise full offline operation.

If the browser is offline or the transport is unavailable:

- prefer a clear reconnect or unavailable state over pretending interaction is still live
- do not queue arbitrary writes unless the reference app explicitly models durable replay
- allow only the minimal fallback path needed to keep the session understandable

## Minimum Experiment Scope

The first experiment should limit itself to:

- event transport from browser to server
- server-side ownership of interactive state
- DOM diff or patch streaming back to the browser
- reconnect handling after the transport drops
- a small reference app that demonstrates one narrow workflow end to end

The experiment should answer these questions:

1. Can the browser send user intent to the server with acceptable latency?
2. Can the server own the state and produce a consistent next UI?
3. Can the browser apply the response without inventing a second unrelated runtime model?
4. Can reconnect restore a sensible interactive session?

## Out Of Scope

The first experiment should not try to solve:

- full parity with the browser-owned runtime
- partial hydration or event replay for a new transport model
- arbitrary DOM patch protocol design beyond the narrow reference app
- offline-first behavior
- generalized multi-session collaboration
- replacing SSR or hydration as the default framework path

## Reference App Shape

If the experiment proceeds, the first reference app should be small and bounded:

- one or two interactive screens
- modest state ownership requirements
- a clear reconnect story
- a narrow transport path that can be measured directly

Good candidate shapes are admin-style workflows, operator consoles, or a small control surface with a few stateful actions.

## Success Criteria

The experiment is only worth continuing if it can demonstrate:

- predictable state ownership boundaries
- acceptable round-trip interaction latency for the chosen app shape
- a clear failure mode when the connection drops
- a stable reconnect path that does not corrupt state
- a smaller implementation story than building the same feature as a client-owned app

If those conditions do not hold, the feature should remain an experiment and not be promoted into the core runtime.

## Security And Scalability Review

The first experiment should be sized for a small number of authenticated sessions, not for open-ended multi-tenant fanout.

### Per-Session Memory Cost

Each connected browser session will need its own runtime state, subscription set, queued update state, and reconnect bookkeeping.

The experiment should therefore assume:

- one bounded session object per active browser session
- explicit cleanup when the socket closes or the session expires
- no unbounded retention of stale render trees, queued patches, or disconnected session snapshots

### Multi-Tenant Isolation

The runtime must not let one tenant observe or mutate another tenant's session state.

The safe baseline is:

- separate runtime ownership per authenticated session
- session- or user-keyed state lookup before any patch emission
- no shared mutable channel that can broadcast one user's interactive patch stream to another user's browser

### Auth And Session Propagation

Server-owned interactivity should reuse the same authentication and session source of truth as the rest of the app.

The experiment should require:

- authentication on the initial page request or socket handshake
- revalidation on reconnect before resuming stateful interaction
- rejection of stale or unauthenticated reconnect attempts instead of silent session reuse
- no trust in client-provided session identifiers without server-side validation

### Backpressure And Queueing

The server must defend itself against a slow browser or a bursty interaction stream.

The experiment should:

- cap the number of queued outbound patches per session
- coalesce or drop stale patches when a newer state supersedes them
- close or degrade slow sessions instead of buffering forever
- keep patch generation bounded so one noisy client does not monopolize the server

### Denial Of Service Risks

The easiest way to turn this model into a problem is to let every input event trigger expensive server work.

The experiment should therefore avoid:

- unbounded message sizes
- high-frequency event streams without throttling
- expensive tree diffs on every pointer or key event
- reconnect storms that can create repeated full-resync work without limits

### Practical Recommendation

For the first experiment:

- keep the app small and authenticated
- bound the per-session memory budget
- rate-limit or coalesce inbound intents
- reconnect from a server-authoritative snapshot when needed
- treat shared-room collaboration and large fanout as out of scope until the smaller model is proven

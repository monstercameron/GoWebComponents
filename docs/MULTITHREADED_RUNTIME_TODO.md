# Multithreaded Runtime TODO

Last updated: 2026-03-26

This backlog tracks the proposed worker-backed multithreaded runtime described in [MULTITHREADED_RUNTIME.md](MULTITHREADED_RUNTIME.md).

It is implementation-facing, TDD-first, and intentionally narrower than the main framework backlog in [TODO.md](TODO.md).

## At A Glance

- This file is the execution backlog for the future parallel rendering runtime.
- Work should proceed one unchecked item at a time.
- Prefer the smallest focused failing test before each implementation step.
- Keep the scope narrow enough that one validation run can prove the item.
- Use this file for runtime2 planning and execution, not as a changelog.

## TDD Rules

For each unchecked item:

1. Write the smallest focused failing test that proves the missing behavior.
2. Reproduce the failure with the narrowest validation command.
3. Make the smallest root-cause change that satisfies the test.
4. Re-run the focused validation.
5. Record the validation result near the completed item if the item is checked.
6. Stop before taking the next unchecked item.

Execution rules:

- Do not batch unrelated runtime slices.
- Do not start implementation before the test shape is concrete.
- Do not widen APIs until a failing test justifies the change.
- Prefer package tests over example-app tests until a feature crosses package boundaries.
- Add fuzzing only after the deterministic contract for the slice exists.
- Treat benchmarks as proof of tradeoffs, not as a replacement for correctness tests.

## Scope

This file covers:

- worker-backed parallel region rendering
- public region contracts and runtime2 package boundaries
- region scheduling and worker affinity
- message, binary, and shared-memory transport
- render IR and patch IR
- DOM commit and fallback behavior
- hydration, diagnostics, failure modes, tests, and benchmarks

This file does not cover:

- unrelated generic worker features
- user-facing docs unrelated to the runtime2 effort
- replacing the current runtime as the default renderer

## 1. Package Skeleton And Capability Contracts

- [ ] Create a dedicated package boundary for the multithreaded runtime.
	Scope: add a new package or package subtree that isolates the worker-backed renderer from the shipped runtime.
	TDD:
	- add a package compile smoke test
	- add a constructor or zero-value guard test
	- add an internal package-boundary test that proves the new code does not require the current runtime singleton

- [ ] Add a package-level capability report for the multithreaded runtime.
	Scope: define what the runtime can report about worker support, `MessagePort`, `SharedBuffer`, and optional transport tiers.
	TDD:
	- report defaults correctly when no worker capabilities are present
	- report structured-clone support independently from shared memory
	- report disabled or unsupported features without panicking

- [ ] Add protocol version constants for the region runtime.
	Scope: version the control plane and data plane before implementation expands.
	TDD:
	- matching versions compare equal
	- mismatched versions are rejected with an actionable error
	- empty or unknown version values fail validation

- [ ] Add capability-negotiation fixtures for tests.
	Scope: create deterministic fixtures for protocol version, worker features, and transport availability.
	TDD:
	- fixture encode and decode round-trip
	- invalid fixture version fails
	- missing capability fields fail clearly

## 2. Region Identity And Registration

- [ ] Add a stable renderer-ID type for parallel regions.
	Scope: separate logical renderer identity from instance identity.
	TDD:
	- empty renderer ID is rejected
	- whitespace-only renderer ID is rejected
	- valid renderer ID survives round-trip encode and decode

- [ ] Add a stable region-instance ID type.
	Scope: distinguish one mounted region instance from another even when they share the same renderer ID.
	TDD:
	- empty region instance ID is rejected
	- two distinct region instance IDs do not collide
	- reuse of a disposed ID behaves predictably

- [ ] Add a renderer registry for worker-renderable regions.
	Scope: register region renderers by stable ID instead of anonymous closures.
	TDD:
	- register then resolve succeeds
	- duplicate register fails clearly
	- unregistered lookup fails clearly

- [ ] Add registry reset support for tests.
	Scope: make registration tests deterministic without global test pollution.
	TDD:
	- reset clears registered IDs
	- reset after duplicate registration restores clean state
	- reset does not leak renderers between tests

- [ ] Add renderer metadata support in the registry.
	Scope: store optional metadata such as supported prop type version or supported feature flags.
	TDD:
	- metadata is returned for a registered renderer
	- missing metadata uses defaults
	- invalid metadata is rejected during registration

## 3. Public Region Input Contract

- [ ] Add a `ParallelRegionSpec` shape for serializable region inputs.
	Scope: define the public contract for region renderer ID, region instance ID, props, and declared sources.
	TDD:
	- minimal valid spec passes validation
	- missing renderer ID fails
	- missing region instance ID fails

- [ ] Add pre-dispatch spec validation.
	Scope: validate region specs before any worker scheduling or transport encoding.
	TDD:
	- nil or absent props are accepted when the renderer allows it
	- unsupported prop graph fails with a field path
	- unexpected field kinds fail before transport

- [ ] Add prop-serializability checks.
	Scope: reject functions, DOM handles, channels, and other unsupported runtime-local values.
	TDD:
	- primitive props pass
	- nested map and slice props pass when supported
	- function props fail with the offending path
	- unsupported interface value fails with the offending path

- [ ] Add explicit declared-source validation.
	Scope: regions must declare source IDs instead of implicitly reaching into arbitrary shared state.
	TDD:
	- empty source set behaves consistently
	- duplicate source IDs are normalized or rejected consistently
	- source IDs with invalid format fail validation

- [ ] Add source-order normalization.
	Scope: avoid transport churn caused by unstable source ordering.
	TDD:
	- the same logical source set yields the same canonical order
	- already sorted source lists remain unchanged
	- duplicate entries do not survive canonicalization if the policy is dedupe

## 4. Source Snapshot Model

- [ ] Add a source-snapshot envelope type.
	Scope: define region ID, epoch, input version, props payload, and source values in one transportable structure.
	TDD:
	- minimal valid snapshot envelope passes validation
	- missing input version fails
	- missing epoch fails

- [ ] Add monotonic input-version rules.
	Scope: updates must be versioned so stale worker output can be dropped safely.
	TDD:
	- larger versions are accepted after smaller versions
	- equal versions behave consistently for duplicate delivery
	- older versions are rejected where required

- [ ] Add source-value snapshotting from declared IDs.
	Scope: gather values only for explicitly declared source IDs.
	TDD:
	- declared source IDs are included
	- undeclared source IDs are excluded
	- missing declared source values fail clearly

- [ ] Add snapshot consistency rules across multiple sources.
	Scope: ensure one worker update sees one coherent main-thread snapshot.
	TDD:
	- two sources captured in one version remain consistent
	- mixed old and new source versions are rejected if detected
	- empty snapshot payload behaves consistently

- [ ] Add snapshot hashing or stable fingerprint support.
	Scope: give the coordinator a cheap way to detect no-op re-dispatches later.
	TDD:
	- identical snapshots produce identical fingerprints
	- materially different snapshots produce different fingerprints
	- field order does not change the fingerprint if canonicalization is expected

## 5. Control Plane Envelopes

- [ ] Add a shared control-plane envelope format.
	Scope: define common fields for all control-plane messages.
	TDD:
	- required fields are present
	- unknown envelope kind is rejected
	- missing region ID is rejected where required

- [ ] Add a `ready` message contract.
	Scope: allow worker boot and protocol negotiation to complete before mount traffic begins.
	TDD:
	- ready message decodes successfully
	- malformed ready message is rejected
	- duplicate ready messages behave consistently

- [ ] Add a `capabilities` message contract.
	Scope: report transport and feature support from the worker side.
	TDD:
	- capabilities message decodes successfully
	- unsupported capability values are rejected
	- absent optional fields default cleanly

- [ ] Add a `mount` message contract.
	Scope: start a region lifecycle on an assigned worker.
	TDD:
	- valid mount envelope decodes successfully
	- missing renderer ID fails
	- missing snapshot payload fails

- [ ] Add an `update` message contract.
	Scope: deliver a new input version to an already mounted region.
	TDD:
	- valid update envelope decodes successfully
	- update with older input version is rejected where required
	- update for unknown region fails clearly

- [ ] Add a `cancel` message contract.
	Scope: invalidate in-flight work without disposing the region.
	TDD:
	- valid cancel envelope decodes successfully
	- cancel for unknown region behaves consistently
	- repeated cancel messages do not corrupt state

- [ ] Add a `dispose` message contract.
	Scope: end a region lifecycle on the worker and release cached state.
	TDD:
	- valid dispose envelope decodes successfully
	- dispose for unknown region behaves consistently
	- repeated dispose messages do not panic

- [ ] Add a `patch-ready` message contract.
	Scope: notify the main thread that a patch stream is available to read.
	TDD:
	- valid patch-ready envelope decodes successfully
	- missing patch metadata fails
	- wrong transport tier fails validation

- [ ] Add a `diagnostic` message contract.
	Scope: emit stable worker diagnostics without coupling them to patch delivery.
	TDD:
	- valid diagnostic envelope decodes successfully
	- malformed diagnostic payload fails
	- unknown diagnostic subtype behaves consistently

- [ ] Add a `restart` message contract.
	Scope: let the main thread or worker coordinate intentional restart and epoch changes.
	TDD:
	- valid restart envelope decodes successfully
	- missing epoch fails
	- restart with stale epoch is rejected where required

## 6. Main-Thread Region Coordinator

- [ ] Add a coordinator entry type for live regions.
	Scope: record region ID, renderer ID, epoch, assigned worker, latest versions, and fallback state.
	TDD:
	- mount creates an entry
	- dispose removes the entry
	- duplicate mount for the same active region fails or replaces consistently

- [ ] Add coordinator state transitions for mount, update, cancel, dispose, fallback, and restart.
	Scope: make the state machine explicit before DOM commit or worker scheduling logic grows.
	TDD:
	- valid transition sequence succeeds
	- invalid transition sequence fails
	- restart transitions bump epoch

- [ ] Add `lastDispatchedVersion` tracking.
	Scope: record which input version was most recently sent to the worker.
	TDD:
	- initial dispatch sets the field
	- later dispatch updates the field
	- stale dispatch does not move the field backward

- [ ] Add `lastCommittedVersion` tracking.
	Scope: ensure the main thread never commits older worker output after a newer commit.
	TDD:
	- first successful commit sets the field
	- later newer commit updates the field
	- older commit attempt is rejected

- [ ] Add fallback-mode tracking.
	Scope: mark regions that have exited worker-backed mode and should render locally.
	TDD:
	- fallback can be entered explicitly
	- fallback suppresses later stale patch commit
	- dispose clears fallback state

## 7. Worker-Affinity Scheduler

- [ ] Add a worker-shard identity model.
	Scope: distinguish worker instances from pool slots and region instances.
	TDD:
	- worker shard IDs are stable per live worker
	- different live workers have different shard IDs
	- disposed shard IDs are not reused unsafely in tests

- [ ] Add deterministic region-to-shard assignment.
	Scope: route repeated updates for one region to the same shard.
	TDD:
	- repeated assignment for one region yields the same shard
	- different regions can map to different shards
	- assignment changes only when a rebalance or repair policy says it can

- [ ] Add explicit scheduler mount handling.
	Scope: mount should allocate or resolve a shard and enqueue the first region job.
	TDD:
	- first mount dispatches to one shard
	- mount with no available shards fails clearly
	- mount after prior dispose reuses the policy consistently

- [ ] Add explicit scheduler update handling.
	Scope: update should route to the existing assigned shard instead of the generic pool path.
	TDD:
	- update reuses the prior shard
	- update for unknown region fails clearly
	- update after dispose is rejected

- [ ] Add explicit scheduler cancel handling.
	Scope: cancel should invalidate queued or in-flight work for one region.
	TDD:
	- cancel marks the queued job stale
	- cancel suppresses future stale result commit
	- repeated cancel remains safe

- [ ] Add explicit scheduler dispose handling.
	Scope: dispose should clear assignment and release region-local worker state.
	TDD:
	- dispose removes region-to-shard mapping
	- dispose of a queued region removes queued work
	- dispose after fallback behaves consistently

- [ ] Add bounded queueing rules at the scheduler layer.
	Scope: define backpressure separately from the lower-level worker pool.
	TDD:
	- queue below limit accepts work
	- queue at limit rejects new work clearly
	- canceled queued work frees capacity

- [ ] Add worker health tracking.
	Scope: track worker ready, degraded, restarting, and dead states.
	TDD:
	- healthy worker accepts work
	- degraded worker behavior is explicit
	- dead worker triggers reassignment or fallback

- [ ] Add worker replacement flow.
	Scope: replace failed workers without silently losing capacity.
	TDD:
	- replacement worker joins the scheduler
	- regions on dead worker are reassigned or forced to fallback
	- replacement failure enters explicit degraded mode

## 8. Structured-Clone Snapshot Transport

- [ ] Add structured-clone encoding for snapshot envelopes.
	Scope: get the simplest working transport path first.
	TDD:
	- valid envelope round-trips
	- malformed envelope fails decode
	- optional fields default correctly

- [ ] Add structured-clone encoding for mount envelopes.
	Scope: support the first region attach path.
	TDD:
	- valid mount envelope round-trips
	- missing renderer ID fails
	- missing props section behaves consistently if optional

- [ ] Add structured-clone encoding for update envelopes.
	Scope: support subsequent region updates over the simplest path.
	TDD:
	- valid update envelope round-trips
	- corrupted update payload fails
	- stale version validation remains separate from decode success

- [ ] Add structured-clone decoding guards for unexpected field types.
	Scope: fail clearly when the envelope shape is wrong even if a browser transport accepts it.
	TDD:
	- number where string is required fails
	- map where list is required fails
	- missing nested object fails

## 9. Binary Snapshot Transport

- [ ] Add a binary envelope header format.
	Scope: define magic, version, kind, length, and checksum or integrity fields as needed.
	TDD:
	- valid header decodes
	- bad magic fails
	- unsupported version fails

- [ ] Add binary encoding for snapshot envelopes.
	Scope: support compact transport without arbitrary nested map churn.
	TDD:
	- valid binary snapshot round-trips
	- truncated payload fails
	- incorrect length fails

- [ ] Add binary encoding for source values supported in the first slice.
	Scope: define the supported scalar and collection value kinds for snapshots.
	TDD:
	- bool, number, string, and text-like payloads round-trip
	- supported small lists round-trip
	- unsupported value kind fails explicitly

- [ ] Add binary-decoding bounds checks.
	Scope: prevent panics and out-of-range reads.
	TDD:
	- truncated header fails safely
	- truncated body fails safely
	- oversized declared span fails safely

## 10. Shared-Memory Snapshot Transport

- [ ] Add a shared-memory page header for snapshot publication.
	Scope: define page magic, protocol version, generation, payload kind, and payload length.
	TDD:
	- valid page header decodes
	- bad magic fails
	- unsupported version fails

- [ ] Add snapshot publication to a shared-memory page.
	Scope: publish region snapshot bytes into a shared page with explicit generation semantics.
	TDD:
	- one publish writes a readable page
	- second publish increments generation as expected
	- publish larger than page capacity fails clearly

- [ ] Add snapshot read validation from a shared-memory page.
	Scope: ensure readers reject torn or stale page states.
	TDD:
	- valid page reads successfully
	- stale generation is rejected where required
	- incomplete publish is rejected

- [ ] Add fallback from shared-memory transport to message transport.
	Scope: keep behavior correct even when shared memory is unavailable or invalid.
	TDD:
	- unavailable shared memory falls back to message path
	- invalid shared page falls back cleanly
	- fallback preserves region semantics

## 11. Render IR Node Table

- [ ] Add a render-node kind enum for the first slice.
	Scope: define text, host element, and fragment-like structural kinds needed for display-only regions.
	TDD:
	- supported node kinds decode
	- unknown node kind fails
	- invalid zero value fails if disallowed

- [ ] Add a render-node record layout.
	Scope: define stable fields for node ID, kind, child span, prop span, text reference, and flags.
	TDD:
	- valid record decodes
	- invalid child span fails
	- invalid prop span fails

- [ ] Add stable per-region node identity.
	Scope: allow worker patches and main-thread DOM indices to agree on node identity.
	TDD:
	- node IDs are unique within one region
	- duplicate node IDs fail validation
	- zero or invalid node IDs fail when disallowed

- [ ] Add flat child ordering rules.
	Scope: make child order deterministic without tree pointers or Go object identity.
	TDD:
	- sibling order round-trips
	- missing child referenced by span fails
	- overlapping child spans fail

- [ ] Add support for keyed child metadata.
	Scope: preserve keyed diff behavior for supported small keyed lists.
	TDD:
	- valid keyed metadata round-trips
	- duplicate keys in one sibling set fail
	- invalid key hash or missing key payload fails

## 12. Render IR String And Prop Tables

- [ ] Add a string-table format for render IR.
	Scope: dedupe tags, attribute names, text values, and common small strings.
	TDD:
	- duplicate strings share one entry
	- empty string is represented consistently
	- invalid string-table reference fails

- [ ] Add canonical string-table ordering.
	Scope: make IR output stable for tests and diffing.
	TDD:
	- the same logical content yields the same table ordering
	- insertion-order differences do not change the canonical output if canonicalization is expected
	- empty table remains valid

- [ ] Add a prop-record format for the first slice.
	Scope: define how class, style, aria, data, and text-adjacent props are encoded.
	TDD:
	- supported prop kinds round-trip
	- unsupported prop kind fails
	- missing key reference fails

- [ ] Add prop-key canonical ordering.
	Scope: stabilize test output and reduce spurious patch churn.
	TDD:
	- equivalent prop maps emit the same order
	- duplicate prop keys fail or collapse consistently
	- empty prop set remains valid

- [ ] Add style-value normalization rules for the supported first slice.
	Scope: keep worker render output stable for style-like props.
	TDD:
	- repeated logically equivalent styles normalize consistently
	- invalid style payload fails
	- unsupported nested style shape fails

## 13. Patch IR Contract

- [ ] Add patch op codes for the first slice.
	Scope: define op kinds for insert, remove, set text, set attr, remove attr, and keyed move.
	TDD:
	- supported op codes decode
	- unknown op code fails
	- zero or reserved op code fails if disallowed

- [ ] Add a patch-stream header format.
	Scope: version and scope every patch stream before commit logic grows.
	TDD:
	- valid header decodes
	- wrong region ID fails
	- wrong version fails

- [ ] Add insert-op payload validation.
	Scope: ensure new nodes are structurally complete before commit.
	TDD:
	- valid insert op decodes
	- missing parent reference fails
	- invalid sibling anchor fails

- [ ] Add remove-op payload validation.
	Scope: prevent impossible deletion states from reaching DOM commit.
	TDD:
	- valid remove op decodes
	- removing unknown node fails
	- duplicate remove of the same node fails or is coalesced consistently

- [ ] Add set-text-op payload validation.
	Scope: ensure only text-capable targets receive text updates.
	TDD:
	- valid set-text op decodes
	- missing target node fails
	- invalid text reference fails

- [ ] Add set-attr-op payload validation.
	Scope: ensure only supported attr-like props are committed in the first slice.
	TDD:
	- valid set-attr op decodes
	- unsupported attr kind fails
	- missing attr key reference fails

- [ ] Add remove-attr-op payload validation.
	Scope: allow attr cleanup without forcing subtree replacement.
	TDD:
	- valid remove-attr op decodes
	- unknown attr reference fails if required
	- duplicate remove behaves consistently

- [ ] Add keyed-move-op payload validation.
	Scope: preserve stable small-list reorder support without opening arbitrary mutation shapes.
	TDD:
	- valid keyed move decodes
	- missing source node fails
	- invalid destination position fails

- [ ] Add patch-order validation rules.
	Scope: reject patch streams that violate structural ordering guarantees.
	TDD:
	- valid parent-before-child sequence passes
	- child insert before parent fails
	- move after remove of the same node fails

- [ ] Add patch idempotency metadata.
	Scope: let the main thread ignore duplicate delivery safely.
	TDD:
	- duplicate patch stream with the same identity is ignored
	- different patch stream with same version fails if disallowed
	- idempotency state resets on epoch change

## 14. Worker-Side Region Runtime

- [ ] Add worker-side mount handling.
	Scope: resolve the renderer, render initial IR, and cache region-local state.
	TDD:
	- valid mount stores initial region state
	- missing renderer ID fails clearly
	- unknown renderer ID fails clearly

- [ ] Add worker-side update handling.
	Scope: diff the new input against cached region IR on the same worker.
	TDD:
	- valid update produces a patch or a no-op result
	- update for unknown region fails clearly
	- stale update is ignored or rejected consistently

- [ ] Add worker-side cancel handling.
	Scope: invalidate in-flight or queued work for one region.
	TDD:
	- cancel before completion suppresses patch-ready
	- cancel after completion behaves consistently
	- repeated cancel remains safe

- [ ] Add worker-side dispose handling.
	Scope: release cached IR and region-local structures.
	TDD:
	- dispose clears cached state
	- update after dispose fails clearly
	- repeated dispose remains safe

- [ ] Add worker-side restart handling.
	Scope: clear or rebuild region state when epoch semantics change.
	TDD:
	- restart invalidates prior epoch state
	- restart allows fresh mount afterward
	- patch from prior epoch is not emitted after restart

- [ ] Add no-op patch detection.
	Scope: avoid sending meaningless patch traffic when the IR is unchanged.
	TDD:
	- unchanged input produces explicit no-op or no patch-ready
	- changed input produces a real patch
	- no-op state does not regress version tracking

## 15. Main-Thread DOM Index And Commit

- [ ] Add a region-local DOM index.
	Scope: map region node IDs to real DOM nodes without exposing workers to DOM.
	TDD:
	- insert into index succeeds
	- lookup succeeds for registered node
	- missing node lookup fails clearly

- [ ] Add DOM-index cleanup on region dispose.
	Scope: ensure disposed regions do not leak node references.
	TDD:
	- dispose clears indexed nodes
	- repeated dispose leaves index stable
	- partial cleanup does not leave stale mappings

- [ ] Add text-node commit support.
	Scope: apply set-text patches for the first display-only slice.
	TDD:
	- text update changes the correct DOM node
	- text update for unknown node fails
	- duplicate same-value text update behaves as a no-op if expected

- [ ] Add attr-commit support.
	Scope: apply supported attr-like updates to host nodes.
	TDD:
	- class attr update commits
	- data attr update commits
	- unsupported attr kind fails before DOM mutation

- [ ] Add node-insert commit support.
	Scope: create and place newly inserted nodes under the correct parent or anchor.
	TDD:
	- insert under parent succeeds
	- insert before sibling anchor succeeds
	- invalid anchor fails cleanly

- [ ] Add node-remove commit support.
	Scope: remove DOM nodes and clean up index state.
	TDD:
	- remove existing node succeeds
	- remove unknown node fails or no-ops consistently
	- repeated remove does not panic

- [ ] Add keyed-move commit support for the first slice.
	Scope: let small keyed lists reorder without subtree replacement.
	TDD:
	- valid keyed move changes DOM order
	- invalid move target fails
	- move after prior removal is rejected

- [ ] Add patch-commit transaction boundaries.
	Scope: ensure one bad op cannot partially corrupt the region without entering fallback.
	TDD:
	- fully valid patch commits all ops
	- invalid mid-stream patch triggers fallback
	- partial commit state is not left active without explicit policy

## 16. Fallback And Recovery

- [ ] Add local-render fallback entry for one region.
	Scope: let a single region leave worker-backed mode without breaking the rest of the app.
	TDD:
	- fallback swaps the region to local ownership
	- later worker patch for the fallen-back region is ignored
	- other regions keep working

- [ ] Add fallback triggers for transport decode failure.
	Scope: bad envelopes or page state should fail safe instead of corrupting commit logic.
	TDD:
	- malformed control-plane payload enters fallback
	- malformed patch payload enters fallback
	- malformed shared-memory page enters fallback

- [ ] Add fallback triggers for invalid DOM commit state.
	Scope: impossible patch state should abandon worker mode for that region.
	TDD:
	- missing parent anchor enters fallback
	- missing node lookup during required op enters fallback
	- invalid keyed move enters fallback

- [ ] Add worker-death recovery policy.
	Scope: choose between reassignment and fallback explicitly instead of silently degrading.
	TDD:
	- worker death can trigger reassignment when supported
	- reassignment remounts the region with a fresh epoch
	- repair failure enters fallback

- [ ] Add explicit stale-result dropping after fallback.
	Scope: old worker output must never mutate a fallen-back region.
	TDD:
	- stale patch after fallback is ignored
	- stale diagnostic after fallback does not revive the region
	- fresh local state remains authoritative

## 17. Reactivity Integration

- [ ] Add source-change detection for declared region inputs.
	Scope: only relevant atom and selector changes should enqueue region updates.
	TDD:
	- declared source change enqueues an update
	- unrelated source change does not enqueue an update
	- multiple declared source changes can coalesce consistently

- [ ] Add owner-rerender precedence over worker output.
	Scope: owner component structural changes remain authoritative.
	TDD:
	- owner rerender invalidates older patch streams
	- owner removal drops pending worker output
	- owner structural replacement remounts the region

- [ ] Add region unmount semantics when the owner stops rendering the region.
	Scope: worker-backed regions must dispose cleanly when their owner path disappears.
	TDD:
	- owner unmount triggers region dispose
	- pending patch after unmount is ignored
	- remount after unmount starts a new epoch

- [ ] Add transition-aware dispatch semantics.
	Scope: preserve the current urgent-versus-transition split while keeping version safety.
	TDD:
	- urgent update dispatches immediately
	- transition update can be deferred
	- deferred update still drops stale patch output correctly

## 18. SSR And Hydration Boundaries

- [ ] Add explicit SSR-local-only rules for parallel regions.
	Scope: the first release must not depend on workers during SSR.
	TDD:
	- SSR path renders without worker availability
	- SSR output is deterministic without worker negotiation
	- worker capabilities do not change SSR output

- [ ] Add post-hydration worker attach semantics.
	Scope: worker-backed region mode should begin only after hydration succeeds.
	TDD:
	- initial hydration completes without worker patch input
	- worker attach after hydration succeeds
	- worker patch before hydration completion is ignored or deferred consistently

- [ ] Add hydration-mismatch fallback for a parallel region shell.
	Scope: mismatches must not leave the region half-owned by hydration and half-owned by workers.
	TDD:
	- mismatch enters local remount or fallback path cleanly
	- pending worker patch after mismatch is ignored
	- later reattach uses a fresh epoch

## 19. Event And Interaction Model

- [ ] Define the first-slice event boundary explicitly.
	Scope: the initial worker-backed slice should be display-only unless an event slot model is added later.
	TDD:
	- unsupported interactive event props are rejected by validation
	- display-only props remain accepted
	- validation error names the offending event prop

- [ ] Add placeholder event-slot metadata types for later phases.
	Scope: reserve a stable shape for future event indirection without enabling it yet.
	TDD:
	- metadata can round-trip in tests
	- runtime rejects active event-slot use before implementation
	- unknown event-slot version fails

- [ ] Add explicit validation that refs, portals, and direct DOM interop are not allowed in the first slice.
	Scope: keep the first release narrow and predictable.
	TDD:
	- ref-like prop or feature use fails
	- portal-like feature use fails
	- direct DOM interop markers fail

## 20. Diagnostics And Observability

- [ ] Add stable diagnostic event types for region lifecycle.
	Scope: define mount, update, patch, fallback, restart, and dispose diagnostics.
	TDD:
	- each event type round-trips
	- unknown diagnostic type fails or is ignored consistently
	- required fields are enforced

- [ ] Add timing diagnostics.
	Scope: measure render time, diff time, encode time, transport time, and commit time where possible.
	TDD:
	- timing envelope round-trips
	- negative durations are rejected
	- missing timestamps are rejected where required

- [ ] Add size diagnostics.
	Scope: expose snapshot bytes, IR bytes, patch bytes, and shared-page utilization.
	TDD:
	- size envelope round-trips
	- negative sizes are rejected
	- unsupported size fields default cleanly

- [ ] Add fallback-reason diagnostics.
	Scope: make it obvious why a region left worker-backed mode.
	TDD:
	- transport failure reason round-trips
	- commit failure reason round-trips
	- worker restart reason round-trips

- [ ] Add debug-only trace IDs for one region lifecycle.
	Scope: make it possible to correlate mount, update, patch, and fallback events across threads.
	TDD:
	- trace ID survives mount to patch-ready path
	- missing trace ID behaves consistently when optional
	- duplicate trace IDs do not corrupt bookkeeping

## 21. Positive-Path Tests

- [ ] Add end-to-end mount test for one display-only region over structured-clone transport.
- [ ] Add end-to-end update test for one display-only region over structured-clone transport.
- [ ] Add end-to-end no-op-update test where no patch is emitted.
- [ ] Add end-to-end cancel test where an outdated patch never commits.
- [ ] Add end-to-end dispose test where region state and DOM index are cleaned up.
- [ ] Add end-to-end sticky-affinity test proving repeated updates stay on the same worker shard.
- [ ] Add end-to-end shared-memory snapshot test when shared memory is available.
- [ ] Add end-to-end binary transport test for the compact snapshot path.

## 22. Negative-Path Tests

- [ ] Add malformed control-envelope decode tests.
- [ ] Add malformed snapshot-envelope decode tests.
- [ ] Add malformed binary-header decode tests.
- [ ] Add malformed shared-page-header decode tests.
- [ ] Add unknown renderer-ID tests.
- [ ] Add invalid region-instance-ID tests.
- [ ] Add invalid patch-op tests.
- [ ] Add wrong-epoch patch tests.
- [ ] Add stale-version patch tests.
- [ ] Add duplicate mount for same active region tests.
- [ ] Add worker-restart race tests.
- [ ] Add worker-death-during-patch tests.

## 23. Edge-Case Tests

- [ ] Add zero-child region tests.
- [ ] Add single-text-node region tests.
- [ ] Add empty-text update tests.
- [ ] Add empty-prop-set tests.
- [ ] Add duplicate-key sibling tests.
- [ ] Add rapid mount-dispose-mount churn tests.
- [ ] Add rapid update-cancel-update churn tests.
- [ ] Add dispose-during-fallback tests.
- [ ] Add fallback-then-remount tests.
- [ ] Add multiple-regions-sharing-one-renderer-ID tests.
- [ ] Add one-worker-many-regions pressure tests.
- [ ] Add many-workers-few-regions skew tests.

## 24. Fuzz Tests

- [ ] Add fuzz coverage for control-plane envelope decoding.
- [ ] Add fuzz coverage for snapshot-envelope decoding.
- [ ] Add fuzz coverage for binary snapshot decoding.
- [ ] Add fuzz coverage for shared-page header parsing.
- [ ] Add fuzz coverage for render IR decoding.
- [ ] Add fuzz coverage for string-table decoding.
- [ ] Add fuzz coverage for prop-record decoding.
- [ ] Add fuzz coverage for patch IR decoding.
- [ ] Add fuzz coverage for DOM-index patch-application prevalidation.

## 25. Benchmarks

- [ ] Add a microbenchmark for source snapshot capture.
- [ ] Add a microbenchmark for structured-clone snapshot encoding.
- [ ] Add a microbenchmark for binary snapshot encoding.
- [ ] Add a microbenchmark for shared-page publication.
- [ ] Add a microbenchmark for worker-side IR build.
- [ ] Add a microbenchmark for worker-side diff.
- [ ] Add a microbenchmark for patch decode.
- [ ] Add a microbenchmark for patch commit.
- [ ] Add an end-to-end comparison benchmark for local display-region rendering versus worker-backed rendering.
- [ ] Add a pressure benchmark for many hot regions sharing a bounded worker set.

## 26. Examples And Documentation

- [ ] Add a minimal example that shows one display-only parallel region.
- [ ] Add a stress example that shows many parallel regions across multiple workers.
- [ ] Add a diagnostics example that shows fallback and worker restart behavior.
- [ ] Add authoring docs that explain which region shapes are allowed in the first slice.
- [ ] Add troubleshooting docs for worker-backed region fallback, protocol mismatch, and shared-memory deployment requirements.

## Exit Criteria For The First Shippable Slice

The first release candidate should not be considered ready until all of these are true:

- one display-only parallel region works end-to-end on the main supported browser path
- stale worker output cannot commit after owner rerender, cancel, restart, or fallback
- worker death cannot silently corrupt region ownership
- structured-clone fallback preserves semantics when shared memory is unavailable
- deterministic, negative, edge-case, and fuzz tests exist for the public contracts
- focused benchmarks show where the runtime helps and where it does not
- diagnostics make fallback and transport failures actionable

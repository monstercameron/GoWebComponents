# Multithreaded Runtime TODO

Last updated: 2026-03-27

This backlog tracks the proposed worker-backed multithreaded runtime described in [MULTITHREADED_RUNTIME.md](MULTITHREADED_RUNTIME.md).

It is implementation-facing, TDD-first, and intentionally narrower than the main framework backlog in [TODO.md](TODO.md).

## At A Glance

- This file is the execution backlog for the future parallel rendering runtime.
- Work should proceed one unchecked item at a time.
- Prefer the smallest focused failing test before each implementation step.
- Keep the scope narrow enough that one validation run can prove the item.
- Use this file for runtime2 planning and execution, not as a changelog.
- This file is agent-owned from top to bottom: all work is grouped under exactly four agents.

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

## Four-Agent Operating Model

- Agent 1 owns public `ui` surface, authoring, examples, and adoption docs.
- Agent 2 owns capability contracts, protocol, scheduler, and transport tiers.
- Agent 3 owns render IR, diff, patch generation, DOM commit, and end-to-end orchestration.
- Agent 4 owns host lifecycle, recovery, diagnostics, SSR or hydration, and cross-cutting performance validation.
- Each agent still takes one unchecked item at a time inside its own lane.
- When one item is completed, update this file immediately.
- Prefer disjoint write areas and avoid cross-lane edits unless the handoff section says the dependency is ready.

## Agent 1. Public Surface, Authoring, And Adoption

Goal: expose the worker-backed runtime through a real `ui` API and make the feature understandable and usable once the core runtime path is ready.

Primary write area:

- `ui`
- examples
- docs
- small registry bridge points in `internal/runtime2`

### Completed Foundations

- [x] Add a stable renderer-ID type for parallel regions.
- [x] Add a stable region-instance ID type.
- [x] Add a renderer registry for worker-renderable regions.
- [x] Add registry reset support for tests.
- [x] Add renderer metadata support in the registry.
- [x] Add a runtime2 `ParallelRegionSpec` shape for serializable region inputs.
- [x] Add pre-dispatch spec validation.
- [x] Add prop-serializability checks.
- [x] Add explicit declared-source validation.
- [x] Add source-order normalization.
- [x] Define the first-slice allowed host-tag set explicitly.
- [x] Define the first-slice allowed prop-family set explicitly.
- [x] Reject unsupported host tags for worker-renderable regions.
- [x] Reject unsupported prop families for worker-renderable regions.
- [x] Reject refs in worker-renderable region specs or metadata.
- [x] Reject portal-like output in worker-renderable region render results.
- [x] Reject direct DOM interop markers in runtime2 inputs.
- [x] Reject direct DOM interop markers in runtime2 outputs.
- [x] Reject event-closure props in worker-renderable region inputs.
- [x] Add placeholder event-slot metadata types.
- [x] Add validation for event-slot metadata shape.
- [x] Add a separate placeholder encoding path for event-slot metadata.
- [x] Keep event-slot metadata optional and non-operative in slice one.

### Open Implementation

- [x] Add a public `ui.ParallelRegionSpec[...]` shape that maps cleanly into `runtime2.ParallelRegionSpec`.
- [x] Add a public `ui.RegisterParallelRegion(...)` API that bridges into the runtime2 renderer registry.
- [x] Add a public `ui.ParallelRegion(...)` API that renders one local-first region shell.
- [x] Add public source-binding helpers that preserve declared source ordering into runtime2.
- [x] Add local-first initial render behavior at the public `ui` layer without waiting on worker output.
- [ ] Add public owner-removal disposal wiring from `ui` into runtime2 cleanup.
- [ ] Add browser-only gating so unsupported targets fail predictably or stay local-only.
- [ ] Add native fallback behavior so non-browser builds remain deterministic.

### Open Validation And Adoption

- [ ] Add duplicate-registration tests at the public `ui` layer.
- [ ] Add missing-renderer tests at the public `ui` layer.
- [ ] Add invalid-props tests at the public `ui` layer.
- [ ] Add invalid-source-ID tests at the public `ui` layer.
- [ ] Add invalid-region-instance-ID tests at the public `ui` layer.
- [ ] Add browser-native parity tests that prove native builds stay deterministic when worker-backed rendering is unavailable.
- [ ] Add a minimal example app with one display-only parallel region.
- [ ] Add a stress example app with many parallel regions across multiple workers.
- [ ] Add a diagnostics example that demonstrates fallback, downgrade, and worker restart behavior.
- [ ] Add authoring docs for first-slice allowed region shapes.
- [ ] Add troubleshooting docs for fallback, protocol mismatch, binary transport, and shared-memory deployment requirements.

### Recommended First Pick

- [x] Add a public `ui.ParallelRegionSpec[...]` shape that maps cleanly into `runtime2.ParallelRegionSpec`.
- [x] Add a public `ui.RegisterParallelRegion(...)` API that bridges into the runtime2 renderer registry.
- [x] Add a public `ui.ParallelRegion(...)` API that renders one local-first region shell.
- [x] Add local-first initial render behavior at the public `ui` layer without waiting on worker output.

## Agent 2. Capability Contracts, Scheduler, And Transport

Goal: own capability negotiation, protocol contracts, worker affinity, and all snapshot transport tiers.

Primary write area:

- `internal/runtime2` capability, protocol, scheduler, and transport files

### Completed Foundations

- [x] Create a dedicated package boundary for the multithreaded runtime.
- [x] Add a package-level capability report for the multithreaded runtime.
- [x] Add protocol version constants for the region runtime.
- [x] Add capability-negotiation fixtures for tests.
- [x] Add a shared control-plane envelope format.
- [x] Add a `ready` message contract.
- [x] Add a `capabilities` message contract.
- [x] Add a `mount` message contract.
- [x] Add an `update` message contract.
- [x] Add a `cancel` message contract.
- [x] Add a `dispose` message contract.
- [x] Add a `patch-ready` message contract.
- [x] Add a `diagnostic` message contract.
- [x] Add a `restart` message contract.
- [x] Add a worker-shard identity model.
- [x] Add deterministic region-to-shard assignment.
- [x] Add explicit scheduler mount handling.
- [x] Add explicit scheduler update handling.
- [x] Add explicit scheduler cancel handling.
- [x] Add explicit scheduler dispose handling.
- [x] Add bounded queueing rules at the scheduler layer.
- [x] Add worker health tracking.
- [x] Add worker replacement flow.
- [x] Add structured-clone encoding for snapshot envelopes.
- [x] Add structured-clone encoding for mount envelopes.
- [x] Add structured-clone encoding for update envelopes.
- [x] Add structured-clone decoding guards for unexpected field types.
- [x] Add the complete binary snapshot transport slice.
- [x] Add the complete shared-memory snapshot transport slice.
- [x] Add malformed binary-header negative tests.
- [x] Add malformed shared-page-header negative tests.
- [x] Add end-to-end binary transport coverage.
- [x] Add end-to-end shared-memory transport coverage.
- [x] Add end-to-end shared-memory downgrade coverage.
- [x] Add fuzz coverage for binary snapshot decoding.
- [x] Add fuzz coverage for shared-page parsing.
- [x] Add a microbenchmark for binary snapshot encode and decode.
- [x] Add a microbenchmark for shared-page publish and read.

### Open Implementation

- [x] Add helper builders for `ready`, `capabilities`, `mount`, `update`, `cancel`, `dispose`, and `restart` control envelopes.
  Validation: `go test ./internal/runtime2 -run "TestBuildControl(Ready|Capabilities|Mount|Update|Cancel|Dispose|Restart)EnvelopeBuildsValidatedEnvelope"`
- [x] Add helper builders for `patch-ready` and `diagnostic` control envelopes.
  Validation: `go test ./internal/runtime2 -run "TestBuildControl.*EnvelopeBuildsValidatedEnvelope"`
- [x] Add a worker-side control dispatcher that routes mount, update, cancel, dispose, and restart envelopes into `WorkerRegionRuntime`.
  Validation: `go test ./internal/runtime2 -run "TestHandleWorkerControlEnvelope(Dispatches(Mount|Update|Cancel|Dispose|Restart)|RejectsUnsupportedKind)"`
- [x] Add a host-side control dispatcher that routes `patch-ready`, `diagnostic`, and `restart` envelopes into the host runtime2 path.
  Validation: `go test ./internal/runtime2 -run "TestHandleHostControlEnvelope(Dispatches(PatchReady|Diagnostic|Restart)|RejectsUnsupportedKind)"`
- [ ] Add snapshot-transport selection hooks into the host update-dispatch path.
- [ ] Add patch-transport selection hooks into the worker patch-ready path.

### Open Validation And Benchmarks

- [ ] Add malformed control-envelope decode tests.
- [ ] Add malformed snapshot-envelope decode tests.
- [ ] Add fuzz coverage for control-plane envelope decoding.
- [ ] Add fuzz coverage for snapshot-envelope decoding.
- [ ] Add a microbenchmark for structured-clone snapshot encoding.

### Recommended First Pick

- [ ] Add helper builders for `ready`, `capabilities`, `mount`, `update`, `cancel`, `dispose`, and `restart` control envelopes.

## Agent 3. Render IR, Patch Pipeline, And End-To-End Orchestration

Goal: replace placeholder worker output with canonical IR and a real patch stream, then wire that into commit orchestration.

Primary write area:

- `internal/runtime2` render files
- `internal/runtime2` patch files
- `internal/runtime2/worker_region_runtime.go`
- `internal/runtime2/dom_commit.go`
- runtime2 end-to-end tests

### Completed Foundations

- [x] Add a render-node kind enum for the first slice.
- [x] Add a render-node record layout.
- [x] Add stable per-region node identity.
- [x] Add flat child ordering rules.
- [x] Add support for keyed child metadata.
- [x] Add a string-table format for render IR.
- [x] Add canonical string-table ordering.
- [x] Add a prop-record format for the first slice.
- [x] Add prop-key canonical ordering.
- [x] Add style-value normalization rules for the supported first slice.
- [x] Add patch op codes for the first slice.
- [x] Add a patch-stream header format.
- [x] Add insert-op payload validation.
- [x] Add remove-op payload validation.
- [x] Add set-text-op payload validation.
- [x] Add set-attr-op payload validation.
- [x] Add remove-attr-op payload validation.
- [x] Add keyed-move-op payload validation.
- [x] Add patch-order validation rules.
- [x] Add patch idempotency metadata.
- [x] Add worker-side mount handling.
- [x] Add worker-side update handling.
- [x] Add worker-side cancel handling.
- [x] Add worker-side dispose handling.
- [x] Add worker-side restart handling.
- [x] Add no-op patch detection.
- [x] Add a region-local DOM index.
- [x] Add DOM-index cleanup on region dispose.
- [x] Add text-node commit support.
- [x] Add attr-commit support.
- [x] Add node-insert commit support.
- [x] Add node-remove commit support.
- [x] Add keyed-move commit support for the first slice.
- [x] Add patch-commit transaction boundaries.

### Open Implementation

- [ ] Add conversion from validated display-only render output into canonical render-node raw records.
- [ ] Add canonical string-table extraction from one render output tree.
- [ ] Add canonical prop-record extraction from one render output tree.
- [ ] Add a stable per-render node-ID allocation strategy for one region render.
- [ ] Add root-node conventions for empty, single-text, and host-element region outputs.
- [ ] Add worker-region state that stores parsed canonical IR instead of raw `any`.
- [ ] Add one diff entrypoint from previous canonical IR to next canonical IR.
- [ ] Add insert patch generation from newly introduced nodes.
- [ ] Add remove patch generation from missing nodes.
- [ ] Add set-text patch generation from text changes.
- [ ] Add set-attr patch generation from prop changes.
- [ ] Add remove-attr patch generation from prop removal.
- [ ] Add keyed-move patch generation from keyed sibling reorders.
- [ ] Add canonical patch ordering output from the diff engine.
- [ ] Replace `reflect.DeepEqual(...)` no-op detection with canonical-IR equality.
- [ ] Replace the `{previous,next}` patch placeholder with typed patch-stream output.
- [ ] Add one patch payload parser entrypoint that validates patch-stream header, op ordering, and idempotency before commit.
- [ ] Add one host commit entrypoint that feeds parsed patch transactions into `CommitRegionPatchTransaction(...)`.
- [ ] Add stale patch-version suppression in the orchestration layer.
- [ ] Add region-ID mismatch rejection in the orchestration layer.
- [ ] Add epoch mismatch rejection in the orchestration layer.

### Open Positive, Negative, And Edge Tests

- [ ] Add end-to-end mount test for one display-only region over structured-clone transport.
- [ ] Add end-to-end update test for one display-only region over structured-clone transport.
- [ ] Add end-to-end no-op-update test where no patch is emitted.
- [ ] Add end-to-end cancel test where an outdated patch never commits.
- [ ] Add end-to-end dispose test where region state and DOM index are cleaned up.
- [ ] Add end-to-end sticky-affinity test proving repeated updates stay on the same worker shard.
- [ ] Add unknown renderer-ID tests through the real runtime path.
- [ ] Add invalid patch-op tests.
- [ ] Add wrong-epoch patch tests.
- [ ] Add stale-version patch tests.
- [ ] Add duplicate mount for the same active region tests.
- [ ] Add zero-child region tests.
- [ ] Add single-text-node region tests.
- [ ] Add empty-text update tests.
- [ ] Add empty-prop-set tests.
- [ ] Add duplicate-key sibling tests.

### Open Fuzzing And Benchmarks

- [ ] Add fuzz coverage for render IR decoding.
- [ ] Add fuzz coverage for string-table decoding.
- [ ] Add fuzz coverage for prop-record decoding.
- [ ] Add fuzz coverage for patch IR decoding.
- [ ] Add fuzz coverage for DOM-index patch-application prevalidation.
- [ ] Add a microbenchmark for worker-side IR build.
- [ ] Add a microbenchmark for worker-side diff.
- [ ] Add a microbenchmark for patch decode.
- [ ] Add a microbenchmark for patch commit.
- [ ] Add an end-to-end comparison benchmark for local display-region rendering versus worker-backed rendering.

### Recommended First Pick

- [ ] Add conversion from validated display-only render output into canonical render-node raw records.

## Agent 4. Host Lifecycle, Recovery, Diagnostics, SSR Or Hydration, And Pressure Validation

Goal: own host-side state, fallback and remount correctness, emitted diagnostics, shell attach rules, and pressure or race validation.

Primary write area:

- `internal/runtime2/host_region_adapter.go`
- `internal/runtime2/recovery_coordinator.go`
- `internal/runtime2/coordinator.go`
- `internal/runtime2` diagnostics and shell files
- pressure, race, and lifecycle tests

### Completed Foundations

- [x] Add a source-snapshot envelope type.
- [x] Add monotonic input-version rules.
- [x] Add source-value snapshotting from declared IDs.
- [x] Add snapshot consistency rules across multiple sources.
- [x] Add snapshot hashing or stable fingerprint support.
- [x] Add a coordinator entry type for live regions.
- [x] Add coordinator state transitions for mount, update, cancel, dispose, fallback, and restart.
- [x] Add `lastDispatchedVersion` tracking.
- [x] Add `lastCommittedVersion` tracking.
- [x] Add fallback-mode tracking.
- [x] Add local-render fallback entry for one region.
- [x] Add fallback triggers for transport decode failure.
- [x] Add fallback triggers for invalid DOM commit state.
- [x] Add worker-death recovery policy.
- [x] Add explicit stale-result dropping after fallback.
- [x] Add source-change detection for declared region inputs.
- [x] Add owner-rerender precedence over worker output.
- [x] Add stable diagnostic event kinds for mount, update, cancel, dispose, restart, patch-ready, fallback, and repair.
- [x] Add diagnostic timing schema validation.
- [x] Add one host-side region adapter that owns coordinator, scheduler, recovery, and DOM-index handles for a live runtime2 region.
- [x] Add host-side mount entry for one validated parallel region.
- [x] Add host-side update entry for one mounted parallel region.
- [x] Add host-side dispose entry for one mounted parallel region.
- [x] Add local-first shell ownership on initial mount before worker output commits.
- [x] Add declared-source lookup from the shipped runtime into runtime2 snapshot production.
- [x] Add declared-source missing-value failure handling.
- [x] Add props-plus-sources snapshot capture for one region update.
- [x] Add stable snapshot fingerprint reuse for no-change detection.
- [x] Add no-change short-circuit before scheduling worker updates.
- [x] Add input-version advancement only when a real update is dispatched.
- [x] Add urgent dispatch classification for runtime2 updates.
- [x] Add deferred dispatch classification for runtime2 updates.
- [x] Add deferred-update supersession rules.
- [x] Add deferred-update cancellation when a newer urgent update arrives.
- [x] Add cancellation cleanup for queued region jobs after owner-side invalidation.

### Open Implementation

- [ ] Add region unmount semantics when the owner stops rendering the region.
- [ ] Add transition-aware dispatch semantics.
- [ ] Add owner removal handling that disposes the region and suppresses late worker output.
- [ ] Add structural remount detection when renderer identity or shell ownership changes.
- [ ] Add remount epoch advancement on structural remount.
- [ ] Add `HandleHostRegionOwnerRemove(...)`.
- [ ] Add `HandleHostRegionStructuralRemount(...)`.
- [ ] Add `HandleHostRegionFallbackMirror(...)`.
- [ ] Add `HandleHostRegionPatchReady(...)`.
- [ ] Add `HandleHostRegionWorkerDeath(...)`.
- [ ] Add `HandleHostRegionRepairRemount(...)`.
- [ ] Add stable getter helpers for fallback-pending, fallback-active, repair-pending, repair epoch, repair version floor, and latest valid version when tests need explicit visibility.
- [ ] Route structured-clone decode failure through the existing recovery coordinator.
- [ ] Route binary decode failure through the existing recovery coordinator.
- [ ] Route shared-page decode failure through the existing recovery coordinator.
- [ ] Route DOM patch-transaction failure through the existing recovery coordinator.
- [ ] Mirror recovery fallback state into the main coordinator state machine.
- [ ] Mirror recovery fallback state into scheduler fallback ownership.
- [ ] Block worker commit attempts once fallback ownership is active in the host pipeline.
- [ ] Wire dead-worker detection from scheduler or transport failure into `HandleWorkerDeath(...)`.
- [ ] Allocate a fresh remount epoch after successful worker reassignment.
- [ ] Propagate the remount epoch back into coordinator state.
- [ ] Reissue a clean mount after worker reassignment before allowing updates.
- [ ] Keep the latest valid input version during repair-driven remount.
- [ ] Reject patch-ready results produced before repair completes.
- [ ] Reject patch-ready results produced before fallback ownership begins.
- [ ] Keep fallback ownership active until one fresh remount succeeds.
- [ ] Clear fallback ownership only after a healthy remount handshake completes.
- [x] Add structured timing diagnostics for queue, render, diff, encode, transport, and commit stages.
  Validation: `go test internal/runtime2/protocol.go internal/runtime2/registry.go internal/runtime2/spec.go internal/runtime2/capabilities.go internal/runtime2/snapshot.go internal/runtime2/recovery_coordinator.go internal/runtime2/scheduler_shard_identity.go internal/runtime2/diagnostic_event_kind.go internal/runtime2/diagnostic_timing.go internal/runtime2/diagnostic_size.go internal/runtime2/diagnostic_fallback_reason.go internal/runtime2/diagnostic_trace.go internal/runtime2/diagnostic_shard.go internal/runtime2/diagnostic_downgrade_reason.go internal/runtime2/control.go internal/runtime2/control_diagnostic_timing_internal_test.go internal/runtime2/control_diagnostic_size_internal_test.go internal/runtime2/control_diagnostic_fallback_internal_test.go internal/runtime2/control_diagnostic_trace_internal_test.go internal/runtime2/control_diagnostic_shard_internal_test.go internal/runtime2/control_diagnostic_transport_internal_test.go internal/runtime2/control_diagnostic_downgrade_internal_test.go`
- [x] Add structured size diagnostics for snapshot, IR, patch, and shared-page usage.
  Validation: `go test internal/runtime2/protocol.go internal/runtime2/registry.go internal/runtime2/spec.go internal/runtime2/capabilities.go internal/runtime2/snapshot.go internal/runtime2/recovery_coordinator.go internal/runtime2/scheduler_shard_identity.go internal/runtime2/diagnostic_event_kind.go internal/runtime2/diagnostic_timing.go internal/runtime2/diagnostic_size.go internal/runtime2/diagnostic_fallback_reason.go internal/runtime2/diagnostic_trace.go internal/runtime2/diagnostic_shard.go internal/runtime2/diagnostic_downgrade_reason.go internal/runtime2/control.go internal/runtime2/control_diagnostic_timing_internal_test.go internal/runtime2/control_diagnostic_size_internal_test.go internal/runtime2/control_diagnostic_fallback_internal_test.go internal/runtime2/control_diagnostic_trace_internal_test.go internal/runtime2/control_diagnostic_shard_internal_test.go internal/runtime2/control_diagnostic_transport_internal_test.go internal/runtime2/control_diagnostic_downgrade_internal_test.go`
- [x] Add structured fallback-reason diagnostics aligned with transport, DOM, and worker-death recovery reasons.
  Validation: `go test internal/runtime2/protocol.go internal/runtime2/registry.go internal/runtime2/spec.go internal/runtime2/capabilities.go internal/runtime2/snapshot.go internal/runtime2/recovery_coordinator.go internal/runtime2/scheduler_shard_identity.go internal/runtime2/diagnostic_event_kind.go internal/runtime2/diagnostic_timing.go internal/runtime2/diagnostic_size.go internal/runtime2/diagnostic_fallback_reason.go internal/runtime2/diagnostic_trace.go internal/runtime2/diagnostic_shard.go internal/runtime2/diagnostic_downgrade_reason.go internal/runtime2/control.go internal/runtime2/control_diagnostic_timing_internal_test.go internal/runtime2/control_diagnostic_size_internal_test.go internal/runtime2/control_diagnostic_fallback_internal_test.go internal/runtime2/control_diagnostic_trace_internal_test.go internal/runtime2/control_diagnostic_shard_internal_test.go internal/runtime2/control_diagnostic_transport_internal_test.go internal/runtime2/control_diagnostic_downgrade_internal_test.go`
- [x] Add debug-only trace IDs spanning scheduler, worker, and commit attempts.
  Validation: `go test internal/runtime2/protocol.go internal/runtime2/registry.go internal/runtime2/spec.go internal/runtime2/capabilities.go internal/runtime2/snapshot.go internal/runtime2/recovery_coordinator.go internal/runtime2/scheduler_shard_identity.go internal/runtime2/diagnostic_event_kind.go internal/runtime2/diagnostic_timing.go internal/runtime2/diagnostic_size.go internal/runtime2/diagnostic_fallback_reason.go internal/runtime2/diagnostic_trace.go internal/runtime2/diagnostic_shard.go internal/runtime2/diagnostic_downgrade_reason.go internal/runtime2/control.go internal/runtime2/control_diagnostic_timing_internal_test.go internal/runtime2/control_diagnostic_size_internal_test.go internal/runtime2/control_diagnostic_fallback_internal_test.go internal/runtime2/control_diagnostic_trace_internal_test.go internal/runtime2/control_diagnostic_shard_internal_test.go internal/runtime2/control_diagnostic_transport_internal_test.go internal/runtime2/control_diagnostic_downgrade_internal_test.go`
- [x] Add shard ID reporting in diagnostics.
  Validation: `go test internal/runtime2/protocol.go internal/runtime2/registry.go internal/runtime2/spec.go internal/runtime2/capabilities.go internal/runtime2/snapshot.go internal/runtime2/recovery_coordinator.go internal/runtime2/scheduler_shard_identity.go internal/runtime2/diagnostic_event_kind.go internal/runtime2/diagnostic_timing.go internal/runtime2/diagnostic_size.go internal/runtime2/diagnostic_fallback_reason.go internal/runtime2/diagnostic_trace.go internal/runtime2/diagnostic_shard.go internal/runtime2/diagnostic_downgrade_reason.go internal/runtime2/control.go internal/runtime2/control_diagnostic_timing_internal_test.go internal/runtime2/control_diagnostic_size_internal_test.go internal/runtime2/control_diagnostic_fallback_internal_test.go internal/runtime2/control_diagnostic_trace_internal_test.go internal/runtime2/control_diagnostic_shard_internal_test.go internal/runtime2/control_diagnostic_transport_internal_test.go internal/runtime2/control_diagnostic_downgrade_internal_test.go`
- [x] Add transport-tier reporting in diagnostics.
  Validation: `go test internal/runtime2/protocol.go internal/runtime2/registry.go internal/runtime2/spec.go internal/runtime2/capabilities.go internal/runtime2/snapshot.go internal/runtime2/recovery_coordinator.go internal/runtime2/scheduler_shard_identity.go internal/runtime2/diagnostic_event_kind.go internal/runtime2/diagnostic_timing.go internal/runtime2/diagnostic_size.go internal/runtime2/diagnostic_fallback_reason.go internal/runtime2/diagnostic_trace.go internal/runtime2/diagnostic_shard.go internal/runtime2/diagnostic_downgrade_reason.go internal/runtime2/control.go internal/runtime2/control_diagnostic_timing_internal_test.go internal/runtime2/control_diagnostic_size_internal_test.go internal/runtime2/control_diagnostic_fallback_internal_test.go internal/runtime2/control_diagnostic_trace_internal_test.go internal/runtime2/control_diagnostic_shard_internal_test.go internal/runtime2/control_diagnostic_transport_internal_test.go internal/runtime2/control_diagnostic_downgrade_internal_test.go`
- [x] Add downgrade-reason reporting in diagnostics for shared-memory and binary fallback paths.
  Validation: `go test internal/runtime2/protocol.go internal/runtime2/registry.go internal/runtime2/spec.go internal/runtime2/capabilities.go internal/runtime2/snapshot.go internal/runtime2/recovery_coordinator.go internal/runtime2/scheduler_shard_identity.go internal/runtime2/diagnostic_event_kind.go internal/runtime2/diagnostic_timing.go internal/runtime2/diagnostic_size.go internal/runtime2/diagnostic_fallback_reason.go internal/runtime2/diagnostic_trace.go internal/runtime2/diagnostic_shard.go internal/runtime2/diagnostic_downgrade_reason.go internal/runtime2/control.go internal/runtime2/control_diagnostic_timing_internal_test.go internal/runtime2/control_diagnostic_size_internal_test.go internal/runtime2/control_diagnostic_fallback_internal_test.go internal/runtime2/control_diagnostic_trace_internal_test.go internal/runtime2/control_diagnostic_shard_internal_test.go internal/runtime2/control_diagnostic_transport_internal_test.go internal/runtime2/control_diagnostic_downgrade_internal_test.go`
- [x] Add a runtime2 SSR shell marker format.
  Validation: `go test internal/runtime2/registry.go internal/runtime2/ssr_shell_marker.go internal/runtime2/ssr_shell_marker_test.go`
- [x] Add shell marker encoding during local SSR.
  Validation: `go test internal/runtime2/registry.go internal/runtime2/ssr_shell_marker.go internal/runtime2/ssr_shell_marker_test.go`
- [x] Add shell marker parsing on the client.
  Validation: `go test internal/runtime2/registry.go internal/runtime2/ssr_shell_marker.go internal/runtime2/ssr_shell_marker_test.go`
- [x] Keep SSR local-only for the first runtime2 slice.
  Validation: `go test internal/runtime2/ssr_local_policy.go internal/runtime2/ssr_local_policy_test.go`
- [x] Add post-hydration worker attach semantics.
  Validation: `go test ./internal/runtime2 -run "HostRegionPostHydrationAttach|HostRegionHydrationComplete" -count=1`
- [x] Block worker attach before hydration completes.
  Validation: `go test ./internal/runtime2 -run "HostRegionPostHydrationAttach|HostRegionHydrationComplete" -count=1`
- [x] Register hydrated shell anchors into the region DOM index before worker commit begins.
  Validation: `go test ./internal/runtime2 -run "HostRegionPostHydrationAttach|HostRegionHydrationComplete|HydratedShellAnchor" -count=1`
- [ ] Add shell-identity mismatch detection for region ID mismatches.
- [ ] Add shell-identity mismatch detection for renderer ID mismatches.
- [ ] Add shell-missing-anchor detection.
- [ ] Add local remount fallback on shell mismatch.
- [ ] Drop pending worker output after hydration mismatch fallback.
- [ ] Require a fresh epoch before later reattach after mismatch recovery.

### Open Validation And Performance

- [ ] Add worker-restart race tests.
- [ ] Add worker-death-during-patch tests.
- [ ] Add rapid mount-dispose-mount churn tests.
- [ ] Add rapid update-cancel-update churn tests.
- [ ] Add dispose-during-fallback tests.
- [ ] Add fallback-then-remount tests.
- [ ] Add multiple-regions-sharing-one-renderer-ID tests.
- [ ] Add one-worker-many-regions pressure tests.
- [ ] Add many-workers-few-regions skew tests.
- [ ] Add diagnostics payload tests for timing, size, fallback-reason, and trace fields.
- [ ] Add SSR shell-marker encode or decode tests.
- [ ] Add hydration attach tests.
- [ ] Add hydration mismatch fallback tests.
- [ ] Add a microbenchmark for source snapshot capture.
- [ ] Add a pressure benchmark for many hot regions sharing a bounded worker set.

### Recommended First Pick

- [ ] Add region unmount semantics when the owner stops rendering the region.

## Cross-Lane Handoffs

- After Agent 2 finishes control-envelope builders and dispatch hooks, Agent 3 can wire real mount and update orchestration through the control plane.
- After Agent 3 replaces placeholder patches with typed patch streams, Agent 2 can finish patch-transport hooks cleanly.
- After Agent 4 lands patch-ready gating, remount, and fallback mirrors, Agent 3 can tighten stale-output rejection and full end-to-end commit coverage.
- After Agent 1 lands the public `ui` surface, Agent 3 and Agent 4 can validate the real user-facing path instead of only runtime2 internals.
- After Agent 4 lands SSR shell markers and hydration attach, Agent 1 can finalize adoption docs and examples without hand-waving.

## Recommended First Pick Per Agent

- Agent 1: `Add a public ui.ParallelRegionSpec[...] shape that maps cleanly into runtime2.ParallelRegionSpec.`
- Agent 2: `Add helper builders for ready, capabilities, mount, update, cancel, dispose, and restart control envelopes.`
- Agent 3: `Add conversion from validated display-only render output into canonical render-node raw records.`
- Agent 4: `Add region unmount semantics when the owner stops rendering the region.`

## Exit Criteria For The First Shippable Slice

The first release candidate should not be considered ready until all of these are true:

- one display-only parallel region works end-to-end on the main supported browser path
- the public `ui` surface exists and is documented
- stale worker output cannot commit
- worker death cannot silently corrupt region ownership
- fallback ownership is explicit and observable
- non-shared-memory environments still work
- shared-memory environments preserve semantics and only change transport cost
- structured-clone, binary, and shared-memory paths all have focused validation
- deterministic, negative, edge-case, fuzz, and benchmark coverage exist for the first slice
- examples and troubleshooting docs are aligned with the shipped behavior

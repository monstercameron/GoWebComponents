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
- [x] Add public owner-removal disposal wiring from `ui` into runtime2 cleanup.
- [x] Add browser-only gating so unsupported targets fail predictably or stay local-only.
- [x] Add native fallback behavior so non-browser builds remain deterministic.
- [x] Add a public per-region input-version counter so browser-side `ui.ParallelRegion(...)` rerenders can emit monotonic runtime2 update versions after the initial mount.
  Validation: `go test ./ui -run "TestBuildParallelRegionNextInputVersionIncrementsAndResets$"` and `GOOS=js GOARCH=wasm go test -exec "C:\Users\Cam\Desktop\GoWebComponents\tools\go_js_wasm_exec.bat" ./ui -run "TestParallelRegionRenderIntoAdvancesInputVersionAcrossRerenders$"`
- [x] Add browser-side rerender wiring that forwards public region prop or source changes into `HandleHostRegionUpdateDispatchWithTransportPriority(...)` instead of leaving the cached adapter mount-only.
  Validation: `GOOS=js GOARCH=wasm go test -exec "C:\Users\Cam\Desktop\GoWebComponents\tools\go_js_wasm_exec.bat" ./ui -run TestParallelRegionRenderIntoPropChangesDispatchRuntime2Update$` and `GOOS=js GOARCH=wasm go test -exec "C:\Users\Cam\Desktop\GoWebComponents\tools\go_js_wasm_exec.bat" ./ui -run TestParallelRegionRenderIntoDeclaredSourcesSupportRuntime2UpdateDispatch$`
- [x] Add public declared-source subscription wiring so source-only reactive changes can rerender `ui.ParallelRegion(...)` and dispatch runtime2 updates even when props and parent render output stay unchanged.
  Validation: `GOOS=js GOARCH=wasm go test -exec "C:\Users\Cam\Desktop\GoWebComponents\tools\go_js_wasm_exec.bat" ./ui -run TestParallelRegionDeclaredSourceOnlyChangesDispatchRuntime2Update$`
- [x] Add public transition-aware dispatch wiring that forwards owner-side `StartTransition(...)` or `UseTransition()` state into runtime2 deferred dispatch classification.
  Validation: `GOOS=js GOARCH=wasm go test -exec "C:\Users\Cam\Desktop\GoWebComponents\tools\go_js_wasm_exec.bat" ./ui -run TestParallelRegionTransitionWrappedRerendersUseDeferredDispatch$`
- [x] Add public structural-remount wiring that compares the last mounted renderer ID and shell-ownership mode before calling `HandleHostRegionStructuralRemount(...)`.
  Validation: `GOOS=js GOARCH=wasm go test -exec "C:\Users\Cam\Desktop\GoWebComponents\tools\go_js_wasm_exec.bat" ./ui -run TestParallelRegionRendererIdentityChangesTriggerStructuralRemount$`
- [x] Add public hydration bridge logic that discovers runtime2 shell markers during `ui.Hydrate(...)` or `ui.HydrateInto(...)` and maps them back to cached parallel-region adapters.
  Validation: `GOOS=js GOARCH=wasm go test -exec "C:\Users\Cam\Desktop\GoWebComponents\tools\go_js_wasm_exec.bat" ./ui -run "TestHydrateIntoMarksParallelRegionAdapterHydrationComplete$"`
- [x] Add public hydrated-shell anchor registration that calls `HandleHostRegionRegisterHydratedShellAnchor(...)` with the resumed shell node before worker attach is attempted.
  Validation: `GOOS=js GOARCH=wasm go test -exec "C:\Users\Cam\Desktop\GoWebComponents\tools\go_js_wasm_exec.bat" ./ui -run TestHydrateIntoMarksParallelRegionAdapterHydrationComplete$`
- [x] Add a runtime-to-`ui` helper path to extract durable hydrated shell node metadata (at minimum tag and a non-zero node identity) so public anchor registration does not rely on placeholder values.
  Validation: `GOOS=js GOARCH=wasm go test -exec "C:\Users\Cam\Desktop\GoWebComponents\tools\go_js_wasm_exec.bat" ./ui -run TestHandleParallelRegionHydrationNodesRejectsInvalidShellAnchor$`
- [x] Add shell-identity mismatch handling in public hydration bridging so marker-region-id or marker-renderer-id mismatches cannot attach a wrong worker-bound runtime path.
  Validation: `GOOS=js GOARCH=wasm go test -exec "C:\Users\Cam\Desktop\GoWebComponents\tools\go_js_wasm_exec.bat" ./ui -run TestHandleParallelRegionHydrationNodesRejectsShellIdentityMismatch$`
- [x] Add public post-hydration attach wiring that calls `HandleHostRegionHydrationComplete()` and then `HandleHostRegionPostHydrationAttach()` once the owning shell finishes hydration.
  Validation: `GOOS=js GOARCH=wasm go test -exec "C:\Users\Cam\Desktop\GoWebComponents\tools\go_js_wasm_exec.bat" ./ui -run TestHydrateMarksParallelRegionAdapterAnchorAndAttachBySelector$`
- [x] Add public hydration-anchor registration validation so anchor metadata (node handle/tag) captured from resumed shell markers is consistent before calling `HandleHostRegionRegisterHydratedShellAnchor(...)`.
  Validation: `GOOS=js GOARCH=wasm go test -exec "C:\Users\Cam\Desktop\GoWebComponents\tools\go_js_wasm_exec.bat" ./ui -run TestHandleParallelRegionHydrationNodesRejectsInvalidShellAnchor$`
- [ ] Add a read-only public region-status helper for examples and tooling that exposes local-or-worker ownership, shard ID, epoch, dispatched version, committed version, and fallback state without exposing mutable runtime2 handles.

### Open Validation And Adoption

- [x] Add duplicate-registration tests at the public `ui` layer.
- [x] Add missing-renderer tests at the public `ui` layer.
- [x] Add invalid-props tests at the public `ui` layer.
- [x] Add invalid-source-ID tests at the public `ui` layer.
- [x] Add invalid-region-instance-ID tests at the public `ui` layer.
- [x] Add browser-native parity tests that prove native builds stay deterministic when worker-backed rendering is unavailable.
- [x] Add a minimal example app with one display-only parallel region.
- [x] Add a stress example app with many parallel regions across multiple workers.
- [x] Add a diagnostics example that demonstrates fallback, downgrade, and worker restart behavior.
- [x] Add authoring docs for first-slice allowed region shapes.
- [x] Add troubleshooting docs for fallback, protocol mismatch, binary transport, and shared-memory deployment requirements.
- [x] Add wasm tests proving browser-side prop changes advance the public region input version and produce runtime2 update dispatch instead of mount-only lifecycle state.
  Validation: `GOOS=js GOARCH=wasm go test -exec "C:\Users\Cam\Desktop\GoWebComponents\tools\go_js_wasm_exec.bat" ./ui -run TestParallelRegionRenderIntoPropChangesDispatchRuntime2Update$`
- [x] Add wasm tests proving declared-source-only changes rerender `ui.ParallelRegion(...)` and dispatch runtime2 updates without requiring prop changes or parent rerenders.
  Validation: `GOOS=js GOARCH=wasm go test -exec "C:\Users\Cam\Desktop\GoWebComponents\tools\go_js_wasm_exec.bat" ./ui -run TestParallelRegionDeclaredSourceOnlyChangesDispatchRuntime2Update$`
- [x] Add wasm tests proving transition-wrapped public region rerenders choose deferred runtime2 dispatch while urgent rerenders stay immediate.
  Validation: `GOOS=js GOARCH=wasm go test -exec "C:\Users\Cam\Desktop\GoWebComponents\tools\go_js_wasm_exec.bat" ./ui -run TestParallelRegionTransitionWrappedRerendersUseDeferredDispatch$`
- [x] Add wasm tests proving renderer-ID changes through the public `ui.ParallelRegion(...)` path trigger structural remount and epoch advancement.
  Validation: `GOOS=js GOARCH=wasm go test -exec "C:\Users\Cam\Desktop\GoWebComponents\tools\go_js_wasm_exec.bat" ./ui -run TestParallelRegionRendererIdentityChangesTriggerStructuralRemount$`
- [x] Add wasm tests proving `HydrateInto(...)` maps runtime2 parallel-region shell markers to cached adapters and marks hydration complete.
  Validation: `GOOS=js GOARCH=wasm go test -exec "C:\Users\Cam\Desktop\GoWebComponents\tools\go_js_wasm_exec.bat" ./ui -run "TestHydrateIntoMarksParallelRegionAdapterHydrationComplete$"`
- [x] Add hydration tests proving shell-marker discovery, hydrated-anchor registration, and post-hydration attach happen through the public `ui` APIs rather than only through runtime2 internals.
  Validation: `GOOS=js GOARCH=wasm go test -exec "C:\Users\Cam\Desktop\GoWebComponents\tools\go_js_wasm_exec.bat" ./ui -run TestHydrateIntoMarksParallelRegionAdapterHydrationComplete$` and `GOOS=js GOARCH=wasm go test -exec "C:\Users\Cam\Desktop\GoWebComponents\tools\go_js_wasm_exec.bat" ./ui -run TestHydrateMarksParallelRegionAdapterAnchorAndAttachBySelector$`
- [ ] Add an example page that surfaces the public read-only region status helper so adopters can see ownership, shard, versions, and fallback state without internal runtime2 code.
- [ ] Add authoring docs for transition semantics and deferred snapshot publication with `StartTransition(...)`.
- [ ] Add operator-facing docs for region runtime status fields: local or worker ownership, shard, epoch, input/commit versions, and fallback reason.
- [ ] Add a diagnostics example panel that surfaces per-region round-trip latency and dropped stale patch count.
- [ ] Update the parallel-region docs and examples to remove stale "still being wired" or "simulates transitions" copy and describe the current local-shell plus runtime2-dispatch boundary precisely.

### Recommended First Pick

- [ ] Add a read-only public region-status helper for examples and tooling that exposes local-or-worker ownership, shard ID, epoch, dispatched version, committed version, and fallback state without exposing mutable runtime2 handles.

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
- [x] Add snapshot-transport selection hooks into the host update-dispatch path.
  Validation: `go test ./internal/runtime2 -run "TestHandleHostRegionUpdateDispatchWithTransport(SelectsSharedTier|SkipsNoChange)"`
- [x] Add patch-transport selection hooks into the worker patch-ready path.
  Validation: `go test ./internal/runtime2 -run "TestHandleWorkerRegionUpdateWithPatchTransport(BuildsPatchReadyPayload|SkipsNoOp|RejectsUnsupportedCapability)"`
- [x] Add a `pong` control-plane message contract for runtime liveness signaling.
  Validation: `go test ./internal/runtime2 -run "TestParseControlEnvelopeJSON(AcceptsPongEnvelope|RejectsMissingPongShardID)"`
- [x] Add helper builders for `pong` control envelopes.
  Validation: `go test ./internal/runtime2 -run "TestBuildControl(PongEnvelopeBuildsValidatedEnvelope|.*EnvelopeBuildsValidatedEnvelope)"`
- [x] Add host-side control dispatch routing for `pong` keepalive messages.
  Validation: `go test ./internal/runtime2 -run "TestHandleHostControlEnvelope(Dispatches(PatchReady|Diagnostic|Restart|Pong)|RejectsUnsupportedKind)"`
- [x] Add scheduler keepalive timeout handling that marks workers degraded or dead after missed `pong` windows.
  Validation: `go test ./internal/runtime2 -run "TestHandleSchedulerKeepalive(TimeoutTransitionsToDegradedThenDead|PongRestoresReady|PongRejectsStaleSequence)|TestHandleHostControlEnvelopeDispatchesPong"`
- [x] Add a runtime2 shard-session abstraction backed by real `interop` worker or `MessagePort` primitives instead of dispatcher-only helper functions.
  Validation: `go test ./internal/runtime2 -run "Test(BuildShardSessionBindsInboundPortHandler|HandleShardSessionSendPayloadUsesPort)"`
- [x] Add package-level runtime2 capability initialization, override, and reset hooks so `GetCapabilityReport()` can reflect real platform support instead of always normalizing an empty source.
  Validation: `go test ./internal/runtime2 -run "Test(InitCapabilityReportStoresDetectedCapabilities|SetCapabilityReportOverrideAndReset)$"`
- [x] Add browser-backed or `interop`-backed capability detection that populates worker, `MessagePort`, binary-transport, and shared-memory support for live wasm dispatch.
  Validation: `go test ./internal/runtime2 -run "Test(DetectCapabilitySourceDefaultsOutsideBrowserWASM|InitCapabilityReportFromRuntimeUsesDetectedSource)$"`
- [x] Add a real `ready` or `capabilities` handshake over one shard session before mount and update traffic is accepted.
  Validation: `go test ./internal/runtime2 -run "Test(BuildShardSessionBindsInboundPortHandler|HandleShardSessionSendPayloadUsesPort|HandleShardSessionAcceptControlEnvelopeRequiresHandshakeBeforeMountOrUpdate)"`
- [x] Add host-side control send helpers that serialize and post mount, update, cancel, dispose, and restart envelopes over one shard session.
  Validation: `go test ./internal/runtime2 -run "Test(HandleShardSessionSendMountControlEnvelopeRequiresHandshake|HandleShardSessionSendLifecycleControlHelpersPostEnvelopes|HandleShardSessionAcceptControlEnvelopeRequiresHandshakeBeforeMountOrUpdate)"`
- [x] Add worker-side control receive or send helpers that parse control envelopes from one shard session and emit patch-ready, diagnostic, restart, ready, capabilities, and pong envelopes back over the same session.
  Validation: `go test ./internal/runtime2 -run "Test(HandleShardSessionReceiveControlEnvelopeParsesInboundControlTraffic|HandleShardSessionSendWorkerControlHelpersPostEnvelopes|HandleShardSessionSendLifecycleControlHelpersPostEnvelopes)"`
- [x] Add shard-session port teardown semantics so repaired or replaced sessions can unbind handlers, reject late sends, and clear queued inbound payloads cleanly.
  Validation: `go test ./internal/runtime2 -run "TestHandleShardSession(TeardownRejectsLateSendsAndClearsQueuedPayloads|ReplacePortResetsHandshakeAndRebindsInboundHandler)$"`
- [x] Add shard-session inbound queue bounds or synchronization so bursty inbound control or payload traffic cannot race or grow without limit.
  Validation: `go test ./internal/runtime2 -run "Test(BuildShardSessionWithQueueLimitCapsInboundPayloadQueue|BuildShardSessionBindsInboundPortHandler|HandleShardSessionSendPayloadUsesPort)$"`
- [x] Add shard-session raw patch-payload send and receive helpers so `patch-ready` control envelopes can reference real queued payload bytes over one live session instead of only in-memory test payloads.
  Validation: `go test ./internal/runtime2 -run "TestHandleShardSession(SendAndReceivePatchReadyWithPayload|SendPatchReadyWithPayloadRejectsNonPatchReadyEnvelope)$"`
- [x] Add a structured-clone patch payload envelope format that carries region ID, epoch, patch version, and raw patch payload bytes together.
  Validation: `go test ./internal/runtime2 -run "Test(BuildStructuredClonePatchEnvelopeJSONRoundTrips|ParseStructuredClonePatchEnvelopeJSONRejectsMalformedPayload)"`
- [x] Add binary patch payload encoding and decoding for `PatchStreamRaw`.
  Validation: `go test ./internal/runtime2 -run "Test(BuildBinaryPatchPayloadRoundTrips|ParseBinaryPatchPayloadRejectsMalformedFrame)"`
- [x] Add shared-buffer patch page layout, publish, read, and downgrade handling for patch payloads.
  Validation: `go test ./internal/runtime2 -run "Test(HandleSharedPatchPublishAndReadRoundTrip|BuildSharedPatchTransportResultPrefersSharedBuffer|BuildSharedPatchTransportResultDowngradesWhenSharedUnavailable|ParseSharedSnapshotPageHeaderRejectsUnsupportedKind)"`
- [x] Add patch-transport selection that prefers shared-buffer, then binary, then structured-clone symmetrically with snapshot transport.
  Validation: `go test ./internal/runtime2 -run "Test(SelectPatchTransportTier(PrefersShared|PrefersBinaryWithoutSharedPage|FallsBackToStructuredClone|RejectsUnsupportedContract)|HandleWorkerRegionUpdateWithPatchTransport(BuildsPatchReadyPayload|SkipsNoOp|RejectsUnsupportedCapability))"`
- [x] Add host-side patch payload parsing with transport fallback so one `patch-ready` control envelope can drive structured-clone, binary, or shared-buffer payload decode consistently.
  Validation: `go test ./internal/runtime2 -run "TestParseHostPatchPayloadWithFallback(StructuredEnvelope|BinaryTierFallsBackToStructured|SharedTier)"`
- [x] Add session restart or renegotiation handling that re-runs `ready` and `capabilities` handshake after worker repair or channel replacement.
  Validation: `go test ./internal/runtime2 -run "TestHandleShardSession(AcceptControlEnvelopeRestartResetsHandshake|ReplacePortResetsHandshakeAndRebindsInboundHandler|SendMountControlEnvelopeRequiresHandshake|SendLifecycleControlHelpersPostEnvelopes|SendWorkerControlHelpersPostEnvelopes|AcceptControlEnvelopeRequiresHandshakeBeforeMountOrUpdate)$"`

### Open Validation And Benchmarks

- [x] Add malformed control-envelope decode tests.
  Validation: `go test internal/runtime2/protocol.go internal/runtime2/registry.go internal/runtime2/spec.go internal/runtime2/capabilities.go internal/runtime2/snapshot.go internal/runtime2/scheduler_shard_identity.go internal/runtime2/recovery_coordinator.go internal/runtime2/diagnostic_event_kind.go internal/runtime2/diagnostic_timing.go internal/runtime2/diagnostic_size.go internal/runtime2/diagnostic_fallback_reason.go internal/runtime2/diagnostic_trace.go internal/runtime2/diagnostic_shard.go internal/runtime2/diagnostic_downgrade_reason.go internal/runtime2/control.go internal/runtime2/control_malformed_internal_test.go -run "TestParseControlEnvelopeJSONRejects(MalformedJSON|NonObjectPayload)"`
- [x] Add malformed snapshot-envelope decode tests.
  Validation: `go test internal/runtime2/registry.go internal/runtime2/spec.go internal/runtime2/snapshot.go internal/runtime2/structured_clone_snapshot_transport.go internal/runtime2/snapshot_malformed_decode_internal_test.go -run "TestParseStructuredCloneSnapshotEnvelopeJSONRejects(MalformedJSON|NonObjectPayload)"`
- [x] Add fuzz coverage for control-plane envelope decoding.
  Validation: `go test internal/runtime2/protocol.go internal/runtime2/registry.go internal/runtime2/spec.go internal/runtime2/capabilities.go internal/runtime2/snapshot.go internal/runtime2/scheduler_shard_identity.go internal/runtime2/recovery_coordinator.go internal/runtime2/diagnostic_event_kind.go internal/runtime2/diagnostic_timing.go internal/runtime2/diagnostic_size.go internal/runtime2/diagnostic_fallback_reason.go internal/runtime2/diagnostic_trace.go internal/runtime2/diagnostic_shard.go internal/runtime2/diagnostic_downgrade_reason.go internal/runtime2/control.go internal/runtime2/control_fuzz_internal_test.go -run=^$ -fuzz=FuzzParseControlEnvelopeJSON -fuzztime=3s`
- [x] Add fuzz coverage for snapshot-envelope decoding.
  Validation: `go test internal/runtime2/registry.go internal/runtime2/spec.go internal/runtime2/snapshot.go internal/runtime2/structured_clone_snapshot_transport.go internal/runtime2/snapshot_fuzz_internal_test.go -run=^$ -fuzz=FuzzParseStructuredCloneSnapshotEnvelopeJSON -fuzztime=3s`
- [x] Add a microbenchmark for structured-clone snapshot encoding.
  Validation: `go test internal/runtime2/registry.go internal/runtime2/spec.go internal/runtime2/snapshot.go internal/runtime2/structured_clone_snapshot_transport.go internal/runtime2/structured_clone_snapshot_bench_internal_test.go -run=^$ -bench BenchmarkBuildStructuredCloneSnapshotEnvelopeJSON -benchmem`
- [x] Add package-level capability-init tests proving live capability detection, test overrides, and reset paths affect transport-tier selection deterministically.
  Validation: `go test ./internal/runtime2 -run "TestCapabilityInitOverrideResetAffectPatchTransportSelectionDeterministically$"`
- [x] Add malformed `pong` control-envelope decode tests.
  Validation: `go test ./internal/runtime2 -run "TestParseControlEnvelopeJSON(RejectsMissingPongSequence|RejectsMalformedPongShardIDType|RejectsMalformedPongSequenceType|RejectsMissingPongShardID|AcceptsPongEnvelope)$"`
- [x] Add keepalive timeout tests for healthy, degraded, and dead transitions under missed `pong` delivery.
  Validation: `go test ./internal/runtime2 -run "TestHandleSchedulerKeepalive(TimeoutTransitionsHealthyToDegradedToDead|TimeoutTransitionsToDegradedThenDead|PongRestoresReady|PongRejectsStaleSequence)$"`
- [x] Add session-level tests proving `ready` and `capabilities` handshake must complete before mount or update envelopes are accepted.
  Validation: `go test ./internal/runtime2 -run "TestHandleShardSession(AcceptControlEnvelopeRequiresHandshakeBeforeMountOrUpdate|SendMountControlEnvelopeRequiresHandshake)$"`
- [x] Add shard-session teardown tests proving late sends fail clearly, late inbound payloads are ignored, and repair can bind one fresh handler without duplicate delivery.
  Validation: `go test ./internal/runtime2 -run "TestHandleShardSession(TeardownRejectsLateSendsAndClearsQueuedPayloads|ReplacePortResetsHandshakeAndRebindsInboundHandler)$"`
- [ ] Add bounded-inbound-queue tests proving burst traffic is either capped or rejected deterministically instead of growing unbounded.
- [x] Add end-to-end transport tests for structured-clone, binary, and shared-buffer patch payload selection over one shard session.
  Validation: `go test ./internal/runtime2 -run "TestHandleShardSessionPatchTransportRoundTrip(StructuredClone|Binary|SharedBuffer)$"`
- [x] Add negative tests for malformed patch payload envelopes, wrong-region patch payloads, and wrong-version patch payloads before host commit.
  Validation: `go test ./internal/runtime2 -run "TestParseHostPatchPayloadWithFallback(StructuredEnvelope|BinaryTierFallsBackToStructured|SharedTier|RejectsMalformedPayload|RejectsWrongRegionBeforeCommit|RejectsWrongVersionBeforeCommit)$"`
- [x] Add fuzz coverage for structured-clone patch payload decoding, binary patch payload decoding, and shared-buffer patch-page parsing.
  Validation: `go test ./internal/runtime2 -run=^$ -fuzz=FuzzParseStructuredClonePatchEnvelopeJSON -fuzztime=3s`
  Validation: `go test ./internal/runtime2 -run=^$ -fuzz=FuzzParseBinaryPatchPayload -fuzztime=3s`
  Validation: `go test ./internal/runtime2 -run=^$ -fuzz=FuzzParseSharedPatchPayloadFromPage -fuzztime=3s`
- [x] Add microbenchmarks for binary patch encode or decode and shared-buffer patch publish or read.
  Validation: `go test ./internal/runtime2 -run ^$ -bench "Benchmark(BuildBinaryPatchPayload|ParseBinaryPatchPayload|HandleSharedPatchPublishPayload|GetSharedPatchReadPayload)$" -benchtime=1x`
- [x] Add control-plane liveness tests that prove the runtime remains event-driven and does not rely on blocking waits.
  Validation: `go test ./internal/runtime2 -run "TestHandleShardSessionReceiveControlEnvelope(ReturnsWithoutBlockingWait|DrainsQueueWithoutBlockingWait)$"`

### Recommended First Pick

- [ ] Add shard-session inbound queue bounds or synchronization so bursty inbound control or payload traffic cannot race or grow without limit.

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

- [x] Add conversion from validated display-only render output into canonical render-node raw records.
- [x] Add canonical string-table extraction from one render output tree.
- [x] Add canonical prop-record extraction from one render output tree.
- [x] Add a stable per-render node-ID allocation strategy for one region render.
- [x] Add root-node conventions for empty, single-text, and host-element region outputs.
- [x] Add worker-region state that stores parsed canonical IR instead of raw `any`.
- [x] Add one diff entrypoint from previous canonical IR to next canonical IR.
- [x] Add insert patch generation from newly introduced nodes.
- [x] Add remove patch generation from missing nodes.
- [x] Add set-text patch generation from text changes.
- [x] Add set-attr patch generation from prop changes.
- [x] Add remove-attr patch generation from prop removal.
- [x] Add keyed-move patch generation from keyed sibling reorders.
- [x] Add canonical patch ordering output from the diff engine.
- [x] Replace `reflect.DeepEqual(...)` no-op detection with canonical-IR equality.
- [x] Replace the `{previous,next}` patch placeholder with typed patch-stream output.
- [x] Add one patch payload parser entrypoint that validates patch-stream header, op ordering, and idempotency before commit.
- [x] Add one host commit entrypoint that feeds parsed patch transactions into `CommitRegionPatchTransaction(...)`.
- [x] Add stale patch-version suppression in the orchestration layer.
- [x] Add region-ID mismatch rejection in the orchestration layer.
- [x] Add epoch mismatch rejection in the orchestration layer.
- [x] Add set-style patch generation from normalized style value changes.
- [x] Add remove-style patch generation from normalized style removals.
- [x] Add replace-subtree patch generation for structural mismatch paths that cannot be represented as incremental ops.
- [x] Add host commit support for set-style and remove-style operations.
- [x] Add host commit support for replace-subtree operations with region DOM-index rebuild guarantees.
- [x] Extend `WorkerRegionMountSpec` so worker renderers receive the full validated snapshot envelope instead of only region ID, renderer ID, epoch, and input version.
- [x] Extend `WorkerRegionUpdateSpec` so worker update handlers also receive the full validated snapshot envelope.
- [x] Add worker-state snapshot caching so the worker can diff or validate against the last accepted props and source snapshot as well as canonical IR.
- [x] Add worker-side snapshot consistency validation that rejects region-ID, renderer-ID, or epoch mismatches between control envelopes and snapshot payloads.
- [x] Add one worker render-input adapter that maps snapshot props and declared sources into the registered worker renderer contract deterministically.
- [x] Add one host-side patch-consume entrypoint that accepts transport tier plus raw payload bytes, decodes the patch payload, and then calls `HandleHostRegionPatchCommit(...)`.
- [x] Add explicit `patch-ready` correlation so worker `patch-ready` envelopes carry the emitted patch-stream patch version even when no-op updates make patch version diverge from input version.
- [x] Add host-side patch-ready gating that tracks patch version and input version separately so stale patch streams are rejected without confusing repair floors or commit versions.
- [x] Add one end-to-end orchestration helper that chains host snapshot dispatch, worker update, patch transport encode, host patch decode, and host commit for one region update.

### Open Positive, Negative, And Edge Tests

- [x] Add end-to-end mount test for one display-only region over structured-clone transport.
- [x] Add end-to-end update test for one display-only region over structured-clone transport.
- [x] Add end-to-end no-op-update test where no patch is emitted.
- [x] Add end-to-end cancel test where an outdated patch never commits.
- [x] Add end-to-end dispose test where region state and DOM index are cleaned up.
- [x] Add end-to-end sticky-affinity test proving repeated updates stay on the same worker shard.
- [x] Add unknown renderer-ID tests through the real runtime path.
- [x] Add invalid patch-op tests.
- [x] Add wrong-epoch patch tests.
- [x] Add stale-version patch tests.
- [x] Add duplicate mount for the same active region tests.
- [x] Add zero-child region tests.
- [x] Add single-text-node region tests.
- [x] Add empty-text update tests.
- [x] Add empty-prop-set tests.
- [x] Add duplicate-key sibling tests.
- [x] Add set-style patch tests for add or update and remove-style patch tests for deletion semantics.
- [x] Add replace-subtree patch tests that verify atomic commit behavior and post-commit DOM-index consistency.
- [x] Add invalid set-style, remove-style, and replace-subtree patch-op payload tests.
- [x] Add worker-runtime tests proving snapshot props, declared source values, and source version changes all can change rendered output without relying on `InputVersion` alone.
- [x] Add negative tests proving worker mount or update rejects snapshot payloads whose region ID, renderer ID, or epoch disagree with the surrounding control envelope.
- [x] Add tests proving `patch-ready` envelope patch version matches the emitted patch-stream header after no-op updates create input-version gaps.
- [x] Add end-to-end tests that drive host patch commit from decoded structured-clone patch payload bytes rather than from an in-memory `PatchStreamRaw`.
- [x] Add end-to-end tests for binary patch payload decode plus host commit once Agent 2 lands binary patch transport.
- [x] Add end-to-end tests for shared-buffer patch payload read plus host commit once Agent 2 lands shared-buffer patch transport.
- [x] Add stale patch-payload correlation tests proving wrong patch versions or mismatched region payloads are ignored before DOM commit.
- [x] Add host-gating tests proving the next real patch after one or more no-op updates is accepted by patch version while stale patch versions are still rejected before commit.

### Open Fuzzing And Benchmarks

- [x] Add fuzz coverage for render IR decoding.
- [x] Add fuzz coverage for string-table decoding.
- [x] Add fuzz coverage for prop-record decoding.
- [x] Add fuzz coverage for patch IR decoding.
- [x] Add fuzz coverage for DOM-index patch-application prevalidation.
- [x] Add a microbenchmark for worker-side IR build.
- [x] Add a microbenchmark for worker-side diff.
- [x] Add a microbenchmark for patch decode.
- [x] Add a microbenchmark for patch commit.
- [x] Add an end-to-end comparison benchmark for local display-region rendering versus worker-backed rendering.
- [x] Add fuzz coverage for worker snapshot-to-render input adaptation so malformed snapshot props or source maps cannot panic the worker diff path.
- [x] Add a microbenchmark for snapshot-driven worker render invocation versus the current metadata-only worker render path.

### Recommended First Pick

All items in this lane are complete.

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

- [x] Add region unmount semantics when the owner stops rendering the region.
- [x] Add transition-aware dispatch semantics.
- [x] Add owner removal handling that disposes the region and suppresses late worker output.
- [x] Add structural remount detection when renderer identity or shell ownership changes.
- [x] Add remount epoch advancement on structural remount.
- [x] Add `HandleHostRegionOwnerRemove(...)`.
- [x] Add `HandleHostRegionStructuralRemount(...)`.
- [x] Add `HandleHostRegionFallbackMirror(...)`.
- [x] Add `HandleHostRegionPatchReady(...)`.
- [x] Add `HandleHostRegionWorkerDeath(...)`.
- [x] Add `HandleHostRegionRepairRemount(...)`.
- [x] Add stable getter helpers for fallback-pending, fallback-active, repair-pending, repair epoch, repair version floor, and latest valid version when tests need explicit visibility.
- [x] Route structured-clone decode failure through the existing recovery coordinator.
- [x] Route binary decode failure through the existing recovery coordinator.
- [x] Route shared-page decode failure through the existing recovery coordinator.
- [x] Route DOM patch-transaction failure through the existing recovery coordinator.
- [x] Mirror recovery fallback state into the main coordinator state machine.
- [x] Mirror recovery fallback state into scheduler fallback ownership.
- [x] Block worker commit attempts once fallback ownership is active in the host pipeline.
- [x] Wire dead-worker detection from scheduler or transport failure into `HandleWorkerDeath(...)`.
- [x] Allocate a fresh remount epoch after successful worker reassignment.
- [x] Propagate the remount epoch back into coordinator state.
- [x] Reissue a clean mount after worker reassignment before allowing updates.
- [x] Keep the latest valid input version during repair-driven remount.
- [x] Reject patch-ready results produced before repair completes.
- [x] Reject patch-ready results produced before fallback ownership begins.
- [x] Keep fallback ownership active until one fresh remount succeeds.
- [x] Clear fallback ownership only after a healthy remount handshake completes.
  Validation: `go test ./internal/runtime2 -run "HostRegion|ParseControlEnvelopeJSONAcceptsDiagnosticEnvelopeWithStructuredPayloadFields|ValidateControlEnvelopeAcceptsStructuredTimingDiagnostic|ValidateControlEnvelopeAcceptsStructuredSizeDiagnostic|ValidateControlEnvelopeAcceptsStructuredFallbackDiagnostic|ValidateControlEnvelopeAcceptsDebugTraceDiagnostic" -count=1`
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
- [x] Add shell-identity mismatch detection for region ID mismatches.
  Validation: `go test ./internal/runtime2 -run "HostRegionShellIdentityMismatchDetection" -count=1`
- [x] Add shell-identity mismatch detection for renderer ID mismatches.
  Validation: `go test ./internal/runtime2 -run "HostRegionShellIdentityMismatchDetection" -count=1`
- [x] Add shell-missing-anchor detection.
  Validation: `go test ./internal/runtime2 -run "HostRegionShellMissingAnchorDetection" -count=1`
- [x] Add local remount fallback on shell mismatch.
  Validation: `go test ./internal/runtime2 -run "HostRegionShellMismatchFallback" -count=1`
- [x] Drop pending worker output after hydration mismatch fallback.
  Validation: `go test ./internal/runtime2 -run "HostRegionShellMismatchFallback" -count=1`
- [x] Require a fresh epoch before later reattach after mismatch recovery.
  Validation: `go test ./internal/runtime2 -run "HostRegionShellMismatchFallback" -count=1`
- [x] Add coordinator tracking for `attached` state transitions at mount, hydration attach, fallback, and dispose boundaries.
  Validation: `go test ./internal/runtime2 -run "Test(SetRegionAttachedStoresAttachedState|HandleHostRegionMountCoordinatorStartsDetached|HandleHostRegionPostHydrationAttachSetsCoordinatorAttached|HandleHostRegionFallbackOwnershipBeginClearsCoordinatorAttached|HandleHostRegionDisposeAfterAttachRemovesCoordinatorEntry)" -count=1`
- [x] Add coordinator tracking for canonical `sourceIDs` and `lastSnapshotVersion` per live region.
  Validation: `go test ./internal/runtime2 -run "Test(SetRegionSourceIDsStoresCanonicalSourceIDs|SetRegionLastSnapshotVersionTracksMonotonicVersion|HandleHostRegionMountMountsValidatedParallelRegion|HandleHostRegionUpdateSnapshotCapturesPropsAndSources)" -count=1`
- [x] Add a region runtime-status getter for observability surfaces with region mode, shard, renderer, epoch, versions, transport tier, and fallback reason.
  Validation: `go test ./internal/runtime2 -run "Test(GetHostRegionRuntimeStatusReports(LocalShellMode|WorkerAttachedMode|FallbackMode)|HandleHostRegionUpdateDispatchWithTransport(SelectsSharedTier|SkipsNoChange)|HandleHostRegionStructuredCloneDecodeFailureRoutesThroughRecovery)" -count=1`
- [x] Add dropped stale patch-result counters at the host coordinator layer.
  Validation: `go test ./internal/runtime2 -run "Test(HandleHostRegionPatchReadyStaleBeforeRepairFloorIncrementsDroppedCounter|HandleHostRegionWorkerOutputOlderVersionIncrementsDroppedCounter)" -count=1`
- [x] Add round-trip timing capture from host dispatch through patch-ready and commit completion.
  Validation: `go test ./internal/runtime2 -run "Test(HandleHostRegionRoundTripTimingCapturesDispatchPatchAndCommit|GetHostRegionRuntimeStatusReports(LocalShellMode|WorkerAttachedMode|FallbackMode))" -count=1`
- [x] Add diagnostic redaction rules that prevent source snapshot values from leaking through emitted diagnostics.
  Validation: `go test internal/runtime2/protocol.go internal/runtime2/registry.go internal/runtime2/spec.go internal/runtime2/capabilities.go internal/runtime2/snapshot.go internal/runtime2/scheduler_shard_identity.go internal/runtime2/recovery_coordinator.go internal/runtime2/diagnostic_event_kind.go internal/runtime2/diagnostic_timing.go internal/runtime2/diagnostic_size.go internal/runtime2/diagnostic_fallback_reason.go internal/runtime2/diagnostic_trace.go internal/runtime2/diagnostic_shard.go internal/runtime2/diagnostic_downgrade_reason.go internal/runtime2/control.go internal/runtime2/control_builders.go internal/runtime2/diagnostic_redaction.go internal/runtime2/control_test.go internal/runtime2/control_diagnostic_payload_internal_test.go internal/runtime2/control_diagnostic_timing_internal_test.go internal/runtime2/control_diagnostic_size_internal_test.go internal/runtime2/control_diagnostic_fallback_internal_test.go internal/runtime2/control_diagnostic_trace_internal_test.go internal/runtime2/control_diagnostic_shard_internal_test.go internal/runtime2/control_diagnostic_transport_internal_test.go internal/runtime2/control_diagnostic_downgrade_internal_test.go internal/runtime2/diagnostic_redaction_test.go -run "Test(ParseControlEnvelopeJSON|ValidateControlEnvelope|RedactDiagnosticTextRedactsJSONSourceSnapshotAndSecrets|RedactDiagnosticTextRedactsKeyValueSecrets|BuildControlDiagnosticEnvelopeRedactsDiagnosticText|BuildControlEnvelopeJSONRedactsDiagnosticText)" -count=1`
- [x] Add a host-side diagnostic ring buffer per region so `diagnostic` control envelopes become durable runtime state instead of transient parse results.
  Validation: `go test internal/runtime2/host_region_recovery_mirror_test.go internal/runtime2/host_control_dispatcher_test.go internal/runtime2/host_region_diagnostic_ring_test.go -run "Test(HandleHostControlEnvelopeDispatchesDiagnostic|HandleHostControlEnvelopeDiagnosticStoresHostDiagnosticRing|HandleHostRegionDisposeClearsHostDiagnosticRing)" -count=1`
- [x] Add stale-diagnostic suppression keyed by region, epoch, and input version so delayed worker diagnostics cannot overwrite newer fallback or repair state.
  Validation: `go test internal/runtime2/host_region_recovery_mirror_test.go internal/runtime2/host_control_dispatcher_test.go internal/runtime2/host_region_diagnostic_ring_test.go internal/runtime2/host_region_diagnostic_stale_test.go -run "Test(HandleHostControlEnvelopeDispatchesDiagnostic|HandleHostControlEnvelopeDiagnosticStoresHostDiagnosticRing|HandleHostRegionDisposeClearsHostDiagnosticRing|HandleHostControlEnvelopeDiagnosticIgnoresStaleEpoch|HandleHostControlEnvelopeDiagnosticIgnoresStaleVersion|HandleHostControlEnvelopeDiagnosticIgnoresFallbackOwnedDiagnostic)" -count=1`
- [x] Add host-side downgrade accounting that records the latest snapshot transport downgrade and latest patch transport downgrade separately for one region.
  Validation: `go test internal/runtime2/host_region_recovery_mirror_test.go internal/runtime2/host_snapshot_transport_hook_test.go internal/runtime2/host_control_dispatcher_test.go internal/runtime2/host_region_downgrade_accounting_test.go -run "Test(HandleHostRegionUpdateDispatchWithTransportSelectsSharedTier|HandleHostRegionUpdateDispatchWithTransportSkipsNoChange|HandleHostControlEnvelopeDispatchesPatchReady|HostRegionTransportDowngradeStatusTracksSnapshotAndPatchSeparately)" -count=1`
- [x] Add host-side counters for ignored stale diagnostics and repair-triggered remounts alongside the existing stale patch-result tracking.
  Validation: `go test internal/runtime2/coordinator_state_test.go internal/runtime2/host_region_recovery_mirror_test.go internal/runtime2/host_region_diagnostic_stale_test.go internal/runtime2/host_region_worker_death_repair_test.go -run "Test(IncrementRegionIgnoredStaleDiagnosticCountTracksStaleDiagnosticDrops|IncrementRegionRepairRemountCountTracksSuccessfulRepairs|HandleHostControlEnvelopeDiagnosticIgnoresStaleEpoch|HandleHostRegionRepairRemountIncrementsCoordinatorRepairCounter|HandleHostRegionRepairRemountReissuesCleanMountBeforeUpdates)" -count=1`
- [x] Add host-side hydration attach helpers that can be called from the public `ui` layer without exposing the mutable adapter directly.
  Validation: `go test internal/runtime2/host_region_recovery_mirror_test.go internal/runtime2/host_region_hydration_attach_test.go internal/runtime2/host_region_coordinator_attached_test.go internal/runtime2/host_region_hydration_helper_test.go -run "Test(BuildHostRegionHydrationAttachHelperRejectsNilAdapter|HostRegionHydrationAttachHelperRequiresHydrationAndAnchor|HandleHostRegionPostHydrationAttachBlocksBeforeHydrationComplete|HandleHostRegionPostHydrationAttachBlocksWithoutRegisteredAnchor|HandleHostRegionPostHydrationAttachSetsCoordinatorAttached)" -count=1`
- [x] Add a read-only host-side diagnostics snapshot getter that returns recent redacted diagnostic events, downgrade state, and counters for one region without exposing mutable coordinator state.
  Validation: `go test internal/runtime2/host_region_recovery_mirror_test.go internal/runtime2/host_snapshot_transport_hook_test.go internal/runtime2/host_control_dispatcher_test.go internal/runtime2/host_region_diagnostics_snapshot_test.go -run "Test(GetHostRegionDiagnosticsSnapshotReportsReadOnlyRedactedState|HandleHostControlEnvelopeDispatchesDiagnostic|HandleHostRegionUpdateDispatchWithTransportSelectsSharedTier)" -count=1`
- [x] Extend host runtime-status snapshots with hydration-attach state, latest snapshot and patch downgrade reasons, and stale-output counters so public status helpers do not need multiple runtime2 calls.
  Validation: `go test internal/runtime2/host_region_recovery_mirror_test.go internal/runtime2/host_control_dispatcher_test.go internal/runtime2/host_region_runtime_status_test.go -run "Test(GetHostRegionRuntimeStatusReports(LocalShellMode|WorkerAttachedMode|FallbackMode|DowngradeReasonsAndStaleCounters)|HandleHostControlEnvelopeDispatchesDiagnostic)" -count=1`

### Open Validation And Performance

- [x] Add worker-restart race tests.
- [x] Add worker-death-during-patch tests.
- [x] Add rapid mount-dispose-mount churn tests.
- [x] Add rapid update-cancel-update churn tests.
- [x] Add dispose-during-fallback tests.
- [x] Add fallback-then-remount tests.
- [x] Add multiple-regions-sharing-one-renderer-ID tests.
- [x] Add one-worker-many-regions pressure tests.
- [x] Add many-workers-few-regions skew tests.
- [x] Add diagnostics payload tests for timing, size, fallback-reason, and trace fields.
- [x] Add SSR shell-marker encode or decode tests.
- [x] Add hydration attach tests.
- [x] Add hydration mismatch fallback tests.
- [x] Add a microbenchmark for source snapshot capture.
- [x] Add a pressure benchmark for many hot regions sharing a bounded worker set.
  Validation: `go test ./internal/runtime2 -run "HostRegion|SSRShellMarker|ParseControlEnvelopeJSONAcceptsDiagnosticEnvelopeWithStructuredPayloadFields" -count=1`
  Validation: `go test ./internal/runtime2 -run ^$ -bench "BenchmarkHandleHostRegion(UpdateSnapshot|ManyHotRegionsBoundedWorkers)$" -benchtime=1x`
- [x] Add region runtime-status payload tests for mode, shard, renderer, epoch, versions, transport, and fallback fields.
  Validation: `go test internal/runtime2/host_region_recovery_mirror_test.go internal/runtime2/host_control_dispatcher_test.go internal/runtime2/host_region_runtime_status_test.go -run "Test(GetHostRegionRuntimeStatusReports(LocalShellMode|WorkerAttachedMode|FallbackMode|DowngradeReasonsAndStaleCounters)|HandleHostControlEnvelopeDispatchesDiagnostic)" -count=1`
- [x] Add stale patch-result drop-counter tests.
  Validation: `go test internal/runtime2/host_region_recovery_mirror_test.go internal/runtime2/host_region_stale_patch_counter_test.go -run "Test(HandleHostRegionPatchReadyStaleBeforeRepairFloorIncrementsDroppedCounter|HandleHostRegionWorkerOutputOlderVersionIncrementsDroppedCounter)" -count=1`
- [x] Add round-trip timing metric tests for dispatch to patch-ready to commit flow.
  Validation: `go test internal/runtime2/host_region_recovery_mirror_test.go internal/runtime2/host_region_round_trip_timing_test.go -run "TestHandleHostRegionRoundTripTimingCapturesDispatchPatchAndCommit" -count=1`
- [x] Add diagnostics redaction tests that verify source values and potential secrets are not emitted.
  Validation: `go test internal/runtime2/protocol.go internal/runtime2/registry.go internal/runtime2/spec.go internal/runtime2/capabilities.go internal/runtime2/capabilities_detect_native.go internal/runtime2/snapshot.go internal/runtime2/scheduler_shard_identity.go internal/runtime2/recovery_coordinator.go internal/runtime2/diagnostic_event_kind.go internal/runtime2/diagnostic_timing.go internal/runtime2/diagnostic_size.go internal/runtime2/diagnostic_fallback_reason.go internal/runtime2/diagnostic_trace.go internal/runtime2/diagnostic_shard.go internal/runtime2/diagnostic_downgrade_reason.go internal/runtime2/control.go internal/runtime2/control_builders.go internal/runtime2/diagnostic_redaction.go internal/runtime2/diagnostic_redaction_test.go -run "Test(RedactDiagnosticTextRedactsJSONSourceSnapshotAndSecrets|RedactDiagnosticTextRedactsKeyValueSecrets|BuildControlDiagnosticEnvelopeRedactsDiagnosticText|ParseControlEnvelopeJSONRedactsDiagnosticText|BuildControlEnvelopeJSONRedactsDiagnosticText)" -count=1`
- [x] Add diagnostic ring-buffer trim tests that keep the newest fallback, repair, and transport-downgrade events in deterministic order.
  Validation: `go test internal/runtime2/host_region_recovery_mirror_test.go internal/runtime2/host_region_diagnostic_ring_test.go -run "Test(HandleHostControlEnvelopeDiagnosticStoresHostDiagnosticRing|HandleHostRegionDisposeClearsHostDiagnosticRing|HandleHostControlEnvelopeDiagnosticRingTrimKeepsNewestDeterministicOrder)" -count=1`
- [x] Add stale-diagnostic suppression tests proving older-epoch or older-version diagnostics are ignored after fallback, repair, or remount.
  Validation: `go test internal/runtime2/host_region_recovery_mirror_test.go internal/runtime2/host_region_diagnostic_stale_test.go -run "Test(HandleHostControlEnvelopeDiagnosticIgnoresStaleEpoch|HandleHostControlEnvelopeDiagnosticIgnoresStaleVersion|HandleHostControlEnvelopeDiagnosticIgnoresFallbackOwnedDiagnostic)" -count=1`
- [x] Add downgrade-accounting tests that prove snapshot-tier and patch-tier downgrade state are tracked independently.
  Validation: `go test internal/runtime2/host_region_recovery_mirror_test.go internal/runtime2/host_snapshot_transport_hook_test.go internal/runtime2/host_control_dispatcher_test.go internal/runtime2/host_region_downgrade_accounting_test.go -run "Test(HostRegionTransportDowngradeStatusTracksSnapshotAndPatchSeparately|HandleHostRegionUpdateDispatchWithTransportSelectsSharedTier|HandleHostControlEnvelopeDispatchesPatchReady)" -count=1`
- [ ] Add host-side hydration-helper tests that prove public attach calls cannot mark the region attached without both hydration completion and shell-anchor registration.
- [ ] Add diagnostics-snapshot getter tests proving returned entries are redacted, ordered deterministically, and isolated per region.
- [ ] Add extended runtime-status payload tests for hydration-attach state, latest downgrade reasons, and stale-output counters.

### Recommended First Pick

- [ ] Extend host runtime-status snapshots with hydration-attach state, latest snapshot and patch downgrade reasons, and stale-output counters so public status helpers do not need multiple runtime2 calls.

## Cross-Lane Handoffs

- After Agent 2 finishes control-envelope builders and dispatch hooks, Agent 3 can wire real mount and update orchestration through the control plane.
- After Agent 3 replaces placeholder patches with typed patch streams, Agent 2 can finish patch-transport hooks cleanly.
- After Agent 4 lands patch-ready gating, remount, and fallback mirrors, Agent 3 can tighten stale-output rejection and full end-to-end commit coverage.
- After Agent 1 lands the public `ui` surface, Agent 3 and Agent 4 can validate the real user-facing path instead of only runtime2 internals.
- After Agent 4 lands SSR shell markers and hydration attach, Agent 1 can finalize adoption docs and examples without hand-waving.

## Recommended First Pick Per Agent

- Agent 1: `Add public hydrated-shell anchor registration that calls HandleHostRegionRegisterHydratedShellAnchor(...) with the resumed shell node before worker attach is attempted.`
- Agent 2: `Add shard-session inbound queue bounds or synchronization so bursty inbound control or payload traffic cannot race or grow without limit.`
- Agent 3: All items in this lane are complete.
- Agent 4: `Extend host runtime-status snapshots with hydration-attach state, latest snapshot and patch downgrade reasons, and stale-output counters so public status helpers do not need multiple runtime2 calls.`

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

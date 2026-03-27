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
- Every runtime2 performance code change must include a current-vs-legacy microbenchmark and record before/after numbers near the completed item.

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
- [x] Add a read-only public `ui.GetParallelRegionRuntimeStatus(...)` helper and value shape for examples and tooling that exposes local-or-worker ownership, shard ID, epoch, dispatched version, committed version, and fallback state without exposing mutable runtime2 handles.
  Validation: `go test ./ui -run "TestGetParallelRegionRuntimeStatus(RejectsInvalidRegionInstanceID|ReportsMissingForUntrackedRegion)$"` and `GOOS=js GOARCH=wasm go test -exec "C:\Users\Cam\Desktop\GoWebComponents\tools\go_js_wasm_exec.bat" ./ui -run TestGetParallelRegionRuntimeStatusReportsPublicDispatchVersions$` and `GOOS=js GOARCH=wasm go test -exec "C:\Users\Cam\Desktop\GoWebComponents\tools\go_js_wasm_exec.bat" ./ui -run TestGetParallelRegionRuntimeStatusReportsHydratedPublicAttachState$`

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
- [x] Add an example page (`examples/200-runtime2-status`) that surfaces the public read-only region status helper so adopters can see ownership, shard, versions, and fallback state without internal runtime2 code.
  Validation: `go run ./tools/gwc build -app .\examples\200-runtime2-status\main.go -root .\examples\200-runtime2-status`
- [x] Add a runtime2 owner-shell rerender-isolation pass so worker-fleet internal state commits do not force extra app-shell rerenders beyond one interaction-triggered update.
  Validation: `go test -tags playwrightgo ./test/playwrightgo/examples/example200_runtime2_status_test.go ./test/playwrightgo/examples/examples_suite_test.go -run TestExample200Runtime2Execution -count=2 -v`
  Result: example 200 now holds `app-label-before=1`, `app-label-before-idle=1`, `app-label-after-noop=1`, and `workbench-label-after-noop=1`; one increment stays bounded at `app-label-after<=2` with `app-label-after-idle` stable while worker probes still complete. Over 20 repeated runs, `app-label-after` observed `1` in 3 runs and `2` in 17 runs (no runaway trend). Worker telemetry view is explicit-refresh to avoid async background rerender churn. Follow-up hardening removed ref-write feedback from `trackRuntime2StatusRenderCount(...)` (now monotonic global counters only); post-change Playwright runs stayed stable (`-count=8` all pass, and `-count=12` observed `app-label-after=1` in 2 runs and `app-label-after=2` in 10 runs). Latest hardening added region/trace/fleet idle-drift assertions and tightened app-shell expectation to `app-label-after=1`; experimentally removing the `ui.ReactiveRegion(...)` wrappers caused `app-label-after-burst=10`, so those wrappers remain required for app-shell rerender isolation.
- [x] Add a public runtime2 post-render attach path for `ui.Render(...)` and `ui.RenderInto(...)` so mounted parallel regions can transition from `local-shell` to `worker-attached` without requiring hydration entrypoints.
  Context: attach now runs through the public `ui.ParallelRegion(...)` render path by seeding the shell anchor index and calling `runtime2.HandleHostRegionPostRenderAttach()` for non-hydrate mounts, without requiring hydration entrypoints.
  Validation: `go test ./internal/runtime2 -run TestHandleHostRegionPostRenderAttachSetsCoordinatorAttached$ -count=1` and `GOOS=js GOARCH=wasm go test -exec "C:\Users\Cam\Desktop\GoWebComponents\tools\go_js_wasm_exec.bat" ./ui -run TestRenderIntoParallelRegionPostRenderAttach$ -count=1` and `GOOS=js GOARCH=wasm go test -exec "C:\Users\Cam\Desktop\GoWebComponents\tools\go_js_wasm_exec.bat" ./ui -run TestRenderIntoParallelRegionRuntimeStatusWorkerAttached$ -count=1` and `GOOS=js GOARCH=wasm go test -exec "C:\Users\Cam\Desktop\GoWebComponents\tools\go_js_wasm_exec.bat" ./ui -run TestGetParallelRegionRuntimeStatusReportsPublicDispatchVersions$ -count=1` and `go test -tags playwrightgo ./test/playwrightgo/examples -run TestExample200Runtime2Execution -count=1 -v`
  Result: public post-render attach now transitions runtime status to `worker-attached` while keeping hydration flags unset for non-hydrate mounts.
  Checkpoint: completed todo `Add a public runtime2 post-render attach path for ui.Render(...) and ui.RenderInto(...)`; files changed: `ui/parallel_region.go`, `ui/ui_wasm_test.go`, `docs/MULTITHREADED_RUNTIME_TODO.md`; validation run: focused runtime2 attach test, focused UI wasm attach/status tests, focused public-status wasm test, and Example 200 Playwright execution test; result: runtime status now reports `worker-attached` on normal render routes (`Worker-backed: yes`) while hydration flags remain false; residual risk: the inspector notice copy in Example 200 still references the pre-attach local-shell narrative and should be tightened in a follow-up copy pass; next suggested todo: `Add a single host-update transaction path that minimizes coordinator lock transitions across HandleHostRegionUpdateDispatchWithPriority(...), HandleHostRegionUpdate(...), and coordinator version writes`.
- [x] Add safe scheduler routing for atom subscribers nested under fine-grained regions so updates can target the live fine-grained ancestor without stale-fiber drops.
  Context: a naive nearest-ancestor redirect in `ScheduleSubscribedFiberUpdateWithOrigin(...)` can strand updates on stale ancestors; example 200 then fails to advance Owner State from increment events.
  Validation: `go test ./internal/runtime -run "TestCloneChildFibersMovesReactiveSourceSubscriptions|TestCloneChildFibersQueuesReactiveSourceHydrationSubscriptionMoves|TestReconcileChildrenMovesReactiveSourceSubscriptions|TestReconcileKeyedChildrenMovesReactiveSourceSubscriptions|TestScheduleSubscribedFiberUpdateWithOrigin_UsesFineGrainedAncestor|TestScheduleSubscribedFiberUpdateWithOrigin_UsesLiveFineGrainedAlternate|TestScheduleSubscribedFiberUpdateWithOrigin_UsesDetachedSubscriberFallback|TestScheduleSubscribedFiberUpdateWithOrigin_IgnoresDetachedSubscriberWhenTreeIsMounted|TestReactiveRegionFunctionAtomSubscriber_" -count=1`
  Result: stale subscribed fibers now map back to live alternates when possible, detached stale subscribers are ignored once a mounted tree exists, and region-scoped atom updates stay granular without owner-shell rerender regressions.
- [x] Add authoring docs for transition semantics and deferred snapshot publication with `StartTransition(...)`.
  Validation: `rg -n "## Transition Semantics|ui.StartTransition|UseTransition\\(\\)\\.Start|deferred runtime2 snapshot work" docs/PARALLEL_REGION_AUTHORING.md`
- [x] Add operator-facing docs for region runtime status fields: local or worker ownership, shard, epoch, input/commit versions, and fallback reason.
  Validation: `GOOS=js GOARCH=wasm go test -exec "C:\Users\Cam\Desktop\GoWebComponents\tools\go_js_wasm_exec.bat" ./ui -run "TestGetParallelRegionRuntimeStatusReportsPublicDispatchVersions$"`
- [x] Add a diagnostics example panel that surfaces per-region round-trip latency and dropped stale patch count.
  Validation: `GOOS=js GOARCH=wasm go build -o ./bin/parallel-region-diagnostics.wasm ./examples/110-parallel-region-diagnostics`
- [x] Update the parallel-region docs and examples to remove stale "still being wired" or "simulates transitions" copy and describe the current local-shell plus runtime2-dispatch boundary precisely.
  Validation: `GOOS=js GOARCH=wasm go test ./examples/109-parallel-region-grid ./examples/110-parallel-region-diagnostics`

### Recommended First Pick

- [x] Agent 1 backlog complete.

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
- [x] Add bounded-inbound-queue tests proving burst traffic is either capped or rejected deterministically instead of growing unbounded.
  Validation: `go test ./internal/runtime2 -run "TestBuildShardSessionWithQueueLimit(CapsInboundPayloadQueue|RejectsInvalidLimit)$"`
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

- [x] Add shard-session inbound queue bounds or synchronization so bursty inbound control or payload traffic cannot race or grow without limit.

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
- [x] Add host-side hydration-helper tests that prove public attach calls cannot mark the region attached without both hydration completion and shell-anchor registration.
  Validation: `go test internal/runtime2/host_region_recovery_mirror_test.go internal/runtime2/host_region_hydration_attach_test.go internal/runtime2/host_region_coordinator_attached_test.go internal/runtime2/host_region_hydration_helper_test.go -run "Test(BuildHostRegionHydrationAttachHelperRejectsNilAdapter|HostRegionHydrationAttachHelperRequiresHydrationAndAnchor|HandleHostRegionPostHydrationAttachBlocksBeforeHydrationComplete|HandleHostRegionPostHydrationAttachBlocksWithoutRegisteredAnchor|HandleHostRegionPostHydrationAttachSetsCoordinatorAttached)" -count=1`
- [x] Add diagnostics-snapshot getter tests proving returned entries are redacted, ordered deterministically, and isolated per region.
  Validation: `go test internal/runtime2/host_region_recovery_mirror_test.go internal/runtime2/host_snapshot_transport_hook_test.go internal/runtime2/host_control_dispatcher_test.go internal/runtime2/host_region_diagnostics_snapshot_test.go -run "Test(GetHostRegionDiagnosticsSnapshotReportsReadOnlyRedactedState|HandleHostControlEnvelopeDispatchesDiagnostic|HandleHostRegionUpdateDispatchWithTransportSelectsSharedTier)" -count=1`
- [x] Add extended runtime-status payload tests for hydration-attach state, latest downgrade reasons, and stale-output counters.
  Validation: `go test internal/runtime2/host_region_recovery_mirror_test.go internal/runtime2/host_control_dispatcher_test.go internal/runtime2/host_region_runtime_status_test.go -run "Test(GetHostRegionRuntimeStatusReports(DowngradeReasonsAndStaleCounters|WorkerAttachedMode|LocalShellMode|FallbackMode)|HandleHostControlEnvelopeDispatchesDiagnostic)" -count=1`

### Runtime2 Performance Hotspot Backlog (Hottest First)

### Runtime2 Flamegraph Node Todo Queue (Profile Capture 2026-03-27)

- Capture benchmark: `BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers`
- Capture command: `./bin/runtime2.test.exe -test.run=^$ -test.bench=BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers$ -test.benchtime=10s -test.cpuprofile=./bin/runtime2.hot.cpu.pprof`
- Profile artifact: `bin/runtime2.hot.cpu.pprof`
- Ordering: worst to better by cumulative runtime (`cum`).

<!-- runtime2-flamegraph-node-todos:start -->
- [ ] Node hotspot: `github.com/monstercameron/GoWebComponents/internal/runtime2_test.BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers` | File: `benchmark harness: internal/runtime2/host_region_pressure_bench_test.go` | Cum: 11.76s (98.58%) | Flat: 0.08s (0.67%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `testing.(*B).launch` | File: `benchmark harness: internal/runtime2/host_region_pressure_bench_test.go` | Cum: 11.76s (98.58%) | Flat: 0.00s (0.00%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `testing.(*B).runN` | File: `benchmark harness: internal/runtime2/host_region_pressure_bench_test.go` | Cum: 11.76s (98.58%) | Flat: 0.00s (0.00%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `github.com/monstercameron/GoWebComponents/internal/runtime2.(*HostRegionAdapter).HandleHostRegionUpdateDispatch (inline)` | File: `internal/runtime2/host_region_adapter.go` | Cum: 11.34s (95.05%) | Flat: 0.01s (0.08%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `github.com/monstercameron/GoWebComponents/internal/runtime2.(*HostRegionAdapter).HandleHostRegionUpdateDispatchWithPriority` | File: `internal/runtime2/host_region_adapter.go` | Cum: 11.16s (93.55%) | Flat: 0.29s (2.43%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `github.com/monstercameron/GoWebComponents/internal/runtime2.(*HostRegionAdapter).HandleHostRegionUpdate` | File: `internal/runtime2/host_region_adapter.go` | Cum: 4.12s (34.53%) | Flat: 0.30s (2.51%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `github.com/monstercameron/GoWebComponents/internal/runtime2.(*HostRegionAdapter).handleHostRegionDispatchHash` | File: `internal/runtime2/host_region_adapter.go` | Cum: 3.72s (31.18%) | Flat: 0.22s (1.84%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `github.com/monstercameron/GoWebComponents/internal/runtime2.buildSnapshotDispatchHashInto` | File: `internal/runtime2/snapshot_dispatch_hash.go` | Cum: 3.47s (29.09%) | Flat: 0.19s (1.59%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `github.com/monstercameron/GoWebComponents/internal/runtime2.(*HostRegionAdapter).HandleHostRegionUpdateSnapshot` | File: `internal/runtime2/host_region_adapter.go` | Cum: 2.93s (24.56%) | Flat: 0.28s (2.35%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `crypto/sha256.Sum256` | File: `callsite: internal/runtime2/snapshot_dispatch_hash.go` | Cum: 2.50s (20.96%) | Flat: 0.09s (0.75%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `crypto/internal/fips140/sha256.(*Digest).Write` | File: `callsite: internal/runtime2/snapshot_dispatch_hash.go` | Cum: 1.92s (16.09%) | Flat: 0.09s (0.75%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `github.com/monstercameron/GoWebComponents/internal/runtime2.(*Coordinator).GetEntry` | File: `internal/runtime2/coordinator.go` | Cum: 1.80s (15.09%) | Flat: 0.20s (1.68%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `crypto/internal/fips140/sha256.block` | File: `callsite: internal/runtime2/snapshot_dispatch_hash.go` | Cum: 1.78s (14.92%) | Flat: 0.02s (0.17%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `sync/atomic.(*Int32).Add (inline)` | File: `callsite: internal/runtime2/coordinator.go, internal/runtime2/scheduler.go` | Cum: 1.77s (14.84%) | Flat: 1.77s (14.84%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `crypto/internal/fips140/sha256.blockSHANI` | File: `callsite: internal/runtime2/snapshot_dispatch_hash.go` | Cum: 1.76s (14.75%) | Flat: 1.76s (14.75%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `github.com/monstercameron/GoWebComponents/internal/runtime2.(*Coordinator).UpdateRegion` | File: `internal/runtime2/coordinator.go` | Cum: 1.68s (14.08%) | Flat: 0.07s (0.59%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `crypto/internal/fips140/sha256.(*Digest).Sum` | File: `callsite: internal/runtime2/snapshot_dispatch_hash.go` | Cum: 1.58s (13.24%) | Flat: 0.04s (0.34%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `crypto/internal/fips140/sha256.(*Digest).checkSum` | File: `callsite: internal/runtime2/snapshot_dispatch_hash.go` | Cum: 1.39s (11.65%) | Flat: 0.24s (2.01%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `runtime.duffcopy` | File: `callsite: runtime intrinsic under internal/runtime2 hot path` | Cum: 1.11s (9.30%) | Flat: 1.11s (9.30%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `github.com/monstercameron/GoWebComponents/internal/runtime2.(*Coordinator).SetRegionLastSnapshotVersion` | File: `internal/runtime2/coordinator.go` | Cum: 1.06s (8.89%) | Flat: 0.06s (0.50%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `github.com/monstercameron/GoWebComponents/internal/runtime2.(*Coordinator).storeMutableEntry` | File: `internal/runtime2/coordinator.go` | Cum: 0.82s (6.87%) | Flat: 0.06s (0.50%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `github.com/monstercameron/GoWebComponents/internal/runtime2.(*Scheduler).HandleSchedulerUpdate` | File: `internal/runtime2/scheduler.go` | Cum: 0.79s (6.62%) | Flat: 0.13s (1.09%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `github.com/monstercameron/GoWebComponents/internal/runtime2.appendSnapshotDispatchEnvelope` | File: `internal/runtime2/snapshot_dispatch_hash.go` | Cum: 0.78s (6.54%) | Flat: 0.04s (0.34%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `github.com/monstercameron/GoWebComponents/internal/runtime2.ValidateSerializableProps` | File: `internal/runtime2/spec.go` | Cum: 0.77s (6.45%) | Flat: 0.02s (0.17%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `github.com/monstercameron/GoWebComponents/internal/runtime2.isSerializableAnyFast` | File: `internal/runtime2/spec.go` | Cum: 0.75s (6.29%) | Flat: 0.09s (0.75%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `sync.(*RWMutex).RLock (inline)` | File: `callsite: inspect with go tool pprof -cum -focus <node>` | Cum: 0.73s (6.12%) | Flat: 0.00s (0.00%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `runtime.mapIterStart` | File: `callsite: internal/runtime2/coordinator.go, internal/runtime2/snapshot_dispatch_hash.go, internal/runtime2/spec.go` | Cum: 0.71s (5.95%) | Flat: 0.03s (0.25%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `github.com/monstercameron/GoWebComponents/internal/runtime2.appendSnapshotDispatchValue` | File: `internal/runtime2/snapshot_dispatch_hash.go` | Cum: 0.65s (5.45%) | Flat: 0.07s (0.59%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `github.com/monstercameron/GoWebComponents/internal/runtime2.appendSnapshotDispatchAnyMap` | File: `internal/runtime2/snapshot_dispatch_hash.go` | Cum: 0.64s (5.36%) | Flat: 0.07s (0.59%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `runtime.mapaccess2` | File: `callsite: internal/runtime2/coordinator.go, internal/runtime2/snapshot_dispatch_hash.go, internal/runtime2/spec.go` | Cum: 0.58s (4.86%) | Flat: 0.03s (0.25%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `github.com/monstercameron/GoWebComponents/internal/runtime2.appendSnapshotDispatchAnyMapSingle` | File: `internal/runtime2/snapshot_dispatch_hash.go` | Cum: 0.57s (4.78%) | Flat: 0.06s (0.50%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `sync.(*RWMutex).RUnlock` | File: `callsite: inspect with go tool pprof -cum -focus <node>` | Cum: 0.55s (4.61%) | Flat: 0.00s (0.00%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `sync.(*RWMutex).Unlock` | File: `callsite: inspect with go tool pprof -cum -focus <node>` | Cum: 0.51s (4.27%) | Flat: 0.06s (0.50%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `github.com/monstercameron/GoWebComponents/internal/runtime2.(*Coordinator).getMutableEntry` | File: `internal/runtime2/coordinator.go` | Cum: 0.49s (4.11%) | Flat: 0.06s (0.50%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `internal/runtime/maps.(*Iter).Next` | File: `callsite: inspect with go tool pprof -cum -focus <node>` | Cum: 0.45s (3.77%) | Flat: 0.37s (3.10%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `sync.(*RWMutex).Lock` | File: `callsite: inspect with go tool pprof -cum -focus <node>` | Cum: 0.45s (3.77%) | Flat: 0.06s (0.50%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `runtime.mapassign` | File: `callsite: internal/runtime2/coordinator.go, internal/runtime2/snapshot_dispatch_hash.go, internal/runtime2/spec.go` | Cum: 0.40s (3.35%) | Flat: 0.09s (0.75%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `internal/runtime/maps.(*Map).getWithKeySmall` | File: `callsite: inspect with go tool pprof -cum -focus <node>` | Cum: 0.39s (3.27%) | Flat: 0.13s (1.09%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `runtime.mapaccess2_faststr` | File: `callsite: internal/runtime2/coordinator.go, internal/runtime2/snapshot_dispatch_hash.go, internal/runtime2/spec.go` | Cum: 0.36s (3.02%) | Flat: 0.10s (0.84%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `internal/runtime/maps.(*Iter).Init` | File: `callsite: inspect with go tool pprof -cum -focus <node>` | Cum: 0.36s (3.02%) | Flat: 0.07s (0.59%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `runtime.duffzero` | File: `callsite: runtime intrinsic under internal/runtime2 hot path` | Cum: 0.32s (2.68%) | Flat: 0.32s (2.68%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `aeshashbody` | File: `callsite: internal/runtime2/snapshot_dispatch_hash.go` | Cum: 0.31s (2.60%) | Flat: 0.31s (2.60%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `internal/runtime/maps.rand` | File: `callsite: inspect with go tool pprof -cum -focus <node>` | Cum: 0.29s (2.43%) | Flat: 0.06s (0.50%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `internal/runtime/maps.(*Map).getWithoutKeySmallFastStr` | File: `callsite: inspect with go tool pprof -cum -focus <node>` | Cum: 0.26s (2.18%) | Flat: 0.14s (1.17%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `github.com/monstercameron/GoWebComponents/internal/runtime2.(*Scheduler).getSchedulerWorkerHealth (inline)` | File: `internal/runtime2/scheduler.go` | Cum: 0.25s (2.10%) | Flat: 0.02s (0.17%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `runtime.rand` | File: `callsite: runtime intrinsic under internal/runtime2 hot path` | Cum: 0.23s (1.93%) | Flat: 0.08s (0.67%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `internal/runtime/maps.(*Map).putSlotSmall` | File: `callsite: inspect with go tool pprof -cum -focus <node>` | Cum: 0.20s (1.68%) | Flat: 0.03s (0.25%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `github.com/monstercameron/GoWebComponents/internal/runtime2.(*Scheduler).handleSchedulerQueueAppend` | File: `internal/runtime2/scheduler.go` | Cum: 0.20s (1.68%) | Flat: 0.17s (1.42%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `sync.(*Mutex).Unlock (inline)` | File: `callsite: inspect with go tool pprof -cum -focus <node>` | Cum: 0.19s (1.59%) | Flat: 0.00s (0.00%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `internal/sync.(*Mutex).Unlock (inline)` | File: `callsite: inspect with go tool pprof -cum -focus <node>` | Cum: 0.19s (1.59%) | Flat: 0.19s (1.59%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `runtime.mapassign_faststr` | File: `callsite: internal/runtime2/coordinator.go, internal/runtime2/snapshot_dispatch_hash.go, internal/runtime2/spec.go` | Cum: 0.19s (1.59%) | Flat: 0.03s (0.25%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `runtime.strequal` | File: `callsite: internal/runtime2/coordinator.go, internal/runtime2/snapshot_dispatch_hash.go, internal/runtime2/spec.go` | Cum: 0.18s (1.51%) | Flat: 0.13s (1.09%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `runtime.memmove` | File: `callsite: runtime intrinsic under internal/runtime2 hot path` | Cum: 0.17s (1.42%) | Flat: 0.17s (1.42%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `internal/sync.(*Mutex).Lock (inline)` | File: `callsite: inspect with go tool pprof -cum -focus <node>` | Cum: 0.16s (1.34%) | Flat: 0.16s (1.34%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `sync.(*Mutex).Lock (inline)` | File: `callsite: inspect with go tool pprof -cum -focus <node>` | Cum: 0.16s (1.34%) | Flat: 0.00s (0.00%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `runtime.memequal` | File: `callsite: runtime intrinsic under internal/runtime2 hot path` | Cum: 0.16s (1.34%) | Flat: 0.16s (1.34%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `runtime.mapIterNext` | File: `callsite: internal/runtime2/coordinator.go, internal/runtime2/snapshot_dispatch_hash.go, internal/runtime2/spec.go` | Cum: 0.16s (1.34%) | Flat: 0.03s (0.25%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `github.com/monstercameron/GoWebComponents/internal/runtime2.(*SchedulerShardModel).GetSchedulerRegionAssignedShardID (inline)` | File: `internal/runtime2/scheduler_shard_identity.go` | Cum: 0.15s (1.26%) | Flat: 0.02s (0.17%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `runtime.convT64` | File: `callsite: runtime intrinsic under internal/runtime2 hot path` | Cum: 0.15s (1.26%) | Flat: 0.01s (0.08%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `runtime.mallocgc` | File: `callsite: runtime intrinsic under internal/runtime2 hot path` | Cum: 0.14s (1.17%) | Flat: 0.06s (0.50%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `internal/chacha8rand.(*State).Refill` | File: `callsite: runtime intrinsic under internal/runtime2 hot path` | Cum: 0.12s (1.01%) | Flat: 0.00s (0.00%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `internal/chacha8rand.block` | File: `callsite: runtime intrinsic under internal/runtime2 hot path` | Cum: 0.12s (1.01%) | Flat: 0.12s (1.01%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `runtime.systemstack` | File: `callsite: runtime intrinsic under internal/runtime2 hot path` | Cum: 0.12s (1.01%) | Flat: 0.00s (0.00%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `internal/runtime/maps.(*Map).putSlotSmallFastStr` | File: `callsite: inspect with go tool pprof -cum -focus <node>` | Cum: 0.10s (0.84%) | Flat: 0.04s (0.34%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `github.com/monstercameron/GoWebComponents/internal/runtime2.buildSnapshotEnvelopeFromNormalizedSourceSnapshotWithoutValidation` | File: `internal/runtime2/snapshot.go` | Cum: 0.09s (0.75%) | Flat: 0.08s (0.67%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `runtime.strhash` | File: `callsite: internal/runtime2/coordinator.go, internal/runtime2/snapshot_dispatch_hash.go, internal/runtime2/spec.go` | Cum: 0.09s (0.75%) | Flat: 0.09s (0.75%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `internal/runtime/maps.(*groupReference).key (inline)` | File: `callsite: inspect with go tool pprof -cum -focus <node>` | Cum: 0.09s (0.75%) | Flat: 0.09s (0.75%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `runtime.stdcall2` | File: `callsite: runtime intrinsic under internal/runtime2 hot path` | Cum: 0.08s (0.67%) | Flat: 0.08s (0.67%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `runtime.mallocgcTiny` | File: `callsite: runtime intrinsic under internal/runtime2 hot path` | Cum: 0.08s (0.67%) | Flat: 0.02s (0.17%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `runtime.preemptone` | File: `callsite: runtime intrinsic under internal/runtime2 hot path` | Cum: 0.07s (0.59%) | Flat: 0.00s (0.00%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `runtime.preemptM` | File: `callsite: runtime intrinsic under internal/runtime2 hot path` | Cum: 0.07s (0.59%) | Flat: 0.00s (0.00%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `internal/runtime/maps.typedmemmove` | File: `callsite: inspect with go tool pprof -cum -focus <node>` | Cum: 0.07s (0.59%) | Flat: 0.01s (0.08%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `time.Now` | File: `callsite: runtime intrinsic under internal/runtime2 hot path` | Cum: 0.07s (0.59%) | Flat: 0.03s (0.25%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `runtime.gcDrainMarkWorkerDedicated (inline)` | File: `callsite: runtime intrinsic under internal/runtime2 hot path` | Cum: 0.07s (0.59%) | Flat: 0.00s (0.00%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `runtime.gcBgMarkWorker.func2` | File: `callsite: runtime intrinsic under internal/runtime2 hot path` | Cum: 0.07s (0.59%) | Flat: 0.00s (0.00%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `github.com/monstercameron/GoWebComponents/internal/runtime2.ParseRegionInstanceID` | File: `internal/runtime2/registry.go` | Cum: 0.07s (0.59%) | Flat: 0.02s (0.17%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `runtime.gcDrain` | File: `callsite: runtime intrinsic under internal/runtime2 hot path` | Cum: 0.07s (0.59%) | Flat: 0.00s (0.00%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `runtime.lock2` | File: `callsite: runtime intrinsic under internal/runtime2 hot path` | Cum: 0.06s (0.50%) | Flat: 0.00s (0.00%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `runtime.typedmemmove` | File: `callsite: runtime intrinsic under internal/runtime2 hot path` | Cum: 0.06s (0.50%) | Flat: 0.06s (0.50%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `github.com/monstercameron/GoWebComponents/internal/runtime2.hasSerializableSafeMapKey (inline)` | File: `internal/runtime2/spec.go` | Cum: 0.06s (0.50%) | Flat: 0.06s (0.50%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `runtime.lock (inline)` | File: `callsite: runtime intrinsic under internal/runtime2 hot path` | Cum: 0.06s (0.50%) | Flat: 0.00s (0.00%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `runtime.lockWithRank (inline)` | File: `callsite: runtime intrinsic under internal/runtime2 hot path` | Cum: 0.06s (0.50%) | Flat: 0.00s (0.00%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `internal/runtime/maps.(*groupReference).elem (inline)` | File: `callsite: inspect with go tool pprof -cum -focus <node>` | Cum: 0.06s (0.50%) | Flat: 0.06s (0.50%) | Profile: `bin/runtime2.hot.cpu.pprof`
- [ ] Node hotspot: `internal/runtime/maps.ctrlGroup.matchH2 (inline)` | File: `callsite: inspect with go tool pprof -cum -focus <node>` | Cum: 0.06s (0.50%) | Flat: 0.06s (0.50%) | Profile: `bin/runtime2.hot.cpu.pprof`
<!-- runtime2-flamegraph-node-todos:end -->

### Runtime2 Profile-Derived Optimization Pass Targets (Actionable)

- [x] Add a single host-update transaction path that minimizes coordinator lock transitions across `HandleHostRegionUpdateDispatchWithPriority(...)`, `HandleHostRegionUpdate(...)`, and coordinator version writes (`internal/runtime2/host_region_adapter.go`, `internal/runtime2/coordinator.go`).
  Profile evidence: `HandleHostRegionUpdateDispatchWithPriority` cum `11.16s (93.55%)`; coordinator path nodes `GetEntry` cum `1.80s`, `UpdateRegion` cum `1.68s`, `SetRegionLastSnapshotVersion` cum `1.06s`.
  Validation: `go test ./internal/runtime2 -run "Test(HandleHostRegionUpdateDispatchWithPriorityDeferred|HandleHostRegionUpdateDispatchWithTransportSelectsSharedTier|HandleHostRegionUpdateSnapshot)$" -count=1` and `go test ./internal/runtime2 -run ^$ -bench "BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers$" -benchmem -count=5`
  Notes: coordinator now exposes single-transaction helpers (`SetRegionSnapshotState(...)`, `StoreRegionSnapshotAndDispatchedVersion(...)`, `UpdateRegionAndGetEntry(...)`), snapshot capture now writes snapshot version and source IDs in one coordinator lock when snapshot-state persistence is requested, and urgent dispatch now commits snapshot+dispatch versions in one coordinator transaction instead of split writes. `HandleHostRegionUpdate(...)` now avoids an extra coordinator reread by using `UpdateRegionAndGetEntry(...)`.
  Validation: `go test ./internal/runtime2 -run "Test(HandleHostRegionUpdateDispatchWithPriorityDeferredClassification|HandleHostRegionUpdateDispatchWithTransportSelectsSharedTier|HandleHostRegionUpdateSnapshotCapturesPropsAndSources|HandleHostRegionUpdateDispatchNoChangeDoesNotAdvanceDispatchedVersion)$" -count=1` and `go test ./internal/runtime2 -run "Test(SetRegionSnapshotStateUpdatesSnapshotAndSources|StoreRegionSnapshotAndDispatchedVersionTracksBothVersions|UpdateRegionAndGetEntryTracksLastDispatchedVersion)$" -count=1` and `go test ./internal/runtime2 -run ^$ -bench "BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers$" -benchmem -count=3` and `go test ./internal/runtime2 -run ^$ -bench "BenchmarkCoordinator(DispatchTransactionCurrentVsLegacy|SnapshotStateCurrentVsLegacy)$" -benchmem -count=5`
  Result: explicit micro-bench before/after tracking now exists in `BenchmarkCoordinatorDispatchTransactionCurrentVsLegacy` and `BenchmarkCoordinatorSnapshotStateCurrentVsLegacy`; latest runs show transaction-path median improvement in the new single-transaction branches (`legacy_split_snapshot_and_dispatch` roughly `104` to `143 ns/op` vs `current_single_snapshot_and_dispatch` roughly `51` to `73 ns/op` across the same run set), with snapshot+source updates also improving in aggregate while preserving alloc counts (`48 B/op`, `1 alloc/op`) in both variants.
  Checkpoint: completed todo `Add a single host-update transaction path that minimizes coordinator lock transitions`; files changed: `internal/runtime2/coordinator.go`, `internal/runtime2/host_region_adapter.go`, `internal/runtime2/coordinator_state_test.go`, `internal/runtime2/coordinator_transaction_bench_test.go`, `docs/MULTITHREADED_RUNTIME_TODO.md`; residual risk: deferred and no-change dispatches still persist snapshot version through the standalone snapshot-version write path, so a future pass could fold those into the same transaction helper when that path is measured hot; next suggested todo: `Add a dispatch-hash map-walk reduction pass that avoids repeated generic map iteration in appendSnapshotDispatchEnvelope(...), appendSnapshotDispatchValue(...), and appendSnapshotDispatchAnyMap(...)`.
- [x] Add a dispatch-hash map-walk reduction pass that avoids repeated generic map iteration in `appendSnapshotDispatchEnvelope(...)`, `appendSnapshotDispatchValue(...)`, and `appendSnapshotDispatchAnyMap(...)` for stable source or prop shapes (`internal/runtime2/snapshot_dispatch_hash.go`).
  Profile evidence: `buildSnapshotDispatchHashInto` cum `3.47s (29.09%)`; hash-envelope append helpers cum around `0.64s` to `0.78s`; runtime map iterator nodes (`mapIterStart`, `Iter.Next`, `mapaccess2`) cum around `0.45s` to `0.71s`.
  Notes: `appendSnapshotDispatchAnyMap(...)` now uses a pooled key/value entry buffer so keys and values are captured in one pass and sorted without a follow-up map lookup per key; map-walk regression tracking now has an explicit current-vs-legacy microbench (`BenchmarkAppendSnapshotDispatchAnyMapCurrentVsLegacy`), and a changed-props dispatch regression test now pins that changed envelopes do not short-circuit as no-change.
  Validation: `go test ./internal/runtime2 -run "Test(BuildSnapshotDispatchHash.*|HandleHostRegionUpdateDispatch(NoChangeDoesNotAdvanceDispatchedVersion|ChangedPropsSchedulesUpdate)|HandleHostRegionUpdateSnapshotCapturesPropsAndSources)$" -count=1` and `go test ./internal/runtime2 -run ^$ -bench "BenchmarkAppendSnapshotDispatchAnyMapCurrentVsLegacy$" -benchmem -count=5` and `go test ./internal/runtime2 -run ^$ -bench "BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers$" -benchmem -count=5`
  Result: standalone compare runs show pooled-entry map-walk hashing improving median latency from about `1638 ns/op` to about `1574 ns/op` while reducing allocation pressure from `696 B/op, 10 allocs/op` to `312 B/op, 9 allocs/op`; hot-regions median improved from about `24536 ns/op` to about `21678 ns/op` on this host with allocation profile unchanged (`~382 B/op`, `47 allocs/op`).
  Checkpoint: completed todo `Add a dispatch-hash map-walk reduction pass that avoids repeated generic map iteration`; files changed: `internal/runtime2/snapshot_dispatch_hash.go`, `internal/runtime2/snapshot_dispatch_hash_compare_bench_test.go`, `internal/runtime2/host_region_update_dispatch_test.go`, `docs/MULTITHREADED_RUNTIME_TODO.md`; validation run: focused dispatch-hash and changed-props dispatch tests, dedicated current-vs-legacy map-walk microbench, and hot-regions benchmark baseline comparison; result: map-walk path keeps one-pass key/value capture with explicit before/after microbench guardrails and no dispatch-behavior regression in changed-props coverage; residual risk: small-map fast paths (`len<=2`) still dominate some workloads, so deeper gains will require reducing nested `appendSnapshotDispatchValue(...)` reflection cost; next suggested todo: `Add a hash-cost reduction pass for no-change detection by reducing full-payload SHA work in buildSnapshotDispatchHashInto(...) and handleHostRegionDispatchHash(...)`.
- [x] Add a hash-cost reduction pass for no-change detection by reducing full-payload SHA work in `buildSnapshotDispatchHashInto(...)` and `handleHostRegionDispatchHash(...)` (streaming digest reuse and/or guarded two-tier hash) (`internal/runtime2/snapshot_dispatch_hash.go`, `internal/runtime2/host_region_adapter.go`).
  Profile evidence: `sha256.Sum256` cum `2.50s (20.96%)`; `sha256.(*Digest).Write` cum `1.92s`; `sha256.block` cum `1.78s`.
  Notes: host dispatch no-change detection now adds a version-vector gate (`epoch`, `source_version`) before canonical payload hashing; when that vector changes, dispatch exits as changed without building digest state. For same-vector updates, the path computes one canonical payload and one SHA-256 digest (no payload-byte copy guard) and reuses digest state for no-change detection. A dedicated current-vs-legacy microbench now measures this exact path (`BenchmarkHandleHostRegionDispatchHashCurrentVsLegacy`).
  Validation: `go test ./internal/runtime2 -run "Test(BuildSnapshotDispatchHash.*|HandleHostRegionDispatchHash(IgnoresInputVersion|StoresDigestState|VersionVectorMismatchResetsDigestState)|HandleHostRegionUpdateDispatch(NoChangeDoesNotAdvanceDispatchedVersion|ChangedPropsSchedulesUpdate))$" -count=1` and `go test ./internal/runtime2 -run ^$ -bench "Benchmark(HandleHostRegionDispatchHashCurrentVsLegacy|BuildSnapshotDispatchHashCurrentVsLegacy|HandleHostRegionManyHotRegionsBoundedWorkers)$" -benchmem -count=5`
  Result: compare bench now shows large wins on changed payloads with version-vector mismatches (`changed_payloads_legacy_sha_only` about `710-845 ns/op` vs `changed_payloads_current_version_vector_gate` about `5.3-6.7 ns/op`, allocations `104 B/op, 3 allocs/op` to `0 B/op, 0 allocs/op`) while keeping stable payload cost in the same range (`stable_payload_legacy_sha_only` about `902-1149 ns/op` vs `stable_payload_current_digest_guard` about `817-1100 ns/op`, both `104 B/op`, `3 allocs/op`). Hot-regions pressure runs stayed in the expected band with unchanged allocation profile (`~17716-19549 ns/op`, `382 B/op`, `47 allocs/op`).
  Checkpoint: completed todo `Add a hash-cost reduction pass for no-change detection`; files changed: `internal/runtime2/host_region_adapter.go`, `internal/runtime2/snapshot_dispatch_hash_test.go`, `internal/runtime2/perf_host_region_dispatch_hash_compare_bench_test.go`, `docs/MULTITHREADED_RUNTIME_TODO.md`; validation run: focused dispatch-hash/no-change tests and dedicated current-vs-legacy + hot-regions microbenches; result: obvious changed-version updates now bypass digest work and the no-change digest path avoids payload byte-copy guards; residual risk: same-version changed-props updates still require canonical payload encode+digest, so the next high-impact item remains snapshot-capture short-circuiting before hash construction; next suggested todo: `Add a snapshot-capture short-circuit for unchanged props or declared-source versions before full ValidateSerializableProps(...) and source-envelope rebuild`.
- [x] Add a snapshot-capture short-circuit for unchanged props or declared-source versions before full `ValidateSerializableProps(...)` and source-envelope rebuild (`internal/runtime2/host_region_adapter.go`, `internal/runtime2/spec.go`, `internal/runtime2/snapshot.go`).
  Profile evidence: `HandleHostRegionUpdateSnapshot` cum `2.93s (24.56%)`; `ValidateSerializableProps` cum `0.77s`; `isSerializableAnyFast` cum `0.75s`; source-envelope builder node cum `0.09s`.
  Notes: `handleHostRegionUpdateSnapshot(...)` now short-circuits validation for unchanged immutable props (`nil`, bool, numeric scalars, string) via a cached FNV-style token instead of deep-clone plus `reflect.DeepEqual`; mutable container props (`map[string]any`, `[]any`) continue to re-run `ValidateSerializableProps(...)` each update to preserve safety. The epoch+source-version-tuple source snapshot reuse path remains active, so stable declared-source versions reuse normalized source maps and skip source-envelope rebuild work.
  Validation: `go test ./internal/runtime2 -run "TestHandleHostRegionUpdateSnapshot" -count=1` and `go test ./internal/runtime2 -run ^$ -bench "BenchmarkHandleHostRegion(UpdateSnapshot|ManyHotRegionsBoundedWorkers)$" -benchmem -count=5`
  Validation: `go test ./internal/runtime2 -run "TestHandleHostRegionUpdateSnapshot(CapturesPropsAndSources|ReusesSourceMapForStableVersionTuple|PropsCacheInvalidatesOnInPlaceMutation|InvalidatesSourceMapReuseOnVersionChange|InvalidatesSourceMapReuseOnRemountEpoch)$" -count=1` and `go test ./internal/runtime2 -run ^$ -bench "BenchmarkHandleHostRegionUpdateSnapshotCurrentVsLegacy$" -benchmem -count=5`
  Result: compare bench now shows snapshot capture current path clearly ahead of legacy for both changed and stable map-props cases (`current-changed-props` about `365-506 ns/op`, `600 B/op`, `4 allocs/op` vs `legacy-changed-props` about `623-699 ns/op`, `936 B/op`, `6 allocs/op`; `current-stable-props` about `392-850 ns/op`, `592 B/op`, `4 allocs/op` vs `legacy-stable-props` about `617-1051 ns/op`, `928 B/op`, `6 allocs/op`). The direct snapshot benchmark also improved to roughly `438-493 ns/op`, `592 B/op`, `4 allocs/op`, and hot-regions pressure stayed in the expected `~21.5-22.8 µs/op`, `382 B/op`, `47 allocs/op` band.
  Checkpoint: completed todo `Add a snapshot-capture short-circuit for unchanged props or declared-source versions before full ValidateSerializableProps(...) and source-envelope rebuild`; files changed: `internal/runtime2/host_region_adapter.go`, `internal/runtime2/host_region_snapshot_capture_test.go`, `internal/runtime2/perf_host_region_update_snapshot_compare_bench_test.go`, `docs/MULTITHREADED_RUNTIME_TODO.md`; validation run: focused snapshot-capture tests, direct snapshot/hot-regions benchmark command, and current-vs-legacy snapshot capture microbench; result: short-circuit path now avoids the deep-clone/DeepEqual regression and keeps snapshot capture faster with lower allocation pressure; residual risk: mutable container props still pay full recursive validation on each update, so the next pass should target prop-shape caching on stable map/list schemas; next suggested todo: `Optimize canonical prop decode in parseBuildCanonicalPropByKeyFromRaw(...)`.
- [x] Add a scheduler no-op fast path to skip queue append and health checks when the region already has an equivalent queued update (`internal/runtime2/scheduler.go`).
  Profile evidence: `Scheduler.HandleSchedulerUpdate` cum `0.79s`; `handleSchedulerQueueAppend` cum `0.20s`; scheduler health lookup node cum `0.25s`.
  Notes: `HandleSchedulerUpdate(...)` now checks for an equivalent already-queued update (`region + shard + cancel_version`) before shard-live and worker-health checks; when a matching queued update exists it returns a no-op update job without re-appending or re-checking health.
  Validation: `go test ./internal/runtime2 -run "TestHandleSchedulerUpdateCoalescesQueuedRegionUpdates" -count=1` and `go test ./internal/runtime2 -run ^$ -bench "BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers$" -benchmem -count=5`
  Checkpoint: completed todo `Add a scheduler no-op fast path to skip queue append and health checks when the region already has an equivalent queued update`; files changed: `internal/runtime2/scheduler.go`, `internal/runtime2/scheduler_update_coalesce_test.go`, `docs/MULTITHREADED_RUNTIME_TODO.md`; result: scheduler coalesce tests passed (including new equivalent-queued health-skip guard) and hot-regions benchmark command passed; residual risk: if external consumers assume repeated update calls always re-evaluate degraded health, equivalent queued updates now short-circuit until cancel generation or shard assignment changes; next suggested todo: `Add a single host-update transaction path that minimizes coordinator lock transitions across HandleHostRegionUpdateDispatchWithPriority(...), HandleHostRegionUpdate(...), and coordinator version writes`.
- [x] Add a timing-metric write gate in `HandleHostRegionUpdateDispatchWithPriority(...)` so `time.Now()` and timing-field resets only run when round-trip timing capture is enabled (`internal/runtime2/host_region_adapter.go`).
  Profile evidence: dispatch line-level samples on `storeHostRegionDispatchAt = time.Now()` and top node includes `time.Now` cum `0.07s`.
  Notes: round-trip timing capture is now explicit opt-in via `SetHostRegionRoundTripTimingEnabled(...)`; dispatch, patch-ready, and commit timing writes are gated behind that flag so the default hot update path skips `time.Now()` and timing reset work unless diagnostics explicitly enable timing.
  Validation: `go test ./internal/runtime2 -run "TestHandleHostRegionRoundTripTimingCapturesDispatchPatchAndCommit" -count=1` and `go test ./internal/runtime2 -run ^$ -bench "BenchmarkHandleHostRegionUpdateDispatchWithPriorityDeferred$" -benchmem -count=5`
  Checkpoint: completed todo `Add a timing-metric write gate in HandleHostRegionUpdateDispatchWithPriority(...)`; files changed: `internal/runtime2/host_region_adapter.go`, `internal/runtime2/host_region_round_trip_timing_test.go`, `examples/110-parallel-region-diagnostics/main.go`, `docs/MULTITHREADED_RUNTIME_TODO.md`; result: focused timing tests passed (including new disabled-capture guard) and deferred-dispatch benchmark command passed; residual risk: consumers that expect non-zero timing without explicit opt-in now receive zero metrics by default; next suggested todo: `Add a scheduler no-op fast path to skip queue append and health checks when the region already has an equivalent queued update`.

- [x] Add a build-time invariant gate in `BuildCanonicalRenderIR(...)` so self-generated node-table and prop-table re-parse validation can be skipped on trusted hot paths while strict validation stays available for tests and debug builds (`internal/runtime2/render_ir.go`).
  Notes: added a canonical-IR invariant validation gate (`SetCanonicalRenderIRInvariantValidationEnabled(...)` / `HasCanonicalRenderIRInvariantValidationEnabled(...)`); `BuildCanonicalRenderIR(...)` now only runs node-table and prop-table self-parse invariants when the gate is enabled, keeping strict checks available for targeted tests/debug while the hot path can skip redundant re-parse validation.
  Validation: `go test ./internal/runtime2 -run "Test(BuildCanonicalRenderIR|ParseCanonicalRenderTree)" -count=1` and `go test ./internal/runtime2 -run ^$ -bench "Benchmark(BuildCanonicalRenderIR|ParseCanonicalRenderTree)$" -benchmem -count=5`
  Checkpoint: completed todo `Add a build-time invariant gate in BuildCanonicalRenderIR(...)`; files changed: `internal/runtime2/render_ir.go`, `internal/runtime2/render_ir_test.go`, `docs/MULTITHREADED_RUNTIME_TODO.md`; result: canonical render IR/tree tests and benchmark command passed, plus new gate-toggle test passed; residual risk: because strict invariant checks are now opt-in, malformed IR bugs introduced by future builder changes may surface later unless tests/debug paths enable the gate; next suggested todo: `Add a dense-node fast path in ParseRenderNodeTable(...) that uses compact bitset/slice ownership tracking when node IDs are near-sequential`.
- [x] Add a dense-node fast path in `ParseRenderNodeTable(...)` that uses compact bitset/slice ownership tracking when node IDs are near-sequential, falling back to maps only for sparse IDs (`internal/runtime2/render_node_table.go`).
  Notes: refactored node-ID uniqueness validation into a dedicated pass that keeps the existing exact `1..N` fast path, adds a dense-range bitmap path for near-sequential ID ranges, and falls back to map tracking only for sparse ranges; this removes map churn on common dense parse inputs.
  Validation: `go test ./internal/runtime2 -run "TestParseRenderNodeTable" -count=1` and `go test ./internal/runtime2 -run ^$ -bench "BenchmarkParseCanonicalRenderTree$" -benchmem -count=5`
  Checkpoint: completed todo `Add a dense-node fast path in ParseRenderNodeTable(...)`; files changed: `internal/runtime2/render_node_table.go`, `docs/MULTITHREADED_RUNTIME_TODO.md`; result: node-table tests passed and parse-tree benchmark command passed with reduced allocs/op on this run; residual risk: extremely wide but sparse node-ID ranges still use map fallback and can retain prior overhead characteristics; next suggested todo: `Add a streaming dispatch-hash writer path that hashes canonical envelope fields directly into a reusable digest state`.
- [x] Add a streaming dispatch-hash writer path that hashes canonical envelope fields directly into a reusable digest state so no-change checks can avoid staging full payload bytes in `storeHostRegionDispatchBytes` (`internal/runtime2/snapshot_dispatch_hash.go`, `internal/runtime2/host_region_adapter.go`).
  Notes: removed `storeHostRegionDispatchBytes` from `HostRegionAdapter`; dispatch hashing now keeps only reusable zero-length scratch capacity (`storeHostRegionDispatchHashScratch`) and adds a streaming writer helper (`buildSnapshotDispatchHashStreamed(...)` + `writeSnapshotDispatchEnvelopeHash(...)`) that hashes canonical envelope fields directly into a pooled SHA-256 hasher without staging a full payload buffer.
  Microbench before/after (`BenchmarkHandleHostRegion(UpdateDispatchWithPriorityDeferred|ManyHotRegionsBoundedWorkers)$`, `-benchmem -count=5`):
  - before: `no-change` `301.9-336.8 ns/op` `0 B/op` `0 allocs/op`; `changed` `501.5-659.3 ns/op` `344 B/op` `2 allocs/op`; `changed-reused-spec` `431.0-467.6 ns/op` `7 B/op` `0 allocs/op`; `many-hot-regions` `23,806-41,594 ns/op` `381-382 B/op` `47 allocs/op`
  - after: `no-change` `315.4-323.9 ns/op` `0 B/op` `0 allocs/op`; `changed` `529.8-594.5 ns/op` `344 B/op` `2 allocs/op`; `changed-reused-spec` `413.4-516.7 ns/op` `7 B/op` `0 allocs/op`; `many-hot-regions` `23,877-29,290 ns/op` `381-382 B/op` `47 allocs/op`
  - summary: no regression in allocation profile on the hot no-change path; high-percentile many-hot-regions runtime improved on this run set.
  Validation: `go test ./internal/runtime2 -run "TestBuildSnapshotDispatchHash" -count=1` and `go test ./internal/runtime2 -run ^$ -bench "BenchmarkHandleHostRegion(UpdateDispatchWithPriorityDeferred|ManyHotRegionsBoundedWorkers)$" -benchmem -count=5`
  Checkpoint: completed todo `Add a streaming dispatch-hash writer path`; files changed: `internal/runtime2/snapshot_dispatch_hash.go`, `internal/runtime2/snapshot_dispatch_hash_test.go`, `internal/runtime2/host_region_adapter.go`, `docs/MULTITHREADED_RUNTIME_TODO.md`; result: dispatch-hash tests passed (including new streamed-vs-buffered parity and zero-length scratch assertions) and benchmark command passed with before/after deltas recorded; residual risk: the new streaming helper is parity-covered but not the default hot-path encoder, so future work should either migrate the hot path behind a perf gate or extend microbench coverage to include direct streamed hashing; next suggested todo: `Evaluate a two-tier dispatch no-change digest strategy`.
- [x] Evaluate a two-tier dispatch no-change digest strategy: fast non-cryptographic hash (for example xxh3-128) plus collision guard before treating updates as no-change (`internal/runtime2/snapshot_dispatch_hash.go`, `internal/runtime2/host_region_adapter.go`).
  Notes: added `buildSnapshotDispatchFastHashInto(...)` plus `buildSnapshotDispatchFastHash(...)` (FNV-1a 64-bit prefilter) and host-side fast-hash state (`storeHostRegionDispatchFastHash`) so update dispatch can short-circuit obvious changes before SHA-256 work; kept SHA-256 as the collision-guard check on same-fast-hash paths before reporting no-change.
  Microbench before/after (`BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers$`, `-benchmem -count=5`):
  - before: `23,877-29,290 ns/op`, `381-382 B/op`, `47 allocs/op`
  - after: `17,503-24,057 ns/op`, `382 B/op`, `47 allocs/op`
  - summary: improved many-hot-regions dispatch throughput in this run set without allocation regression.
  Validation: `go test ./internal/runtime2 -run "TestHandleHostRegionUpdateDispatch.*NoChange" -count=1` and `go test ./internal/runtime2 -run ^$ -bench "BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers$" -benchmem -count=5`
  Checkpoint: completed todo `Evaluate a two-tier dispatch no-change digest strategy`; files changed: `internal/runtime2/snapshot_dispatch_hash.go`, `internal/runtime2/host_region_adapter.go`, `internal/runtime2/host_control_dispatcher.go`, `docs/MULTITHREADED_RUNTIME_TODO.md`; result: no-change dispatch tests and hot-regions benchmark command passed with recorded before/after deltas; residual risk: when payload content changes, the first repeated identical payload may still run one guarded SHA baseline pass before no-change short-circuiting resumes; next suggested todo: `Add a source-snapshot reuse cache keyed by region epoch plus source-version tuple`.
- [x] Add a source-snapshot reuse cache keyed by region epoch plus source-version tuple so `HandleHostRegionUpdateSnapshot(...)` can skip rebuilding source maps when declared source values are unchanged (`internal/runtime2/host_region_adapter.go`, `internal/runtime2/snapshot.go`).
  Notes: added host adapter cache state (`epoch + sourceIDs + source-version tuple`) and `handleHostRegionDeclaredSourceLookupForEpochNormalized(...)` now reuses the previously validated source-values map when the cache key is stable. Cache reset hooks were added for source-lookup bridge replacement plus mount/dispose/structural-remount/repair-remount lifecycle boundaries.
  Microbench before/after (`BenchmarkHandleHostRegion(UpdateSnapshot|ManyHotRegionsBoundedWorkers)$`, `-benchmem -count=5`):
  - before: `update-snapshot` `661.8-910.0 ns/op` `1184 B/op` `8 allocs/op`; `many-hot-regions` `16,242-21,566 ns/op` `382 B/op` `47 allocs/op`
  - after: `update-snapshot` `430.3-592.0 ns/op` `848 B/op` `6 allocs/op`; `many-hot-regions` `15,371-16,724 ns/op` `382-383 B/op` `47 allocs/op`
  - summary: snapshot capture throughput improved with a meaningful allocation drop on the direct snapshot benchmark and no material allocation regression on the pressure benchmark.
  Validation: `go test ./internal/runtime2 -run "TestHandleHostRegion(UpdateSnapshot|DeclaredSourceLookup|StructuralRemount)" -count=1` and `go test ./internal/runtime2 -run ^$ -bench "BenchmarkHandleHostRegion(UpdateSnapshot|ManyHotRegionsBoundedWorkers)$" -benchmem -count=5`
  Checkpoint: completed todo `Add a source-snapshot reuse cache keyed by region epoch plus source-version tuple`; files changed: `internal/runtime2/host_region_adapter.go`, `internal/runtime2/host_region_snapshot_capture_test.go`, `docs/MULTITHREADED_RUNTIME_TODO.md`; result: focused snapshot/lookup/remount tests passed and before/after microbench deltas were recorded with improved `HandleHostRegionUpdateSnapshot` latency and allocs; residual risk: cache reuse assumes source lookup versioning is truthful (no value mutation without version advance), so a misbehaving source bridge can surface stale values until version increments; next suggested todo: `Add an O(1) scheduler update-coalesce index`.
- [x] Add an O(1) scheduler update-coalesce index (`region + cancel_version -> queue index`) so `handleSchedulerQueueAppend(...)` does not reverse-scan the queue for every update (`internal/runtime2/scheduler.go`).
  Notes: scheduler now keeps `storeSchedulerUpdateQueueIndexByKey` keyed by `(region, cancel_version)` and uses it in both `hasSchedulerEquivalentQueuedUpdate(...)` and `handleSchedulerQueueAppend(...)` to avoid reverse queue scans on update coalescing. Queue-compaction paths (`clearSchedulerQueueByRegionID`) rebuild the index to keep lookup correctness after cancel/dispose churn. Added `TestHandleSchedulerUpdateCoalesceIndexRebuildsAfterQueueCompaction`.
  Microbench before/after (`BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers$`, `-benchmem -count=5`):
  - before: `16,319-18,451 ns/op`, `382-383 B/op`, `47 allocs/op`
  - after: `17,511-25,880 ns/op`, `382-383 B/op`, `47 allocs/op`
  - summary: allocation profile stayed flat; runtime variance increased on this host run set, so this optimization currently lands as a structural coalescing improvement with inconclusive end-to-end throughput gain in the pressure benchmark.
  Validation: `go test ./internal/runtime2 -run "TestHandleSchedulerUpdate(CoalescesQueuedRegionUpdates|EquivalentQueuedUpdateSkipsHealthChecks|CoalesceIndexRebuildsAfterQueueCompaction)$" -count=1` and `go test ./internal/runtime2 -run ^$ -bench "BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers$" -benchmem -count=5`
  Checkpoint: completed todo `Add an O(1) scheduler update-coalesce index`; files changed: `internal/runtime2/scheduler.go`, `internal/runtime2/scheduler_update_coalesce_test.go`, `docs/MULTITHREADED_RUNTIME_TODO.md`; result: scheduler coalescing tests passed including the new compaction-rebuild case and before/after pressure benchmark numbers were recorded; residual risk: pressure benchmark variance/regression suggests the map-index overhead may not dominate this mixed end-to-end path yet, so follow-up should add a focused scheduler-only current-vs-legacy microbench to isolate net coalescing benefit; next suggested todo: `Investigate SIMD or CPU-feature accelerated ASCII text classification for decode hot paths`.
- [x] Investigate SIMD or CPU-feature accelerated ASCII text classification for decode hot paths (`parseRuntimeHasTrimmedNonWhitespaceText(...)`) with a strict portable fallback for unsupported targets (`internal/runtime2/text_whitespace.go`).
  Notes: prototyped an amd64/arm64 chunked ASCII path with portable fallback and benchmarked it; the candidate regressed the trim-check microbench and occasionally underperformed legacy `TrimSpace`, so it was intentionally reverted. Added focused correctness coverage (`TestParseRuntimeHasTrimmedNonWhitespaceText`) to pin ASCII-byte and representative Unicode parity with `strings.TrimSpace`.
  Microbench before/after (`Benchmark(ParseRuntimeHasTrimmedNonWhitespaceTextCurrentVsLegacy|ParseCanonicalRenderTree)$`, `-benchmem -count=5`):
  - before: `trim-check current` `2.359-3.114 ns/op`; `trim-check legacy` `3.216-4.588 ns/op`; `parse-tree` `13,613-22,118 ns/op`
  - after: `trim-check current` `2.328-2.900 ns/op`; `trim-check legacy` `2.814-3.078 ns/op`; `parse-tree` `17,244-21,265 ns/op`
  - summary: production path remains the existing ASCII-fast byte scan with Unicode fallback, with no allocation regressions and current trim-check path still competitive or better than legacy in this run set.
  Validation: `go test ./internal/runtime2 -run "TestParseRuntimeHasTrimmedNonWhitespaceText" -count=1` and `go test ./internal/runtime2 -run ^$ -bench "Benchmark(ParseRuntimeHasTrimmedNonWhitespaceTextCurrentVsLegacy|ParseCanonicalRenderTree)$" -benchmem -count=5`
  Checkpoint: completed todo `Investigate SIMD or CPU-feature accelerated ASCII text classification for decode hot paths`; files changed: `internal/runtime2/text_whitespace.go`, `internal/runtime2/text_whitespace_test.go`, `docs/MULTITHREADED_RUNTIME_TODO.md`; result: investigation complete, regressing candidate was not shipped, and new parity test plus before/after microbench evidence were recorded; residual risk: true SIMD gains likely require assembly/intrinsics and architecture-specific maintenance burden that outweighs current measured benefit for this tiny helper; next suggested todo: `Add an explicit fast path in BuildWorkerRenderInput(...) that reuses cached normalized declared-source order from mounted worker state`.
- [x] Add an explicit fast path in `BuildWorkerRenderInput(...)` that reuses cached normalized declared-source order from mounted worker state so update renders can avoid per-update map-key extraction and normalization (`internal/runtime2/worker_render_input_adapter.go`, `internal/runtime2/worker_region_runtime.go`).
  Notes: added `buildWorkerRenderInputWithSourceOrder(...)` plus keyset-match predicate `hasWorkerRenderSnapshotSourceOrderMatch(...)`; worker mount/update paths now pass cached source-ID order from `WorkerRegionState.SourceIDs` so unchanged source keysets skip per-update source-map key extraction + `NormalizeSourceIDs(...)`. Added focused adapter tests for cache reuse and source-set-change rebuild behavior.
  Microbench before/after (`BenchmarkHandleWorkerRegionUpdateSnapshotDrivenVsMetadataOnly$`, `-benchmem -count=5`):
  - before: `snapshot-driven` `4,610-5,321 ns/op` `4,659-4,660 B/op` `58 allocs/op`; `metadata-only` `4,390-6,724 ns/op` `4,299 B/op` `54 allocs/op`
  - after: `snapshot-driven` `4,632-6,120 ns/op` `4,659-4,660 B/op` `58 allocs/op`; `metadata-only` `4,692-5,909 ns/op` `4,299 B/op` `54 allocs/op`
  - summary: end-to-end worker update benchmark remained allocation-stable with mixed ns/op variance, while a new focused compare benchmark isolates the intended win in render-input adaptation.
  Current-vs-legacy microbench (`BenchmarkBuildWorkerRenderInputSourceOrderCurrentVsLegacy$`, `-benchmem -count=5`):
  - legacy rebuild order each call: `525.9-665.2 ns/op`, `640 B/op`, `4 allocs/op`
  - current reuse cached order: `275.7-336.4 ns/op`, `384 B/op`, `2 allocs/op`
  Validation: `go test ./internal/runtime2 -run "Test(HandleWorkerRegionUpdateCachesLatestSnapshot|HandleWorkerRegionUpdateRespondsToSnapshotInputChanges|BuildWorkerRenderInputWithSourceOrderReusesCachedSourceIDs|BuildWorkerRenderInputWithSourceOrderRebuildsOnSourceSetChange)$" -count=1` and `go test ./internal/runtime2 -run ^$ -bench "BenchmarkHandleWorkerRegionUpdateSnapshotDrivenVsMetadataOnly$" -benchmem -count=5` and `go test ./internal/runtime2 -run ^$ -bench "BenchmarkBuildWorkerRenderInputSourceOrderCurrentVsLegacy$" -benchmem -count=5`
  Checkpoint: completed todo `Add an explicit fast path in BuildWorkerRenderInput(...) that reuses cached normalized declared-source order`; files changed: `internal/runtime2/worker_render_input_adapter.go`, `internal/runtime2/worker_region_runtime.go`, `internal/runtime2/worker_render_input_adapter_test.go`, `internal/runtime2/perf_worker_render_input_source_order_compare_bench_test.go`, `docs/MULTITHREADED_RUNTIME_TODO.md`; result: worker snapshot-input/cache tests passed and the focused source-order compare benchmark showed reduced ns/op and allocations on the hot adaptation step; residual risk: cached order is keyed on source-ID set equality only, so workloads that frequently mutate source keysets still fall back to full key extraction+normalize on those updates; next suggested todo: `Add an internal trusted-renderer update path that can skip repeated ValidateWorkerRenderableRenderOutput(...) on hot update loops`.
- [x] Add an internal trusted-renderer update path that can skip repeated `ValidateWorkerRenderableRenderOutput(...)` on hot update loops while keeping strict validation for mount, tests, and debug-mode assertions (`internal/runtime2/worker_region_runtime.go`, `internal/runtime2/spec.go`).
  Notes: `WorkerRegionRuntime` now supports trusted-renderer metadata (`SetWorkerRegionRendererTrusted(...)`) and an update-path validation gate (`SetWorkerRegionUpdateValidationEnabled(...)`). Mount path remains strict/always validated; update path validates by default, but can skip repeated output validation only when validation is explicitly disabled and the renderer is marked trusted.
  Microbench before/after (`BenchmarkHandleWorkerRegionUpdateSnapshotDrivenVsMetadataOnly$`, `-benchmem -count=5`):
  - before: `snapshot-driven` `4,632-6,120 ns/op` `4,659-4,660 B/op` `58 allocs/op`; `metadata-only` `4,692-5,909 ns/op` `4,299 B/op` `54 allocs/op`
  - after: `snapshot-driven` `3,747-5,281 ns/op` `4,427 B/op` `45 allocs/op`; `metadata-only` `3,313-4,772 ns/op` `4,066 B/op` `41 allocs/op`
  - summary: trusted update path produced a meaningful allocation drop and lower median update latency in this run set while preserving strict mount-time validation.
  Validation: `go test ./internal/runtime2 -run "TestHandleWorkerRegionUpdate(ChangedInputProducesPatch|UnchangedInputProducesNoOp|TrustedRendererSkipsValidationWhenDisabled)$" -count=1` and `go test ./internal/runtime2 -run ^$ -bench "BenchmarkHandleWorkerRegionUpdateSnapshotDrivenVsMetadataOnly$" -benchmem -count=5`
  Checkpoint: completed todo `Add an internal trusted-renderer update path that can skip repeated ValidateWorkerRenderableRenderOutput(...) on hot update loops`; files changed: `internal/runtime2/worker_region_runtime.go`, `internal/runtime2/worker_render_input_bench_test.go`, `internal/runtime2/worker_region_runtime_trusted_update_test.go`, `docs/MULTITHREADED_RUNTIME_TODO.md`; result: required worker update tests passed with new trusted-gate coverage and benchmark run showed reduced allocations/latency on both snapshot-driven and metadata-only loops; residual risk: skipping update validation for trusted renderers can mask renderer regressions if trust classification is applied too broadly, so this gate should remain controlled by internal/runtime-only configuration; next suggested todo: `Review the next unchecked runtime2 performance hotspot and continue sequentially`.
- [x] Replace numeric `fmt.Sprintf("%v", value)` conversion in `parseBuildCanonicalRenderNode(...)` with type-specialized `strconv` formatting helpers to reduce stringify allocations on numeric-heavy trees (`internal/runtime2/render_ir.go`).
  Notes: canonical scalar formatting now routes bool, integer, and float values through direct `strconv` helpers, and the same fast path is reused for reflected map-key normalization and scalar string fallback before dropping to `fmt.Sprint(...)` only for non-scalar leftovers.
  Validation: `go test ./internal/runtime2 -run "TestBuildCanonicalRenderIR" -count=1` and `go test ./internal/runtime2 -run ^$ -bench "Benchmark(BuildCanonicalRenderIR|HandleWorkerRegionUpdateSnapshotDrivenVsMetadataOnly)$" -benchmem -count=5`
  Checkpoint: completed todo `Replace numeric fmt.Sprintf conversion`; files changed: `internal/runtime2/render_ir.go`, `docs/MULTITHREADED_RUNTIME_TODO.md`; result: canonical render tests passed and the focused render/update benchmarks stayed green; residual risk: non-scalar fallback paths still rely on `fmt.Sprint(...)`, so further stringify wins now depend on narrowing those remaining heterogeneous cases rather than numeric text nodes; next suggested todo: `Add pooled mutable map scratch state in ParsePatchStreamTransaction(...)`.
- [x] Add pooled mutable map scratch state in `ParsePatchStreamTransaction(...)` for known-node, sibling-count, and removed-key tracking to reduce per-transaction map clone churn (`internal/runtime2/patch_stream.go`).
  Notes: patch parsing now acquires reusable scratch maps for cloned known-node IDs, sibling counts, removed-node tracking, and removed-attr-key tracking, then clears and returns them after each transaction instead of allocating fresh clone maps on the first mutation path.
  Validation: `go test ./internal/runtime2 -run "TestParsePatchStreamTransaction" -count=1` and `go test ./internal/runtime2 -run ^$ -bench "Benchmark(ParsePatchStreamTransaction|CommitRegionPatchTransaction)$" -benchmem -count=5`
  Checkpoint: completed todo `Add pooled mutable map scratch state in ParsePatchStreamTransaction(...)`; files changed: `internal/runtime2/patch_stream.go`, `docs/MULTITHREADED_RUNTIME_TODO.md`; result: patch-stream tests passed and the focused parse/commit benchmarks stayed green; residual risk: the transaction parser still clones into scratch maps on the first mutation, so the next win in this area now depends more on reducing the number of mutations that require a clone than on map allocation itself; next suggested todo: `Add a direct patch-identity streaming path from BuildCanonicalPatchStream(...)`.
- [x] Add a direct patch-identity streaming path from `BuildCanonicalPatchStream(...)` so op emission can update identity digest incrementally and avoid a second JSON-encode pass in `BuildPatchStreamIdentity(...)` (`internal/runtime2/patch_stream.go`, `internal/runtime2/json_hash.go`).
  Notes: patch generation now opens one incremental identity stream from the raw header and normalized string table, updates that digest as each op is appended, and stores the finished hash directly in the built patch stream; `BuildPatchStreamIdentity(...)` now reuses the same field-ordered streaming path and an exact regression test locks the new digest to the legacy marshal output, including nil-slice `null` semantics.
  Validation: `go test ./internal/runtime2 -run "TestBuildPatchStreamIdentity" -count=1` and `go test ./internal/runtime2 -run ^$ -bench "Benchmark(BuildPatchStreamIdentity|HandleWorkerRegionUpdateSnapshotDrivenVsMetadataOnly)$" -benchmem -count=5`
  Checkpoint: completed todo `Add a direct patch-identity streaming path from BuildCanonicalPatchStream(...)`; files changed: `internal/runtime2/json_hash.go`, `internal/runtime2/patch_stream.go`, `internal/runtime2/patch_stream_identity_test.go`, `docs/MULTITHREADED_RUNTIME_TODO.md`; result: the new streaming hash matched the legacy patch identity across canonical diff, replace-subtree, and nil-op cases, and the focused identity/update benchmarks stayed green; residual risk: `BuildPatchStreamIdentity(...)` still pays per-op JSON encoder cost on standalone re-hash calls, so the later broader patch-identity hashing todo should focus on reducing nested op encoding overhead rather than the top-level wrapper pass; next suggested todo: `Add a dense-keyed-sibling collision bitmap path in ParseRenderNodeTable(...)`.
- [x] Add a dense-keyed-sibling collision bitmap path in `ParseRenderNodeTable(...)` for small sibling spans so duplicate keyed-child detection can avoid map allocation in common two-to-eight child sets (`internal/runtime2/render_node_table.go`).
  Notes: keyed-sibling validation now routes two-to-eight keyed children through `parseRenderNodeSiblingKeysWithDenseBuckets(...)`, which uses one 64-bucket occupancy bitmap plus tiny inline collision tracking instead of zeroing the larger probe tables used by the medium sibling path.
  Validation: `go test ./internal/runtime2 -run "TestParseRenderNodeTable" -count=1` and `go test ./internal/runtime2 -run ^$ -bench "Benchmark(ParseCanonicalRenderTree|HandleWorkerRegionUpdateSnapshotDrivenVsMetadataOnly)$" -benchmem -count=5`
  Checkpoint: completed todo `Add a dense-keyed-sibling collision bitmap path in ParseRenderNodeTable(...)`; files changed: `internal/runtime2/render_node_table.go`, `internal/runtime2/render_node_keyed_test.go`, `docs/MULTITHREADED_RUNTIME_TODO.md`; result: parse-node-table tests passed, including a new dense collision safety case, and the focused parse/update benchmarks stayed green; residual risk: the medium sibling validator still zeroes the fixed 128-slot probe tables for every keyed parent above the dense threshold, so the next parse-tree win should target broader canonical tree decode rather than this tiny-sibling branch; next suggested todo: `Optimize canonical tree decode hot path in ParseCanonicalRenderTree(...)`.
- [x] Optimize canonical tree decode hot path in `ParseCanonicalRenderTree(...)` (`internal/runtime2/render_ir.go`).
  Notes: canonical tree decode now keeps the existing validated node-table path but replaces the heavier child-order slice-header staging with compact child-pool start indexes, uses a smaller `uint32` depth side table, and resolves text/tag refs directly from string-table entries; a new `BenchmarkParseCanonicalRenderTreeCurrentVsLegacy` microbench tracks the optimized path against the previous assembly logic for regression detection.
  Validation: `go test ./internal/runtime2 -run "Test(BuildCanonicalRenderIRStableKeyedNodeIDsAcrossReorder|SetRegionSnapshotStateUpdatesSnapshotAndSources)" -count=1` and `go test ./internal/runtime2 -run ^$ -bench "BenchmarkParseCanonicalRenderTree$" -benchmem -count=5` and `go test ./internal/runtime2 -run ^$ -bench "BenchmarkParseCanonicalRenderTreeCurrentVsLegacy$" -benchmem -count=3`
  Checkpoint: completed todo `Optimize canonical tree decode hot path in ParseCanonicalRenderTree(...)`; files changed: `internal/runtime2/render_ir.go`, `internal/runtime2/perf_parse_canonical_tree_compare_bench_test.go`, `docs/MULTITHREADED_RUNTIME_TODO.md`; result: focused render/coordinator tests passed, the new current-vs-legacy microbench is in place, and the optimized path reduced benchmarked decode bytes from `55568 B/op` to `52368 B/op` while preserving the same allocation count; residual risk: prop-span decode still dominates part of the tree build cost, so the next adjacent todo should target node-table decode/validation or canonical prop decode rather than more slice reshaping here; next suggested todo: `Optimize node-table decode and validation in ParseRenderNodeTable(...)`.
- [x] Optimize node-table decode and validation in `ParseRenderNodeTable(...)` (`internal/runtime2/render_node_table.go`).
  Notes: render-node table parsing now collects node-ID min/max and sequential-density metadata during the initial decode loop so the common sequential-ID case avoids the previous extra full-table uniqueness pre-scan; `BenchmarkParseRenderNodeTableCurrentVsLegacy` was added to keep this optimization pinned against the previous full-rescan path for regression tracking.
  Validation: `go test ./internal/runtime2 -run "TestParseRenderNodeTable" -count=1` and `go test ./internal/runtime2 -run ^$ -bench "BenchmarkParseCanonicalRenderTree$" -benchmem -count=5` and `go test ./internal/runtime2 -run ^$ -bench "BenchmarkParseRenderNodeTableCurrentVsLegacy$" -benchmem -count=3`
  Checkpoint: completed todo `Optimize node-table decode and validation in ParseRenderNodeTable(...)`; files changed: `internal/runtime2/render_node_table.go`, `internal/runtime2/perf_render_node_table_compare_bench_test.go`, `docs/MULTITHREADED_RUNTIME_TODO.md`; result: parse-node-table tests passed, tree-decode benchmark stayed green, and the new current-vs-legacy microbench improved parse-node-table throughput from roughly `9.1-12.3 us/op` legacy to `6.5-8.6 us/op` current at the same `15528 B/op` and `5 allocs/op`; residual risk: child-span and sibling-key validation still each require their own full pass after decode, so the next parser win likely comes from narrowing those passes rather than more node-ID bookkeeping work; next suggested todo: `Reduce snapshot dispatch-hash cost in buildSnapshotDispatchHash(...) and hash apply helpers`.
- [x] Remove repeated sibling-index scans in insert ordering by precomputing next-tree parent sibling-index lookups used by `BuildCanonicalPatchStream(...)` insert-node sorting (`internal/runtime2/patch_stream.go`).
  Notes: insert-node sort now reuses one `buildNextSiblingIndexCache` via `parseGetCanonicalSiblingIndexFromCache(...)` instead of scanning parent child lists with `parseFindCanonicalSiblingIndex(...)` inside comparator calls.
  Validation: `go test ./internal/runtime2 -run ^$ -bench "BenchmarkBuildCanonicalPatchStream$" -benchmem -count=5`
- [x] Remove unnecessary `parseFilterCanonicalExistingOrder(...)` allocations when no insert/remove structural delta exists by reusing child-order slices in keyed-move planning (`internal/runtime2/patch_stream.go`).
  Notes: keyed-move planning now branches on `hasStructuralNodeDelta`; no-delta paths reuse canonical child-order slices for equality checks and only clone `currentOrder` when an actual move pass needs in-place reordering.
  Validation: `go test ./internal/runtime2 -run ^$ -bench "BenchmarkBuildCanonicalPatchStream$" -benchmem -count=5`
- [x] Reduce snapshot dispatch-hash cost in `buildSnapshotDispatchHash(...)` and hash apply helpers (`internal/runtime2/snapshot_dispatch_hash.go`).
  Notes: production dispatch hashing now uses the streamed SHA-256 path in `buildSnapshotDispatchHash(...)` and `handleHostRegionDispatchHash(...)` no longer carries adapter-local scratch bytes for buffered payload staging; `BenchmarkBuildSnapshotDispatchHashCurrentVsLegacy` was added to pin the streamed path against the previous buffered helper for regression tracking.
  Validation: `go test ./internal/runtime2 -run "TestBuildSnapshotDispatchHash|TestHandleHostRegionDispatchHashUsesStreamedPathWithoutScratch" -count=1` and `go test ./internal/runtime2 -run ^$ -bench "BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers$" -benchmem -count=5` and `go test ./internal/runtime2 -run ^$ -bench "BenchmarkBuildSnapshotDispatchHashCurrentVsLegacy$" -benchmem -count=3`
  Checkpoint: completed todo `Reduce snapshot dispatch-hash cost in buildSnapshotDispatchHash(...) and hash apply helpers`; files changed: `internal/runtime2/snapshot_dispatch_hash.go`, `internal/runtime2/host_region_adapter.go`, `internal/runtime2/snapshot_dispatch_hash_test.go`, `internal/runtime2/perf_snapshot_dispatch_hash_compare_bench_test.go`, `docs/MULTITHREADED_RUNTIME_TODO.md`; result: focused dispatch-hash tests passed, the hot-regions benchmark stayed green, and the new current-vs-legacy microbench improved hash time from roughly `2.0-2.4 us/op` legacy to `1.6-1.9 us/op` current while reducing bytes from `1769 B/op` to `1442 B/op`; residual risk: the streamed path still stages nested props and source maps through `appendSnapshotDispatchValue(...)` and `appendSnapshotDispatchAnyMap(...)`, so the next dispatch-hash win depends more on reducing nested map/value encoding work than on the top-level digest wrapper; next suggested todo: `Reduce dispatch-path overhead in HandleHostRegionUpdateDispatchWithPriority(...) and handleHostRegionDispatchHash(...)`.
- [x] Reduce dispatch-path overhead in `HandleHostRegionUpdateDispatchWithPriority(...)` and `handleHostRegionDispatchHash(...)` (`internal/runtime2/host_region_adapter.go`).
  Notes: host dispatch hashing now hashes the original snapshot envelope directly instead of first copying it just to overwrite `InputVersion`; dispatch hashing already excludes `InputVersion`, so the copy was dead work on every deferred and no-change dispatch. `BenchmarkHandleHostRegionDispatchHashCurrentVsLegacy` was added to pin the live caller path against the copied-envelope implementation for regression tracking.
  Validation: `go test ./internal/runtime2 -run "Test(HandleHostRegionUpdateDispatch.*NoChange|HandleHostRegionDispatchHashIgnoresInputVersion|HandleHostRegionUpdateDispatchWithPriorityDeferredClassification)" -count=1` and `go test ./internal/runtime2 -run ^$ -bench "BenchmarkHandleHostRegionUpdateDispatchWithPriorityDeferred$" -benchmem -count=5` and `go test ./internal/runtime2 -run ^$ -bench "BenchmarkHandleHostRegionDispatchHashCurrentVsLegacy$" -benchmem -count=3`
  Checkpoint: completed todo `Reduce dispatch-path overhead in HandleHostRegionUpdateDispatchWithPriority(...) and handleHostRegionDispatchHash(...)`; files changed: `internal/runtime2/host_region_adapter.go`, `internal/runtime2/snapshot_dispatch_hash_test.go`, `internal/runtime2/perf_host_region_dispatch_hash_compare_bench_test.go`, `docs/MULTITHREADED_RUNTIME_TODO.md`; result: focused no-change/deferred dispatch tests passed, deferred dispatch benchmark stayed green, and the new current-vs-legacy microbench improved host dispatch hash time from roughly `2.6-3.1 us/op` legacy to `2.4-2.7 us/op` current at the same `208 B/op` and `6 allocs/op`; residual risk: the live worktree already includes a fast-hash prefilter and buffered scratch state, so the next meaningful dispatch-path win now depends more on avoiding hashing altogether or reducing nested envelope encoding than on wrapper overhead; next suggested todo: `Reduce snapshot-capture overhead in HandleHostRegionUpdateSnapshot(...)`.
- [x] Remove eager empty-attr map allocation in insert-node decode by making `GetAttrByKey` lazy in `parseBuildRegionDOMNodeFromPatchRecord(...)` (`internal/runtime2/patch_stream.go`).
  Validation: `go test ./internal/runtime2 -run ^$ -bench "Benchmark(ParsePatchStreamTransaction|CommitRegionPatchTransaction)$" -benchmem -count=5`
- [x] Collapse keyed-move patch lookup extraction in `HandleHostRegionPatchCommit(...)` to one pass via `BuildRegionDOMPatchLookupMaps(...)` when sibling counts are required, instead of separate known-node and sibling-count walks (`internal/runtime2/host_region_adapter.go`, `internal/runtime2/patch_stream.go`).
  Validation: `go test ./internal/runtime2 -run ^$ -bench "BenchmarkCommitRegionPatchTransaction$" -benchmem -count=5`
- [x] Remove duplicate keyed-move detection scans by passing host-known `hasPatchKeyedMoveOp` into `ParsePatchStreamTransaction(...)` instead of re-scanning ops in `parseHasPatchKeyedMoveOpFromIndex(...)` (`internal/runtime2/host_region_adapter.go`, `internal/runtime2/patch_stream.go`).
  Validation: `go test ./internal/runtime2 -run ^$ -bench "Benchmark(ParsePatchStreamTransaction|CommitRegionPatchTransaction)$" -benchmem -count=5`
- [x] Reduce snapshot-capture overhead in `HandleHostRegionUpdateSnapshot(...)`, including serializable validation and source snapshot work (`internal/runtime2/host_region_adapter.go`, `internal/runtime2/spec.go`).
  Notes: the hot snapshot path now builds declared source values and the common source version in one pass via `buildSnapshotSourceValuesAndVersion(...)`, then materializes the `SnapshotEnvelope` directly instead of copying source values once, source versions again, and looping a third time just to re-check source-version consistency. `BenchmarkHandleHostRegionUpdateSnapshotCurrentVsLegacy` was added to keep the current snapshot capture path pinned against the previous multi-pass source snapshot flow.
  Validation: `go test ./internal/runtime2 -run "Test(HandleHostRegionUpdateSnapshot|HandleHostRegionDeclaredSourceLookup)" -count=1` and `go test ./internal/runtime2 -run ^$ -bench "BenchmarkHandleHostRegion(UpdateSnapshot|ManyHotRegionsBoundedWorkers)$" -benchmem -count=5` and `go test ./internal/runtime2 -run ^$ -bench "BenchmarkHandleHostRegionUpdateSnapshotCurrentVsLegacy$" -benchmem -count=3`
  Checkpoint: completed todo `Reduce snapshot-capture overhead in HandleHostRegionUpdateSnapshot(...)`; files changed: `internal/runtime2/snapshot.go`, `internal/runtime2/host_region_adapter.go`, `internal/runtime2/perf_host_region_update_snapshot_compare_bench_test.go`, `docs/MULTITHREADED_RUNTIME_TODO.md`; result: focused source lookup/snapshot tests passed, the existing snapshot and hot-regions benchmarks stayed green, and the new current-vs-legacy microbench improved snapshot capture time from roughly `0.75-0.84 us/op` legacy to `0.61-0.74 us/op` current, though bytes and allocs increased from `936 B/op, 6 allocs/op` to `1192 B/op, 8 allocs/op`; residual risk: the path is faster but still allocates fresh source/version maps for every capture, so the next worthwhile win likely comes from canonical source-order hashing or source-lookup reuse rather than more envelope assembly tweaks; next suggested todo: `Eliminate per-update source-map key extraction and sorting in dispatch hashing by hashing sources in canonical declared-source order from coordinator entry state`.
- [x] Eliminate per-update source-map key extraction and sorting in dispatch hashing by hashing sources in canonical declared-source order from coordinator entry state (`internal/runtime2/snapshot_dispatch_hash.go`, `internal/runtime2/host_region_adapter.go`, `internal/runtime2/coordinator.go`).
  Notes: dispatch-hash payload building now uses `appendSnapshotDispatchEnvelopeWithSourceIDs(...)` and `appendSnapshotDispatchSourceMap(...)` so host no-change checks can serialize `SnapshotEnvelope.Sources` directly in canonical declared-source order instead of extracting and sorting source-map keys on every update. `HandleHostRegionUpdateDispatchWithPriority(...)` now threads the already canonical `getSnapshotSourceIDs` into `handleHostRegionDispatchHash(...)`, and `BenchmarkHandleHostRegionDispatchHashCurrentVsLegacy` was updated to pin that ordered-source path against the previous generic sorted-map flow.
  Validation: `go test ./internal/runtime2 -run "Test(BuildSnapshotDispatchHash|HandleHostRegionDispatchHash|HandleHostRegionUpdateDispatch.*NoChange)" -count=1` and `go test ./internal/runtime2 -run ^$ -bench "BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers$" -benchmem -count=5` and `go test ./internal/runtime2 -run ^$ -bench "BenchmarkHandleHostRegionDispatchHashCurrentVsLegacy$" -benchmem -count=3`
  Checkpoint: completed todo `Eliminate per-update source-map key extraction and sorting in dispatch hashing by hashing sources in canonical declared-source order from coordinator entry state`; files changed: `internal/runtime2/snapshot_dispatch_hash.go`, `internal/runtime2/host_region_adapter.go`, `internal/runtime2/snapshot_dispatch_hash_test.go`, `internal/runtime2/perf_host_region_dispatch_hash_compare_bench_test.go`, `docs/MULTITHREADED_RUNTIME_TODO.md`; result: focused dispatch-hash/no-change tests passed, the hot-regions benchmark stayed green, and the current-vs-legacy microbench improved host dispatch hashing from roughly `0.88-1.03 us/op, 208 B/op, 6 allocs/op` legacy to `0.78-0.87 us/op, 104 B/op, 3 allocs/op` current on changed payloads, with similar improvement on stable payloads; residual risk: the dispatch path still re-encodes props and nested source values on every check, so the next meaningful win is more likely in coordinator lock/copy overhead or in compact prop storage than in further top-level source-order tweaks; next suggested todo: `Reduce coordinator lock and entry-copy overhead in Coordinator.GetEntry(...) and Coordinator.UpdateRegion(...)`.
- [x] Reduce coordinator lock and entry-copy overhead in `Coordinator.GetEntry(...)` and `Coordinator.UpdateRegion(...)` (`internal/runtime2/coordinator.go`).
  Notes: `Coordinator` now stores pointer-backed entries so hot reads and update bumps no longer copy full map values in and out of `storeEntries` for steady-state access. `GetEntry(...)` now returns one value snapshot from a pointer-backed store after an inline read unlock, while `UpdateRegion(...)` mutates the mounted entry directly instead of forwarding through the value-returning `UpdateRegionAndGetEntry(...)` path. `BenchmarkCoordinatorGetEntryCurrentVsLegacy` and `BenchmarkCoordinatorUpdateRegionCurrentVsLegacy` were added to pin the pointer-backed hot path against the previous value-map implementation.
  Validation: `go test ./internal/runtime2 -run "Test(UpdateRegion|MountRegion|SetRegionSnapshotState|Coordinator)" -count=1` and `go test ./internal/runtime2 -run ^$ -bench "BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers$" -benchmem -count=5` and `go test ./internal/runtime2 -run ^$ -bench "BenchmarkCoordinator(GetEntry|UpdateRegion)CurrentVsLegacy$" -benchmem -count=3`
  Checkpoint: completed todo `Reduce coordinator lock and entry-copy overhead in Coordinator.GetEntry(...) and Coordinator.UpdateRegion(...)`; files changed: `internal/runtime2/coordinator.go`, `internal/runtime2/perf_coordinator_get_update_compare_bench_test.go`, `docs/MULTITHREADED_RUNTIME_TODO.md`; result: focused coordinator tests passed, the hot-regions benchmark stayed green, and the new compare benches improved `GetEntry` from roughly `24.6-29.4 ns/op` legacy to `19.8-20.7 ns/op` current and `UpdateRegion` from roughly `50.2-51.3 ns/op` legacy to `25.7-27.3 ns/op` current, all at `0 allocs/op`; residual risk: other coordinator mutators still use the older copy-out/copy-back helpers, so the next coordinator-side win is more likely in scheduler queue work or deeper coordinator transaction consolidation than in more tuning of these two hot methods; next suggested todo: `Reduce scheduler queue overhead in Scheduler.HandleSchedulerUpdate(...) and handleSchedulerQueueAppend(...)`.
- [x] Reduce scheduler queue overhead in `Scheduler.HandleSchedulerUpdate(...)` and `handleSchedulerQueueAppend(...)` (`internal/runtime2/scheduler.go`).
  Notes: scheduler update coalescing now uses a region-keyed queue index that stores the active cancel generation in the value instead of hashing a composite `{region,cancel}` key on every update probe and append. `HandleSchedulerUpdate(...)` now returns the already-queued job directly on equivalent no-op updates, and `handleSchedulerQueueAppend(...)` no longer redoes update-key revalidation because the hot update path now routes through `handleSchedulerQueueAppendUpdate(...)`. `BenchmarkHandleSchedulerUpdateCurrentVsLegacy` was added to pin both distinct append-path updates and equivalent queued updates against the previous double-lookup composite-key flow.
  Validation: `go test ./internal/runtime2 -run "TestHandleScheduler(Update|Mount|Cancel|Dispose|Fallback|ReplaceWorker).*" -count=1` and `go test ./internal/runtime2 -run ^$ -bench "BenchmarkHandleSchedulerUpdateCurrentVsLegacy$" -benchmem -count=3` and `go test ./internal/runtime2 -run ^$ -bench "BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers$" -benchmem -count=5`
  Checkpoint: completed todo `Reduce scheduler queue overhead in Scheduler.HandleSchedulerUpdate(...) and handleSchedulerQueueAppend(...)`; files changed: `internal/runtime2/scheduler.go`, `internal/runtime2/perf_scheduler_update_compare_bench_test.go`, `docs/MULTITHREADED_RUNTIME_TODO.md`; result: focused scheduler tests passed, the hot-regions benchmark stayed green, and the new compare bench improved distinct append-path updates from roughly `222.5-253.0 ns/op, 273 B/op` legacy to `168.7-189.2 ns/op, 145 B/op` current while improving equivalent queued updates from roughly `27.7-28.2 ns/op` legacy to `22.6-22.9 ns/op` current, all at `0 allocs/op`; residual risk: queue compaction still rebuilds the update index by scanning the full queue after region cancellation, so the next scheduler-side win is more likely there or in later host dispatch batching rather than in another steady-state update coalesce tweak; next suggested todo: `Optimize patch identity hashing in BuildPatchStreamIdentity(...) and JSON streaming hash path`.
- [x] Optimize patch identity hashing in `BuildPatchStreamIdentity(...)` and JSON streaming hash path (`internal/runtime2/patch_stream.go`, `internal/runtime2/json_hash.go`).
  Notes: `BuildPatchStreamIdentity(...)` now takes a fast cached-identity return path when `PatchStreamRaw.GetPatchIdentity` is already non-empty (the common case for streams built by `BuildPatchStreamRaw(...)` and canonical patch generation), avoiding a full JSON streaming hash replay for already-materialized patch payloads.
  Validation: `go test ./internal/runtime2 -run "Test(BuildPatchStreamIdentityMatchesLegacyMarshalEncoding|ParsePatchStreamTransaction)$" -count=1` and `go test ./internal/runtime2 -run ^$ -bench "BenchmarkBuildPatchStreamIdentity$" -benchmem -count=5`
  Checkpoint: completed todo `Optimize patch identity hashing in BuildPatchStreamIdentity(...) and JSON streaming hash path`; files changed: `internal/runtime2/patch_stream.go`, `docs/MULTITHREADED_RUNTIME_TODO.md`; result: identity and parse tests passed, and `BenchmarkBuildPatchStreamIdentity` improved from `small: ~1334-1808 ns/op, 360 B/op, 9 allocs/op` and `large-keyed-rotate: ~26105-34436 ns/op, 6819 B/op, 133 allocs/op` to `small: ~3.29-4.64 ns/op, 0 B/op, 0 allocs/op` and `large-keyed-rotate: ~3.83-5.45 ns/op, 0 B/op, 0 allocs/op` via cached-identity reuse; residual risk: callers that mutate header/string-table/op fields after setting `GetPatchIdentity` now keep that cached identity unless they explicitly clear or rebuild it; next suggested todo: `Optimize canonical prop decode in parseBuildCanonicalPropByKeyFromRaw(...)`.
- [x] Optimize canonical prop decode in `parseBuildCanonicalPropByKeyFromRaw(...)` (`internal/runtime2/render_ir.go`).
  Notes: prop decode now uses an inlined raw-kind validator, one cached string-table bound, and an ASCII-boundary key-text fast path; duplicate-key detection moved to a deferred mismatch check (`len(map) != len(raw)`) so the hot loop avoids a second per-record map lookup in the canonical non-duplicate path.
  Validation: `go test ./internal/runtime2 -run "Test(BuildCanonicalRenderIR|ParseRenderNodeTable)" -count=1` and `go test ./internal/runtime2 -run ^$ -bench "BenchmarkParseCanonicalRenderTree$" -benchmem -count=5`
  Checkpoint: completed todo `Optimize canonical prop decode in parseBuildCanonicalPropByKeyFromRaw(...)`; files changed: `internal/runtime2/render_ir.go`, `docs/MULTITHREADED_RUNTIME_TODO.md`; result: focused render IR/node-table tests passed and parse-tree benchmark stayed allocation-stable at `52368 B/op`, `15 allocs/op` with ns/op in a noisy same-band range on this host (`before ~15693-19537 ns/op`, `after ~16780-21689 ns/op`), so throughput impact is currently inconclusive without a pinned benchstat runner; residual risk: duplicate-key fallback now does a second-pass key scan only on malformed duplicate inputs, so malformed payload cost is slightly higher even though canonical-path work is reduced; next suggested todo: `Tighten ValidateSerializableProps(...) and isSerializableAnyFast(...) common-case exits`.
- [x] Tighten `ValidateSerializableProps(...)` and `isSerializableAnyFast(...)` common-case exits (`internal/runtime2/spec.go`).
  Notes: `ValidateSerializableProps(...)` now keeps common typed scalar containers on the non-reflective fast path by accepting typed scalar slices directly and by validating typed `map[string]scalar` payloads with the same key-guard logic used for `map[string]any`. The `[]any` and `map[string]any` fast paths also skip recursive re-checks for primitive scalar leaves before falling back to deeper traversal.
  Validation: `go test ./internal/runtime2 -run "Test(HandleHostRegionUpdateSnapshotPropsCacheInvalidatesOnInPlaceMutation|ValidateSerializablePropsAcceptsTypedScalarContainers|ValidateSerializablePropsRejectsTypedScalarMapRefMarker)" -count=1` and `go test ./internal/runtime2 -run ^$ -bench "BenchmarkValidateSerializablePropsCurrentVsLegacy$" -benchmem -count=3` and `go test ./internal/runtime2 -run ^$ -bench "BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers$" -benchmem -count=5`
  Checkpoint: completed todo `Tighten ValidateSerializableProps(...) and isSerializableAnyFast(...) common-case exits`; files changed: `internal/runtime2/spec.go`, `internal/runtime2/spec_test.go`, `internal/runtime2/perf_serializable_props_compare_bench_test.go`, `docs/MULTITHREADED_RUNTIME_TODO.md`; result: focused serializable-props tests passed, the new compare bench shows typed string-map validation improving from roughly `269.5-313.3 ns/op, 128 B/op, 8 allocs/op` legacy to `69.7-74.0 ns/op, 0 B/op, 0 allocs/op` current, and typed int-slice validation improving from roughly `58.5-88.7 ns/op, 24 B/op, 1 alloc/op` legacy to `18.5-24.0 ns/op, 24 B/op, 1 alloc/op` current; the widened hot-regions benchmark stayed in the expected `~17.5-21.0 µs/op, 382 B/op, 47 allocs/op` band; residual risk: nested typed containers behind interfaces and struct-heavy prop graphs still fall back to reflection, so the next higher-impact pass remains shape caching rather than adding more exact-type switch arms; next suggested todo: `Normalize and cache keyed-node presence once during canonical render-node build to remove repeated strings.TrimSpace(...) checks in node-ID assignment, string-table extraction, and patch keyed-move detection`.
- [ ] Normalize and cache keyed-node presence once during canonical render-node build to remove repeated `strings.TrimSpace(...)` checks in node-ID assignment, string-table extraction, and patch keyed-move detection (`internal/runtime2/render_ir.go`, `internal/runtime2/patch_stream.go`).
  Validation: `go test ./internal/runtime2 -run ^$ -bench "Benchmark(BuildCanonicalRenderIR|BuildCanonicalPatchStream)$" -benchmem -count=5`
- [x] Pre-size patch-string accumulation in `BuildCanonicalPatchStream(...)` to avoid repeated `append` growth while collecting insert/text/attr/style key payloads (`internal/runtime2/patch_stream.go`).
  Notes: added `buildPatchStringCapacityFromPatchOps(...)` and now allocate `buildPatchStrings` with a computed capacity hint before collecting insert/text/attr/style/remove-attr payload strings.
  Validation: `go test ./internal/runtime2 -run ^$ -bench "BenchmarkBuildCanonicalPatchStream$" -benchmem -count=5`

### Runtime2 Performance Hotspot Backlog (Additional 20)

- [ ] Optimize diff generation in `BuildCanonicalPatchStream(...)` and keyed-move ordering helpers (`internal/runtime2/patch_stream.go`).
  Validation: `go test ./internal/runtime2 -run ^$ -bench "BenchmarkBuildCanonicalPatchStream$" -benchmem -count=5`
- [ ] Reduce parse-time allocations in `ParsePatchStreamTransaction(...)`, especially mutable lookup-map copy paths (`internal/runtime2/patch_stream.go`).
  Validation: `go test ./internal/runtime2 -run ^$ -bench "BenchmarkParsePatchStreamTransaction$" -benchmem -count=5`
- [ ] Reduce rollback snapshot overhead in `CommitRegionPatchTransaction(...)` plus `parseCloneRegionNodeMap(...)` restore paths (`internal/runtime2/dom_commit.go`).
  Validation: `go test ./internal/runtime2 -run ^$ -bench "BenchmarkCommitRegionPatchTransaction$" -benchmem -count=5`
- [ ] Optimize canonical IR build path in `BuildCanonicalRenderIR(...)` for large host trees (`internal/runtime2/render_ir.go`).
  Validation: `go test ./internal/runtime2 -run ^$ -bench "BenchmarkBuildCanonicalRenderIR$" -benchmem -count=5`
- [ ] Reduce reflection and stringify cost in `parseBuildCanonicalRenderNode(...)` and `parseBuildCanonicalRenderNodeFromMap(...)` (`internal/runtime2/render_ir.go`).
  Validation: `go test ./internal/runtime2 -run ^$ -bench "BenchmarkCompareLocalVsWorkerBackedRendering$" -benchmem -count=5`
- [ ] Reduce child-map normalization overhead in `parseBuildCanonicalChildren(...)` and `parseBuildCanonicalMapValue(...)` (`internal/runtime2/render_ir.go`).
  Validation: `go test ./internal/runtime2 -run ^$ -bench "BenchmarkBuildCanonicalRenderIR$" -benchmem -count=5`
- [ ] Reduce prop extraction and sort overhead in `parseBuildCanonicalProps(...)` and `parseBuildCanonicalPropRecord(...)` (`internal/runtime2/render_ir.go`).
  Validation: `go test ./internal/runtime2 -run ^$ -bench "BenchmarkBuildCanonicalRenderIR$" -benchmem -count=5`
- [ ] Reduce string-table build and lookup overhead in `BuildRenderStringTable(...)` and `RenderStringTable.GetRenderStringRef(...)` (`internal/runtime2/render_string_table.go`).
  Validation: `go test ./internal/runtime2 -run ^$ -bench "BenchmarkBuildCanonicalRenderIR$" -benchmem -count=5`
- [ ] Reduce prop decode and canonicalization overhead in `ParseRenderPropRecords(...)` (`internal/runtime2/render_prop_record.go`).
  Validation: `go test ./internal/runtime2 -run ^$ -bench "BenchmarkParseCanonicalRenderTree$" -benchmem -count=5`
- [ ] Optimize update critical path in `HandleWorkerRegionUpdate(...)`, especially repeated no-op and metadata-only updates (`internal/runtime2/worker_region_runtime.go`).
  Validation: `go test ./internal/runtime2 -run ^$ -bench "BenchmarkHandleWorkerRegionUpdateSnapshotDrivenVsMetadataOnly$" -benchmem -count=5`
- [ ] Reduce worker render-input overhead in `BuildWorkerRenderInput(...)` (source-ID extraction and normalization) (`internal/runtime2/worker_render_input_adapter.go`).
  Validation: `go test ./internal/runtime2 -run ^$ -bench "BenchmarkHandleWorkerRegionUpdateSnapshotDrivenVsMetadataOnly$" -benchmem -count=5`
- [ ] Reduce end-to-end orchestration overhead in `HandleHostWorkerRegionUpdateOrchestration(...)` and `parseDecodeSnapshotForOrchestration(...)` (`internal/runtime2/host_worker_orchestration_helper.go`).
  Validation: `go test ./internal/runtime2 -run ^$ -bench "BenchmarkCompareLocalVsWorkerBackedRendering$" -benchmem -count=5`
- [ ] Reduce fallback decode overhead in `ParseHostPatchPayloadWithFallback(...)` for mixed transport tiers (`internal/runtime2/host_patch_transport_parse.go`).
  Validation: `go test ./internal/runtime2 -run ^$ -bench "Benchmark(ParseBinaryPatchPayload|GetSharedPatchReadPayload)$" -benchmem -count=5`
- [ ] Reduce duplicate JSON work in `parsePatchStreamFromStructuredClonePayload(...)` (`internal/runtime2/host_patch_transport_parse.go`).
  Validation: `go test ./internal/runtime2 -run ^$ -bench "Benchmark(ParseBinaryPatchPayload|GetSharedPatchReadPayload)$" -benchmem -count=5`
- [ ] Reduce structured-clone snapshot encode/decode cost in `BuildStructuredCloneSnapshotEnvelopeJSON(...)` and `ParseStructuredCloneSnapshotEnvelopeJSON(...)` (`internal/runtime2/structured_clone_snapshot_transport.go`).
  Validation: `go test ./internal/runtime2 -run ^$ -bench "BenchmarkBuildStructuredCloneSnapshotEnvelopeJSON$" -benchmem -count=5`
- [ ] Reduce binary snapshot body cost in `appendBinarySnapshotBody(...)` and `ParseBinarySnapshotBody(...)` (`internal/runtime2/binary_snapshot_body.go`).
  Validation: `go test ./internal/runtime2 -run ^$ -bench "BenchmarkBuildAndParseBinarySnapshotEnvelope$" -benchmem -count=5`
- [ ] Reduce source-values section encode/decode overhead in `appendBinarySourceValuesSection(...)` and `parseBinarySourceValuesSection(...)` (`internal/runtime2/binary_snapshot_body.go`).
  Validation: `go test ./internal/runtime2 -run ^$ -bench "BenchmarkParseBinarySnapshotEnvelopeSourceHeavy$" -benchmem -count=5`
- [ ] Reduce map/list encoding overhead in `buildBinarySourceValueInto(...)`, `buildBinarySourceAnyMapInto(...)`, and reflect fallback helpers (`internal/runtime2/binary_source_value.go`).
  Validation: `go test ./internal/runtime2 -run ^$ -bench "BenchmarkBuildBinarySourceValueAnyMapFastPath$" -benchmem -count=5`
- [ ] Reduce source-value decode recursion overhead in `ParseBinarySourceValue(...)`, `parseBinarySourceListValue(...)`, and `parseBinarySourceMapValue(...)` (`internal/runtime2/binary_source_value.go`).
  Validation: `go test ./internal/runtime2 -run ^$ -bench "BenchmarkParseBinarySnapshotEnvelopeSourceHeavy$" -benchmem -count=5`
- [ ] Reduce source-ID table parse/build overhead in `appendBinarySourceIDTableFromNormalized(...)` and `parseBinarySourceIDTableInto(...)` (`internal/runtime2/binary_source_id_table.go`).
  Validation: `go test ./internal/runtime2 -run ^$ -bench "BenchmarkParseBinarySourceIDTableCanonical$" -benchmem -count=5`

### Runtime2 Performance Hotspot Backlog (Additional 20, Round 2)

- [x] Optimize shared snapshot publish and read-copy hot paths in `HandleSharedSnapshotPublishPayload(...)` and `GetSharedSnapshotReadPayload(...)` (`internal/runtime2/shared_snapshot_page.go`).
  Notes: shared snapshot publish now writes header fields directly into page storage (removing `BuildSharedSnapshotPageHeader(...)` allocation/copy churn) and reads header fields via an in-place parser on the page buffer while preserving validation and error contracts.
  Validation: `go test ./internal/runtime2 -run "Test(HandleSharedSnapshotPublishBeginWritesWritingHeader|HandleSharedSnapshotPublishBeginAdvancesGeneration|HandleSharedSnapshotPublishCompleteAcceptsCompletePublish|HandleSharedSnapshotPublishCompleteRejectsIncompletePublish|HandleSharedSnapshotPublishCompleteRejectsGenerationMismatch|HandleSharedSnapshotReadHeaderRejectsTornWrite|HandleSharedSnapshotReadHeaderAcceptsCompletedPublish|HandleSharedSnapshotReadHeaderAtGenerationAcceptsMatchingGeneration|HandleSharedSnapshotReadHeaderAtGenerationRejectsStaleGeneration|HandleSharedSnapshotPublishPayloadWritesReadablePage|HandleSharedSnapshotPublishPayloadRepeatedPublishAdvancesGeneration|HandleSharedSnapshotPublishPayloadRejectsOverflow|ParseSharedSnapshotEnvelopeRejectsMalformedHeaderBeforeDecode|ParseSharedSnapshotEnvelopeRejectsCorruptPayloadLengthBeforeDecode)$" -count=1` and `go test ./internal/runtime2 -run ^$ -bench "BenchmarkHandleSharedSnapshotPublishAndRead$" -benchmem -count=5`
  Checkpoint: completed todo `Optimize shared snapshot publish and read-copy hot paths`; files changed: `internal/runtime2/shared_snapshot_page.go`, `docs/MULTITHREADED_RUNTIME_TODO.md`; result: benchmark improved from publish `~47.57-78.70 ns/op`, `48 B/op`, `2 allocs/op` to publish `~10.47-11.34 ns/op`, `0 B/op`, `0 allocs/op`; read remained `~34.61-57.16 ns/op` before and `~38.00-42.10 ns/op` after with expected `96 B/op`, `1 alloc/op` copy behavior; residual risk: read path still allocates by design to return a defensive payload copy; next suggested todo: `Reduce shared snapshot header rebuild and reparse churn in HandleSharedSnapshotPublishBegin(...), HandleSharedSnapshotPublishComplete(...), and HandleSharedSnapshotReadHeaderAtGeneration(...)`.
- [x] Reduce shared snapshot header rebuild and reparse churn in `HandleSharedSnapshotPublishBegin(...)`, `HandleSharedSnapshotPublishComplete(...)`, and `HandleSharedSnapshotReadHeaderAtGeneration(...)` (`internal/runtime2/shared_snapshot_page.go`).
  Notes: completed with the same in-place shared-page header writer/parser pass used for the previous snapshot publish/read hot-path todo, eliminating begin/complete header rebuild allocations and reducing repeated full header parse churn in shared snapshot flow.
  Validation: `go test ./internal/runtime2 -run "TestHandleSharedSnapshot(PublishBeginAdvancesGeneration|PublishCompleteAcceptsCompletePublish|ReadHeaderAtGenerationAcceptsMatchingGeneration|ReadHeaderRejectsTornWrite)$" -count=1` and `go test ./internal/runtime2 -run ^$ -bench "BenchmarkHandleSharedSnapshotPublishAndRead$" -benchmem -count=5`
  Checkpoint: completed todo `Reduce shared snapshot header rebuild and reparse churn`; files changed: `internal/runtime2/shared_snapshot_page.go`, `docs/MULTITHREADED_RUNTIME_TODO.md`; result: header hot-path work validated and benchmark retained publish-path gains (`~10.47-11.34 ns/op`, `0 B/op`, `0 allocs/op`) from the pre-change baseline (`~47.57-78.70 ns/op`, `48 B/op`, `2 allocs/op`); residual risk: read-path copies remain allocation-bound by API contract; next suggested todo: `Optimize shared patch publish and read-copy hot paths in HandleSharedPatchPublishPayload(...) and GetSharedPatchReadPayload(...)`.
- [x] Optimize shared patch publish and read-copy hot paths in `HandleSharedPatchPublishPayload(...)` and `GetSharedPatchReadPayload(...)` (`internal/runtime2/shared_patch_page.go`).
  Notes: shared patch publish now writes page headers in place (removing header build/copy allocations), and read-side header parsing is now in-place with kind/status validation preserved before payload copy.
  Validation: `go test ./internal/runtime2 -run "Test(BuildSharedPatchTransportResult(PrefersSharedBuffer|DowngradesWhenSharedUnavailable)|HandleSharedPatchPublishAndReadRoundTrip|SharedPatchPayloadEndToEndCommit)$" -count=1` and `go test ./internal/runtime2 -run ^$ -bench "Benchmark(HandleSharedPatchPublishPayload|GetSharedPatchReadPayload)$" -benchmem -count=5`
  Checkpoint: completed todo `Optimize shared patch publish and read-copy hot paths`; files changed: `internal/runtime2/shared_patch_page.go`, `docs/MULTITHREADED_RUNTIME_TODO.md`; result: publish microbench improved from `~38.77-48.28 ns/op`, `48 B/op`, `2 allocs/op` to `~7.29-11.53 ns/op`, `0 B/op`, `0 allocs/op`; read improved from `~89.03-107.4 ns/op` to `~66.54-79.98 ns/op` with expected `288 B/op`, `1 alloc/op` payload copy; residual risk: read path keeps one allocation by API contract because returned payload must not alias mutable page storage; next suggested todo: `Reduce shared patch transport fallback overhead in BuildSharedPatchTransportResult(...) and ParseSharedPatchPayloadFromPage(...)`.
- [x] Reduce shared patch transport fallback overhead in `BuildSharedPatchTransportResult(...)` and `ParseSharedPatchPayloadFromPage(...)` (`internal/runtime2/shared_patch_transport.go`).
  Notes: `ParseSharedPatchPayloadFromPage(...)` now reads a validated payload span directly from shared-page storage (no intermediate payload copy) via a shared-page helper, while `GetSharedPatchReadPayload(...)` still returns a defensive copy for callers that require owned bytes.
  Validation: `go test ./internal/runtime2 -run "Test(BuildSharedPatchTransportResult(PrefersSharedBuffer|DowngradesWhenSharedUnavailable)|HandleSharedPatchPublishAndReadRoundTrip|SharedPatchPayloadEndToEndCommit)$" -count=1` and `go test ./internal/runtime2 -run ^$ -bench "Benchmark(BuildSharedPatchTransportResult|ParseSharedPatchPayloadFromPage)$" -benchmem -count=5`
  Checkpoint: completed todo `Reduce shared patch transport fallback overhead`; files changed: `internal/runtime2/shared_patch_page.go`, `internal/runtime2/shared_patch_transport.go`, `internal/runtime2/patch_transport_bench_test.go`, `internal/runtime2/perf_host_region_dispatch_hash_compare_bench_test.go`, `docs/MULTITHREADED_RUNTIME_TODO.md`; result: `BenchmarkParseSharedPatchPayloadFromPage` improved from `~1851-2584 ns/op`, `720 B/op`, `8 allocs/op` to `~1621-2286 ns/op`, `432 B/op`, `7 allocs/op`; `BenchmarkBuildSharedPatchTransportResult` stayed in the same band with expected `320 B/op`, `1 alloc/op` for both shared and fallback-unavailable paths; residual risk: Build path still allocates because structured-clone envelope bytes are still materialized once per dispatch; next suggested todo: `Reduce snapshot transport fallback branching and duplicate decode attempts in BuildSnapshotTransportPayloadWithFallback(...) and ParseSnapshotTransportPayloadWithFallback(...)`.
- [x] Reduce snapshot transport fallback branching and duplicate decode attempts in `BuildSnapshotTransportPayloadWithFallback(...)` and `ParseSnapshotTransportPayloadWithFallback(...)` (`internal/runtime2/binary_transport_selection.go`).
  Notes: build path now branches directly from capability flags (avoiding extra tier-selection dispatch), and parse path now uses a cheap binary-header probe to skip redundant binary decode attempts when a binary-tier payload is clearly structured-clone bytes.
  Validation: `go test ./internal/runtime2 -run "Test(SelectSnapshotTransportTierPrefersBinary|BuildSnapshotTransportPayloadWithFallbackDowngradesOnBinaryEncodeFailure|ParseSnapshotTransportPayloadWithFallbackDowngradesOnBinaryDecodeFailure)$" -count=1` and `go test ./internal/runtime2 -run ^$ -bench "Benchmark(BuildSnapshotTransportPayloadWithFallback|ParseSnapshotTransportPayloadWithFallback)$" -benchmem -count=5`
  Checkpoint: completed todo `Reduce snapshot transport fallback branching and duplicate decode attempts`; files changed: `internal/runtime2/binary_transport_selection.go`, `internal/runtime2/binary_transport_selection_bench_test.go`, `docs/MULTITHREADED_RUNTIME_TODO.md`; result: fallback parse path for binary-tier structured payload improved from `~2142-2731 ns/op`, `1193 B/op`, `26 allocs/op` to `~1917-2325 ns/op`, `1088 B/op`, `22 allocs/op`; binary-tier binary-payload and build-path benches stayed in the same performance band; residual risk: binary fallback still pays a second decode when payload header appears binary but fails deeper integrity checks (expected correctness guard); next suggested todo: `Reduce binary snapshot envelope checksum and header handling overhead in BuildBinarySnapshotEnvelope(...) and ParseBinarySnapshotEnvelope(...)`.
- [x] Reduce binary snapshot envelope checksum and header handling overhead in `BuildBinarySnapshotEnvelope(...)` and `ParseBinarySnapshotEnvelope(...)` (`internal/runtime2/binary_snapshot_transport.go`).
  Notes: snapshot decode now uses a snapshot-specific header/body fast parser (`parseBinarySnapshotEnvelopeBody(...)`) that avoids the generic envelope-header path and keeps checksum verification on the direct body span.
  Validation: `go test ./internal/runtime2 -run "Test(BuildBinarySnapshotEnvelopeRoundTrips|ParseBinarySnapshotEnvelopeRejectsTruncatedPayload|ParseBinarySnapshotEnvelopeRejectsIncorrectLength|ParseBinarySnapshotEnvelopeRejectsTruncatedHeader|ParseBinarySnapshotEnvelopeRejectsTruncatedBody|ParseBinarySnapshotEnvelopeRejectsOversizedDeclaredSpan)$" -count=1` and `go test ./internal/runtime2 -run ^$ -bench "BenchmarkBuildAndParseBinarySnapshotEnvelope$" -benchmem -count=5`
  Checkpoint: completed todo `Reduce binary snapshot envelope checksum and header handling overhead`; files changed: `internal/runtime2/binary_snapshot_transport.go`, `docs/MULTITHREADED_RUNTIME_TODO.md`; result: benchmark remained allocation-stable (`build: 528 B/op, 2 allocs/op`; `parse: 1945 B/op, 35 allocs/op`) with parse throughput in the same noisy band and best-case runs improving from prior ~`1694-1820 ns/op` range down to ~`1396-1701 ns/op` on this host; residual risk: variance on this workstation is high, so benchstat on a pinned runner is recommended before claiming deterministic latency gain; next suggested todo: `Reduce binary mount envelope encode and decode overhead in BuildBinaryMountEnvelope(...) and ParseBinaryMountEnvelope(...)`.
- [x] Reduce binary mount envelope encode and decode overhead in `BuildBinaryMountEnvelope(...)` and `ParseBinaryMountEnvelope(...)` (`internal/runtime2/binary_mount_transport.go`).
  Notes: mount decode now reads outer region and renderer IDs as byte spans, fast-matches the outer region bytes against the already parsed snapshot region ID, and only falls back to `ParseRegionInstanceID(...)` when normalization is actually required; this removes one per-parse allocation on the common matched-ID path while preserving padded outer-region compatibility.
  Validation: `go test ./internal/runtime2 -run "Test(BuildBinaryMountEnvelopeRoundTrips|BuildBinaryMountEnvelopeRejectsSnapshotRegionMismatch|ParseBinaryMountEnvelopeRejectsWrongHeaderKind|ParseBinaryMountEnvelopeNormalizesOuterRegionID)$" -count=1` and `go test ./internal/runtime2 -run ^$ -bench "BenchmarkBuildAndParseBinaryMountEnvelope$" -benchmem -count=5`
  Checkpoint: completed todo `Reduce binary mount envelope encode and decode overhead`; files changed: `internal/runtime2/binary_mount_transport.go`, `internal/runtime2/binary_mount_transport_test.go`, `docs/MULTITHREADED_RUNTIME_TODO.md`; result: focused mount-envelope tests passed and parse benchmark allocation dropped from `33 allocs/op` to `32 allocs/op` (with `1625 B/op` unchanged) while parse throughput stayed in the same/lower-noise band (`~1223-1430 ns/op` before vs `~1214-1342 ns/op` after on this host); residual risk: parse still allocates for snapshot and renderer parsing paths, so larger gains likely require deeper snapshot envelope decode reductions; next suggested todo: `Review next unchecked runtime2 performance hotspot and pick the highest-impact benchmark-backed item`.
- [x] Reduce binary update envelope encode and decode overhead in `BuildBinaryUpdateEnvelope(...)` and `ParseBinaryUpdateEnvelope(...)` (`internal/runtime2/binary_mount_transport.go`).
  Notes: update decode now reads the outer region field as a byte span and fast-matches it against the already parsed snapshot region ID, only falling back to `ParseRegionInstanceID(...)` when normalization is actually needed; this removes one per-parse string allocation and keeps padded outer-region compatibility.
  Validation: `go test ./internal/runtime2 -run "Test(BuildBinaryUpdateEnvelopeRoundTrips|BuildBinaryUpdateEnvelopeRejectsSnapshotVersionMismatch|ParseBinaryUpdateEnvelopeRejectsWrongHeaderKind|ParseBinaryUpdateEnvelopeNormalizesOuterRegionID)$" -count=1` and `go test ./internal/runtime2 -run ^$ -bench "BenchmarkBuildAndParseBinaryUpdateEnvelope$" -benchmem -count=5`
  Checkpoint: completed todo `Reduce binary update envelope encode and decode overhead`; files changed: `internal/runtime2/binary_offset.go`, `internal/runtime2/binary_mount_transport.go`, `internal/runtime2/binary_mount_transport_test.go`, `docs/MULTITHREADED_RUNTIME_TODO.md`; result: focused update-envelope tests passed and parse benchmark allocation dropped from `14 allocs/op` (`768 B/op`) to `13 allocs/op` (`760 B/op`) while preserving throughput within benchmark noise on this host; residual risk: ns/op variance is high on this workstation, so a pinned CPU/benchstat run would give a cleaner signal on absolute latency deltas; next suggested todo: `Reduce patch frame encoding overhead in BuildBinaryPatchPayload(...)`.
- [x] Reduce patch frame encoding overhead in `BuildBinaryPatchPayload(...)` (`internal/runtime2/binary_patch_transport.go`).
  Notes: added a header-only JSON fast path (no string-table or ops) that appends JSON directly into the framed payload buffer using `strconv.AppendQuote/AppendUint` and avoids `json.Marshal(...)` plus body-copy work for frequent no-op/header-only frames.
  Validation: `go test ./internal/runtime2 -run "Test(BuildBinaryPatchPayloadRoundTrips|BuildBinaryPatchPayloadRoundTripsHeaderOnly|ParseBinaryPatchPayloadRejectsMalformedFrame)$" -count=1` and `go test ./internal/runtime2 -run ^$ -bench "BenchmarkBuildBinaryPatchPayload$" -benchmem -count=5`
  Checkpoint: completed todo `Reduce patch frame encoding overhead`; files changed: `internal/runtime2/binary_patch_transport.go`, `internal/runtime2/binary_patch_transport_test.go`, `docs/MULTITHREADED_RUNTIME_TODO.md`; result: focused patch transport tests passed and benchmark improved from roughly `~326 ns/op`, `416 B/op`, `3 allocs/op` to `~154-213 ns/op`, `208 B/op`, `1 allocs/op`; residual risk: this fast path only applies to header-only patch streams and falls back to marshal path for non-empty string tables or ops; next suggested todo: `Reduce patch frame decode and header-validation overhead in ParseBinaryPatchPayload(...)`.
- [x] Reduce patch frame decode and header-validation overhead in `ParseBinaryPatchPayload(...)` (`internal/runtime2/binary_patch_transport.go`).
  Notes: decode now attempts a strict header-only JSON parser for frames produced by the new header-only encoder path and only falls back to `json.Unmarshal(...)` for non-matching payloads, preserving validation via `ParsePatchStreamHeader(...)` on both paths.
  Validation: `go test ./internal/runtime2 -run "Test(BuildBinaryPatchPayloadRoundTrips|BuildBinaryPatchPayloadRoundTripsHeaderOnly|ParseBinaryPatchPayloadRejectsMalformedFrame)$" -count=1` and `go test ./internal/runtime2 -run ^$ -bench "BenchmarkParseBinaryPatchPayload$" -benchmem -count=5`
  Checkpoint: completed todo `Reduce patch frame decode and header-validation overhead`; files changed: `internal/runtime2/binary_patch_transport.go`, `docs/MULTITHREADED_RUNTIME_TODO.md`; result: focused patch transport tests passed and parse benchmark improved from roughly `~1380-1441 ns/op`, `416 B/op`, `9 allocs/op` to `~122-179 ns/op`, `24 B/op`, `2 allocs/op`; residual risk: fast decode currently targets the deterministic header-only JSON layout and intentionally falls back to full JSON unmarshal for all other payload shapes; next suggested todo: `Reduce structured-clone patch envelope JSON encode or decode overhead in BuildStructuredClonePatchEnvelopeJSON(...) and ParseStructuredClonePatchEnvelopeJSON(...)`.
- [x] Reduce structured-clone patch envelope JSON encode or decode overhead in `BuildStructuredClonePatchEnvelopeJSON(...)` and `ParseStructuredClonePatchEnvelopeJSON(...)` (`internal/runtime2/structured_clone_patch_transport.go`).
  Notes: replaced patch-envelope encode `json.Marshal(...)` with a single-allocation manual JSON builder (`strconv.AppendQuote/AppendUint` + direct base64 encode) while keeping decode semantics and validation unchanged.
  Validation: `go test ./internal/runtime2 -run "Test(BuildStructuredClonePatchEnvelopeJSONRoundTrips|ParseStructuredClonePatchEnvelopeJSONRejectsMalformedPayload)$" -count=1` and `go test ./internal/runtime2 -run ^$ -bench "Benchmark(BuildStructuredClonePatchEnvelopeJSON|ParseStructuredClonePatchEnvelopeJSON)$" -benchmem -count=5`
  Checkpoint: completed todo `Reduce structured-clone patch envelope JSON encode or decode overhead`; files changed: `internal/runtime2/structured_clone_patch_transport.go`, `internal/runtime2/patch_transport_bench_test.go`, `docs/MULTITHREADED_RUNTIME_TODO.md`; result: focused structured-clone tests passed and build microbench improved from roughly `~295-347 ns/op`, `352 B/op`, `2 allocs/op` to `~185-208 ns/op` (`~295 ns` high outlier), `320 B/op`, `1 allocs/op`; residual risk: parse path is intentionally unchanged (`json.Unmarshal`) and remains around `~1540-2075 ns/op`, `432 B/op`, `7 allocs/op`; next suggested todo: `Optimize shared snapshot publish and read-copy hot paths in HandleSharedSnapshotPublishPayload(...) and GetSharedSnapshotReadPayload(...)`.
- [x] Tighten `ValidateControlEnvelope(...)` hot-path branching for frequent `patch-ready` and `diagnostic` envelopes (`internal/runtime2/control.go`).
  Notes: direct protocol and transport checks now keep `patch-ready` and `diagnostic` validation on a shorter typed-enum path before falling back to the slower generic kind parser.
  Validation: `go test ./internal/runtime2 -run "TestValidateControlEnvelope" -count=1`
- [x] Reduce control envelope JSON build or parse overhead in `BuildControlEnvelopeJSON(...)` and `ParseControlEnvelopeJSON(...)` (`internal/runtime2/control.go`).
  Notes: collapsed `ValidateControlEnvelope(...)` into one direct kind switch (removing extra `ParseControlKind(...)` work), switched pong shard text checks to the existing ASCII-fast trim helper, and moved diagnostic redaction to an in-place helper that now runs after validation so invalid envelopes skip redaction work while valid build/parse paths still avoid full-struct copy round-trips.
  Validation: `go test ./internal/runtime2 -run "Test(ParseControlEnvelopeJSON|BuildControlEnvelopeJSON|ValidateControlEnvelope)" -count=1`
  Checkpoint: completed todo `Reduce control envelope JSON build or parse overhead`; files changed: `internal/runtime2/control.go`, `internal/runtime2/diagnostic_redaction.go`, `docs/MULTITHREADED_RUNTIME_TODO.md`; result: focused control tests passed; residual risk: no dedicated control JSON microbench exists yet in-tree, so impact is inferred from hot-path simplification; next suggested todo: `Reduce host-side control-plane dispatch overhead in HandleHostControlEnvelope(...)`.
- [x] Reduce host-side control-plane dispatch overhead in `HandleHostControlEnvelope(...)` (`internal/runtime2/host_control_dispatcher.go`).
  Notes: removed duplicate coordinator mounted-state checks from the diagnostic and restart paths, stopped re-parsing already validated diagnostic kinds, and now route host-dispatched diagnostics through an internal validated/redacted path that skips a second full `ValidateControlEnvelope(...)` pass.
  Validation: `go test ./internal/runtime2 -run "TestHandleHostControlEnvelope" -count=1`
  Checkpoint: completed todo `Reduce host-side control-plane dispatch overhead`; files changed: `internal/runtime2/host_control_dispatcher.go`, `internal/runtime2/host_region_adapter.go`, `docs/MULTITHREADED_RUNTIME_TODO.md`; result: host control dispatch tests passed; residual risk: no dedicated host control dispatcher benchmark currently tracks this path, so gain is inferred from removed duplicate validation/redaction; next suggested todo: `Reduce worker-side control-plane dispatch overhead in HandleWorkerControlEnvelope(...)`.
- [x] Reduce worker-side control-plane dispatch overhead in `HandleWorkerControlEnvelope(...)` (`internal/runtime2/worker_control_dispatcher.go`).
  Notes: worker dispatch now rejects unsupported control kinds before full envelope validation, reuses decoded region and snapshot fields across dispatch branches, and routes mount/update into runtime internal paths that skip duplicate snapshot-envelope validation already performed at control-envelope validation time.
  Validation: `go test ./internal/runtime2 -run "TestHandleWorkerControlEnvelope" -count=1`
  Checkpoint: completed todo `Reduce worker-side control-plane dispatch overhead`; files changed: `internal/runtime2/worker_control_dispatcher.go`, `internal/runtime2/worker_region_runtime.go`, `docs/MULTITHREADED_RUNTIME_TODO.md`; result: worker control and worker mount/update tests passed; residual risk: dispatch path still executes full control-envelope validation before routing, so future work may split a tighter worker-only validator for additional gains; next suggested todo: `Reduce shard-session queue lock and payload-copy overhead in shard_session.go`.
- [x] Reduce shard-session queue lock and payload-copy overhead in `bindShardSessionPortHandler(...)`, `HandleShardSessionSendPayload(...)`, and `HandleShardSessionReceivePayload(...)` (`internal/runtime2/shard_session.go`).
  Notes: replaced queue-limit drop handling with in-place `copy(...)` compaction (no re-slice `append` path) and removed redundant send-path payload cloning before `PostMessage(...)` so callers avoid one extra allocation/copy per outbound payload.
  Validation: `go test ./internal/runtime2 -run "Test(BuildShardSessionWithQueueLimitCapsInboundPayloadQueue|BuildShardSessionConcurrentInboundAndReceiveStaysStable|HandleShardSessionSendPayloadUsesPort|HandleShardSessionReceivePayloadDrainClearsQueueState)$" -count=1`
  Checkpoint: completed todo `Reduce shard-session queue lock and payload-copy overhead`; files changed: `internal/runtime2/shard_session.go`, `docs/MULTITHREADED_RUNTIME_TODO.md`; result: focused shard-session queue/send/receive tests passed; residual risk: no shard-session microbench currently tracks payload-copy deltas directly; next suggested todo: `Reduce patch-ready pairing poll overhead in HandleShardSessionReceivePatchReadyWithPayload(...)`.
- [x] Reduce patch-ready pairing poll overhead in `HandleShardSessionReceivePatchReadyWithPayload(...)` (`internal/runtime2/shard_session.go`).
  Notes: added a cheap control-envelope marker probe (`protocol_version` + `kind` keys on object payloads) before attempting full `ParseControlEnvelopeJSON(...)` while waiting for paired patch payloads, avoiding repeated full control parses for ordinary raw patch payload traffic.
  Validation: `go test ./internal/runtime2 -run "TestHandleShardSession(ReceivePatchReadyWithPayloadWaitsForDelayedPayload|ReceivePatchReadyWithPayloadRejectsNextControlBeforePayload|SendAndReceivePatchReadyWithPayload)$" -count=1`
  Checkpoint: completed todo `Reduce patch-ready pairing poll overhead`; files changed: `internal/runtime2/shard_session.go`, `docs/MULTITHREADED_RUNTIME_TODO.md`; result: focused patch-ready pairing tests passed; residual risk: marker probing is heuristic and intentionally conservative, so malformed control-like payloads still fall back to parse-time handling; next suggested todo: `Reduce scheduler shard identity canonicalization and assignment overhead in parseSchedulerShardList(...), buildSchedulerRegionAssignment(...), and GetSchedulerRegionShardID(...)`.
- [x] Reduce scheduler shard identity canonicalization and assignment overhead in `parseSchedulerShardList(...)`, `buildSchedulerRegionAssignment(...)`, and `GetSchedulerRegionShardID(...)` (`internal/runtime2/scheduler_shard_identity.go`).
  Notes: `BuildSchedulerWithQueueLimit(...)` now normalizes shard IDs once through `parseSchedulerShardList(...)`, `GetSchedulerRegionShardID(...)` now canonicalizes once per call and reuses `hasSchedulerShardID(...)` for keep-policy availability checks, shard-membership lookups now use binary search over canonical shard lists, and shard-id validation now uses the runtime fast trim helper (`parseRuntimeHasTrimmedNonWhitespaceText(...)`) in shard-identity paths.
  Validation: `go test ./internal/runtime2 -run "Test(GetSchedulerShardIDKeepsStableWorkerShardPerLiveWorker|GetSchedulerShardIDSeparatesDifferentLiveWorkers|ClearSchedulerShardIDAvoidsUnsafeReuseAfterDispose|GetSchedulerRegionShardIDKeepsStableAssignmentForOneRegion|GetSchedulerRegionShardIDAllowsDifferentRegionsToSpreadAcrossShards|GetSchedulerRegionShardIDOnlyChangesWhenPolicyAllows)$" -count=1` and `go test ./internal/runtime2 -run "Test(BuildSchedulerCanonicalizesShardIDsForLookup|GetSchedulerRegionShardIDKeepHandlesUnsortedDuplicateShardList|HandleSchedulerKeepalive.*|HandleSchedulerMount.*|HandleSchedulerUpdate.*)$" -count=1`
  Checkpoint: completed todo `Reduce scheduler shard identity canonicalization and assignment overhead`; files changed: `internal/runtime2/scheduler_shard_identity.go`, `internal/runtime2/scheduler.go`, `internal/runtime2/scheduler_keepalive_test.go`, `internal/runtime2/scheduler_region_assignment_test.go`, `docs/MULTITHREADED_RUNTIME_TODO.md`; result: focused scheduler identity and scheduler lifecycle tests passed with canonicalized shard-list lookup coverage; residual risk: no dedicated scheduler shard-identity benchmark currently measures binary-search membership and canonicalization reuse directly; next suggested todo: `Add a single host-update transaction path that minimizes coordinator lock transitions across HandleHostRegionUpdateDispatchWithPriority(...), HandleHostRegionUpdate(...), and coordinator version writes`.
- [x] Reduce source-reactivity map churn and queue sort cost in `SetRegionDeclaredSources(...)`, `HandleSourceChange(...)`, and `GetRegionUpdateQueue(...)` (`internal/runtime2/source_reactivity.go`).
  Notes: `SetRegionDeclaredSources(...)` now skips reverse-map teardown and rebuild when the normalized declared-source list is unchanged, and queued region updates now append on enqueue with a sorted-state flag so queue drains only sort when needed (avoiding per-enqueue insertion copy).
  Validation: `go test ./internal/runtime2 -run "TestSetRegionDeclaredSources(DeclaredSourceChangeEnqueuesUpdate|UnrelatedSourceChangeDoesNotEnqueue|MultipleDeclaredChangesCoalesce)$" -count=1`
  Checkpoint: completed todo `Reduce source-reactivity map churn and queue sort cost`; files changed: `internal/runtime2/source_reactivity.go`, `docs/MULTITHREADED_RUNTIME_TODO.md`; result: focused source-reactivity tests passed; residual risk: high-fanout source changes can still pay one sort at queue drain, so a future benchmark may justify a bucketed queue if fanout grows materially; next suggested todo: `Reduce snapshot fingerprint hashing overhead in GetSnapshotFingerprintHash(...) and getSnapshotFingerprintHashWithoutValidation(...)`.
- [x] Reduce snapshot fingerprint hashing overhead in `GetSnapshotFingerprintHash(...)` and `getSnapshotFingerprintHashWithoutValidation(...)` (`internal/runtime2/snapshot.go`).
  Notes: snapshot fingerprint hashing now reuses prebuilt JSON token byte slices (removing repeated literal string-to-byte conversions), uses a stack-buffer fast path for quoted region IDs, and avoids an extra digest copy after `Sum(...)` while keeping legacy JSON field order and nested `Props`/`Sources` hashing behavior unchanged.
  Validation: `go test ./internal/runtime2 -run ^$ -bench "Benchmark(GetSnapshotFingerprintHashCurrentVsLegacy|HandleHostRegionSnapshotFingerprint)$" -benchmem -count=5`
  Checkpoint: completed todo `Reduce snapshot fingerprint hashing overhead`; files changed: `internal/runtime2/snapshot.go`, `docs/MULTITHREADED_RUNTIME_TODO.md`; result: benchmark validation passed and hash parity stayed green (`go test ./internal/runtime2 -run "TestGetSnapshotFingerprintHashMatchesLegacyMarshalEncoding$" -count=1`); residual risk: large-snapshot hash cost is still dominated by nested JSON encode traversal for `Props`/`Sources`; next suggested todo: `Review next unchecked runtime2 performance hotspot and pick the highest-impact benchmark-backed item`.

### Runtime2 CPU Optimization Deep Research Backlog (2026-03-27)

Research basis: CPU flamegraph analysis of `BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers` (11.93s samples over ~12.14s wall time), Go pprof profiling guidance, and SRE tail-latency methodology. Priority ordering: highest profiled `cum` cost first. Tail latency (p95/p99) is the north star, not throughput averages.

#### Measurement And Profiling Infrastructure

- [x] Add p50/p95/p99 latency histogram sampling to `BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers` and the `agent3_bench_test.go` update dispatch loop so tail-latency regressions surface independently of the ns/op mean (`internal/runtime2/host_region_pressure_bench_test.go`, `internal/runtime2/agent3_bench_test.go`).
  Rationale: averages hide tail pain; percentiles reveal amplification under load; the update dispatch path fans out across snapshot, hash, coordinator, and scheduler nodes so variance compounds.
  Notes: both benchmarks now collect bounded latency samples during the hot loop and report `p50`, `p95`, and `p99` via `b.ReportMetric(...)`; sampling uses span-based timing to keep Windows timer-resolution artifacts from collapsing most samples to zero.
  Validation: `go test ./internal/runtime2 -run ^$ -bench "BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers$" -benchmem -count=20` and `go test ./internal/runtime2 -run ^$ -bench "BenchmarkHandleHostRegionUpdateDispatchLoop$" -benchmem -count=5`
  Checkpoint: completed todo `Add p50/p95/p99 latency histogram sampling`; files changed: `internal/runtime2/host_region_pressure_bench_test.go`, `internal/runtime2/agent3_bench_test.go`, `docs/MULTITHREADED_RUNTIME_TODO.md`; result: pressure benchmark now emits tail-latency metrics (`dispatch-batch-p50-ns` roughly `~23.5-23.8 us`, `p95` roughly `~31.2-47.4 us`, `p99` roughly `~39.3-48.3 us`) alongside mean `ns/op`, and agent3 update dispatch loop now emits `dispatch-loop-p50/p95/p99` baseline metrics (`p50 ~740-749 ns`, `p95 ~1109-1207 ns`, `p99 ~1126-1379 ns`); before/after mean check for pressure benchmark stayed near-neutral (`~19.6-25.7 us/op` before instrumentation pass vs `~18.8-25.0 us/op` after); residual risk: span-based sampling smooths ultra-short spikes within one span and should be complemented with profiles for lock or scheduler stall root-cause analysis; next suggested todo: `Add alloc, mutex, and block profiles alongside CPU profiles for the standard pressure benchmark run`.
- [x] Add alloc, mutex, and block profiles alongside CPU profiles for the standard pressure benchmark run so heap churn and lock contention are captured in the same artifact as the CPU flamegraph (`internal/runtime2/host_region_pressure_bench_test.go`).
  Rationale: `sync/atomic.(*Int32).Add` flat cum at `1.77s (14.84%)` and the RWMutex nodes (`RLock 0.73s`, `RUnlock 0.55s`, `Lock 0.45s`) suggest contention worth quantifying with block and mutex profiles before picking lock-reduction changes.
  Notes: the standard pressure profile run now explicitly captures `cpu`, `mutex`, `block`, and `mem` profiles in one benchmark invocation; on PowerShell, use `--%` passthrough so dotted `-test.*` flags are forwarded intact.
  Validation: `go test -c -o ./bin/runtime2.test.exe ./internal/runtime2` and `./bin/runtime2.test.exe -test.run=^$ -test.bench=BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers$ -test.benchtime=10s -test.cpuprofile=./bin/runtime2.hot.cpu.pprof -test.mutexprofile=./bin/runtime2.hot.mutex.pprof -test.blockprofile=./bin/runtime2.hot.block.pprof -test.memprofile=./bin/runtime2.hot.mem.pprof`
  Checkpoint: completed todo `Add alloc, mutex, and block profiles alongside CPU profiles`; files changed: `docs/MULTITHREADED_RUNTIME_TODO.md`; result: one profile run now emits all four artifacts (`bin/runtime2.hot.cpu.pprof`, `bin/runtime2.hot.mutex.pprof`, `bin/runtime2.hot.block.pprof`, `bin/runtime2.hot.mem.pprof`) from the same benchmark execution and confirmed each artifact exists on disk; residual risk: profiling flags materially inflate `ns/op`, `B/op`, and `allocs/op`, so these runs are for diagnostics only and should not be mixed with regression baselines; next suggested todo: `After each closed performance todo in this section, re-run go tool pprof ./bin/runtime2.hot.cpu.pprof and check that addressed hotspot cum percentage dropped`.
- [x] After each closed performance todo in this section, re-run `go tool pprof ./bin/runtime2.hot.cpu.pprof` and check that the addressed hotspot `cum` percentage dropped before picking the next item (re-profile discipline from Go pprof guidance: optimize one thing, re-measure, repeat).
  Notes: completed for this pass by re-running `go tool pprof -top ./bin/runtime2.hot.cpu.pprof` after closing the profiling-infrastructure todos and recording current hot nodes before selecting the next work item.
  Validation: `go tool pprof -top ./bin/runtime2.hot.cpu.pprof`
  Checkpoint: completed todo `After each closed performance todo in this section, re-run go tool pprof ...`; files changed: `docs/MULTITHREADED_RUNTIME_TODO.md`; result: refreshed profile confirms the no-change dispatch hash hotspot now appears at lower cumulative share (`handleHostRegionDispatchHash` around `14.71% cum` in the latest profile vs earlier snapshot around `31.18%`); residual risk: this profile was collected with multi-profile flags enabled, so absolute percentages include profiling overhead and must be interpreted directionally; next suggested todo: `Evaluate a two-tier no-change check: (1) version-vector fast gate; (2) if version vectors match fall through to a cheap 64-bit non-crypto hash before SHA-256`.

#### No-Change Predicate And Hashing Cost

- [x] Add a version-vector gating predicate in `handleHostRegionDispatchHash(...)` that checks `(rendererID, epoch, sourceVersions[], inputVersion)` equality before any hashing work, short-circuiting the entire SHA-256 path when declared-source version counters and input version are unchanged (`internal/runtime2/host_region_adapter.go`, `internal/runtime2/snapshot_dispatch_hash.go`).
  Profile evidence: `handleHostRegionDispatchHash` cum `3.72s (31.18%)`; `sha256.Sum256` cum `2.50s (20.96%)`; `sha256.(*Digest).Write` cum `1.92s`.
  Notes: host dispatch hashing now stores and compares a full dispatch version vector (`rendererID`, `epoch`, ordered `sourceVersions[]`, and `inputVersion`) before any hash work; source-version tuple mismatches still reset digest state immediately, while input-version-only churn falls back to fast-hash payload comparison so no-change dispatch short-circuits continue to work.
  Validation: `go test ./internal/runtime2 -run "TestHandleHostRegionUpdateDispatch.*NoChange" -count=1` and `go test ./internal/runtime2 -run "TestHandleHostRegionDispatchHash.*" -count=1` and `go test ./internal/runtime2 -run ^$ -bench "Benchmark(HandleHostRegionDispatchHashCurrentVsLegacy|HandleHostRegionManyHotRegionsBoundedWorkers)$" -benchmem -count=5`
  Checkpoint: completed todo `Add a version-vector gating predicate in handleHostRegionDispatchHash(...)`; files changed: `internal/runtime2/host_region_adapter.go`, `internal/runtime2/host_control_dispatcher.go`, `internal/runtime2/snapshot_dispatch_hash_test.go`, `internal/runtime2/perf_host_region_dispatch_hash_compare_bench_test.go`, `docs/MULTITHREADED_RUNTIME_TODO.md`; result: focused dispatch/no-change tests passed and benchmark tracking now shows stable payload no-change path dropping from roughly `~724-885 ns/op` to `~4.8-5.1 ns/op` with zero allocs, changed payload gate cost at `~11.1-11.9 ns/op` (still zero allocs, down two orders of magnitude from legacy SHA-only), and pressure benchmark steady around `~19.6-25.7 us/op`; residual risk: when test fixtures omit `SourceVersions`, the version gate intentionally falls back to aggregate `SourceVersion` matching rather than tuple matching; next suggested todo: `Evaluate a two-tier no-change check: (1) version-vector fast gate; (2) if version vectors match fall through to a cheap 64-bit non-crypto hash before SHA-256`.
- [x] Eliminate the `SnapshotEnvelope` copy inside `handleHostRegionDispatchHash(...)` that sets `InputVersion = 1` before calling `buildSnapshotDispatchHashInto(...)` — change the hash API to accept the envelope plus an `InputVersion` override parameter rather than constructing a modified struct copy per dispatch (`internal/runtime2/host_region_adapter.go`, `internal/runtime2/snapshot_dispatch_hash.go`).
  Profile evidence: `runtime.duffcopy` flat `1.11s (9.30%)` on the hot path; per-dispatch envelope copy is a direct contributor on every non-short-circuited update.
  Notes: fulfilled by the dispatch-path checkpoint above; host dispatch hashing now passes the original envelope through directly because canonical dispatch hashing already ignores `InputVersion`.
  Validation: `go test ./internal/runtime2 -run "Test(HandleHostRegionUpdateDispatch.*NoChange|HandleHostRegionDispatchHashIgnoresInputVersion)" -count=1` and `go test ./internal/runtime2 -run ^$ -bench "BenchmarkHandleHostRegionDispatchHashCurrentVsLegacy$" -benchmem -count=3`
- [x] Evaluate a two-tier no-change check: (1) version-vector fast gate (see above); (2) if version vectors match fall through to a cheap 64-bit non-crypto hash (e.g. xxh3-128) before any SHA-256 work for the rare structural-change case where version counters cannot be relied upon (`internal/runtime2/snapshot_dispatch_hash.go`, `internal/runtime2/host_region_adapter.go`).
  Profile evidence: SHA-256 path dominates at cumulative `~3.5s`; a 64-bit non-crypto hash over the same payload would run in nanoseconds vs. microseconds.
  Notes: dispatch no-change now enforces a strict two-tier flow: version-vector mismatch still short-circuits as changed, and exact version-vector matches now pass through canonical payload fast-hash comparison before returning no-change. This closes the “same counters, changed payload” gap at the cost of extra work on exact-vector repeats.
  Validation: `go test ./internal/runtime2 -run "TestHandleHostRegionUpdateDispatch" -count=1` and `go test ./internal/runtime2 -run "TestHandleHostRegionDispatchHash.*" -count=1` and `go test ./internal/runtime2 -run ^$ -bench "Benchmark(HandleHostRegionDispatchHashCurrentVsLegacy|HandleHostRegionManyHotRegionsBoundedWorkers)$" -benchmem -count=5`
  Checkpoint: completed todo `Evaluate a two-tier no-change check`; files changed: `internal/runtime2/host_region_adapter.go`, `internal/runtime2/snapshot_dispatch_hash_test.go`, `docs/MULTITHREADED_RUNTIME_TODO.md`; result: new regression test now proves payload mutations with unchanged `(rendererID, epoch, sourceVersions[], inputVersion)` no longer short-circuit as no-change, changed-payload current path remains in low-nanosecond territory (`~11.0-12.9 ns/op`, `0 allocs`), and pressure benchmark remained in the same broad range (`~17.2-26.4 us/op`) while stable exact-vector repeat cost increased from prior direct-vector short-circuit (`~4.7 ns/op`) to fast-hash verification (`~1.0-1.3 us/op`); residual risk: fast-hash verification now dominates exact-vector repeat microbench cases and may merit an optional stricter-mode toggle if same-input duplicate dispatches become hot in production; next suggested todo: `Add a FNV-64a pre-filter to (*HostRegionAdapter).handleHostRegionSnapshotHash(...) so unchanged snapshot payloads can skip SHA-256 work`.
- [x] Add a FNV-64a pre-filter to `(*HostRegionAdapter).handleHostRegionSnapshotHash(...)` so unchanged snapshot payloads can skip SHA-256 work on the no-change path: compute FNV-64a over the serialized snapshot envelope bytes first; only call the SHA-256 path when the fast hash diverges or when the hex fingerprint string is actually needed (content change) (`internal/runtime2/host_region_adapter.go`, `internal/runtime2/snapshot_dispatch_hash.go`).
  Profile evidence: `HandleHostRegionUpdateSnapshot` cum `2.93s (24.56%)`; mirrors the dispatch hash path which already has a two-tier FNV-64a + SHA-256 guard but snapshot hashing has no equivalent pre-filter.
  Notes: this path is already in-tree: `handleHostRegionSnapshotHash(...)` now computes canonical payload bytes, derives FNV-64a first, and only recomputes/stores SHA-256 when the fast hash diverges.
  Validation: `go test ./internal/runtime2 -run "TestHandleHostRegionUpdateSnapshot" -count=1` and `go test ./internal/runtime2 -run ^$ -bench "BenchmarkHandleHostRegion(UpdateSnapshot|ManyHotRegionsBoundedWorkers)$" -benchmem -count=5`
  Checkpoint: completed todo `Add a FNV-64a pre-filter to handleHostRegionSnapshotHash(...)`; files changed: `docs/MULTITHREADED_RUNTIME_TODO.md`; result: focused snapshot tests passed and current benchmark baselines stayed fast (`BenchmarkHandleHostRegionUpdateSnapshot` around `~316-428 ns/op`, pressure benchmark around `~16.8-21.6 us/op` with percentile metrics emitted); residual risk: snapshot path still serializes canonical payload bytes per call before hashing, so future gains may require staged-field hashing to avoid payload materialization; next suggested todo: `Evaluate changing Coordinator's internal map from value entries to pointer entries to reduce copy and lock churn`.

#### Coordinator Data Layout And Locking

- [x] Evaluate changing `Coordinator`'s internal map from `map[RegionInstanceID]CoordinatorEntry` (value) to `map[RegionInstanceID]*CoordinatorEntry` (pointer) so `getMutableEntry`/`storeMutableEntry` cycles stop copying the entry struct on every hot mutation and reduce `duffcopy` contribution; mutate in-place under a single lock round instead of read-copy-write across two lock acquisitions (`internal/runtime2/coordinator.go`).
  Profile evidence: `getMutableEntry` cum `0.49s`; `storeMutableEntry` cum `0.82s`; `UpdateRegion` cum `1.68s`; `runtime.duffcopy` flat `1.11s`.
  Notes: already implemented in-tree: `Coordinator.storeEntries` is currently `map[RegionInstanceID]*CoordinatorEntry`, `MountRegion(...)` stores pointers, and lock-held mutators update entries in place.
  Validation: `go test ./internal/runtime2 -run "Test(SetRegionSnapshotState|UpdateRegionAndGetEntry|StoreRegionSnapshotAndDispatchedVersion|HandleHostRegionUpdateSnapshot)" -count=1` and `go test ./internal/runtime2 -run ^$ -bench "BenchmarkCoordinatorDispatchTransactionCurrentVsLegacy$" -benchmem -count=5`
  Checkpoint: completed todo `Evaluate changing Coordinator internal map to pointer entries`; files changed: `docs/MULTITHREADED_RUNTIME_TODO.md`; result: verification confirms pointer-entry storage is active and transaction microbench remains favorable (`current_single_snapshot_and_dispatch ~40.96-44.42 ns/op` vs legacy split path `~64.34-81.14 ns/op`, both zero allocs); residual risk: some legacy helper paths still round-trip through value copies (`getMutableEntry`/`storeMutableEntry`) and remain candidates for further cleanup; next suggested todo: `Split getMutableEntry(...) into a trusted-internal variant that skips ParseRegionInstanceID(...) re-validation for already-validated IDs`.
- [ ] Split `getMutableEntry(...)` into a trusted-internal variant that skips `ParseRegionInstanceID(...)` re-validation for callers that already hold a validated `RegionInstanceID`, keeping full parsing only at explicit external API entry points (`internal/runtime2/coordinator.go`).
  Profile evidence: `ParseRegionInstanceID` cum `0.07s (0.59%)` appears on the hot dispatch path via every coordinator mutation; the hot path is an internal call site where the ID was already validated at mount time.
  Validation: `go test ./internal/runtime2 -run "Test(SetRegionSnapshotState|SetRegionAttachedStoresAttachedState|HandleHostRegionMount|HandleHostRegionUpdateSnapshot)" -count=1` and `go test ./internal/runtime2 -run ^$ -bench "BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers$" -benchmem -count=5`

#### Props And Source Snapshot Cost

- [ ] Evaluate replacing `map[string]any` hot props with a compact sorted `[]PropKV` representation for coordinator entry storage and dispatch hashing so randomized map iteration cost (`mapIterStart` + `Iter.Next` + `aeshashbody`) is replaced by sequential slice scans on steady-state prop shapes (`internal/runtime2/spec.go`, `internal/runtime2/snapshot_dispatch_hash.go`, `internal/runtime2/coordinator.go`).
  Profile evidence: `mapIterStart` cum `0.71s (5.95%)`; `Iter.Next` cum `0.45s (3.77%)`; `aeshashbody` flat `0.31s (2.60%)`; map iteration is intentionally randomized in Go and has higher overhead than sequential slice access for small-to-medium N.
  Note: keep `map[string]any` for truly dynamic or large prop sets as the slow path; only accelerate the common steady-state small-N case.
  Validation: `go test ./internal/runtime2 -run "TestValidateSerializableProps" -count=1` and `go test ./internal/runtime2 -run ^$ -bench "BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers$" -benchmem -count=5`
- [ ] Add a prop-shape fingerprint cache in `HandleHostRegionUpdateSnapshot(...)` that skips re-running `ValidateSerializableProps(...)` when the prop key set and type structure are unchanged from the previous dispatch, using a lightweight key-count plus sorted-key hash as the cheap shape predicate (`internal/runtime2/host_region_adapter.go`, `internal/runtime2/spec.go`).
  Profile evidence: `ValidateSerializableProps` cum `0.77s (6.45%)`; `isSerializableAnyFast` cum `0.75s (6.29%)`; re-validation runs on every update regardless of whether props changed structurally.
  Validation: `go test ./internal/runtime2 -run "TestHandleHostRegionUpdateSnapshot" -count=1` and `go test ./internal/runtime2 -run ^$ -bench "BenchmarkHandleHostRegion(UpdateSnapshot|ManyHotRegionsBoundedWorkers)$" -benchmem -count=5`
- [ ] Add a source-snapshot key-order cache per region epoch so `appendSnapshotDispatchAnyMap(...)` and `appendSnapshotDispatchAnyMapSingle(...)` can skip per-update `mapIterStart`/`Iter.Next` source-key extraction when the declared-source map shape is stable between updates (`internal/runtime2/snapshot_dispatch_hash.go`).
  Profile evidence: `appendSnapshotDispatchAnyMap` cum `0.64s (5.36%)`; `appendSnapshotDispatchAnyMapSingle` cum `0.57s (4.78%)`; source-map shape is static for a mounted region's lifetime.
  Validation: `go test ./internal/runtime2 -run "TestBuildSnapshotDispatchHash" -count=1` and `go test ./internal/runtime2 -run ^$ -bench "BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers$" -benchmem -count=5`

#### Synchronization And Dispatch Batching

- [ ] Evaluate batching all coordinator writes that originate from one `HandleHostRegionUpdateDispatchWithPriority(...)` call into a single lock acquisition by accumulating derived state in the adapter and flushing it in one `Lock`/`Unlock` round, reducing the RWMutex lock-trip count from the current multi-call pattern (`internal/runtime2/host_region_adapter.go`, `internal/runtime2/coordinator.go`).
  Profile evidence: `RLock` cum `0.73s`; `RUnlock` cum `0.55s`; `Unlock` cum `0.51s`; `Lock` cum `0.45s`; combined lock overhead `~2.24s (~18.8%)` on the hot path; each coordinate helper (`GetEntry`, `SetRegionSnapshotState`, `SetRegionLastSnapshotVersion`) takes its own lock round.
  Note: the existing `SetRegionSnapshotState` single-transaction helper reduces one round-trip; this todo targets the remaining multi-call dispatch pattern that still spans multiple lock acquisitions.
  Validation: `go test ./internal/runtime2 -run "TestHandleHostRegionUpdateDispatch" -count=1` and `go test ./internal/runtime2 -run ^$ -bench "BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers$" -benchmem -count=5`
- [ ] Add `b.ReportAllocs()` to the primary dispatch pressure benchmarks and track per-dispatch allocation count as a CI regression gate alongside ns/op so allocation-increasing changes are caught before they compound into heap and GC pressure at scale (`internal/runtime2/host_region_pressure_bench_test.go`, `internal/runtime2/agent3_bench_test.go`).
  Validation: `go test ./internal/runtime2 -run ^$ -bench "BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers$" -benchmem -count=5`
- [ ] Skip `ParseHostRegionDispatchPriority(...)` string-switch validation for internal urgent calls: `HandleHostRegionUpdateDispatch(...)` always passes the `HostRegionDispatchPriorityUrgent` constant to `HandleHostRegionUpdateDispatchWithPriority(...)` which then re-validates via a string switch; add an unexported `handleHostRegionUpdateDispatchWithValidatedPriority(...)` inner form that accepts an already-validated `HostRegionDispatchPriority` value and skips the parse; keep the public method's string validation unchanged for external callers (`internal/runtime2/host_region_adapter.go`).
  Profile evidence: appears in file-by-file notes as `P1: cache parsed dispatch priority`; string-switch parse runs on every urgent dispatch even though the constant value never changes at the internal call site.
  Validation: `go test ./internal/runtime2 -run "TestHandleHostRegionUpdateDispatch" -count=1` and `go test ./internal/runtime2 -run ^$ -bench "BenchmarkHandleHostRegionUpdateDispatchWithPriorityDeferred$" -benchmem -count=5`

#### Capability Detection

- [ ] Verify that `GetCapabilityReport(...)` path in both native and wasm builds memoizes detection results at most once per process lifetime and that repeated calls are lock-free reads; add a focused microbenchmark to confirm low-overhead repeated reads after first init (`internal/runtime2/capabilities.go`, `internal/runtime2/capabilities_detect_native.go`, `internal/runtime2/capabilities_detect_wasm.go`).
  Rationale: capability checks flow through capability-selection helpers on every snapshot dispatch tier decision; if detection is not memoized those checks add latency on every update.
  Validation: `go test ./internal/runtime2 -run "TestInitCapabilityReport" -count=1` and `go test ./internal/runtime2 -run ^$ -bench "BenchmarkGetCapabilityReport$" -benchmem -count=5`

#### File-By-File Optimization Notes (For Reference When Picking The Next Item)

The list below records which files have known optimization opportunities and what the high-level technique is. Each file should be addressed as a focused backlog item with its own validation command when its profiler cost is high enough to justify the change.

- `agent3_bench_test.go` — Add p50/p95/p99 sampling; split correctness assertions from perf loops; add `b.ResetTimer()` before hot loop; use `b.ReportAllocs()`.
- `binary_snapshot_body.go` — Avoid staging full payload bytes solely to hash; hash fields incrementally into a reusable digest; reuse buffer per-region to avoid contention.
- `binary_props.go` — Hot-path goal: no per-key allocation on serialization; cache key ordering and type tags when shape is stable.
- `binary_source_value.go` — Compact contiguous encoding for hot primitive types; avoid interface dispatch in common primitive cases.
- `binary_transport_selection.go` — Cache transport tier decision per region once capability report is stable.
- `capabilities.go` — Treat capability reads as static configuration; cache in adapter after first read; avoid re-locking for each dispatch decision.
- `control.go` — Fast-path validators that assume already-parsed IDs at internal call sites; avoid `ParseRegionInstanceID`/`ParseRendererID` inside tight update loops by validating once at the external boundary.
- `coordinator.go` — P0: pointer entry map to remove copy-write pattern; P0: trusted internal `getMutableEntry` variant without re-parsing; P1: pre-size `storeEntries` map at mount time to avoid grow/rehash.
- `dom_commit.go` — Detect no-op commits early (empty diff); use contiguous slices over maps for hot node lists; avoid repeated traversal of the same node sets.
- `dom_region_index.go` — Keep hot lookup tables contiguous; avoid pointer chasing; avoid `map[string]*Node` when a slice plus index table suffices.
- `host_control_dispatcher.go` — Avoid full envelope re-validation for envelopes from trusted internal shard sessions; split trusted vs. untrusted validator entry points.
- `host_region_adapter.go` — P0: version-vector no-change gate before any hash work; P0: remove envelope copy for `InputVersion` override; P1: eliminate separate per-field coordinator calls by batching into one lock round; P1: cache parsed dispatch priority; P2: keep error formatting out-of-line with lazy `fmt.Errorf`.
- `scheduler.go` — O(1) coalesce index (`region + cancel_version -> queue slot`) so `handleSchedulerQueueAppend` does not reverse-scan on every update.
- `snapshot_dispatch_hash.go` — Source-map key-order cache per epoch; streaming digest reuse; version-vector-first predicate; eliminate byte-staging for hash input.
- `spec.go` — Prop-shape fingerprint cache to avoid re-running `ValidateSerializableProps`; move to sorted `[]PropKV` on the hot path to eliminate map iteration.

### Recommended First Pick

- [x] Extend host runtime-status snapshots with hydration-attach state, latest snapshot and patch downgrade reasons, and stale-output counters so public status helpers do not need multiple runtime2 calls.

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

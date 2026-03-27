# Parallel Region Troubleshooting

Last updated: 2026-03-27

This guide covers the current public and runtime2 diagnostics you are most likely to hit while adopting `ui.ParallelRegion(...)`.

## Quick Checks

If a parallel region is not behaving the way you expect:

1. Confirm the renderer is registered exactly once.
2. Confirm `RendererID` and `RegionInstanceID` are non-empty.
3. Confirm props are fully serializable.
4. Confirm declared source IDs are valid.
5. Confirm you are not expecting worker-owned DOM behavior yet.

## Failure: Renderer Is Not Registered

Typical message:

- `parallel-region renderer resolution failed`

Cause:

- `ui.RegisterParallelRegion(...)` was never called
- the renderer ID string does not match
- registration ran too late

Fix:

- register during startup before the tree renders
- keep the ID stable and shared between call sites
- add a focused duplicate or missing-renderer test at the `ui` layer

## Failure: Duplicate Registration

Typical message:

- `parallel-region renderer "...\" is already registered`

Cause:

- the same renderer ID is registered more than once in one process

Fix:

- centralize registration
- avoid registering from hot paths, request paths, or repeated component code

## Failure: Props Are Not Serializable

Typical messages include:

- `unsupported event-closure prop`
- `unsupported direct DOM interop marker`
- `unsupported ref marker`
- `unsupported kind`

Cause:

- props include closures, refs, channels, DOM handles, or other runtime-local values

Fix:

- move interactive logic back to the owner component
- pass only display data into the registered region renderer
- keep refs, portals, direct DOM interop, and event handlers outside the region props

## Failure: Invalid Source IDs

Typical message:

- `source ID ... contains unsupported character`

Cause:

- source IDs were hand-built incorrectly

Fix:

- prefer `ui.BuildParallelRegionSourceIDs(...)`
- keep source IDs stable and string-safe

## Failure: Invalid Region Instance IDs

Typical message:

- `region instance ID is required`

Cause:

- `RegionInstanceID` is empty or unstable

Fix:

- assign a stable instance ID for each mounted region
- do not reuse one ID for multiple distinct mounted regions at the same time

## Local-Only Native Behavior

Current behavior on non-browser targets:

- the region still renders its shell and content locally
- runtime2 browser lifecycle state is not attached

This is intentional. The public shell remains deterministic on native builds even though the worker-backed path is browser-oriented.

## Fallback, Downgrade, And Restart

These terms are current runtime2 diagnostics, not user-facing marketing labels.

Fallback:

- means the region has switched to local ownership after a recovery-triggering failure
- common triggers include malformed transport payloads or invalid DOM patch state

Downgrade:

- means a preferred transport path fell back to a safer one
- current examples include shared-memory downgrade to structured-clone message transport

Restart:

- means worker-owned execution had to remount with a fresh epoch
- repair-remount handshakes keep stale results from committing

The diagnostics example app demonstrates these transitions explicitly:

- `examples/110-parallel-region-diagnostics`

## Protocol Mismatch

This is primarily a runtime2 or internal integration problem today.

If you are experimenting with the lower-level runtime2 control plane:

- confirm both sides use the same protocol version
- confirm control envelopes validate before dispatch
- confirm restart and patch-ready envelopes carry the expected epoch and region ID

The public `ui.ParallelRegion(...)` shell does not currently require you to manage protocol envelopes directly.

## Binary Transport

Current guidance:

- binary snapshot transport is part of runtime2 transport selection
- decode failures should downgrade or fail clearly, not silently mutate state

If you are validating binary transport paths:

- reproduce with focused runtime2 tests first
- check malformed-header and fallback coverage before widening to app-level flows

## Shared-Memory Deployment Requirements

Shared-memory transport still depends on browser deployment policy.

If you want shared-memory paths to stay available:

- serve with cross-origin isolation where required by the platform
- keep structured-clone transport available as the safety path
- treat downgrade to message transport as an expected resilience path, not a surprise

If shared memory is unavailable, the runtime should remain functional through structured-clone fallback.

## Recommended Debug Path

1. Reproduce with the smallest `ui` example or focused package test.
2. Check whether the failure is a public validation error, a transport downgrade, or a recovery fallback.
3. If it is transport- or recovery-related, inspect the runtime2 diagnostics next.
4. Only widen to full app debugging after the focused contract is understood.

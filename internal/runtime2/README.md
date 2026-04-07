# Internal Runtime2

Location: `internal/runtime2/`

This folder contains the experimental multithreaded runtime and its transport, scheduling, and host-region machinery.

## File Layout

- `scheduler*.go`, `shard_session.go`: worker scheduling, shard identity, and session coordination
- `host_region_adapter*.go`: host-region lifecycle, diagnostics, dispatch, recovery, and snapshot handling
- `worker_region_runtime*.go`: worker-side region execution
- `render_ir*.go`, `render_node_*.go`, `render_prop_*.go`, `render_string_table.go`: canonical render IR build and validation
- `snapshot*.go`, `shared_snapshot*.go`, `structured_clone_snapshot*.go`, `binary_snapshot*.go`: snapshot capture and transport paths
- `patch_*.go`, `patch_stream*.go`, `shared_patch*.go`, `structured_clone_patch*.go`, `binary_patch*.go`: patch encoding, streaming, and replay
- `control*.go`, `coordinator.go`, `recovery_*.go`, `registry.go`: control-plane state and fallback behavior
- `perf_*.go`, `*_bench_test.go`, `*_test.go`: hotspot benchmarks and regression coverage

## Start Here

- Read [../../docs/REFERENCE_MANUAL/10-browser-interop-and-workers.md](../../docs/REFERENCE_MANUAL/10-browser-interop-and-workers.md) for the worker and browser-coordination boundary.
- Read [../../docs/REFERENCE_MANUAL/15-design-notes-and-boundaries.md](../../docs/REFERENCE_MANUAL/15-design-notes-and-boundaries.md) for the current experimental and non-default runtime posture.
- Start in `scheduler.go` and `host_region_adapter.go` for host-side behavior changes.
- Start in `worker_region_runtime.go` for worker execution changes.

This folder is still implementation-owned and not part of the stable external API surface.

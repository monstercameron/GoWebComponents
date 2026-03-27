# Worker Pools Vs Lanes

This note explains when to use `interop.OpenWorkerPool(...)` versus direct worker lanes (`[]interop.Worker`) for browser worker fanout.

Use it when worker-backed rendering or preprocessing is performance-sensitive and you need a repeatable decision.

## Terms

- pool: one shared scheduler (`interop.WorkerPool`) that admits requests and assigns each request to the next available worker.
- lane: one explicit worker handle per lane (`[]interop.Worker`) where the caller maps work units to workers directly.

## Quick Decision

Prefer lanes when all of these are true:

- worker count is fixed for the active mode
- each batch has deterministic work partitioning
- one request per lane per batch is natural
- minimizing per-request dispatch overhead matters

Prefer a pool when any of these are true:

- request arrivals are bursty or irregular
- many callers share the same worker fleet
- you need explicit queue/backpressure controls
- you want built-in replacement behavior when one pooled worker is disposed

## Tradeoffs

### Lane strengths

- lower dispatch overhead for deterministic fanout
- simple mental model for chunk-index to worker mapping
- no admission queue coordination in the hot path

### Lane costs

- app owns worker lifecycle and close behavior
- app must define mapping and guardrails
- no built-in queueing policy for surplus work

### Pool strengths

- central admission and queue-limit policy
- easier shared-service model across many callers
- built-in worker replacement path on disposed workers

### Pool costs

- extra scheduling and queue coordination overhead per request
- less direct control over strict chunk-to-worker affinity

## Measured Notes

The repo includes `BenchmarkRequestWorkerDecodedFanoutDispatch` in `interop/worker_fanout_bench_test.go` to compare the two dispatch shapes under identical request payloads.

Recent local runs (Windows/amd64, i7-12700) showed:

- direct lanes: about `18.4-21.5 us/op`
- worker pool: about `28.6-30.1 us/op`

Interpretation:

- direct lanes reduce dispatch overhead for fanout-heavy deterministic batches
- end-to-end page latency improvements are usually smaller when worker CPU time dominates total batch time

## Guardrails For Lane Mode

- validate `len(workers) > 0` before fanout
- cap or validate requested worker count at mode-boot time
- use deterministic mapping (`workerIndex := chunkIndex % len(workers)`)
- keep request contexts bounded with timeouts
- cancel prior batch work when a new revision supersedes it
- emit warn or error logs on fleet-size mismatch or malformed results

## Migration Checklist (Pool To Lanes)

1. Open workers directly with `interop.OpenGoWASMWorker(...)` and store `[]interop.Worker`.
2. Replace `pool.Request(...)` callers with direct `RequestWorkerDecoded(...)` against the mapped worker handle.
3. Preserve cancellation and timeout behavior from the old path.
4. Add explicit close handling (`worker.Terminate()`) for app unmount or mode switch.
5. Re-run scenario tests and perf checks, then compare before/after benchmark outputs.

## Example References

- `examples/200-runtime2-status`: direct worker lanes for one-probe-per-worker fanout.
- `examples/201-render-benchmark`: direct worker lanes for deterministic chunk preparation.
- `interop/worker_fanout_bench_test.go`: focused dispatch overhead comparison harness.
